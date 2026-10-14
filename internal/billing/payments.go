package billing

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type TermAllocation struct {
	TermID string `json:"term_id"` // empty: the Account Level row (suspense)
	Amount int64  `json:"amount"`
}

// CardInput is what the simulated gateway receives. Only the brand, the last
// four digits and the name on the card are kept.
type CardInput struct {
	Number        string `json:"number"`
	Name          string `json:"name"`
	Month         string `json:"month"`
	Year          string `json:"year"`
	CVC           string `json:"cvc"`
	Email         string `json:"email"`
	AcceptedTerms bool   `json:"accepted_terms"`
}

type PaymentInput struct {
	Version      int              `json:"version"`
	Method       string           `json:"method"`
	Amount       int64            `json:"amount"`
	EntryDate    string           `json:"entry_date"`
	CheckNumber  string           `json:"check_number"`
	NameOnCheck  string           `json:"name_on_check"`
	PostmarkDate string           `json:"postmark_date"`
	Allocation   string           `json:"allocation"` // account | policies
	Allocations  []TermAllocation `json:"allocations"`
	Card         *CardInput       `json:"card,omitempty"`
}

type Receipt struct {
	TransactionNumber string `json:"transaction_number"`
	Base              int64  `json:"base"`
	Fee               int64  `json:"fee"`
	Total             int64  `json:"total"`
	Email             string `json:"email,omitempty"`
}

type PaymentResult struct {
	Payment *Payment `json:"payment"`
	Message string   `json:"message"`
	Receipt *Receipt `json:"receipt,omitempty"`
}

var digits = regexp.MustCompile(`^\d+$`)

// CardFee is the 2.7% processing fee the customer pays to the gateway.
func CardFee(amount int64) int64 { return percentOf(amount, Codes.CardFeeBasisPoints) }

func cardBrand(number string) string {
	for _, c := range Codes.TestCards {
		if c.Number == number {
			return c.Brand
		}
	}
	return ""
}

func validateCard(c *CardInput, systemDate string, f fieldErrors) string {
	number := strings.NewReplacer(" ", "", "-", "").Replace(c.Number)
	brand := cardBrand(number)
	switch {
	case number == "":
		f.add("card.number", "This field is required.")
	case brand == "":
		f.add("card.number", "Simulated gateway: only the published Global Payments test card numbers are accepted. No real card is charged.")
	}
	if strings.TrimSpace(c.Name) == "" {
		f.add("card.name", "This field is required.")
	}
	m, err := strconv.Atoi(c.Month)
	if err != nil || m < 1 || m > 12 {
		f.add("card.month", "This field is required.")
	}
	y, err := strconv.Atoi(c.Year)
	if err != nil || y < 2000 || y > 2100 {
		f.add("card.year", "This field is required.")
	} else if m >= 1 && fmt.Sprintf("%04d-%02d", y, m) < systemDate[:7] {
		f.add("card.month", "The card has expired.")
	}
	want := 3
	if brand == "American Express" {
		want = 4
	}
	if !digits.MatchString(c.CVC) || len(c.CVC) != want {
		f.add("card.cvc", fmt.Sprintf("Enter the %d-digit security code.", want))
	}
	return brand
}

func (t *tx) validatePayment(in *PaymentInput, acct string, f fieldErrors) {
	if !oneOf(Codes.PaymentMethods, in.Method) {
		f.add("method", "Select a payment method.")
	}
	if in.Amount <= 0 {
		f.add("amount", "Enter a payment amount greater than $0.00.")
	}
	if in.EntryDate == "" {
		in.EntryDate = t.SystemDate
	}
	if !validDate(in.EntryDate) {
		f.add("entry_date", "Enter a valid date.")
	} else if in.EntryDate > t.SystemDate {
		f.add("entry_date", "The entry date cannot be after the system date.")
	}
	if in.PostmarkDate != "" && (!validDate(in.PostmarkDate) || in.PostmarkDate > t.SystemDate) {
		f.add("postmark_date", "Enter a postmark date on or before the system date.")
	}
	if in.Method != "Check" && (in.CheckNumber != "" || in.NameOnCheck != "") {
		f.add("check_number", "Check details apply to check payments only.")
	}
	if len(in.CheckNumber) > 30 || len(in.NameOnCheck) > 100 {
		f.add("check_number", "Too long.")
	}
	switch in.Allocation {
	case "account", "":
		in.Allocation = "account"
		if len(in.Allocations) > 0 {
			f.add("allocations", "Allocations apply to Pay Specific Policies / Arrangements only.")
		}
	case "policies":
		var total int64
		seen := map[string]bool{}
		for i, a := range in.Allocations {
			key := fmt.Sprintf("allocations.%d", i)
			if a.Amount < 0 {
				f.add(key, "Enter an amount of $0.00 or more.")
				continue
			}
			if seen[a.TermID] {
				f.add(key, "Each policy can be allocated once.")
			}
			seen[a.TermID] = true
			total += a.Amount
			if a.TermID == "" {
				continue
			}
			tm := t.Terms[a.TermID]
			if tm == nil || tm.AccountNumber != acct {
				f.add(key, "This policy is not on the account.")
				continue
			}
			if out := t.termLedger(tm).Outstanding; a.Amount > out {
				f.add(key, "Allocate no more than the outstanding balance of "+USD(out)+".")
			}
		}
		if total > in.Amount {
			f.add("allocations", "Total allocated "+USD(total)+" is more than the payment amount "+USD(in.Amount)+".")
		}
	default:
		f.add("allocation", "Choose Pay To Account or Pay Specific Policies / Arrangements.")
	}
}

func (t *tx) newPayment(in *PaymentInput) *Payment {
	return &Payment{ID: t.next("PMT"), Version: 1, Amount: in.Amount, Currency: "USD", Method: in.Method, EntryDate: in.EntryDate, PostmarkDate: in.PostmarkDate, CheckNumber: strings.TrimSpace(in.CheckNumber), NameOnCheck: strings.TrimSpace(in.NameOnCheck), Status: "Active", CreatedBy: t.user(), CreatedAt: t.stamp(), Allocations: []*Allocation{}}
}

// receivedText is the Payment Received activity wording.
func receivedText(p *Payment) string {
	if p.Method == "Check" {
		check := "Check Number: "
		if p.CheckNumber != "" {
			check += p.CheckNumber + " "
		}
		return fmt.Sprintf("Check payment for %s received. %sReceipt date: %s", USD(p.Amount), check, US2(p.EntryDate))
	}
	return fmt.Sprintf("%s payment for %s received. Receipt date: %s", p.Method, USD(p.Amount), US2(p.EntryDate))
}

// post allocates a payment to an account and writes the activity.
func (t *tx) post(p *Payment, acct *Account, in *PaymentInput) {
	p.AccountNumber = acct.Number
	terms := t.termsOf(acct.Number)
	rest := p.Amount
	if in.Allocation == "policies" {
		for _, a := range in.Allocations {
			if a.TermID == "" || a.Amount == 0 {
				continue
			}
			rest -= a.Amount - t.applyCash(p, []*Term{t.Terms[a.TermID]}, a.Amount)
		}
	} else {
		rest = t.applyCash(p, terms, rest)
	}
	if rest > 0 {
		t.toSuspense(p, acct.Number, rest)
	}
	t.Payments[p.ID] = p
	a := t.activity(acct.Number, "Payment Received", p.CreatedBy, cents(p.Amount), receivedText(p))
	a.PaymentID = p.ID
}

// MakePayment records a check, cash or simulated card payment on an account.
func (s *Store) MakePayment(number string, in PaymentInput) (*PaymentResult, error) {
	var out *PaymentResult
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		t.validatePayment(&in, acct.Number, f)
		brand := ""
		if in.Method == "Credit/Debit Card" {
			if in.Card == nil {
				f.add("card.number", "Card details are required.")
			} else {
				brand = validateCard(in.Card, t.SystemDate, f)
			}
		} else if in.Card != nil {
			f.add("card.number", "Card details apply to card payments only.")
		}
		if err := f.err(); err != nil {
			return err
		}
		if in.Card != nil && !in.Card.AcceptedTerms {
			return rule("Accept the payment terms to submit the card payment.")
		}
		p := t.newPayment(&in)
		var receipt *Receipt
		if in.Card != nil {
			number := strings.NewReplacer(" ", "", "-", "").Replace(in.Card.Number)
			p.CardBrand, p.CardLast4, p.NameOnCard = brand, number[len(number)-4:], strings.TrimSpace(in.Card.Name)
			p.Fee = CardFee(p.Amount)
			t.Seq["TXN"]++
			p.TransactionNumber = strconv.Itoa(10542529 + t.Seq["TXN"])
			receipt = &Receipt{TransactionNumber: p.TransactionNumber, Base: p.Amount, Fee: p.Fee, Total: p.Amount + p.Fee, Email: strings.TrimSpace(in.Card.Email)}
		}
		t.post(p, acct, &in)
		out = &PaymentResult{Payment: p, Receipt: receipt, Message: fmt.Sprintf("Payment for %s entered for Account %s.", USD(p.Amount), acct.Number)}
		return nil
	})
	return out, err
}

// ReversePayment sets a payment to Reversed and re-opens what it paid. Funds
// it left in suspense must still be there.
func (s *Store) ReversePayment(id string, version int, reason string) (*Payment, error) {
	var out *Payment
	err := s.update(func(t *tx) error {
		p := t.Payments[id]
		if p == nil {
			return ErrNotFound
		}
		if p.Version != version {
			return ErrConflict
		}
		if !oneOf(Codes.ReverseReasons, reason) {
			return &ValidationError{map[string]string{"reason": "Select a reverse reason."}}
		}
		if p.Status != "Active" {
			return rule("This payment is already %s.", strings.ToLower(p.Status))
		}
		for _, a := range p.Allocations {
			if a.Type == "Suspense" && a.Amount > 0 {
				if s := t.Suspense[a.SuspenseID]; s == nil || s.Amount < a.Amount {
					return rule("Part of this payment was already disbursed or allocated from suspense. Reverse that first.")
				}
			}
		}
		for _, a := range p.Allocations {
			switch a.Type {
			case "Scheduled Item":
				if it := t.findItem(a.TermID, a.ItemID); it != nil {
					it.Cash -= a.Amount
				}
			case "Suspense":
				if s := t.Suspense[a.SuspenseID]; s != nil {
					s.Amount -= a.Amount
					if s.Amount == 0 {
						delete(t.Suspense, s.ID)
					}
				}
			}
			a.Status = "Reversed"
		}
		p.Status, p.ReverseReason, p.ReversedOn = "Reversed", reason, t.SystemDate
		p.Version++
		if p.AccountNumber != "" && !p.Unidentified {
			if acct := t.Accounts[p.AccountNumber]; acct != nil {
				acct.Version++
			}
			a := t.activity(p.AccountNumber, "Reverse Payment", CurrentUser, cents(p.Amount), USD(p.Amount)+" Payment was reversed. Reason: "+strings.TrimPrefix(reason, "RP-"))
			a.PaymentID = p.ID
		}
		out = p
		return nil
	})
	return out, err
}

// ---- unidentified payments ----

type UnidentifiedInput struct {
	Identity
	Method      string `json:"method"`
	Amount      int64  `json:"amount"`
	EntryDate   string `json:"entry_date"`
	CheckNumber string `json:"check_number"`
	NameOnCheck string `json:"name_on_check"`
}

func checkIdentity(id *Identity, f fieldErrors) {
	for _, p := range []*string{&id.Last, &id.First, &id.Middle, &id.AccountRef, &id.PolicyRef, &id.Other} {
		*p = strings.TrimSpace(*p)
		if len(*p) > 100 {
			f.add("identity", "Identifiers are limited to 100 characters.")
		}
	}
	if id.Last == "" && id.AccountRef == "" && id.PolicyRef == "" && id.Other == "" {
		const msg = "Enter at least one of the following pieces of information."
		for _, k := range []string{"last", "account_ref", "policy_ref", "other"} {
			f.add(k, msg)
		}
	}
}

// IdentifierOf is the Identifier Type and Identifier the list shows.
func IdentifierOf(id *Identity) (string, string) {
	switch {
	case id == nil:
		return "", ""
	case id.AccountRef != "":
		return "Account", id.AccountRef
	case id.PolicyRef != "":
		return "Policy", id.PolicyRef
	case id.Last != "":
		return "Last", id.Last
	default:
		return "Other", id.Other
	}
}

func (s *Store) EnterUnidentified(in UnidentifiedInput) (*Payment, error) {
	var out *Payment
	err := s.update(func(t *tx) error {
		f := fieldErrors{}
		checkIdentity(&in.Identity, f)
		pin := PaymentInput{Method: in.Method, Amount: in.Amount, EntryDate: in.EntryDate, CheckNumber: in.CheckNumber, NameOnCheck: in.NameOnCheck}
		t.validatePayment(&pin, "", f)
		if in.Method == "Credit/Debit Card" {
			f.add("method", "Card payments are taken on an account through the payment portal.")
		}
		if err := f.err(); err != nil {
			return err
		}
		p := t.newPayment(&pin)
		p.Unidentified, p.Identity = true, &in.Identity
		t.Payments[p.ID] = p
		out = p
		return nil
	})
	return out, err
}

func (s *Store) UpdateReference(id string, version int, ident Identity) (*Payment, error) {
	var out *Payment
	err := s.update(func(t *tx) error {
		p, err := t.unidentified(id, version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		checkIdentity(&ident, f)
		if err := f.err(); err != nil {
			return err
		}
		p.Identity = &ident
		out = p
		return nil
	})
	return out, err
}

func (t *tx) unidentified(id string, version int) (*Payment, error) {
	p := t.Payments[id]
	if p == nil || !p.Unidentified {
		return nil, ErrNotFound
	}
	if p.Version != version {
		return nil, ErrConflict
	}
	if p.Status != "Active" {
		return nil, rule("This payment is %s.", strings.ToLower(p.Status))
	}
	p.Version++
	return p, nil
}

// ApplyUnidentified posts an unidentified payment to an account.
func (s *Store) ApplyUnidentified(id string, version int, account string) (*PaymentResult, error) {
	var out *PaymentResult
	err := s.update(func(t *tx) error {
		p, err := t.unidentified(id, version)
		if err != nil {
			return err
		}
		acct := t.Accounts[strings.TrimSpace(account)]
		if acct == nil {
			return &ValidationError{map[string]string{"account": "No billing account matches that number."}}
		}
		acct.Version++
		p.Unidentified, p.AppliedOn = false, t.SystemDate
		t.post(p, acct, &PaymentInput{Allocation: "account"})
		out = &PaymentResult{Payment: p, Message: fmt.Sprintf("Payment for %s applied to Account %s.", USD(p.Amount), acct.Number)}
		return nil
	})
	return out, err
}

// ---- search ----

type PaymentQuery struct {
	Account, Policy, DateFrom, DateTo, Method, Currency, Other string
	AmountFrom, AmountTo                                       int64 // -1: not set
}

func methodMatches(filter, method string) bool {
	switch filter {
	case "":
		return true
	case "Credit Card", "Debit Card", "Credit/Debit Card":
		return method == "Credit/Debit Card"
	default:
		return method == filter
	}
}

// SearchPayments is Search For A Payment.
func (s *Store) SearchPayments(q PaymentQuery) ([]*Payment, error) {
	if q.Account == "" && q.Policy == "" && q.DateFrom == "" && q.DateTo == "" && q.Method == "" && q.Currency == "" && q.Other == "" && q.AmountFrom < 0 && q.AmountTo < 0 {
		return nil, &ValidationError{map[string]string{"query": "Enter at least one piece of information."}}
	}
	out := []*Payment{}
	s.read(func(st *State) {
		other := strings.ToLower(strings.TrimSpace(q.Other))
		for _, p := range st.Payments {
			if q.Account != "" && !strings.EqualFold(p.AccountNumber, q.Account) {
				continue
			}
			if q.Policy != "" {
				hit := false
				for _, a := range p.Allocations {
					if tm := st.Terms[a.TermID]; tm != nil && strings.EqualFold(tm.PolicyNumber, q.Policy) {
						hit = true
					}
				}
				if !hit {
					continue
				}
			}
			if (q.DateFrom != "" && p.EntryDate < q.DateFrom) || (q.DateTo != "" && p.EntryDate > q.DateTo) {
				continue
			}
			if (q.AmountFrom >= 0 && p.Amount < q.AmountFrom) || (q.AmountTo >= 0 && p.Amount > q.AmountTo) {
				continue
			}
			if !methodMatches(q.Method, p.Method) || (q.Currency != "" && q.Currency != p.Currency) {
				continue
			}
			if other != "" {
				hay := []string{p.CheckNumber, p.NameOnCheck, p.NameOnCard, p.CardLast4}
				if p.Identity != nil {
					hay = append(hay, p.Identity.First, p.Identity.Last)
				}
				hit := false
				for _, h := range hay {
					if h != "" && strings.Contains(strings.ToLower(h), other) {
						hit = true
					}
				}
				if !hit {
					continue
				}
			}
			out = append(out, p)
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}
