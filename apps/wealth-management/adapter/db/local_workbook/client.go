// Package local_workbook serves the sheet contract (port.DatabaseClient)
// from a local .xlsx export of the workbook instead of Google Sheets. It
// backs the live e2e harness (WM-TEST-1): the file is read once into memory
// and never written, so every engine start sees the same data and a run
// never reaches Google.
//
// Reads mirror the Sheets API with UNFORMATTED_VALUE: numbers and date
// serials come back as numbers, booleans as true/false, formulas as the
// value cached in the file. Writes mirror USER_ENTERED for what the engine
// writes: numeric text becomes a number, a dd/MM/yyyy date becomes a date
// serial, a leading apostrophe keeps text. Formulas are not evaluated, so a
// written formula reads back empty and cells that depend on written rows
// (balances, totals) keep their exported values.
package local_workbook

import (
	"apps/wealth-management/domain"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client struct {
	title string
	// mu guards sheets; first-empty-row writes read then write under it.
	mu     sync.Mutex
	sheets []*sheet
}

// Open loads the workbook at path; its title is the file name.
func Open(path string) (*Client, error) {
	sheets, err := readWorkbook(path)
	if err != nil {
		return nil, err
	}
	return &Client{title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), sheets: sheets}, nil
}

func (c *Client) ReadSheet(rangeValue string) ([][]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.read(rangeValue)
}

func (c *Client) ReadSheets(ranges []string) ([][][]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([][][]string, 0, len(ranges))
	for _, rangeValue := range ranges {
		rows, err := c.read(rangeValue)
		if err != nil {
			return nil, err
		}
		result = append(result, rows)
	}
	return result, nil
}

// UpdateRow writes values to the right of the range's first cell.
func (c *Client) UpdateRow(rangeValue string, values []any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	target, err := parseRange(rangeValue)
	if err != nil {
		return err
	}
	tab, err := c.sheet(target.sheet)
	if err != nil {
		return err
	}
	tab.writeRow(target.startRow, target.startCol, values)
	return nil
}

// WriteRowsAtFirstEmptyRow writes rows from column A below the last
// non-empty cell of the anchor column, like the Sheets client.
func (c *Client) WriteRowsAtFirstEmptyRow(sheetName string, anchorRange string, rows [][]any) (int, error) {
	if len(rows) == 0 {
		return 0, fmt.Errorf("no rows to write")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	anchor, err := parseRange(anchorRange)
	if err != nil {
		return 0, err
	}
	existing, err := c.read(anchorRange)
	if err != nil {
		return 0, err
	}
	lastDataIdx := -1
	for i, row := range existing {
		if len(row) > 0 && row[0] != "" {
			lastDataIdx = i
		}
	}
	tab, err := c.sheet(sheetName)
	if err != nil {
		return 0, err
	}
	targetRow := anchor.startRow + lastDataIdx + 1
	for offset, values := range rows {
		tab.writeRow(targetRow+offset, 1, values)
	}
	return targetRow, nil
}

func (c *Client) SheetTitles() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	titles := make([]string, 0, len(c.sheets))
	for _, tab := range c.sheets {
		titles = append(titles, tab.title)
	}
	return titles, nil
}

func (c *Client) SpreadsheetTitle() (string, error) {
	return c.title, nil
}

// CreateSheet adds a tab with a header row; tab names are unique ignoring
// case, as in Sheets.
func (c *Client) CreateSheet(title string, header []any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.sheet(title); err == nil {
		return fmt.Errorf("a tab named %q already exists", title)
	}
	tab := &sheet{title: title}
	tab.writeRow(1, 1, header)
	c.sheets = append(c.sheets, tab)
	return nil
}

func (c *Client) read(rangeValue string) ([][]string, error) {
	target, err := parseRange(rangeValue)
	if err != nil {
		return nil, err
	}
	tab, err := c.sheet(target.sheet)
	if err != nil {
		return nil, err
	}
	return tab.read(target), nil
}

// sheet finds a tab by name, ignoring case like A1 ranges in Sheets, then
// ignoring the characters an .xlsx tab name cannot hold: the export drops
// them, so the sheet's "Loan / Dept" is the file's "Loan  Dept".
func (c *Client) sheet(name string) (*sheet, error) {
	for _, matches := range []func(title string) bool{
		func(title string) bool { return title == name },
		func(title string) bool { return strings.EqualFold(title, name) },
		func(title string) bool { return strings.EqualFold(xlsxTabName(title), xlsxTabName(name)) },
	} {
		for _, tab := range c.sheets {
			if matches(tab.title) {
				return tab, nil
			}
		}
	}
	return nil, fmt.Errorf("the workbook has no tab named %q", name)
}

// xlsxTabName drops the characters Excel forbids in tab names.
func xlsxTabName(name string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/?*[]:`, r) {
			return -1
		}
		return r
	}, name)
}

// sheet holds a tab's cells; rows[r-1][c-1] is the cell at row r, column c.
type sheet struct {
	title string
	rows  [][]cell
}

func (s *sheet) set(row int, col int, value cell) {
	for len(s.rows) < row {
		s.rows = append(s.rows, nil)
	}
	cells := s.rows[row-1]
	for len(cells) < col {
		cells = append(cells, cell{})
	}
	cells[col-1] = value
	s.rows[row-1] = cells
}

func (s *sheet) writeRow(row int, col int, values []any) {
	for offset, value := range values {
		s.set(row, col+offset, userEntered(value))
	}
}

// read returns the range's rendered cells without trailing empty cells or
// rows, as the Sheets API does.
func (s *sheet) read(target cellRange) [][]string {
	lastRow := len(s.rows)
	if target.endRow > 0 {
		lastRow = min(lastRow, target.endRow)
	}
	rows := [][]string{}
	for r := target.startRow; r <= lastRow; r++ {
		cells := s.rows[r-1]
		lastCol := len(cells)
		if target.endCol > 0 {
			lastCol = min(lastCol, target.endCol)
		}
		rendered := []string{}
		for c := target.startCol; c <= lastCol; c++ {
			rendered = append(rendered, cells[c-1].render())
		}
		for len(rendered) > 0 && rendered[len(rendered)-1] == "" {
			rendered = rendered[:len(rendered)-1]
		}
		rows = append(rows, rendered)
	}
	for len(rows) > 0 && len(rows[len(rows)-1]) == 0 {
		rows = rows[:len(rows)-1]
	}
	return rows
}

type cellKind uint8

const (
	cellEmpty cellKind = iota
	cellText
	cellNumber
	cellBool
	cellError
)

type cell struct {
	kind    cellKind
	text    string
	number  float64
	boolean bool
}

func textCell(text string) cell {
	if text == "" {
		return cell{}
	}
	return cell{kind: cellText, text: text}
}

// render formats the cell as the Sheets client turns an UNFORMATTED_VALUE
// into a string (fmt.Sprint of the decoded JSON value).
func (c cell) render() string {
	switch c.kind {
	case cellNumber:
		return fmt.Sprint(c.number)
	case cellBool:
		return fmt.Sprint(c.boolean)
	case cellText, cellError:
		return c.text
	default:
		return ""
	}
}

// userEntered types a written value the way USER_ENTERED does for the
// values the engine writes; see the package comment.
func userEntered(value any) cell {
	switch typed := value.(type) {
	case nil:
		return cell{}
	case bool:
		return cell{kind: cellBool, boolean: typed}
	case float64:
		return cell{kind: cellNumber, number: typed}
	case float32:
		return cell{kind: cellNumber, number: float64(typed)}
	case int:
		return cell{kind: cellNumber, number: float64(typed)}
	case int64:
		return cell{kind: cellNumber, number: float64(typed)}
	case string:
		return enteredText(typed)
	default:
		return enteredText(fmt.Sprint(typed))
	}
}

func enteredText(text string) cell {
	if literal, ok := strings.CutPrefix(text, "'"); ok {
		return textCell(literal)
	}
	if strings.HasPrefix(text, "=") {
		return cell{}
	}
	trimmed := strings.TrimSpace(text)
	if number, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return cell{kind: cellNumber, number: number}
	}
	if date, err := time.Parse(domain.SheetDateLayout, trimmed); err == nil {
		return cell{kind: cellNumber, number: float64(domain.DateToSerial(date))}
	}
	return textCell(text)
}
