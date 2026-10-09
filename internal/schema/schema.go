// Package schema caches what the migrator needs from the Helix model:
// field types per leaf variant (/describe) and the list of leaf variants (/catalogue).
package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/ankitkumarflarre/datamigration/internal/helix"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// Source is the part of the Helix client the schema needs.
type Source interface {
	Describe(ctx context.Context, variant string) (helix.Description, error)
	Catalogue(ctx context.Context) (string, []helix.CatalogueEntity, error)
}

// Field is one field of a leaf variant.
type Field struct {
	Key       string              `json:"key"`
	Type      transform.FieldType `json:"type"`
	Required  bool                `json:"required"`
	WriteOnce bool                `json:"write_once"`
	RefEntity string              `json:"ref_entity,omitempty"`
}

// Variant is a leaf variant with its fields, sorted by key.
type Variant struct {
	ID      string               `json:"variant"`
	Entity  string               `json:"entity"`
	Fields  []Field              `json:"fields"`
	Initial map[string][]string  `json:"initial,omitempty"` // status field → states a new record may start in
	Rules   []helix.EnforcedRule `json:"enforced_rules,omitempty"`
	byKey   map[string]Field
}

// Check returns the Helix rules a new record with these field values would
// break: start states of status fields, "required when" and date order. Rules
// of other kinds are left to Helix.
func (v *Variant) Check(fields map[string]any) []string {
	var out []string
	str := func(x any) string {
		if x == nil {
			return ""
		}
		return fmt.Sprint(x)
	}
	for f, allowed := range v.Initial {
		if val, ok := fields[f]; ok && len(allowed) > 0 && !slices.Contains(allowed, str(val)) {
			out = append(out, fmt.Sprintf("%s cannot start at %s (allowed: %s)", f, str(val), strings.Join(allowed, ", ")))
		}
	}
	for _, r := range v.Rules {
		switch r.Rule.Kind {
		case "required_when":
			c := r.Rule.Condition
			w, set := fields[r.Rule.When]
			if c == nil || !set {
				continue
			}
			hit := false
			switch c.Kind {
			case "equals":
				hit = str(w) == str(c.Value)
			case "in":
				hit = slices.Contains(c.Values, str(w))
			case "is_set":
				hit = str(w) != ""
			}
			if _, has := fields[r.Rule.Field]; hit && !has {
				out = append(out, fmt.Sprintf("%s is required when %s is %s (%s)", r.Rule.Field, r.Rule.When, str(w), r.Name))
			}
		case "compare":
			if r.Rule.Right == nil || r.Rule.Right.Kind != "field" {
				continue
			}
			l, r2 := str(fields[r.Rule.Left]), str(fields[r.Rule.Right.Key])
			if l == "" || r2 == "" {
				continue
			}
			ok := true // ISO dates and timestamps compare as text
			switch r.Rule.Op {
			case ">":
				ok = l > r2
			case ">=":
				ok = l >= r2
			case "<":
				ok = l < r2
			case "<=":
				ok = l <= r2
			}
			if !ok {
				out = append(out, fmt.Sprintf("%s (%s) must be %s %s (%s): %s", r.Rule.Left, l, r.Rule.Op, r.Rule.Right.Key, r2, r.Name))
			}
		}
	}
	sort.Strings(out)
	return out
}

// Field returns a field by key.
func (v *Variant) Field(key string) (Field, bool) {
	f, ok := v.byKey[key]
	return f, ok
}

// Leaf is one entry of the variant picker.
type Leaf struct {
	Variant string `json:"variant"`
	Entity  string `json:"entity"`
	Title   string `json:"title"`
	Module  string `json:"module"`
}

// Schema caches describe and catalogue answers.
type Schema struct {
	src      Source
	mu       sync.Mutex
	variants map[string]*Variant
	leaves   []Leaf
	entities map[string]Entity
	bundle   string
}

// Entity is one catalogue entity: its leaf variants and the entities that reference it.
type Entity struct {
	Entity       string   `json:"entity"`
	Title        string   `json:"title"`
	Module       string   `json:"module"`
	Leaves       []string `json:"leaves"`
	ReferencedBy []string `json:"referenced_by"`
	Records      int      `json:"records"`
}

// New wraps a source.
func New(src Source) *Schema { return &Schema{src: src, variants: map[string]*Variant{}} }

// Variant describes one leaf variant (cached).
func (s *Schema) Variant(ctx context.Context, id string) (*Variant, error) {
	s.mu.Lock()
	if v, ok := s.variants[id]; ok {
		s.mu.Unlock()
		return v, nil
	}
	s.mu.Unlock()
	d, err := s.src.Describe(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("describe %s: %w", id, err)
	}
	v := &Variant{ID: id, Entity: d.Entity, byKey: map[string]Field{}, Rules: d.EnforcedRules, Initial: map[string][]string{}}
	for _, t := range d.Transitions {
		v.Initial[t.Field] = t.Initial
	}
	for _, f := range d.Fields {
		fld := Field{Key: f.Key, Type: transform.ParseType(f.Type, f.Enum), Required: f.Required, WriteOnce: f.WriteOnce}
		if f.Reference != nil {
			fld.RefEntity = f.Reference.Entity
		}
		v.Fields = append(v.Fields, fld)
		v.byKey[f.Key] = fld
	}
	sort.Slice(v.Fields, func(i, j int) bool { return v.Fields[i].Key < v.Fields[j].Key })
	s.mu.Lock()
	s.variants[id] = v
	s.mu.Unlock()
	return v, nil
}

// Leaves lists every leaf variant (cached) and the model bundle version.
func (s *Schema) Leaves(ctx context.Context) ([]Leaf, string, error) {
	s.mu.Lock()
	if s.leaves != nil {
		defer s.mu.Unlock()
		return s.leaves, s.bundle, nil
	}
	s.mu.Unlock()
	bundle, ents, err := s.src.Catalogue(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("catalogue: %w", err)
	}
	var out []Leaf
	entities := map[string]Entity{}
	for _, e := range ents {
		entities[e.Entity] = Entity{Entity: e.Entity, Title: e.Title, Module: e.Module, Leaves: e.Leaves, ReferencedBy: e.ReferencedBy, Records: e.Records}
		for _, l := range e.Leaves {
			out = append(out, Leaf{Variant: l, Entity: e.Entity, Title: e.Title, Module: e.Module})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Variant < out[j].Variant })
	s.mu.Lock()
	s.leaves, s.bundle, s.entities = out, bundle, entities
	s.mu.Unlock()
	return out, bundle, nil
}

// Entities lists catalogue entities sorted by name.
func (s *Schema) Entities(ctx context.Context) ([]Entity, error) {
	if _, _, err := s.Leaves(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entity, 0, len(s.entities))
	for _, e := range s.entities {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Entity < out[j].Entity })
	return out, nil
}

// Entity returns one catalogue entity.
func (s *Schema) Entity(ctx context.Context, name string) (Entity, bool, error) {
	if _, _, err := s.Leaves(ctx); err != nil {
		return Entity{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entities[name]
	return e, ok, nil
}

// EntityField is a field of any leaf of an entity.
type EntityField struct {
	Field
	Variants []string `json:"variants"` // leaves that have it
}

// EntityFields is the union of an entity's leaf fields, sorted by key.
func (s *Schema) EntityFields(ctx context.Context, name string) ([]EntityField, error) {
	e, ok, err := s.Entity(ctx, name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no entity %q", name)
	}
	byKey := map[string]*EntityField{}
	for _, leaf := range e.Leaves {
		v, err := s.Variant(ctx, leaf)
		if err != nil {
			return nil, err
		}
		for _, f := range v.Fields {
			if ef, ok := byKey[f.Key]; ok {
				ef.Variants = append(ef.Variants, leaf)
				continue
			}
			byKey[f.Key] = &EntityField{Field: f, Variants: []string{leaf}}
		}
	}
	out := make([]EntityField, 0, len(byKey))
	for _, f := range byKey {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// FileSource serves describe answers from a JSON file {variant: description}
// (the format of the Python export's describe.json). Used by tests and offline runs.
type FileSource struct {
	Bundle string
	Descs  map[string]helix.Description
}

// LoadFileSource reads a describe snapshot.
func LoadFileSource(path, bundle string) (*FileSource, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fs := &FileSource{Bundle: bundle}
	return fs, json.Unmarshal(b, &fs.Descs)
}

func (f *FileSource) Describe(_ context.Context, v string) (helix.Description, error) {
	d, ok := f.Descs[v]
	if !ok {
		return d, &helix.Error{Status: 404, Message: "no variant " + v}
	}
	return d, nil
}

func (f *FileSource) Catalogue(context.Context) (string, []helix.CatalogueEntity, error) {
	byEntity := map[string][]string{}
	refBy := map[string]map[string]bool{}
	for id, d := range f.Descs {
		byEntity[d.Entity] = append(byEntity[d.Entity], id)
		for _, fld := range d.Fields {
			if fld.Reference != nil && fld.Reference.Entity != "" {
				if refBy[fld.Reference.Entity] == nil {
					refBy[fld.Reference.Entity] = map[string]bool{}
				}
				refBy[fld.Reference.Entity][d.Entity] = true
			}
		}
	}
	var out []helix.CatalogueEntity
	for e, leaves := range byEntity {
		sort.Strings(leaves)
		var rb []string
		for x := range refBy[e] {
			rb = append(rb, x)
		}
		sort.Strings(rb)
		out = append(out, helix.CatalogueEntity{Entity: e, Title: e, Leaves: leaves, ReferencedBy: rb})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Entity < out[j].Entity })
	return f.Bundle, out, nil
}
