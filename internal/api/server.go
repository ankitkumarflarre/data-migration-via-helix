// Package api is the JSON API behind the UI: upload → plan → overrides →
// approve → execute → results. Jobs live in memory (single user, D11).
package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/browse"
	"github.com/ankitkumarflarre/datamigration/internal/excel"
	"github.com/ankitkumarflarre/datamigration/internal/execute"
	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/plan"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
)

const maxUpload = 25 << 20

// Server holds the shared dependencies and the jobs.
type Server struct {
	Helix  *helix.Client
	API    execute.API
	Schema *schema.Schema
	Ledger *ledger.Ledger
	Web    fs.FS // built UI; nil serves a notice
	Browse *browse.Browser

	mu      sync.Mutex
	jobs    map[string]*job
	running string // id of the job currently writing (one at a time)
}

type approval struct {
	PlanHash     string    `json:"plan_hash"`
	Acknowledged []string  `json:"acknowledged"`
	RowLimit     int       `json:"row_limit"`
	DryRun       bool      `json:"dry_run"`
	At           time.Time `json:"approved_at"`
	RunID        string    `json:"run_id"`
}

type job struct {
	mu         sync.Mutex
	ID         string
	FileName   string
	FileSHA    string
	RuleSet    string
	UploadedAt time.Time
	sheet      *excel.Sheet
	bundle     *rules.Bundle
	Overrides  plan.Overrides
	plan       *plan.Plan
	approvals  []approval
	exec       *execute.Executor
}

// NewServer wires the dependencies.
func NewServer(h *helix.Client, api execute.API, s *schema.Schema, l *ledger.Ledger, web fs.FS) *Server {
	srv := &Server{Helix: h, API: api, Schema: s, Ledger: l, Web: web, jobs: map[string]*job{}}
	srv.Browse = &browse.Browser{Helix: h, Schema: s, Ledger: l}
	go srv.Browse.Warm(context.Background(), "policy", "party", "location", "dwelling_asset", "wind_mitigation_verification")
	return srv
}

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/rulesets", s.ruleSets)
	mux.HandleFunc("POST /api/jobs", s.upload)
	mux.HandleFunc("GET /api/jobs/{id}", s.getJob)
	mux.HandleFunc("GET /api/jobs/{id}/issues", s.issues)
	mux.HandleFunc("GET /api/jobs/{id}/rows", s.rows)
	mux.HandleFunc("PUT /api/jobs/{id}/overrides", s.overrides)
	mux.HandleFunc("POST /api/jobs/{id}/approve", s.approve)
	mux.HandleFunc("GET /api/jobs/{id}/results", s.results)
	mux.HandleFunc("GET /api/jobs/{id}/results.csv", s.resultsCSV)
	mux.HandleFunc("GET /api/schema/variants", s.variants)
	mux.HandleFunc("GET /api/schema/variants/{variant}", s.variant)
	mux.HandleFunc("GET /api/schema/entities", s.entities)
	mux.HandleFunc("GET /api/schema/entities/{entity}/fields", s.entityFields)
	mux.HandleFunc("GET /api/browse/policy", s.browsePolicy)
	mux.HandleFunc("GET /api/browse/table", s.browseTable)
	mux.HandleFunc("/", s.static)
	return logRequests(mux)
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") && !(r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/results")) {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, status int, format string, a ...any) {
	writeJSON(w, status, map[string]any{"error": fmt.Sprintf(format, a...)})
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) job(w http.ResponseWriter, r *http.Request) *job {
	s.mu.Lock()
	j := s.jobs[r.PathValue("id")]
	s.mu.Unlock()
	if j == nil {
		fail(w, http.StatusNotFound, "no job %s", r.PathValue("id"))
	}
	return j
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	out := map[string]any{"helix_url": s.Helix.Base(), "rule_sets": rules.Names(), "ledger_records": len(s.Ledger.All())}
	if err := s.Helix.Ping(ctx); err != nil {
		out["helix_reachable"], out["helix_error"] = false, err.Error()
	} else {
		out["helix_reachable"] = true
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) ruleSets(w http.ResponseWriter, _ *http.Request) {
	var out []map[string]any
	for _, n := range rules.Names() {
		b, err := rules.Load(n)
		if err != nil {
			continue
		}
		out = append(out, map[string]any{"name": n, "rules": len(b.RuleSet.Rules), "sheet": b.RuleSet.Sheet,
			"source_report": b.RuleSet.SourceReport.File, "sha256": b.SHA256})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		fail(w, http.StatusBadRequest, "upload must be a multipart form under 25 MB: %v", err)
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		fail(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()
	name := strings.ToLower(hdr.Filename)
	if !strings.HasSuffix(name, ".xlsx") && !strings.HasSuffix(name, ".xlsm") {
		fail(w, http.StatusBadRequest, "only .xlsx and .xlsm workbooks are accepted")
		return
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		fail(w, http.StatusBadRequest, "read upload: %v", err)
		return
	}
	ruleSet := r.FormValue("rule_set")
	if ruleSet == "" {
		ruleSet = rules.Names()[0]
	}
	b, err := rules.Load(ruleSet)
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	sheet, err := excel.Read(bytes.NewReader(raw), b.RuleSet.Sheet, b.Templates.RowKeyColumn)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, "%v", err)
		return
	}
	sum := sha256.Sum256(raw)
	j := &job{ID: newID(), FileName: hdr.Filename, FileSHA: hex.EncodeToString(sum[:]), RuleSet: ruleSet,
		UploadedAt: time.Now().UTC(), sheet: sheet, bundle: b}
	if err := s.replan(r.Context(), j, plan.Overrides{}); err != nil {
		fail(w, http.StatusBadGateway, "plan: %v", err)
		return
	}
	s.mu.Lock()
	s.jobs[j.ID] = j
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, s.view(j))
}

func (s *Server) replan(ctx context.Context, j *job, ov plan.Overrides) error {
	p, err := plan.Build(ctx, plan.Input{Bundle: j.bundle, Sheet: j.sheet, FileSHA: j.FileSHA, Overrides: ov, Schema: s.Schema})
	if err != nil {
		return err
	}
	j.mu.Lock()
	j.plan, j.Overrides = p, ov
	j.mu.Unlock()
	return nil
}

type jobView struct {
	ID         string            `json:"id"`
	FileName   string            `json:"file_name"`
	FileSHA    string            `json:"file_sha"`
	RuleSet    string            `json:"rule_set"`
	UploadedAt time.Time         `json:"uploaded_at"`
	SheetRows  int               `json:"sheet_rows"`
	Headers    map[string]string `json:"headers"`
	Overrides  plan.Overrides    `json:"overrides"`
	Plan       *plan.Plan        `json:"plan"`
	Approvals  []approval        `json:"approvals"`
	Progress   *execute.Progress `json:"progress,omitempty"`
	Status     string            `json:"status"`
	HelixURL   string            `json:"helix_url"`
}

func (s *Server) view(j *job) jobView {
	j.mu.Lock()
	defer j.mu.Unlock()
	v := jobView{ID: j.ID, FileName: j.FileName, FileSHA: j.FileSHA, RuleSet: j.RuleSet, UploadedAt: j.UploadedAt,
		SheetRows: len(j.sheet.Rows), Headers: j.sheet.Headers, Overrides: j.Overrides, Plan: j.plan,
		Approvals: append([]approval{}, j.approvals...), Status: "review", HelixURL: s.Helix.Base()}
	if j.exec != nil {
		p := j.exec.Snapshot()
		p.Results = nil
		v.Progress = &p
		v.Status = p.State
	}
	return v
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	if j := s.job(w, r); j != nil {
		writeJSON(w, http.StatusOK, s.view(j))
	}
}

func page(r *http.Request, total int) (int, int) {
	off, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	lim, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if lim <= 0 || lim > 500 {
		lim = 50
	}
	if off < 0 || off > total {
		off = total
	}
	end := off + lim
	if end > total {
		end = total
	}
	return off, end
}

func (s *Server) issues(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	j.mu.Lock()
	all := j.plan.Issues
	j.mu.Unlock()
	sev := r.URL.Query().Get("severity")
	var list []plan.Issue
	for _, is := range all {
		if sev == "" || is.Severity == sev {
			list = append(list, is)
		}
	}
	off, end := page(r, len(list))
	writeJSON(w, http.StatusOK, map[string]any{"total": len(list), "offset": off, "items": nonNil(list[off:end])})
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (s *Server) rows(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	j.mu.Lock()
	all := j.plan.Rows
	j.mu.Unlock()
	off, end := page(r, len(all))
	writeJSON(w, http.StatusOK, map[string]any{"total": len(all), "offset": off, "items": nonNil(all[off:end])})
}

func (s *Server) overrides(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	if s.isRunning(j.ID) {
		fail(w, http.StatusConflict, "the job is writing to Helix; overrides are locked")
		return
	}
	var ov plan.Overrides
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&ov); err != nil {
		fail(w, http.StatusBadRequest, "overrides: %v", err)
		return
	}
	if err := s.replan(r.Context(), j, ov); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, s.view(j))
}

func (s *Server) isRunning(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running == id
}

type approveRequest struct {
	PlanHash     string   `json:"plan_hash"`
	Acknowledged []string `json:"acknowledged"`
	RowLimit     int      `json:"row_limit"`
	DryRun       bool     `json:"dry_run"`
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	var req approveRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "approve: %v", err)
		return
	}
	j.mu.Lock()
	p := j.plan
	j.mu.Unlock()
	if req.PlanHash != p.Hash {
		fail(w, http.StatusConflict, "the plan changed since you reviewed it; review again")
		return
	}
	if p.BlockingCount > 0 {
		fail(w, http.StatusUnprocessableEntity, "%d blocking issues must be fixed or overridden first", p.BlockingCount)
		return
	}
	acked := map[string]bool{}
	for _, a := range req.Acknowledged {
		acked[a] = true
	}
	var missing []string
	for _, a := range p.Attention {
		if !acked[a.ID] {
			missing = append(missing, a.ID)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		fail(w, http.StatusUnprocessableEntity, "acknowledge every attention item first: %s", strings.Join(missing, ", "))
		return
	}
	s.mu.Lock()
	if s.running != "" {
		s.mu.Unlock()
		fail(w, http.StatusConflict, "job %s is already writing to Helix", s.running)
		return
	}
	s.running = j.ID
	s.mu.Unlock()

	a := approval{PlanHash: p.Hash, Acknowledged: req.Acknowledged, RowLimit: req.RowLimit, DryRun: req.DryRun, At: time.Now().UTC(), RunID: newID()}
	x := &execute.Executor{API: s.API, Ledger: s.Ledger}
	j.mu.Lock()
	j.approvals = append(j.approvals, a)
	j.exec = x
	j.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			s.running = ""
			s.mu.Unlock()
		}()
		x.Run(context.Background(), p, execute.Options{JobID: j.ID, RunID: a.RunID, RowLimit: req.RowLimit, DryRun: req.DryRun})
		pr := x.Snapshot()
		log.Printf("job %s run %s %s: written=%d failed=%d blocked=%d dry_run=%v %s", j.ID, a.RunID, pr.State, pr.Written, pr.Failed, pr.Blocked, pr.DryRun, pr.Error)
	}()
	writeJSON(w, http.StatusAccepted, s.view(j))
}

func (s *Server) results(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	j.mu.Lock()
	x := j.exec
	j.mu.Unlock()
	if x == nil {
		writeJSON(w, http.StatusOK, map[string]any{"total": 0, "offset": 0, "items": []any{}})
		return
	}
	all := x.Snapshot().Results
	status := r.URL.Query().Get("status")
	var list []execute.RowResult
	for _, res := range all {
		if status == "" || res.Status == status {
			list = append(list, res)
		}
	}
	off, end := page(r, len(list))
	writeJSON(w, http.StatusOK, map[string]any{"total": len(list), "offset": off, "items": nonNil(list[off:end])})
}

func (s *Server) resultsCSV(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	j.mu.Lock()
	x, p := j.exec, j.plan
	j.mu.Unlock()
	if x == nil {
		fail(w, http.StatusNotFound, "the job has not run")
		return
	}
	snap := x.Snapshot()
	var variants []string
	for _, vi := range p.Impact {
		if vi.Scope == "row" {
			variants = append(variants, vi.Variant)
		}
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "migration-"+j.ID+".csv"))
	cw := csv.NewWriter(w)
	head := []string{"sheet_row", "policy_number", "status", "message"}
	for _, v := range variants {
		head = append(head, v+" action", v+" record_id")
	}
	_ = cw.Write(head)
	for _, res := range snap.Results {
		line := []string{strconv.Itoa(res.Row), res.Key, res.Status, res.Message}
		byVariant := map[string]execute.Action{}
		for _, a := range res.Actions {
			byVariant[a.Variant] = a
		}
		for _, v := range variants {
			a := byVariant[v]
			line = append(line, a.Action, a.RecordID)
		}
		_ = cw.Write(line)
	}
	cw.Flush()
}

func (s *Server) variants(w http.ResponseWriter, r *http.Request) {
	leaves, bundle, err := s.Schema.Leaves(r.Context())
	if err != nil {
		fail(w, http.StatusBadGateway, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"bundle": bundle, "variants": leaves})
}

func (s *Server) variant(w http.ResponseWriter, r *http.Request) {
	v, err := s.Schema.Variant(r.Context(), r.PathValue("variant"))
	if err != nil {
		status := http.StatusBadGateway
		if helix.IsStatus(err, 404) {
			status = http.StatusNotFound
		}
		fail(w, status, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		fail(w, http.StatusNotFound, "no route %s %s", r.Method, r.URL.Path)
		return
	}
	if s.Web == nil {
		http.Error(w, "UI not built: run `make ui`", http.StatusNotFound)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	if _, err := fs.Stat(s.Web, path); errors.Is(err, fs.ErrNotExist) {
		path = "index.html" // single-page app
	}
	http.ServeFileFS(w, r, s.Web, path)
}

func (s *Server) entities(w http.ResponseWriter, r *http.Request) {
	ents, err := s.Schema.Entities(r.Context())
	if err != nil {
		fail(w, http.StatusBadGateway, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, ents)
}

func (s *Server) entityFields(w http.ResponseWriter, r *http.Request) {
	fields, err := s.Schema.EntityFields(r.Context(), r.PathValue("entity"))
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, fields)
}

func (s *Server) browsePolicy(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	res, err := s.Browse.Policy(ctx, r.URL.Query().Get("policy_number"))
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) browseTable(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	res, err := s.Browse.Table(r.Context(), q.Get("entity"), q.Get("field"), q.Get("value"), q.Get("after"), limit)
	if err != nil {
		status := http.StatusBadRequest
		var he *helix.Error
		if errors.As(err, &he) && he.Status >= 500 {
			status = http.StatusBadGateway
		}
		fail(w, status, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
