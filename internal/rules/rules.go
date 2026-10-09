// Package rules holds the deterministic mapping rules. Rule sets are generated
// by cmd/rulegen from a schema-validation report, reviewed, committed and
// embedded; nothing is parsed from HTML at runtime.
package rules

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed rulesets/*.json
var files embed.FS

// Target is one Helix field: a leaf variant and a field key.
type Target struct {
	Variant string `json:"variant"`
	Field   string `json:"field"`
}

func (t Target) String() string { return t.Variant + "." + t.Field }

// Transform turns a cell into the value given to type coercion.
type Transform struct {
	Case string            `json:"case,omitempty"` // "upper" | "lower"
	Map  map[string]string `json:"map,omitempty"`  // trimmed, case-insensitive source → value
}

// Condition limits a rule to rows whose cell in Column is one of In
// (trimmed, case-insensitive).
type Condition struct {
	Column string   `json:"column"`
	In     []string `json:"in"`
}

// Matches reports whether a row's cells satisfy the condition. A nil
// condition always matches.
func (c *Condition) Matches(cells map[string]string) bool {
	if c == nil {
		return true
	}
	v := strings.TrimSpace(cells[c.Column])
	for _, w := range c.In {
		if strings.EqualFold(v, w) {
			return true
		}
	}
	return false
}

func (c *Condition) String() string {
	return "column " + c.Column + " is " + strings.Join(c.In, " or ")
}

// Rule maps one Excel column (by letter) to one Helix field. One column may
// have several rules with disjoint conditions (e.g. a different target per form).
type Rule struct {
	ID              string     `json:"id"`
	ReportNo        int        `json:"report_no"`
	Column          string     `json:"excel_column"`
	Header          string     `json:"header"`
	ReportStatus    string     `json:"report_status"`
	Confidence      string     `json:"confidence"`
	When            *Condition `json:"when,omitempty"`
	Target          Target     `json:"target"`
	TargetBasis     string     `json:"target_basis,omitempty"` // set when the target is not one of the report's locations
	Alternatives    []Target   `json:"alternatives"`
	ReportLocations []string   `json:"report_locations"`
	Transform       Transform  `json:"transform"`
	Attention       string     `json:"attention,omitempty"`
	Note            string     `json:"note,omitempty"`
}

// Excluded is a confirmed report row that deliberately has no rule.
type Excluded struct {
	ReportNo int    `json:"report_no"`
	Column   string `json:"excel_column"`
	Header   string `json:"header"`
	Reason   string `json:"reason"`
}

// RuleSet is the committed output of rulegen.
type RuleSet struct {
	Name         string `json:"rule_set"`
	Version      string `json:"version"`
	SourceReport struct {
		File   string `json:"file"`
		SHA256 string `json:"sha256"`
	} `json:"source_report"`
	Sheet         string            `json:"sheet"`
	SchemaRenames map[string]string `json:"schema_renames"`
	Rules         []Rule            `json:"rules"`
	Excluded      []Excluded        `json:"excluded,omitempty"`
}

// FieldSource says where a template field's value comes from. Exactly one of
// Const, Col, Format, Ref, Generate or MinCol is set.
type FieldSource struct {
	Const     *string           `json:"const,omitempty"`
	Col       string            `json:"col,omitempty"`    // a sheet column (coerced to the field type)
	Map       map[string]string `json:"map,omitempty"`    // optional value map for Col
	Format    string            `json:"format,omitempty"` // "PH-{PN}", "{C}, FL": {PN} = row key, {X} = column X
	Ref       string            `json:"ref,omitempty"`    // variant of another template record
	Generate  bool              `json:"generate,omitempty"`
	MinCol    string            `json:"min_col,omitempty"` // earliest value of a column across the file (job scope)
	Attention string            `json:"attention,omitempty"`
}

// TemplateRecord describes a record the migrator writes besides the mapped fields.
type TemplateRecord struct {
	Variant string                 `json:"variant"`
	Scope   string                 `json:"scope"` // "job" | "row"
	Key     string                 `json:"key,omitempty"`
	Anchor  bool                   `json:"anchor,omitempty"` // always written (the policy)
	Fields  map[string]FieldSource `json:"fields,omitempty"`
}

// Templates is the companion file of a rule set.
type Templates struct {
	RowKeyColumn  string           `json:"row_key_column"`
	RowDateColumn string           `json:"row_date_column"`
	Records       []TemplateRecord `json:"records"`
}

// Bundle is a loaded rule set with its templates and content hash.
type Bundle struct {
	RuleSet   RuleSet
	Templates Templates
	SHA256    string // over both files' bytes
}

// Names lists embedded rule sets.
func Names() []string {
	entries, _ := files.ReadDir("rulesets")
	var out []string
	for _, e := range entries {
		n := e.Name()
		if strings.HasSuffix(n, ".json") && !strings.HasSuffix(n, ".templates.json") && !strings.HasSuffix(n, ".pins.json") {
			out = append(out, strings.TrimSuffix(n, ".json"))
		}
	}
	sort.Strings(out)
	return out
}

// Load reads an embedded rule set and its templates.
func Load(name string) (*Bundle, error) {
	rb, err := files.ReadFile("rulesets/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("unknown rule set %q", name)
	}
	tb, err := files.ReadFile("rulesets/" + name + ".templates.json")
	if err != nil {
		return nil, fmt.Errorf("rule set %q has no templates file", name)
	}
	b := &Bundle{}
	if err := json.Unmarshal(rb, &b.RuleSet); err != nil {
		return nil, fmt.Errorf("rule set %s: %w", name, err)
	}
	if err := json.Unmarshal(tb, &b.Templates); err != nil {
		return nil, fmt.Errorf("templates %s: %w", name, err)
	}
	h := sha256.New()
	h.Write(rb)
	h.Write(tb)
	b.SHA256 = hex.EncodeToString(h.Sum(nil))
	return b, b.validate()
}

func (b *Bundle) validate() error {
	seen := map[string]bool{}
	for _, r := range b.RuleSet.Rules {
		if r.ID == "" || r.Column == "" || r.Target.Variant == "" || r.Target.Field == "" {
			return fmt.Errorf("rule %q is incomplete", r.ID)
		}
		if seen[r.ID] {
			return fmt.Errorf("duplicate rule id %s", r.ID)
		}
		seen[r.ID] = true
		if r.When != nil && (r.When.Column == "" || len(r.When.In) == 0) {
			return fmt.Errorf("rule %s has an incomplete condition", r.ID)
		}
	}
	variants := map[string]bool{}
	for _, t := range b.Templates.Records {
		variants[t.Variant] = true
	}
	for _, t := range b.Templates.Records {
		for f, src := range t.Fields {
			if src.Ref != "" && !variants[src.Ref] {
				return fmt.Errorf("template %s.%s refers to unknown record %s", t.Variant, f, src.Ref)
			}
		}
	}
	return nil
}
