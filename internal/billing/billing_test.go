package billing

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ankitkumarflarre/datamigration/internal/quote"
)

func open(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetSystemDate(1, "2027-01-01"); err != nil {
		t.Fatal(err)
	}
	return s, dir
}

// fixture creates account A1 with one term: premium 1,805.00 and fees 25.00
// and 2.00 billed on one installment due 1/24/2027 (the SFLH3074412 shape).
func fixture(t *testing.T, s *Store, plan string) (acct string, term string) {
	t.Helper()
	err := s.update(func(tx *tx) error {
		p := tx.newParty("Person", map[string]string{"first": "Fix", "last": "Ture"}, tx.SystemDate)
		tx.addRow(p, "address", tx.SystemDate, addr("Legal", "1 Test St", "Tampa", "FL", "33601"))
		a := tx.newAccount("A1", p.ID, p.ID, "SPD9", 1)
		tm := &Term{PolicyNumber: "A1", Product: "Homeowners", Effective: "2027-01-01", Expiration: "2028-01-01", Status: "Active", PaymentPlan: plan, RenewalPlan: plan}
		tx.issueTerm(a, tm, []*Receivable{rcv("2027-01-01", "Premium", "", 180500, tm), rcv("2027-01-01", "Fee", "MGA", 2500, tm), rcv("2027-01-01", "Fee", "EMPA", 200, tm)}, "2027-01-24")
		acct, term = a.Number, tm.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func version(t *testing.T, s *Store, acct string) int {
	t.Helper()
	v, err := s.Account(acct)
	if err != nil {
		t.Fatal(err)
	}
	return v.Account.Version
}

func ledger(t *testing.T, s *Store, acct string) AccountLedger {
	v, err := s.Account(acct)
	if err != nil {
		t.Fatal(err)
	}
	return v.Ledger
}

func lastActivity(t *testing.T, s *Store, acct, typ string) *Activity {
	t.Helper()
	v, _ := s.Account(acct)
	for _, a := range v.Activities {
		if a.Type == typ {
			return a
		}
	}
	t.Fatalf("no %s activity", typ)
	return nil
}

func TestPaymentAllocationAndSuspenseInCents(t *testing.T) {
	cases := []struct {
		name        string
		in          PaymentInput
		outstanding int64
		suspense    int64
		allocations int
	}{
		{"pay to account, exact", PaymentInput{Method: "Check", Amount: 183200}, 0, 0, 3},
		{"pay to account, partial pays fees first", PaymentInput{Method: "Check", Amount: 2701}, 180499, 0, 3},
		{"overpayment goes to suspense", PaymentInput{Method: "Check", Amount: 183201}, 0, 1, 4},
		{"specific policies with remainder", PaymentInput{Method: "Check", Amount: 100000, Allocation: "policies", Allocations: []TermAllocation{{TermID: "TERM", Amount: 60000}, {Amount: 15000}}}, 123200, 40000, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := open(t)
			acct, term := fixture(t, s, "Full Pay")
			for i := range c.in.Allocations {
				if c.in.Allocations[i].TermID == "TERM" {
					c.in.Allocations[i].TermID = term
				}
			}
			c.in.Version = version(t, s, acct)
			res, err := s.MakePayment(acct, c.in)
			if err != nil {
				t.Fatal(err)
			}
			l := ledger(t, s, acct)
			if l.Outstanding != c.outstanding || l.InSuspense != c.suspense {
				t.Fatalf("outstanding %d suspense %d, want %d %d", l.Outstanding, l.InSuspense, c.outstanding, c.suspense)
			}
			if len(res.Payment.Allocations) != c.allocations {
				t.Fatalf("%d allocations, want %d", len(res.Payment.Allocations), c.allocations)
			}
			var sum int64
			for _, a := range res.Payment.Allocations {
				sum += a.Amount
			}
			if sum != c.in.Amount {
				t.Fatalf("allocations sum %d, want %d", sum, c.in.Amount)
			}
			if res.Message != "Payment for "+USD(c.in.Amount)+" entered for Account A1." {
				t.Fatalf("message %q", res.Message)
			}
		})
	}
}

func TestPaymentValidation(t *testing.T) {
	s, _ := open(t)
	acct, term := fixture(t, s, "Full Pay")
	cases := []struct {
		in    PaymentInput
		field string
	}{
		{PaymentInput{Method: "", Amount: 100}, "method"},
		{PaymentInput{Method: "Check", Amount: 0}, "amount"},
		{PaymentInput{Method: "Check", Amount: 100, EntryDate: "2027-01-02"}, "entry_date"},
		{PaymentInput{Method: "Check", Amount: 100, Allocation: "policies", Allocations: []TermAllocation{{TermID: term, Amount: 200}}}, "allocations"},
		{PaymentInput{Method: "Check", Amount: 999999999, Allocation: "policies", Allocations: []TermAllocation{{TermID: term, Amount: 183201}}}, "allocations.0"},
		{PaymentInput{Method: "Credit/Debit Card", Amount: 1000}, "card.number"},
	}
	for _, c := range cases {
		c.in.Version = version(t, s, acct)
		_, err := s.MakePayment(acct, c.in)
		var v *ValidationError
		if !errors.As(err, &v) || v.Fields[c.field] == "" {
			t.Errorf("%+v: got %v, want field %s", c.in, err, c.field)
		}
	}
}

func TestCardPaymentPostsBaseOnlyAndKeepsLast4(t *testing.T) {
	s, dir := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	in := PaymentInput{Version: version(t, s, acct), Method: "Credit/Debit Card", Amount: 1000, Card: &CardInput{Number: "4111111111111111", Name: "Test", Month: "12", Year: "2030", CVC: "123", AcceptedTerms: true}}
	if _, err := s.MakePayment(acct, in); err == nil || !strings.Contains(err.(*ValidationError).Fields["card.number"], "Simulated gateway") {
		t.Fatalf("non-test card: %v", err)
	}
	in.Card.Number = "4012 0020 0006 0016"
	in.Card.AcceptedTerms = false
	if _, err := s.MakePayment(acct, in); err == nil {
		t.Fatal("terms not accepted must be refused")
	}
	in.Card.AcceptedTerms = true
	res, err := s.MakePayment(acct, in)
	if err != nil {
		t.Fatal(err)
	}
	if res.Receipt.Fee != 27 || res.Receipt.Total != 1027 || res.Payment.Amount != 1000 || res.Payment.CardLast4 != "0016" {
		t.Fatalf("receipt %+v payment %+v", res.Receipt, res.Payment)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "billing.json"))
	if bytes.Contains(b, []byte("4012002000060016")) || bytes.Contains(b, []byte(`"123"`)) {
		t.Fatal("full card number or CVV was stored")
	}
	if a := lastActivity(t, s, acct, "Payment Received"); a.Text != "Credit/Debit Card payment for $10.00 received. Receipt date: 01/01/2027" {
		t.Fatalf("activity %q", a.Text)
	}
}

func TestReversalReopensWhatThePaymentPaid(t *testing.T) {
	s, _ := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	res, err := s.MakePayment(acct, PaymentInput{Version: version(t, s, acct), Method: "Check", Amount: 200000})
	if err != nil {
		t.Fatal(err)
	}
	if l := ledger(t, s, acct); l.Outstanding != 0 || l.InSuspense != 16800 {
		t.Fatalf("after payment %+v", l)
	}
	if _, err := s.ReversePayment(res.Payment.ID, res.Payment.Version, "bad"); err == nil {
		t.Fatal("reason must be from the list")
	}
	p, err := s.ReversePayment(res.Payment.ID, res.Payment.Version, "RP-Reverse Payment")
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != "Reversed" {
		t.Fatal(p.Status)
	}
	if l := ledger(t, s, acct); l.Outstanding != 183200 || l.InSuspense != 0 {
		t.Fatalf("after reversal %+v", l)
	}
	if a := lastActivity(t, s, acct, "Reverse Payment"); a.Text != "$2,000.00 Payment was reversed. Reason: Reverse Payment" {
		t.Fatal(a.Text)
	}
	if _, err := s.ReversePayment(p.ID, p.Version, "RP-Reverse Payment"); err == nil {
		t.Fatal("second reversal must be refused")
	}
}

func TestReversalBlockedWhenSuspenseWasDisbursed(t *testing.T) {
	s, _ := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	res, _ := s.MakePayment(acct, PaymentInput{Version: version(t, s, acct), Method: "Check", Amount: 184200})
	v, _ := s.Account(acct)
	if _, _, err := s.Disburse(acct, DisburseInput{Version: v.Account.Version, SuspenseIDs: []string{v.Suspense[0].ID}, Amount: 1000, Reason: "Overpayment"}); err != nil {
		t.Fatal(err)
	}
	var r *RuleError
	if _, err := s.ReversePayment(res.Payment.ID, res.Payment.Version, "RP-Reverse Payment"); !errors.As(err, &r) {
		t.Fatalf("got %v", err)
	}
}

func TestDisbursementAuthorizationThreshold(t *testing.T) {
	for _, c := range []struct {
		amount int64
		status string
		by     string
	}{{1000, "Authorized", "System"}, {100000, "Authorized", "System"}, {100001, "Pending Authorization", ""}} {
		s, _ := open(t)
		acct, _ := fixture(t, s, "Full Pay")
		s.MakePayment(acct, PaymentInput{Version: version(t, s, acct), Method: "Credit/Debit Card", Amount: 183200 + c.amount, Card: &CardInput{Number: "4012002000060016", Name: "T", Month: "1", Year: "2031", CVC: "123", AcceptedTerms: true}})
		v, _ := s.Account(acct)
		if v.Suspense[0].PaymentSource != "Credit/Debit Card" || v.Suspense[0].RefundSource != "Check" {
			t.Fatalf("suspense %+v", v.Suspense[0])
		}
		_, err := func() (any, error) {
			_, _, err := s.Disburse(acct, DisburseInput{Version: v.Account.Version, SuspenseIDs: []string{v.Suspense[0].ID}, Amount: c.amount})
			return nil, err
		}()
		var ve *ValidationError
		if !errors.As(err, &ve) || ve.Fields["reason"] == "" {
			t.Fatalf("refund without a reason: %v", err)
		}
		d, msg, err := s.Disburse(acct, DisburseInput{Version: v.Account.Version, SuspenseIDs: []string{v.Suspense[0].ID}, Amount: c.amount, Reason: "Overpayment"})
		if err != nil {
			t.Fatal(err)
		}
		if d.Status != c.status || d.AuthorizedBy != c.by || d.Method != "Check" {
			t.Fatalf("%d: %+v", c.amount, d)
		}
		if msg != "Disbursement for "+USD(c.amount)+" was submitted for authorization. Payee: Fix Ture" {
			t.Fatal(msg)
		}
		l := ledger(t, s, acct)
		if l.PendingDisbursements != c.amount || l.InSuspense != 0 {
			t.Fatalf("ledger %+v", l)
		}
		if c.status == "Pending Authorization" {
			if _, err := s.DecideDisbursement(acct, d.ID, version(t, s, acct), false); err != nil {
				t.Fatal(err)
			}
			if l := ledger(t, s, acct); l.PendingDisbursements != 0 || l.InSuspense != c.amount {
				t.Fatalf("after rejection %+v", l)
			}
		}
	}
}

func TestHoldStartsOnSystemDateAndResumeAdjustsDueDates(t *testing.T) {
	for _, c := range []struct {
		option string
		due    string
	}{{"Adjust due dates", "2027-02-03"}, {"Do not adjust due dates", "2027-01-24"}} {
		s, _ := open(t)
		acct, term := fixture(t, s, "Full Pay")
		if _, _, err := s.PlaceHold(acct, HoldInput{Version: version(t, s, acct), Kind: "individual", Type: "Hold Follow Up", Reason: "Company request", End: "2026-12-31"}); err == nil {
			t.Fatal("end before start must be refused")
		}
		h, msg, err := s.PlaceHold(acct, HoldInput{Version: version(t, s, acct), Kind: "individual", Type: "Hold Follow Up", Reason: "Company request"})
		if err != nil {
			t.Fatal(err)
		}
		if h.Start != "2027-01-01" || msg != "Account hold set. Hold type: Hold Follow Up. Reason: Company request. Beginning: 1/1/2027 Ending: Indefinite" {
			t.Fatalf("%+v %q", h, msg)
		}
		if _, _, err := s.PlaceHold(acct, HoldInput{Version: version(t, s, acct), Kind: "individual", Type: "Hold Follow Up", Reason: "Company request"}); err == nil {
			t.Fatal("second active hold must be refused")
		}
		sys := s.System()
		s.SetSystemDate(sys.Version, "2027-01-11")
		sys = s.System()
		if r, _ := s.RunDueProcesses(sys.Version); r.Invoices != 0 {
			t.Fatal("held terms must not be invoiced")
		}
		if _, err := s.UpdateHold(acct, h.ID, HoldUpdateInput{Version: version(t, s, acct), Action: "resume", ResumeOption: c.option}); err != nil {
			t.Fatal(err)
		}
		tv, _ := s.Term(term)
		if got := tv.Term.Installments[0].DueDate; got != c.due {
			t.Fatalf("%s: due %s, want %s", c.option, got, c.due)
		}
	}
}

func TestCollectionsBlock(t *testing.T) {
	s, _ := open(t)
	acct, term := fixture(t, s, "Full Pay")
	if _, err := s.TransferToCollections(acct, term, version(t, s, acct)); err != nil {
		t.Fatalf("a term with a balance can go to collections: %v", err)
	}
	s2, _ := open(t)
	acct, term = fixture(t, s2, "Full Pay")
	s2.MakePayment(acct, PaymentInput{Version: version(t, s2, acct), Method: "Check", Amount: 183200})
	_, err := s2.TransferToCollections(acct, term, version(t, s2, acct))
	var r *RuleError
	if !errors.As(err, &r) || r.Message != "The term outstanding balance: $0.00 is not enough to make a transfer to collections." {
		t.Fatalf("got %v", err)
	}
}

func TestPaidInFullDiscount(t *testing.T) {
	s, _ := open(t)
	acct, term := fixture(t, s, "Full Pay")
	tv, _ := s.Term(term)
	if tv.PIF.Configured || tv.PIF.Message != "Cannot be calculated as the Paid In Full configuration has not been established" {
		t.Fatalf("%+v", tv.PIF)
	}
	if _, err := s.ApplyPIF(acct, term, PIFInput{Version: version(t, s, acct), Amount: 100, Reason: "Paid in Full Discount Request"}); err == nil {
		t.Fatal("PIF without a configuration must be refused")
	}
	sys := s.System()
	if err := s.SaveSettings(SettingsUpdate{Version: sys.Version, AutoAuthorizeLimit: 100000, OfferedPlans: []string{"Full Pay"}, PIF: &PIFConfig{DiscountBasisPoints: 500, EvaluationDays: 30}}); err != nil {
		t.Fatal(err)
	}
	tv, _ = s.Term(term)
	if tv.PIF.PotentialDiscount != 9025 {
		t.Fatalf("%+v", tv.PIF)
	}
	if _, err := s.ApplyPIF(acct, term, PIFInput{Version: version(t, s, acct), Amount: 9025, Reason: "Paid in Full Discount Request"}); err != nil {
		t.Fatal(err)
	}
	if l := ledger(t, s, acct); l.Outstanding != 183200-9025 {
		t.Fatalf("%+v", l)
	}
}

func TestPartyCorrectVersusReplace(t *testing.T) {
	s, _ := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	v, _ := s.Account(acct)
	pid := v.Payor.ID
	p, _ := s.Party(pid)
	row := p.Rows["address"][0]
	fields := map[string]string{"kind": "Legal", "country": "United States", "line1": "2 Test St", "city": "Tampa", "state": "FL", "zip": "33602"}

	sys := s.System()
	s.SetSystemDate(sys.Version, "2027-02-01")
	if _, err := s.SaveRow(pid, "address", RowInput{Version: p.Version, Mode: "correct", RowID: row.ID, Fields: fields}); err != nil {
		t.Fatal(err)
	}
	p, _ = s.Party(pid)
	if len(p.Rows["address"]) != 1 || p.Rows["address"][0].Effective != "2027-01-01" || p.Rows["address"][0].Fields["line1"] != "2 Test St" {
		t.Fatalf("correct: %+v", p.Rows["address"][0].PartyRow)
	}
	fields["line1"] = "3 Test St"
	if _, err := s.SaveRow(pid, "address", RowInput{Version: p.Version, Mode: "replace", RowID: row.ID, Fields: fields}); err != nil {
		t.Fatal(err)
	}
	p, _ = s.Party(pid)
	old, cur := p.Rows["address"][0], p.Rows["address"][1]
	if old.Expiration != "2027-02-01" || old.Current || old.Fields["line1"] != "2 Test St" || cur.Effective != "2027-02-01" || cur.Expiration != OpenEnded || !cur.Current {
		t.Fatalf("replace: %+v / %+v", old.PartyRow, cur.PartyRow)
	}
	if _, err := s.SaveRow(pid, "address", RowInput{Version: p.Version, Mode: "remove", RowID: cur.ID}); err == nil {
		t.Fatal("addresses have no Remove")
	}
	s.SaveRow(pid, "othername", RowInput{Version: p.Version, Mode: "add", Fields: map[string]string{"type": "Formerly Known As", "last": "OLD"}})
	p, _ = s.Party(pid)
	on := p.Rows["othername"][0]
	if _, err := s.SaveRow(pid, "othername", RowInput{Version: p.Version, Mode: "remove", RowID: on.ID}); err != nil {
		t.Fatal(err)
	}
	p, _ = s.Party(pid)
	if p.Rows["othername"][0].Current {
		t.Fatal("removed name is still current")
	}
	if _, err := s.SaveRow(pid, "phone", RowInput{Version: p.Version, Mode: "add", Fields: map[string]string{"type": "Home", "number": "5550100"}}); err == nil {
		t.Fatal("phone format must be checked")
	}
}

func TestUnidentifiedPayments(t *testing.T) {
	s, _ := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	_, err := s.EnterUnidentified(UnidentifiedInput{Method: "Check", Amount: 5000})
	var v *ValidationError
	if !errors.As(err, &v) || len(v.Fields) != 4 {
		t.Fatalf("no identifier: %v", err)
	}
	p, err := s.EnterUnidentified(UnidentifiedInput{Identity: Identity{First: "Only"}, Method: "Check", Amount: 5000})
	if err == nil {
		t.Fatal("first name alone is not an identifier")
	}
	p, err = s.EnterUnidentified(UnidentifiedInput{Identity: Identity{AccountRef: "ZZZ"}, Method: "Check", Amount: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if typ, id := IdentifierOf(p.Identity); typ != "Account" || id != "ZZZ" {
		t.Fatal(typ, id)
	}
	p, err = s.UpdateReference(p.ID, p.Version, Identity{Last: "Smith"})
	if err != nil || p.Identity.Last != "Smith" {
		t.Fatal(err)
	}
	if _, err := s.ApplyUnidentified(p.ID, p.Version, "NOPE"); err == nil {
		t.Fatal("unknown account")
	}
	res, err := s.ApplyUnidentified(p.ID, p.Version, acct)
	if err != nil {
		t.Fatal(err)
	}
	if res.Payment.Unidentified || ledger(t, s, acct).Outstanding != 183200-5000 || len(s.Unidentified()) != 0 {
		t.Fatal("not applied")
	}
}

func TestStoreSurvivesReopen(t *testing.T) {
	s, dir := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	s.MakePayment(acct, PaymentInput{Version: version(t, s, acct), Method: "Check", Amount: 100000, CheckNumber: "77"})
	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s2.System().Date != "2027-01-01" || ledger(t, s2, acct).Outstanding != 83200 {
		t.Fatal("state lost on reopen")
	}
	if a := lastActivity(t, s2, acct, "Payment Received"); a.Text != "Check payment for $1,000.00 received. Check Number: 77 Receipt date: 01/01/2027" {
		t.Fatal(a.Text)
	}
}

func TestDueProcessesAndDisbursementService(t *testing.T) {
	s, _ := open(t)
	acct, _ := fixture(t, s, "Semi-Annual (55% Down)")
	sys := s.System()
	s.SetSystemDate(sys.Version, "2027-01-04")
	sys = s.System()
	r, err := s.RunDueProcesses(sys.Version)
	if err != nil || r.Invoices != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	v, _ := s.Account(acct)
	if v.LastInvoice == nil || v.LastInvoice.Amount != 2700+99275 || v.NextInvoice.DueDate != "2027-07-24" {
		t.Fatalf("%+v %+v", v.LastInvoice, v.NextInvoice)
	}
	sys = s.System()
	if err := s.DisbursementService(sys.Version, "start"); err != nil {
		t.Fatal(err)
	}
	sys = s.System()
	var rule *RuleError
	if err := s.DisbursementService(sys.Version, "start"); !errors.As(err, &rule) {
		t.Fatalf("duplicate start: %v", err)
	}
	if err := s.DisbursementService(sys.Version-1, "stop"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale version: %v", err)
	}
}

func TestTermPlanChangeRespreadsOpenBalance(t *testing.T) {
	s, _ := open(t)
	acct, term := fixture(t, s, "Full Pay")
	sys := s.System()
	s.SaveSettings(SettingsUpdate{Version: sys.Version, AutoAuthorizeLimit: 1, OfferedPlans: []string{"Full Pay", "Semi-Annual (55% Down)"}})
	if err := s.SetTermPlan(acct, term, PlanInput{Version: version(t, s, acct), Plan: "Premium Finance Annual"}); err == nil {
		t.Fatal("plans not offered must be refused")
	}
	if err := s.SetTermPlan(acct, term, PlanInput{Version: version(t, s, acct), Plan: "Semi-Annual (55% Down)"}); err != nil {
		t.Fatal(err)
	}
	tv, _ := s.Term(term)
	if len(tv.Term.Installments) != 2 || tv.Ledger.Outstanding != 183200 {
		t.Fatalf("%d installments, %d outstanding", len(tv.Term.Installments), tv.Ledger.Outstanding)
	}
}

func TestErrorResubmitRunsTheJob(t *testing.T) {
	s, _ := open(t)
	if err := s.Seed(true); err != nil {
		t.Fatal(err)
	}
	var inv *ErrorItem
	for _, e := range s.Errors() {
		if e.Type == "Generate Invoice" {
			inv = e
		}
	}
	e, err := s.Resubmit(inv.ID, ResubmitInput{Version: inv.Version, Sections: inv.Sections, InvoiceID: inv.InvoiceID, AsyncID: inv.AsyncID})
	if err != nil || e.Status != "Open" || e.Description != "Account SFLH3070506 was not found." {
		t.Fatalf("%+v %v", e, err)
	}
	sec := append([]PayloadSection{}, e.Sections...)
	sec[0].Value = strings.Replace(sec[0].Value, "SFLH3070506", "CLAD3057414", 1)
	e, err = s.Resubmit(e.ID, ResubmitInput{Version: e.Version, Sections: sec, InvoiceID: e.InvoiceID, AsyncID: e.AsyncID})
	if err != nil || e.Status != "Resolved" {
		t.Fatalf("%+v %v", e, err)
	}
}

func TestSeedReproducesScreenshots(t *testing.T) {
	s, _ := open(t)
	if err := s.Seed(false); err == nil {
		// the store from open() has no records yet, so the seed loads
	} else {
		t.Fatal(err)
	}
	if err := s.Seed(false); err == nil {
		t.Fatal("seeding a store with data must be refused")
	}
	if s.System().Date != SeedDate {
		t.Fatal(s.System().Date)
	}
	v, _ := s.Account("CLAD3057414")
	if v.Ledger.Outstanding != 98100 {
		t.Fatalf("CLAD3057414 outstanding %d", v.Ledger.Outstanding)
	}
	fl, _ := s.Account("SFLH3074412")
	if fl.Ledger.InSuspense != 1000 || fl.Ledger.Outstanding != 0 || fl.Account.AccountClass != "SPD9" || fl.Payee.Name != "TEST" {
		t.Fatalf("SFLH3074412 %+v", fl.Ledger)
	}
	tx, _ := s.Account("STXH3074413")
	if tx.Payor.Number != 95371 {
		t.Fatal(tx.Payor.Number)
	}
	pays, _ := s.SearchPayments(PaymentQuery{Account: "CLAD3057412", AmountFrom: -1, AmountTo: -1})
	if len(pays) != 1 || len(pays[0].Allocations) != 7 {
		t.Fatal("the $722 payment should split over seven items")
	}
	for _, c := range []struct{ report, want string }{
		{"daily-cash-journal", "722.00"},
		{"cash-in-suspense", "$237,497.63"},
		{"aged-receivables", "4,275.00"},
		{"disbursements-pending-authorization", "242,914.00"},
		{"unidentified-cash", "8,504.00"},
	} {
		r, err := s.Report(c.report)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(r)
		if !strings.Contains(string(b), c.want) {
			t.Errorf("%s lacks %s", c.report, c.want)
		}
	}
	r, _ := s.Report("cash-in-suspense")
	if !strings.Contains(strings.Join(r.Sections[0].Rows[1], "|"), "983") {
		t.Errorf("days pending: %v", r.Sections[0].Rows[1])
	}
	b, _ := json.Marshal(s.st)
	for _, real := range []string{"wipro", "Geethika", "Janani", "Baronne", "Doubloon", "Vega Creek"} {
		if strings.Contains(string(b), real) {
			t.Errorf("seed contains real-looking data %q", real)
		}
	}
}

func TestAccountFromQuote(t *testing.T) {
	s, _ := open(t)
	qs, _ := quote.Open(t.TempDir())
	q := &quote.Quote{ID: "q1", Number: "Q-ABC", Status: "draft", Values: map[string]any{"AccountInput.FirstName": "Ann", "AccountInput.LastName": "Test", "PolicyInput.EffectiveDate": "2027-02-01", "Billing.PaymentPlan": "Quarterly", "Billing.InvoiceDueDate": "2027-02-15"}}
	_ = qs
	if _, err := s.AccountFromQuote(q); err == nil {
		t.Fatal("draft quotes have no billing account")
	}
	q.Status = "ready_for_rating"
	a, err := s.AccountFromQuote(q)
	if err != nil {
		t.Fatal(err)
	}
	if a.Number != "Q-ABC" {
		t.Fatal(a.Number)
	}
	v, _ := s.Account(a.Number)
	if len(v.Schedule) != 4 || v.Schedule[0].DueDate != "2027-02-15" || v.Terms[0].Outstanding != 253800 {
		t.Fatalf("%d installments %+v", len(v.Schedule), v.Terms[0])
	}
	if _, err := s.AccountFromQuote(q); err == nil {
		t.Fatal("one account per policy")
	}
}

func TestHTTPStatusCodes(t *testing.T) {
	s, _ := open(t)
	acct, _ := fixture(t, s, "Full Pay")
	mux := http.NewServeMux()
	Register(mux, s)
	do := func(method, path, body string) (int, map[string]any) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	v := version(t, s, acct)
	if code, out := do("POST", "/api/billing/accounts/A1/payments", `{"version":`+itoa(v)+`,"method":"Check","amount":0}`); code != 422 || out["fields"].(map[string]any)["amount"] == nil {
		t.Fatalf("validation: %d %v", code, out)
	}
	if code, _ := do("POST", "/api/billing/accounts/A1/payments", `{"version":`+itoa(v-1)+`,"method":"Check","amount":100}`); code != 409 {
		t.Fatalf("stale version: %d", code)
	}
	if code, _ := do("POST", "/api/billing/accounts/A1/payments", `{"version":1,"bogus":1}`); code != 400 {
		t.Fatalf("unknown field: %d", code)
	}
	if code, _ := do("GET", "/api/billing/accounts/NOPE", ""); code != 404 {
		t.Fatalf("missing: %d", code)
	}
	if code, _ := do("POST", "/api/billing/accounts/A1/payments", `{"version":`+itoa(v)+`,"method":"Check","amount":100}`); code != 200 {
		t.Fatalf("payment: %d", code)
	}
	if code, out := do("GET", "/api/billing/payments", ""); code != 422 || out["fields"].(map[string]any)["query"] == nil {
		t.Fatalf("empty search: %d", code)
	}
	if code, _ := do("GET", "/api/billing/reports/aged-receivables", ""); code != 200 {
		t.Fatalf("report: %d", code)
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestCreateParty(t *testing.T) {
	s, _ := open(t)
	_, err := s.CreateParty(NewPartyInput{Kind: "Person", Name: map[string]string{"first": "A"}, Address: map[string]string{"line1": "1 St", "city": "X", "state": "ZZ", "zip": "1"}})
	var v *ValidationError
	if !errors.As(err, &v) || v.Fields["last"] == "" || v.Fields["state"] == "" || v.Fields["zip"] == "" {
		t.Fatalf("%v", err)
	}
	p, err := s.CreateParty(NewPartyInput{Kind: "Person", Name: map[string]string{"first": "Ann", "last": "Test"}, Address: map[string]string{"line1": "1 St", "city": "Tampa", "state": "FL", "zip": "33601"}})
	if err != nil || p.DisplayName("2027-01-01") != "Ann Test" || p.Rows["name"][0].Fields["correspondence"] != "Ann Test" {
		t.Fatalf("%+v %v", p, err)
	}
}
