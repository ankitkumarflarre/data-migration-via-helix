package quotesync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ankitkumarflarre/datamigration/internal/excel"
	"github.com/ankitkumarflarre/datamigration/internal/execute"
	"github.com/ankitkumarflarre/datamigration/internal/ledger"
	"github.com/ankitkumarflarre/datamigration/internal/plan"
	"github.com/ankitkumarflarre/datamigration/internal/quote"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
)

// RuleSet is the reviewed rule set quotes are mapped with.
const RuleSet = "manatee_fl_select_ho_12_1_25"

// Sync states.
const (
	Pending = "pending"
	Syncing = "syncing"
	Synced  = "synced"
	Blocked = "blocked"
	Failed  = "failed"
)

// Syncer writes quotes to Helix after each save, one quote at a time per
// quote, coalescing saves made while a sync is running.
type Syncer struct {
	Quotes *quote.Store
	Bundle func() (*rules.Bundle, error) // the rule set with Rules-tab edits
	Schema *schema.Schema
	API    execute.API
	Ledger *ledger.Ledger

	mu      sync.Mutex
	running map[string]bool
	again   map[string]bool
}

// Enqueue starts a sync of the saved quote in the background. It marks q as
// pending so the save reply already says a sync is on its way.
func (s *Syncer) Enqueue(q *quote.Quote) {
	st := &quote.HelixSync{Status: Pending, PolicyNumber: q.PolicyNumber(), QuoteVersion: q.Version, At: time.Now().UTC()}
	if q.Helix != nil {
		st.Records, st.Counts = q.Helix.Records, q.Helix.Counts
	}
	q.Helix = st
	if _, err := s.Quotes.SetHelix(q.ID, st); err != nil {
		log.Printf("quote %s: %v", q.ID, err)
	}
	s.mu.Lock()
	if s.running == nil {
		s.running, s.again = map[string]bool{}, map[string]bool{}
	}
	if s.running[q.ID] {
		s.again[q.ID] = true
		s.mu.Unlock()
		return
	}
	s.running[q.ID] = true
	s.mu.Unlock()
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			s.Sync(ctx, q.ID)
			cancel()
			s.mu.Lock()
			if s.again[q.ID] {
				delete(s.again, q.ID)
				s.mu.Unlock()
				continue
			}
			delete(s.running, q.ID)
			s.mu.Unlock()
			return
		}
	}()
}

// Sync writes the quote's current values to Helix and records the outcome on
// the quote. It returns the recorded state.
func (s *Syncer) Sync(ctx context.Context, id string) *quote.HelixSync {
	q, err := s.Quotes.Get(id)
	if err != nil {
		return nil
	}
	st := &quote.HelixSync{Status: Syncing, PolicyNumber: q.PolicyNumber(), QuoteVersion: q.Version, At: time.Now().UTC()}
	if q.Helix != nil {
		st.Records, st.Counts = q.Helix.Records, q.Helix.Counts
	}
	_, _ = s.Quotes.SetHelix(id, st)
	done := s.run(ctx, q)
	done.PolicyNumber, done.QuoteVersion, done.At = q.PolicyNumber(), q.Version, time.Now().UTC()
	if _, err := s.Quotes.SetHelix(id, done); err != nil {
		log.Printf("quote %s: record sync state: %v", id, err)
	}
	log.Printf("quote %s → Helix policy %s: %s %s", q.Number, done.PolicyNumber, done.Status, done.Message)
	return done
}

// Prepare builds the plan of one quote without writing anything. Columns whose
// value Helix would refuse (a value outside an enum, text in a number field)
// are left out and reported, so one odd input never holds back the rest.
func (s *Syncer) Prepare(ctx context.Context, q *quote.Quote) (*plan.Plan, []string, error) {
	base, err := s.Bundle()
	if err != nil {
		return nil, nil, err
	}
	bundle, notes := Merge(ctx, base, q, s.Schema)
	cells := Cells(q)
	sheet := &excel.Sheet{Headers: map[string]string{}}
	for _, r := range bundle.RuleSet.Rules {
		sheet.Headers[r.Column] = r.Header
	}
	for col, h := range bundle.Templates.Headers {
		sheet.Headers[col] = h
	}
	for col := range sheet.Headers {
		sheet.Columns = append(sheet.Columns, col)
	}
	sort.Strings(sheet.Columns)
	key := bundle.Templates.RowKeyColumn
	var warnings []string
	for attempt := 0; ; attempt++ {
		sheet.Rows = []excel.Row{{Number: 1, Cells: cells}}
		p, err := plan.Build(ctx, plan.Input{Bundle: bundle, Sheet: sheet, FileSHA: fmt.Sprintf("quote:%s:%d", q.ID, q.Version), Schema: s.Schema})
		if err != nil {
			return nil, nil, err
		}
		dropped := false
		for _, is := range p.Issues {
			if is.Row == 0 || is.Severity != "blocking" || is.Column == "" || is.Column == key || cells[is.Column] == "" || attempt >= 20 {
				continue
			}
			warnings = append(warnings, fmt.Sprintf("Not written: %s %q (%s).", label(sheet, is.Column), cells[is.Column], is.Message))
			delete(cells, is.Column)
			dropped = true
		}
		if !dropped {
			for _, is := range p.Issues {
				if is.Row != 0 && is.Severity == "warning" {
					warnings = append(warnings, fmt.Sprintf("%s: %s", label(sheet, is.Column), is.Message))
				}
			}
			return p, append(notes, warnings...), nil
		}
	}
}

func label(sheet *excel.Sheet, col string) string {
	if h := sheet.Headers[col]; h != "" {
		if len(col) <= 3 {
			return h + " (column " + col + ")"
		}
		return h
	}
	return col
}

func (s *Syncer) run(ctx context.Context, q *quote.Quote) *quote.HelixSync {
	st := &quote.HelixSync{}
	if text(q.Values["PolicyInput.EffectiveDate"]) == "" {
		st.Status, st.Message = Blocked, "An effective date is needed before the policy can be written to Helix."
		return st
	}
	p, warnings, err := s.Prepare(ctx, q)
	st.Warnings = warnings
	if err != nil {
		st.Status, st.Message = Failed, err.Error()
		return st
	}
	if len(p.Rows) == 0 || p.Rows[0].Blocked {
		var msgs []string
		for _, is := range p.Issues {
			if is.Severity == "blocking" {
				msgs = append(msgs, strings.TrimSpace(is.Target+" "+is.Message))
			}
		}
		if len(msgs) > 5 {
			msgs = append(msgs[:5], fmt.Sprintf("and %d more", len(msgs)-5))
		}
		st.Status, st.Message = Blocked, "Helix would refuse this policy: "+strings.Join(msgs, "; ")
		return st
	}
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	x := &execute.Executor{API: s.API, Ledger: s.Ledger}
	x.Run(ctx, p, execute.Options{JobID: "quote-" + q.ID, RunID: "quote-" + hex.EncodeToString(b), Workers: 1})
	snap := x.Snapshot()
	st.Counts = map[string]int{}
	add := func(a execute.Action) {
		st.Counts[a.Action]++
		st.Records = append(st.Records, quote.SyncRecord{Variant: a.Variant, Action: a.Action, RecordID: a.RecordID})
	}
	for _, a := range snap.JobActions {
		add(a)
	}
	if snap.State == "aborted" {
		st.Status, st.Message = Failed, snap.Error
		return st
	}
	if len(snap.Results) == 0 {
		st.Status, st.Message = Failed, "the policy row was not written"
		return st
	}
	res := snap.Results[0]
	for _, a := range res.Actions {
		add(a)
	}
	switch res.Status {
	case "written":
		st.Status = Synced
		st.Message = fmt.Sprintf("%d created, %d updated, %d unchanged.", st.Counts[execute.Created], st.Counts[execute.Updated], st.Counts[execute.Unchanged])
	case "blocked":
		st.Status, st.Message = Blocked, res.Message
	default:
		st.Status, st.Message = Failed, res.Message
	}
	return st
}
