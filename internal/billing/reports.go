package billing

import (
	"sort"
	"strconv"
	"strings"
)

// Report is a rendered report: header facts and one or more grids.
type Report struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Date     string          `json:"date"`
	Meta     []NameValue     `json:"meta"`
	Sections []ReportSection `json:"sections"`
}

type ReportSection struct {
	Heading      string     `json:"heading,omitempty"`
	HeadingRight string     `json:"heading_right,omitempty"`
	Columns      []string   `json:"columns"`
	Numeric      []bool     `json:"numeric"`
	Rows         [][]string `json:"rows"`
	Kinds        []string   `json:"kinds"` // per row: "", group, subtotal, total
	Empty        string     `json:"empty,omitempty"`
}

func (r *ReportSection) add(kind string, cells ...string) {
	r.Rows = append(r.Rows, cells)
	r.Kinds = append(r.Kinds, kind)
}

func section(cols string, numeric ...int) ReportSection {
	c := strings.Split(cols, "|")
	n := make([]bool, len(c))
	for _, i := range numeric {
		n[i] = true
	}
	return ReportSection{Columns: c, Numeric: n, Rows: [][]string{}, Kinds: []string{}}
}

// ReportSlug is the report id used in URLs.
func ReportSlug(name string) string { return strings.ToLower(strings.ReplaceAll(name, " ", "-")) }

// Company facts in the report header. The walkthrough shows the Duck Creek
// sample "Top Insurance Company" (O10); the copy uses the carrier's name.
const companyName = "SafePoint Insurance"

func (s *Store) Report(id string) (*Report, error) {
	var r *Report
	s.read(func(st *State) {
		for _, name := range Codes.Reports {
			if ReportSlug(name) == id {
				r = st.report(name)
			}
		}
	})
	if r == nil {
		return nil, ErrNotFound
	}
	return r, nil
}

func (st *State) report(name string) *Report {
	r := &Report{ID: ReportSlug(name), Title: name + " Report", Date: US(st.SystemDate), Meta: []NameValue{{"Company Name", companyName}, {"Processing Branch", "Cajun UnderWriters"}, {"Currency", "US Dollars ($)"}}}
	sd := st.SystemDate
	switch name {
	case "Aged Receivables":
		sec := section("Account|Policy|+30 Days|+60 Days|+90 Days|+180 Days|Total", 2, 3, 4, 5, 6)
		terms := sortedValues(st.Terms, func(a, b *Term) bool {
			return a.AccountNumber+a.PolicyNumber+a.Effective < b.AccountNumber+b.PolicyNumber+b.Effective
		})
		for _, tm := range terms {
			var buckets [4]int64
			for _, in := range tm.Installments {
				if b := in.Balance(); b > 0 && in.DueDate < sd {
					d := daysBetween(in.DueDate, sd)
					switch {
					case d >= 180:
						buckets[3] += b
					case d >= 90:
						buckets[2] += b
					case d >= 60:
						buckets[1] += b
					default:
						buckets[0] += b
					}
				}
			}
			total := buckets[0] + buckets[1] + buckets[2] + buckets[3]
			if total > 0 {
				sec.add("", tm.AccountNumber, tm.PolicyNumber, Plain(buckets[0]), Plain(buckets[1]), Plain(buckets[2]), Plain(buckets[3]), Plain(total))
			}
		}
		sec.Empty = "No Aged Receivables."
		r.Sections = []ReportSection{sec}
	case "Agency Open Items":
		r.Title = "Open Items"
		sec := section("Agency|Bill Account|Policy Number|Amount", 3)
		sec.Empty = "No Agency Open Items."
		r.Sections = []ReportSection{sec}
	case "Cash In Suspense":
		sec := section("Account Reference|Allocation Date|Days Pending|Amount", 2, 3)
		byAcct := map[string][]*SuspenseItem{}
		accts := []string{}
		for _, it := range st.Suspense {
			if it.Amount > 0 {
				if byAcct[it.AccountNumber] == nil {
					accts = append(accts, it.AccountNumber)
				}
				byAcct[it.AccountNumber] = append(byAcct[it.AccountNumber], it)
			}
		}
		sort.Strings(accts)
		for _, a := range accts {
			items := byAcct[a]
			sort.Slice(items, func(i, j int) bool { return items[i].Date < items[j].Date })
			sec.add("group", a, "", "", "")
			var sum int64
			for _, it := range items {
				sum += it.Amount
				sec.add("", "", US(it.Date), strconv.Itoa(max(daysBetween(it.Date, sd), 0)), USD(it.Amount))
			}
			sec.add("subtotal", "", "", "", USD(sum))
		}
		sec.Empty = "No Cash In Suspense."
		r.Sections = []ReportSection{sec}
	case "Commission Activity":
		sec := section("Agent Number|Bill Account|Policy Number|Premium Amt|Pay Option|Commission Amt", 3, 5)
		sec.Empty = "No Commission Activity"
		r.Sections = []ReportSection{sec}
	case "Daily Cash Journal":
		sec := section("Account|Entry User Id|Payment Method|Amount", 3)
		sec.Heading, sec.HeadingRight = "Batch: N/A", "Description: Operator Entered"
		byMethod := map[string]int64{}
		methods := []string{}
		var total int64
		pays := sortedValues(st.Payments, func(a, b *Payment) bool { return a.CreatedAt < b.CreatedAt })
		for _, p := range pays {
			if p.EntryDate != sd || p.Status != "Active" || p.Unidentified || p.AccountNumber == "" {
				continue
			}
			code := Codes.PaymentMethodCodes[p.Method]
			if _, ok := byMethod[code]; !ok {
				methods = append(methods, code)
			}
			byMethod[code] += p.Amount
			total += p.Amount
			sec.add("", p.AccountNumber, p.CreatedBy, code, Plain(p.Amount))
		}
		sec.Empty = "No payments were entered on " + US(sd) + "."
		if len(sec.Rows) > 0 {
			sec.add("total", "", "", "Batch Total:", Plain(total))
		}
		bym := section("Payment Method|Amount", 1)
		bym.Heading = "Daily Cash Journal by Payment Method"
		for _, m := range methods {
			bym.add("", m, Plain(byMethod[m]))
		}
		if len(methods) > 0 {
			bym.add("total", "Total:", Plain(total))
		}
		r.Sections = []ReportSection{sec, bym}
	case "Disbursement Activity", "Disbursements Pending Authorization":
		sec := section("Account|Request Date|Method|Method Detail|User|Description|Amount|Reason|Payee Changed", 6)
		ds := sortedValues(st.Disbursements, func(a, b *Disbursement) bool { return a.Amount > b.Amount })
		for _, d := range ds {
			pending := d.Status == "Pending Authorization"
			if (name == "Disbursements Pending Authorization") != pending || d.Status == "Rejected" {
				continue
			}
			changed := "No"
			if d.PayeeChanged {
				changed = "Yes"
			}
			sec.add("", d.AccountNumber, US(d.RequestDate), d.Method, d.MethodDetail, d.User, d.Description, Plain(d.Amount), d.Reason, changed)
		}
		sec.Empty = "No " + name + "."
		r.Meta = r.Meta[:1]
		r.Sections = []ReportSection{sec}
	case "Payment Batch":
		r.Title = "Payment Batch Activity Report"
		sec := section("Batch|Batch Total|Batch Status|Description|Bill Account|Policy Ref|Payment Method|Amount|Detail Status", 1, 7)
		for _, b := range sortedValues(st.Batches, func(a, b *Batch) bool { return a.Number < b.Number }) {
			if b.EntryDate == sd {
				sec.add("", b.Number, Plain(b.BatchTotal), b.Status, b.Description, "", "", b.DefaultMethod, Plain(b.BatchTotal), "")
			}
		}
		sec.Empty = "No Payment Batch Activity."
		r.Meta = r.Meta[:1]
		r.Sections = []ReportSection{sec}
	case "Unidentified Cash":
		sec := section("Account|Policy|Other Identifier|Entry Date|Name|Amount", 5)
		sec.Heading, sec.HeadingRight = "Batch: N/A", "Description: Operator Entered"
		ps := sortedValues(st.Payments, func(a, b *Payment) bool {
			ia, ib := a.Identity, b.Identity
			if ia == nil || ib == nil {
				return a.ID < b.ID
			}
			if ia.AccountRef != ib.AccountRef {
				return ia.AccountRef < ib.AccountRef
			}
			return a.EntryDate < b.EntryDate
		})
		for _, p := range ps {
			if !p.Unidentified || p.Status != "Active" || p.Identity == nil {
				continue
			}
			id := p.Identity
			sec.add("", id.AccountRef, id.PolicyRef, id.Other, US(p.EntryDate), strings.TrimSpace(id.First+" "+id.Last), Plain(p.Amount))
		}
		sec.Empty = "No Unidentified Cash."
		r.Sections = []ReportSection{sec}
	}
	return r
}
