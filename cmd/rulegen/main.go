// Command rulegen turns a schema-validation report (HTML) into a committed rule
// set. Only rows whose status is "Confirmed" become rules. Every TABLE location
// the report lists is translated to current leaf variants using the variant
// registry in ddl.sql; VIEW locations are ignored. The reviewed primary target
// of each rule comes from a pins file and must be one of those candidates,
// unless the pin gives a reason for a target outside the report ("outside_report",
// e.g. a Helix-native field found when cross-checking the UI field inventory).
// A pin can also exclude a confirmed row, limit a rule to some rows ("when"), or
// add further rules for the same column ("also", ids get a suffix). Rules for
// columns that are not confirmed report rows (e.g. added in the Rules tab and
// exported) come from "additional_rules". The pin types live in internal/rules.
//
//	go run ./cmd/rulegen -report <report.html> -ddl <ddl.sql> \
//	   -pins internal/rules/rulesets/<name>.pins.json -out internal/rules/rulesets/<name>.json
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/ankitkumarflarre/datamigration/internal/rules"
)

func main() {
	report := flag.String("report", "", "schema validation report (HTML)")
	ddl := flag.String("ddl", "", "ddl.sql of the current Helix bundle")
	pinsPath := flag.String("pins", "", "pins JSON (reviewed primary targets)")
	out := flag.String("out", "", "output rule set JSON")
	flag.Parse()
	if *report == "" || *ddl == "" || *pinsPath == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	rs, err := Generate(*report, *ddl, *pinsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "rulegen:", err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(rs, "", "  ")
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "rulegen:", err)
		os.Exit(1)
	}
	fmt.Printf("rulegen: %d rules, %d excluded columns → %s\n", len(rs.Rules), len(rs.Excluded), *out)
}

// Generate builds the rule set deterministically from its three inputs.
func Generate(reportPath, ddlPath, pinsPath string) (*rules.RuleSet, error) {
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	var pins rules.Pins
	pb, err := os.ReadFile(pinsPath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(pb, &pins); err != nil {
		return nil, fmt.Errorf("pins: %w", err)
	}
	reg, err := readRegistry(ddlPath)
	if err != nil {
		return nil, err
	}
	results, locations, err := parseReport(string(raw))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	rs := &rules.RuleSet{Name: pins.RuleSet, Version: pins.Version, Sheet: pins.Sheet, SchemaRenames: pins.SchemaRenames}
	rs.SourceReport.File = filepath.Base(reportPath)
	rs.SourceReport.SHA256 = hex.EncodeToString(sum[:])

	used := map[string]bool{}
	for _, row := range results {
		if !strings.HasPrefix(row.status, "Confirmed") {
			continue
		}
		p, ok := pins.Pins[row.no]
		if !ok {
			return nil, fmt.Errorf("confirmed row #%s (%s, column %s) has no pin", row.no, row.header, row.column)
		}
		used[row.no] = true
		n, _ := strconv.Atoi(row.no)
		if p.Exclude != "" {
			rs.Excluded = append(rs.Excluded, rules.Excluded{ReportNo: n, Column: row.column, Header: row.header, Reason: p.Exclude})
			continue
		}
		cands := map[rules.Target]bool{}
		var locs []string
		for _, l := range locations[row.no] {
			locs = append(locs, l.fq)
			for _, t := range reg.candidates(l, pins.SchemaRenames) {
				cands[t] = true
			}
		}
		sort.Strings(locs)
		all := []rules.ExtraPin{{Pin: p}}
		all = append(all, p.Also...)
		for _, ep := range all {
			if err := reg.check(ep.Pin, cands); err != nil {
				return nil, fmt.Errorf("rule #%s%s: %w", row.no, ep.Suffix, err)
			}
			alts := []rules.Target{}
			for t := range cands {
				if t != ep.Target {
					alts = append(alts, t)
				}
			}
			sort.Slice(alts, func(i, j int) bool { return alts[i].String() < alts[j].String() })
			rs.Rules = append(rs.Rules, rules.Rule{
				ID: fmt.Sprintf("%s-%02d%s", pins.IDPrefix, n, ep.Suffix), ReportNo: n, Column: row.column, Header: row.header,
				ReportStatus: row.status, Confidence: row.confidence, When: ep.When, Target: ep.Target, TargetBasis: ep.OutsideReport,
				Alternatives: alts, ReportLocations: locs, Transform: ep.Transform, Attention: ep.Attention, Note: ep.Note,
				Disabled: ep.Disabled,
			})
		}
	}
	for no := range pins.Pins {
		if !used[no] {
			return nil, fmt.Errorf("pin #%s does not match a confirmed row", no)
		}
	}
	byColumn := map[string]resultRow{}
	for _, row := range results {
		byColumn[row.column] = row
	}
	ids := map[string]bool{}
	for _, r := range rs.Rules {
		ids[r.ID] = true
	}
	for _, a := range pins.Additional {
		if a.Exclude != "" {
			if a.Column == "" {
				return nil, fmt.Errorf("an excluded additional column needs a column letter")
			}
			no := 0
			if row, ok := byColumn[a.Column]; ok {
				no, _ = strconv.Atoi(row.no)
			}
			rs.Excluded = append(rs.Excluded, rules.Excluded{ReportNo: no, Column: a.Column, Header: a.Header, Reason: a.Exclude})
			continue
		}
		if a.ID == "" || a.Column == "" || a.Header == "" || ids[a.ID] {
			return nil, fmt.Errorf("additional rule %q needs a unique id, a column and a header", a.ID)
		}
		ids[a.ID] = true
		if v, ok := reg.variants[a.Target.Variant]; !ok || !v.leaf {
			return nil, fmt.Errorf("additional rule %s: target %s is not a leaf variant in the ddl", a.ID, a.Target)
		}
		status, confidence, no := "Not in the report", "", 0
		if row, ok := byColumn[a.Column]; ok {
			status, confidence = row.status, row.confidence
			no, _ = strconv.Atoi(row.no)
		}
		rs.Rules = append(rs.Rules, rules.Rule{
			ID: a.ID, ReportNo: no, Column: a.Column, Header: a.Header, ReportStatus: status, Confidence: confidence,
			When: a.When, Target: a.Target, TargetBasis: a.OutsideReport, Alternatives: []rules.Target{}, ReportLocations: []string{},
			Transform: a.Transform, Attention: a.Attention, Note: a.Note, Disabled: a.Disabled,
		})
	}
	sort.SliceStable(rs.Rules, func(i, j int) bool { return colNumber(rs.Rules[i].Column) < colNumber(rs.Rules[j].Column) })
	sort.SliceStable(rs.Excluded, func(i, j int) bool { return colNumber(rs.Excluded[i].Column) < colNumber(rs.Excluded[j].Column) })
	return rs, nil
}

// ---- report HTML ----------------------------------------------------------

type resultRow struct{ no, column, header, status, confidence string }
type location struct{ schema, table, column, fq string }

func parseReport(doc string) ([]resultRow, map[string][]location, error) {
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return nil, nil, err
	}
	var tables [][][]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			tables = append(tables, tableRows(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	if len(tables) < 2 {
		return nil, nil, errors.New("report needs the column-level results table and the confirmed mapping locations table")
	}
	var results []resultRow
	for _, r := range tables[0] {
		if len(r) >= 5 && r[0] != "#" {
			results = append(results, resultRow{no: r[0], column: r[1], header: r[2], status: r[3], confidence: r[4]})
		}
	}
	// Only TABLE locations become rule locations; VIEW rows are read-only
	// joins over the same tables and are dropped.
	locs := map[string][]location{}
	for _, r := range tables[1] {
		if len(r) >= 7 && r[0] != "Excel #" && strings.EqualFold(r[5], "TABLE") {
			locs[r[0]] = append(locs[r[0]], location{schema: r[2], table: r[3], column: r[4], fq: r[6] + " [" + r[5] + "]"})
		}
	}
	return results, locs, nil
}

func tableRows(t *html.Node) [][]string {
	var rows [][]string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cells []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cells = append(cells, strings.Join(strings.Fields(text(c)), " "))
				}
			}
			rows = append(rows, cells)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(t)
	return rows
}

func text(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(text(c))
		b.WriteString(" ")
	}
	return b.String()
}

// ---- variant registry from ddl.sql -----------------------------------------

type variant struct {
	id   string
	leaf bool
}

type registry struct {
	variants map[string]variant
	byTable  map[string][]string // schema.table → variants whose lineage includes it
}

func readRegistry(path string) (*registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reg := &registry{variants: map[string]variant{}, byTable: map[string][]string{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	const vPrefix, lPrefix = "INSERT INTO _engine.variant VALUES (", "INSERT INTO _engine.variant_lineage VALUES ("
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, vPrefix):
			vals := sqlValues(line[len(vPrefix):])
			if len(vals) < 17 {
				continue
			}
			reg.variants[vals[0]] = variant{id: vals[0], leaf: vals[15] == "true"}
		case strings.HasPrefix(line, lPrefix):
			vals := sqlValues(line[len(lPrefix):])
			if len(vals) == 4 {
				key := vals[2] + "." + vals[3]
				reg.byTable[key] = append(reg.byTable[key], vals[0])
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(reg.variants) == 0 {
		return nil, errors.New("no _engine.variant rows found in ddl")
	}
	return reg, nil
}

// sqlValues splits "'a', 1, NULL, 'it”s');" into unquoted values.
func sqlValues(s string) []string {
	var out []string
	var cur strings.Builder
	inQ := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case inQ && ch == '\'' && i+1 < len(s) && s[i+1] == '\'':
			cur.WriteByte('\'')
			i++
		case ch == '\'':
			inQ = !inQ
		case !inQ && (ch == ',' || ch == ')'):
			v := strings.TrimSpace(cur.String())
			if v == "NULL" {
				v = ""
			}
			out = append(out, v)
			cur.Reset()
			if ch == ')' {
				return out
			}
		default:
			cur.WriteByte(ch)
		}
	}
	return out
}

func (r *registry) candidates(l location, renames map[string]string) []rules.Target {
	schema := l.schema
	if to, ok := renames[schema]; ok {
		schema = to
	}
	var vs []string
	for _, id := range r.byTable[schema+"."+l.table] {
		if r.variants[id].leaf {
			vs = append(vs, id)
		}
	}
	out := make([]rules.Target, 0, len(vs))
	for _, v := range vs {
		out = append(out, rules.Target{Variant: v, Field: l.column})
	}
	return out
}

// check accepts a pinned target that is one of the report's candidates, or a
// leaf variant outside the report when the pin says why.
func (r *registry) check(p rules.Pin, cands map[rules.Target]bool) error {
	if cands[p.Target] {
		if p.OutsideReport != "" {
			return fmt.Errorf("target %s is a report location; drop outside_report", p.Target)
		}
		return nil
	}
	if p.OutsideReport == "" {
		return fmt.Errorf("pinned target %s is not among the report's locations", p.Target)
	}
	if v, ok := r.variants[p.Target.Variant]; !ok || !v.leaf {
		return fmt.Errorf("target %s is not a leaf variant in the ddl", p.Target)
	}
	return nil
}

func colNumber(c string) int {
	n := 0
	for _, ch := range c {
		n = n*26 + int(ch-'A'+1)
	}
	return n
}
