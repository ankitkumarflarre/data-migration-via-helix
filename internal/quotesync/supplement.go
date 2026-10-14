package quotesync

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ankitkumarflarre/datamigration/internal/quote"
	"github.com/ankitkumarflarre/datamigration/internal/rules"
	"github.com/ankitkumarflarre/datamigration/internal/schema"
)

// supp is a best-effort rule for a quote input the rater does not have.
type supp struct {
	Col    string // a quote field key, or a computed column "Q:…"
	Key    string // the quote field filled from the column on import ("" = not read back)
	Target rules.Target
	Map    map[string]string // cell → Helix value
	When   *rules.Condition
}

func t(variant, field string) rules.Target { return rules.Target{Variant: variant, Field: field} }
func ti(variant, instance, field string) rules.Target {
	return rules.Target{Variant: variant, Instance: instance, Field: field}
}

const (
	dwelling     = "dwelling.property.us.personal"
	policy       = "policy.property.us-fl.personal.safepoint"
	underwriting = "underwriting_personal_home.property.us.personal"
	coverageInst = "coverage_instance.property.us.personal"
)

// direct copies a quote field to a Helix field and reads it back.
func direct(key string, target rules.Target) supp { return supp{Col: key, Key: key, Target: target} }

var yes = &rules.Condition{In: []string{"Yes"}}

func when(col string) *rules.Condition { return &rules.Condition{Column: col, In: yes.In} }

// supplements: quote inputs → Helix (best effort; agreed targets replace these).
var supplements = []supp{
	// New quote
	direct("PolicyDetails.PriorPolicyNumber", t(policy, "previous_policy_number")),
	{Col: "Q:QuoteNumber", Target: t(policy, "quote_number")},
	{Col: "PolicyInput.ExpirationDate", Target: t(policy, "expiration_date")},
	{Col: "ReplacementCostDwellingInput.Indicator", Target: t(policy, "property_valuation_basis"),
		Map: map[string]string{"Yes": "replacement_cost", "No": "actual_cash_value"}},

	// Applicant: the named insured is a person on the row's party (PH-{PN}).
	direct("AccountInput.FirstName", t("person", "given_name")),
	direct("AccountInput.LastName", t("person", "family_name")),
	{Col: "Q:FullName", Target: t("person", "recorded_name")},
	{Col: "Q:FullName", Target: t("person", "display_name")},
	direct("Applicant.DateOfBirth", t("person", "date_of_birth")),
	direct("AccountInput.PrimaryPhone", ti("contact_point", "phone", "channel_value")),
	direct("AccountInput.Email", ti("contact_point", "email", "channel_value")),
	direct("AccountInput.Address1", t("mailing_address.property.us.personal", "address1")),
	direct("AccountInput.Address2", t("mailing_address.property.us.personal", "address2")),
	direct("AccountInput.City", t("mailing_address.property.us.personal", "city")),
	direct("AccountInput.State", t("mailing_address.property.us.personal", "state")),
	direct("AccountInput.ZipCode", t("mailing_address.property.us.personal", "zip")),

	// Risk address
	direct("LocationInput.Address1", t("location_address.property.us.personal", "address1")),
	direct("LocationInput.Address2", t("location_address.property.us.personal", "address2")),
	direct("LocationInput.City", t("location_address.property.us.personal", "city")),
	direct("LocationInput.State", t("location_address.property.us.personal", "state")),
	direct("LocationInput.ZipCode", t("location_address.property.us.personal", "zip_code")),
	{Col: "Q:RiskAddress", Target: t("location.property.us.personal", "address_text")},

	// Dwelling: the Duck Creek copy table keeps the quote's own option codes.
	direct("DwellingInput.UseType", t(dwelling, "use_type")),
	direct("DwellingInput.OccupancyType", t(dwelling, "occupancy_type")),
	direct("DwellingInput.MonthsUnoccupied", t(dwelling, "months_unoccupied")),
	direct("DwellingInput.BuildingType", t(dwelling, "building_type")),
	direct("DwellingInput.Construction", t(dwelling, "construction")),
	direct("DwellingInput.YearBuilt", t(dwelling, "year_built")),
	direct("DwellingInput.RatedProtectionClass", t(dwelling, "rated_protection_class")),
	direct("DwellingInput.NumberOfFamilies", t(dwelling, "number_of_families")),
	direct("DwellingInput.FoundationType", t(dwelling, "foundation_type")),
	direct("DwellingInput.SquareFeet", t(dwelling, "square_feet")),
	direct("DwellingInput.PoolSlide", t(dwelling, "pool_slide")),
	direct("DwellingInput.PoolApprovedFence", t(dwelling, "pool_approved_fence")),
	direct("DwellingInput.PoolDivingBoard", t(dwelling, "pool_diving_board")),
	direct("DwellingInput.PrimaryHeatType", t(dwelling, "primary_heat_type")),
	direct("DwellingInput.SupplementalHeatType", t(dwelling, "supplemental_heat_type")),
	direct("DwellingInput.ElectricalSystem", t(dwelling, "electrical_system")),
	direct("DwellingInput.OpeningProtection", t(dwelling, "opening_protection")),
	direct("DwellingInput.RoofType", t(dwelling, "roof_type")),
	direct("DwellingInput.Sprinkler", t(dwelling, "sprinkler")),
	direct("DwellingInput.LockedSecurityGate", t(dwelling, "locked_security_gate")),
	direct("DwellingInput.SecurityAttendant", t(dwelling, "security_attendant")),
	direct("DwellingInput.Wiring", t(dwelling, "wiring")),
	direct("DwellingInput.NumberOfAmps", t(dwelling, "amps")),
	direct("DwellingInput.PlumbingType", t(dwelling, "plumbing_type")),
	direct("DwellingInput.RoofGeometry", t(dwelling, "roof_geometry")),
	direct("DwellingInput.DistanceToFireStation", t(dwelling, "distance_to_fire_station")),
	direct("DwellingInput.DistanceToHydrant", t(dwelling, "distance_to_hydrant")),
	direct("DwellingInput.RespondingFireDepartment", t(dwelling, "responding_fire_department")),
	direct("UpdatedServicesInfo.RemainingRoofLife", t(dwelling, "remaining_roof_life")),
	direct("UpdatedServicesInfo.ElectricSystemUpdateYear", t(dwelling, "electric_system_update_year")),
	direct("UpdatedServicesInfo.HeatingUpdateYear", t(dwelling, "heating_update_year")),
	direct("UpdatedServicesInfo.PlumbingUpdateYear", t(dwelling, "plumbing_update_year")),
	direct("UpdatedServicesInfo.RoofUpdateYear", t(dwelling, "roof_update_year")),
	direct("DwellingInput.SecondaryWaterResistance", t(dwelling, "secondary_water_resistance")),
	direct("DwellingInput.DwellingAccessibleToFireEquipment", t(dwelling, "dwelling_accessible_to_fire_equipment")),
	direct("DwellingInput.RoofToWallAttachment", t(dwelling, "roof_to_wall_attachment")),
	direct("DwellingInput.RoofCover", t(dwelling, "roof_cover")),
	direct("DwellingInput.Terrain", t(dwelling, "terrain")),
	direct("DwellingInput.FloorToFoundationAttachment", t(dwelling, "floor_to_foundation_attachment")),
	direct("Underwriting.renovation", t("dwelling_asset.property.personal", "is_under_renovation")),

	// Coverages the rater has no column for (coverage codes proposed, S18).
	direct("CoverageDLossOfUseInput.Limit", ti(coverageInst, "coverage_d", "limit_amount")),
	{Col: "CoverageFMedicalInput.Limit", Target: ti(coverageInst, "coverage_f", "limit_amount")},
	{Col: "IdentityFraudExpenseCoverageInput.Indicator", Key: "IdentityFraudExpenseCoverageInput.Indicator", Target: ti(coverageInst, "identity_fraud", "coverage_status"),
		When: when("IdentityFraudExpenseCoverageInput.Indicator"), Map: map[string]string{"Yes": "active"}},
	{Col: "IdentityFraudExpenseCoverageInput.Limit", Key: "IdentityFraudExpenseCoverageInput.Limit", Target: ti(coverageInst, "identity_fraud", "limit_amount"), When: when("IdentityFraudExpenseCoverageInput.Indicator")},
	{Col: "UnscheduledJewelryInput.Indicator", Key: "UnscheduledJewelryInput.Indicator", Target: ti(coverageInst, "unscheduled_jewelry", "coverage_status"),
		When: when("UnscheduledJewelryInput.Indicator"), Map: map[string]string{"Yes": "active"}},
	{Col: "UnscheduledJewelryInput.Limit", Key: "UnscheduledJewelryInput.Limit", Target: ti(coverageInst, "unscheduled_jewelry", "limit_amount"), When: when("UnscheduledJewelryInput.Indicator")},
	{Col: "PersonalInjuryInput.Indicator", Key: "PersonalInjuryInput.Indicator", Target: ti(coverageInst, "personal_injury", "coverage_status"),
		When: when("PersonalInjuryInput.Indicator"), Map: map[string]string{"Yes": "active"}},
	{Col: "PersonalInjuryInput.Limit", Key: "PersonalInjuryInput.Limit", Target: ti(coverageInst, "personal_injury", "limit_amount"), When: when("PersonalInjuryInput.Indicator")},

	// Underwriting answers with a matching question in Helix.
	direct("Underwriting.animals", t(underwriting, "any_animals_or_exotic_pets")),
	direct("Underwriting.trampoline", t(underwriting, "is_there_atrampoline_on_premises")),
	direct("Underwriting.business", t(underwriting, "is_business_conducted_on_premises")),
	direct("Underwriting.declined", t(underwriting, "any_coverage_declined_cancelled")),
	direct("Underwriting.sinkhole", t(underwriting, "is_applicant_aware_of_sinkhole_claim")),
	direct("Underwriting.ForSale", t(underwriting, "is_home_for_sale")),
	direct("Underwriting.Disrepair", t(underwriting, "is_property_disrepair")),
	direct("Underwriting.DayCare", t(underwriting, "is_day_care")),
	direct("Underwriting.Foreclosure", t(underwriting, "had_recent_foreclosure")),
	direct("Underwriting.LeadPaint", t(underwriting, "any_lead_paint")),
	direct("Underwriting.Landfill", t(underwriting, "is_swelling_located_on_landfill")),
	{Col: "Q:UnderwritingRemarks", Target: t(underwriting, "agent_remarks")},
}

// remarkAnswers have no Helix question; they are kept as lines of agent_remarks.
var remarkAnswers = []string{"Underwriting.financialHistory", "Underwriting.criminalHistory", "Underwriting.leadOrLandfill",
	"Underwriting.disrepair", "Underwriting.SwimmingPool", "Underwriting.HotTub", "Underwriting.Farming",
	"Underwriting.IncidentalOccupancy", "Underwriting.Bankruptcy", "Underwriting.RenovationCompletionDate", "Underwriting.RenovationValue"}

const remarksMarker = "--- Other underwriting answers ---"

// list maps one quote collection to one Helix record per entry.
type list struct {
	Key      string // quote collection
	Variant  string
	Instance string // instance prefix; entry n is <prefix>_<n>
	Fields   [][2]string
	Template map[string]rules.FieldSource
}

var lists = []list{
	{Key: "losses", Variant: "loss.property.us.personal", Instance: "loss", Fields: [][2]string{
		{"LossInput.LossType", "loss_type"}, {"LossInput.DateOfLoss", "date_of_loss"}, {"LossInput.AmountPaid", "amount_paid"},
		{"LossInput.Description", "description"}, {"LossInput.Source", "source"}}},
	{Key: "interests", Variant: "additional_other_interest.property.us.personal", Instance: "interest", Fields: [][2]string{
		{"type", "interest_type"}, {"name", "name"}, {"address", "address_line1"}, {"reference", "loan_number"}}},
	{Key: "coapplicants", Variant: "person", Instance: "coapplicant", Fields: [][2]string{
		{"PersonInput.FirstName", "given_name"}, {"PersonInput.LastName", "family_name"}, {"Q:FullName", "recorded_name"}},
		Template: personFields()},
}

func cst(v string) rules.FieldSource { return rules.FieldSource{Const: &v} }

func personFields() map[string]rules.FieldSource {
	return map[string]rules.FieldSource{"party_reference": {Format: "PH-{PN}"}, "record_status": cst("active"), "vital_status": cst("living")}
}

func contact(kind string) rules.TemplateRecord {
	return rules.TemplateRecord{Variant: "contact_point", Instance: kind, Scope: "row", Key: "contact_point_reference",
		Fields: map[string]rules.FieldSource{
			"contact_point_reference": {Format: "CP-{PN}-" + strings.ToUpper(kind)},
			"party_reference":         {Format: "PH-{PN}"},
			"channel_type":            cst(kind),
			"contact_use":             cst("primary"),
			"is_primary":              cst("true"),
			"verification_status":     cst("self_declared"),
		}}
}

// coverage is a product coverage (written once) and its instance on a policy.
func coverage(instance, code, name, section string) []rules.TemplateRecord {
	attention := "Coverage code " + code + " is proposed for the quote flow; agree it with the product owner (S18)."
	return []rules.TemplateRecord{
		{Variant: "coverage.property.us.personal", Instance: instance, Scope: "job", Key: "coverage_code", Fields: map[string]rules.FieldSource{
			"coverage_code": {Const: &code, Attention: attention}, "coverage_name": cst(name), "coverage_status": cst("active"),
			"coverage_type": cst("optional"), "effective_date": {MinCol: "F"}, "homeowners_coverage_section": cst(section),
			"peril_basis": cst("not_applicable"), "trigger_basis": cst("occurrence"),
			"statutory_status_code": {Generate: true, Attention: "Statutory status code is not in the quote; generated placeholder (D6, S11)."},
		}},
		{Variant: coverageInst, Instance: instance, Scope: "row", Key: "coverage_instance_reference", Fields: map[string]rules.FieldSource{
			"coverage_instance_reference": {Format: "CI-{PN}-" + code}, "coverage_identity_key": {Format: "{PN}-" + code},
			"coverage_section_code": cst(code), "coverage_status": cst("active"),
			"product_coverage_reference": {Ref: "coverage.property.us.personal#" + instance}, "policy_version_reference": {Ref: "policy_version"},
		}},
	}
}

func supplementTemplates() []rules.TemplateRecord {
	out := []rules.TemplateRecord{
		{Variant: "person", Scope: "row", Fields: personFields()},
		contact("phone"), contact("email"),
	}
	out = append(out, coverage("coverage_d", "COD", "Coverage D - Loss of Use", "optional")...)
	out = append(out, coverage("coverage_f", "COF", "Coverage F - Medical Payments", "optional")...)
	out = append(out, coverage("identity_fraud", "IDENTITY_FRAUD", "Identity Fraud Expense", "optional")...)
	out = append(out, coverage("unscheduled_jewelry", "UNSCHEDULED_JEWELRY", "Unscheduled Jewelry", "optional")...)
	out = append(out, coverage("personal_injury", "PERSONAL_INJURY", "Personal Injury", "optional")...)
	return out
}

// listCol is the column of field key in entry n (1-based) of a collection.
func listCol(collection string, n int, key string) string {
	return fmt.Sprintf("%s.%d.%s", collection, n, key)
}

// Merge adds the quote-only rules and templates to the reviewed rule set,
// with one record per entry of the quote's lists. Rules and template fields
// the Helix model does not have are left out and reported, so a model change
// never stops a sync. q may be nil (no list entries).
func Merge(ctx context.Context, base *rules.Bundle, q *quote.Quote, sch *schema.Schema) (*rules.Bundle, []string) {
	b := &rules.Bundle{RuleSet: base.RuleSet, Templates: base.Templates, SHA256: base.SHA256 + "+quote"}
	b.RuleSet.Rules = append([]rules.Rule{}, base.RuleSet.Rules...)
	b.Templates.Records = append([]rules.TemplateRecord{}, base.Templates.Records...)
	var notes []string
	variants := map[string]*schema.Variant{}
	describe := func(id string) *schema.Variant {
		v, ok := variants[id]
		if !ok {
			var err error
			if v, err = sch.Variant(ctx, rules.VariantOf(id)); err != nil {
				v = nil
			}
			variants[id] = v
		}
		return v
	}
	known := map[string]bool{}
	for _, r := range b.Templates.Records {
		known[r.ID()] = true
	}
	addTemplate := func(r rules.TemplateRecord) {
		if known[r.ID()] {
			return
		}
		v := describe(r.ID())
		if v == nil {
			notes = append(notes, fmt.Sprintf("Helix has no %s; its quote fields are not written.", r.Variant))
			return
		}
		fields := map[string]rules.FieldSource{}
		for f, src := range r.Fields {
			if _, ok := v.Field(f); ok {
				fields[f] = src
			}
		}
		r.Fields = fields
		known[r.ID()] = true
		b.Templates.Records = append(b.Templates.Records, r)
	}
	for _, r := range supplementTemplates() {
		addTemplate(r)
	}
	n := 0
	addRule := func(s supp, header string) {
		n++
		target := s.Target
		if target.Instance != "" && !known[target.Record()] {
			return
		}
		v := describe(target.Record())
		if v == nil {
			return
		}
		if _, ok := v.Field(target.Field); !ok {
			notes = append(notes, fmt.Sprintf("Helix %s has no field %s; %s is not written.", target.Variant, target.Field, header))
			return
		}
		b.RuleSet.Rules = append(b.RuleSet.Rules, rules.Rule{
			ID: fmt.Sprintf("Q-%02d", n), Column: s.Col, Header: header, ReportStatus: "Quote form", Confidence: "Best effort",
			Target: target, Transform: rules.Transform{Map: s.Map}, When: s.When,
			Note: "Quote input the rater does not have; best-effort target.",
		})
	}
	for _, s := range supplements {
		addRule(s, header(s.Col))
	}
	for _, l := range lists {
		count := 0
		if q != nil {
			count = len(q.Collections[l.Key])
		}
		for i := 1; i <= count; i++ {
			inst := fmt.Sprintf("%s_%d", l.Instance, i)
			fields := l.Template
			if fields == nil {
				fields = map[string]rules.FieldSource{}
			}
			addTemplate(rules.TemplateRecord{Variant: l.Variant, Instance: inst, Scope: "row", Fields: fields})
			for _, f := range l.Fields {
				col := listCol(l.Key, i, f[0])
				addRule(supp{Col: col, Target: ti(l.Variant, inst, f[1])}, header(col))
			}
		}
	}
	sort.Strings(notes)
	return b, notes
}

// header is the column's display name: the quote field label.
func header(col string) string {
	if strings.HasPrefix(col, "Q:") {
		return map[string]string{"Q:QuoteNumber": "Quote number", "Q:FullName": "Applicant name", "Q:RiskAddress": "Risk address",
			"Q:UnderwritingRemarks": "Underwriting remarks"}[col]
	}
	parts := strings.Split(col, ".")
	if len(parts) >= 3 && fieldsOf(parts[0]) != nil {
		key := strings.Join(parts[2:], ".")
		if key == "Q:FullName" {
			return "Co-applicant " + parts[1] + " name"
		}
		return fieldsOf(parts[0])[key].Label + " (" + parts[0] + " " + parts[1] + ")"
	}
	if f, ok := fields[col]; ok {
		return f.Label
	}
	return col
}

func fieldsOf(collection string) map[string]quote.Field { return collectionFields[collection] }

func fullName(v map[string]any, first, middle, last string) string {
	var parts []string
	for _, k := range []string{first, middle, last} {
		if s := text(v[k]); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// supplementCells adds the quote-only columns.
func supplementCells(q *quote.Quote, cells map[string]string) {
	v := q.Values
	for _, s := range supplements {
		if !strings.HasPrefix(s.Col, "Q:") {
			if c := plain(v[s.Col]); c != "" {
				cells[s.Col] = c
			}
		}
	}
	// A state on its own (the form's FL default) is not an address.
	for _, prefix := range []string{"AccountInput.", "LocationInput."} {
		if cells[prefix+"Address1"] == "" && cells[prefix+"City"] == "" && cells[prefix+"ZipCode"] == "" {
			delete(cells, prefix+"State")
		}
	}
	cells["Q:QuoteNumber"] = q.Number
	if name := fullName(v, "AccountInput.FirstName", "AccountInput.MiddleName", "AccountInput.LastName"); name != "" {
		cells["Q:FullName"] = name
	}
	var addr []string
	for _, k := range []string{"LocationInput.Address1", "LocationInput.Address2", "LocationInput.City"} {
		if s := text(v[k]); s != "" {
			addr = append(addr, s)
		}
	}
	if len(addr) > 0 {
		cells["Q:RiskAddress"] = strings.TrimSpace(strings.Join(addr, ", ") + ", " + text(v["LocationInput.State"]) + " " + text(v["LocationInput.ZipCode"]))
	}
	remarks := text(v["Underwriting.Notes"])
	var lines []string
	for _, k := range remarkAnswers {
		if s := plain(v[k]); s != "" {
			lines = append(lines, fields[k].Label+" "+s)
		}
	}
	if len(lines) > 0 {
		remarks = strings.TrimSpace(remarks + "\n" + remarksMarker + "\n" + strings.Join(lines, "\n"))
	}
	if remarks != "" {
		cells["Q:UnderwritingRemarks"] = remarks
	}
	if cells["PolicyDetails.PriorPolicyNumber"] == "" {
		for _, row := range q.Collections["priorInsurance"] {
			if s := text(row["PriorInsurance.PolicyNumber"]); s != "" {
				cells["PolicyDetails.PriorPolicyNumber"] = s
				break
			}
		}
	}
	for _, l := range lists {
		for i, row := range q.Collections[l.Key] {
			for _, f := range l.Fields {
				val := plain(row[f[0]])
				if f[0] == "Q:FullName" {
					val = fullName(row, "PersonInput.FirstName", "PersonInput.MiddleName", "PersonInput.LastName")
				}
				if val != "" {
					cells[listCol(l.Key, i+1, f[0])] = val
				}
			}
		}
	}
}

// supplementValues reads the quote-only columns back. They hold the quote's
// own codes, so they win over values derived from rater columns.
func supplementValues(cells map[string]string, values map[string]any, notes *[]string) map[string][]map[string]any {
	for _, s := range supplements {
		raw := strings.TrimSpace(cells[s.Col])
		f, ok := fields[s.Key]
		if s.Key == "" || raw == "" || !ok || f.ReadOnly {
			continue
		}
		if s.Map != nil {
			if k, ok := inverse(s.Map, raw); ok {
				raw = k
			}
		}
		if v, ok := convert(f, raw); ok {
			values[s.Key] = v
		} else {
			*notes = append(*notes, fmt.Sprintf("%s %q does not fit the quote's options.", f.Label, raw))
		}
	}
	if name := cells["Q:FullName"]; name != "" {
		first, last := text(values["AccountInput.FirstName"]), text(values["AccountInput.LastName"])
		mid := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(name, first), last))
		if first != "" && last != "" && mid != "" && mid != name {
			values["AccountInput.MiddleName"] = mid
		}
	}
	if remarks := cells["Q:UnderwritingRemarks"]; remarks != "" {
		notesText, other, _ := strings.Cut(remarks, remarksMarker)
		if s := strings.TrimSpace(notesText); s != "" {
			values["Underwriting.Notes"] = s
		}
		for _, line := range strings.Split(other, "\n") {
			for _, k := range remarkAnswers {
				f := fields[k]
				if rest, ok := strings.CutPrefix(strings.TrimSpace(line), f.Label+" "); ok {
					if v, ok := convert(f, rest); ok {
						values[k] = v
					}
				}
			}
		}
	}
	colls := map[string][]map[string]any{}
	for _, l := range lists {
		for i := 1; ; i++ {
			row := map[string]any{}
			for _, f := range l.Fields {
				raw := cells[listCol(l.Key, i, f[0])]
				if fd, ok := collectionFields[l.Key][f[0]]; ok && raw != "" {
					if v, ok := convert(fd, raw); ok {
						row[f[0]] = v
					}
				}
			}
			if len(row) == 0 {
				break
			}
			colls[l.Key] = append(colls[l.Key], row)
		}
	}
	return colls
}
