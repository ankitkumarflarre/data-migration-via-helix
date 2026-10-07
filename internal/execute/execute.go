// Package execute writes an approved plan to Helix: job records once, then each
// row as one unit — find or create by business key (D3), update only
// sheet-sourced fields of existing records (D12), and undo a row's writes when
// any of them fails.
package execute

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/plan"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// API is the part of the Helix client the executor uses.
type API interface {
	Create(ctx context.Context, variant string, fields map[string]any, idempotencyKey string) (helix.Record, error)
	Get(ctx context.Context, variant, id string) (helix.Record, error)
	Patch(ctx context.Context, variant, id string, version int64, fields map[string]any) (helix.Record, error)
	Delete(ctx context.Context, variant, id string) error
	FindByKey(ctx context.Context, entity, field, value string) ([]helix.Record, error)
}

// Actions on a record.
const (
	Created   = "created"
	Updated   = "updated"
	Unchanged = "unchanged"
	Reused    = "reused"
)

// Options of one run.
type Options struct {
	JobID    string
	RunID    string
	RowLimit int  // 0 = all rows
	DryRun   bool // look up existing records, write nothing
	Workers  int
}

// Action is what happened to one record.
type Action struct {
	Variant  string `json:"variant"`
	Action   string `json:"action"`
	RecordID string `json:"record_id,omitempty"`
}

// RowResult is the outcome of one row.
type RowResult struct {
	Row     int      `json:"row"`
	Key     string   `json:"policy_number"`
	Status  string   `json:"status"` // written | failed | blocked | skipped
	Message string   `json:"message,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

// Counts per variant and action.
type Counts map[string]map[string]int

// Progress is a snapshot of a run.
type Progress struct {
	State      string      `json:"state"` // running | done | aborted
	DryRun     bool        `json:"dry_run"`
	Total      int         `json:"total"`
	Done       int         `json:"done"`
	Written    int         `json:"written"`
	Failed     int         `json:"failed"`
	Blocked    int         `json:"blocked"`
	Counts     Counts      `json:"counts"`
	JobActions []Action    `json:"job_actions"`
	Error      string      `json:"error,omitempty"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
	Results    []RowResult `json:"-"`
}

// Executor runs plans.
type Executor struct {
	API    API
	Ledger *ledger.Ledger

	mu   sync.Mutex
	prog *Progress
}

// Snapshot returns a copy of the progress (Results included, sorted by row).
func (x *Executor) Snapshot() Progress {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.prog == nil {
		return Progress{State: "starting", Counts: Counts{}}
	}
	p := *x.prog
	p.Counts = Counts{}
	for v, m := range x.prog.Counts {
		p.Counts[v] = map[string]int{}
		for a, n := range m {
			p.Counts[v][a] = n
		}
	}
	p.JobActions = append([]Action{}, x.prog.JobActions...)
	p.Results = append([]RowResult{}, x.prog.Results...)
	sort.Slice(p.Results, func(i, j int) bool { return p.Results[i].Row < p.Results[j].Row })
	return p
}

func (x *Executor) count(variant, action string) {
	if x.prog.Counts[variant] == nil {
		x.prog.Counts[variant] = map[string]int{}
	}
	x.prog.Counts[variant][action]++
}

// Run executes the plan synchronously. Call Snapshot from another goroutine for progress.
func (x *Executor) Run(ctx context.Context, p *plan.Plan, o Options) {
	if o.Workers <= 0 {
		o.Workers = 4
	}
	var rows []plan.Row
	blocked := []RowResult{}
	for _, r := range p.Rows {
		if r.Blocked {
			blocked = append(blocked, RowResult{Row: r.Row, Key: r.Key, Status: "blocked", Message: "row has blocking issues"})
			continue
		}
		if o.RowLimit > 0 && len(rows) >= o.RowLimit {
			continue
		}
		rows = append(rows, r)
	}
	x.mu.Lock()
	x.prog = &Progress{State: "running", DryRun: o.DryRun, Total: len(rows), Blocked: len(blocked), Counts: Counts{}, StartedAt: time.Now().UTC(), Results: blocked}
	x.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	finish := func(state, msg string) {
		x.mu.Lock()
		now := time.Now().UTC()
		x.prog.State, x.prog.Error, x.prog.FinishedAt = state, msg, &now
		x.mu.Unlock()
	}

	jobIDs := map[string]string{}
	for _, rec := range p.JobRecords {
		w := &rowWriter{x: x, o: o, ids: jobIDs, policy: ""}
		act, err := w.record(ctx, rec, 0)
		if err != nil {
			finish("aborted", fmt.Sprintf("job record %s: %v", rec.Variant, err))
			return
		}
		x.mu.Lock()
		x.prog.JobActions = append(x.prog.JobActions, act)
		x.count(rec.Variant, act.Action)
		x.mu.Unlock()
	}

	work := make(chan plan.Row)
	var wg sync.WaitGroup
	var fatalOnce sync.Once
	fatal := ""
	for i := 0; i < o.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range work {
				res := x.row(ctx, r, jobIDs, o)
				x.mu.Lock()
				x.prog.Done++
				if res.Status == "written" {
					x.prog.Written++
					for _, a := range res.Actions {
						x.count(a.Variant, a.Action)
					}
				} else {
					x.prog.Failed++
				}
				x.prog.Results = append(x.prog.Results, res.RowResult)
				x.mu.Unlock()
				if res.fatal {
					fatalOnce.Do(func() { fatal = res.Message; cancel() })
				}
			}
		}()
	}
feed:
	for _, r := range rows {
		select {
		case work <- r:
		case <-ctx.Done():
			break feed
		}
	}
	close(work)
	wg.Wait()
	if fatal != "" {
		finish("aborted", "stopped after an authorization or server error: "+fatal)
		return
	}
	finish("done", "")
}

type rowResult struct {
	RowResult
	fatal bool
}

func (x *Executor) row(ctx context.Context, r plan.Row, jobIDs map[string]string, o Options) (out rowResult) {
	out.RowResult = RowResult{Row: r.Row, Key: r.Key}
	ids := map[string]string{}
	for k, v := range jobIDs {
		ids[k] = v
	}
	w := &rowWriter{x: x, o: o, ids: ids, policy: r.Key, sheetRow: r.Row}
	for _, rec := range r.Records {
		act, err := w.record(ctx, rec, r.Row)
		if err != nil {
			out.Status, out.Message = "failed", fmt.Sprintf("%s: %v", rec.Variant, err)
			out.fatal = helix.IsStatus(err, 401) || helix.IsStatus(err, 403) || errors.Is(err, context.Canceled)
			if undo := w.rollback(context.WithoutCancel(ctx)); undo != "" {
				out.Message += "; rollback: " + undo
			}
			return out
		}
		out.Actions = append(out.Actions, act)
	}
	out.Status = "written"
	return out
}

type patchUndo struct {
	variant, id string
	prior       map[string]any
}

type rowWriter struct {
	x        *Executor
	o        Options
	ids      map[string]string // variant → record id resolved so far
	policy   string
	sheetRow int
	created  []ledger.Entry
	patched  []patchUndo
}

func idemKey(runID string, row int, variant string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%d|%s", runID, row, variant)))
	return hex.EncodeToString(sum[:16])
}

// existing finds a record previously written for this key or policy.
func (w *rowWriter) existing(ctx context.Context, rec plan.Record) (*helix.Record, error) {
	if rec.KeyField != "" {
		if e, ok := w.x.Ledger.ByKey(rec.Entity, rec.KeyField, rec.KeyValue); ok {
			got, err := w.x.API.Get(ctx, e.Variant, e.RecordID)
			if err == nil {
				return &got, checkVariant(rec, e.Variant)
			}
			if !helix.IsStatus(err, 404) {
				return nil, err
			}
			_ = w.x.Ledger.Remove(e.RecordID) // deleted outside the tool
		}
		found, err := w.x.API.FindByKey(ctx, rec.Entity, rec.KeyField, rec.KeyValue)
		if err != nil {
			return nil, fmt.Errorf("look up %s=%q: %w", rec.KeyField, rec.KeyValue, err)
		}
		if len(found) == 0 {
			return nil, nil
		}
		if len(found) > 1 {
			return nil, fmt.Errorf("%d records of %s have %s=%q", len(found), rec.Entity, rec.KeyField, rec.KeyValue)
		}
		if err := checkVariant(rec, found[0].Variant); err != nil {
			return nil, err
		}
		got, err := w.x.API.Get(ctx, rec.Variant, found[0].ID)
		return &got, err
	}
	if w.policy == "" {
		return nil, nil
	}
	e, ok := w.x.Ledger.ByPolicy(w.policy, rec.Variant)
	if !ok {
		return nil, nil
	}
	got, err := w.x.API.Get(ctx, rec.Variant, e.RecordID)
	if helix.IsStatus(err, 404) {
		_ = w.x.Ledger.Remove(e.RecordID)
		return nil, nil
	}
	return &got, err
}

func checkVariant(rec plan.Record, variant string) error {
	if variant != "" && variant != rec.Variant {
		return fmt.Errorf("%s=%q already exists as variant %s, not %s", rec.KeyField, rec.KeyValue, variant, rec.Variant)
	}
	return nil
}

func (w *rowWriter) record(ctx context.Context, rec plan.Record, row int) (Action, error) {
	fields := make(map[string]any, len(rec.Fields)+len(rec.Refs))
	for k, v := range rec.Fields {
		fields[k] = v
	}
	for f, to := range rec.Refs {
		id := w.ids[to]
		if id == "" {
			return Action{}, fmt.Errorf("reference %s → %s has no record", f, to)
		}
		fields[f] = id
	}
	cur, err := w.existing(ctx, rec)
	if err != nil {
		return Action{}, err
	}
	if cur != nil {
		w.ids[rec.Variant] = cur.ID
		if rec.Scope == "job" {
			return Action{Variant: rec.Variant, Action: Reused, RecordID: cur.ID}, nil
		}
		patch, prior := map[string]any{}, map[string]any{}
		for _, f := range rec.SheetFields {
			if transform.Canonical(cur.Fields[f]) != transform.Canonical(rec.Fields[f]) {
				patch[f], prior[f] = rec.Fields[f], cur.Fields[f]
			}
		}
		if len(patch) == 0 {
			return Action{Variant: rec.Variant, Action: Unchanged, RecordID: cur.ID}, nil
		}
		if w.o.DryRun {
			return Action{Variant: rec.Variant, Action: Updated, RecordID: cur.ID}, nil
		}
		_, err := w.x.API.Patch(ctx, rec.Variant, cur.ID, cur.Version, patch)
		if helix.IsStatus(err, 409) { // someone else wrote it; read again and retry once
			again, gerr := w.x.API.Get(ctx, rec.Variant, cur.ID)
			if gerr != nil {
				return Action{}, gerr
			}
			_, err = w.x.API.Patch(ctx, rec.Variant, cur.ID, again.Version, patch)
		}
		if err != nil {
			return Action{}, err
		}
		w.patched = append(w.patched, patchUndo{rec.Variant, cur.ID, prior})
		return Action{Variant: rec.Variant, Action: Updated, RecordID: cur.ID}, nil
	}
	if w.o.DryRun {
		w.ids[rec.Variant] = "dry-run:" + rec.Variant
		return Action{Variant: rec.Variant, Action: Created}, nil
	}
	got, err := w.x.API.Create(ctx, rec.Variant, fields, idemKey(w.o.RunID, row, rec.Variant))
	if err != nil {
		return Action{}, err
	}
	w.ids[rec.Variant] = got.ID
	e := ledger.Entry{RecordID: got.ID, Variant: rec.Variant, Entity: rec.Entity, KeyField: rec.KeyField, KeyValue: rec.KeyValue,
		JobID: w.o.JobID, RunID: w.o.RunID, Generated: rec.Generated}
	if rec.Scope == "row" {
		e.PolicyNumber, e.SheetRow = w.policy, w.sheetRow
	}
	if err := w.x.Ledger.Add(e); err != nil {
		return Action{}, fmt.Errorf("created %s but could not record it in the ledger: %w", got.ID, err)
	}
	w.created = append(w.created, e)
	return Action{Variant: rec.Variant, Action: Created, RecordID: got.ID}, nil
}

// rollback deletes what this row created (children first, verified by 404) and
// restores patched fields. It returns a description of anything it could not undo.
func (w *rowWriter) rollback(ctx context.Context) string {
	var problems []string
	for i := len(w.created) - 1; i >= 0; i-- {
		e := w.created[i]
		if err := DeleteVerified(ctx, w.x.API, e.Variant, e.RecordID); err != nil {
			problems = append(problems, fmt.Sprintf("%s/%s: %v", e.Variant, e.RecordID, err))
			continue
		}
		_ = w.x.Ledger.Remove(e.RecordID)
	}
	for i := len(w.patched) - 1; i >= 0; i-- {
		p := w.patched[i]
		cur, err := w.x.API.Get(ctx, p.variant, p.id)
		if err == nil {
			_, err = w.x.API.Patch(ctx, p.variant, p.id, cur.Version, p.prior)
		}
		if err != nil {
			problems = append(problems, fmt.Sprintf("restore %s/%s: %v", p.variant, p.id, err))
		}
	}
	if len(problems) == 0 {
		return ""
	}
	return fmt.Sprint(problems)
}

// DeleteVerified deletes a record and checks it reads back as 404.
func DeleteVerified(ctx context.Context, api API, variant, id string) error {
	if err := api.Delete(ctx, variant, id); err != nil && !helix.IsStatus(err, 404) {
		return err
	}
	if _, err := api.Get(ctx, variant, id); !helix.IsStatus(err, 404) {
		if err == nil {
			return errors.New("still readable after delete")
		}
		return err
	}
	return nil
}

// Cleanup deletes every record the ledger says this tool created, newest first.
func Cleanup(ctx context.Context, api API, l *ledger.Ledger, log func(string)) (deleted, failed int) {
	for _, e := range l.All() {
		if err := DeleteVerified(ctx, api, e.Variant, e.RecordID); err != nil {
			failed++
			log(fmt.Sprintf("FAIL %s %s: %v", e.Variant, e.RecordID, err))
			continue
		}
		_ = l.Remove(e.RecordID)
		deleted++
		log(fmt.Sprintf("deleted %s %s", e.Variant, e.RecordID))
	}
	return deleted, failed
}
