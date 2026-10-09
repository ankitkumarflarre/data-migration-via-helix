package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
)

const ho = "/api/rules/manatee_fl_select_ho_12_1_25"

func rulesServer(t *testing.T) http.Handler {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	src, err := schema.LoadFileSource(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "describe.json"), "test")
	if err != nil {
		t.Fatal(err)
	}
	store, err := rules.OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{Schema: schema.New(src), Rules: store, jobs: map[string]*job{}}
	return s.Handler()
}

func call(t *testing.T, h http.Handler, method, path string, body any) (int, ruleSetView, string) {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, rd))
	var v ruleSetView
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	return rec.Code, v, rec.Body.String()
}

func ruleByID(v ruleSetView, id string) *ruleView {
	for i := range v.Rules {
		if v.Rules[i].ID == id {
			return &v.Rules[i]
		}
	}
	return nil
}

func TestRulesTabViewEditAddRevertExport(t *testing.T) {
	h := rulesServer(t)
	code, v, body := call(t, h, "GET", ho, nil)
	if code != 200 || len(v.Rules) != 55 || v.SHA != v.ReviewedSHA || v.Headers["B"] != "Form" || len(v.Excluded) != 34 {
		t.Fatalf("view: %d %s", code, body)
	}
	if r := ruleByID(v, "HO-26"); r.Status != statusReviewed || r.TargetType == nil || r.TargetType.Kind != "enum" {
		t.Fatalf("HO-26 view: %+v", r)
	}

	// A value the enum does not allow is refused.
	in := ruleInput{Target: ruleByID(v, "HO-26").Target, Transform: rules.Transform{Map: map[string]string{"Class A": "impact"}}}
	if code, _, body := call(t, h, "PUT", ho+"/rules/HO-26", map[string]any{"rule": in, "reason": "test"}); code != 400 || !strings.Contains(body, "impact") {
		t.Fatalf("invalid enum value accepted: %d %s", code, body)
	}
	// A reason is required.
	in.Transform.Map = map[string]string{"Class A": "hurricane_rated", "Class B": "hurricane_rated"}
	if code, _, _ := call(t, h, "PUT", ho+"/rules/HO-26", map[string]any{"rule": in, "reason": ""}); code != 400 {
		t.Fatal("a change without a reason was accepted")
	}
	code, v, body = call(t, h, "PUT", ho+"/rules/HO-26", map[string]any{"rule": in, "reason": "Class B is impact rated too"})
	if r := ruleByID(v, "HO-26"); code != 200 || r.Status != statusEdited || r.Reviewed == nil || r.Transform.Map["Class B"] != "hurricane_rated" || r.Column != "Z" {
		t.Fatalf("edit: %d %s", code, body)
	}
	if v.SHA == v.ReviewedSHA || len(v.Changes) != 1 {
		t.Fatalf("edit not recorded: %+v", v.Changes)
	}

	// Retarget outside the report: the reason becomes the target basis.
	in = ruleInput{Target: rules.Target{Variant: "dwelling.property.us.personal", Field: "number_of_floor"}}
	_, v, _ = call(t, h, "PUT", ho+"/rules/HO-14", map[string]any{"rule": in, "reason": "test basis"})
	if r := ruleByID(v, "HO-14"); !strings.Contains(r.TargetBasis, "test basis") {
		t.Fatalf("target basis: %+v", r)
	}
	// A location the report names needs no reason.
	in = ruleInput{Target: rules.Target{Variant: "dwelling_asset.property.personal", Field: "number_of_units"}}
	_, v, _ = call(t, h, "PUT", ho+"/rules/HO-14", map[string]any{"rule": in, "reason": "units belong to the asset"})
	if r := ruleByID(v, "HO-14"); r.TargetBasis != "" {
		t.Fatalf("report location given a basis: %+v", r)
	}

	// Links to other records cannot take sheet values.
	add := ruleInput{Column: "t", Header: "Roof Covering", Target: rules.Target{Variant: "dwelling_asset.property.personal", Field: "location_reference"}}
	if code, _, _ := call(t, h, "POST", ho+"/rules", map[string]any{"rule": add, "reason": "phase A"}); code != 400 {
		t.Fatal("a reference target was accepted")
	}
	add.Target = rules.Target{Variant: "wind_mitigation_verification.property.us-fl.personal.safepoint", Field: "roof_covering_code_compliance"}
	add.Transform.Map = map[string]string{"FBC Equivalent": "meets_2001_fbc", "Non-FBC Equivalent": "pre_2001_fbc"}
	code, v, body = call(t, h, "POST", ho+"/rules", map[string]any{"rule": add, "reason": "phase A"})
	if r := ruleByID(v, "HO-T"); code != 200 || r == nil || r.Status != statusAdded || r.Column != "T" {
		t.Fatalf("add: %d %s", code, body)
	}

	// Template: replace the generated issuer name with a fixed value.
	code, v, body = call(t, h, "PUT", ho+"/templates/organization/legal_name", map[string]any{"source": map[string]any{"const": "Manatee Insurance"}, "reason": "real issuer"})
	if code != 200 {
		t.Fatalf("template edit: %d %s", code, body)
	}
	if code, _, _ := call(t, h, "PUT", ho+"/templates/policy.property.us-fl.personal.safepoint/product_reference", map[string]any{"source": map[string]any{"const": "x"}, "reason": "x x"}); code != 400 {
		t.Fatal("a reference template field was editable")
	}

	// Export carries the edits.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", ho+"/export", nil))
	var pins rules.Pins
	err := json.Unmarshal(rec.Body.Bytes(), &pins)
	added := 0
	for _, a := range pins.Additional {
		if a.ID == "HO-T" {
			added++
		}
	}
	if err != nil || added != 1 || pins.Pins["26"].Transform.Map["Class B"] != "hurricane_rated" {
		t.Fatalf("export: %v %s", err, rec.Body.String())
	}

	// Revert HO-26 to the reviewed default and remove the added rule.
	_, v, _ = call(t, h, "POST", ho+"/revert", map[string]any{"kind": "rule", "key": "HO-26", "seq": 0, "reason": "back to reviewed"})
	_, v, _ = call(t, h, "POST", ho+"/revert", map[string]any{"kind": "rule", "key": "HO-T", "seq": 0, "reason": "not yet"})
	if ruleByID(v, "HO-26").Status != statusReviewed || ruleByID(v, "HO-T") != nil {
		t.Fatal("revert to default failed")
	}
	// Revert to an earlier version brings the edit back.
	_, v, _ = call(t, h, "POST", ho+"/revert", map[string]any{"kind": "rule", "key": "HO-26", "seq": 1, "reason": "it was right"})
	if r := ruleByID(v, "HO-26"); r.Status != statusEdited || r.Transform.Map["Class B"] != "hurricane_rated" {
		t.Fatalf("revert to version 1: %+v", r)
	}
}
