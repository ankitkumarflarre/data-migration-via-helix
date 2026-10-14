package billing

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func failure(w http.ResponseWriter, err error) {
	var validation *ValidationError
	var blocked *RuleError
	switch {
	case errors.As(err, &validation):
		reply(w, 422, map[string]any{"error": err.Error(), "fields": validation.Fields})
	case errors.As(err, &blocked):
		reply(w, 409, map[string]any{"error": blocked.Message})
	case errors.Is(err, ErrNotFound):
		reply(w, 404, map[string]any{"error": err.Error()})
	case errors.Is(err, ErrConflict):
		reply(w, 409, map[string]any{"error": err.Error(), "stale": true})
	default:
		reply(w, 500, map[string]any{"error": "Could not access billing storage. Your changes were not saved."})
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		reply(w, 400, map[string]any{"error": "Invalid billing request."})
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		reply(w, 400, map[string]any{"error": "Expected one JSON object."})
		return false
	}
	return true
}

// handle wires a JSON endpoint: decode the body into In, run fn, reply.
func handle[In any](mux *http.ServeMux, pattern string, fn func(r *http.Request, in In) (any, error)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var in In
		if r.Method != http.MethodGet && !decode(w, r, &in) {
			return
		}
		out, err := fn(r, in)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, out)
	})
}

type none struct{}

type versioned struct {
	Version int `json:"version"`
}

type message struct {
	Message string `json:"message"`
}

func amountParam(r *http.Request, key string) int64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return -1
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return -1
	}
	return n
}

// Register adds the billing routes under /api/billing.
func Register(mux *http.ServeMux, s *Store) {
	q := func(r *http.Request, k string) string { return strings.TrimSpace(r.URL.Query().Get(k)) }
	id := func(r *http.Request, k string) string { return r.PathValue(k) }

	// System date, settings, seed.
	handle(mux, "GET /api/billing/system", func(r *http.Request, _ none) (any, error) { return s.System(), nil })
	handle(mux, "GET /api/billing/codes", func(r *http.Request, _ none) (any, error) { return json.RawMessage(codesJSON), nil })
	handle(mux, "PUT /api/billing/system/date", func(r *http.Request, in struct {
		Version int    `json:"version"`
		Date    string `json:"date"`
	}) (any, error) {
		if err := s.SetSystemDate(in.Version, in.Date); err != nil {
			return nil, err
		}
		return s.System(), nil
	})
	handle(mux, "POST /api/billing/system/run", func(r *http.Request, in versioned) (any, error) { return s.RunDueProcesses(in.Version) })
	handle(mux, "PUT /api/billing/settings", func(r *http.Request, in SettingsUpdate) (any, error) {
		if err := s.SaveSettings(in); err != nil {
			return nil, err
		}
		return s.System(), nil
	})
	handle(mux, "POST /api/billing/seed", func(r *http.Request, _ struct{}) (any, error) {
		if err := s.Seed(false); err != nil {
			return nil, err
		}
		return s.System(), nil
	})
	handle(mux, "POST /api/billing/disbursement-service", func(r *http.Request, in struct {
		Version int    `json:"version"`
		Action  string `json:"action"`
	}) (any, error) {
		if err := s.DisbursementService(in.Version, in.Action); err != nil {
			return nil, err
		}
		return s.System(), nil
	})

	// Home and account search.
	handle(mux, "GET /api/billing/home", func(r *http.Request, _ none) (any, error) { return s.Home(), nil })
	handle(mux, "GET /api/billing/accounts", func(r *http.Request, _ none) (any, error) { return s.SearchAccounts(q(r, "by"), q(r, "q")) })
	handle(mux, "POST /api/billing/accounts", func(r *http.Request, in NewAccountInput) (any, error) { return s.CreateAccount(in) })
	handle(mux, "GET /api/billing/accounts/{n}", func(r *http.Request, _ none) (any, error) { return s.Account(id(r, "n")) })
	handle(mux, "POST /api/billing/accounts/{n}/recent", func(r *http.Request, _ struct{}) (any, error) { return none{}, s.Touch(id(r, "n")) })
	handle(mux, "PUT /api/billing/accounts/{n}/settings", func(r *http.Request, in SettingsInput) (any, error) {
		return message{"Account settings saved."}, s.SaveAccountSettings(id(r, "n"), in)
	})
	handle(mux, "PUT /api/billing/accounts/{n}/party", func(r *http.Request, in PartyRoleInput) (any, error) {
		return message{"The account " + in.Role + " was changed."}, s.ChangeParty(id(r, "n"), in)
	})

	// Payments.
	handle(mux, "POST /api/billing/accounts/{n}/payments", func(r *http.Request, in PaymentInput) (any, error) { return s.MakePayment(id(r, "n"), in) })
	handle(mux, "GET /api/billing/payments", func(r *http.Request, _ none) (any, error) {
		ps, err := s.SearchPayments(PaymentQuery{Account: q(r, "account"), Policy: q(r, "policy"), DateFrom: q(r, "from"), DateTo: q(r, "to"), Method: q(r, "method"), Currency: q(r, "currency"), Other: q(r, "other"), AmountFrom: amountParam(r, "min"), AmountTo: amountParam(r, "max")})
		if err != nil {
			return nil, err
		}
		return s.PaymentViews(ps), nil
	})
	handle(mux, "GET /api/billing/payments/{id}", func(r *http.Request, _ none) (any, error) { return s.Payment(id(r, "id")) })
	handle(mux, "POST /api/billing/payments/{id}/reverse", func(r *http.Request, in struct {
		Version int    `json:"version"`
		Reason  string `json:"reason"`
	}) (any, error) {
		return s.ReversePayment(id(r, "id"), in.Version, in.Reason)
	})
	handle(mux, "GET /api/billing/unidentified", func(r *http.Request, _ none) (any, error) { return s.Unidentified(), nil })
	handle(mux, "POST /api/billing/unidentified", func(r *http.Request, in UnidentifiedInput) (any, error) { return s.EnterUnidentified(in) })
	handle(mux, "PUT /api/billing/unidentified/{id}/reference", func(r *http.Request, in struct {
		Version int `json:"version"`
		Identity
	}) (any, error) {
		return s.UpdateReference(id(r, "id"), in.Version, in.Identity)
	})
	handle(mux, "POST /api/billing/unidentified/{id}/apply", func(r *http.Request, in struct {
		Version int    `json:"version"`
		Account string `json:"account"`
	}) (any, error) {
		return s.ApplyUnidentified(id(r, "id"), in.Version, in.Account)
	})

	// Suspense and disbursements.
	handle(mux, "POST /api/billing/accounts/{n}/suspense/allocate", func(r *http.Request, in SuspenseAllocationInput) (any, error) {
		m, err := s.AllocateSuspense(id(r, "n"), in)
		return message{m}, err
	})
	handle(mux, "POST /api/billing/accounts/{n}/suspense/action", func(r *http.Request, in SuspenseActionInput) (any, error) {
		return message{in.Action + " applied to the selected items."}, s.SuspenseAction(id(r, "n"), in)
	})
	handle(mux, "POST /api/billing/accounts/{n}/disbursements", func(r *http.Request, in DisburseInput) (any, error) {
		d, m, err := s.Disburse(id(r, "n"), in)
		return map[string]any{"disbursement": d, "message": m}, err
	})
	handle(mux, "POST /api/billing/accounts/{n}/disbursements/{id}/decision", func(r *http.Request, in struct {
		Version int  `json:"version"`
		Approve bool `json:"approve"`
	}) (any, error) {
		return s.DecideDisbursement(id(r, "n"), id(r, "id"), in.Version, in.Approve)
	})

	// Holds and tasks.
	handle(mux, "POST /api/billing/accounts/{n}/holds", func(r *http.Request, in HoldInput) (any, error) {
		h, m, err := s.PlaceHold(id(r, "n"), in)
		return map[string]any{"hold": h, "message": m}, err
	})
	handle(mux, "PUT /api/billing/accounts/{n}/holds/{id}", func(r *http.Request, in HoldUpdateInput) (any, error) {
		m, err := s.UpdateHold(id(r, "n"), id(r, "id"), in)
		return message{m}, err
	})
	handle(mux, "GET /api/billing/hold-events", func(r *http.Request, _ none) (any, error) { return s.HoldEvents(), nil })
	handle(mux, "POST /api/billing/hold-events/{id}/end", func(r *http.Request, in versioned) (any, error) {
		return message{"Hold event ended."}, s.EndHoldEvent(id(r, "id"), in.Version)
	})
	handle(mux, "POST /api/billing/accounts/{n}/tasks", func(r *http.Request, in TaskInput) (any, error) { return s.AddTask(id(r, "n"), in) })
	handle(mux, "POST /api/billing/accounts/{n}/tasks/{id}/pickup", func(r *http.Request, in versioned) (any, error) {
		return message{"Task picked up."}, s.PickupTask(id(r, "n"), id(r, "id"), in.Version)
	})

	// Policy terms.
	handle(mux, "GET /api/billing/terms/{id}", func(r *http.Request, _ none) (any, error) { return s.Term(id(r, "id")) })
	handle(mux, "PUT /api/billing/accounts/{n}/terms/{id}/plan", func(r *http.Request, in PlanInput) (any, error) {
		return message{"Payment plan saved."}, s.SetTermPlan(id(r, "n"), id(r, "id"), in)
	})
	handle(mux, "PUT /api/billing/accounts/{n}/terms/{id}/renewal-plan", func(r *http.Request, in PlanInput) (any, error) {
		return message{"Renewal payment plan saved."}, s.SetRenewalPlan(id(r, "n"), id(r, "id"), in)
	})
	handle(mux, "POST /api/billing/accounts/{n}/terms/{id}/transfer", func(r *http.Request, in TransferInput) (any, error) {
		m, err := s.TransferTerm(id(r, "n"), id(r, "id"), in)
		return message{m}, err
	})
	handle(mux, "POST /api/billing/accounts/{n}/terms/{id}/collections", func(r *http.Request, in versioned) (any, error) {
		m, err := s.TransferToCollections(id(r, "n"), id(r, "id"), in.Version)
		return message{m}, err
	})
	handle(mux, "POST /api/billing/accounts/{n}/terms/{id}/pif", func(r *http.Request, in PIFInput) (any, error) {
		m, err := s.ApplyPIF(id(r, "n"), id(r, "id"), in)
		return message{m}, err
	})

	// Parties.
	handle(mux, "POST /api/billing/parties/search", func(r *http.Request, in PartyQuery) (any, error) { return s.SearchParties(in) })
	handle(mux, "POST /api/billing/parties", func(r *http.Request, in NewPartyInput) (any, error) {
		p, err := s.CreateParty(in)
		if err != nil {
			return nil, err
		}
		return s.Party(p.ID)
	})
	handle(mux, "GET /api/billing/parties/{id}", func(r *http.Request, _ none) (any, error) { return s.Party(id(r, "id")) })
	handle(mux, "POST /api/billing/parties/{id}/rows/{kind}", func(r *http.Request, in RowInput) (any, error) {
		if _, err := s.SaveRow(id(r, "id"), id(r, "kind"), in); err != nil {
			return nil, err
		}
		return s.Party(id(r, "id"))
	})
	handle(mux, "PUT /api/billing/parties/{id}/demographics", func(r *http.Request, in struct {
		Version int               `json:"version"`
		Fields  map[string]string `json:"fields"`
	}) (any, error) {
		if _, err := s.SaveDemographics(id(r, "id"), in.Version, in.Fields); err != nil {
			return nil, err
		}
		return s.Party(id(r, "id"))
	})

	// Back office.
	handle(mux, "GET /api/billing/batches", func(r *http.Request, _ none) (any, error) { return s.Batches(), nil })
	handle(mux, "POST /api/billing/batches", func(r *http.Request, in BatchInput) (any, error) { return s.CreateBatch(in) })
	handle(mux, "POST /api/billing/batches/{n}/reverse", func(r *http.Request, in versioned) (any, error) {
		return message{"Batch " + id(r, "n") + " was reversed."}, s.ReverseBatch(id(r, "n"), in.Version)
	})
	handle(mux, "GET /api/billing/errors", func(r *http.Request, _ none) (any, error) { return s.Errors(), nil })
	handle(mux, "POST /api/billing/errors/{id}/resubmit", func(r *http.Request, in ResubmitInput) (any, error) { return s.Resubmit(id(r, "id"), in) })
	handle(mux, "GET /api/billing/statements", func(r *http.Request, _ none) (any, error) { return s.Statements(), nil })
	handle(mux, "POST /api/billing/statements/import", func(r *http.Request, in struct {
		IDs []string `json:"ids"`
	}) (any, error) {
		n, err := s.ImportStatements(in.IDs)
		return message{strconv.Itoa(n) + " statement(s) imported."}, err
	})
	handle(mux, "GET /api/billing/reports/{id}", func(r *http.Request, _ none) (any, error) { return s.Report(id(r, "id")) })
}
