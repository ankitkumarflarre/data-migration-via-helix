package rules

import (
	"encoding/json"
	"testing"
)

const ho = "manatee_fl_select_ho_12_1_25"

func find(b *Bundle, id string) *Rule {
	for i := range b.RuleSet.Rules {
		if b.RuleSet.Rules[i].ID == id {
			return &b.RuleSet.Rules[i]
		}
	}
	return nil
}

func TestStoreAppliesEditsAddsAndReverts(t *testing.T) {
	s, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base, _ := Load(ho)
	b, err := s.Load(ho)
	if err != nil || b.SHA256 != base.SHA256 {
		t.Fatalf("no edits must give the reviewed bundle: %v", err)
	}

	edited := *find(base, "HO-05")
	edited.Disabled = "Territory is recalculated from the address"
	if _, err := s.Append(ho, Change{Kind: ChangeRule, Key: "HO-05", Reason: "test", Rule: &edited}); err != nil {
		t.Fatal(err)
	}
	added := Rule{Column: "T", Header: "Roof Covering",
		Target:    Target{Variant: "wind_mitigation_verification.property.us-fl.personal.safepoint", Field: "roof_covering_code_compliance"},
		Transform: Transform{Map: map[string]string{"FBC Equivalent": "meets_2001_fbc"}}}
	if _, err := s.Append(ho, Change{Kind: ChangeRule, Key: "HO-T", Reason: "test", Rule: &added}); err != nil {
		t.Fatal(err)
	}
	legal := "Manatee Insurance"
	if _, err := s.Append(ho, Change{Kind: ChangeTemplate, Key: "organization|legal_name", Reason: "test", Source: &FieldSource{Const: &legal}}); err != nil {
		t.Fatal(err)
	}

	b, err = s.Load(ho)
	if err != nil {
		t.Fatal(err)
	}
	if b.SHA256 == base.SHA256 {
		t.Fatal("edits must change the bundle hash")
	}
	if find(b, "HO-05").Disabled == "" || find(b, "HO-T") == nil || len(b.RuleSet.Rules) != len(base.RuleSet.Rules)+1 {
		t.Fatalf("edit or added rule missing: %+v", b.RuleSet.Rules)
	}
	if last := b.RuleSet.Rules[len(b.RuleSet.Rules)-1]; last.ID != "HO-T" {
		t.Fatalf("added rules come after the reviewed ones, got %s last", last.ID)
	}
	if got := *b.Templates.Records[0].Fields["legal_name"].Const; got != legal {
		t.Fatalf("template edit not applied: %q", got)
	}
	if find(base, "HO-05").Disabled != "" || base.Templates.Records[0].Fields["legal_name"].Const != nil {
		t.Fatal("applying edits must not change the reviewed bundle")
	}

	// Back to the reviewed default; an added rule is removed.
	for _, c := range []Change{{Kind: ChangeRule, Key: "HO-05", Reason: "undo"}, {Kind: ChangeRule, Key: "HO-T", Reason: "undo"},
		{Kind: ChangeTemplate, Key: "organization|legal_name", Reason: "undo"}} {
		if _, err := s.Append(ho, c); err != nil {
			t.Fatal(err)
		}
	}
	b, _ = s.Load(ho)
	if find(b, "HO-05").Disabled != "" || find(b, "HO-T") != nil || !b.Templates.Records[0].Fields["legal_name"].Generate {
		t.Fatal("revert to default failed")
	}
	if b.SHA256 != base.SHA256 {
		t.Fatal("rules back at the reviewed version must have the reviewed hash")
	}
	changes, _ := s.Changes(ho)
	if len(changes) != 6 || changes[5].Seq != 6 || changes[0].At.IsZero() {
		t.Fatalf("history not kept: %+v", changes)
	}
}

func TestStoreRejectsInvalidChanges(t *testing.T) {
	s, _ := OpenStore(t.TempDir())
	bad := Rule{Column: "T"} // no target
	if _, err := s.Append(ho, Change{Kind: ChangeRule, Key: "HO-T", Reason: "x", Rule: &bad}); err == nil {
		t.Fatal("an incomplete rule must be rejected")
	}
	if _, err := s.Append(ho, Change{Kind: ChangeTemplate, Key: "nope|field", Reason: "x", Source: &FieldSource{}}); err == nil {
		t.Fatal("a change to an unknown template must be rejected")
	}
	if _, err := s.Append(ho, Change{Kind: ChangeRule, Key: "HO-05", Rule: &bad}); err == nil {
		t.Fatal("a change without a reason must be rejected")
	}
	if c, _ := s.Changes(ho); len(c) != 0 {
		t.Fatalf("rejected changes were saved: %+v", c)
	}
}

func TestExportPinsCarriesEdits(t *testing.T) {
	base, _ := Load(ho)
	edited := *find(base, "HO-13b")
	edited.Note = "edited note"
	added := Rule{ID: "HO-T", Column: "T", Header: "Roof Covering",
		Target: Target{Variant: "wind_mitigation_verification.property.us-fl.personal.safepoint", Field: "roof_covering_code_compliance"}}
	b, err := Apply(base, []Change{{Seq: 1, Kind: ChangeRule, Key: "HO-13b", Reason: "x", Rule: &edited}, {Seq: 2, Kind: ChangeRule, Key: "HO-T", Reason: "x", Rule: &added}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := ExportPins(ho, b)
	if err != nil {
		t.Fatal(err)
	}
	if also := p.Pins["13"].Also; len(also) != 1 || also[0].Note != "edited note" || p.Pins["13"].When == nil {
		t.Fatalf("pin 13 wrong: %+v", p.Pins["13"])
	}
	if p.Pins["73"].Exclude == "" {
		t.Fatal("excluded pins must be kept")
	}
	found := false
	for _, a := range p.Additional {
		found = found || (a.ID == "HO-T" && a.Header == "Roof Covering")
	}
	if !found {
		t.Fatalf("added rule not exported: %+v", p.Additional)
	}
	// Unedited: the export equals the committed pins.
	p, _ = ExportPins(ho, base)
	committed, _ := BasePins(ho)
	if !sameJSONForTest(p, committed) {
		t.Fatal("exporting the reviewed rules must reproduce the committed pins")
	}
}

func sameJSONForTest(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
