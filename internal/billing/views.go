package billing

import (
	"sort"
	"strings"
)

type SystemView struct {
	Date     string   `json:"date"`
	Version  int      `json:"version"`
	Settings Settings `json:"settings"`
	Empty    bool     `json:"empty"`
	User     string   `json:"user"`
}

func (s *Store) System() SystemView {
	var v SystemView
	s.read(func(st *State) { v = SystemView{st.SystemDate, st.Version, st.Settings, st.Empty(), CurrentUser} })
	return v
}

type AccountRef struct {
	Number  string `json:"number"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
}

type HomeView struct {
	Recent        []AccountRef    `json:"recent"`
	Notifications []*Notification `json:"notifications"`
	TasksDue      []*Task         `json:"tasks_due"`
	Reports       []string        `json:"reports"`
}

func (st *State) ref(a *Account) AccountRef {
	r := AccountRef{Number: a.Number, Name: st.partyName(a.PayorID), Status: a.Status}
	if p := st.Parties[a.PayorID]; p != nil {
		l1, l2 := p.Address(st.SystemDate)
		r.Address = strings.Trim(strings.TrimSpace(l1+", "+l2), ", ")
	}
	return r
}

func (s *Store) Home() HomeView {
	v := HomeView{Recent: []AccountRef{}, Notifications: []*Notification{}, TasksDue: []*Task{}, Reports: Codes.Reports}
	s.read(func(st *State) {
		for _, n := range st.Recent {
			if a := st.Accounts[n]; a != nil {
				v.Recent = append(v.Recent, st.ref(a))
			}
		}
		v.Notifications = append(v.Notifications, st.Notifications...)
		for _, t := range st.Tasks {
			if t.Status == "Open" && t.DueDate == st.SystemDate && t.Assignee == CurrentUser {
				v.TasksDue = append(v.TasksDue, t)
			}
		}
	})
	return v
}

// SearchAccounts finds accounts by number (Find a Billing Account, the
// header quick search and Enter A Single Payment).
func (s *Store) SearchAccounts(by, q string) ([]AccountRef, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, &ValidationError{map[string]string{"q": "Enter an account number."}}
	}
	if by != "" && !oneOf([]string{"Account Number"}, by) {
		return nil, &ValidationError{map[string]string{"by": "Search by Account Number."}}
	}
	out := []AccountRef{}
	s.read(func(st *State) {
		for _, a := range st.Accounts {
			if strings.EqualFold(a.Number, q) {
				out = append(out, st.ref(a))
			}
		}
	})
	return out, nil
}

type PartyCard struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Name   string `json:"name"`
	First  string `json:"first"`
	Last   string `json:"last"`
	Line1  string `json:"line1"`
	Line2  string `json:"line2"`
	Email  string `json:"email"`
}

func (st *State) card(id string) *PartyCard {
	p := st.Parties[id]
	if p == nil {
		return nil
	}
	l1, l2 := p.Address(st.SystemDate)
	c := &PartyCard{ID: p.ID, Number: p.Number, Name: p.DisplayName(st.SystemDate), Line1: l1, Line2: l2, Email: p.Email(st.SystemDate)}
	if rows := p.Current("name", st.SystemDate); len(rows) > 0 {
		c.First, c.Last = rows[0].Fields["first"], rows[0].Fields["last"]
	}
	return c
}

type TermRow struct {
	ID          string `json:"id"`
	Policy      string `json:"policy"`
	Reference   string `json:"reference"`
	Effective   string `json:"effective"`
	Expiration  string `json:"expiration"`
	Status      string `json:"status"`
	CurrentDue  int64  `json:"current_due"`
	PastDue     int64  `json:"past_due"`
	Outstanding int64  `json:"outstanding"`
	OnHold      bool   `json:"on_hold"`
}

type InvoiceTile struct {
	InvoiceDate string `json:"invoice_date"`
	Amount      int64  `json:"amount"`
	DueDate     string `json:"due_date"`
}

type PaymentTile struct {
	ID     string `json:"id"`
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
	Method string `json:"method"`
}

type ScheduleItem struct {
	Recv     string `json:"recv"`
	Trans    string `json:"trans"`
	Amount   int64  `json:"amount"`
	Cash     int64  `json:"cash"`
	Credit   int64  `json:"credit"`
	WriteOff int64  `json:"write_off"`
	Redist   int64  `json:"redist"`
	Balance  int64  `json:"balance"`
}

type ScheduleRow struct {
	ID          string         `json:"id"`
	InstallDate string         `json:"install_date"`
	DueDate     string         `json:"due_date"`
	Term        string         `json:"term"`
	TermID      string         `json:"term_id"`
	Invoiced    bool           `json:"invoiced"`
	Items       []ScheduleItem `json:"items"`
}

type HoldView struct {
	*Hold
	Banner string `json:"banner"`
	Term   string `json:"term,omitempty"`
}

type AccountView struct {
	Account      *Account           `json:"account"`
	Name         string             `json:"name"`
	SystemDate   string             `json:"system_date"`
	Payor        *PartyCard         `json:"payor"`
	Payee        *PartyCard         `json:"payee"`
	Ledger       AccountLedger      `json:"ledger"`
	Terms        []TermRow          `json:"terms"`
	Activities   []*Activity        `json:"activities"`
	LastInvoice  *InvoiceTile       `json:"last_invoice"`
	NextInvoice  *InvoiceTile       `json:"next_invoice"`
	LastPayment  *PaymentTile       `json:"last_payment"`
	Holds        []HoldView         `json:"holds"`
	HoldEvents   []*HoldEvent       `json:"hold_events"`
	Suspense     []*SuspenseItem    `json:"suspense"`
	Disburse     []*Disbursement    `json:"disbursements"`
	Tasks        []*Task            `json:"tasks"`
	Payments     []*Payment         `json:"payments"`
	Schedule     []ScheduleRow      `json:"schedule"`
	History      []ScheduleNote     `json:"schedule_history"`
	OfferedPlans []string           `json:"offered_plans"`
	AuthLimit    int64              `json:"auto_authorize_limit"`
	PIF          map[string]PIFView `json:"pif"`
}

func (st *State) termRow(tm *Term) TermRow {
	l := st.termLedger(tm)
	return TermRow{ID: tm.ID, Policy: tm.PolicyNumber, Reference: strings.TrimSpace(tm.PolicyNumber + " " + tm.Product), Effective: tm.Effective, Expiration: tm.Expiration, Status: tm.Status, CurrentDue: l.CurrentDue, PastDue: l.PastDue, Outstanding: l.Outstanding, OnHold: st.holdActive(tm)}
}

func (st *State) scheduleOf(terms []*Term) []ScheduleRow {
	rows := []ScheduleRow{}
	for _, tm := range terms {
		for _, in := range tm.Installments {
			r := ScheduleRow{ID: in.ID, InstallDate: in.InstallDate, DueDate: in.DueDate, Term: termLabel(tm), TermID: tm.ID, Invoiced: in.InvoiceID != "", Items: []ScheduleItem{}}
			for _, it := range in.Items {
				r.Items = append(r.Items, ScheduleItem{it.Recv, it.Trans, it.Amount, it.Cash, it.Credit, it.WriteOff, it.Redist, it.Balance()})
			}
			rows = append(rows, r)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].DueDate < rows[j].DueDate })
	return rows
}

func (st *State) holdView(h *Hold) HoldView {
	v := HoldView{Hold: h}
	what := "account has"
	if tm := st.Terms[h.TermID]; tm != nil {
		v.Term = termLabel(tm)
		what = "policy term " + v.Term + " has"
	}
	v.Banner = "The " + what + " been placed on hold. The " + h.Type + " is due to " + h.Reason + " beginning on " + US(h.Start)
	return v
}

func (s *Store) Account(number string) (*AccountView, error) {
	var v *AccountView
	s.read(func(st *State) {
		a := st.Accounts[number]
		if a == nil {
			return
		}
		terms := st.termsOf(number)
		v = &AccountView{Account: a, Name: st.partyName(a.PayorID), SystemDate: st.SystemDate, Payor: st.card(a.PayorID), Payee: st.card(a.PayeeID), Ledger: st.accountLedger(number), Terms: []TermRow{}, Activities: []*Activity{}, Holds: []HoldView{}, HoldEvents: []*HoldEvent{}, Suspense: st.suspenseOf(number), Disburse: []*Disbursement{}, Tasks: []*Task{}, Payments: []*Payment{}, Schedule: st.scheduleOf(terms), History: []ScheduleNote{}, OfferedPlans: st.Settings.OfferedPlans, AuthLimit: st.Settings.AutoAuthorizeLimit, PIF: map[string]PIFView{}}
		for _, tm := range terms {
			v.Terms = append(v.Terms, st.termRow(tm))
			v.History = append(v.History, tm.History...)
			v.PIF[tm.ID] = st.pif(tm)
		}
		for i := len(st.Activities) - 1; i >= 0; i-- {
			if st.Activities[i].AccountNumber == number {
				v.Activities = append(v.Activities, st.Activities[i])
			}
		}
		sort.SliceStable(v.Activities, func(i, j int) bool { return v.Activities[i].At > v.Activities[j].At })
		for _, inv := range st.Invoices {
			if inv.AccountNumber == number && (v.LastInvoice == nil || inv.InvoiceDate > v.LastInvoice.InvoiceDate) {
				v.LastInvoice = &InvoiceTile{inv.InvoiceDate, inv.Amount, inv.DueDate}
			}
		}
		for _, r := range v.Schedule {
			if !r.Invoiced {
				var amt int64
				for _, it := range r.Items {
					amt += it.Balance
				}
				if amt > 0 && (v.NextInvoice == nil || r.InstallDate < v.NextInvoice.InvoiceDate) {
					v.NextInvoice = &InvoiceTile{r.InstallDate, amt, r.DueDate}
				}
			}
		}
		for _, p := range st.Payments {
			if p.AccountNumber != number || p.Unidentified {
				continue
			}
			v.Payments = append(v.Payments, p)
			if p.Status == "Active" && (v.LastPayment == nil || p.CreatedAt > v.LastPayment.Date) {
				v.LastPayment = &PaymentTile{p.ID, p.CreatedAt, p.Amount, p.Method}
			}
		}
		if v.LastPayment != nil {
			v.LastPayment.Date = st.Payments[v.LastPayment.ID].EntryDate
		}
		sort.Slice(v.Payments, func(i, j int) bool { return v.Payments[i].CreatedAt < v.Payments[j].CreatedAt })
		for _, h := range st.Holds {
			if h.AccountNumber == number && h.Status == "Active" && (h.End == "" || h.End >= st.SystemDate) {
				v.Holds = append(v.Holds, st.holdView(h))
			}
		}
		sort.Slice(v.Holds, func(i, j int) bool { return v.Holds[i].ID < v.Holds[j].ID })
		for _, e := range st.HoldEvents {
			if e.End == "" || e.End >= st.SystemDate {
				v.HoldEvents = append(v.HoldEvents, e)
			}
		}
		sort.Slice(v.HoldEvents, func(i, j int) bool { return v.HoldEvents[i].ID < v.HoldEvents[j].ID })
		for _, d := range st.Disbursements {
			if d.AccountNumber == number {
				v.Disburse = append(v.Disburse, d)
			}
		}
		sort.Slice(v.Disburse, func(i, j int) bool { return v.Disburse[i].ID < v.Disburse[j].ID })
		for _, t := range st.Tasks {
			if t.AccountNumber == number {
				v.Tasks = append(v.Tasks, t)
			}
		}
		sort.Slice(v.Tasks, func(i, j int) bool { return v.Tasks[i].Created < v.Tasks[j].Created })
	})
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

type TermView struct {
	Term             *Term        `json:"term"`
	AccountNumber    string       `json:"account_number"`
	AccountName      string       `json:"account_name"`
	AccountStatus    string       `json:"account_status"`
	Version          int          `json:"version"` // the account's version
	SystemDate       string       `json:"system_date"`
	Ledger           TermLedger   `json:"ledger"`
	LastInvoice      *InvoiceTile `json:"last_invoice"`
	NextInstallment  *InvoiceTile `json:"next_installment"`
	OtherTerms       []TermRow    `json:"other_terms"`
	CollectionsBlock string       `json:"collections_block"`
	PIF              PIFView      `json:"pif"`
	Hold             *HoldView    `json:"hold"`
	HoldEvents       []*HoldEvent `json:"hold_events"`
	OfferedPlans     []string     `json:"offered_plans"`
	Commission       int64        `json:"potential_commission"`
}

func (s *Store) Term(id string) (*TermView, error) {
	var v *TermView
	s.read(func(st *State) {
		tm := st.Terms[id]
		if tm == nil {
			return
		}
		a := st.Accounts[tm.AccountNumber]
		l := st.termLedger(tm)
		v = &TermView{Term: tm, AccountNumber: a.Number, AccountName: st.partyName(a.PayorID), AccountStatus: a.Status, Version: a.Version, SystemDate: st.SystemDate, Ledger: l, OtherTerms: []TermRow{}, CollectionsBlock: CollectionsBlock(l), PIF: st.pif(tm), OfferedPlans: st.Settings.OfferedPlans, Commission: tm.PotentialCommission, HoldEvents: []*HoldEvent{}}
		for _, inv := range st.Invoices {
			if inv.TermID == tm.ID && (v.LastInvoice == nil || inv.InvoiceDate > v.LastInvoice.InvoiceDate) {
				v.LastInvoice = &InvoiceTile{inv.InvoiceDate, inv.Amount, inv.DueDate}
			}
		}
		for _, in := range tm.Installments {
			if in.DueDate >= st.SystemDate && (v.NextInstallment == nil || in.DueDate < v.NextInstallment.DueDate) {
				v.NextInstallment = &InvoiceTile{in.InstallDate, in.Balance(), in.DueDate}
			}
		}
		for _, o := range st.Terms {
			if o.PolicyNumber == tm.PolicyNumber {
				v.OtherTerms = append(v.OtherTerms, st.termRow(o))
			}
		}
		sort.Slice(v.OtherTerms, func(i, j int) bool { return v.OtherTerms[i].Effective < v.OtherTerms[j].Effective })
		if h := st.activeHold(tm.AccountNumber, tm.ID); h != nil {
			hv := st.holdView(h)
			v.Hold = &hv
		}
	})
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

type PaymentView struct {
	*Payment
	AccountName    string `json:"account_name"`
	IdentifierType string `json:"identifier_type"`
	Identifier     string `json:"identifier"`
	SystemDate     string `json:"system_date"`
}

func (s *Store) Payment(id string) (*PaymentView, error) {
	var v *PaymentView
	s.read(func(st *State) {
		p := st.Payments[id]
		if p == nil {
			return
		}
		v = &PaymentView{Payment: p, SystemDate: st.SystemDate}
		if a := st.Accounts[p.AccountNumber]; a != nil {
			v.AccountName = st.partyName(a.PayorID)
		}
		v.IdentifierType, v.Identifier = IdentifierOf(p.Identity)
	})
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

func (s *Store) PaymentViews(ps []*Payment) []PaymentView {
	out := []PaymentView{}
	s.read(func(st *State) {
		for _, p := range ps {
			v := PaymentView{Payment: p, SystemDate: st.SystemDate}
			if a := st.Accounts[p.AccountNumber]; a != nil {
				v.AccountName = st.partyName(a.PayorID)
			}
			v.IdentifierType, v.Identifier = IdentifierOf(p.Identity)
			out = append(out, v)
		}
	})
	return out
}

// Unidentified lists the unidentified payments, newest first.
func (s *Store) Unidentified() []PaymentView {
	ps := []*Payment{}
	s.read(func(st *State) {
		for _, p := range st.Payments {
			if p.Unidentified && p.Status == "Active" {
				ps = append(ps, p)
			}
		}
	})
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].EntryDate != ps[j].EntryDate {
			return ps[i].EntryDate > ps[j].EntryDate
		}
		return ps[i].ID < ps[j].ID
	})
	return s.PaymentViews(ps)
}

type PartyRowView struct {
	*PartyRow
	Current bool `json:"current"`
}

type PartyView struct {
	ID             string                    `json:"id"`
	Number         int                       `json:"number"`
	Version        int                       `json:"version"`
	Kind           string                    `json:"kind"`
	AssociatedWith string                    `json:"associated_with"`
	Name           string                    `json:"name"`
	Rows           map[string][]PartyRowView `json:"rows"`
	Demographics   map[string]string         `json:"demographics"`
	SystemDate     string                    `json:"system_date"`
	Accounts       []string                  `json:"accounts"`
}

func (s *Store) Party(id string) (*PartyView, error) {
	var v *PartyView
	s.read(func(st *State) {
		p := st.Parties[id]
		if p == nil {
			return
		}
		v = &PartyView{ID: p.ID, Number: p.Number, Version: p.Version, Kind: p.Kind, AssociatedWith: p.AssociatedWith, Name: p.DisplayName(st.SystemDate), Rows: map[string][]PartyRowView{}, Demographics: p.Demographics, SystemDate: st.SystemDate, Accounts: []string{}}
		for kind := range rowSpecs {
			v.Rows[kind] = []PartyRowView{}
			for _, r := range p.Rows[kind] {
				v.Rows[kind] = append(v.Rows[kind], PartyRowView{r, r.current(st.SystemDate)})
			}
		}
		for _, a := range st.Accounts {
			if a.PayorID == id || a.PayeeID == id {
				v.Accounts = append(v.Accounts, a.Number)
			}
		}
		sort.Strings(v.Accounts)
	})
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

func sortedValues[T any](m map[string]*T, less func(a, b *T) bool) []*T {
	out := make([]*T, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return less(out[i], out[j]) })
	return out
}

func (s *Store) Batches() (out []*Batch) {
	s.read(func(st *State) {
		out = sortedValues(st.Batches, func(a, b *Batch) bool {
			return a.CreatedAt < b.CreatedAt || (a.CreatedAt == b.CreatedAt && a.Number < b.Number)
		})
	})
	return
}

func (s *Store) Errors() (out []*ErrorItem) {
	s.read(func(st *State) { out = sortedValues(st.Errors, func(a, b *ErrorItem) bool { return a.At < b.At }) })
	return
}

type HoldEventView struct {
	*HoldEvent
	Accounts []string `json:"accounts"`
}

func (s *Store) HoldEvents() []HoldEventView {
	out := []HoldEventView{}
	s.read(func(st *State) {
		for _, e := range sortedValues(st.HoldEvents, func(a, b *HoldEvent) bool { return a.Start < b.Start }) {
			v := HoldEventView{HoldEvent: e, Accounts: []string{}}
			for _, h := range st.Holds {
				if h.EventID == e.ID {
					v.Accounts = append(v.Accounts, h.AccountNumber)
				}
			}
			sort.Strings(v.Accounts)
			out = append(out, v)
		}
	})
	return out
}

func (s *Store) Statements() (out []*Statement) {
	s.read(func(st *State) { out = sortedValues(st.Statements, func(a, b *Statement) bool { return a.ID < b.ID }) })
	return
}
