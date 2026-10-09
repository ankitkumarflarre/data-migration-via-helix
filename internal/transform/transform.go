// Package transform holds the pure, deterministic value functions: cell
// normalisation, value maps and coercion to a Helix field type.
package transform

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"github.com/ankitkumarflarre/datamigration/internal/rules"
)

// Normalize trims a raw cell, treats NULL as empty and prints integral numbers without a fraction.
func Normalize(raw string) string {
	v := strings.TrimSpace(raw)
	if strings.EqualFold(v, "null") {
		return ""
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil && !strings.ContainsAny(v, "xX") {
		if f == math.Trunc(f) && math.Abs(f) < 1e15 {
			return strconv.FormatInt(int64(f), 10)
		}
	}
	return v
}

// Apply runs a rule's case change and value map. Map keys match trimmed and case-insensitively;
// a value missing from the map passes through unchanged (coercion then decides).
func Apply(t rules.Transform, v string) string {
	return MapValue(t.Map, applyCase(t.Case, v))
}

func applyCase(c, v string) string {
	switch c {
	case "upper":
		return strings.ToUpper(v)
	case "lower":
		return strings.ToLower(v)
	}
	return v
}

// MapValue looks v up in m (trimmed, case-insensitive).
func MapValue(m map[string]string, v string) string {
	if len(m) == 0 {
		return v
	}
	if out, ok := m[v]; ok {
		return out
	}
	for k, out := range m {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(v)) {
			return out
		}
	}
	return v
}

// Kind is the coercion class of a Helix field type.
type Kind string

const (
	String    Kind = "string"
	Integer   Kind = "integer"
	Number    Kind = "number"
	Boolean   Kind = "boolean"
	Date      Kind = "date"
	Timestamp Kind = "timestamp"
	Enum      Kind = "enum"
	Reference Kind = "reference"
)

// FieldType is what coercion needs to know about a target field.
type FieldType struct {
	Raw    string   `json:"type"`
	Kind   Kind     `json:"kind"`
	MaxLen int      `json:"max_len,omitempty"`
	Enum   []string `json:"enum,omitempty"`
}

var lenRe = regexp.MustCompile(`^\w+\((\d+)\)$`)

// ParseType classifies a /describe type string such as "string(255)" or "enum(a,b)".
func ParseType(raw string, enum []string) FieldType {
	t := FieldType{Raw: raw, Enum: enum}
	r := strings.ToLower(raw)
	switch {
	case len(enum) > 0 || strings.HasPrefix(r, "enum"):
		t.Kind = Enum
		if len(t.Enum) == 0 {
			inner := strings.TrimSuffix(strings.TrimPrefix(raw, "enum("), ")")
			for _, e := range strings.Split(inner, ",") {
				t.Enum = append(t.Enum, strings.TrimSpace(e))
			}
		}
	case strings.HasPrefix(r, "reference"):
		t.Kind = Reference
	case r == "integer" || r == "bigint" || r == "int":
		t.Kind = Integer
	case strings.HasPrefix(r, "decimal") || strings.HasPrefix(r, "numeric") || r == "number" || strings.HasPrefix(r, "money") || r == "float":
		t.Kind = Number
	case r == "boolean" || r == "bool":
		t.Kind = Boolean
	case r == "date":
		t.Kind = Date
	case strings.HasPrefix(r, "timestamp") || r == "datetime":
		t.Kind = Timestamp
	default:
		t.Kind = String
		if m := lenRe.FindStringSubmatch(r); m != nil {
			t.MaxLen, _ = strconv.Atoi(m[1])
		}
	}
	return t
}

var dateLayouts = []string{"2006-01-02", "1/2/2006", "01/02/2006", "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05"}

func parseDate(v string, date1904 bool) (time.Time, error) {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		t, err := excelize.ExcelDateToTime(f, date1904)
		if err != nil {
			return time.Time{}, fmt.Errorf("%q is not a valid Excel date", v)
		}
		return t, nil
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("%q is not a date", v)
}

// Coerce converts a normalised, transformed value to the JSON value Helix expects.
func Coerce(t FieldType, v string, date1904 bool) (any, error) {
	switch t.Kind {
	case Integer:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, nil
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil && f == math.Trunc(f) {
			return int64(f), nil
		}
		return nil, fmt.Errorf("%q is not a whole number", v)
	case Number:
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return nil, fmt.Errorf("%q is not a number", v)
		}
		return v, nil
	case Boolean:
		switch strings.ToLower(v) {
		case "true", "yes", "y", "1":
			return true, nil
		case "false", "no", "n", "0":
			return false, nil
		}
		return nil, fmt.Errorf("%q is not true/false", v)
	case Date:
		d, err := parseDate(v, date1904)
		if err != nil {
			return nil, err
		}
		return d.Format("2006-01-02"), nil
	case Timestamp:
		d, err := parseDate(v, date1904)
		if err != nil {
			return nil, err
		}
		return d.UTC().Format(time.RFC3339), nil
	case Enum:
		for _, e := range t.Enum {
			if v == e {
				return e, nil
			}
		}
		for _, e := range t.Enum {
			if strings.EqualFold(v, e) {
				return e, nil
			}
		}
		return nil, fmt.Errorf("%q is not one of %s", v, strings.Join(t.Enum, ", "))
	default:
		if t.MaxLen > 0 && utf8.RuneCountInString(v) > t.MaxLen {
			return nil, fmt.Errorf("value is longer than %d characters", t.MaxLen)
		}
		return v, nil
	}
}

// Placeholder is the deterministic generated value for a required field the
// sheet does not provide (D6). scope is the row key or the rule-set name.
func Placeholder(t FieldType, field, scope, rowDate string) any {
	switch t.Kind {
	case Enum:
		if len(t.Enum) > 0 {
			return t.Enum[0]
		}
	case Boolean:
		return false
	case Integer:
		return int64(0)
	case Number:
		return "0"
	case Date:
		if rowDate != "" {
			return rowDate
		}
		return "1970-01-01"
	case Timestamp:
		if rowDate != "" {
			return rowDate + "T00:00:00Z"
		}
		return "1970-01-01T00:00:00Z"
	}
	v := "GEN-" + field + "-" + shortHash(scope+"|"+field)
	if t.MaxLen > 0 && len(v) > t.MaxLen {
		v = v[:t.MaxLen]
	}
	return v
}

// Canonical prints a JSON value the same way whether it came from us or from Helix,
// so existing and incoming values can be compared.
func Canonical(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case int:
		return strconv.Itoa(x)
	case float64:
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// AddYears reads v as a date and returns it plus n whole years as YYYY-MM-DD,
// or "" when v is not a date.
func AddYears(v string, n int, date1904 bool) string {
	d, err := Coerce(FieldType{Kind: Date}, v, date1904)
	if err != nil || d == nil {
		return ""
	}
	t, err := time.Parse("2006-01-02", fmt.Sprint(d))
	if err != nil {
		return ""
	}
	return t.AddDate(n, 0, 0).Format("2006-01-02")
}
