package billing

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed codes.json
var codesJSON []byte

// CodeLists are the dropdown values. The UI imports the same file.
type CodeLists struct {
	PaymentMethodFilters []string          `json:"paymentMethodFilters"`
	PaymentMethods       []string          `json:"paymentMethods"`
	PaymentMethodCodes   map[string]string `json:"paymentMethodCodes"`
	ReverseReasons       []string          `json:"reverseReasons"`
	SuspenseActions      []string          `json:"suspenseActions"`
	DisbursementTypes    []string          `json:"disbursementTypes"`
	DisbursementReasons  []string          `json:"disbursementReasons"`
	DisbursementGroups   []string          `json:"disbursementGroups"`
	HoldTypes            []string          `json:"holdTypes"`
	HoldReasons          []string          `json:"holdReasons"`
	HoldResumeOptions    []string          `json:"holdResumeOptions"`
	AccountPlans         []string          `json:"accountPlans"`
	PaymentPlans         []string          `json:"paymentPlans"`
	AccountClasses       []string          `json:"accountClasses"`
	TaskTypes            []string          `json:"taskTypes"`
	TaskQueues           []string          `json:"taskQueues"`
	PartySearchTargets   []string          `json:"partySearchTargets"`
	PartySearchBy        []string          `json:"partySearchBy"`
	MembershipTypes      []string          `json:"membershipTypes"`
	OtherNameTypes       []string          `json:"otherNameTypes"`
	ErrorTypes           []string          `json:"errorTypes"`
	BatchCurrencies      []string          `json:"batchCurrencies"`
	ProcessingOrgUnits   []string          `json:"processingOrgUnits"`
	PIFReasons           []string          `json:"pifReasons"`
	Reports              []string          `json:"reports"`
	States               []string          `json:"states"`
	TestCards            []struct {
		Brand  string `json:"brand"`
		Number string `json:"number"`
	} `json:"testCards"`
	CardFeeBasisPoints int64 `json:"cardFeeBasisPoints"`
	Assumed            struct {
		PhoneTypes      []string `json:"phoneTypes"`
		EmailTypes      []string `json:"emailTypes"`
		AddressKinds    []string `json:"addressKinds"`
		LicenseTypes    []string `json:"licenseTypes"`
		LicenseStatuses []string `json:"licenseStatuses"`
		NamePrefixes    []string `json:"namePrefixes"`
		NameSuffixes    []string `json:"nameSuffixes"`
		ContactMethods  []string `json:"contactMethods"`
		Languages       []string `json:"languages"`
		Genders         []string `json:"genders"`
		EthnicOrigins   []string `json:"ethnicOrigins"`
		MaritalStatuses []string `json:"maritalStatuses"`
		OtherIDs        []string `json:"otherIds"`
		Professions     []string `json:"professions"`
		TaskAssignees   []string `json:"taskAssignees"`
	} `json:"assumed"`
}

var Codes = func() CodeLists {
	var c CodeLists
	if err := json.Unmarshal(codesJSON, &c); err != nil {
		panic(err)
	}
	return c
}()

func oneOf(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// ValidationError carries per-field messages (422).
type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e *ValidationError) Error() string { return "Please correct the highlighted fields." }

// RuleError is a business rule that blocks the action in the current state (409).
type RuleError struct{ Message string }

func (e *RuleError) Error() string { return e.Message }

var ErrNotFound = errors.New("Billing record not found.")
var ErrConflict = errors.New("This record changed in another tab or session. Reload it before trying again.")

func rule(format string, a ...any) error { return &RuleError{fmt.Sprintf(format, a...)} }

// fieldErrors collects validation messages.
type fieldErrors map[string]string

func (f fieldErrors) add(k, msg string) {
	if _, ok := f[k]; !ok {
		f[k] = msg
	}
}
func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{map[string]string(f)}
}

// Store keeps the billing state in memory and in one JSON file, written
// atomically after every change. Changes run on a copy so a failed rule or
// write leaves the state untouched.
type Store struct {
	mu   sync.Mutex
	path string
	st   *State
	now  func() time.Time
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "billing.json"), now: time.Now}
	b, err := os.ReadFile(s.path)
	switch {
	case os.IsNotExist(err):
		s.st = emptyState(time.Now().Format(dateLayout))
	case err != nil:
		return nil, err
	default:
		var st State
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, fmt.Errorf("read %s: %w", s.path, err)
		}
		st.init()
		s.st = &st
	}
	return s, nil
}

func emptyState(date string) *State {
	st := &State{SystemDate: date, Version: 1, Settings: Settings{AutoAuthorizeLimit: 100000, OfferedPlans: []string{"Premium Finance Annual"}, DisbursementService: ServiceState{Status: "Stopped"}}}
	st.init()
	return st
}

func (st *State) init() {
	if st.Seq == nil {
		st.Seq = map[string]int{}
	}
	if st.Accounts == nil {
		st.Accounts = map[string]*Account{}
	}
	if st.Terms == nil {
		st.Terms = map[string]*Term{}
	}
	if st.Parties == nil {
		st.Parties = map[string]*Party{}
	}
	if st.Payments == nil {
		st.Payments = map[string]*Payment{}
	}
	if st.Suspense == nil {
		st.Suspense = map[string]*SuspenseItem{}
	}
	if st.Disbursements == nil {
		st.Disbursements = map[string]*Disbursement{}
	}
	if st.Holds == nil {
		st.Holds = map[string]*Hold{}
	}
	if st.HoldEvents == nil {
		st.HoldEvents = map[string]*HoldEvent{}
	}
	if st.Tasks == nil {
		st.Tasks = map[string]*Task{}
	}
	if st.Batches == nil {
		st.Batches = map[string]*Batch{}
	}
	if st.Errors == nil {
		st.Errors = map[string]*ErrorItem{}
	}
	if st.Invoices == nil {
		st.Invoices = map[string]*Invoice{}
	}
	if st.Statements == nil {
		st.Statements = map[string]*Statement{}
	}
}

// Empty reports whether nothing but the system date has been stored.
func (st *State) Empty() bool {
	return len(st.Accounts) == 0 && len(st.Parties) == 0 && len(st.Payments) == 0
}

func (s *Store) write(st *State) error {
	b, err := json.MarshalIndent(st, "", " ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".billing-*")
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
	return os.Rename(name, s.path)
}

func clone(st *State) *State {
	b, err := json.Marshal(st)
	if err != nil {
		panic(err)
	}
	var out State
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	out.init()
	return &out
}

// tx is one change in progress.
type tx struct {
	*State
	clock string // wall-clock time for activity timestamps
	as    string // the seed records work as the testers in the screenshots
}

func (t *tx) user() string {
	if t.as != "" {
		return t.as
	}
	return CurrentUser
}

func (s *Store) update(fn func(t *tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.st)
	t := &tx{State: next, clock: s.now().Format("15:04")}
	if err := fn(t); err != nil {
		return err
	}
	if err := s.write(next); err != nil {
		return fmt.Errorf("save billing: %w", err)
	}
	s.st = next
	return nil
}

func (s *Store) read(fn func(st *State)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.st)
}

func (t *tx) next(prefix string) string {
	t.Seq[prefix]++
	return fmt.Sprintf("%s%d", prefix, 1000+t.Seq[prefix])
}

func (t *tx) stamp() string { return t.SystemDate + "T" + t.clock }

func (t *tx) activity(acct, typ, by string, amount *int64, text string) *Activity {
	if by == CurrentUser {
		by = t.user()
	}
	a := &Activity{ID: t.next("ACT"), AccountNumber: acct, At: t.stamp(), Type: typ, CreatedBy: by, Amount: amount, Text: text}
	t.Activities = append(t.Activities, a)
	return a
}

func cents(v int64) *int64 { return &v }

// account loads an account and checks the caller's version.
func (t *tx) account(number string, version int) (*Account, error) {
	a := t.Accounts[number]
	if a == nil {
		return nil, ErrNotFound
	}
	if a.Version != version {
		return nil, ErrConflict
	}
	a.Version++
	return a, nil
}

func (t *tx) term(acct *Account, id string) (*Term, error) {
	tm := t.Terms[id]
	if tm == nil || tm.AccountNumber != acct.Number {
		return nil, ErrNotFound
	}
	return tm, nil
}

func (st *State) termsOf(acct string) []*Term {
	out := []*Term{}
	for _, tm := range st.Terms {
		if tm.AccountNumber == acct {
			out = append(out, tm)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Effective != out[j].Effective {
			return out[i].Effective < out[j].Effective
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (st *State) partyName(id string) string {
	p := st.Parties[id]
	if p == nil {
		return ""
	}
	return p.DisplayName(st.SystemDate)
}

// ---- dates and money ----

const dateLayout = "2006-01-02"

var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func validDate(s string) bool {
	if !datePattern.MatchString(s) {
		return false
	}
	_, err := time.Parse(dateLayout, s)
	return err == nil
}

func mustDate(s string) time.Time {
	d, err := time.Parse(dateLayout, s)
	if err != nil {
		panic(fmt.Sprintf("bad date %q", s))
	}
	return d
}

func addDays(s string, n int) string { return mustDate(s).AddDate(0, 0, n).Format(dateLayout) }

func addMonths(s string, n int) string {
	d := mustDate(s)
	y, m, day := d.Date()
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC).Format(dateLayout)
}

func daysBetween(from, to string) int { return int(mustDate(to).Sub(mustDate(from)).Hours() / 24) }

// US formats a date as M/D/YYYY.
func US(s string) string {
	if !validDate(s) {
		return s
	}
	d := mustDate(s)
	return fmt.Sprintf("%d/%d/%d", int(d.Month()), d.Day(), d.Year())
}

// US2 formats a date as MM/DD/YYYY, the receipt-date style in activity text.
func US2(s string) string {
	if !validDate(s) {
		return s
	}
	return mustDate(s).Format("01/02/2006")
}

// USD formats cents as $1,234.56.
func USD(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	whole, frac := c/100, c%100
	s := fmt.Sprint(whole)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	out := fmt.Sprintf("$%s.%02d", s, frac)
	if neg {
		return "-" + out
	}
	return out
}

// Plain formats cents as 1,234.56 for the report grids.
func Plain(c int64) string { return strings.TrimPrefix(USD(c), "$") }

func percentOf(amount, basisPoints int64) int64 { return (amount*basisPoints + 5000) / 10000 }

func termLabel(tm *Term) string {
	return fmt.Sprintf("%s %s - %s", tm.PolicyNumber, US(tm.Effective), US(tm.Expiration))
}
