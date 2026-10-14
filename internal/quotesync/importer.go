package quotesync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/quote"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// ErrNoPolicy means Helix has no policy with the number.
var ErrNoPolicy = errors.New("no policy with this number in Helix")

// Import opens a Helix policy as a quote. A policy imported before reopens its
// existing quote, so edits made in the quote flow are not overwritten.
func (s *Syncer) Import(ctx context.Context, policyNumber string) (*quote.Quote, bool, error) {
	policyNumber = strings.TrimSpace(policyNumber)
	if q, err := s.Quotes.FindBySource(policyNumber); err != nil || q != nil {
		return q, false, err
	}
	cells, notes, err := s.ReadCells(ctx, policyNumber)
	if err != nil {
		return nil, false, err
	}
	values, colls, more := Values(cells)
	src := quote.Source{Kind: "helix", PolicyNumber: policyNumber, ImportedAt: time.Now().UTC(), Cells: cells, Notes: append(notes, more...)}
	q, err := s.Quotes.Import(src, values, colls)
	return q, true, err
}

// ReadCells reads a policy's records from Helix and turns them back into
// rater-shaped cells by inverting the rules and template columns.
func (s *Syncer) ReadCells(ctx context.Context, pn string) (map[string]string, []string, error) {
	base, err := s.Bundle()
	if err != nil {
		return nil, nil, err
	}
	bundle, _ := Merge(ctx, base, nil, s.Schema)
	templates := map[string]rules.TemplateRecord{}
	var anchor string
	for _, t := range bundle.Templates.Records {
		templates[t.ID()] = t
		if t.Anchor {
			anchor = t.ID()
		}
	}
	keyCol := bundle.Templates.RowKeyColumn
	records := map[string]map[string]any{}
	var notes []string
	find := func(id string) map[string]any {
		if rec, ok := records[id]; ok {
			return rec
		}
		records[id] = nil
		variant := rules.VariantOf(id)
		if e, ok := s.Ledger.ByPolicy(pn, id); ok {
			if got, err := s.API.Get(ctx, e.Variant, e.RecordID); err == nil {
				records[id] = got.Fields
				return got.Fields
			}
		}
		t, ok := templates[id]
		if !ok || t.Key == "" {
			return nil
		}
		value := ""
		if src := t.Fields[t.Key]; src.Format != "" && !strings.Contains(strings.ReplaceAll(src.Format, "{PN}", ""), "{") {
			value = strings.ReplaceAll(src.Format, "{PN}", pn)
		}
		for _, r := range bundle.RuleSet.Rules {
			if r.Column == keyCol && r.Target.Record() == id && r.Target.Field == t.Key {
				value = pn
			}
		}
		v, err := s.Schema.Variant(ctx, variant)
		if value == "" || err != nil {
			return nil
		}
		found, err := s.API.FindByKey(ctx, v.Entity, t.Key, value)
		if err != nil || len(found) != 1 {
			return nil
		}
		got, err := s.API.Get(ctx, variant, found[0].ID)
		if err != nil {
			notes = append(notes, fmt.Sprintf("Could not read %s: %v", id, err))
			return nil
		}
		records[id] = got.Fields
		return got.Fields
	}
	if anchor == "" || find(anchor) == nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrNoPolicy, pn)
	}
	cells := map[string]string{keyCol: pn}
	value := func(id, field string) string {
		rec := find(id)
		if rec == nil || rec[field] == nil {
			return ""
		}
		return transform.Normalize(transform.Canonical(rec[field]))
	}
	// Template columns first: the policy form decides which form-specific rules apply.
	for _, id := range sortedIDs(templates) {
		t := templates[id]
		for _, field := range sortedFieldKeys(t.Fields) {
			src := t.Fields[field]
			if src.Col == "" || src.PlusYears != 0 || cells[src.Col] != "" {
				continue
			}
			v := value(id, field)
			if v == "" {
				continue
			}
			if c, ok := inverse(src.Map, v); ok {
				v = c
			}
			cells[src.Col] = v
		}
	}
	byTarget := map[string][]rules.Rule{}
	for _, r := range bundle.RuleSet.Rules {
		byTarget[r.Target.String()] = append(byTarget[r.Target.String()], r)
	}
	for _, r := range bundle.RuleSet.Rules {
		if r.Disabled != "" || cells[r.Column] != "" {
			continue
		}
		v := value(r.Target.Record(), r.Target.Field)
		if v == "" {
			continue
		}
		cell := v
		if len(r.Transform.Map) > 0 {
			if c, ok := inverse(r.Transform.Map, v); ok {
				cell = c
			} else if claimed(byTarget[r.Target.String()], r, v) {
				continue // another rule on this field produces this value
			} else if transform.MapValue(r.Transform.Map, v) != v {
				continue
			}
		}
		if w := r.When; w != nil && w.Column != r.Column {
			if cells[w.Column] != "" {
				if !w.Matches(cells) {
					continue
				}
			} else if len(w.In) > 0 {
				cells[w.Column] = w.In[0]
			}
		}
		cells[r.Column] = cell
	}
	return cells, notes, nil
}

func claimed(rs []rules.Rule, self rules.Rule, v string) bool {
	for _, r := range rs {
		if r.ID == self.ID || r.Disabled != "" {
			continue
		}
		if _, ok := inverse(r.Transform.Map, v); ok {
			return true
		}
	}
	return false
}

func sortedIDs(m map[string]rules.TemplateRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedFieldKeys(m map[string]rules.FieldSource) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Imported is a policy migrated from a workbook that can be opened as a quote.
type Imported struct {
	PolicyNumber string    `json:"policy_number"`
	Records      int       `json:"records"`
	WrittenAt    time.Time `json:"written_at"`
	QuoteID      string    `json:"quote_id,omitempty"`
	QuoteNumber  string    `json:"quote_number,omitempty"`
}

// Policies lists the policies the migrator wrote from workbooks (the ledger),
// with the quote each one is open in.
func (s *Syncer) Policies() ([]Imported, error) {
	by := map[string]*Imported{}
	for _, e := range s.Ledger.All() {
		if e.PolicyNumber == "" || strings.HasPrefix(e.JobID, "quote-") {
			continue
		}
		p := by[e.PolicyNumber]
		if p == nil {
			p = &Imported{PolicyNumber: e.PolicyNumber}
			by[e.PolicyNumber] = p
		}
		p.Records++
		if e.CreatedAt.After(p.WrittenAt) {
			p.WrittenAt = e.CreatedAt
		}
	}
	quotes, err := s.Quotes.List()
	if err != nil {
		return nil, err
	}
	for _, q := range quotes {
		if q.Source != nil {
			if p := by[q.Source.PolicyNumber]; p != nil {
				p.QuoteID, p.QuoteNumber = q.ID, q.Number
			}
		}
	}
	out := make([]Imported, 0, len(by))
	for _, p := range by {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PolicyNumber < out[j].PolicyNumber })
	return out, nil
}
