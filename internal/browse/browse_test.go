package browse

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
)

type rec struct {
	entity string
	helix.Record
}

type fakeReader struct{ recs []rec }

func (f *fakeReader) List(_ context.Context, entity string, q helix.ListQuery) ([]helix.Record, string, error) {
	var out []helix.Record
	field, raw, _ := strings.Cut(q.Where, "=")
	var want string
	if err := json.Unmarshal([]byte(raw), &want); err != nil {
		want = raw
	}
	for _, r := range f.recs {
		if r.entity != entity {
			continue
		}
		if q.Where == "" || (field == entity+"_id" && r.ID == want) || r.Fields[field] == want {
			out = append(out, r.Record)
		}
	}
	return out, "", nil
}

func (f *fakeReader) Get(_ context.Context, variant, id string) (helix.Record, error) {
	for _, r := range f.recs {
		if r.ID == id && r.Variant == variant {
			return r.Record, nil
		}
	}
	return helix.Record{}, &helix.Error{Status: 404}
}

func r(entity, variant, id string, fields map[string]any) rec {
	return rec{entity, helix.Record{ID: id, Variant: variant, Version: 1, Fields: fields}}
}

func testSchema(t *testing.T) *schema.Schema {
	_, file, _, _ := runtime.Caller(0)
	src, err := schema.LoadFileSource(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "describe.json"), "test")
	if err != nil {
		t.Fatal(err)
	}
	return schema.New(src)
}

func TestPolicyTraversal(t *testing.T) {
	const pol = "policy.property.us-fl.personal.safepoint"
	fr := &fakeReader{recs: []rec{
		r("organization", "organization", "org", map[string]any{"party_reference": "ORG"}),
		r("product", "product", "prod", map[string]any{"product_code": "P"}),
		r("party", "party", "ph1", map[string]any{"party_reference": "PH-1"}),
		r("party", "party", "ph2", map[string]any{"party_reference": "PH-2"}),
		r("policy", pol, "pol1", map[string]any{"policy_number": "PN1", "product_reference": "prod", "issuing_party_reference": "org", "policyholder_reference": "ph1"}),
		r("policy", pol, "pol2", map[string]any{"policy_number": "PN2", "product_reference": "prod", "issuing_party_reference": "org", "policyholder_reference": "ph2"}),
		r("location", "location.property.us.personal", "loc1", map[string]any{"location_identifier": "LOC-PN1", "party_reference": "ph1"}),
		r("dwelling_asset", "dwelling_asset.property.personal", "da1", map[string]any{"location_reference": "loc1"}),
		r("wind_mitigation_verification", "wind_mitigation_verification.property.us-fl.personal.safepoint", "wmv1", map[string]any{"dwelling_asset_reference": "da1"}),
		r("dwelling", "dwelling.property.us.personal", "dw1", map[string]any{"year_built": float64(1996)}),
		r("dwelling", "dwelling.property.us.personal", "dw2", map[string]any{"year_built": float64(2001)}),
	}}
	l, err := ledger.Open(filepath.Join(t.TempDir(), "l.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	_ = l.Add(ledger.Entry{RecordID: "loc1", Variant: "location.property.us.personal", Entity: "location", PolicyNumber: "PN1", SheetRow: 2})
	_ = l.Add(ledger.Entry{RecordID: "dw1", Variant: "dwelling.property.us.personal", Entity: "dwelling", PolicyNumber: "PN1", SheetRow: 2, Generated: []string{"x"}})
	_ = l.Add(ledger.Entry{RecordID: "gone", Variant: "line.property.us.personal", Entity: "line", PolicyNumber: "PN1", SheetRow: 2})

	b := &Browser{Helix: fr, Schema: testSchema(t), Ledger: l}
	res, err := b.Policy(context.Background(), "PN1")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, tb := range res.Tables {
		for _, it := range tb.Records {
			got[it.ID] = true
		}
	}
	for _, id := range []string{"pol1", "org", "prod", "ph1", "loc1", "da1", "wmv1", "dw1"} {
		if !got[id] {
			t.Errorf("missing %s; got %v", id, got)
		}
	}
	for _, id := range []string{"pol2", "ph2", "dw2"} {
		if got[id] {
			t.Errorf("%s belongs to another policy but was returned", id)
		}
	}
	if res.Tables[0].Variant != pol {
		t.Errorf("policy should be the first table, got %s", res.Tables[0].Variant)
	}
	if len(res.Notes) == 0 || !strings.Contains(strings.Join(res.Notes, " "), "no longer exists") {
		t.Errorf("stale ledger entry not reported: %v", res.Notes)
	}

	empty, _ := b.Policy(context.Background(), "NOPE")
	if empty.Records != 0 || len(empty.Notes) == 0 {
		t.Errorf("unknown policy: %+v", empty)
	}
}

func TestTableFilter(t *testing.T) {
	fr := &fakeReader{recs: []rec{
		r("dwelling", "dwelling.property.us.personal", "dw1", map[string]any{"construction": "Masonry", "year_built": float64(1996)}),
		r("dwelling", "dwelling.property.us.personal", "dw2", map[string]any{"construction": "Frame", "year_built": float64(2001)}),
	}}
	b := &Browser{Helix: fr, Schema: testSchema(t)}
	all, err := b.Table(context.Background(), "dwelling", "", "", "", 0)
	if err != nil || len(all.Records) != 2 || all.Where != "" {
		t.Fatalf("all: %+v %v", all, err)
	}
	one, err := b.Table(context.Background(), "dwelling", "construction", "Masonry", "", 0)
	if err != nil || len(one.Records) != 1 || one.Where != `construction="Masonry"` {
		t.Fatalf("filtered: %+v %v", one, err)
	}
	if q, _ := b.Table(context.Background(), "dwelling", "year_built", "1996", "", 0); q.Where != "year_built=1996" {
		t.Fatalf("integer values must not be quoted: %q", q.Where)
	}
	if _, err := b.Table(context.Background(), "dwelling", "nope", "x", "", 0); err == nil {
		t.Fatal("unknown column accepted")
	}
}
