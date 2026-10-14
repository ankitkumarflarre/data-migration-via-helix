package quotesync

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/quote"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

func sampleQuote() *quote.Quote {
	return &quote.Quote{ID: "0123456789abcdef01234567", Number: "Q-0123456789", Version: 3, Values: map[string]any{
		"DwellingInput.Form": "HO3", "PolicyInput.EffectiveDate": "2026-06-01", "LocationInput.County": "Citrus",
		"DwellingInput.UseType": "Seasonal3to6", "DwellingInput.OccupancyType": "Owner", "DwellingInput.YearBuilt": 1990.0,
		"DwellingInput.Construction": "MasonryVeneer", "DwellingInput.RoofDeckAttachment": "A6d@612", "DwellingInput.Sprinkler": "None",
		"DwellingInput.RoofGeometry": "Hip", "DwellingInput.SecondaryWaterResistance": "No", "DwellingInput.WindSpeedDesign": "120 mph+",
		"DwellingInput.HurricaneDeductible": "2%", "DwellingInput.Deductible": "1000", "CoverageADwellingInput.Limit": 300000.0,
		"CoverageCPersonalPropertyHO3Input.Limit": 150000.0, "OrdinanceOrLawInput.Limit": "50", "SinkholeInput.Indicator": true,
		"LossAssessmentInput.Indicator": true, "PolicyDetails.ConsentToRate": false, "LimitedFungiBacteriaInput.Limit": "25/50",
		"AccountInput.FirstName": "Ada", "AccountInput.MiddleName": "B", "AccountInput.LastName": "Lovelace", "AccountInput.State": "FL",
		"LocationInput.Address1": "1 Main St", "LocationInput.City": "Inverness", "LocationInput.State": "FL", "LocationInput.ZipCode": "34450",
		"Underwriting.Notes": "Dog is a poodle.", "Underwriting.HotTub": true, "Underwriting.animals": true,
	}, Collections: map[string][]map[string]any{
		"losses": {{"LossInput.LossType": "Water", "LossInput.DateOfLoss": "2024-01-02", "LossInput.AmountPaid": 1200.0}},
	}}
}

func TestCellsSpellValuesTheRaterWay(t *testing.T) {
	c := Cells(sampleQuote())
	want := map[string]string{"A": "Q-0123456789", "B": "HO3", "C": "CITRUS", "D": "1", "F": "2026-06-01", "H": "Seasonal", "I": "O",
		"J": "1990", "K": "Masonry Veneer", "R": "Neither", "V": "Level A: 6d @ 6/12", "W": "Hip", "Y": "No SWR",
		"AA": "Greater than or equal to 120", "AH": "0.02", "AC": "300000", "AE": "150000", "AK": "0.5", "AP": "Yes",
		"AL": "Increase to $2,000", "AX": "0", "AQ": "25000",
		"Q:FullName": "Ada B Lovelace", "Q:RiskAddress": "1 Main St, Inverness, FL 34450", "losses.1.LossInput.LossType": "Water",
		"losses.1.LossInput.AmountPaid": "1200", "AccountInput.FirstName": "Ada", "Underwriting.animals": "Yes"}
	for col, v := range want {
		if c[col] != v {
			t.Errorf("cell %s = %q, want %q", col, c[col], v)
		}
	}
	if _, ok := c["AccountInput.State"]; ok {
		t.Error("a state without an address must not be written")
	}
	if !strings.Contains(c["Q:UnderwritingRemarks"], "Dog is a poodle.\n"+remarksMarker+"\nHot tub on premises? Yes") {
		t.Errorf("remarks = %q", c["Q:UnderwritingRemarks"])
	}
}

func TestValuesReadCellsBack(t *testing.T) {
	q := sampleQuote()
	v, colls, notes := Values(Cells(q))
	for _, k := range []string{"DwellingInput.Form", "PolicyInput.EffectiveDate", "DwellingInput.OccupancyType", "DwellingInput.YearBuilt",
		"DwellingInput.Construction", "DwellingInput.RoofDeckAttachment", "DwellingInput.Sprinkler", "DwellingInput.RoofGeometry",
		"DwellingInput.SecondaryWaterResistance", "DwellingInput.WindSpeedDesign", "DwellingInput.HurricaneDeductible", "OrdinanceOrLawInput.Limit",
		"SinkholeInput.Indicator", "LossAssessmentInput.Indicator", "LimitedFungiBacteriaInput.Limit", "AccountInput.FirstName",
		"AccountInput.MiddleName", "Underwriting.Notes", "Underwriting.HotTub", "Underwriting.animals", "DwellingInput.UseType"} {
		if transform.Canonical(v[k]) != transform.Canonical(q.Values[k]) {
			t.Errorf("%s = %v, want %v", k, v[k], q.Values[k])
		}
	}
	if v["LocationInput.County"] != "CITRUS" || v["LossAssessmentInput.Limit"] != 2000.0 || v["DwellingInput.BuildingType"] != "Dwelling" {
		t.Errorf("county %v, loss assessment limit %v, building type %v", v["LocationInput.County"], v["LossAssessmentInput.Limit"], v["DwellingInput.BuildingType"])
	}
	if len(colls["losses"]) != 1 || colls["losses"][0]["LossInput.AmountPaid"] != 1200.0 {
		t.Errorf("losses = %v", colls["losses"])
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v", notes)
	}
	// Rater spellings the quote does not write read back too.
	v, _, _ = Values(map[string]string{"W": "Hip Roof", "H": "Seasonal", "AQ": "$10,000 Included", "S": "Ungraded", "B": "HO6"})
	if v["DwellingInput.RoofGeometry"] != "Hip" || v["DwellingInput.UseType"] != "Seasonal6Mths" || v["LimitedFungiBacteriaInput.Limit"] != "10/50" {
		t.Errorf("rater spellings: %v", v)
	}
	if _, ok := v["DwellingInput.BuildingCodeEffectivenessGrading"]; ok || v["DwellingInput.BuildingType"] != "Condo" {
		t.Errorf("Ungraded must be left empty; HO6 is a condo: %v", v)
	}
}

func TestImportedCellsSurviveButQuoteDecides(t *testing.T) {
	q := sampleQuote()
	q.Source = &quote.Source{PolicyNumber: "TEST0001", Cells: map[string]string{"A": "TEST0001", "D": "3", "E": "38", "K": "Frame", "Q:FullName": "Old"}}
	c := Cells(q)
	if c["A"] != "TEST0001" || c["D"] != "3" || c["E"] != "38" || c["K"] != "Masonry Veneer" || c["Q:FullName"] != "Ada B Lovelace" {
		t.Errorf("cells = A %q D %q E %q K %q name %q", c["A"], c["D"], c["E"], c["K"], c["Q:FullName"])
	}
}

// fakeAPI is an in-memory Helix.
type fakeAPI struct {
	mu     sync.Mutex
	seq    int
	recs   map[string]*helix.Record
	entity func(variant string) string
}

func (f *fakeAPI) Create(_ context.Context, v string, fields map[string]any, _ string) (helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	r := &helix.Record{ID: fmt.Sprintf("id-%d", f.seq), Version: 1, Variant: v, Fields: map[string]any{}}
	for k, val := range fields {
		r.Fields[k] = val
	}
	f.recs[r.ID] = r
	return *r, nil
}

func (f *fakeAPI) Get(_ context.Context, v, id string) (helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.recs[id]; ok && r.Variant == v {
		return *r, nil
	}
	return helix.Record{}, &helix.Error{Status: 404, Message: "not found"}
}

func (f *fakeAPI) Patch(_ context.Context, v, id string, version int64, fields map[string]any) (helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.recs[id]
	if r.Version != version {
		return helix.Record{}, &helix.Error{Status: 409}
	}
	for k, val := range fields {
		r.Fields[k] = val
	}
	r.Version++
	return *r, nil
}

func (f *fakeAPI) Delete(_ context.Context, v, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.recs, id)
	return nil
}

func (f *fakeAPI) FindByKey(_ context.Context, entity, field, value string) ([]helix.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []helix.Record
	for _, r := range f.recs {
		if f.entity(r.Variant) == entity && transform.Canonical(r.Fields[field]) == value {
			out = append(out, *r)
		}
	}
	return out, nil
}

func newSyncer(t *testing.T) (*Syncer, *fakeAPI) {
	_, file, _, _ := runtime.Caller(0)
	src, err := schema.LoadFileSource(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "describe.json"), "test-bundle")
	if err != nil {
		t.Fatal(err)
	}
	l, err := ledger.Open(filepath.Join(t.TempDir(), "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	qs, err := quote.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	api := &fakeAPI{recs: map[string]*helix.Record{}, entity: func(v string) string { return src.Descs[v].Entity }}
	return &Syncer{Quotes: qs, Schema: schema.New(src), API: api, Ledger: l,
		Bundle: func() (*rules.Bundle, error) { return rules.Load(RuleSet) }}, api
}

func create(t *testing.T, s *Syncer) *quote.Quote {
	t.Helper()
	q, err := s.Quotes.Create(map[string]any{"Quote.Product": "Manatee FL Select PersonalHome", "Quote.State": "FL",
		"DwellingInput.Form": "HO3", "PolicyInput.EffectiveDate": "2026-06-01"})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func TestSyncWritesAndUpdatesTheSamePolicy(t *testing.T) {
	s, api := newSyncer(t)
	q := create(t, s)
	st := s.Sync(context.Background(), q.ID)
	if st.Status != Synced {
		t.Fatalf("first sync: %s %s %v", st.Status, st.Message, st.Warnings)
	}
	first := len(api.recs)
	if first == 0 || st.Counts["created"] == 0 {
		t.Fatalf("nothing written: %+v", st)
	}
	var policy *helix.Record
	for _, r := range api.recs {
		if r.Variant == "policy.property.us-fl.personal.safepoint" {
			policy = r
		}
	}
	if policy == nil || policy.Fields["policy_number"] != q.Number || policy.Fields["effective_date"] != "2026-06-01" {
		t.Fatalf("policy = %+v", policy)
	}
	// Later pages change the dwelling: the same policy's records are updated, nothing is duplicated.
	q.Values["DwellingInput.YearBuilt"] = 1990.0
	q.Values["DwellingInput.Construction"] = "Masonry"
	writeValues(t, s, q)
	st = s.Sync(context.Background(), q.ID)
	if st.Status != Synced {
		t.Fatalf("second sync: %s %s", st.Status, st.Message)
	}
	if st.Counts["created"] != 0 || st.Counts["updated"] == 0 || len(api.recs) != first {
		t.Errorf("second sync should update in place: %v, %d → %d records", st.Counts, first, len(api.recs))
	}
	var asset *helix.Record
	for _, r := range api.recs {
		if r.Variant == "dwelling_asset.property.personal" {
			asset = r
		}
	}
	if asset == nil || transform.Canonical(asset.Fields["construction_year"]) != "1990" || asset.Fields["construction_type"] != "Masonry" {
		t.Fatalf("dwelling asset = %+v", asset)
	}
	got, _ := s.Quotes.Get(q.ID)
	if got.Helix == nil || got.Helix.Status != Synced || got.Version != q.Version {
		t.Errorf("sync state %+v, version %d → %d", got.Helix, q.Version, got.Version)
	}
}

// writeValues stores values directly, as a save of several pages would.
func writeValues(t *testing.T, s *Syncer, q *quote.Quote) {
	t.Helper()
	src := quote.Source{Kind: "test", PolicyNumber: q.Number, ImportedAt: time.Now()}
	imported, err := s.Quotes.Import(src, q.Values, q.Collections)
	if err != nil {
		t.Fatal(err)
	}
	*q = *imported
}

func TestImportReadsAMigratedPolicyBack(t *testing.T) {
	s, _ := newSyncer(t)
	q := create(t, s)
	q.Values["DwellingInput.YearBuilt"] = 1975.0
	q.Values["DwellingInput.Construction"] = "Frame"
	q.Values["DwellingInput.OccupancyType"] = "Tenant"
	q.Values["DwellingInput.UseType"] = "Primary"
	q.Values["SinkholeInput.Indicator"] = true
	writeValues(t, s, q)
	if st := s.Sync(context.Background(), q.ID); st.Status != Synced {
		t.Fatalf("sync: %s %s", st.Status, st.Message)
	}
	cells, _, err := s.ReadCells(context.Background(), q.PolicyNumber())
	if err != nil {
		t.Fatal(err)
	}
	for col, want := range map[string]string{"B": "HO3", "F": "2026-06-01", "J": "1975", "K": "Frame", "H": "Primary", "I": "T", "AP": "Yes"} {
		if cells[col] != want {
			t.Errorf("cell %s = %q, want %q", col, cells[col], want)
		}
	}
	if _, _, err := s.ReadCells(context.Background(), "NOPE"); err == nil {
		t.Error("a missing policy must be reported")
	}
}

func TestMergeLeavesOutWhatHelixDoesNotHave(t *testing.T) {
	s, _ := newSyncer(t)
	base, _ := rules.Load(RuleSet)
	q := sampleQuote()
	b, notes := Merge(context.Background(), base, q, s.Schema)
	if len(b.RuleSet.Rules) <= len(base.RuleSet.Rules) {
		t.Error("no quote rules were added")
	}
	for _, r := range b.RuleSet.Rules {
		if r.Target.Variant == "person" {
			t.Error("the test schema has no person; its rules must be left out")
		}
	}
	if len(notes) == 0 {
		t.Error("left-out records must be reported")
	}
	if len(base.RuleSet.Rules) != 55 {
		t.Error("Merge must not change the reviewed rule set")
	}
}
