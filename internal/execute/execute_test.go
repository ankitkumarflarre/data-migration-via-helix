package execute

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/plan"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// fake is an in-memory Helix records API with reference checks and failure injection.
type fake struct {
	mu      sync.Mutex
	seq     int
	recs    map[string]*helix.Record
	entity  map[string]string // variant → entity
	failOn  string            // variant whose create fails
	creates int
	patches int
}

func newFake() *fake {
	return &fake{recs: map[string]*helix.Record{}, entity: map[string]string{
		"organization": "organization", "party": "party", "policy.x": "policy", "dwelling.x": "dwelling"}}
}

func (f *fake) Create(_ context.Context, v string, fields map[string]any, _ string) (helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v == f.failOn {
		return helix.Record{}, &helix.Error{Status: 400, Message: "injected"}
	}
	for k, val := range fields {
		if s, ok := val.(string); ok && strings.HasPrefix(s, "id-") && f.recs[s] == nil {
			return helix.Record{}, &helix.Error{Status: 400, Message: "missing reference " + k}
		}
	}
	f.seq++
	f.creates++
	r := &helix.Record{ID: fmt.Sprintf("id-%d", f.seq), Version: 1, Variant: v, Fields: map[string]any{}}
	for k, val := range fields {
		r.Fields[k] = val
	}
	f.recs[r.ID] = r
	return *r, nil
}

func (f *fake) Get(_ context.Context, v, id string) (helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.recs[id]
	if !ok || r.Variant != v {
		return helix.Record{}, &helix.Error{Status: 404, Message: "not found"}
	}
	return *r, nil
}

func (f *fake) Patch(_ context.Context, v, id string, version int64, fields map[string]any) (helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.recs[id]
	if !ok {
		return helix.Record{}, &helix.Error{Status: 404}
	}
	if r.Version != version {
		return helix.Record{}, &helix.Error{Status: 409, Message: "stale version"}
	}
	for k, val := range fields {
		r.Fields[k] = val
	}
	r.Version++
	f.patches++
	return *r, nil
}

func (f *fake) Delete(_ context.Context, v, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.recs {
		for k, val := range r.Fields {
			if s, ok := val.(string); ok && strings.HasSuffix(k, "_reference") && s == id {
				return &helix.Error{Status: 409, Message: "referenced"}
			}
		}
	}
	if _, ok := f.recs[id]; !ok {
		return &helix.Error{Status: 404}
	}
	delete(f.recs, id)
	return nil
}

func (f *fake) FindByKey(_ context.Context, entity, field, value string) ([]helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []helix.Record
	for _, r := range f.recs {
		if f.entity[r.Variant] == entity && transform.Canonical(r.Fields[field]) == value {
			out = append(out, *r)
		}
	}
	return out, nil
}

func testPlan(rows int, stories int64) *plan.Plan {
	p := &plan.Plan{JobRecords: []plan.Record{{Variant: "organization", Entity: "organization", Scope: "job",
		KeyField: "party_reference", KeyValue: "GEN-ORG", Fields: map[string]any{"party_reference": "GEN-ORG"}, SheetFields: []string{}}}}
	for i := 1; i <= rows; i++ {
		pn := fmt.Sprintf("PN%d", i)
		p.Rows = append(p.Rows, plan.Row{Row: i + 1, Key: pn, Records: []plan.Record{
			{Variant: "party", Entity: "party", Scope: "row", KeyField: "party_reference", KeyValue: "PH-" + pn,
				Fields: map[string]any{"party_reference": "PH-" + pn}, SheetFields: []string{}},
			{Variant: "policy.x", Entity: "policy", Scope: "row", KeyField: "policy_number", KeyValue: pn,
				Fields: map[string]any{"policy_number": pn}, SheetFields: []string{"policy_number"},
				Refs: map[string]string{"issuing_party_reference": "organization", "policyholder_reference": "party"}},
			{Variant: "dwelling.x", Entity: "dwelling", Scope: "row",
				Fields: map[string]any{"number_of_stories": stories, "construction": "Masonry"}, SheetFields: []string{"construction", "number_of_stories"}},
		}})
	}
	return p
}

func newExec(t *testing.T, api API) *Executor {
	l, err := ledger.Open(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	return &Executor{API: api, Ledger: l}
}

func TestRunCreatesThenUpdatesOnlySheetFields(t *testing.T) {
	f := newFake()
	x := newExec(t, f)
	x.Run(context.Background(), testPlan(3, 1), Options{RunID: "r1", Workers: 2})
	p := x.Snapshot()
	if p.State != "done" || p.Written != 3 || f.creates != 1+3*3 || p.Counts["dwelling.x"][Created] != 3 {
		t.Fatalf("first run: %+v creates=%d", p, f.creates)
	}
	// Same data again: nothing written.
	x.Run(context.Background(), testPlan(3, 1), Options{RunID: "r2"})
	p = x.Snapshot()
	if f.creates != 10 || f.patches != 0 || p.Counts["dwelling.x"][Unchanged] != 3 || p.JobActions[0].Action != Reused {
		t.Fatalf("re-run should change nothing: %+v", p)
	}
	// A sheet value changed: the keyless dwelling is found through the ledger and patched.
	x.Run(context.Background(), testPlan(3, 2), Options{RunID: "r3"})
	p = x.Snapshot()
	if f.creates != 10 || f.patches != 3 || p.Counts["dwelling.x"][Updated] != 3 {
		t.Fatalf("update run: %+v creates=%d patches=%d", p.Counts, f.creates, f.patches)
	}
}

func TestFailedRowIsRolledBack(t *testing.T) {
	f := newFake()
	f.failOn = "dwelling.x"
	x := newExec(t, f)
	x.Run(context.Background(), testPlan(2, 1), Options{RunID: "r1"})
	p := x.Snapshot()
	if p.Failed != 2 || p.Written != 0 {
		t.Fatalf("expected two failed rows: %+v", p)
	}
	if len(f.recs) != 1 { // only the job-level organization remains
		t.Fatalf("row records not rolled back: %d left", len(f.recs))
	}
	if n := len(x.Ledger.All()); n != 1 {
		t.Fatalf("ledger should only hold the organization, has %d", n)
	}
}

func TestDryRunWritesNothingAndLimitApplies(t *testing.T) {
	f := newFake()
	x := newExec(t, f)
	x.Run(context.Background(), testPlan(5, 1), Options{RunID: "r1", DryRun: true, RowLimit: 2})
	p := x.Snapshot()
	if f.creates != 0 || p.Total != 2 || p.Counts["policy.x"][Created] != 2 {
		t.Fatalf("dry run: %+v creates=%d", p, f.creates)
	}
}

func TestCleanupRemovesEverythingCreated(t *testing.T) {
	f := newFake()
	x := newExec(t, f)
	x.Run(context.Background(), testPlan(3, 1), Options{RunID: "r1"})
	deleted, failed := Cleanup(context.Background(), f, x.Ledger, func(string) {})
	if failed != 0 || deleted != 10 || len(f.recs) != 0 || len(x.Ledger.All()) != 0 {
		t.Fatalf("cleanup: deleted=%d failed=%d left=%d", deleted, failed, len(f.recs))
	}
}

// TestInstancesOfOneVariantAreLinkedSeparately writes two records of the same
// variant in a row (two coverages) and checks each link points at its own one.
func TestInstancesOfOneVariantAreLinkedSeparately(t *testing.T) {
	f := newFake()
	f.entity["cov.x"], f.entity["ci.x"] = "coverage", "coverage_instance"
	cov := func(code string) plan.Record {
		return plan.Record{ID: "cov.x#" + code, Variant: "cov.x", Entity: "coverage", Scope: "job", KeyField: "coverage_code", KeyValue: code,
			Fields: map[string]any{"coverage_code": code}, SheetFields: []string{}}
	}
	inst := func(code string, limit int64) plan.Record {
		return plan.Record{ID: "ci.x#" + code, Variant: "ci.x", Entity: "coverage_instance", Scope: "row", KeyField: "coverage_instance_reference",
			KeyValue: "CI-PN1-" + code, Fields: map[string]any{"coverage_instance_reference": "CI-PN1-" + code, "limit_amount": limit},
			SheetFields: []string{"limit_amount"}, Refs: map[string]string{"product_coverage_reference": "cov.x#" + code}}
	}
	p := &plan.Plan{JobRecords: []plan.Record{cov("COA"), cov("COC")},
		Rows: []plan.Row{{Row: 2, Key: "PN1", Records: []plan.Record{inst("COA", 200000), inst("COC", 80000)}}}}
	x := newExec(t, f)
	x.Run(context.Background(), p, Options{RunID: "r1"})
	if s := x.Snapshot(); s.State != "done" || s.Written != 1 || s.Counts["ci.x"][Created] != 2 {
		t.Fatalf("run: %+v", s)
	}
	byCode := map[string]string{} // coverage code → record id
	for id, r := range f.recs {
		if r.Variant == "cov.x" {
			byCode[r.Fields["coverage_code"].(string)] = id
		}
	}
	for _, r := range f.recs {
		if r.Variant != "ci.x" {
			continue
		}
		code := strings.TrimPrefix(r.Fields["coverage_instance_reference"].(string), "CI-PN1-")
		if r.Fields["product_coverage_reference"] != byCode[code] {
			t.Errorf("%s links to %v, want %s", r.Fields["coverage_instance_reference"], r.Fields["product_coverage_reference"], byCode[code])
		}
	}
}
