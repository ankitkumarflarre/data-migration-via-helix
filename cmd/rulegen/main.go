// Command rulegen turns a schema-validation report (HTML) into a committed rule
// set. Only rows whose status is "Confirmed" become rules. Every location the
// report lists is translated to current leaf variants using the variant
// registry in ddl.sql; the reviewed primary target of each rule comes from a
// pins file and must be one of those candidates.
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

type pin struct {
	Target    rules.Target    `json:"target"`
	Transform rules.Transform `json:"transform"`
	Attention string          `json:"attention,omitempty"`
	Note      string          `json:"note,omitempty"`
}

type pinsFile struct {
	RuleSet       string            `json:"rule_set"`
	Version       string            `json:"version"`
	IDPrefix      string            `json:"id_prefix"`
	Sheet         string            `json:"sheet"`
	SchemaRenames map[string]string `json:"schema_renames"`
	Pins          map[string]pin    `json:"pins"` // by report "#"
}

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
	fmt.Printf("rulegen: %d confirmed rules → %s\n", len(rs.Rules), *out)
}

// Generate builds the rule set deterministically from its three inputs.
func Generate(reportPath, ddlPath, pinsPath string) (*rules.RuleSet, error) {
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, err
	}
	var pins pinsFile
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
		cands := map[rules.Target]bool{}
		var locs []string
		for _, l := range locations[row.no] {
			locs = append(locs, l.fq)
			for _, t := range reg.candidates(l, pins.SchemaRenames) {
				cands[t] = true
			}
		}
		if !cands[p.Target] {
			return nil, fmt.Errorf("rule #%s: pinned target %s is not among the report's locations", row.no, p.Target)
		}
		alts := []rules.Target{}
		for t := range cands {
			if t != p.Target {
				alts = append(alts, t)
			}
		}
		sort.Slice(alts, func(i, j int) bool { return alts[i].String() < alts[j].String() })
		sort.Strings(locs)
		n, _ := strconv.Atoi(row.no)
		rs.Rules = append(rs.Rules, rules.Rule{
			ID: fmt.Sprintf("%s-%02d", pins.IDPrefix, n), ReportNo: n, Column: row.column, Header: row.header,
			ReportStatus: row.status, Confidence: row.confidence, Target: p.Target, Alternatives: alts,
			ReportLocations: locs, Transform: p.Transform, Attention: p.Attention, Note: p.Note,
		})
	}
	for no := range pins.Pins {
		if !used[no] {
			return nil, fmt.Errorf("pin #%s does not match a confirmed row", no)
		}
	}
	sort.Slice(rs.Rules, func(i, j int) bool { return rs.Rules[i].ReportNo < rs.Rules[j].ReportNo })
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
	locs := map[string][]location{}
	for _, r := range tables[1] {
		if len(r) >= 7 && r[0] != "Excel #" {
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
	id, entity, view string
	leaf             bool
}

type registry struct {
	variants map[string]variant
	byView   map[string]string
	leaves   map[string][]string // entity → leaf variants
	byTable  map[string][]string // schema.table → variants whose lineage includes it
}

func readRegistry(path string) (*registry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reg := &registry{variants: map[string]variant{}, byView: map[string]string{}, leaves: map[string][]string{}, byTable: map[string][]string{}}
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
			v := variant{id: vals[0], entity: vals[1], leaf: vals[15] == "true", view: vals[16]}
			reg.variants[v.id] = v
			if v.view != "" {
				reg.byView[v.view] = v.id
			}
			if v.leaf {
				reg.leaves[v.entity] = append(reg.leaves[v.entity], v.id)
			}
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
	var vs []string
	if l.schema == "_view" {
		if leaves, ok := r.leaves[l.table]; ok { // entity-wide union view
			vs = leaves
		} else if id, ok := r.byView["_view."+l.table]; ok {
			vs = []string{id}
		}
	} else {
		schema := l.schema
		if to, ok := renames[schema]; ok {
			schema = to
		}
		for _, id := range r.byTable[schema+"."+l.table] {
			if r.variants[id].leaf {
				vs = append(vs, id)
			}
		}
	}
	out := make([]rules.Target, 0, len(vs))
	for _, v := range vs {
		out = append(out, rules.Target{Variant: v, Field: l.column})
	}
	return out
}
