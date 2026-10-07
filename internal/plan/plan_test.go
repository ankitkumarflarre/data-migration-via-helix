package plan

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/ankitkumarflarre/datamigration/internal/excel"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
)

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func testSchema(t *testing.T) *schema.Schema {
	t.Helper()
	src, err := schema.LoadFileSource(filepath.Join(repoRoot(), "testdata", "describe.json"), "test-bundle")
	if err != nil {
		t.Fatal(err)
	}
	return schema.New(src)
}

// workbook builds a small HO-shaped Policy Data sheet with the rule columns.
func workbook(t *testing.T, rows [][]any) *excel.Sheet {
	t.Helper()
	b, err := rules.Load("manatee_fl_select_ho_12_1_25")
	if err != nil {
		t.Fatal(err)
	}
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "Policy Data")
	headers := map[string]string{"B": "Form", "AP": "Sinkhole Coverage"}
	for _, r := range b.RuleSet.Rules {
		headers[r.Column] = r.Header
	}
	for col, h := range headers {
		f.SetCellValue("Policy Data", col+"1", h)
	}
	for i, r := range rows {
		for j := 0; j+1 < len(r); j += 2 {
			f.SetCellValue("Policy Data", r[j].(string)+itoa(i+2), r[j+1])
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	s, err := excel.Read(&buf, "Policy Data", "A")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func itoa(i int) string { return string(rune('0'+i/10%10)) + string(rune('0'+i%10)) }

func row(pn string, extra ...any) []any {
	base := []any{"A", pn, "B", "HO3", "C", "Miami-dade", "E", 34, "F", 46149, "J", 1996, "K", "Masonry", "L", 2,
		"M", 1, "N", "1 to 4", "P", "No", "S", "Ungraded", "V", "Level C: 8d @ 6/6", "W", "Hip Roof", "Y", "SWR",
		"Z", "Class A", "AA", "Greater than or equal to 100", "AB", 100, "AS", "No", "AX", 0, "AP", "No"}
	return append(base, extra...)
}

func build(t *testing.T, s *excel.Sheet, ov Overrides) *Plan {
	t.Helper()
	b, _ := rules.Load("manatee_fl_select_ho_12_1_25")
	p, err := Build(context.Background(), Input{Bundle: b, Sheet: s, FileSHA: "f", Overrides: ov, Schema: testSchema(t)})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func find(recs []Record, variant string) *Record {
	for i := range recs {
		if recs[i].Variant == variant {
			return &recs[i]
		}
	}
	return nil
}

func TestPlanMapsConfirmedColumnsAndTemplates(t *testing.T) {
	p := build(t, workbook(t, [][]any{row("TEST0001")}), Overrides{})
	if p.BlockingCount != 0 {
		t.Fatalf("unexpected blocking issues: %+v", p.Issues)
	}
	r := p.Rows[0]
	pol := find(r.Records, "policy.property.us-fl.personal.safepoint")
	if pol == nil || pol.KeyValue != "TEST0001" || pol.Fields["effective_date"] != "2026-05-07" ||
		pol.Fields["program_code"] != "safepoint" || pol.Fields["personal_policy_form"] != "ho_3" ||
		pol.Fields["sinkhole_coverage_option"] != "catastrophic_ground_cover_collapse_only" || pol.Refs["product_reference"] != "product" {
		t.Fatalf("policy record wrong: %+v", pol)
	}
	d := find(r.Records, "dwelling.property.us.personal")
	if d.Fields["number_of_units"] != int64(4) || d.Fields["secondary_water_resistance"] != true || d.Fields["year_built"] != int64(1996) {
		t.Fatalf("dwelling wrong: %+v", d.Fields)
	}
	if w := find(r.Records, "wind_mitigation_verification.property.us-fl.personal.safepoint"); w.Fields["roof_shape"] != "hip" ||
		w.Refs["dwelling_asset_reference"] != "dwelling_asset.property.personal" {
		t.Fatalf("wmv wrong: %+v", w)
	}
	if la := find(r.Records, "location_address.property.us.personal"); la.Fields["county"] != "MIAMI-DADE" {
		t.Fatalf("county not upper-cased: %+v", la.Fields)
	}
	if l := find(r.Records, "line.property.us.personal"); l.Fields["consent_to_rate"] != false {
		t.Fatalf("consent_to_rate: %+v", l.Fields)
	}
	if len(p.JobRecords) != 2 || p.JobRecords[0].Variant != "organization" || p.JobRecords[1].Fields["effective_date"] != "2026-05-07" {
		t.Fatalf("job records: %+v", p.JobRecords)
	}
	// Order: everything referenced comes first.
	pos := map[string]int{}
	for i, rec := range r.Records {
		pos[rec.Variant] = i
	}
	if !(pos["party"] < pos["policy.property.us-fl.personal.safepoint"] && pos["location.property.us.personal"] < pos["dwelling_asset.property.personal"] &&
		pos["dwelling_asset.property.personal"] < pos["wind_mitigation_verification.property.us-fl.personal.safepoint"]) {
		t.Fatalf("bad write order: %v", pos)
	}
	att := map[string]bool{}
	for _, a := range p.Attention {
		att[a.ID] = true
	}
	for _, id := range []string{"rule:HO-05", "rule:HO-23", "template:dwelling_asset.property.personal|dwelling_type"} {
		if !att[id] {
			t.Errorf("missing attention item %s (have %v)", id, att)
		}
	}
}

func TestPlanIsDeterministic(t *testing.T) {
	s := workbook(t, [][]any{row("TEST0001"), row("TEST0002")})
	a, b := build(t, s, Overrides{}), build(t, s, Overrides{})
	if a.Hash != b.Hash {
		t.Fatal("same inputs gave different hashes")
	}
	c := build(t, s, Overrides{Rules: map[string]RuleOverride{"HO-05": {Exclude: true}}})
	if c.Hash == a.Hash {
		t.Fatal("an override must change the hash")
	}
}

func TestPlanIssuesAndOverrides(t *testing.T) {
	s := workbook(t, [][]any{row("TEST0001", "N", "lots"), row("TEST0002", "BU", "OTHER"), row("TEST0003", "BU", "TEST0003")})
	p := build(t, s, Overrides{})
	if !p.Rows[0].Blocked || !p.Rows[1].Blocked || p.Rows[2].Blocked {
		t.Fatalf("blocked flags wrong: %+v", p.Issues)
	}
	// Exclude Territory: loss_ratio_analysis is no longer written. Retarget Number of Units.
	units := rules.Target{Variant: "dwelling_asset.property.personal", Field: "number_of_units"}
	p = build(t, s, Overrides{Rules: map[string]RuleOverride{"HO-05": {Exclude: true}, "HO-14": {Target: &units, Map: map[string]string{"lots": "9", "1 to 4": "4"}}}})
	if find(p.Rows[0].Records, "loss_ratio_analysis") != nil {
		t.Fatal("excluded rule still writes its variant")
	}
	if da := find(p.Rows[0].Records, "dwelling_asset.property.personal"); da.Fields["number_of_units"] != int64(9) {
		t.Fatalf("override not applied: %+v", da.Fields)
	}
	p = build(t, s, Overrides{Templates: map[string]string{"organization|legal_name": "Manatee Insurance"}})
	if p.JobRecords[0].Fields["legal_name"] != "Manatee Insurance" {
		t.Fatalf("template override not applied: %+v", p.JobRecords[0].Fields)
	}
}

// TestRealHOWorkbook runs against the confidential workbook when it is present locally.
func TestRealHOWorkbook(t *testing.T) {
	path := os.Getenv("HO_WORKBOOK")
	if path == "" {
		path = filepath.Join(repoRoot(), "..", "pythonProjects", "MappingSQLData", "Confidential - Manatee FL Select HO Rater Effective 12.1.25.xlsm")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Skip("HO workbook not available")
	}
	defer f.Close()
	s, err := excel.Read(f, "Policy Data", "A")
	if err != nil {
		t.Fatal(err)
	}
	p := build(t, s, Overrides{})
	if p.RowCount != 1394 {
		t.Fatalf("rows = %d, want 1394", p.RowCount)
	}
	if p.BlockingCount != 0 {
		t.Fatalf("blocking issues on the real file: %d, first: %+v", p.BlockingCount, p.Issues[0])
	}
	for _, vi := range p.Impact {
		t.Logf("%-62s %-5s records=%d fields=%d", vi.Variant, vi.Scope, vi.Records, len(vi.Fields))
	}
}
