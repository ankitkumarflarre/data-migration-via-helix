// Package quotesync hands PersonalHome quotes to Helix while they are being
// filled in, and opens policies already in Helix (migrated from the rater
// workbook) as quotes.
//
// Both directions go through the rater's Policy Data columns, so the reviewed
// rules of the rule set stay the single mapping to Helix:
//
//	quote values → rater-shaped cells → plan (rules + templates) → execute → Helix
//	Helix records → inverse of the rules → rater-shaped cells → quote values
//
// Quote inputs the rater never had (names, addresses, underwriting answers, …)
// are best-effort rules of their own (supplement.go) whose "column" is the quote
// field key, or a computed column named "Q:…".
package quotesync

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ankitkumarflarre/datamigration/internal/quote"
	"github.com/ankitkumarflarre/datamigration/internal/transform"
)

// link ties one rater column to one quote field.
type link struct {
	Col  string
	Key  string
	Form string // only for quotes of this policy form ("" = any)
	// ToCell turns a quote value (canonical text, booleans as "true"/"false")
	// into the rater's spelling. A value missing from a non-nil map is not
	// written: the rater has no equivalent.
	ToCell map[string]string
	// ToQuote is tried before the inverse of ToCell when reading a cell.
	ToQuote map[string]string
	// ReadOnly links only fill the quote from a cell; they never write it.
	ReadOnly bool
}

var yesNo = map[string]string{"true": "Yes", "false": "No"}

// links are the quote fields the rater's columns hold (docs/QUOTE_FLOW_DATA_COVERAGE.md).
var links = []link{
	{Col: "B", Key: "DwellingInput.Form", ToCell: map[string]string{"HO3": "HO3", "HO6": "HO6"}},
	{Col: "B", Key: "DwellingInput.BuildingType", ReadOnly: true, ToQuote: map[string]string{"HO3": "Dwelling", "HO6": "Condo"}},
	{Col: "C", Key: "LocationInput.County"},
	{Col: "F", Key: "PolicyInput.EffectiveDate"},
	{Col: "G", Key: "PolicyDetails.RiskOrigin"},
	{Col: "H", Key: "DwellingInput.UseType",
		ToCell:  map[string]string{"Primary": "Primary", "Secondary": "Secondary", "Seasonal3to6": "Seasonal", "Seasonal6Mths": "Seasonal"},
		ToQuote: map[string]string{"Seasonal": "Seasonal6Mths"}},
	{Col: "I", Key: "DwellingInput.OccupancyType", ToCell: map[string]string{"Owner": "O", "Tenant": "T"}},
	{Col: "J", Key: "DwellingInput.YearBuilt"},
	{Col: "K", Key: "DwellingInput.Construction", ToCell: map[string]string{"Frame": "Frame", "Masonry": "Masonry", "MasonryVeneer": "Masonry Veneer",
		"Aluminum": "Aluminum", "PlasticSiding": "Plastic Siding", "Superior": "Superior"}},
	{Col: "L", Key: "DwellingInput.RatedProtectionClass"},
	{Col: "M", Key: "DwellingInput.NumberOfStories"},
	{Col: "O", Key: "DwellingInput.NumberOfUnits"},
	{Col: "R", Key: "DwellingInput.Sprinkler", ToCell: map[string]string{"None": "Neither", "ClassA": "Class A", "ClassB": "Class B",
		"SinglePoint": "SinglePoint", "WholeHome": "WholeHome", "Shutoff": "Shutoff"}},
	{Col: "S", Key: "DwellingInput.BuildingCodeEffectivenessGrading"},
	{Col: "T", Key: "DwellingInput.RoofCover", ToCell: map[string]string{"FBCEquivalent": "FBC Equivalent", "NonFBCEquivalent": "Non-FBC Equivalent",
		"ReinforcedConcreteRoofDeck": "Reinforced Concrete Roof Deck"}},
	{Col: "U", Key: "DwellingInput.RoofToWallAttachment", ToCell: map[string]string{"ToeNails": "ToeNails", "Clips": "Clips",
		"SingleWraps": "SingleWraps", "DoubleWraps": "DoubleWraps", "Unknown": "Unknown"}},
	{Col: "V", Key: "DwellingInput.RoofDeckAttachment", ToCell: map[string]string{"A6d@612": "Level A: 6d @ 6/12", "B8d@612": "Level B: 8d @ 6/12",
		"C8d@66": "Level C: 8d @ 6/6", "WoodDeck": "Wood Deck", "MetalDeck": "Metal Deck", "Reinforce": "Reinforced Concrete Roof Deck"}},
	{Col: "W", Key: "DwellingInput.RoofGeometry", ToCell: map[string]string{"Hip": "Hip", "Gable": "Gable", "Flat": "Flat", "Other": "Other"},
		ToQuote: map[string]string{"Hip Roof": "Hip"}},
	{Col: "X", Key: "DwellingInput.Terrain", ToCell: map[string]string{"B": "B", "C": "C", "H": "H"}},
	{Col: "Y", Key: "DwellingInput.SecondaryWaterResistance", ToCell: map[string]string{"Yes": "SWR", "No": "No SWR"}},
	{Col: "Z", Key: "DwellingInput.OpeningProtection", ToCell: map[string]string{"ClassA": "Class A", "ClassB": "Class B", "None": "None", "Unknown": "Unknown"}},
	{Col: "AA", Key: "DwellingInput.WindSpeedDesign", ToCell: map[string]string{"100": "Greater than or equal to 100",
		"110": "Greater than or equal to 110", "120 mph+": "Greater than or equal to 120"}},
	{Col: "AB", Key: "DwellingInput.WindSpeedLocation", ToCell: map[string]string{"100": "100", "110": "110", "120+": "120", "120WBDR": "120 and WBDR"}},
	{Col: "AC", Key: "CoverageADwellingInput.Limit"},
	{Col: "AD", Key: "CoverageBOtherStructuresInput.Limit"},
	{Col: "AE", Key: "CoverageCPersonalPropertyHO3Input.Limit", Form: "HO3"},
	{Col: "AE", Key: "CoverageCPersonalPropertyHO46Input.Limit", Form: "HO6"},
	{Col: "AF", Key: "CoverageELiabilityInput.Limit", ToCell: map[string]string{"100000": "$100,000 Included", "300000": "Increase to $300,000"}},
	{Col: "AG", Key: "DwellingInput.Deductible"},
	{Col: "AH", Key: "DwellingInput.HurricaneDeductible", ToCell: map[string]string{"N": "Ex-Wind", "500": "500", "2%": "0.02", "5%": "0.05", "10%": "0.1"}},
	{Col: "AI", Key: "RiskInput.PermittedIncidentalOccupancy", ToCell: yesNo},
	{Col: "AK", Key: "OrdinanceOrLawInput.Limit", ToCell: map[string]string{"25": "0.25", "50": "0.5"}},
	{Col: "AL", Key: "LossAssessmentInput.Indicator", ToCell: map[string]string{"true": "Increase to $2,000", "false": "$1,000 Included"}},
	{Col: "AL", Key: "LossAssessmentInput.Limit", ReadOnly: true, ToQuote: map[string]string{"Increase to $2,000": "2000"}},
	{Col: "AM", Key: "CarportsPoolCagesInput.Limit"},
	{Col: "AN", Key: "ReplacementCostContentsInput.Indicator", ToCell: yesNo},
	{Col: "AO", Key: "WaterBackupAndSumpOverflowInput.Indicator", ToCell: yesNo},
	{Col: "AP", Key: "SinkholeInput.Indicator", ToCell: yesNo},
	{Col: "AQ", Key: "LimitedFungiBacteriaInput.Limit", ToCell: map[string]string{"10/50": "$10,000 Included", "25/50": "25000", "50/50": "50000"},
		ToQuote: map[string]string{"10000": "10/50"}},
	{Col: "AS", Key: "RiskInput.EquipmentBreakdown", ToCell: yesNo},
	{Col: "AT", Key: "UnitsRegularlyRentedToOthersInput.Indicator", ToCell: yesNo},
	{Col: "AX", Key: "PolicyDetails.ConsentToRate", ToCell: map[string]string{"true": "1", "false": "0"}},
	{Col: "BV", Key: "PolicyDetails.PriorPolicyNumber"},
}

// writtenCols are the columns a quote always decides (an empty value means
// "nothing to write"), so an imported policy's saved cells never override them.
var writtenCols = func() map[string]bool {
	m := map[string]bool{"A": true}
	for _, l := range links {
		if !l.ReadOnly {
			m[l.Col] = true
		}
	}
	return m
}()

// fields indexes the quote schema; an editable definition wins over a
// read-only mirror of the same key on another page.
var fields, collectionFields = func() (map[string]quote.Field, map[string]map[string]quote.Field) {
	top := map[string]quote.Field{}
	coll := map[string]map[string]quote.Field{}
	for _, p := range quote.Schema.Pages {
		for _, f := range p.Fields {
			if prev, ok := top[f.Key]; !ok || (prev.ReadOnly && !f.ReadOnly) {
				top[f.Key] = f
			}
		}
		if c := p.Collection; c != nil {
			coll[c.Key] = map[string]quote.Field{}
			for _, f := range c.Fields {
				coll[c.Key][f.Key] = f
			}
		}
	}
	return top, coll
}()

// text is a quote value as canonical text.
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return transform.Canonical(x)
	case string:
		return strings.TrimSpace(x)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// plain is a quote value as a cell of a column with no value map: booleans
// read Yes/No as the rater writes them.
func plain(v any) string {
	if b, ok := v.(bool); ok {
		return yesNo[strconv.FormatBool(b)]
	}
	return text(v)
}

func (l link) cell(v any) string {
	t := text(v)
	if t == "" {
		return ""
	}
	if l.ToCell == nil {
		return plain(v)
	}
	return l.ToCell[t]
}

// Cells is the quote as one rater-shaped row: rater columns by letter, plus
// the quote-only columns the supplement rules read.
func Cells(q *quote.Quote) map[string]string {
	cells := map[string]string{}
	if q.Source != nil {
		for col, v := range q.Source.Cells {
			if !writtenCols[col] && !strings.Contains(col, ".") && !strings.HasPrefix(col, "Q:") {
				cells[col] = v
			}
		}
	}
	cells["A"] = q.PolicyNumber()
	if cells["D"] == "" {
		cells["D"] = "1" // a new application is the policy's first term
	}
	form := text(q.Values["DwellingInput.Form"])
	for _, l := range links {
		if l.ReadOnly || (l.Form != "" && l.Form != form) {
			continue
		}
		if c := l.cell(q.Values[l.Key]); c != "" {
			cells[l.Col] = c
		}
	}
	if c := cells["C"]; c != "" {
		cells["C"] = strings.ToUpper(c)
	}
	supplementCells(q, cells)
	return cells
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]`)

func squash(s string) string { return nonAlnum.ReplaceAllString(strings.ToLower(s), "") }

// convert reads a cell into a value of the quote field, or reports that it
// does not fit.
func convert(f quote.Field, s string) (any, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	switch f.Type {
	case "boolean":
		switch strings.ToLower(s) {
		case "true", "yes", "y", "1":
			return true, true
		case "false", "no", "n", "0":
			return false, true
		}
		return nil, false
	case "integer", "number":
		n, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimPrefix(s, "$"), ",", ""), 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil, false
		}
		if f.Type == "integer" {
			n = math.Round(n)
		}
		return n, true
	case "date":
		d, err := transform.Coerce(transform.FieldType{Kind: transform.Date}, s, false)
		if err != nil {
			return nil, false
		}
		return d, true
	}
	if len(f.Options) == 0 {
		return s, true
	}
	for _, o := range f.Options {
		if o == s {
			return o, true
		}
	}
	for _, o := range f.Options {
		if strings.EqualFold(o, s) || squash(o) == squash(s) {
			return o, true
		}
	}
	return nil, false
}

// lookup finds a cell in a map, exactly or ignoring case and spacing.
func lookup(m map[string]string, s string) (string, bool) {
	if v, ok := m[s]; ok {
		return v, true
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.EqualFold(k, s) || squash(k) == squash(s) {
			return m[k], true
		}
	}
	return "", false
}

// inverse finds the quote value whose cell is s, preferring the shortest.
func inverse(m map[string]string, s string) (string, bool) {
	var best string
	found := false
	for q, c := range m {
		if strings.EqualFold(c, s) || squash(c) == squash(s) {
			if !found || len(q) < len(best) || (len(q) == len(best) && q < best) {
				best, found = q, true
			}
		}
	}
	return best, found
}

// Values reads rater-shaped cells back into quote values. Read-only quote
// fields are left to the quote's own defaults. notes lists cells that have no
// quote equivalent.
func Values(cells map[string]string) (values map[string]any, collections map[string][]map[string]any, notes []string) {
	values = map[string]any{}
	form := strings.ToUpper(strings.TrimSpace(cells["B"]))
	for _, l := range links {
		raw := strings.TrimSpace(cells[l.Col])
		f, ok := fields[l.Key]
		if raw == "" || !ok || f.ReadOnly || (l.Form != "" && l.Form != form) {
			continue
		}
		s := raw
		if v, ok := lookup(l.ToQuote, raw); ok {
			s = v
		} else if v, ok := inverse(l.ToCell, raw); ok {
			s = v
		} else if l.ReadOnly {
			continue
		}
		v, ok := convert(f, s)
		if !ok {
			notes = append(notes, fmt.Sprintf("Column %s %q does not fit %s.", l.Col, raw, f.Label))
			continue
		}
		values[l.Key] = v
	}
	collections = supplementValues(cells, values, &notes)
	return values, collections, notes
}
