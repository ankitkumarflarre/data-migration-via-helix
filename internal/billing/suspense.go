package billing

import (
	"fmt"
	"sort"
)

func (st *State) suspenseOf(acct string) []*SuspenseItem {
	out := []*SuspenseItem{}
	for _, s := range st.Suspense {
		if s.AccountNumber == acct && s.Amount > 0 {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// selectSuspense loads the chosen items of an account.
func (t *tx) selectSuspense(acct string, ids []string, f fieldErrors) []*SuspenseItem {
	out := []*SuspenseItem{}
	seen := map[string]bool{}
	for _, id := range ids {
		s := t.Suspense[id]
		if s == nil || s.AccountNumber != acct || s.Amount == 0 || seen[id] {
			f.add("suspense_ids", "Select items in suspense on this account.")
			continue
		}
		seen[id] = true
		out = append(out, s)
	}
	if len(ids) == 0 {
		f.add("suspense_ids", "Select at least one item in suspense.")
	}
	return out
}

type SuspenseAllocationInput struct {
	Version     int              `json:"version"`
	SuspenseIDs []string         `json:"suspense_ids"`
	Allocation  string           `json:"allocation"` // account | policies
	Allocations []TermAllocation `json:"allocations"`
}

// AllocateSuspense pays open items from money held in suspense. The paid items
// are recorded against the payment the money came from, so reversing that
// payment re-opens them.
func (s *Store) AllocateSuspense(number string, in SuspenseAllocationInput) (string, error) {
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		items := t.selectSuspense(acct.Number, in.SuspenseIDs, f)
		var total int64
		for _, it := range items {
			if it.Status != "No Hold" {
				f.add("suspense_ids", "Release the hold on the selected items first.")
			}
			total += it.Amount
		}
		pin := PaymentInput{Method: "Check", Amount: total, Allocation: in.Allocation, Allocations: in.Allocations}
		if total > 0 {
			t.validatePayment(&pin, acct.Number, f)
		}
		if err := f.err(); err != nil {
			return err
		}
		targets := [][2]any{}
		if pin.Allocation == "policies" {
			for _, a := range pin.Allocations {
				if a.TermID != "" && a.Amount > 0 {
					targets = append(targets, [2]any{[]*Term{t.Terms[a.TermID]}, a.Amount})
				}
			}
		} else {
			targets = append(targets, [2]any{t.termsOf(acct.Number), total})
		}
		var applied int64
		for _, tg := range targets {
			want := tg[1].(int64)
			for _, it := range items {
				if want == 0 {
					break
				}
				take := min(want, it.Amount)
				p := t.Payments[it.PaymentID]
				if p == nil {
					p = &Payment{ID: it.PaymentID}
				}
				left := t.applyCash(p, tg[0].([]*Term), take)
				used := take - left
				it.Amount -= used
				want -= used
				applied += used
				for _, a := range p.Allocations {
					if a.Type == "Suspense" && a.SuspenseID == it.ID {
						a.Amount -= used
					}
				}
				if left > 0 {
					break
				}
			}
		}
		if applied == 0 {
			return rule("Nothing is open to allocate the suspense amount to.")
		}
		t.activity(acct.Number, "Suspense Allocation", CurrentUser, cents(applied), USD(applied)+" was allocated from suspense.")
		msg = fmt.Sprintf("%s was allocated from suspense for Account %s.", USD(applied), acct.Number)
		return nil
	})
	return msg, err
}

type SuspenseActionInput struct {
	Version     int      `json:"version"`
	SuspenseIDs []string `json:"suspense_ids"`
	Action      string   `json:"action"` // Place Hold | Release Hold
}

// SuspenseAction places or releases a hold on items in suspense. Disburse has
// its own screen; Hold For Premium, Transfer and Write Off are not specified.
func (s *Store) SuspenseAction(number string, in SuspenseActionInput) error {
	return s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		items := t.selectSuspense(acct.Number, in.SuspenseIDs, f)
		status := map[string]string{"Place Hold": "Hold", "Release Hold": "No Hold"}[in.Action]
		if status == "" {
			f.add("action", "This action is not available yet.")
		}
		if err := f.err(); err != nil {
			return err
		}
		for _, it := range items {
			it.Status = status
		}
		return nil
	})
}

type DisburseInput struct {
	Version     int      `json:"version"`
	SuspenseIDs []string `json:"suspense_ids"`
	Amount      int64    `json:"amount"`
	Type        string   `json:"type"`
	Reason      string   `json:"reason"`
	PayeeID     string   `json:"payee_id"` // blank: the account payee
}

// Disburse refunds money in suspense. The refund is always a check; it is
// created Pending Authorization and authorized by System at or under the
// workspace limit.
func (s *Store) Disburse(number string, in DisburseInput) (*Disbursement, string, error) {
	var out *Disbursement
	var msg string
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, in.Version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		items := t.selectSuspense(acct.Number, in.SuspenseIDs, f)
		var available int64
		for _, it := range items {
			if it.Status != "No Hold" {
				f.add("suspense_ids", "Items on hold cannot be disbursed.")
			}
			available += it.Amount
		}
		if in.Amount <= 0 || in.Amount > available {
			f.add("amount", "Enter an amount between $0.01 and "+USD(available)+".")
		}
		if in.Type == "" {
			in.Type = "Regular"
		}
		if !oneOf(Codes.DisbursementTypes, in.Type) {
			f.add("type", "Select a disbursement type.")
		}
		if !oneOf(Codes.DisbursementReasons, in.Reason) {
			f.add("reason", "Select a disbursement reason.")
		}
		payee := acct.PayeeID
		if in.PayeeID != "" {
			payee = in.PayeeID
		}
		if t.Parties[payee] == nil {
			f.add("payee_id", "Select a payee.")
		}
		if err := f.err(); err != nil {
			return err
		}
		d := &Disbursement{ID: t.next("DSB"), AccountNumber: acct.Number, PayeePartyID: payee, PayeeName: t.partyName(payee), Amount: in.Amount, Type: in.Type, Reason: in.Reason, Group: Codes.DisbursementGroups[0], Method: "Check", Description: in.Reason, Status: "Pending Authorization", RequestDate: t.SystemDate, User: CurrentUser, PayeeChanged: payee != acct.PayeeID}
		left := in.Amount
		for _, it := range items {
			if left == 0 {
				break
			}
			take := min(left, it.Amount)
			it.Amount -= take
			left -= take
			d.Sources = append(d.Sources, DisbursementSource{SuspenseID: it.ID, Amount: take})
		}
		t.Disbursements[d.ID] = d
		t.activity(acct.Number, "Disbursement Pending Authorization", CurrentUser, cents(d.Amount), fmt.Sprintf("Disbursement for %s was submitted for authorization. Payee: %s", USD(d.Amount), d.PayeeName))
		if d.Amount <= t.Settings.AutoAuthorizeLimit {
			t.authorize(d, SystemUser)
		}
		out = d
		msg = fmt.Sprintf("Disbursement for %s was submitted for authorization. Payee: %s", USD(d.Amount), d.PayeeName)
		return nil
	})
	return out, msg, err
}

func (t *tx) authorize(d *Disbursement, by string) {
	d.Status, d.AuthorizedBy, d.AuthorizedOn = "Authorized", by, t.SystemDate
	t.activity(d.AccountNumber, "Disbursement Authorized", by, cents(d.Amount), "Disbursement for "+USD(d.Amount)+" was authorized.")
}

// DecideDisbursement is the approver's decision on a disbursement above the
// auto-authorization limit. A rejection returns the money to suspense.
func (s *Store) DecideDisbursement(number, id string, version int, approve bool) (*Disbursement, error) {
	var out *Disbursement
	err := s.update(func(t *tx) error {
		acct, err := t.account(number, version)
		if err != nil {
			return err
		}
		d := t.Disbursements[id]
		if d == nil || d.AccountNumber != acct.Number {
			return ErrNotFound
		}
		if d.Status != "Pending Authorization" {
			return rule("This disbursement is already %s.", d.Status)
		}
		if approve {
			t.authorize(d, CurrentUser)
		} else {
			d.Status = "Rejected"
			for _, src := range d.Sources {
				if it := t.Suspense[src.SuspenseID]; it != nil {
					it.Amount += src.Amount
				}
			}
			t.activity(acct.Number, "Disbursement Rejected", CurrentUser, cents(d.Amount), "Disbursement for "+USD(d.Amount)+" was rejected. The funds were returned to suspense.")
		}
		out = d
		return nil
	})
	return out, err
}
