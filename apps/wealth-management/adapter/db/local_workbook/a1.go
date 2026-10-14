package local_workbook

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// cellRange is a parsed A1 range. Rows and columns are 1-based; an end of 0
// is open (e.g. A2:A runs to the last row, A2 alone is one cell).
type cellRange struct {
	sheet    string
	startRow int
	startCol int
	endRow   int
	endCol   int
}

// cellRefPattern is one end of an A1 range: column letters, row digits or both.
var cellRefPattern = regexp.MustCompile(`^([A-Za-z]{0,3})(\d*)$`)

// parseRange reads "Sheet!A1:B2", "'Tab name'!A2:A" or a bare tab name (the
// whole tab), the forms the Sheets API accepts. A start without a column or
// row begins at column A or row 1; an end without one is open.
func parseRange(value string) (cellRange, error) {
	sheet, cells, err := splitSheet(value)
	if err != nil {
		return cellRange{}, err
	}
	if sheet == "" {
		return cellRange{}, fmt.Errorf("range %q has no tab name", value)
	}
	parsed := cellRange{sheet: sheet, startRow: 1, startCol: 1}
	if cells == "" {
		return parsed, nil
	}
	start, end, hasEnd := strings.Cut(cells, ":")
	startCol, startRow, err := parseCellRef(start, value)
	if err != nil {
		return cellRange{}, err
	}
	if !hasEnd {
		if startCol == 0 || startRow == 0 {
			return cellRange{}, fmt.Errorf("range %q: a single cell needs a column and a row", value)
		}
		return cellRange{sheet: sheet, startRow: startRow, startCol: startCol, endRow: startRow, endCol: startCol}, nil
	}
	if parsed.endCol, parsed.endRow, err = parseCellRef(end, value); err != nil {
		return cellRange{}, err
	}
	parsed.startCol, parsed.startRow = max(startCol, 1), max(startRow, 1)
	return parsed, nil
}

// splitSheet separates the tab name from the cells; a quoted name doubles
// its apostrophes.
func splitSheet(value string) (sheet string, cells string, err error) {
	value = strings.TrimSpace(value)
	if rest, quoted := strings.CutPrefix(value, "'"); quoted {
		var name strings.Builder
		for i := 0; i < len(rest); i++ {
			if rest[i] != '\'' {
				name.WriteByte(rest[i])
				continue
			}
			if i+1 < len(rest) && rest[i+1] == '\'' {
				name.WriteByte('\'')
				i++
				continue
			}
			after := rest[i+1:]
			if after == "" {
				return name.String(), "", nil
			}
			cells, ok := strings.CutPrefix(after, "!")
			if !ok {
				return "", "", fmt.Errorf("range %q: expected ! after the quoted tab name", value)
			}
			return name.String(), cells, nil
		}
		return "", "", fmt.Errorf("range %q: unterminated quoted tab name", value)
	}
	index := strings.LastIndex(value, "!")
	if index < 0 {
		return value, "", nil
	}
	return value[:index], value[index+1:], nil
}

// parseCellRef reads one end of a range; a missing column or row is 0.
func parseCellRef(ref string, value string) (col int, row int, err error) {
	match := cellRefPattern.FindStringSubmatch(ref)
	if match == nil || ref == "" || (match[1] == "" && match[2] == "") {
		return 0, 0, fmt.Errorf("range %q: %q is not an A1 reference", value, ref)
	}
	if match[1] != "" {
		col = columnNumber(match[1])
	}
	if match[2] != "" {
		if row, err = strconv.Atoi(match[2]); err != nil || row < 1 {
			return 0, 0, fmt.Errorf("range %q: bad row in %q", value, ref)
		}
	}
	return col, row, nil
}

// columnNumber turns letters into a 1-based column: A is 1, AA is 27.
func columnNumber(letters string) int {
	number := 0
	for _, letter := range strings.ToUpper(letters) {
		number = number*26 + int(letter-'A'+1)
	}
	return number
}
