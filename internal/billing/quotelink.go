package billing

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ankitkumarflarre/datamigration/internal/quote"
)

// Sample figures the quote's installment schedule shows. A quote is not rated,
// so the billing term created from it carries these and says so.
var quoteSampleReceivables = []struct {
	recv, sub string
	amount    int64
}{{"Premium", "", 226200}, {"Fee", "MGA", 2500}, {"Fee", "EMPA", 200}, {"Fee", "FIGA", 2300}, {"Fee", "Policy", 22600}}

// quotePlans maps the quote's payment plans to the billing plan names.
var quotePlans = map[string]string{"Annual": "Full Pay", "Semi-Annual": "Semi-Annual (55% Down)", "Quarterly": "Quarterly", "Monthly": "Monthly"}

func str(v map[string]any, k string) string {
	s, _ := v[k].(string)
	return strings.TrimSpace(s)
}

// AccountFromQuote creates the billing account and term for a submitted quote.
// The account number is the policy number (one account per policy).
func (s *Store) AccountFromQuote(q *quote.Quote) (*Account, error) {
	var out *Account
	err := s.update(func(t *tx) error {
		if q.Status != "ready_for_rating" {
			return rule("Submit the application for rating before creating its billing account.")
		}
		v := q.Values
		number := q.PolicyNumber()
		if t.Accounts[number] != nil {
			return rule("Billing account %s already exists for this policy.", number)
		}
		f := fieldErrors{}
		plan := quotePlans[str(v, "Billing.PaymentPlan")]
		if plan == "" {
			f.add("Billing.PaymentPlan", "Choose a payment plan on the Payment page first.")
		}
		eff, exp := str(v, "PolicyInput.EffectiveDate"), str(v, "PolicyInput.ExpirationDate")
		if !validDate(eff) {
			f.add("PolicyInput.EffectiveDate", "The quote has no effective date.")
		}
		if !validDate(exp) {
			exp = ""
			if validDate(eff) {
				exp = addMonths(eff, 12)
			}
		}
		if str(v, "AccountInput.LastName") == "" {
			f.add("AccountInput.LastName", "The quote has no applicant name.")
		}
		if err := f.err(); err != nil {
			return err
		}
		due := str(v, "Billing.InvoiceDueDate")
		if !validDate(due) {
			due = eff
		}
		p := t.newParty("Person", map[string]string{"first": str(v, "AccountInput.FirstName"), "middle": str(v, "AccountInput.MiddleName"), "last": str(v, "AccountInput.LastName")}, t.SystemDate)
		if line1, state := str(v, "AccountInput.Address1"), str(v, "AccountInput.State"); line1 != "" && oneOf(Codes.States, state) {
			a := addr("Legal", line1, str(v, "AccountInput.City"), state, str(v, "AccountInput.ZipCode"))
			a["line2"], a["county"] = str(v, "AccountInput.Address2"), str(v, "AccountInput.County")
			t.addRow(p, "address", t.SystemDate, a)
		}
		if e := str(v, "AccountInput.Email"); emailLike.MatchString(e) {
			t.addRow(p, "email", t.SystemDate, map[string]string{"type": "Home", "address": e, "primary": "Yes"})
		}
		acct := t.newAccount(number, p.ID, p.ID, Codes.AccountClasses[0], 1)
		acct.Source = "quote:" + q.ID
		product := "Homeowners"
		if form := str(v, "DwellingInput.Form"); strings.HasPrefix(form, "DP") {
			product = "Dwelling"
		}
		tm := &Term{PolicyNumber: number, Product: product, Effective: eff, Expiration: exp, Status: "Active", PaymentPlan: plan, RenewalPlan: plan, BillingFrequency: "ANNUAL", MasterCompany: "DCT", LineOfBusiness: str(v, "DwellingInput.Form"), EquityDate: exp,
			AdditionalData: []NameValue{{"Quote Number", q.Number}, {"Billed To", str(v, "Billing.BilledTo")}, {"Premium", "Sample figures: the quote is not rated"}}}
		rec := []*Receivable{}
		for _, r := range quoteSampleReceivables {
			rec = append(rec, rcv(eff, r.recv, r.sub, r.amount, tm))
		}
		t.issueTerm(acct, tm, rec, due)
		t.activity(acct.Number, "Account Created From Quote", CurrentUser, nil, fmt.Sprintf("Billing account created from quote %s. The receivables are the quote's sample figures; the quote is not rated.", q.Number))
		out = acct
		return nil
	})
	return out, err
}

// RegisterQuoteLink adds Create billing account for submitted quotes.
func RegisterQuoteLink(mux *http.ServeMux, s *Store, quotes *quote.Store) {
	handle(mux, "POST /api/billing/accounts/from-quote", func(r *http.Request, in struct {
		QuoteID string `json:"quote_id"`
	}) (any, error) {
		q, err := quotes.Get(in.QuoteID)
		if err != nil {
			return nil, ErrNotFound
		}
		return s.AccountFromQuote(q)
	})
	handle(mux, "GET /api/billing/quotes/{id}/account", func(r *http.Request, _ none) (any, error) {
		q, err := quotes.Get(r.PathValue("id"))
		if err != nil {
			return nil, ErrNotFound
		}
		var out map[string]any
		s.read(func(st *State) {
			if a := st.Accounts[q.PolicyNumber()]; a != nil {
				out = map[string]any{"number": a.Number, "exists": true}
			}
		})
		if out == nil {
			out = map[string]any{"number": q.PolicyNumber(), "exists": false}
		}
		return out, nil
	})
}
