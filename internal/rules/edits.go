package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// Change kinds.
const (
	ChangeRule     = "rule"
	ChangeTemplate = "template"
)

// Change is one saved edit made in the Rules tab. The latest change of a key
// wins. A nil Rule (or Source) returns the key to its reviewed default; for a
// rule added in the Rules tab it removes the rule.
type Change struct {
	Seq    int          `json:"seq"`
	At     time.Time    `json:"at"`
	Kind   string       `json:"kind"` // rule | template
	Key    string       `json:"key"`  // rule id, or "variant|field"
	Reason string       `json:"reason"`
	Rule   *Rule        `json:"rule,omitempty"`
	Source *FieldSource `json:"source,omitempty"`
}

type editsFile struct {
	RuleSet string   `json:"rule_set"`
	Changes []Change `json:"changes"`
}

// Store keeps the Rules-tab edits of each rule set as a file in Dir and
// applies them on top of the embedded, reviewed rule set.
type Store struct {
	Dir string
	mu  sync.Mutex
	now func() time.Time
}

// OpenStore creates the directory if needed.
func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{Dir: dir, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Store) path(name string) string { return filepath.Join(s.Dir, name+".edits.json") }

// Changes returns every saved change of a rule set, oldest first.
func (s *Store) Changes(name string) ([]Change, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read(name)
}

func (s *Store) read(name string) ([]Change, error) {
	b, err := os.ReadFile(s.path(name))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f editsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("edits %s: %w", name, err)
	}
	return f.Changes, nil
}

// Base loads the reviewed rule set without edits.
func (s *Store) Base(name string) (*Bundle, error) { return Load(name) }

// Load returns the rule set with every saved change applied.
func (s *Store) Load(name string) (*Bundle, error) {
	base, err := Load(name)
	if err != nil {
		return nil, err
	}
	changes, err := s.Changes(name)
	if err != nil {
		return nil, err
	}
	return Apply(base, changes)
}

// Append saves a change after checking that the result is still a valid
// rule set. Seq and At are assigned here.
func (s *Store) Append(name string, c Change) (Change, error) {
	if c.Kind != ChangeRule && c.Kind != ChangeTemplate {
		return c, fmt.Errorf("unknown change kind %q", c.Kind)
	}
	if c.Key == "" || c.Reason == "" {
		return c, fmt.Errorf("a change needs a key and a reason")
	}
	base, err := Load(name)
	if err != nil {
		return c, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changes, err := s.read(name)
	if err != nil {
		return c, err
	}
	c.Seq, c.At = 1, s.now()
	if n := len(changes); n > 0 {
		c.Seq = changes[n-1].Seq + 1
	}
	next := append(append([]Change{}, changes...), c)
	if _, err := Apply(base, next); err != nil {
		return c, err
	}
	b, err := json.MarshalIndent(editsFile{RuleSet: name, Changes: next}, "", "  ")
	if err != nil {
		return c, err
	}
	tmp := s.path(name) + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return c, err
	}
	return c, os.Rename(tmp, s.path(name))
}

// Latest returns the newest change of each key.
func Latest(changes []Change) map[string]Change {
	out := map[string]Change{}
	for _, c := range changes {
		out[c.Kind+":"+c.Key] = c
	}
	return out
}

// Apply returns a copy of base with the changes applied. Its SHA256 covers the
// effective rules, so plans made with edited rules hash differently.
func Apply(base *Bundle, changes []Change) (*Bundle, error) {
	if len(changes) == 0 {
		return base, nil
	}
	b := &Bundle{}
	if err := deepCopy(&b.RuleSet, base.RuleSet); err != nil {
		return nil, err
	}
	if err := deepCopy(&b.Templates, base.Templates); err != nil {
		return nil, err
	}
	baseRules := map[string]Rule{}
	for _, r := range base.RuleSet.Rules {
		baseRules[r.ID] = r
	}
	latest := Latest(changes)
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		c := latest[k]
		switch c.Kind {
		case ChangeRule:
			idx := -1
			for i, r := range b.RuleSet.Rules {
				if r.ID == c.Key {
					idx = i
				}
			}
			switch {
			case c.Rule == nil && idx >= 0:
				if br, ok := baseRules[c.Key]; ok {
					b.RuleSet.Rules[idx] = br
				} else {
					b.RuleSet.Rules = append(b.RuleSet.Rules[:idx], b.RuleSet.Rules[idx+1:]...)
				}
			case c.Rule != nil:
				r := *c.Rule
				r.ID = c.Key
				if idx >= 0 {
					b.RuleSet.Rules[idx] = r
				} else {
					b.RuleSet.Rules = append(b.RuleSet.Rules, r)
				}
			}
		case ChangeTemplate:
			variant, field, ok := splitKey(c.Key)
			if !ok {
				return nil, fmt.Errorf("template change key %q is not variant|field", c.Key)
			}
			ti := -1
			for i, t := range b.Templates.Records {
				if t.ID() == variant { // the key is "record id|field"
					ti = i
				}
			}
			if ti < 0 {
				return nil, fmt.Errorf("template change: no template record %s", variant)
			}
			var src FieldSource
			if c.Source != nil {
				src = *c.Source
			} else if bs, ok := base.Templates.Records[ti].Fields[field]; ok {
				src = bs
			} else {
				delete(b.Templates.Records[ti].Fields, field)
				continue
			}
			if b.Templates.Records[ti].Fields == nil {
				b.Templates.Records[ti].Fields = map[string]FieldSource{}
			}
			b.Templates.Records[ti].Fields[field] = src
		}
	}
	// Reviewed rules keep their order; added rules follow in column order.
	sort.SliceStable(b.RuleSet.Rules, func(i, j int) bool {
		_, bi := baseRules[b.RuleSet.Rules[i].ID]
		_, bj := baseRules[b.RuleSet.Rules[j].ID]
		if bi != bj {
			return bi
		}
		if bi {
			return false
		}
		return colNum(b.RuleSet.Rules[i].Column) < colNum(b.RuleSet.Rules[j].Column)
	})
	// The hash covers what the rules are, not how they got there: edits that
	// end up back at the reviewed rules give the reviewed hash.
	rb, _ := json.Marshal(b.RuleSet)
	tb, _ := json.Marshal(b.Templates)
	br, _ := json.Marshal(base.RuleSet)
	bt, _ := json.Marshal(base.Templates)
	if string(rb) == string(br) && string(tb) == string(bt) {
		b.SHA256 = base.SHA256
	} else {
		h := sha256.New()
		h.Write([]byte(base.SHA256))
		h.Write(rb)
		h.Write(tb)
		b.SHA256 = hex.EncodeToString(h.Sum(nil))
	}
	return b, b.validate()
}

func splitKey(k string) (string, string, bool) {
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '|' {
			return k[:i], k[i+1:], i > 0 && i < len(k)-1
		}
	}
	return "", "", false
}

func colNum(c string) int {
	n, _ := excelize.ColumnNameToNumber(c)
	return n
}

func deepCopy(dst, src any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
