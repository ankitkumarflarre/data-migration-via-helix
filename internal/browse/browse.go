// Package browse reads Helix data back: everything linked to one policy
// number, or one table filtered by an optional column value.
//
// Policy traversal starts at the policy (matched by policy_number) and at the
// records the migrator's ledger created for it. From those it follows
// reference fields both ways:
//   - outgoing: a record's reference fields → the records they name (one hop;
//     these are often shared, like the product or issuer, so they are shown
//     but not expanded);
//   - incoming: records whose reference fields name a policy-owned record
//     (terms, versions, coverages, …), expanded recursively up to MaxDepth.
package browse

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// Reader is the part of the Helix client browsing needs.
type Reader interface {
	List(ctx context.Context, entity string, q helix.ListQuery) ([]helix.Record, string, error)
	Get(ctx context.Context, variant, id string) (helix.Record, error)
}

// Limits of one policy traversal.
const (
	MaxDepth    = 4
	MaxRecords  = 500
	parallelism = 8
)

// Link says how a record was reached.
type Link struct {
	Kind  string `json:"kind"`            // key | ledger | references | referenced_by
	Text  string `json:"text"`            // human description
	From  string `json:"from,omitempty"`  // record id on the other end
	Field string `json:"field,omitempty"` // reference field involved
}

// Item is one record.
type Item struct {
	ID        string         `json:"id"`
	Entity    string         `json:"entity"`
	Variant   string         `json:"variant"`
	Version   int64          `json:"version"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Fields    map[string]any `json:"fields"`
	Links     []Link         `json:"links,omitempty"`
	Depth     int            `json:"depth"`
	Generated []string       `json:"generated,omitempty"` // from the ledger: placeholder fields (D6)
	order     int
	owned     bool // part of the policy's own sub-tree: expand incoming references
}

// Table groups the records of one variant.
type Table struct {
	Entity  string  `json:"entity"`
	Variant string  `json:"variant"`
	Records []*Item `json:"records"`
}

// PolicyResult is everything found for one policy number.
type PolicyResult struct {
	PolicyNumber string   `json:"policy_number"`
	Tables       []Table  `json:"tables"`
	Records      int      `json:"records"`
	Queries      int      `json:"queries"`
	ElapsedMS    int64    `json:"elapsed_ms"`
	Truncated    bool     `json:"truncated"`
	Notes        []string `json:"notes"`
}

// Browser runs traversals.
type Browser struct {
	Helix  Reader
	Schema *schema.Schema
	Ledger *ledger.Ledger

	mu       sync.Mutex
	incoming map[string][]incomingRef // entity → (referencing entity, field)
}

type incomingRef struct{ entity, field string }

type run struct {
	b       *Browser
	ctx     context.Context
	mu      sync.Mutex
	items   map[string]*Item
	seq     int
	queries int
	notes   []string
	trunc   bool
	sem     chan struct{}
}

func (r *run) note(format string, a ...any) {
	r.mu.Lock()
	r.notes = append(r.notes, fmt.Sprintf(format, a...))
	r.mu.Unlock()
}

// add records an item or a new link to a known one. It reports whether the item is new.
func (r *run) add(rec helix.Record, entity string, depth int, owned bool, link Link) (*Item, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if it, ok := r.items[rec.ID]; ok {
		for _, l := range it.Links {
			if l == link {
				return it, false
			}
		}
		it.Links = append(it.Links, link)
		if owned && !it.owned {
			it.owned = true
			return it, true // now part of the sub-tree: expand it
		}
		return it, false
	}
	if len(r.items) >= MaxRecords {
		r.trunc = true
		return nil, false
	}
	r.seq++
	it := &Item{ID: rec.ID, Entity: entity, Variant: rec.Variant, Version: rec.Version, CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt, Fields: rec.Fields, Links: []Link{link}, Depth: depth, order: r.seq, owned: owned}
	if it.Fields == nil {
		it.Fields = map[string]any{}
	}
	r.items[rec.ID] = it
	return it, true
}

func (r *run) list(entity string, q helix.ListQuery) ([]helix.Record, error) {
	r.sem <- struct{}{}
	defer func() { <-r.sem }()
	r.mu.Lock()
	r.queries++
	r.mu.Unlock()
	recs, _, err := r.b.Helix.List(r.ctx, entity, q)
	return recs, err
}

// Quote formats a where-clause value for a field type: strings JSON-quoted, numbers and booleans bare.
func Quote(t transform.FieldType, v string) string {
	switch t.Kind {
	case transform.Integer, transform.Number, transform.Boolean:
		return v
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// Policy finds every record linked to a policy number.
func (b *Browser) Policy(ctx context.Context, policyNumber string) (*PolicyResult, error) {
	start := time.Now()
	policyNumber = strings.TrimSpace(policyNumber)
	if policyNumber == "" {
		return nil, fmt.Errorf("enter a policy number")
	}
	r := &run{b: b, ctx: ctx, items: map[string]*Item{}, sem: make(chan struct{}, parallelism)}

	var frontier []*Item
	pols, err := r.list("policy", helix.ListQuery{Where: "policy_number=" + Quote(transform.FieldType{Kind: transform.String}, policyNumber), Limit: 20})
	if err != nil {
		return nil, fmt.Errorf("look up policy: %w", err)
	}
	for _, p := range pols {
		if it, isNew := r.add(p, "policy", 0, true, Link{Kind: "key", Text: "policy_number = " + policyNumber}); isNew {
			frontier = append(frontier, it)
		}
	}
	if b.Ledger != nil {
		for _, e := range b.Ledger.ForPolicy(policyNumber) {
			rec, err := b.Helix.Get(ctx, e.Variant, e.RecordID)
			r.queries++
			if helix.IsStatus(err, 404) {
				r.note("ledger lists %s %s, but it no longer exists in Helix", e.Variant, e.RecordID)
				continue
			}
			if err != nil {
				r.note("read %s %s: %v", e.Variant, e.RecordID, err)
				continue
			}
			if rec.Variant == "" {
				rec.Variant = e.Variant
			}
			it, isNew := r.add(rec, e.Entity, 0, true, Link{Kind: "ledger", Text: fmt.Sprintf("created by the migrator for sheet row %d", e.SheetRow)})
			if it != nil {
				it.Generated = e.Generated
			}
			if isNew {
				frontier = append(frontier, it)
			}
		}
	}
	if len(r.items) == 0 {
		return &PolicyResult{PolicyNumber: policyNumber, Tables: []Table{}, Notes: []string{"No policy with this number, and nothing in the migrator's ledger for it."},
			Queries: r.queries, ElapsedMS: time.Since(start).Milliseconds()}, nil
	}

	for depth := 0; len(frontier) > 0 && depth < MaxDepth; depth++ {
		var next []*Item
		var nmu sync.Mutex
		var wg sync.WaitGroup
		for _, it := range frontier {
			wg.Add(1)
			go func(it *Item) {
				defer wg.Done()
				found := r.expand(it, depth+1)
				nmu.Lock()
				next = append(next, found...)
				nmu.Unlock()
			}(it)
		}
		wg.Wait()
		frontier = next
	}
	if len(frontier) > 0 {
		r.note("stopped after %d levels of references", MaxDepth)
	}
	return r.result(policyNumber, start), nil
}

// expand follows one record's references; it returns newly found policy-owned records.
func (r *run) expand(it *Item, depth int) []*Item {
	var out []*Item
	// Outgoing: fields of this record that reference another record.
	if v, err := r.b.Schema.Variant(r.ctx, it.Variant); err == nil {
		for _, f := range v.Fields {
			if f.RefEntity == "" {
				continue
			}
			id, _ := it.Fields[f.Key].(string)
			if id == "" {
				continue
			}
			recs, err := r.list(f.RefEntity, helix.ListQuery{Where: f.RefEntity + "_id=" + Quote(transform.FieldType{}, id), Limit: 2})
			if err != nil {
				r.note("follow %s.%s → %s: %v", it.Entity, f.Key, f.RefEntity, err)
				continue
			}
			for _, rec := range recs {
				r.add(rec, f.RefEntity, depth, false, Link{Kind: "references",
					Text: fmt.Sprintf("referenced by %s.%s", it.Entity, f.Key), From: it.ID, Field: f.Key})
			}
		}
	} else {
		r.note("describe %s: %v", it.Variant, err)
	}
	if !it.owned {
		return nil
	}
	// Incoming: records that reference this one.
	refs, err := r.b.incomingRefs(r.ctx, it.Entity)
	if err != nil {
		r.note("references to %s: %v", it.Entity, err)
		return nil
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, ref := range refs {
		wg.Add(1)
		go func(ref incomingRef) {
			defer wg.Done()
			recs, err := r.list(ref.entity, helix.ListQuery{Where: ref.field + "=" + Quote(transform.FieldType{}, it.ID), Limit: 100})
			if err != nil {
				r.note("find %s by %s: %v", ref.entity, ref.field, err)
				return
			}
			if len(recs) == 100 {
				r.note("%s.%s matched 100+ records; only the first 100 are shown", ref.entity, ref.field)
			}
			for _, rec := range recs {
				if n, isNew := r.add(rec, ref.entity, depth, true, Link{Kind: "referenced_by",
					Text: fmt.Sprintf("its %s references %s %s", ref.field, it.Entity, shortID(it.ID)), From: it.ID, Field: ref.field}); isNew {
					mu.Lock()
					out = append(out, n)
					mu.Unlock()
				}
			}
		}(ref)
	}
	wg.Wait()
	return out
}

// incomingRefs lists (entity, field) pairs whose field references the given entity.
func (b *Browser) incomingRefs(ctx context.Context, entity string) ([]incomingRef, error) {
	b.mu.Lock()
	if b.incoming == nil {
		b.incoming = map[string][]incomingRef{}
	}
	if refs, ok := b.incoming[entity]; ok {
		b.mu.Unlock()
		return refs, nil
	}
	b.mu.Unlock()
	e, ok, err := b.Schema.Entity(ctx, entity)
	if err != nil || !ok {
		return nil, err
	}
	var refs []incomingRef
	for _, x := range e.ReferencedBy {
		fields, err := b.Schema.EntityFields(ctx, x)
		if err != nil {
			continue
		}
		for _, f := range fields {
			if f.RefEntity == entity {
				refs = append(refs, incomingRef{entity: x, field: f.Key})
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].entity+refs[i].field < refs[j].entity+refs[j].field })
	b.mu.Lock()
	b.incoming[entity] = refs
	b.mu.Unlock()
	return refs, nil
}

func (r *run) result(pn string, start time.Time) *PolicyResult {
	byVariant := map[string]*Table{}
	var order []string
	items := make([]*Item, 0, len(r.items))
	for _, it := range r.items {
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].order < items[j].order })
	for _, it := range items {
		t := byVariant[it.Variant]
		if t == nil {
			t = &Table{Entity: it.Entity, Variant: it.Variant}
			byVariant[it.Variant] = t
			order = append(order, it.Variant)
		}
		t.Records = append(t.Records, it)
	}
	res := &PolicyResult{PolicyNumber: pn, Records: len(items), Queries: r.queries, Truncated: r.trunc,
		ElapsedMS: time.Since(start).Milliseconds(), Notes: r.notes}
	if res.Notes == nil {
		res.Notes = []string{}
	}
	if r.trunc {
		res.Notes = append(res.Notes, fmt.Sprintf("stopped at %d records", MaxRecords))
	}
	for _, v := range order {
		res.Tables = append(res.Tables, *byVariant[v])
	}
	return res
}

// TableResult is one page of an entity's records.
type TableResult struct {
	Entity     string   `json:"entity"`
	Where      string   `json:"where,omitempty"`
	Columns    []string `json:"columns"`
	Records    []*Item  `json:"records"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

// Table lists an entity's records, optionally where field = value.
func (b *Browser) Table(ctx context.Context, entity, field, value, after string, limit int) (*TableResult, error) {
	e, ok, err := b.Schema.Entity(ctx, entity)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no table (entity) %q", entity)
	}
	fields, err := b.Schema.EntityFields(ctx, e.Entity)
	if err != nil {
		return nil, err
	}
	q := helix.ListQuery{Limit: limit, After: after}
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	var t transform.FieldType
	if field = strings.TrimSpace(field); field != "" && strings.TrimSpace(value) != "" {
		found := field == e.Entity+"_id"
		for _, f := range fields {
			if f.Key == field {
				t, found = f.Type, true
			}
		}
		if !found {
			return nil, fmt.Errorf("%s has no column %q", e.Entity, field)
		}
		q.Where = field + "=" + Quote(t, strings.TrimSpace(value))
	}
	recs, next, err := b.Helix.List(ctx, e.Entity, q)
	if err != nil {
		return nil, err
	}
	res := &TableResult{Entity: e.Entity, Where: q.Where, NextCursor: next, Records: []*Item{}}
	used := map[string]bool{}
	for _, rec := range recs {
		res.Records = append(res.Records, &Item{ID: rec.ID, Entity: e.Entity, Variant: rec.Variant, Version: rec.Version,
			CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt, Fields: rec.Fields})
		for k, v := range rec.Fields {
			if v != nil && v != "" {
				used[k] = true
			}
		}
	}
	for _, f := range fields { // schema order, only columns with a value on this page
		if used[f.Key] {
			res.Columns = append(res.Columns, f.Key)
		}
	}
	if res.Columns == nil {
		res.Columns = []string{}
	}
	return res, nil
}

// Warm pre-loads the reverse-reference index for entities a policy lookup
// usually visits, so the first lookup is not slowed by schema calls.
func (b *Browser) Warm(ctx context.Context, entities ...string) {
	for _, e := range entities {
		if _, err := b.incomingRefs(ctx, e); err != nil {
			return
		}
	}
}
