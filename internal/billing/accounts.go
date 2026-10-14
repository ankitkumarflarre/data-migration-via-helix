package billing

import (
	"fmt"
	"sort"
	"strings"
)

type NewAccountInput struct {
	PayorID       string `json:"payor_id"`
	PayeeID       string `json:"payee_id"`
	SameParty     bool   `json:"same_party"`
	AccountClass  string `json:"account_class"`
	TargetDueDays int    `json:"target_due_days"`
	EndOfMonth    bool   `json:"end_of_month"`
}

// CreateAccount is Create New Account. Accounts made without a policy get the
// next ten-digit number (as 0000028517 in Recently Accessed).
func (s *Store) CreateAccount(in NewAccountInput) (*Account, error) {
	var out *Account
	err := s.update(func(t *tx) error {
		f := fieldErrors{}
		if t.Parties[in.PayorID] == nil {
			f.add("payor_id", "Search for or create the account payor.")
		}
		if in.SameParty {
			in.PayeeID = in.PayorID
		} else if t.Parties[in.PayeeID] == nil {
			f.add("payee_id", "Select the payee, or tick Payee and payor are the same party.")
		}
		if !oneOf(Codes.AccountClasses, in.AccountClass) {
			f.add("account_class", "Select a bill account class.")
		}
		if in.TargetDueDays < 1 || in.TargetDueDays > 31 {
			f.add("target_due_days", "Enter a day from 1 to 31.")
		}
		if err := f.err(); err != nil {
			return err
		}
		t.Seq["ACCT"]++
		number := fmt.Sprintf("%010d", 28517+t.Seq["ACCT"])
		out = t.newAccount(number, in.PayorID, in.PayeeID, in.AccountClass, in.TargetDueDays)
		out.EndOfMonth = in.EndOfMonth
		return nil
	})
	return out, err
}

func (t *tx) newAccount(number, payor, payee, class string, dueDays int) *Account {
	a := &Account{Number: number, Version: 1, Status: "Active", PayorID: payor, PayeeID: payee, TargetDueDays: dueDays, AccountType: "Account", AccountClass: class, CollectionMethod: "PA", Currency: "USD", AccountPlan: Codes.AccountPlans[0], CreatedOn: t.SystemDate}
	t.Accounts[number] = a
	t.activity(number, "Account Created", CurrentUser, nil, "Account created.")
	return a
}

// issueTerm adds a policy term with its receivables and schedule to an account.
func (t *tx) issueTerm(acct *Account, tm *Term, rec []*Receivable, firstDue string) {
	tm.ID = t.next("TRM")
	tm.AccountNumber = acct.Number
	for _, r := range rec {
		r.ID = t.next("RCV")
		tm.Receivables = append(tm.Receivables, r)
	}
	generateSchedule(t, tm, rec, firstDue)
	t.Terms[tm.ID] = tm
	t.activity(acct.Number, "Policy Term Issued", CurrentUser, nil, fmt.Sprintf("Policy Term %s %s - %s created", tm.PolicyNumber, US2(tm.Effective), US2(tm.Expiration)))
	var total int64
	for _, r := range rec {
		total += r.Amount
	}
	tx := "New Business"
	if len(rec) > 0 {
		tx = rec[0].Transaction
	}
	a := t.activity(acct.Number, "Policy Transaction", CurrentUser, cents(total), fmt.Sprintf("%s (Policy %s): receivable has been processed.", tx, termLabel(tm)))
	a.TermID = tm.ID
}

type SettingsInput struct {
	Version       int    `json:"version"`
	AccountPlan   string `json:"account_plan"`
	InvoiceDueDay int    `json:"invoice_due_day"`
}

// SaveAccountSettings is Select Billing Account Options.
func (s *Store) SaveAccountSettings(number string, in SettingsInput) error {
	return s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		if !oneOf(Codes.AccountPlans, in.AccountPlan) {
			f.add("account_plan", "Select an account plan.")
		}
		if in.InvoiceDueDay < 1 || in.InvoiceDueDay > 31 {
			f.add("invoice_due_day", "Enter a day from 1 to 31.")
		}
		if err := f.err(); err != nil {
			return err
		}
		changes := []string{}
		if acct.AccountPlan != in.AccountPlan {
			changes = append(changes, fmt.Sprintf("Account Plan was changed from %s to %s", acct.AccountPlan, in.AccountPlan))
		}
		if acct.InvoiceDueDay != in.InvoiceDueDay {
			changes = append(changes, fmt.Sprintf("Invoice Due Day was changed to %d", in.InvoiceDueDay))
		}
		acct.AccountPlan, acct.InvoiceDueDay = in.AccountPlan, in.InvoiceDueDay
		if len(changes) > 0 {
			t.activity(acct.Number, "Account Configuration Change", CurrentUser, nil, strings.Join(changes, ". ")+".")
		}
		return nil
	})
}

type PartyRoleInput struct {
	Version int    `json:"version"`
	Role    string `json:"role"` // payee | payor
	PartyID string `json:"party_id"`
}

// ChangeParty is Change Payee / Change Payor.
func (s *Store) ChangeParty(number string, in PartyRoleInput) error {
	return s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		if t.Parties[in.PartyID] == nil {
			return &ValidationError{map[string]string{"party_id": "Select a party from the search results."}}
		}
		name := t.partyName(in.PartyID)
		switch in.Role {
		case "payee":
			acct.PayeeID = in.PartyID
			t.activity(acct.Number, "Payee Changed", CurrentUser, nil, "Account payee was changed to "+name+".")
		case "payor":
			acct.PayorID = in.PartyID
			t.activity(acct.Number, "Payor Changed", CurrentUser, nil, "Account payor was changed to "+name+".")
		default:
			return &ValidationError{map[string]string{"role": "Choose payee or payor."}}
		}
		return nil
	})
}

// ---- holds ----

type HoldInput struct {
	Version int    `json:"version"`
	Kind    string `json:"kind"` // individual | event
	TermID  string `json:"term_id"`
	EventID string `json:"event_id"`
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	End     string `json:"end"`
}

func holdText(scope string, h *Hold) string {
	end := "Indefinite"
	if h.End != "" {
		end = US(h.End)
	}
	return fmt.Sprintf("%s hold set. Hold type: %s. Reason: %s. Beginning: %s Ending: %s", scope, h.Type, h.Reason, US(h.Start), end)
}

// PlaceHold starts a hold on the account or one of its terms. It always starts
// on the system date: holds cannot be back-dated.
func (s *Store) PlaceHold(number string, in HoldInput) (*Hold, string, error) {
	var out *Hold
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		h := &Hold{AccountNumber: acct.Number, Start: t.SystemDate, Status: "Active"}
		if in.TermID != "" {
			if _, err := t.term(acct, in.TermID); err != nil {
				return err
			}
			h.TermID = in.TermID
		}
		switch in.Kind {
		case "event":
			ev := t.HoldEvents[in.EventID]
			if ev == nil || (ev.End != "" && ev.End < t.SystemDate) {
				f.add("event_id", "Select a current hold event.")
				break
			}
			h.EventID, h.Type, h.Reason, h.End = ev.ID, ev.Type, ev.Reason, ev.End
		case "individual", "":
			if !oneOf(Codes.HoldTypes, in.Type) {
				f.add("type", "Select a hold type.")
			}
			if !oneOf(Codes.HoldReasons, in.Reason) {
				f.add("reason", "Select a hold reason.")
			}
			if in.End != "" && (!validDate(in.End) || in.End < t.SystemDate) {
				f.add("end", "The end date cannot be before the start date.")
			}
			h.Type, h.Reason, h.End = in.Type, in.Reason, in.End
		default:
			f.add("kind", "Choose an individual hold or an area hold event.")
		}
		if err := f.err(); err != nil {
			return err
		}
		if t.activeHold(acct.Number, h.TermID) != nil {
			return rule("A hold is already active. Update or resume it instead.")
		}
		h.ID = t.next("HLD")
		t.Holds[h.ID] = h
		scope := "Account"
		if h.TermID != "" {
			scope = "Policy term " + t.Terms[h.TermID].PolicyNumber
		}
		msg = holdText(scope, h)
		t.activity(acct.Number, "Set Hold", CurrentUser, nil, msg)
		out = h
		return nil
	})
	return out, msg, err
}

type HoldUpdateInput struct {
	Version      int    `json:"version"`
	Action       string `json:"action"` // save | resume
	End          string `json:"end"`
	ResumeOption string `json:"resume_option"`
}

// UpdateHold saves a new end date or resumes billing. Adjust due dates moves
// the open installments by the days the hold lasted.
func (s *Store) UpdateHold(number, id string, in HoldUpdateInput) (string, error) {
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		h := t.Holds[id]
		if h == nil || h.AccountNumber != acct.Number {
			return ErrNotFound
		}
		if h.Status != "Active" {
			return rule("This hold was already released.")
		}
		switch in.Action {
		case "save":
			if in.End != "" && (!validDate(in.End) || in.End < h.Start) {
				return &ValidationError{map[string]string{"end": "The end date cannot be before the start date."}}
			}
			h.End = in.End
			msg = "Hold updated."
		case "resume":
			if !oneOf(Codes.HoldResumeOptions, in.ResumeOption) {
				return &ValidationError{map[string]string{"resume_option": "Choose whether to adjust due dates."}}
			}
			h.Status, h.ReleasedOn, h.ResumeOption = "Released", t.SystemDate, in.ResumeOption
			text := "Hold released. Resume option: " + in.ResumeOption + "."
			if in.ResumeOption == "Adjust due dates" {
				days := max(daysBetween(h.Start, t.SystemDate), 0)
				moved := 0
				for _, tm := range t.termsOf(acct.Number) {
					if h.TermID != "" && tm.ID != h.TermID {
						continue
					}
					for _, inst := range tm.Installments {
						if days > 0 && inst.Balance() > 0 && inst.DueDate >= h.Start {
							inst.DueDate = addDays(inst.DueDate, days)
							inst.InstallDate = addDays(inst.InstallDate, days)
							inst.PastDue = false
							moved++
						}
					}
					if moved > 0 {
						tm.History = append(tm.History, ScheduleNote{t.SystemDate, fmt.Sprintf("Due dates moved %d days after the hold from %s.", days, US(h.Start))})
					}
				}
				h.AdjustedDays = days
				text += fmt.Sprintf(" %d open installment(s) moved %d day(s).", moved, days)
			}
			t.activity(acct.Number, "Release Hold", CurrentUser, nil, text)
			msg = text
		default:
			return &ValidationError{map[string]string{"action": "Choose Save or Resume."}}
		}
		return nil
	})
	return msg, err
}

// EndHoldEvent is the End link on Manage Hold Events: the event and the
// account holds attached to it end on the system date.
func (s *Store) EndHoldEvent(id string, version int) error {
	return s.update(func(t *tx) error {
		ev := t.HoldEvents[id]
		if ev == nil {
			return ErrNotFound
		}
		if ev.Version != version {
			return ErrConflict
		}
		if ev.End != "" && ev.End <= t.SystemDate {
			return rule("This hold event has already ended.")
		}
		ev.Version++
		ev.End = t.SystemDate
		for _, h := range t.Holds {
			if h.EventID == id && h.Status == "Active" {
				h.End = t.SystemDate
			}
		}
		return nil
	})
}

// ---- tasks ----

type TaskInput struct {
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	DueDate     string `json:"due_date"`
	Description string `json:"description"`
	Queue       string `json:"queue"`
	Assignee    string `json:"assignee"`
}

func (s *Store) AddTask(number string, in TaskInput) (*Task, error) {
	var out *Task
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		in.Title = strings.TrimSpace(in.Title)
		if in.Title == "" || len(in.Title) > 200 {
			f.add("title", "Enter a title.")
		}
		if in.Type == "" {
			in.Type = Codes.TaskTypes[0]
		}
		if !oneOf(Codes.TaskTypes, in.Type) {
			f.add("type", "Select a task type.")
		}
		if in.DueDate == "" {
			in.DueDate = t.SystemDate
		}
		if !validDate(in.DueDate) {
			f.add("due_date", "Enter a valid date.")
		}
		if !oneOf(Codes.TaskQueues, in.Queue) {
			f.add("queue", "Select a queue.")
		}
		if in.Assignee != "" && !oneOf(Codes.Assumed.TaskAssignees, in.Assignee) {
			f.add("assignee", "Select an assignee.")
		}
		if len(in.Description) > 2000 {
			f.add("description", "Too long.")
		}
		if err := f.err(); err != nil {
			return err
		}
		tk := &Task{ID: t.next("TSK"), Title: in.Title, AccountNumber: acct.Number, Type: in.Type, Category: in.Type, DueDate: in.DueDate, Description: in.Description, Queue: in.Queue, Assignee: in.Assignee, Status: "Open", Created: t.stamp()}
		t.Tasks[tk.ID] = tk
		out = tk
		return nil
	})
	return out, err
}

// PickupTask assigns a task in a queue to the current user.
func (s *Store) PickupTask(number, id string, version int) error {
	return s.update(func(t *tx) error {
		acct, err := t.account(number, version)
		if err != nil {
			return err
		}
		tk := t.Tasks[id]
		if tk == nil || tk.AccountNumber != acct.Number {
			return ErrNotFound
		}
		if tk.Assignee != "" {
			return rule("This task is already assigned to %s.", tk.Assignee)
		}
		tk.Assignee = CurrentUser
		return nil
	})
}

// ---- policy term actions ----

type PlanInput struct {
	Version int    `json:"version"`
	Plan    string `json:"plan"`
}

// SetTermPlan is Term Plan Setup. The unpaid, uninvoiced balance is spread
// again over the new plan from the next due date.
func (s *Store) SetTermPlan(number, termID string, in PlanInput) error {
	return s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		tm, err := t.term(acct, termID)
		if err != nil {
			return err
		}
		if !oneOf(t.Settings.OfferedPlans, in.Plan) {
			return &ValidationError{map[string]string{"plan": "Select a payment plan."}}
		}
		if in.Plan == tm.PaymentPlan {
			return nil
		}
		old := tm.PaymentPlan
		tm.PaymentPlan = in.Plan
		// Take the open balance off the uninvoiced installments, by receivable.
		remaining := map[string]int64{}
		order := []string{}
		var firstDue string
		kept := []*Installment{}
		for _, inst := range tm.Installments {
			if inst.InvoiceID != "" || inst.Balance() == 0 {
				kept = append(kept, inst)
				continue
			}
			if firstDue == "" || inst.DueDate < firstDue {
				firstDue = inst.DueDate
			}
			paidItems := []*InstallItem{}
			for _, it := range inst.Items {
				if b := it.Balance(); b > 0 {
					if _, ok := remaining[it.ReceivableID]; !ok {
						order = append(order, it.ReceivableID)
					}
					remaining[it.ReceivableID] += b
					it.Amount -= b
				}
				if it.Amount > 0 {
					paidItems = append(paidItems, it)
				}
			}
			if len(paidItems) > 0 {
				inst.Items = paidItems
				kept = append(kept, inst)
			}
		}
		tm.Installments = kept
		if len(order) > 0 {
			rec := []*Receivable{}
			for _, id := range order {
				for _, r := range tm.Receivables {
					if r.ID == id {
						c := *r
						c.Amount = remaining[id]
						rec = append(rec, &c)
					}
				}
			}
			if firstDue < t.SystemDate {
				firstDue = t.SystemDate
			}
			generateSchedule(t, tm, rec, firstDue)
			sort.SliceStable(tm.Installments, func(i, j int) bool { return tm.Installments[i].DueDate < tm.Installments[j].DueDate })
		}
		text := fmt.Sprintf("On %s, the following configuration changes were made to policy term %s: Payment Plan was changed from %s to %s.", US(t.SystemDate), termLabel(tm), old, in.Plan)
		tm.History = append(tm.History, ScheduleNote{t.SystemDate, "Payment plan changed from " + old + " to " + in.Plan + "."})
		a := t.activity(acct.Number, "Policy Term Configuration Change", CurrentUser, nil, text)
		a.TermID = tm.ID
		return nil
	})
}

// SetRenewalPlan is Renewal Plan Setup.
func (s *Store) SetRenewalPlan(number, termID string, in PlanInput) error {
	return s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		tm, err := t.term(acct, termID)
		if err != nil {
			return err
		}
		if !oneOf(t.Settings.OfferedPlans, in.Plan) {
			return &ValidationError{map[string]string{"plan": "Select a renewal payment plan."}}
		}
		if in.Plan == tm.RenewalPlan {
			return nil
		}
		old := tm.RenewalPlan
		tm.RenewalPlan = in.Plan
		a := t.activity(acct.Number, "Policy Renewal Configuration Change", CurrentUser, nil, fmt.Sprintf("On %s, the following configuration changes were made to the renewal term for policy %s: Renewal Payment Plan was changed from %s to %s.", US(t.SystemDate), tm.PolicyNumber, old, in.Plan))
		a.TermID = tm.ID
		return nil
	})
}

type TransferInput struct {
	Version int    `json:"version"`
	Target  string `json:"target"`
}

// TransferTerm moves a policy term to another billing account.
func (s *Store) TransferTerm(number, termID string, in TransferInput) (string, error) {
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		tm, err := t.term(acct, termID)
		if err != nil {
			return err
		}
		target := t.Accounts[strings.TrimSpace(in.Target)]
		switch {
		case target == nil:
			return &ValidationError{map[string]string{"target": "No billing account matches that number."}}
		case target.Number == acct.Number:
			return &ValidationError{map[string]string{"target": "Choose a different account."}}
		}
		target.Version++
		tm.AccountNumber = target.Number
		for _, h := range t.Holds {
			if h.TermID == tm.ID {
				h.AccountNumber = target.Number
			}
		}
		t.activity(acct.Number, "Policy Term Transferred", CurrentUser, nil, fmt.Sprintf("Policy term %s was transferred to account %s.", termLabel(tm), target.Number))
		t.activity(target.Number, "Policy Term Transferred", CurrentUser, nil, fmt.Sprintf("Policy term %s was transferred from account %s.", termLabel(tm), acct.Number))
		msg = fmt.Sprintf("Policy term %s was transferred to account %s.", termLabel(tm), target.Number)
		return nil
	})
	return msg, err
}

// CollectionsBlock is the message Transfer To Collections shows when the term
// owes nothing; empty when the transfer is allowed.
func CollectionsBlock(l TermLedger) string {
	if l.Outstanding > 0 {
		return ""
	}
	return fmt.Sprintf("The term outstanding balance: %s is not enough to make a transfer to collections.", USD(l.Outstanding))
}

func (s *Store) TransferToCollections(number, termID string, version int) (string, error) {
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, version)
		if err != nil {
			return err
		}
		tm, err := t.term(acct, termID)
		if err != nil {
			return err
		}
		l := t.termLedger(tm)
		if block := CollectionsBlock(l); block != "" {
			return rule("%s", block)
		}
		if tm.Status == "Collections" {
			return rule("This policy term is already in collections.")
		}
		tm.Status = "Collections"
		msg = fmt.Sprintf("Policy term %s was transferred to collections. Outstanding balance: %s", termLabel(tm), USD(l.Outstanding))
		a := t.activity(acct.Number, "Transfer To Collections", CurrentUser, cents(l.Outstanding), msg)
		a.TermID = tm.ID
		return nil
	})
	return msg, err
}

// PIFView is what Paid In Full Discount shows for a term.
type PIFView struct {
	Configured        bool   `json:"configured"`
	Message           string `json:"message,omitempty"`
	PotentialDiscount int64  `json:"potential_discount"`
	EvaluationStart   string `json:"evaluation_start,omitempty"`
	EvaluationEnd     string `json:"evaluation_end,omitempty"`
}

const pifMissing = "Cannot be calculated as the Paid In Full configuration has not been established"

func (st *State) pif(tm *Term) PIFView {
	c := st.Settings.PIF
	if c == nil {
		return PIFView{Message: pifMissing}
	}
	var premium int64
	for _, r := range tm.Receivables {
		if r.Receivable == "Premium" {
			premium += r.Amount
		}
	}
	return PIFView{Configured: true, PotentialDiscount: percentOf(premium, c.DiscountBasisPoints), EvaluationStart: tm.Effective, EvaluationEnd: addDays(tm.Effective, c.EvaluationDays)}
}

type PIFInput struct {
	Version int    `json:"version"`
	Amount  int64  `json:"amount"`
	Reason  string `json:"reason"`
}

// ApplyPIF credits a paid-in-full discount to the term. Any credit beyond the
// open balance goes to account suspense.
func (s *Store) ApplyPIF(number, termID string, in PIFInput) (string, error) {
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		tm, err := t.term(acct, termID)
		if err != nil {
			return err
		}
		v := t.pif(tm)
		if !v.Configured {
			return rule("%s", pifMissing)
		}
		f := fieldErrors{}
		if in.Amount <= 0 || in.Amount > v.PotentialDiscount {
			f.add("amount", "Enter a discount between $0.01 and the potential discount of "+USD(v.PotentialDiscount)+".")
		}
		if !oneOf(Codes.PIFReasons, in.Reason) {
			f.add("reason", "Select a reason.")
		}
		if err := f.err(); err != nil {
			return err
		}
		if t.SystemDate > v.EvaluationEnd {
			return rule("The evaluation period ended on %s.", US(v.EvaluationEnd))
		}
		left := in.Amount
		items := openItems([]*Term{tm})
		for i := len(items) - 1; i >= 0 && left > 0; i-- {
			c := min(left, items[i].item.Balance())
			items[i].item.Credit += c
			left -= c
		}
		if left > 0 {
			s := &SuspenseItem{ID: t.next("SUS"), AccountNumber: acct.Number, PaymentSource: "Paid In Full Discount", RefundSource: "Check", Date: t.SystemDate, Reason: "Overpayment", Original: left, Amount: left, Status: "No Hold"}
			t.Suspense[s.ID] = s
		}
		msg = fmt.Sprintf("Paid In Full Discount of %s applied to policy term %s. Reason: %s", USD(in.Amount), termLabel(tm), in.Reason)
		a := t.activity(acct.Number, "Paid In Full Discount", CurrentUser, cents(in.Amount), msg)
		a.TermID = tm.ID
		return nil
	})
	return msg, err
}
