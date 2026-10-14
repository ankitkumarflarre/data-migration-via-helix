package billing

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

func (t *tx) system(version int) error {
	if t.Version != version {
		return ErrConflict
	}
	t.Version++
	return nil
}

// SetSystemDate moves the workspace date. Every "today" default and every
// time-based rule uses it.
func (s *Store) SetSystemDate(version int, date string) error {
	return s.update(func(t *tx) error {
		if !validDate(date) {
			return &ValidationError{map[string]string{"date": "Enter a valid date."}}
		}
		if err := t.system(version); err != nil {
			return err
		}
		t.SystemDate = date
		return nil
	})
}

type RunResult struct {
	Invoices int    `json:"invoices"`
	PastDue  int    `json:"past_due"`
	Issued   int    `json:"issued"`
	Message  string `json:"message"`
}

// RunDueProcesses is what the overnight cycle does on the system date:
// invoice installments whose invoice date has come, flag past-due ones, and
// let a queued disbursement interface request issue authorized refunds.
// Terms on hold are skipped.
func (s *Store) RunDueProcesses(version int) (*RunResult, error) {
	r := &RunResult{}
	err := s.update(func(t *tx) error {
		if err := t.system(version); err != nil {
			return err
		}
		terms := []*Term{}
		for _, tm := range t.Terms {
			terms = append(terms, tm)
		}
		sort.Slice(terms, func(i, j int) bool { return terms[i].ID < terms[j].ID })
		touched := map[string]bool{}
		for _, tm := range terms {
			if t.holdActive(tm) {
				continue
			}
			for _, in := range tm.Installments {
				if in.Balance() == 0 {
					continue
				}
				if in.InvoiceID == "" && in.InstallDate <= t.SystemDate {
					t.invoice(tm, in, SystemUser)
					r.Invoices++
					touched[tm.AccountNumber] = true
				}
				if !in.PastDue && in.DueDate < t.SystemDate {
					in.PastDue = true
					r.PastDue++
					touched[tm.AccountNumber] = true
					t.activity(tm.AccountNumber, "Installment Past Due", SystemUser, cents(in.Balance()), fmt.Sprintf("Installment of %s due %s on policy %s is past due.", USD(in.Balance()), US(in.DueDate), tm.PolicyNumber))
				}
			}
		}
		svc := &t.Settings.DisbursementService
		if svc.Queued {
			ids := []string{}
			for id, d := range t.Disbursements {
				if d.Status == "Authorized" {
					ids = append(ids, id)
				}
			}
			sort.Strings(ids)
			for _, id := range ids {
				d := t.Disbursements[id]
				d.Status, d.IssuedOn, d.MethodDetail = "Issued", t.SystemDate, "Check"
				touched[d.AccountNumber] = true
				r.Issued++
				t.activity(d.AccountNumber, "Disbursement Issued", SystemUser, cents(d.Amount), fmt.Sprintf("Disbursement for %s was issued by check. Payee: %s", USD(d.Amount), d.PayeeName))
			}
			svc.Queued, svc.QueuedAt, svc.LastRun = false, "", t.stamp()
		}
		for n := range touched {
			if a := t.Accounts[n]; a != nil {
				a.Version++
			}
		}
		r.Message = fmt.Sprintf("Due processes ran for %s: %d invoice(s) generated, %d installment(s) past due, %d disbursement(s) issued.", US(t.SystemDate), r.Invoices, r.PastDue, r.Issued)
		return nil
	})
	return r, err
}

type SettingsUpdate struct {
	Version            int        `json:"version"`
	AutoAuthorizeLimit int64      `json:"auto_authorize_limit"`
	PIF                *PIFConfig `json:"pif"`
	OfferedPlans       []string   `json:"offered_plans"`
}

func (s *Store) SaveSettings(in SettingsUpdate) error {
	return s.update(func(t *tx) error {
		f := fieldErrors{}
		if in.AutoAuthorizeLimit < 0 {
			f.add("auto_authorize_limit", "Enter $0.00 or more.")
		}
		if in.PIF != nil {
			if in.PIF.DiscountBasisPoints < 1 || in.PIF.DiscountBasisPoints > 10000 {
				f.add("pif.discount_basis_points", "Enter a discount from 0.01% to 100%.")
			}
			if in.PIF.EvaluationDays < 1 || in.PIF.EvaluationDays > 366 {
				f.add("pif.evaluation_days", "Enter 1 to 366 days.")
			}
		}
		if len(in.OfferedPlans) == 0 {
			f.add("offered_plans", "Offer at least one payment plan.")
		}
		for _, p := range in.OfferedPlans {
			if !oneOf(Codes.PaymentPlans, p) {
				f.add("offered_plans", "Choose plans from the list.")
			}
		}
		if err := f.err(); err != nil {
			return err
		}
		if err := t.system(in.Version); err != nil {
			return err
		}
		t.Settings.AutoAuthorizeLimit, t.Settings.PIF, t.Settings.OfferedPlans = in.AutoAuthorizeLimit, in.PIF, in.OfferedPlans
		return nil
	})
}

// DisbursementService starts or stops the disbursement interface. Starting it
// while a request is already queued is refused: Duck Creek would add a second
// async request and run the interface twice.
func (s *Store) DisbursementService(version int, action string) error {
	return s.update(func(t *tx) error {
		if err := t.system(version); err != nil {
			return err
		}
		svc := &t.Settings.DisbursementService
		switch action {
		case "start":
			if svc.Queued {
				return rule("A disbursement interface request is already on the async queue (added %s). Starting the service again would create a duplicate request.", strings.Replace(svc.QueuedAt, "T", " ", 1))
			}
			svc.Status, svc.Queued, svc.QueuedAt = "Running", true, t.stamp()
		case "stop":
			svc.Status, svc.Queued, svc.QueuedAt = "Stopped", false, ""
		default:
			return &ValidationError{map[string]string{"action": "Choose Start Service or Stop Service."}}
		}
		return nil
	})
}

// ---- batches ----

type BatchInput struct {
	Number        string `json:"number"`
	ControlTotal  int64  `json:"control_total"`
	Currency      string `json:"currency"`
	DefaultMethod string `json:"default_method"`
	OrgUnit       string `json:"org_unit"`
	Description   string `json:"description"`
}

var batchNumber = regexp.MustCompile(`^[A-Za-z0-9-]{1,20}$`)

func (s *Store) CreateBatch(in BatchInput) (*Batch, error) {
	var out *Batch
	err := s.update(func(t *tx) error {
		f := fieldErrors{}
		in.Number = strings.TrimSpace(in.Number)
		switch {
		case !batchNumber.MatchString(in.Number):
			f.add("number", "Enter a batch number (letters, digits and dashes).")
		case t.Batches[in.Number] != nil:
			f.add("number", "Batch "+in.Number+" already exists.")
		}
		if in.ControlTotal < 0 {
			f.add("control_total", "Enter $0.00 or more.")
		}
		if !oneOf(Codes.BatchCurrencies, in.Currency) {
			f.add("currency", "Select a currency.")
		}
		if !oneOf(Codes.PaymentMethods, in.DefaultMethod) {
			f.add("default_method", "Select a default payment method.")
		}
		if !oneOf(Codes.ProcessingOrgUnits, in.OrgUnit) {
			f.add("org_unit", "Select a processing org. unit.")
		}
		if len(in.Description) > 200 {
			f.add("description", "Too long.")
		}
		if err := f.err(); err != nil {
			return err
		}
		out = &Batch{Number: in.Number, Version: 1, EntryDate: t.SystemDate, CreatedAt: t.stamp(), User: CurrentUser, Status: "Open", ControlTotal: in.ControlTotal, Currency: in.Currency, DefaultMethod: in.DefaultMethod, OrgUnit: in.OrgUnit, Description: strings.TrimSpace(in.Description)}
		t.Batches[out.Number] = out
		return nil
	})
	return out, err
}

func (s *Store) ReverseBatch(number string, version int) error {
	return s.update(func(t *tx) error {
		b := t.Batches[number]
		if b == nil {
			return ErrNotFound
		}
		if b.Version != version {
			return ErrConflict
		}
		if b.Status != "Deposited" {
			return rule("Only deposited batches can be reversed.")
		}
		b.Version++
		b.Status, b.ReversedOn = "Reversed", t.SystemDate
		return nil
	})
}

// ---- error queue ----

type ResubmitInput struct {
	Version   int              `json:"version"`
	Sections  []PayloadSection `json:"sections"`
	InvoiceID string           `json:"invoice_id"`
	AsyncID   string           `json:"async_id"`
}

var accountRef = regexp.MustCompile(`<AccountReference>\s*([^<\s]+)\s*</AccountReference>`)

// Resubmit saves the edited payload and runs the job again against the store.
func (s *Store) Resubmit(id string, in ResubmitInput) (*ErrorItem, error) {
	var out *ErrorItem
	err := s.update(func(t *tx) error {
		e := t.Errors[id]
		if e == nil {
			return ErrNotFound
		}
		if e.Version != in.Version {
			return ErrConflict
		}
		if e.Status != "Open" {
			return rule("This request was already resolved.")
		}
		f := fieldErrors{}
		if len(in.Sections) != len(e.Sections) {
			f.add("sections", "Keep the payload sections of the request.")
		} else {
			for i, sec := range in.Sections {
				if sec.Name != e.Sections[i].Name {
					f.add("sections", "Keep the payload sections of the request.")
				}
				if len(sec.Value) > 20000 {
					f.add(fmt.Sprintf("sections.%d", i), "Too long.")
				}
			}
		}
		if len(in.InvoiceID) > 40 || len(in.AsyncID) > 80 {
			f.add("invoice_id", "Too long.")
		}
		if err := f.err(); err != nil {
			return err
		}
		e.Version++
		e.Sections, e.InvoiceID, e.AsyncID = in.Sections, in.InvoiceID, in.AsyncID
		e.Attempts++
		e.At = t.stamp()
		if done, detail := t.runJob(e); done {
			e.Status, e.ResolvedAt, e.Description = "Resolved", t.stamp(), detail
		} else {
			e.Description = detail
		}
		out = e
		return nil
	})
	return out, err
}

// runJob re-runs a failed async request.
func (t *tx) runJob(e *ErrorItem) (bool, string) {
	payload := ""
	for _, s := range e.Sections {
		payload += s.Value
	}
	ref := e.Reference
	if m := accountRef.FindStringSubmatch(payload); m != nil {
		ref = m[1]
	}
	if ref == "" {
		return false, "The payload has no AccountReference."
	}
	switch e.Type {
	case "Generate Invoice":
		acct := t.Accounts[ref]
		if acct == nil {
			return false, "Account " + ref + " was not found."
		}
		for _, tm := range t.termsOf(acct.Number) {
			for _, in := range tm.Installments {
				if in.InvoiceID == "" && in.Balance() > 0 {
					inv := t.invoice(tm, in, SystemUser)
					acct.Version++
					return true, "Resubmitted: invoice " + inv.ID + " generated for account " + ref + "."
				}
			}
		}
		return true, "Resubmitted: account " + ref + " has nothing left to invoice."
	case "Automatic Pending Cancellation Process", "Receivable":
		for _, tm := range t.Terms {
			if tm.PolicyNumber == ref || tm.AccountNumber == ref {
				return true, "Resubmitted: policy term " + termLabel(tm) + " was processed."
			}
		}
		return false, "Policy " + ref + " was not found."
	}
	return false, "Unknown request type."
}

// ImportStatements imports the selected agent statements.
func (s *Store) ImportStatements(ids []string) (int, error) {
	n := 0
	err := s.update(func(t *tx) error {
		if len(ids) == 0 {
			return &ValidationError{map[string]string{"ids": "Select the statements to import."}}
		}
		for _, id := range ids {
			st := t.Statements[id]
			if st == nil {
				return ErrNotFound
			}
			if st.Imported == "" {
				st.Imported = t.SystemDate
				n++
			}
		}
		return nil
	})
	return n, err
}

// Touch records an account in Recently Accessed.
func (s *Store) Touch(number string) error {
	return s.update(func(t *tx) error {
		if t.Accounts[number] == nil {
			return ErrNotFound
		}
		out := []string{number}
		for _, n := range t.Recent {
			if n != number && len(out) < 10 {
				out = append(out, n)
			}
		}
		t.Recent = out
		return nil
	})
}
