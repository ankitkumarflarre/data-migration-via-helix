// Package ledger records every Helix record the migrator created. It is the
// only link between a policy and records whose variant has no reference to it
// (D2), lets re-runs find what they wrote before (D12), and drives cleanup.
// Storage is an append-only JSON-lines file, replayed on open.
package ledger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Entry is one record this tool created.
type Entry struct {
	RecordID     string    `json:"record_id"`
	Variant      string    `json:"variant"`
	Entity       string    `json:"entity"`
	KeyField     string    `json:"key_field,omitempty"`
	KeyValue     string    `json:"key_value,omitempty"`
	PolicyNumber string    `json:"policy_number,omitempty"`
	SheetRow     int       `json:"sheet_row,omitempty"`
	JobID        string    `json:"job_id"`
	RunID        string    `json:"run_id"`
	Generated    []string  `json:"generated,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	Seq          int64     `json:"seq"`
}

type op struct {
	Op    string `json:"op"` // add | remove
	Entry *Entry `json:"entry,omitempty"`
	ID    string `json:"record_id,omitempty"`
}

// Ledger is safe for concurrent use.
type Ledger struct {
	mu    sync.Mutex
	path  string
	file  *os.File
	byID  map[string]*Entry
	byKey map[string]string // entity|field|value → record id (newest)
	byPol map[string]string // policy|variant → record id (newest)
	seq   int64
}

func keyIdx(entity, field, value string) string { return entity + "\x00" + field + "\x00" + value }
func polIdx(policy, variant string) string      { return policy + "\x00" + variant }

// Open loads (or creates) the ledger file.
func Open(path string) (*Ledger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	l := &Ledger{path: path, byID: map[string]*Entry{}, byKey: map[string]string{}, byPol: map[string]string{}}
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 16<<20)
		for sc.Scan() {
			var o op
			if json.Unmarshal(sc.Bytes(), &o) != nil {
				continue
			}
			l.apply(o)
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("ledger %s: %w", path, err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	l.file = f
	return l, nil
}

func (l *Ledger) apply(o op) {
	switch o.Op {
	case "add":
		if e := o.Entry; e != nil {
			l.byID[e.RecordID] = e
			if e.Seq > l.seq {
				l.seq = e.Seq
			}
			if e.KeyField != "" {
				l.byKey[keyIdx(e.Entity, e.KeyField, e.KeyValue)] = e.RecordID
			}
			if e.PolicyNumber != "" {
				l.byPol[polIdx(e.PolicyNumber, e.Variant)] = e.RecordID
			}
		}
	case "remove":
		if e, ok := l.byID[o.ID]; ok {
			if k := keyIdx(e.Entity, e.KeyField, e.KeyValue); l.byKey[k] == o.ID {
				delete(l.byKey, k)
			}
			if k := polIdx(e.PolicyNumber, e.Variant); l.byPol[k] == o.ID {
				delete(l.byPol, k)
			}
		}
		delete(l.byID, o.ID)
	}
}

func (l *Ledger) write(o op) error {
	b, _ := json.Marshal(o)
	if _, err := l.file.Write(append(b, '\n')); err != nil {
		return err
	}
	return l.file.Sync()
}

// Add records a created record.
func (l *Ledger) Add(e Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq++
	e.Seq = l.seq
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if err := l.write(op{Op: "add", Entry: &e}); err != nil {
		return err
	}
	l.apply(op{Op: "add", Entry: &e})
	return nil
}

// Remove forgets a record (after it was deleted from Helix).
func (l *Ledger) Remove(recordID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.byID[recordID]; !ok {
		return nil
	}
	if err := l.write(op{Op: "remove", ID: recordID}); err != nil {
		return err
	}
	l.apply(op{Op: "remove", ID: recordID})
	return nil
}

// ByKey finds a created record of an entity by business key.
func (l *Ledger) ByKey(entity, field, value string) (Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if id, ok := l.byKey[keyIdx(entity, field, value)]; ok {
		return *l.byID[id], true
	}
	return Entry{}, false
}

// ByPolicy finds the record of a variant created for a policy number.
func (l *Ledger) ByPolicy(policyNumber, variant string) (Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if id, ok := l.byPol[polIdx(policyNumber, variant)]; ok {
		return *l.byID[id], true
	}
	return Entry{}, false
}

// All returns entries newest first (children before the parents they reference).
func (l *Ledger) All() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, 0, len(l.byID))
	for _, e := range l.byID {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seq > out[j].Seq })
	return out
}

// Close closes the file.
func (l *Ledger) Close() error { return l.file.Close() }
