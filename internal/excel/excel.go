// Package excel reads the "Policy Data" sheet of a rater workbook. Columns are
// addressed by letter because header text repeats (two "Policy Number"s).
package excel

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Row is one data row: its 1-based sheet row number and raw cell values by column letter.
type Row struct {
	Number int               `json:"row"`
	Cells  map[string]string `json:"cells"`
}

// Sheet is the header row and the data rows.
type Sheet struct {
	Headers  map[string]string `json:"headers"` // column letter → header text
	Columns  []string          `json:"columns"` // letters with a non-blank header, in sheet order
	Rows     []Row             `json:"-"`
	Date1904 bool              `json:"-"`
}

// Read opens a workbook and reads the named sheet. Reading stops at the first
// row whose keyColumn is empty, so sheets padded to their maximum row are safe.
// Cell values are raw (numbers unformatted, dates as serial numbers).
func Read(r io.Reader, sheet, keyColumn string) (*Sheet, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("not a readable Excel workbook: %w", err)
	}
	defer f.Close()
	found := ""
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(name), sheet) {
			found = name
			break
		}
	}
	if found == "" {
		return nil, fmt.Errorf("workbook has no %q sheet (sheets: %s)", sheet, strings.Join(f.GetSheetList(), ", "))
	}
	rows, err := f.GetRows(found, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %q is empty", sheet)
	}
	s := &Sheet{Headers: map[string]string{}}
	if props, err := f.GetWorkbookProps(); err == nil && props.Date1904 != nil && *props.Date1904 {
		s.Date1904 = true
	}
	for i, h := range rows[0] {
		if h = strings.TrimSpace(h); h != "" {
			col, _ := excelize.ColumnNumberToName(i + 1)
			s.Headers[col] = h
			s.Columns = append(s.Columns, col)
		}
	}
	keyIdx, err := excelize.ColumnNameToNumber(keyColumn)
	if err != nil {
		return nil, err
	}
	for i, cells := range rows[1:] {
		if keyIdx > len(cells) || strings.TrimSpace(cells[keyIdx-1]) == "" {
			break
		}
		row := Row{Number: i + 2, Cells: make(map[string]string, len(cells))}
		for j, v := range cells {
			if v = strings.TrimSpace(v); v != "" {
				col, _ := excelize.ColumnNumberToName(j + 1)
				row.Cells[col] = v
			}
		}
		s.Rows = append(s.Rows, row)
	}
	return s, nil
}
