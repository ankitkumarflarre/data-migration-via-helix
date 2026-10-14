package quotesync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/quote"
)

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Register adds the Helix routes of the quote flow.
func (s *Syncer) Register(mux *http.ServeMux) {
	// Policies migrated from workbooks, to open in the quote flow.
	mux.HandleFunc("GET /api/quotes/imports", func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Policies()
		if err != nil {
			reply(w, 500, map[string]any{"error": err.Error()})
			return
		}
		reply(w, 200, out)
	})
	// Open a Helix policy as a quote (or reopen the quote it is already in).
	mux.HandleFunc("POST /api/quotes/import", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			PolicyNumber string `json:"policy_number"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&in); err != nil || in.PolicyNumber == "" {
			reply(w, 400, map[string]any{"error": "Give the policy number to open."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		q, created, err := s.Import(ctx, in.PolicyNumber)
		switch {
		case errors.Is(err, ErrNoPolicy):
			reply(w, 404, map[string]any{"error": err.Error()})
		case err != nil:
			reply(w, 502, map[string]any{"error": "Could not read the policy from Helix: " + err.Error()})
		case created:
			reply(w, 201, q)
		default:
			reply(w, 200, q)
		}
	})
	// Write the quote to Helix again now (e.g. after a failed sync).
	mux.HandleFunc("POST /api/quotes/{id}/helix", func(w http.ResponseWriter, r *http.Request) {
		q, err := s.Quotes.Get(r.PathValue("id"))
		if errors.Is(err, quote.ErrNotFound) {
			reply(w, 404, map[string]any{"error": err.Error()})
			return
		}
		if err != nil {
			reply(w, 500, map[string]any{"error": err.Error()})
			return
		}
		s.Enqueue(q)
		reply(w, 202, q)
	})
}
