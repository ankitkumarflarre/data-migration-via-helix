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
	// 20 confirmed columns (one excluded, one split by form) and 35 rules for
	// the Review / Not found columns of the under-review analysis.
	if n := len(b.RuleSet.Rules); n != 55 {
		t.Fatalf("rules = %d, want 55", n)
	}
	for _, r := range b.RuleSet.Rules {
		if !strings.HasPrefix(r.ReportStatus, "Confirmed") {
			continue // rules for non-confirmed columns name no report location
		}
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

// TestEveryColumnIsAccounted checks that each column of the Policy Data sheet
// (A–CI) has a rule, is excluded with a reason, or is read by a template.
func TestEveryColumnIsAccounted(t *testing.T) {
	b, err := Load("manatee_fl_select_ho_12_1_25")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, r := range b.RuleSet.Rules {
		seen[r.Column] = "rule " + r.ID
	}
	for _, e := range b.RuleSet.Excluded {
		if e.Reason == "" {
			t.Errorf("column %s is excluded without a reason", e.Column)
		}
		if prev, dup := seen[e.Column]; dup {
			t.Errorf("column %s is excluded but also has %s", e.Column, prev)
		}
		seen[e.Column] = "excluded"
	}
	for _, rec := range b.Templates.Records {
		for _, src := range rec.Fields {
			if src.Col != "" && seen[src.Col] == "" {
				seen[src.Col] = "template"
			}
		}
	}
	for n := 1; n <= colNum("CI"); n++ {
		c := ""
		for x := n; x > 0; x = (x - 1) / 26 {
			c = string(rune('A'+(x-1)%26)) + c
		}
		if seen[c] == "" {
			t.Errorf("column %s has no rule, template or exclusion", c)
		}
	}
	off := 0
	for _, r := range b.RuleSet.Rules {
		if r.Disabled != "" {
			off++
		}
	}
	if len(b.RuleSet.Excluded) != 34 || off != 6 {
		t.Fatalf("excluded = %d, switched off = %d; want 34 and 6", len(b.RuleSet.Excluded), off)
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
		"HO-16":  "dwelling_feature.property.personal#burglar_alarm.value",
		"HO-29":  "coverage_instance.property.us.personal#coverage_a.limit_amount",
		"HO-34":  "coverage_deductible.property.us-fl.personal.safepoint#hurricane.florida_hurricane_deductible_option",
		"HO-04":  "policy_term.term_number",
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
