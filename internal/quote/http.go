package quote

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, err error) {
	var validation *ValidationError
	switch {
	case errors.As(err, &validation):
		reply(w, 422, map[string]any{"error": err.Error(), "fields": validation.Fields})
	case errors.Is(err, ErrNotFound):
		reply(w, 404, map[string]any{"error": err.Error()})
	case errors.Is(err, ErrConflict), errors.Is(err, ErrTransition):
		reply(w, 409, map[string]any{"error": err.Error()})
	default:
		reply(w, 500, map[string]any{"error": "Could not access quote storage. Your changes were not saved."})
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		reply(w, 400, map[string]any{"error": "Invalid quote request."})
		return false
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		reply(w, 400, map[string]any{"error": "Expected one JSON object."})
		return false
	}
	return true
}
func Register(mux *http.ServeMux, s *Store) {
	mux.HandleFunc("GET /api/quotes", func(w http.ResponseWriter, r *http.Request) {
		q, err := s.List()
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, q)
	})
	mux.HandleFunc("POST /api/quotes", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Values map[string]any `json:"values"`
		}
		if !decode(w, r, &in) {
			return
		}
		q, err := s.Create(in.Values)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 201, q)
	})
	mux.HandleFunc("GET /api/quotes/{id}", func(w http.ResponseWriter, r *http.Request) {
		q, err := s.Get(r.PathValue("id"))
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, q)
	})
	mux.HandleFunc("PUT /api/quotes/{id}", func(w http.ResponseWriter, r *http.Request) {
		var in Input
		if !decode(w, r, &in) {
			return
		}
		q, err := s.Save(r.PathValue("id"), in)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, q)
	})
	mux.HandleFunc("POST /api/quotes/{id}/submit", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Version int `json:"version"`
		}
		if !decode(w, r, &in) {
			return
		}
		q, err := s.Submit(r.PathValue("id"), in.Version)
		if err != nil {
			failure(w, err)
			return
		}
		reply(w, 200, q)
	})
}
