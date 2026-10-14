package billing

import (
	"sort"
)

// TermLedger is the Term Ledger panel.
type TermLedger struct {
	Outstanding int64 `json:"outstanding"`
	CurrentDue  int64 `json:"current_due"`
	PastDue     int64 `json:"past_due"`
	Receivable  int64 `json:"receivable"`
	Paid        int64 `json:"paid"`
	WrittenOff  int64 `json:"written_off"`
}

// AccountLedger is the Account Ledger panel.
type AccountLedger struct {
	Outstanding          int64 `json:"outstanding"`
	CurrentDue           int64 `json:"current_due"`
	PastDue              int64 `json:"past_due"`
	InSuspense           int64 `json:"in_suspense"`
	PendingDisbursements int64 `json:"pending_disbursements"`
}

func (st *State) termLedger(tm *Term) TermLedger {
	var l TermLedger
	for _, r := range tm.Receivables {
		l.Receivable += r.Amount
	}
	for _, in := range tm.Installments {
		b := in.Balance()
		l.Outstanding += b
		switch {
		case in.DueDate < st.SystemDate:
			l.PastDue += b
		case in.InvoiceID != "":
			l.CurrentDue += b
		}
		for _, it := range in.Items {
			l.Paid += it.Cash
			l.WrittenOff += it.WriteOff + it.Credit
		}
	}
	return l
}

func (st *State) accountLedger(number string) AccountLedger {
	var l AccountLedger
	for _, tm := range st.termsOf(number) {
		tl := st.termLedger(tm)
		l.Outstanding += tl.Outstanding
		l.CurrentDue += tl.CurrentDue
		l.PastDue += tl.PastDue
	}
	for _, s := range st.Suspense {
		if s.AccountNumber == number {
			l.InSuspense += s.Amount
		}
	}
	for _, d := range st.Disbursements {
		if d.AccountNumber == number && (d.Status == "Pending Authorization" || d.Status == "Authorized") {
			l.PendingDisbursements += d.Amount
		}
	}
	return l
}

type openItem struct {
	term *Term
	inst *Installment
	item *InstallItem
}

// openItems lists unpaid scheduled items, oldest due date first, keeping the
// installment's item order (fees before premium).
func openItems(terms []*Term) []openItem {
	out := []openItem{}
	for _, tm := range terms {
		for _, in := range tm.Installments {
			for _, it := range in.Items {
				if it.Balance() > 0 {
					out = append(out, openItem{tm, in, it})
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].inst.DueDate < out[j].inst.DueDate })
	return out
}

// applyCash pays open items of the terms with amount from the payment and
// returns what could not be applied.
func (t *tx) applyCash(p *Payment, terms []*Term, amount int64) int64 {
	for _, o := range openItems(terms) {
		if amount == 0 {
			break
		}
		pay := min(amount, o.item.Balance())
		o.item.Cash += pay
		amount -= pay
		p.Allocations = append(p.Allocations, &Allocation{ID: t.next("AL"), AccountNumber: o.term.AccountNumber, TermID: o.term.ID, Direction: termLabelDash(o.term), Date: t.SystemDate, Status: "Active", Type: "Scheduled Item", Amount: pay, InstallmentID: o.inst.ID, ItemID: o.item.ID})
	}
	return amount
}

// termLabelDash is the Direction text in Payment Details: policy and term dates.
func termLabelDash(tm *Term) string {
	return tm.PolicyNumber + " " + US(tm.Effective) + " - " + US(tm.Expiration)
}

// toSuspense puts an unallocated amount on hold in suspense at the account level.
func (t *tx) toSuspense(p *Payment, acct string, amount int64) *SuspenseItem {
	s := &SuspenseItem{ID: t.next("SUS"), AccountNumber: acct, PaymentID: p.ID, PaymentSource: p.Method, RefundSource: "Check", Date: t.SystemDate, Reason: "Overpayment", Original: amount, Amount: amount, Status: "No Hold"}
	t.Suspense[s.ID] = s
	p.Allocations = append(p.Allocations, &Allocation{ID: t.next("AL"), AccountNumber: acct, Direction: acct + " Account Level", Date: t.SystemDate, Status: "Active", Type: "Suspense", Amount: amount, SuspenseID: s.ID})
	return s
}

// findItem locates a scheduled item by id.
func (st *State) findItem(termID, itemID string) *InstallItem {
	tm := st.Terms[termID]
	if tm == nil {
		return nil
	}
	for _, in := range tm.Installments {
		for _, it := range in.Items {
			if it.ID == itemID {
				return it
			}
		}
	}
	return nil
}

// holdActive reports whether billing of the term is on hold today (an account
// hold or a hold on the term).
func (st *State) holdActive(tm *Term) bool {
	for _, h := range st.Holds {
		if h.Status != "Active" || h.AccountNumber != tm.AccountNumber || (h.TermID != "" && h.TermID != tm.ID) {
			continue
		}
		if h.Start <= st.SystemDate && (h.End == "" || h.End >= st.SystemDate) {
			return true
		}
	}
	return false
}

func (st *State) activeHold(acct, termID string) *Hold {
	for _, h := range st.Holds {
		if h.Status == "Active" && h.AccountNumber == acct && h.TermID == termID && (h.End == "" || h.End >= st.SystemDate) {
			return h
		}
	}
	return nil
}

// invoice bills an installment.
func (t *tx) invoice(tm *Term, in *Installment, by string) *Invoice {
	inv := &Invoice{ID: t.next("INV"), AccountNumber: tm.AccountNumber, TermID: tm.ID, InstallmentID: in.ID, InvoiceDate: t.SystemDate, DueDate: in.DueDate, Amount: in.Balance()}
	t.Invoices[inv.ID] = inv
	in.InvoiceID = inv.ID
	t.activity(tm.AccountNumber, "Invoice Generated", by, cents(inv.Amount), "Invoice for "+USD(inv.Amount)+" generated for policy "+tm.PolicyNumber+". Due date: "+US(in.DueDate))
	return inv
}

// generateSchedule spreads receivables over the plan's installments. Fees are
// billed with the first installment; invoices go out 20 days before each due
// date, as the Policy Term Summary shows (invoice 1/4, due 1/24).
func generateSchedule(t *tx, tm *Term, rec []*Receivable, firstDue string) {
	n, splits := planSplits(tm.PaymentPlan)
	var premium []*Receivable
	var fees []*Receivable
	for _, r := range rec {
		if r.Receivable == "Premium" && !r.BillInFull {
			premium = append(premium, r)
		} else {
			fees = append(fees, r)
		}
	}
	step := 12 / n
	for i := 0; i < n; i++ {
		due := addMonths(firstDue, i*step)
		in := &Installment{ID: t.next("IN"), InstallDate: addDays(due, -20), DueDate: due}
		if i == 0 {
			for _, r := range fees {
				in.Items = append(in.Items, &InstallItem{ID: t.next("IT"), ReceivableID: r.ID, Recv: "FEE", Trans: transCode(r.Transaction), Amount: r.Amount})
			}
		}
		for _, r := range premium {
			share := r.Amount * splits[i] / 10000
			if i == n-1 {
				share = r.Amount
				for j := 0; j < n-1; j++ {
					share -= r.Amount * splits[j] / 10000
				}
			}
			if share > 0 {
				in.Items = append(in.Items, &InstallItem{ID: t.next("IT"), ReceivableID: r.ID, Recv: "PREM", Trans: transCode(r.Transaction), Amount: share})
			}
		}
		if len(in.Items) > 0 {
			tm.Installments = append(tm.Installments, in)
		}
	}
}

// planSplits returns the number of installments and each one's share of the
// premium in basis points.
func planSplits(plan string) (int, []int64) {
	switch plan {
	case "Semi-Annual (55% Down)", "Semi-Annual":
		return 2, []int64{5500, 4500}
	case "Quarterly":
		return 4, []int64{2500, 2500, 2500, 2500}
	case "Monthly":
		s := make([]int64, 12)
		for i := range s {
			s[i] = 833
		}
		return 12, s
	default:
		return 1, []int64{10000}
	}
}

func transCode(transaction string) string {
	if transaction == "Renewal" {
		return "RENW"
	}
	return "NBUS"
}
