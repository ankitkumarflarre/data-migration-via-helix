package quote

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// deriveValues is also applied on reads so older drafts receive the same
// read-only fields as new quotes. Provider-dependent outputs stay unavailable.
func deriveValues(q *Quote) {
	if q.Values == nil {
		q.Values = map[string]any{}
	}
	if q.Collections == nil {
		q.Collections = map[string][]map[string]any{}
	}
	for _, p := range Schema.Pages {
		for _, f := range p.Fields {
			deriveField(f, q.Values, 0)
		}
		if p.Collection != nil {
			for i, row := range q.Collections[p.Collection.Key] {
				if row == nil {
					row = map[string]any{}
					q.Collections[p.Collection.Key][i] = row
				}
				for _, f := range p.Collection.Fields {
					deriveField(f, row, i)
				}
			}
		}
	}
}
func deriveField(f Field, values map[string]any, row int) {
	if !f.ReadOnly {
		return
	}
	if f.Unavailable != "" {
		delete(values, f.Key)
		return
	}
	var result any
	switch f.Derive {
	case "fullName":
		parts := []string{}
		for _, key := range []string{"AccountInput.FirstName", "AccountInput.MiddleName", "AccountInput.LastName"} {
			if s, ok := values[key].(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
		}
		result = strings.Join(parts, " ")
	case "expirationDate":
		if s, ok := values["PolicyInput.EffectiveDate"].(string); ok {
			if date, err := time.Parse("2006-01-02", s); err == nil {
				lastDay := time.Date(date.Year()+1, date.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
				day := date.Day()
				if day > lastDay {
					day = lastDay
				}
				result = time.Date(date.Year()+1, date.Month(), day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
			}
		}
	case "coapplicantLabel":
		result = fmt.Sprintf("Co-applicant #%d", row+1)
	case "rowNumber":
		result = fmt.Sprint(row + 1)
	case "interestDescription":
		parts := []string{}
		for _, key := range []string{"name", "type"} {
			if s, ok := values[key].(string); ok && s != "" {
				parts = append(parts, s)
			}
		}
		result = strings.Join(parts, " · ")
	case "lossOfUseIncluded":
		key, factor := "CoverageADwellingInput.Limit", 0.2
		if values["DwellingInput.Form"] == "HO6" {
			key, factor = "CoverageCPersonalPropertyHO46Input.Limit", 0.4
		}
		if n, ok := values[key].(float64); ok {
			result = math.Round(n * factor)
		}
	case "constant":
		result = f.Default
	default:
		// Read-only mirrors, such as form selection on Risk schedule, retain their
		// authoritative value from the editable page.
		if f.Default == nil {
			return
		}
		result = f.Default
	}
	if result == nil {
		delete(values, f.Key)
	} else {
		values[f.Key] = result
	}
}
