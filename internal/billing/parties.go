package billing

import (
	"regexp"
	"sort"
	"strings"
)

type rowField struct {
	key      string
	required bool
	options  func() []string
	pattern  *regexp.Regexp
	message  string
}

// rowSpec describes one effective-dated party section: its fields and which
// of Correct / Replace / Remove it offers.
type rowSpec struct {
	fields  []rowField
	modes   []string
	ownDate bool // the form has its own Effective/Expiration dates (licenses)
}

var (
	zip5      = regexp.MustCompile(`^\d{5}$`)
	plus4     = regexp.MustCompile(`^\d{4}$`)
	phoneUS   = regexp.MustCompile(`^\d{3}-\d{3}-\d{4}$`)
	emailLike = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	ssnLike   = regexp.MustCompile(`^\d{3}-?\d{2}-?\d{4}$`)
)

func list(l []string) func() []string { return func() []string { return l } }

var addressFields = []rowField{
	{key: "country"}, {key: "line1", required: true}, {key: "line2"}, {key: "city", required: true},
	{key: "state", required: true, options: list(Codes.States)}, {key: "zip", required: true, pattern: zip5, message: "Enter a 5-digit ZIP code."},
	{key: "plus4", pattern: plus4, message: "Enter 4 digits."}, {key: "county"}, {key: "comments"}, {key: "attention"},
}

var rowSpecs = map[string]rowSpec{
	"name": {modes: []string{"correct", "replace"}, fields: []rowField{
		{key: "prefix", options: list(Codes.Assumed.NamePrefixes)}, {key: "first"}, {key: "middle"}, {key: "last"}, {key: "suffix", options: list(Codes.Assumed.NameSuffixes)},
		{key: "organization"}, {key: "correspondence", required: true}, {key: "contact", options: list(Codes.Assumed.ContactMethods)}, {key: "comments"},
	}},
	"address":  {modes: []string{"correct", "replace"}, fields: append([]rowField{{key: "kind", required: true, options: list(Codes.Assumed.AddressKinds)}}, addressFields...)},
	"location": {fields: addressFields},
	"phone": {fields: []rowField{
		{key: "country"}, {key: "type", required: true, options: list(Codes.Assumed.PhoneTypes)}, {key: "number", required: true, pattern: phoneUS, message: "Use the format XXX-XXX-XXXX."},
		{key: "ext"}, {key: "primary", options: list([]string{"Yes", "No"})}, {key: "from"}, {key: "to"}, {key: "comments"},
	}},
	"email": {modes: []string{"correct"}, fields: []rowField{
		{key: "type", required: true, options: list(Codes.Assumed.EmailTypes)}, {key: "address", required: true, pattern: emailLike, message: "Enter a valid email address."},
		{key: "comments"}, {key: "primary", options: list([]string{"Yes", "No"})},
	}},
	"othername": {modes: []string{"correct", "replace", "remove"}, fields: []rowField{
		{key: "type", required: true, options: list(Codes.OtherNameTypes)}, {key: "prefix", options: list(Codes.Assumed.NamePrefixes)}, {key: "first"}, {key: "middle"},
		{key: "last", required: true}, {key: "suffix", options: list(Codes.Assumed.NameSuffixes)}, {key: "comments"},
	}},
	"license": {ownDate: true, fields: []rowField{
		{key: "type", required: true, options: list(Codes.Assumed.LicenseTypes)}, {key: "number", required: true}, {key: "status", options: list(Codes.Assumed.LicenseStatuses)},
		{key: "state", options: list(Codes.States)}, {key: "city"}, {key: "county"},
	}},
	"membership": {fields: []rowField{{key: "type", required: true, options: list(Codes.MembershipTypes)}, {key: "id", required: true}}},
}

var demographicFields = []rowField{
	{key: "language", options: list(Codes.Assumed.Languages)}, {key: "gender", options: list(Codes.Assumed.Genders)}, {key: "ethnic_origin", options: list(Codes.Assumed.EthnicOrigins)},
	{key: "dob"}, {key: "marital_status", options: list(Codes.Assumed.MaritalStatuses)}, {key: "occupation"}, {key: "ssn", pattern: ssnLike, message: "Enter 9 digits."},
	{key: "us_citizen", options: list([]string{"Yes", "No"})}, {key: "other_id", options: list(Codes.Assumed.OtherIDs)}, {key: "date_of_death"},
	{key: "dependents", pattern: regexp.MustCompile(`^\d{1,2}$`), message: "Enter a number."}, {key: "profession", options: list(Codes.Assumed.Professions)},
	{key: "company_personnel", options: list([]string{"Yes", "No"})}, {key: "comments"},
}

func checkFields(spec []rowField, in map[string]string, f fieldErrors) map[string]string {
	out := map[string]string{}
	known := map[string]bool{}
	for _, fd := range spec {
		known[fd.key] = true
		v := strings.TrimSpace(in[fd.key])
		switch {
		case v == "" && fd.required:
			f.add(fd.key, "This field is required.")
		case v == "":
		case len(v) > 200:
			f.add(fd.key, "Too long.")
		case fd.options != nil && !oneOf(fd.options(), v):
			f.add(fd.key, "Select a value from the list.")
		case fd.pattern != nil && !fd.pattern.MatchString(v):
			f.add(fd.key, fd.message)
		}
		if v != "" {
			out[fd.key] = v
		}
	}
	for k := range in {
		if !known[k] {
			f.add(k, "Unknown field.")
		}
	}
	return out
}

// current reports whether a row is in effect on the date.
func (r *PartyRow) current(date string) bool { return r.Effective <= date && date < r.Expiration }

// Current returns the rows of a kind in effect on the date.
func (p *Party) Current(kind, date string) []*PartyRow {
	out := []*PartyRow{}
	for _, r := range p.Rows[kind] {
		if r.current(date) {
			out = append(out, r)
		}
	}
	return out
}

// DisplayName is the party's current name.
func (p *Party) DisplayName(date string) string {
	rows := p.Current("name", date)
	if len(rows) == 0 && len(p.Rows["name"]) > 0 {
		rows = p.Rows["name"][len(p.Rows["name"])-1:]
	}
	if len(rows) == 0 {
		return ""
	}
	f := rows[0].Fields
	if f["organization"] != "" {
		return f["organization"]
	}
	return strings.Join(strings.Fields(f["first"]+" "+f["middle"]+" "+f["last"]), " ")
}

// Address is the party's current legal address on one line.
func (p *Party) Address(date string) (line1, line2 string) {
	for _, r := range p.Current("address", date) {
		if r.Fields["kind"] == "Legal" {
			f := r.Fields
			return strings.TrimSpace(f["line1"] + " " + f["line2"]), f["city"] + ", " + f["state"] + " " + f["zip"]
		}
	}
	return "", ""
}

func (p *Party) Email(date string) string {
	rows := p.Current("email", date)
	for _, r := range rows {
		if r.Fields["primary"] == "Yes" {
			return r.Fields["address"]
		}
	}
	if len(rows) > 0 {
		return rows[0].Fields["address"]
	}
	return ""
}

type RowInput struct {
	Version int               `json:"version"`
	Mode    string            `json:"mode"` // add | correct | replace | remove
	RowID   string            `json:"row_id"`
	Fields  map[string]string `json:"fields"`
	// Effective and Expiration are entered on forms that have their own dates.
	Effective  string `json:"effective"`
	Expiration string `json:"expiration"`
}

func (t *tx) party(id string, version int) (*Party, error) {
	p := t.Parties[id]
	if p == nil {
		return nil, ErrNotFound
	}
	if p.Version != version {
		return nil, ErrConflict
	}
	p.Version++
	return p, nil
}

// SaveRow adds, corrects, replaces or removes an effective-dated party row.
// Correct edits the current row and keeps its effective date. Replace ends the
// current row on the system date and adds a new one effective from it. Remove
// ends the row on the system date.
func (s *Store) SaveRow(partyID, kind string, in RowInput) (*Party, error) {
	var out *Party
	err := s.update(func(t *tx) error {
		spec, ok := rowSpecs[kind]
		if !ok {
			return ErrNotFound
		}
		p, err := t.party(partyID, in.Version)
		if err != nil {
			return err
		}
		if p.Rows == nil {
			p.Rows = map[string][]*PartyRow{}
		}
		var row *PartyRow
		if in.Mode != "add" {
			if !oneOf(spec.modes, in.Mode) {
				return &ValidationError{map[string]string{"mode": "This action is not available here."}}
			}
			for _, r := range p.Rows[kind] {
				if r.ID == in.RowID {
					row = r
				}
			}
			if row == nil {
				return ErrNotFound
			}
			if !row.current(t.SystemDate) {
				return rule("Only rows in effect on the system date can be changed.")
			}
		}
		if in.Mode == "remove" {
			row.Expiration = t.SystemDate
			out = p
			return nil
		}
		f := fieldErrors{}
		fields := checkFields(spec.fields, in.Fields, f)
		if kind == "name" {
			if p.Kind == "Organization" && fields["organization"] == "" {
				f.add("organization", "This field is required.")
			}
			if p.Kind == "Person" {
				for _, k := range []string{"first", "last"} {
					if fields[k] == "" {
						f.add(k, "This field is required.")
					}
				}
			}
		}
		eff, exp := t.SystemDate, OpenEnded
		if spec.ownDate {
			if in.Effective != "" {
				eff = in.Effective
			}
			if in.Expiration != "" {
				exp = in.Expiration
			}
			if !validDate(eff) {
				f.add("effective", "Enter a valid date.")
			}
			if !validDate(exp) || exp <= eff {
				f.add("expiration", "The expiration date must be after the effective date.")
			}
		}
		if err := f.err(); err != nil {
			return err
		}
		switch in.Mode {
		case "add":
			p.Rows[kind] = append(p.Rows[kind], &PartyRow{ID: t.next("PR"), Effective: eff, Expiration: exp, Fields: fields})
		case "correct":
			row.Fields = fields
			if spec.ownDate {
				row.Effective, row.Expiration = eff, exp
			}
		case "replace":
			if row.Effective == t.SystemDate {
				row.Fields = fields // a same-day replacement leaves nothing to keep
				break
			}
			row.Expiration = t.SystemDate
			p.Rows[kind] = append(p.Rows[kind], &PartyRow{ID: t.next("PR"), Effective: t.SystemDate, Expiration: OpenEnded, Fields: fields})
		default:
			return &ValidationError{map[string]string{"mode": "Choose add, correct, replace or remove."}}
		}
		out = p
		return nil
	})
	return out, err
}

func (s *Store) SaveDemographics(partyID string, version int, in map[string]string) (*Party, error) {
	var out *Party
	err := s.update(func(t *tx) error {
		p, err := t.party(partyID, version)
		if err != nil {
			return err
		}
		f := fieldErrors{}
		fields := checkFields(demographicFields, in, f)
		for _, k := range []string{"dob", "date_of_death"} {
			if v := fields[k]; v != "" && (!validDate(v) || v > t.SystemDate) {
				f.add(k, "Enter a date on or before the system date.")
			}
		}
		if err := f.err(); err != nil {
			return err
		}
		if fields["us_citizen"] == "" {
			fields["us_citizen"] = "Yes"
		}
		p.Demographics = fields
		out = p
		return nil
	})
	return out, err
}

// NewParty creates a party (used by the seed and by Create billing account
// from a quote).
func (t *tx) newParty(kind string, name map[string]string, eff string) *Party {
	t.Seq["PARTY"]++
	p := &Party{ID: t.next("PTY"), Number: 95370 + t.Seq["PARTY"], Version: 1, Kind: kind, AssociatedWith: "Internal Users", Rows: map[string][]*PartyRow{}, Demographics: map[string]string{"us_citizen": "Yes", "dependents": "0"}}
	if name["correspondence"] == "" {
		n := name["organization"]
		if n == "" {
			n = strings.TrimSpace(name["first"] + " " + name["last"])
		}
		name["correspondence"] = n
	}
	p.Rows["name"] = []*PartyRow{{ID: t.next("PR"), Effective: eff, Expiration: OpenEnded, Fields: name}}
	t.Parties[p.ID] = p
	return p
}

func (t *tx) addRow(p *Party, kind, eff string, fields map[string]string) {
	p.Rows[kind] = append(p.Rows[kind], &PartyRow{ID: t.next("PR"), Effective: eff, Expiration: OpenEnded, Fields: fields})
}

// PartyQuery is Search for a Person or Place.
type PartyQuery struct {
	Target   string `json:"target"` // Agency | Person
	By       string `json:"by"`
	Last     string `json:"last"` // organization or last name
	First    string `json:"first"`
	Middle   string `json:"middle"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	License  string `json:"license"`
	Phone    string `json:"phone"`
	Member   string `json:"membership"`
	Phonetic bool   `json:"phonetic"`
}

type PartyHit struct {
	ID         string `json:"id"`
	Number     int    `json:"number"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	DOB        string `json:"dob"`
	EntityName string `json:"entity_name"`
	PartyType  string `json:"party_type"`
}

func (s *Store) SearchParties(q PartyQuery) ([]PartyHit, error) {
	f := fieldErrors{}
	if !oneOf(Codes.PartySearchTargets, q.Target) {
		f.add("target", "This field is required.")
	}
	switch q.By {
	case "Name and Address", "Other":
		if strings.TrimSpace(q.Last) == "" {
			f.add("last", "This field is required.")
		}
		if !oneOf(Codes.States, q.State) {
			f.add("state", "This field is required.")
		}
	case "License Number":
		if strings.TrimSpace(q.License) == "" {
			f.add("license", "This field is required.")
		}
	case "Phone Number":
		if !phoneUS.MatchString(q.Phone) {
			f.add("phone", "Use the format XXX-XXX-XXXX.")
		}
	case "Membership ID":
		if strings.TrimSpace(q.Member) == "" {
			f.add("membership", "This field is required.")
		}
	default:
		f.add("by", "This field is required.")
	}
	if err := f.err(); err != nil {
		return nil, err
	}
	kind := map[string]string{"Agency": "Organization", "Person": "Person"}[q.Target]
	norm := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	has := func(hay, needle string) bool { return needle == "" || strings.Contains(norm(hay), norm(needle)) }
	out := []PartyHit{}
	s.read(func(st *State) {
		date := st.SystemDate
		for _, p := range st.Parties {
			if p.Kind != kind {
				continue
			}
			hit := false
			switch q.By {
			case "Name and Address", "Other":
				for _, n := range p.Rows["name"] {
					fl := n.Fields
					last := fl["last"]
					if kind == "Organization" {
						last = fl["organization"]
					}
					match := strings.HasPrefix(norm(last), norm(q.Last))
					if q.Phonetic {
						match = soundex(last) == soundex(q.Last)
					}
					if match && has(fl["first"], q.First) && has(fl["middle"], q.Middle) {
						hit = true
					}
				}
				if hit {
					addrHit := false
					for _, a := range append(p.Current("address", date), p.Current("location", date)...) {
						af := a.Fields
						if af["state"] == q.State && has(af["line1"], q.Address) && has(af["city"], q.City) && (q.Zip == "" || af["zip"] == q.Zip) {
							addrHit = true
						}
					}
					hit = addrHit
				}
			case "License Number":
				for _, r := range p.Current("license", date) {
					hit = hit || strings.EqualFold(r.Fields["number"], strings.TrimSpace(q.License))
				}
			case "Phone Number":
				for _, r := range p.Current("phone", date) {
					hit = hit || r.Fields["number"] == q.Phone
				}
			case "Membership ID":
				for _, r := range p.Current("membership", date) {
					hit = hit || strings.EqualFold(r.Fields["id"], strings.TrimSpace(q.Member))
				}
			}
			if !hit {
				continue
			}
			l1, l2 := p.Address(date)
			addr := "No address on file."
			if l1 != "" {
				addr = l1 + ", " + l2
			}
			entity := p.DisplayName(date)
			if p.Kind == "Person" {
				entity = st.agentOf(p.ID)
			}
			out = append(out, PartyHit{ID: p.ID, Number: p.Number, Name: p.DisplayName(date), Address: addr, DOB: p.Demographics["dob"], EntityName: entity, PartyType: p.Kind})
		}
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out, nil
}

// agentOf is the Entity Name column for a person: the agency on their policy.
func (st *State) agentOf(partyID string) string {
	for _, a := range st.Accounts {
		if a.PayorID == partyID || a.PayeeID == partyID {
			for _, tm := range st.termsOf(a.Number) {
				if tm.Agent != nil {
					return tm.Agent.Name
				}
			}
		}
	}
	return ""
}

// soundex is the American Soundex code, for Use phonetic search.
func soundex(s string) string {
	s = strings.ToUpper(s)
	codes := map[rune]byte{'B': '1', 'F': '1', 'P': '1', 'V': '1', 'C': '2', 'G': '2', 'J': '2', 'K': '2', 'Q': '2', 'S': '2', 'X': '2', 'Z': '2', 'D': '3', 'T': '3', 'L': '4', 'M': '5', 'N': '5', 'R': '6'}
	out := []byte{}
	var last byte
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			continue
		}
		c := codes[r]
		if len(out) == 0 {
			out = append(out, byte(r))
			last = c
			continue
		}
		if c != 0 && c != last {
			out = append(out, c)
		}
		if r != 'H' && r != 'W' {
			last = c
		}
		if len(out) == 4 {
			break
		}
	}
	for len(out) > 0 && len(out) < 4 {
		out = append(out, '0')
	}
	return string(out)
}

type NewPartyInput struct {
	Kind    string            `json:"kind"` // Person | Organization
	Name    map[string]string `json:"name"`
	Address map[string]string `json:"address"`
}

// CreateParty is Create A New Party / Add New. The walkthrough does not open
// this form, so it asks for the least a billing account needs: a name and a
// legal address.
func (s *Store) CreateParty(in NewPartyInput) (*Party, error) {
	var out *Party
	err := s.update(func(t *tx) error {
		f := fieldErrors{}
		if in.Kind != "Person" && in.Kind != "Organization" {
			f.add("kind", "Choose Person or Organization.")
		}
		name := checkFields(rowSpecs["name"].fields, withDefault(in.Name, "correspondence", "-"), f)
		if name["correspondence"] == "-" {
			delete(name, "correspondence")
		}
		if in.Kind == "Organization" && name["organization"] == "" {
			f.add("organization", "This field is required.")
		}
		if in.Kind == "Person" {
			for _, k := range []string{"first", "last"} {
				if name[k] == "" {
					f.add(k, "This field is required.")
				}
			}
		}
		address := checkFields(addressFields, in.Address, f)
		if err := f.err(); err != nil {
			return err
		}
		p := t.newParty(in.Kind, name, t.SystemDate)
		address["kind"] = "Legal"
		if address["country"] == "" {
			address["country"] = "United States"
		}
		t.addRow(p, "address", t.SystemDate, address)
		out = p
		return nil
	})
	return out, err
}

func withDefault(m map[string]string, k, v string) map[string]string {
	out := map[string]string{}
	for key, val := range m {
		out[key] = val
	}
	if out[k] == "" {
		out[k] = v
	}
	return out
}
