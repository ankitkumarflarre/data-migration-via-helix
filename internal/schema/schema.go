// Package schema caches what the migrator needs from the Helix model:
// field types per leaf variant (/describe) and the list of leaf variants (/catalogue).
package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	ID     string  `json:"variant"`
	Entity string  `json:"entity"`
	Fields []Field `json:"fields"`
	byKey  map[string]Field
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
	bundle   string
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
	v := &Variant{ID: id, Entity: d.Entity, byKey: map[string]Field{}}
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
	for _, e := range ents {
		for _, l := range e.Leaves {
			out = append(out, Leaf{Variant: l, Entity: e.Entity, Title: e.Title, Module: e.Module})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Variant < out[j].Variant })
	s.mu.Lock()
	s.leaves, s.bundle = out, bundle
	s.mu.Unlock()
	return out, bundle, nil
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
	for id, d := range f.Descs {
		byEntity[d.Entity] = append(byEntity[d.Entity], id)
	}
	var out []helix.CatalogueEntity
	for e, leaves := range byEntity {
		sort.Strings(leaves)
		out = append(out, helix.CatalogueEntity{Entity: e, Title: e, Leaves: leaves})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Entity < out[j].Entity })
	return f.Bundle, out, nil
}
