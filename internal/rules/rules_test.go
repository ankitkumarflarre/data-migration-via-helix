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
	if n := len(b.RuleSet.Rules); n != 20 {
		t.Fatalf("rules = %d, want the report's 20 confirmed columns", n)
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
