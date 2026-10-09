package rules

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Pin is the reviewed choice for one confirmed report row (see cmd/rulegen).
type Pin struct {
	Target        Target     `json:"target"`
	Transform     Transform  `json:"transform"`
	When          *Condition `json:"when,omitempty"`
	OutsideReport string     `json:"outside_report,omitempty"` // why the target is not a report location
	Attention     string     `json:"attention,omitempty"`
	Note          string     `json:"note,omitempty"`
	Disabled      string     `json:"disabled,omitempty"` // the rule is generated but switched off
	Exclude       string     `json:"exclude,omitempty"`  // why the confirmed row gets no rule
	Also          []ExtraPin `json:"also,omitempty"`
}

// ExtraPin is a further rule for the same column; its id gets Suffix.
type ExtraPin struct {
	Suffix string `json:"suffix"`
	Pin
}

// AdditionalRule is a rule for a column that is not a confirmed report row,
// e.g. one added in the Rules tab.
type AdditionalRule struct {
	ID     string `json:"id"`
	Column string `json:"excel_column"`
	Header string `json:"header"`
	Pin
}

// Pins is the reviewed input of rulegen.
type Pins struct {
	RuleSet       string            `json:"rule_set"`
	Version       string            `json:"version"`
	IDPrefix      string            `json:"id_prefix"`
	Sheet         string            `json:"sheet"`
	SchemaRenames map[string]string `json:"schema_renames"`
	Pins          map[string]Pin    `json:"pins"` // by report "#"
	Additional    []AdditionalRule  `json:"additional_rules,omitempty"`
}

// BasePins returns the committed pins file of a rule set.
func BasePins(name string) (*Pins, error) {
	b, err := files.ReadFile("rulesets/" + name + ".pins.json")
	if err != nil {
		return nil, fmt.Errorf("rule set %q has no pins file", name)
	}
	var p Pins
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("pins %s: %w", name, err)
	}
	return &p, nil
}

// PinOf turns a rule back into the pin that generates it.
func PinOf(r Rule) Pin {
	return Pin{Target: r.Target, Transform: r.Transform, When: r.When, OutsideReport: r.TargetBasis,
		Attention: r.Attention, Note: r.Note, Disabled: r.Disabled}
}

// ExportPins writes the effective rules of b back into the committed pins
// file, so Rules-tab edits can be reviewed, committed and regenerated.
func ExportPins(name string, b *Bundle) (*Pins, error) {
	p, err := BasePins(name)
	if err != nil {
		return nil, err
	}
	base, err := Load(name)
	if err != nil {
		return nil, err
	}
	fromPins := map[string]bool{} // reviewed rules generated from a report pin
	for _, r := range base.RuleSet.Rules {
		fromPins[r.ID] = true
	}
	var excluded []AdditionalRule // columns deliberately without a rule
	for _, a := range p.Additional {
		if a.Exclude != "" {
			excluded = append(excluded, a)
			continue
		}
		fromPins[a.ID] = false
	}
	p.Additional = excluded
	for _, r := range b.RuleSet.Rules {
		if !fromPins[r.ID] {
			p.Additional = append(p.Additional, AdditionalRule{ID: r.ID, Column: r.Column, Header: r.Header, Pin: PinOf(r)})
			continue
		}
		no := strconv.Itoa(r.ReportNo)
		suffix := strings.TrimPrefix(r.ID, fmt.Sprintf("%s-%02d", p.IDPrefix, r.ReportNo))
		pin, ok := p.Pins[no]
		if !ok {
			return nil, fmt.Errorf("rule %s has no pin #%s", r.ID, no)
		}
		if suffix == "" {
			also := pin.Also
			pin = PinOf(r)
			pin.Also = also
		} else {
			found := false
			for i := range pin.Also {
				if pin.Also[i].Suffix == suffix {
					pin.Also[i].Pin, found = PinOf(r), true
				}
			}
			if !found {
				return nil, fmt.Errorf("rule %s has no pin #%s%s", r.ID, no, suffix)
			}
		}
		p.Pins[no] = pin
	}
	sort.SliceStable(p.Additional, func(i, j int) bool { return colNum(p.Additional[i].Column) < colNum(p.Additional[j].Column) })
	return p, nil
}
