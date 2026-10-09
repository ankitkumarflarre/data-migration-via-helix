// Package plan turns (workbook, rule set, templates, overrides) into the exact
// records the migrator will write. Build is deterministic: the same inputs give
// the same plan and the same hash.
package plan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ankitkumarflarre/datamigration/internal/excel"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// RuleOverride changes one rule. A nil Target or Map keeps the rule's own.
type RuleOverride struct {
	Exclude bool              `json:"exclude,omitempty"`
	Target  *rules.Target     `json:"target,omitempty"`
	Map     map[string]string `json:"map,omitempty"`
}

// Overrides are the user's changes, keyed by rule id and by "variant|field".
type Overrides struct {
	Rules     map[string]RuleOverride `json:"rules,omitempty"`
	Templates map[string]string       `json:"templates,omitempty"`
}

// Input is everything a plan depends on.
type Input struct {
	Bundle    *rules.Bundle
	Sheet     *excel.Sheet
	FileSHA   string
	Overrides Overrides
	Schema    *schema.Schema
}

// Field sources.
const (
	SrcSheet     = "sheet"
	SrcTemplate  = "template"
	SrcGenerated = "generated"
	SrcReference = "reference"
	SrcOverride  = "override"
)

// Sample is one before/after example of a mapped value.
type Sample struct {
	Row    int    `json:"row"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// FieldImpact is one impacted field of a variant.
type FieldImpact struct {
	Field        string              `json:"field"`
	Type         transform.FieldType `json:"type"`
	Required     bool                `json:"required"`
	Source       string              `json:"source"`
	RuleID       string              `json:"rule_id,omitempty"`
	Column       string              `json:"column,omitempty"`
	Header       string              `json:"header,omitempty"`
	Transform    *rules.Transform    `json:"transform,omitempty"`
	Template     string              `json:"template,omitempty"` // human description of a template source
	RefVariant   string              `json:"ref_variant,omitempty"`
	Values       int                 `json:"values"`
	Errors       int                 `json:"errors"`
	Samples      []Sample            `json:"samples,omitempty"`
	Attention    string              `json:"attention,omitempty"`
	Condition    string              `json:"condition,omitempty"` // the rule only applies to some rows
	Overridden   bool                `json:"overridden,omitempty"`
	Alternatives []rules.Target      `json:"alternatives,omitempty"`
}

// VariantImpact groups the impacted fields of one variant.
type VariantImpact struct {
	Variant  string        `json:"variant"`
	Entity   string        `json:"entity"`
	Scope    string        `json:"scope"`
	Key      string        `json:"key,omitempty"`
	Records  int           `json:"records"`
	Fields   []FieldImpact `json:"fields"`
	Template bool          `json:"template"`
}

// EffectiveRule is a rule after overrides.
type EffectiveRule struct {
	rules.Rule
	Excluded   bool            `json:"excluded"`
	Overridden bool            `json:"overridden"`
	Effective  rules.Target    `json:"effective_target"`
	EffMap     rules.Transform `json:"effective_transform"`
}

// Issue is a validation finding. Row 0 means the whole job.
type Issue struct {
	Row      int    `json:"row"`
	Column   string `json:"column,omitempty"`
	RuleID   string `json:"rule_id,omitempty"`
	Target   string `json:"target,omitempty"`
	Severity string `json:"severity"` // blocking | warning
	Message  string `json:"message"`
}

// AttentionItem must be acknowledged before approval (D4).
type AttentionItem struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"` // rule | template | generated | override
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// Record is one record to write.
type Record struct {
	ID          string            `json:"id"` // record id: the variant, or "variant#instance"
	Variant     string            `json:"variant"`
	Entity      string            `json:"entity"`
	Scope       string            `json:"scope"`
	KeyField    string            `json:"key_field,omitempty"`
	KeyValue    string            `json:"key_value,omitempty"`
	Fields      map[string]any    `json:"fields"`
	SheetFields []string          `json:"sheet_fields"`
	Refs        map[string]string `json:"refs,omitempty"` // field → id of the referenced record
	Generated   []string          `json:"generated,omitempty"`
}

// Row is the plan for one sheet row.
type Row struct {
	Row     int      `json:"row"`
	Key     string   `json:"policy_number"`
	Blocked bool     `json:"blocked"`
	Records []Record `json:"records"`
}

// Plan is the full, deterministic result.
type Plan struct {
	Hash          string          `json:"plan_hash"`
	RuleSet       string          `json:"rule_set"`
	RuleSetSHA    string          `json:"rule_set_sha"`
	HelixBundle   string          `json:"helix_bundle"`
	Rules         []EffectiveRule `json:"rules"`
	Impact        []VariantImpact `json:"impact"`
	Attention     []AttentionItem `json:"attention"`
	Issues        []Issue         `json:"-"`
	BlockingCount int             `json:"blocking_count"`
	WarningCount  int             `json:"warning_count"`
	BlockedRows   int             `json:"blocked_rows"`
	JobRecords    []Record        `json:"job_records"`
	Rows          []Row           `json:"-"`
	RowCount      int             `json:"row_count"`
}

type builder struct {
	in        Input
	ctx       context.Context
	plan      *Plan
	variants  map[string]*schema.Variant
	templates map[string]rules.TemplateRecord
	order     []string // included variants, write order
	scope     map[string]string
	refsOf    map[string]map[string]string // variant → field → referenced variant (template + generic)
	impact    map[string]map[string]*FieldImpact
	records   map[string]int
}

// Build makes the plan.
func Build(ctx context.Context, in Input) (*Plan, error) {
	if in.Overrides.Rules == nil {
		in.Overrides.Rules = map[string]RuleOverride{}
	}
	if in.Overrides.Templates == nil {
		in.Overrides.Templates = map[string]string{}
	}
	_, bundleVersion, err := in.Schema.Leaves(ctx)
	if err != nil {
		return nil, err
	}
	b := &builder{in: in, ctx: ctx, variants: map[string]*schema.Variant{}, templates: map[string]rules.TemplateRecord{},
		scope: map[string]string{}, refsOf: map[string]map[string]string{}, impact: map[string]map[string]*FieldImpact{}, records: map[string]int{}}
	b.plan = &Plan{RuleSet: in.Bundle.RuleSet.Name, RuleSetSHA: in.Bundle.SHA256, HelixBundle: bundleVersion, Attention: []AttentionItem{}}
	b.plan.Hash = hashInputs(in, bundleVersion)
	if err := b.effectiveRules(); err != nil {
		return nil, err
	}
	if err := b.includeVariants(); err != nil {
		return nil, err
	}
	b.jobRecords()
	firstRow := map[string]int{}
	for _, r := range in.Sheet.Rows {
		b.row(r)
		rp := &b.plan.Rows[len(b.plan.Rows)-1]
		if prev, dup := firstRow[rp.Key]; dup && rp.Key != "" {
			rp.Blocked = true
			b.issue(Issue{Row: r.Number, Column: in.Bundle.Templates.RowKeyColumn, Severity: "blocking",
				Message: fmt.Sprintf("policy number %s also appears on row %d", rp.Key, prev)})
		} else {
			firstRow[rp.Key] = r.Number
		}
	}
	b.finish()
	return b.plan, nil
}

func hashInputs(in Input, bundle string) string {
	canon, _ := json.Marshal(struct {
		File      string    `json:"file"`
		RuleSet   string    `json:"rule_set"`
		Bundle    string    `json:"bundle"`
		Overrides Overrides `json:"overrides"`
	}{in.FileSHA, in.Bundle.SHA256, bundle, in.Overrides})
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:])
}

func (b *builder) issue(i Issue) {
	b.plan.Issues = append(b.plan.Issues, i)
}

func (b *builder) variant(id string) (*schema.Variant, error) {
	if v, ok := b.variants[id]; ok {
		return v, nil
	}
	v, err := b.in.Schema.Variant(b.ctx, rules.VariantOf(id)) // id may be a record id
	if err != nil {
		return nil, err
	}
	b.variants[id] = v
	return v, nil
}

func (b *builder) effectiveRules() error {
	for _, r := range b.in.Bundle.RuleSet.Rules {
		er := EffectiveRule{Rule: r, Effective: r.Target, EffMap: r.Transform, Excluded: r.Disabled != ""}
		if o, ok := b.in.Overrides.Rules[r.ID]; ok {
			er.Excluded = er.Excluded || o.Exclude
			if o.Target != nil && *o.Target != r.Target {
				er.Effective, er.Overridden = *o.Target, true
			}
			if o.Map != nil {
				er.EffMap = rules.Transform{Case: r.Transform.Case, Map: o.Map}
				er.Overridden = true
			}
			er.Overridden = er.Overridden || o.Exclude
		}
		if !er.Excluded {
			v, err := b.variant(er.Effective.Record())
			if err != nil {
				return err
			}
			if _, ok := v.Field(er.Effective.Field); !ok {
				return fmt.Errorf("rule %s: %s has no field %q", r.ID, er.Effective.Variant, er.Effective.Field)
			}
			if er.Effective.Instance != "" {
				if _, ok := b.templateByID(er.Effective.Record()); !ok {
					return fmt.Errorf("rule %s: %s has no template record", r.ID, er.Effective.Record())
				}
			}
			if h := b.in.Sheet.Headers[r.Column]; !strings.EqualFold(h, r.Header) {
				b.issue(Issue{Column: r.Column, RuleID: r.ID, Severity: "blocking",
					Message: fmt.Sprintf("column %s header is %q; rule expects %q — wrong workbook or layout", r.Column, h, r.Header)})
			}
		}
		b.plan.Rules = append(b.plan.Rules, er)
	}
	return nil
}

// includeVariants decides which variants are written and in which order:
// rule targets plus the anchor, closed over template and required references.
func (b *builder) includeVariants() error {
	tmplOrder := map[string]int{}
	for i, t := range b.in.Bundle.Templates.Records {
		b.templates[t.ID()] = t
		tmplOrder[t.ID()] = i
	}
	want := map[string]bool{}
	for _, t := range b.in.Bundle.Templates.Records {
		if t.Anchor {
			want[t.ID()] = true
		}
	}
	for _, r := range b.plan.Rules {
		if !r.Excluded {
			want[r.Effective.Record()] = true
		}
	}
	entityToTemplate := map[string]string{}
	for _, t := range b.in.Bundle.Templates.Records {
		v, err := b.variant(t.ID())
		if err != nil {
			return err
		}
		// Instances are one of several records; references to them are explicit.
		if _, dup := entityToTemplate[v.Entity]; !dup && t.Instance == "" {
			entityToTemplate[v.Entity] = t.ID()
		}
	}
	for changed := true; changed; {
		changed = false
		ids := sortedKeys(want)
		for _, id := range ids {
			v, err := b.variant(id)
			if err != nil {
				return err
			}
			refs := map[string]string{}
			if t, ok := b.templates[id]; ok {
				for f, src := range t.Fields {
					if src.Ref != "" {
						refs[f] = src.Ref
					}
				}
			}
			for _, f := range v.Fields {
				if f.Required && f.RefEntity != "" && refs[f.Key] == "" {
					if tv, ok := entityToTemplate[f.RefEntity]; ok {
						refs[f.Key] = tv
					} else {
						b.issue(Issue{Target: id + "." + f.Key, Severity: "blocking",
							Message: fmt.Sprintf("%s requires a reference to %s, which no template provides", id, f.RefEntity)})
					}
				}
			}
			b.refsOf[id] = refs
			for _, to := range refs {
				if !want[to] {
					want[to], changed = true, true
				}
			}
		}
	}
	for id := range want {
		b.order = append(b.order, id)
		b.scope[id] = "row"
		if t, ok := b.templates[id]; ok && t.Scope == "job" {
			b.scope[id] = "job"
		}
	}
	sort.Slice(b.order, func(i, j int) bool {
		oi, iok := tmplOrder[b.order[i]]
		oj, jok := tmplOrder[b.order[j]]
		switch {
		case iok && jok:
			return oi < oj
		case iok != jok:
			return iok
		}
		return b.order[i] < b.order[j]
	})
	// Records must come after what they reference.
	b.order = topo(b.order, b.refsOf)
	return nil
}

func topo(order []string, refs map[string]map[string]string) []string {
	pos := map[string]int{}
	for i, v := range order {
		pos[v] = i
	}
	var out []string
	done := map[string]bool{}
	var visit func(string)
	visit = func(v string) {
		if done[v] {
			return
		}
		done[v] = true
		deps := []string{}
		for _, to := range refs[v] {
			deps = append(deps, to)
		}
		sort.Slice(deps, func(i, j int) bool { return pos[deps[i]] < pos[deps[j]] })
		for _, d := range deps {
			visit(d)
		}
		out = append(out, v)
	}
	for _, v := range order {
		visit(v)
	}
	return out
}

func (b *builder) fieldImpact(variant, field string) *FieldImpact {
	m := b.impact[variant]
	if m == nil {
		m = map[string]*FieldImpact{}
		b.impact[variant] = m
	}
	fi := m[field]
	if fi == nil {
		v := b.variants[variant]
		f, _ := v.Field(field)
		fi = &FieldImpact{Field: field, Type: f.Type, Required: f.Required}
		m[field] = fi
	}
	return fi
}

func addSample(fi *FieldImpact, row int, before, after string) {
	if len(fi.Samples) >= 3 {
		return
	}
	for _, s := range fi.Samples {
		if s.Before == before {
			return
		}
	}
	fi.Samples = append(fi.Samples, Sample{Row: row, Before: before, After: after})
}

func (b *builder) jobRecords() {
	for _, id := range b.order {
		if b.scope[id] != "job" {
			continue
		}
		rec := b.newRecord(id)
		b.fillTemplate(&rec, 0, "", nil)
		b.fillRequired(&rec, 0, b.in.Bundle.RuleSet.Name)
		b.setKey(&rec, 0)
		b.checkHelixRules(&rec, 0)
		b.plan.JobRecords = append(b.plan.JobRecords, rec)
		b.records[id]++
	}
}

func (b *builder) newRecord(id string) Record {
	v := b.variants[id]
	rec := Record{ID: id, Variant: rules.VariantOf(id), Entity: v.Entity, Scope: b.scope[id], Fields: map[string]any{}, SheetFields: []string{}, Refs: map[string]string{}}
	if t, ok := b.templates[id]; ok {
		rec.KeyField = t.Key
	}
	return rec
}

func (b *builder) row(r excel.Row) {
	tm := b.in.Bundle.Templates
	key := transform.Normalize(r.Cells[tm.RowKeyColumn])
	before := len(b.plan.Issues)
	recs := map[string]*Record{}
	get := func(id string) *Record {
		if recs[id] == nil {
			rec := b.newRecord(id)
			recs[id] = &rec
		}
		return recs[id]
	}
	setBy := map[string]string{} // variant.field → rule id that set it
	for _, er := range b.plan.Rules {
		if er.Excluded || !er.When.Matches(r.Cells) {
			continue
		}
		raw := r.Cells[er.Column]
		norm := transform.Normalize(raw)
		rid := er.Effective.Record()
		fi := b.fieldImpact(rid, er.Effective.Field)
		if norm == "" {
			continue
		}
		after := transform.Apply(er.EffMap, norm)
		if after == "" {
			// The value map says this value has no equivalent: leave the field empty.
			addSample(fi, r.Number, norm, "(left empty)")
			b.issue(Issue{Row: r.Number, Column: er.Column, RuleID: er.ID, Target: er.Effective.String(), Severity: "warning",
				Message: fmt.Sprintf("%q has no equivalent in %s; the field is left empty", norm, er.Effective.Field)})
			continue
		}
		f, _ := b.variants[rid].Field(er.Effective.Field)
		val, err := transform.Coerce(f.Type, after, b.in.Sheet.Date1904)
		target := er.Effective.String()
		if err != nil {
			fi.Errors++
			addSample(fi, r.Number, norm, "✗ "+err.Error())
			b.issue(Issue{Row: r.Number, Column: er.Column, RuleID: er.ID, Target: target, Severity: "blocking", Message: err.Error()})
			continue
		}
		fi.Values++
		addSample(fi, r.Number, norm, transform.Canonical(val))
		rec := get(rid)
		if prev, ok := rec.Fields[er.Effective.Field]; ok {
			if transform.Canonical(prev) != transform.Canonical(val) {
				b.issue(Issue{Row: r.Number, Column: er.Column, RuleID: er.ID, Target: target, Severity: "blocking",
					Message: fmt.Sprintf("conflict: %s already set to %q by %s; column %s gives %q", target, transform.Canonical(prev), setBy[target], er.Column, transform.Canonical(val))})
			}
			continue
		}
		rec.Fields[er.Effective.Field] = val
		rec.SheetFields = append(rec.SheetFields, er.Effective.Field)
		setBy[target] = er.ID
	}
	// Which records does this row write? Those with sheet values, the anchor,
	// and everything they reference (row scope; job records are written once).
	write := map[string]bool{}
	var need func(string)
	need = func(id string) {
		if write[id] || b.scope[id] == "job" {
			return
		}
		write[id] = true
		for _, to := range b.refsOf[id] {
			need(to)
		}
	}
	for id, rec := range recs {
		if len(rec.SheetFields) > 0 {
			need(id)
		}
	}
	for id, t := range b.templates {
		if t.Anchor && b.scope[id] == "row" {
			need(id)
		}
	}
	rp := Row{Row: r.Number, Key: key}
	for _, id := range b.order {
		if !write[id] {
			continue
		}
		rec := get(id)
		b.fillTemplate(rec, r.Number, key, r.Cells)
		b.fillRequired(rec, r.Number, key)
		b.setKey(rec, r.Number)
		b.checkHelixRules(rec, r.Number)
		sort.Strings(rec.SheetFields)
		rp.Records = append(rp.Records, *rec)
		b.records[id]++
	}
	for _, is := range b.plan.Issues[before:] {
		if is.Severity == "blocking" {
			rp.Blocked = true
		}
	}
	if key == "" {
		rp.Blocked = true
		b.issue(Issue{Row: r.Number, Column: b.in.Bundle.Templates.RowKeyColumn, Severity: "blocking", Message: "policy number is empty"})
	}
	b.plan.Rows = append(b.plan.Rows, rp)
}

// rowDateOf returns the coerced row date column for placeholders.
func (b *builder) rowDateOf(cells map[string]string) string {
	if cells == nil {
		return ""
	}
	v := transform.Normalize(cells[b.in.Bundle.Templates.RowDateColumn])
	if v == "" {
		return ""
	}
	d, err := transform.Coerce(transform.FieldType{Kind: transform.Date}, v, b.in.Sheet.Date1904)
	if err != nil {
		return ""
	}
	return d.(string)
}

func (b *builder) fillTemplate(rec *Record, rowNum int, key string, cells map[string]string) {
	t, ok := b.templates[rec.ID]
	if !ok {
		return
	}
	v := b.variants[rec.ID]
	for _, field := range sortedKeys(t.Fields) {
		src := t.Fields[field]
		if src.Ref != "" {
			rec.Refs[field] = src.Ref
			fi := b.fieldImpact(rec.ID, field)
			fi.Source, fi.RefVariant, fi.Template = SrcReference, src.Ref, "→ "+src.Ref
			fi.Values++
			continue
		}
		if _, set := rec.Fields[field]; set {
			continue // the sheet wins over a template
		}
		f, ok := v.Field(field)
		fi := b.fieldImpact(rec.ID, field)
		if fi.Source == "" {
			fi.Source, fi.Template, fi.Attention = SrcTemplate, describeSource(src), src.Attention
			if src.Generate {
				fi.Source = SrcGenerated
			}
			if src.Col != "" {
				fi.Column, fi.Header = src.Col, b.in.Sheet.Headers[src.Col]
			}
		}
		if !ok {
			b.issue(Issue{Target: rec.ID + "." + field, Severity: "blocking", Message: "template field does not exist on the variant"})
			continue
		}
		var raw string
		overrideKey := rec.ID + "|" + field
		ov, overridden := b.in.Overrides.Templates[overrideKey]
		switch {
		case overridden:
			raw = ov
			fi.Source, fi.Overridden = SrcOverride, true
		case src.Const != nil:
			raw = *src.Const
		case src.Col != "" && src.PlusYears != 0:
			raw = transform.AddYears(transform.Normalize(cells[src.Col]), src.PlusYears, b.in.Sheet.Date1904)
		case src.Col != "":
			raw = transform.MapValue(src.Map, transform.Normalize(cells[src.Col]))
		case src.Format != "":
			raw = format(src.Format, key, cells)
		case src.MinCol != "":
			raw = b.minColumn(src.MinCol, f.Type)
		case src.Generate:
			scope := key
			if rec.Scope == "job" {
				scope = b.in.Bundle.RuleSet.Name
			}
			val := transform.Placeholder(f.Type, field, scope, b.rowDateOf(cells))
			rec.Fields[field] = val
			rec.Generated = append(rec.Generated, field)
			fi.Values++
			addSample(fi, rowNum, "", transform.Canonical(val))
			continue
		}
		if raw == "" {
			continue // optional fields stay empty; fillRequired generates required ones (D6)
		}
		val, err := transform.Coerce(f.Type, raw, b.in.Sheet.Date1904)
		if err != nil {
			fi.Errors++
			b.issue(Issue{Row: rowNum, Column: src.Col, Target: rec.ID + "." + field, Severity: "blocking", Message: "template value: " + err.Error()})
			continue
		}
		rec.Fields[field] = val
		fi.Values++
		addSample(fi, rowNum, strings.TrimSpace(cellOr(cells, src.Col)), transform.Canonical(val))
	}
}

func cellOr(cells map[string]string, col string) string {
	if cells == nil || col == "" {
		return ""
	}
	return transform.Normalize(cells[col])
}

func describeSource(s rules.FieldSource) string {
	switch {
	case s.Const != nil:
		return "constant “" + *s.Const + "”"
	case s.Col != "" && len(s.Map) > 0:
		return "column " + s.Col + " via value map"
	case s.Col != "":
		return "column " + s.Col
	case s.Format != "":
		return "format “" + s.Format + "”"
	case s.MinCol != "":
		return "earliest value of column " + s.MinCol
	case s.Generate:
		return "generated placeholder"
	case s.Ref != "":
		return "→ " + s.Ref
	}
	return ""
}

func format(f, key string, cells map[string]string) string {
	out := strings.ReplaceAll(f, "{PN}", key)
	for {
		i := strings.Index(out, "{")
		j := strings.Index(out, "}")
		if i < 0 || j < i {
			return out
		}
		col := out[i+1 : j]
		out = out[:i] + transform.Normalize(cells[col]) + out[j+1:]
	}
}

func (b *builder) minColumn(col string, t transform.FieldType) string {
	best := ""
	for _, r := range b.in.Sheet.Rows {
		v := transform.Normalize(r.Cells[col])
		if v == "" {
			continue
		}
		c, err := transform.Coerce(t, v, b.in.Sheet.Date1904)
		if err != nil {
			continue
		}
		s := transform.Canonical(c)
		if best == "" || s < best {
			best = s
		}
	}
	return best
}

// fillRequired supplies generated placeholders for required fields still empty (D6),
// and references for required reference fields.
func (b *builder) fillRequired(rec *Record, rowNum int, scope string) {
	v := b.variants[rec.ID]
	for _, f := range v.Fields {
		if !f.Required {
			continue
		}
		if _, ok := rec.Fields[f.Key]; ok {
			continue
		}
		if _, ok := rec.Refs[f.Key]; ok {
			continue
		}
		if to := b.refsOf[rec.ID][f.Key]; to != "" {
			rec.Refs[f.Key] = to
			fi := b.fieldImpact(rec.ID, f.Key)
			fi.Source, fi.RefVariant, fi.Template = SrcReference, to, "→ "+to
			fi.Values++
			continue
		}
		if f.Type.Kind == transform.Reference {
			continue // already reported in includeVariants
		}
		if ov, ok := b.in.Overrides.Templates[rec.ID+"|"+f.Key]; ok {
			fi := b.fieldImpact(rec.ID, f.Key)
			fi.Source, fi.Overridden = SrcOverride, true
			val, err := transform.Coerce(f.Type, ov, b.in.Sheet.Date1904)
			if err != nil {
				fi.Errors++
				b.issue(Issue{Row: rowNum, Target: rec.ID + "." + f.Key, Severity: "blocking", Message: "override value: " + err.Error()})
				continue
			}
			rec.Fields[f.Key] = val
			fi.Values++
			fi.Template = "your value"
			addSample(fi, rowNum, "", transform.Canonical(val))
			continue
		}
		val := transform.Placeholder(f.Type, f.Key, scope, "")
		rec.Fields[f.Key] = val
		rec.Generated = append(rec.Generated, f.Key)
		fi := b.fieldImpact(rec.ID, f.Key)
		if fi.Source == "" || fi.Source == SrcTemplate {
			fi.Source = SrcGenerated
			if fi.Template == "" {
				fi.Template = "generated placeholder (required, no source)"
			}
		}
		fi.Values++
		addSample(fi, rowNum, "", transform.Canonical(val))
	}
	sort.Strings(rec.Generated)
}

func (b *builder) setKey(rec *Record, rowNum int) {
	if rec.KeyField == "" {
		return
	}
	rec.KeyValue = transform.Canonical(rec.Fields[rec.KeyField])
	if rec.KeyValue == "" {
		b.issue(Issue{Row: rowNum, Target: rec.ID + "." + rec.KeyField, Severity: "blocking", Message: "business key is empty"})
	}
}

func (b *builder) finish() {
	p := b.plan
	p.RowCount = len(p.Rows)
	for _, r := range p.Rows {
		if r.Blocked {
			p.BlockedRows++
		}
	}
	for _, is := range p.Issues {
		if is.Severity == "blocking" {
			p.BlockingCount++
		} else {
			p.WarningCount++
		}
	}
	// Rule-sourced field metadata.
	for _, er := range p.Rules {
		if er.Excluded {
			continue
		}
		fi := b.fieldImpact(er.Effective.Record(), er.Effective.Field)
		if fi.Source == "" || fi.Source == SrcTemplate || fi.Source == SrcGenerated {
			fi.Source = SrcSheet
		}
		if fi.RuleID != "" {
			fi.RuleID += ", " + er.ID
			fi.Column += ", " + er.Column
			continue
		}
		tr := er.EffMap
		fi.RuleID, fi.Column, fi.Header, fi.Transform, fi.Overridden = er.ID, er.Column, er.Header, &tr, er.Overridden
		if er.When != nil {
			fi.Condition = "only when " + er.When.String()
		}
		fi.Alternatives = append([]rules.Target{}, er.Alternatives...)
		if er.Overridden {
			fi.Alternatives = append(fi.Alternatives, er.Target)
		}
		if er.Attention != "" && !er.Overridden {
			fi.Attention = er.Attention
		}
	}
	for _, id := range b.order {
		vi := VariantImpact{Variant: id, Entity: b.variants[id].Entity, Scope: b.scope[id], Records: b.records[id]}
		if t, ok := b.templates[id]; ok {
			vi.Key, vi.Template = t.Key, true
		}
		for _, f := range sortedKeys(b.impact[id]) {
			vi.Fields = append(vi.Fields, *b.impact[id][f])
		}
		sort.SliceStable(vi.Fields, func(i, j int) bool {
			return srcRank(vi.Fields[i].Source) < srcRank(vi.Fields[j].Source)
		})
		p.Impact = append(p.Impact, vi)
	}
	// Attention list (D4): rules, template/generated values, overrides.
	for _, er := range p.Rules {
		switch {
		case er.Excluded && er.Disabled != "" && !er.Overridden:
			// Switched off in the Rules tab: a default, not a choice made for this file.
		case er.Excluded:
			p.Attention = append(p.Attention, AttentionItem{ID: "override:" + er.ID, Kind: SrcOverride,
				Title: fmt.Sprintf("%s (%s, column %s) is excluded", er.ID, er.Header, er.Column), Detail: "This column will not be written."})
		case er.Overridden:
			p.Attention = append(p.Attention, AttentionItem{ID: "override:" + er.ID, Kind: SrcOverride,
				Title:  fmt.Sprintf("%s (%s) overridden", er.ID, er.Header),
				Detail: fmt.Sprintf("Default %s → now %s", er.Target, er.Effective)})
		case er.Attention != "":
			p.Attention = append(p.Attention, AttentionItem{ID: "rule:" + er.ID, Kind: "rule",
				Title: fmt.Sprintf("%s: %s → %s", er.ID, er.Header, er.Effective), Detail: er.Attention})
		}
	}
	for _, vi := range p.Impact {
		for _, f := range vi.Fields {
			id := vi.Variant + "|" + f.Field
			switch {
			case f.Source == SrcOverride:
				p.Attention = append(p.Attention, AttentionItem{ID: "override:" + id, Kind: SrcOverride,
					Title: vi.Variant + "." + f.Field + " value overridden", Detail: "Value set to “" + b.in.Overrides.Templates[id] + "”."})
			case f.Attention != "" && f.Source != SrcSheet:
				p.Attention = append(p.Attention, AttentionItem{ID: "template:" + id, Kind: f.Source,
					Title: vi.Variant + "." + f.Field, Detail: f.Attention})
			case f.Source == SrcGenerated:
				p.Attention = append(p.Attention, AttentionItem{ID: "generated:" + id, Kind: SrcGenerated,
					Title: vi.Variant + "." + f.Field, Detail: "Required by Helix but not in the sheet; a generated placeholder is written (D6, S11)."})
			}
		}
	}
}

func srcRank(s string) int {
	return map[string]int{SrcSheet: 0, SrcOverride: 1, SrcTemplate: 2, SrcGenerated: 3, SrcReference: 4}[s]
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (b *builder) templateByID(id string) (rules.TemplateRecord, bool) {
	for _, t := range b.in.Bundle.Templates.Records {
		if t.ID() == id {
			return t, true
		}
	}
	return rules.TemplateRecord{}, false
}

// checkHelixRules reports, before anything is written, the rules Helix would
// refuse the record for: start states and record-level checks (see schema.Check).
func (b *builder) checkHelixRules(rec *Record, rowNum int) {
	for _, msg := range b.variants[rec.ID].Check(rec.Fields) {
		b.issue(Issue{Row: rowNum, Target: rec.ID, Severity: "blocking", Message: "Helix rule: " + msg})
	}
}
