package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/ankitkumarflarre/datamigration/internal/plan"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// Rule statuses in the Rules tab.
const (
	statusReviewed = "reviewed" // the committed, reviewed default
	statusEdited   = "edited"   // changed in the Rules tab
	statusAdded    = "added"    // added in the Rules tab
)

type ruleView struct {
	rules.Rule
	Status       string               `json:"status"`
	Reviewed     *rules.Rule          `json:"reviewed,omitempty"` // the reviewed default of an edited rule
	Changes      int                  `json:"changes"`
	TargetType   *transform.FieldType `json:"target_type,omitempty"`
	TargetEntity string               `json:"target_entity,omitempty"`
	Required     bool                 `json:"target_required,omitempty"`
	TargetError  string               `json:"target_error,omitempty"`
}

type templateFieldView struct {
	Field    string               `json:"field"`
	Source   rules.FieldSource    `json:"source"`
	Reviewed *rules.FieldSource   `json:"reviewed,omitempty"`
	Status   string               `json:"status"`
	Changes  int                  `json:"changes"`
	Type     *transform.FieldType `json:"type,omitempty"`
	Required bool                 `json:"required,omitempty"`
	Editable bool                 `json:"editable"` // references are structural
}

type templateView struct {
	ID       string              `json:"id"` // record id, used in template change keys
	Variant  string              `json:"variant"`
	Instance string              `json:"instance,omitempty"`
	Entity   string              `json:"entity,omitempty"`
	Scope    string              `json:"scope"`
	Key      string              `json:"key,omitempty"`
	Anchor   bool                `json:"anchor,omitempty"`
	Fields   []templateFieldView `json:"fields"`
}

type ruleSetView struct {
	Name          string            `json:"name"`
	Sheet         string            `json:"sheet"`
	SourceReport  string            `json:"source_report"`
	ReviewedSHA   string            `json:"reviewed_sha"`
	SHA           string            `json:"sha"`
	RowKeyColumn  string            `json:"row_key_column"`
	RowDateColumn string            `json:"row_date_column"`
	Headers       map[string]string `json:"headers"` // every known column → header
	Rules         []ruleView        `json:"rules"`
	Excluded      []rules.Excluded  `json:"excluded"`
	Templates     []templateView    `json:"templates"`
	Changes       []rules.Change    `json:"changes"` // newest first
}

// ruleInput is what the Rules tab may change on a rule.
type ruleInput struct {
	Column    string           `json:"excel_column"`
	Header    string           `json:"header"`
	When      *rules.Condition `json:"when"`
	Target    rules.Target     `json:"target"`
	Transform rules.Transform  `json:"transform"`
	Attention string           `json:"attention"`
	Note      string           `json:"note"`
	Disabled  string           `json:"disabled"`
}

var columnRe = regexp.MustCompile(`^[A-Z]{1,3}$`)

// bundle loads a rule set with its Rules-tab edits.
func (s *Server) bundle(name string) (*rules.Bundle, error) {
	if s.Rules != nil {
		return s.Rules.Load(name)
	}
	return rules.Load(name)
}

func (s *Server) changes(name string) ([]rules.Change, error) {
	if s.Rules == nil {
		return nil, nil
	}
	return s.Rules.Changes(name)
}

func (s *Server) ruleSetView(ctx context.Context, name string) (*ruleSetView, error) {
	base, err := rules.Load(name)
	if err != nil {
		return nil, err
	}
	b, err := s.bundle(name)
	if err != nil {
		return nil, err
	}
	changes, err := s.changes(name)
	if err != nil {
		return nil, err
	}
	count := map[string]int{}
	for _, c := range changes {
		count[c.Kind+":"+c.Key]++
	}
	baseRules := map[string]rules.Rule{}
	for _, r := range base.RuleSet.Rules {
		baseRules[r.ID] = r
	}
	v := &ruleSetView{Name: name, Sheet: b.RuleSet.Sheet, SourceReport: b.RuleSet.SourceReport.File, ReviewedSHA: base.SHA256,
		SHA: b.SHA256, RowKeyColumn: b.Templates.RowKeyColumn, RowDateColumn: b.Templates.RowDateColumn,
		Headers: map[string]string{}, Excluded: b.RuleSet.Excluded, Rules: []ruleView{}, Templates: []templateView{}}
	for c, h := range b.Templates.Headers {
		v.Headers[c] = h
	}
	for _, e := range b.RuleSet.Excluded {
		v.Headers[e.Column] = e.Header
	}
	if v.Excluded == nil {
		v.Excluded = []rules.Excluded{}
	}
	for _, r := range b.RuleSet.Rules {
		v.Headers[r.Column] = r.Header
		rv := ruleView{Rule: r, Status: statusReviewed, Changes: count[rules.ChangeRule+":"+r.ID]}
		if br, ok := baseRules[r.ID]; !ok {
			rv.Status = statusAdded
		} else if !sameJSON(br, r) {
			br := br
			rv.Status, rv.Reviewed = statusEdited, &br
		}
		if sv, err := s.Schema.Variant(ctx, r.Target.Variant); err != nil {
			rv.TargetError = err.Error()
		} else if f, ok := sv.Field(r.Target.Field); !ok {
			rv.TargetError = fmt.Sprintf("%s has no field %q", r.Target.Variant, r.Target.Field)
		} else {
			t := f.Type
			rv.TargetType, rv.TargetEntity, rv.Required = &t, sv.Entity, f.Required
		}
		v.Rules = append(v.Rules, rv)
	}
	for ti, t := range b.Templates.Records {
		tv := templateView{Variant: t.Variant, Instance: t.Instance, ID: t.ID(), Scope: t.Scope, Key: t.Key, Anchor: t.Anchor, Fields: []templateFieldView{}}
		sv, verr := s.Schema.Variant(ctx, t.Variant)
		if verr == nil {
			tv.Entity = sv.Entity
		}
		for _, f := range sortedFieldNames(t.Fields) {
			src := t.Fields[f]
			fv := templateFieldView{Field: f, Source: src, Status: statusReviewed, Editable: src.Ref == "",
				Changes: count[rules.ChangeTemplate+":"+t.ID()+"|"+f]}
			if bs, ok := base.Templates.Records[ti].Fields[f]; !ok {
				fv.Status = statusAdded
			} else if !sameJSON(bs, src) {
				bs := bs
				fv.Status, fv.Reviewed = statusEdited, &bs
			}
			if verr == nil {
				if sf, ok := sv.Field(f); ok {
					ft := sf.Type
					fv.Type, fv.Required = &ft, sf.Required
				}
			}
			tv.Fields = append(tv.Fields, fv)
		}
		v.Templates = append(v.Templates, tv)
	}
	v.Changes = make([]rules.Change, len(changes))
	for i, c := range changes {
		v.Changes[len(changes)-1-i] = c
	}
	return v, nil
}

func sortedFieldNames(m map[string]rules.FieldSource) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sameJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func (s *Server) getRuleSet(w http.ResponseWriter, r *http.Request) {
	v, err := s.ruleSetView(r.Context(), r.PathValue("set"))
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v); err != nil {
		fail(w, http.StatusBadRequest, "request: %v", err)
		return false
	}
	return true
}

func cleanReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if len(reason) < 3 {
		return "", fmt.Errorf("say why you are making this change")
	}
	return reason, nil
}

// checkValue reports whether a mapped or constant value can be written to the field.
func checkValue(t transform.FieldType, v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("a value must not be empty")
	}
	if _, err := transform.Coerce(t, v, false); err != nil {
		return fmt.Errorf("%q: %v", v, err)
	}
	return nil
}

// checkRule validates an edited or added rule against the live Helix schema.
func (s *Server) checkRule(ctx context.Context, in ruleInput) error {
	if !columnRe.MatchString(in.Column) {
		return fmt.Errorf("column must be a letter such as E or AB")
	}
	if strings.TrimSpace(in.Header) == "" {
		return fmt.Errorf("give the column header as written in the sheet")
	}
	if in.Transform.Case != "" && in.Transform.Case != "upper" && in.Transform.Case != "lower" {
		return fmt.Errorf("case must be upper or lower")
	}
	if c := in.When; c != nil {
		if !columnRe.MatchString(c.Column) {
			return fmt.Errorf("condition column must be a letter such as B")
		}
		values := append(append([]string{}, c.In...), c.NotIn...)
		n := 0
		for _, x := range values {
			if strings.TrimSpace(x) != "" {
				n++
			}
		}
		if n == 0 || n != len(values) || (len(c.In) > 0 && len(c.NotIn) > 0) {
			return fmt.Errorf("the condition needs values to match, or values to skip, and no empty values")
		}
	}
	sv, err := s.Schema.Variant(ctx, in.Target.Variant)
	if err != nil {
		return fmt.Errorf("target table: %v", err)
	}
	f, ok := sv.Field(in.Target.Field)
	if !ok {
		return fmt.Errorf("%s has no field %q", in.Target.Variant, in.Target.Field)
	}
	if f.Type.Kind == transform.Reference {
		return fmt.Errorf("%s is a link to another record and cannot take a sheet value", in.Target.Field)
	}
	for from, to := range in.Transform.Map {
		if strings.TrimSpace(from) == "" {
			return fmt.Errorf("a value map entry has no source value")
		}
		if strings.TrimSpace(to) == "" {
			continue // leave the field empty for this value (a warning per row)
		}
		if err := checkValue(f.Type, to); err != nil {
			return fmt.Errorf("value map %q → %v", from, err)
		}
	}
	return nil
}

func (in ruleInput) apply(r *rules.Rule) {
	r.When, r.Target, r.Transform = in.When, in.Target, in.Transform
	r.Attention, r.Note, r.Disabled = strings.TrimSpace(in.Attention), strings.TrimSpace(in.Note), strings.TrimSpace(in.Disabled)
	if len(r.Transform.Map) == 0 {
		r.Transform.Map = nil
	}
}

func (s *Server) saveChange(w http.ResponseWriter, r *http.Request, c rules.Change) {
	if s.Rules == nil {
		fail(w, http.StatusServiceUnavailable, "rule editing is not enabled on this server")
		return
	}
	set := r.PathValue("set")
	if _, err := s.Rules.Append(set, c); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	s.getRuleSet(w, r)
}

// putRule saves a new version of an existing rule.
func (s *Server) putRule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rule   ruleInput `json:"rule"`
		Reason string    `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	reason, err := cleanReason(body.Reason)
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	set, id := r.PathValue("set"), r.PathValue("id")
	b, err := s.bundle(set)
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	var cur *rules.Rule
	for i := range b.RuleSet.Rules {
		if b.RuleSet.Rules[i].ID == id {
			cur = &b.RuleSet.Rules[i]
		}
	}
	if cur == nil {
		fail(w, http.StatusNotFound, "no rule %s", id)
		return
	}
	// Column and header belong to the sheet; only added rules may change them.
	base, _ := rules.Load(set)
	reviewed := map[string]rules.Rule{}
	for _, br := range base.RuleSet.Rules {
		reviewed[br.ID] = br
	}
	br, isReviewed := reviewed[id]
	if isReviewed {
		body.Rule.Column, body.Rule.Header = cur.Column, cur.Header
	}
	if err := s.checkRule(r.Context(), body.Rule); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	next := *cur
	next.Column, next.Header = body.Rule.Column, strings.TrimSpace(body.Rule.Header)
	body.Rule.apply(&next)
	next.TargetBasis = targetBasis(next.Target, cur, br, isReviewed, reason)
	s.saveChange(w, r, rules.Change{Kind: rules.ChangeRule, Key: id, Reason: reason, Rule: &next})
}

// targetBasis keeps the reason a target is not one of the report's locations.
func targetBasis(t rules.Target, cur *rules.Rule, reviewed rules.Rule, isReviewed bool, reason string) string {
	if t == cur.Target {
		return cur.TargetBasis
	}
	if isReviewed {
		if t == reviewed.Target {
			return reviewed.TargetBasis
		}
		for _, a := range reviewed.Alternatives {
			if a == t {
				return "" // a location the report names
			}
		}
	}
	return "Chosen in the Rules tab: " + reason
}

// addRule saves a rule for a column, usually one without a confirmed rule.
func (s *Server) addRule(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rule   ruleInput `json:"rule"`
		Reason string    `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	reason, err := cleanReason(body.Reason)
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	body.Rule.Column = strings.ToUpper(strings.TrimSpace(body.Rule.Column))
	if err := s.checkRule(r.Context(), body.Rule); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	set := r.PathValue("set")
	b, err := s.bundle(set)
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	pins, err := rules.BasePins(set)
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	taken := map[string]bool{}
	for _, x := range b.RuleSet.Rules {
		taken[x.ID] = true
	}
	changes, _ := s.changes(set)
	for _, c := range changes {
		taken[c.Key] = true // ids of removed rules are not reused
	}
	id := pins.IDPrefix + "-" + body.Rule.Column
	for n := 2; taken[id]; n++ {
		id = fmt.Sprintf("%s-%s%d", pins.IDPrefix, body.Rule.Column, n)
	}
	next := rules.Rule{ID: id, Column: body.Rule.Column, Header: strings.TrimSpace(body.Rule.Header),
		ReportStatus: "Added in the Rules tab", Alternatives: []rules.Target{}, ReportLocations: []string{},
		TargetBasis: "Added in the Rules tab: " + reason}
	body.Rule.apply(&next)
	s.saveChange(w, r, rules.Change{Kind: rules.ChangeRule, Key: id, Reason: reason, Rule: &next})
}

// putTemplateField changes where a template field's value comes from.
func (s *Server) putTemplateField(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source rules.FieldSource `json:"source"`
		Reason string            `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	reason, err := cleanReason(body.Reason)
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	set, variant, field := r.PathValue("set"), r.PathValue("variant"), r.PathValue("field")
	b, err := s.bundle(set)
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	var cur *rules.FieldSource
	for _, t := range b.Templates.Records {
		if t.ID() == variant { // a record id: variant or variant#instance
			if src, ok := t.Fields[field]; ok {
				cur = &src
			}
		}
	}
	if cur == nil {
		fail(w, http.StatusNotFound, "no template field %s.%s", variant, field)
		return
	}
	if cur.Ref != "" {
		fail(w, http.StatusBadRequest, "%s.%s links to another record; it cannot be edited here", variant, field)
		return
	}
	src := body.Source
	src.Attention = strings.TrimSpace(src.Attention)
	if err := s.checkSource(r.Context(), variant, field, src); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	s.saveChange(w, r, rules.Change{Kind: rules.ChangeTemplate, Key: variant + "|" + field, Reason: reason, Source: &src})
}

// checkSource allows a constant, a sheet column (with an optional value map)
// or a format; generated, reference and earliest-value sources stay reviewed.
func (s *Server) checkSource(ctx context.Context, variant, field string, src rules.FieldSource) error {
	kinds := 0
	if src.Const != nil {
		kinds++
	}
	if src.Col != "" {
		kinds++
	}
	if src.Format != "" {
		kinds++
	}
	if kinds != 1 || src.Ref != "" || src.Generate || src.MinCol != "" {
		return fmt.Errorf("choose exactly one source: a fixed value, a sheet column or a format")
	}
	sv, err := s.Schema.Variant(ctx, rules.VariantOf(variant))
	if err != nil {
		return err
	}
	f, ok := sv.Field(field)
	if !ok {
		return fmt.Errorf("%s has no field %q", variant, field)
	}
	switch {
	case src.Const != nil:
		return checkValue(f.Type, *src.Const)
	case src.Col != "":
		if !columnRe.MatchString(src.Col) {
			return fmt.Errorf("column must be a letter such as F")
		}
		for from, to := range src.Map {
			if strings.TrimSpace(from) == "" {
				return fmt.Errorf("a value map entry has no source value")
			}
			if err := checkValue(f.Type, to); err != nil {
				return fmt.Errorf("value map %q → %v", from, err)
			}
		}
	case !strings.Contains(src.Format, "{"):
		return fmt.Errorf("a format needs a placeholder such as {PN} (row key) or {C} (column C)")
	}
	return nil
}

// revert brings a rule or template field back to an earlier version, or to
// the reviewed default when seq is 0 (an added rule is then removed).
func (s *Server) revert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Kind   string `json:"kind"`
		Key    string `json:"key"`
		Seq    int    `json:"seq"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	reason, err := cleanReason(body.Reason)
	if err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	c := rules.Change{Kind: body.Kind, Key: body.Key, Reason: reason}
	if body.Seq != 0 {
		changes, err := s.changes(r.PathValue("set"))
		if err != nil {
			fail(w, http.StatusInternalServerError, "%v", err)
			return
		}
		found := false
		for _, old := range changes {
			if old.Seq == body.Seq && old.Kind == body.Kind && old.Key == body.Key {
				c.Rule, c.Source, found = old.Rule, old.Source, true
			}
		}
		if !found {
			fail(w, http.StatusNotFound, "no version %d of %s", body.Seq, body.Key)
			return
		}
	}
	s.saveChange(w, r, c)
}

// exportRuleSet returns the pins or templates file with every edit applied,
// ready to commit and regenerate with `make rules`.
func (s *Server) exportRuleSet(w http.ResponseWriter, r *http.Request) {
	set := r.PathValue("set")
	b, err := s.bundle(set)
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	var out any
	name := set + ".pins.json"
	switch r.URL.Query().Get("file") {
	case "templates":
		out, name = b.Templates, set+".templates.json"
	default:
		p, err := rules.ExportPins(set, b)
		if err != nil {
			fail(w, http.StatusInternalServerError, "%v", err)
			return
		}
		out = p
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	writeJSON(w, http.StatusOK, out)
}

// refreshRules re-plans a job with the latest rules. Overrides of rules that
// no longer exist are dropped.
func (s *Server) refreshRules(w http.ResponseWriter, r *http.Request) {
	j := s.job(w, r)
	if j == nil {
		return
	}
	if s.isRunning(j.ID) {
		fail(w, http.StatusConflict, "the job is writing to Helix; its rules are locked")
		return
	}
	b, err := s.bundle(j.RuleSet)
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	ids := map[string]bool{}
	for _, x := range b.RuleSet.Rules {
		ids[x.ID] = true
	}
	j.mu.Lock()
	ov := plan.Overrides{Templates: j.Overrides.Templates, Rules: map[string]plan.RuleOverride{}}
	for id, o := range j.Overrides.Rules {
		if ids[id] {
			ov.Rules[id] = o
		}
	}
	old := j.bundle
	j.bundle = b
	j.mu.Unlock()
	if err := s.replan(r.Context(), j, ov); err != nil {
		j.mu.Lock()
		j.bundle = old
		j.mu.Unlock()
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, s.view(j))
}
