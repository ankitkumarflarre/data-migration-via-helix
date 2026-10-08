// Package quote implements a durable local PersonalHome application workflow.
// Local application data is kept separate from reviewed spreadsheet migrations.
package quote

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed schema.json
var schemaJSON []byte

type Condition struct {
	Key    string `json:"key"`
	Values []any  `json:"values"`
}
type Field struct {
	Key          string     `json:"key"`
	Label        string     `json:"label"`
	Type         string     `json:"type"`
	Required     bool       `json:"required"`
	ReadOnly     bool       `json:"readOnly,omitempty"`
	Default      any        `json:"default,omitempty"`
	Options      []string   `json:"options,omitempty"`
	Min          *float64   `json:"min,omitempty"`
	Max          *float64   `json:"max,omitempty"`
	MaxLength    int        `json:"maxLength,omitempty"`
	ShowWhen     *Condition `json:"showWhen,omitempty"`
	RequiredWhen *Condition `json:"requiredWhen,omitempty"`
}
type Collection struct {
	Key    string     `json:"key"`
	Label  string     `json:"label"`
	Fields []Field    `json:"fields"`
	When   *Condition `json:"when,omitempty"`
}
type Page struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Fields     []Field     `json:"fields"`
	Collection *Collection `json:"collection,omitempty"`
	When       *Condition  `json:"when,omitempty"`
}
type Definition struct {
	Version int    `json:"version"`
	Pages   []Page `json:"pages"`
}

var Schema = func() Definition {
	var d Definition
	if err := json.Unmarshal(schemaJSON, &d); err != nil {
		panic(err)
	}
	return d
}()

type Quote struct {
	ID          string                      `json:"id"`
	Number      string                      `json:"number"`
	Version     int                         `json:"version"`
	Status      string                      `json:"status"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
	CurrentPage string                      `json:"current_page"`
	Completed   []string                    `json:"completed"`
	Values      map[string]any              `json:"values"`
	Collections map[string][]map[string]any `json:"collections"`
}
type Input struct {
	Version int              `json:"version"`
	Page    string           `json:"page"`
	Values  map[string]any   `json:"values"`
	Rows    []map[string]any `json:"rows"`
	Advance bool             `json:"advance"`
}
type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e *ValidationError) Error() string { return "Please correct the highlighted fields." }

var ErrConflict = errors.New("This quote changed in another tab. Reload it before saving again.")
var ErrNotFound = errors.New("Quote not found.")
var ErrTransition = errors.New("Complete the preceding pages before continuing.")
var idPattern = regexp.MustCompile(`^[a-f0-9]{24}$`)
var zipPattern = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
var statePattern = regexp.MustCompile(`^[A-Z]{2}$`)

type Store struct {
	mu  sync.Mutex
	dir string
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}
func matches(c *Condition, v map[string]any) bool {
	if c == nil {
		return true
	}
	for _, value := range c.Values {
		if v[c.Key] == value {
			return true
		}
	}
	return false
}
func ActivePages(q *Quote) []Page {
	out := []Page{}
	for _, p := range Schema.Pages {
		if matches(p.When, q.Values) {
			out = append(out, p)
		}
	}
	return out
}
func index(pages []Page, id string) int {
	for i, p := range pages {
		if p.ID == id {
			return i
		}
	}
	return -1
}
func contains(ss []string, s string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}
	return false
}
func defaults() map[string]any {
	out := map[string]any{}
	for _, p := range Schema.Pages {
		for _, f := range p.Fields {
			if f.Default != nil {
				out[f.Key] = f.Default
			}
		}
	}
	return out
}
func (s *Store) read(id string) (*Quote, error) {
	if !idPattern.MatchString(id) {
		return nil, ErrNotFound
	}
	b, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var q Quote
	err = json.Unmarshal(b, &q)
	return &q, err
}
func (s *Store) write(q *Quote) error {
	b, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".quote-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(s.dir, q.ID+".json"))
}
func (s *Store) Get(id string) (*Quote, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.read(id) }
func (s *Store) List() ([]*Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := []*Quote{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		q, err := s.read(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			return nil, fmt.Errorf("read saved quote: %w", err)
		}
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt.After(out[j].UpdatedAt) })
	return out, nil
}
func (s *Store) Create(values map[string]any) (*Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(b)
	now := time.Now().UTC()
	q := &Quote{ID: id, Number: "Q-" + strings.ToUpper(id[:10]), Version: 1, Status: "draft", CreatedAt: now, UpdatedAt: now, Values: defaults(), Collections: map[string][]map[string]any{}, Completed: []string{}, CurrentPage: "newquote"}
	p := Schema.Pages[0]
	if err := apply(q, p, values, nil); err != nil {
		return nil, err
	}
	if fields := validate(q, p, true); len(fields) > 0 {
		return nil, &ValidationError{fields}
	}
	q.Completed = []string{"newquote"}
	q.CurrentPage = "account"
	if err := s.write(q); err != nil {
		return nil, err
	}
	return q, nil
}
func (s *Store) Save(id string, in Input) (*Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, err := s.read(id)
	if err != nil {
		return nil, err
	}
	if q.Version != in.Version {
		return nil, ErrConflict
	}
	pages := ActivePages(q)
	i := index(pages, in.Page)
	if i < 0 {
		return nil, ErrTransition
	}
	for _, p := range pages[:i] {
		if !contains(q.Completed, p.ID) {
			return nil, ErrTransition
		}
	}
	if err := apply(q, pages[i], in.Values, in.Rows); err != nil {
		return nil, err
	}
	p := pages[i]
	if fields := validate(q, p, in.Advance); len(fields) > 0 {
		return nil, &ValidationError{fields}
	}
	// Any edit invalidates this and later completions and any prior submission.
	q.Completed = []string{}
	for _, prev := range pages[:i] {
		q.Completed = append(q.Completed, prev.ID)
	}
	q.Status = "draft"
	q.CurrentPage = p.ID
	if in.Advance {
		q.Completed = append(q.Completed, p.ID)
		active := ActivePages(q)
		at := index(active, p.ID)
		if at+1 < len(active) {
			q.CurrentPage = active[at+1].ID
		}
	}
	q.Version++
	q.UpdatedAt = time.Now().UTC()
	if err := s.write(q); err != nil {
		return nil, err
	}
	return q, nil
}
func (s *Store) Submit(id string, version int) (*Quote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	q, err := s.read(id)
	if err != nil {
		return nil, err
	}
	if q.Version != version {
		return nil, ErrConflict
	}
	if q.Status == "ready_for_rating" {
		return q, nil
	}
	fields := map[string]string{}
	for _, p := range ActivePages(q) {
		if p.ID == "review" {
			continue
		}
		if !contains(q.Completed, p.ID) {
			return nil, ErrTransition
		}
		for k, v := range validate(q, p, true) {
			fields[k] = v
		}
	}
	if len(fields) > 0 {
		return nil, &ValidationError{fields}
	}
	q.Status = "ready_for_rating"
	q.CurrentPage = "review"
	q.Version++
	q.UpdatedAt = time.Now().UTC()
	if err := s.write(q); err != nil {
		return nil, err
	}
	return q, nil
}
func apply(q *Quote, p Page, values map[string]any, rows []map[string]any) error {
	allowed := map[string]Field{}
	for _, f := range p.Fields {
		allowed[f.Key] = f
	}
	for k, v := range values {
		f, ok := allowed[k]
		if !ok || f.ReadOnly {
			return &ValidationError{map[string]string{k: "Unknown or read-only field."}}
		}
		if v == nil || v == "" {
			delete(q.Values, k)
		} else {
			q.Values[k] = v
		}
	}
	if p.Collection != nil {
		if len(rows) > 100 {
			return &ValidationError{map[string]string{p.Collection.Key: "At most 100 entries are allowed."}}
		}
		keys := map[string]bool{}
		for _, f := range p.Collection.Fields {
			keys[f.Key] = true
		}
		for _, row := range rows {
			for k := range row {
				if !keys[k] {
					return &ValidationError{map[string]string{k: "Unknown collection field."}}
				}
			}
		}
		if matches(p.Collection.When, q.Values) {
			q.Collections[p.Collection.Key] = rows
		} else {
			delete(q.Collections, p.Collection.Key)
		}
	}
	// Form changes cannot leave incompatible coverage selections active.
	if q.Values["DwellingInput.Form"] == "HO6" {
		q.Values["ReplacementCostDwellingInput.Indicator"] = false
		q.Values["DwellingInput.RoofGeometry"] = "NA"
	} else {
		q.Values["UnitsRegularlyRentedToOthersInput.Indicator"] = false
		q.Values["SpecialPersonalPropertyCoverageInput.Indicator"] = false
	}
	for _, f := range p.Fields {
		if !matches(f.ShowWhen, q.Values) {
			delete(q.Values, f.Key)
		}
	}
	if p.ID == "dwellingcoverage" && q.Values["DwellingInput.Form"] == "HO3" {
		if a, ok := q.Values["CoverageADwellingInput.Limit"].(float64); ok {
			for key, factor := range map[string]float64{"CoverageCPersonalPropertyHO3Input.Limit": 0.5, "CoverageDLossOfUseInput.Limit": 0.2} {
				if q.Values[key] == nil {
					q.Values[key] = math.Round(a * factor)
				}
			}
			if q.Values["ReplacementCostDwellingInput.Indicator"] == true && q.Values["ReplacementCostDwellingInput.ReplacementCostValue"] == nil {
				q.Values["ReplacementCostDwellingInput.ReplacementCostValue"] = a
			}
		}
	}
	return nil
}
func validateFields(fields []Field, values, context map[string]any, required bool, prefix string) map[string]string {
	out := map[string]string{}
	for _, f := range fields {
		if f.ReadOnly || !matches(f.ShowWhen, context) {
			continue
		}
		v, exists := values[f.Key]
		key := prefix + f.Key
		if !exists || v == nil || v == "" {
			if required && f.Required && matches(f.RequiredWhen, context) {
				out[key] = "Required"
			}
			continue
		}
		bad := ""
		switch f.Type {
		case "boolean":
			if _, ok := v.(bool); !ok {
				bad = "Select Yes or No"
			}
		case "integer", "number":
			n, ok := v.(float64)
			if !ok || math.IsInf(n, 0) || math.IsNaN(n) {
				bad = "Enter a number"
			} else if f.Type == "integer" && math.Trunc(n) != n {
				bad = "Enter a whole number"
			} else if f.Min != nil && n < *f.Min {
				bad = fmt.Sprintf("Minimum %g", *f.Min)
			} else if f.Max != nil && n > *f.Max {
				bad = fmt.Sprintf("Maximum %g", *f.Max)
			}
		default:
			text, ok := v.(string)
			if !ok {
				bad = "Enter text"
				break
			}
			text = strings.TrimSpace(text)
			if required && f.Required && text == "" && matches(f.RequiredWhen, context) {
				bad = "Required"
			}
			limit := f.MaxLength
			if limit == 0 {
				limit = 2000
			}
			if len([]rune(text)) > limit {
				bad = fmt.Sprintf("Maximum %d characters", limit)
			}
			if len(f.Options) > 0 && !contains(f.Options, text) {
				bad = "Choose one of the listed options"
			}
			if f.Type == "date" {
				if _, err := time.Parse("2006-01-02", text); err != nil {
					bad = "Enter a valid date"
				}
			}
			if f.Type == "email" {
				a, err := mail.ParseAddress(text)
				if err != nil || a.Address != text {
					bad = "Enter a valid email address"
				}
			}
			if f.Type == "zip" && !zipPattern.MatchString(text) {
				bad = "Enter a 5-digit ZIP or ZIP+4"
			}
			if strings.HasSuffix(f.Key, ".State") && !statePattern.MatchString(text) {
				bad = "Use a two-letter state code"
			}
		}
		if bad != "" {
			out[key] = bad
		}
	}
	return out
}
func validate(q *Quote, p Page, required bool) map[string]string {
	out := validateFields(p.Fields, q.Values, q.Values, required, "")
	if c := p.Collection; c != nil && matches(c.When, q.Values) {
		rows := q.Collections[c.Key]
		if required && c.When != nil && len(rows) == 0 {
			out[c.Key] = "Add at least one entry or select No"
		}
		for i, row := range rows {
			prefix := fmt.Sprintf("%s.%d.", c.Key, i)
			for k, v := range validateFields(c.Fields, row, q.Values, required, prefix) {
				out[k] = v
			}
			if c.Key == "priorInsurance" {
				start, _ := row["PriorInsurance.EffectiveDate"].(string)
				end, _ := row["PriorInsurance.ExpirationDate"].(string)
				if start != "" && end != "" && end <= start {
					out[prefix+"PriorInsurance.ExpirationDate"] = "Expiration must be after the effective date"
				}
			}
			if c.Key == "losses" {
				date, _ := row["LossInput.DateOfLoss"].(string)
				effective, _ := q.Values["PolicyInput.EffectiveDate"].(string)
				if date > effective {
					out[prefix+"LossInput.DateOfLoss"] = "Prior loss must be on or before the quote effective date"
				}
			}
			if c.Key == "interests" && required && row["type"] == "Mortgagee" {
				if row["rank"] == nil || row["rank"] == "" {
					out[prefix+"rank"] = "Mortgage rank is required"
				}
			}
		}
	}
	if p.ID == "dwellinginfo" {
		year, _ := q.Values["DwellingInput.YearBuilt"].(float64)
		date, _ := q.Values["PolicyInput.EffectiveDate"].(string)
		effective, err := time.Parse("2006-01-02", date)
		if err == nil && year > float64(effective.Year()+1) {
			out["DwellingInput.YearBuilt"] = "Cannot be later than effective year + 1"
		}
		roof, _ := q.Values["UpdatedServicesInfo.RoofUpdateYear"].(float64)
		if roof != 0 && (roof < year || err == nil && roof > float64(effective.Year())) {
			out["UpdatedServicesInfo.RoofUpdateYear"] = "Must be between year built and effective year"
		}
		if q.Values["DwellingInput.UseType"] == "Rental" && q.Values["DwellingInput.Form"] != "HO6" {
			out["DwellingInput.UseType"] = "Rental usage is available for HO6 only"
		}
		if q.Values["DwellingInput.Construction"] == "Masonry" && q.Values["DwellingInput.RoofGeometry"] != "NA" {
			out["DwellingInput.RoofGeometry"] = "Select NA for masonry construction"
		}
	}
	if p.ID == "dwellingcoverage" && q.Values["DwellingInput.Form"] == "HO3" {
		a, _ := q.Values["CoverageADwellingInput.Limit"].(float64)
		for key, factor := range map[string]float64{"CoverageCPersonalPropertyHO3Input.Limit": 0.5, "CoverageDLossOfUseInput.Limit": 0.2} {
			if limit, ok := q.Values[key].(float64); ok && limit < math.Round(a*factor) {
				out[key] = fmt.Sprintf("Must be at least %g based on dwelling coverage", math.Round(a*factor))
			}
		}
	}
	if p.ID == "underwriting" && required {
		yes := false
		for _, f := range p.Fields {
			if q.Values[f.Key] == true {
				yes = true
			}
		}
		notes, _ := q.Values["Underwriting.Notes"].(string)
		if yes && strings.TrimSpace(notes) == "" {
			out["Underwriting.Notes"] = "Explain the Yes answers for underwriting review"
		}
	}
	if p.ID == "billing" && required && q.Values["Billing.BillClass"] == "Mortgagee Escrow" {
		found := false
		for _, r := range q.Collections["interests"] {
			if r["type"] == "Mortgagee" {
				found = true
			}
		}
		if !found {
			out["Billing.BillClass"] = "Add a mortgagee on Additional interests before selecting escrow"
		}
	}
	return out
}
