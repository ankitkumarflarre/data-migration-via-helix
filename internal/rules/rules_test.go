package rules

import (
	"strings"
	"testing"
)

func TestCommittedRuleSetHasOnlyTableLocations(t *testing.T) {
	b, err := Load("manatee_fl_select_ho_12_1_25")
	if err != nil {
		t.Fatal(err)
	}
	// 20 confirmed columns: one excluded, one split into two form-specific rules.
	if n := len(b.RuleSet.Rules); n != 20 {
		t.Fatalf("rules = %d, want 20", n)
	}
	if ex := b.RuleSet.Excluded; len(ex) != 1 || ex[0].Column != "BU" || ex[0].Reason == "" {
		t.Fatalf("excluded = %+v, want only the second Policy Number column (BU)", ex)
	}
	for _, r := range b.RuleSet.Rules {
		if len(r.ReportLocations) == 0 {
			t.Errorf("%s has no TABLE location", r.ID)
		}
		for _, l := range r.ReportLocations {
			if !strings.HasSuffix(l, "[TABLE]") {
				t.Errorf("%s keeps a non-table location %q", r.ID, l)
			}
		}
	}
}

func TestReviewedRuleChanges(t *testing.T) {
	b, err := Load("manatee_fl_select_ho_12_1_25")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Rule{}
	for _, r := range b.RuleSet.Rules {
		byID[r.ID] = r
	}
	want := map[string]string{
		"HO-05":  "dwelling.property.us.personal.rated_territory",
		"HO-10":  "dwelling_asset.property.personal.construction_year",
		"HO-11":  "dwelling_asset.property.personal.construction_type",
		"HO-13":  "dwelling.property.us.personal.number_of_stories",
		"HO-13b": "dwelling.property.us.personal.number_of_floor",
		"HO-25":  "wind_mitigation_verification.property.us-fl.personal.safepoint.secondary_water_resistance_flag",
		"HO-26":  "wind_mitigation_verification.property.us-fl.personal.safepoint.opening_protection",
	}
	for id, target := range want {
		if got := byID[id].Target.String(); got != target {
			t.Errorf("%s target = %s, want %s", id, got, target)
		}
	}
	if w := byID["HO-13b"].When; w == nil || w.Column != "B" || w.In[0] != "HO6" {
		t.Errorf("HO-13b condition = %+v", w)
	}
	for _, id := range []string{"HO-05", "HO-10", "HO-11", "HO-13b", "HO-25"} {
		if byID[id].TargetBasis == "" {
			t.Errorf("%s targets a field outside the report but gives no reason", id)
		}
	}
}

func TestConditionMatches(t *testing.T) {
	c := &Condition{Column: "B", In: []string{"HO6"}}
	if !c.Matches(map[string]string{"B": " ho6 "}) || c.Matches(map[string]string{"B": "HO3"}) || c.Matches(nil) {
		t.Fatal("condition matching is wrong")
	}
	var none *Condition
	if !none.Matches(nil) {
		t.Fatal("a nil condition must match every row")
	}
}
