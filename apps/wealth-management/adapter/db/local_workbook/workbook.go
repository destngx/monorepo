package local_workbook

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

// The workbook is read with the standard library: an .xlsx is a zip of XML
// parts (ECMA-376). Only what the sheet contract needs is decoded: tab
// names, shared strings and cell values; styles, formulas and charts are
// ignored. A formula cell yields the value cached when the file was saved.

const workbookPart = "xl/workbook.xml"

type xmlWorkbook struct {
	Sheets []struct {
		Name string `xml:"name,attr"`
		// RelID is r:id; any namespace matches, so strict OOXML files work too.
		RelID string `xml:"id,attr"`
	} `xml:"sheets>sheet"`
}

type xmlRelationships struct {
	Relationships []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

// xmlRichText is a shared or inline string: plain <t>, or runs <r><t> whose
// texts concatenate. Phonetic hints (<rPh>) are not part of the value.
type xmlRichText struct {
	T *string `xml:"t"`
	R []struct {
		T string `xml:"t"`
	} `xml:"r"`
}

func (r xmlRichText) text() string {
	if r.T != nil {
		return *r.T
	}
	var text strings.Builder
	for _, run := range r.R {
		text.WriteString(run.T)
	}
	return text.String()
}

type xmlSharedStrings struct {
	Items []xmlRichText `xml:"si"`
}

type xmlWorksheet struct {
	Rows []struct {
		R     int `xml:"r,attr"`
		Cells []struct {
			R      string       `xml:"r,attr"`
			T      string       `xml:"t,attr"`
			V      *string      `xml:"v"`
			Inline *xmlRichText `xml:"is"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

// readWorkbook loads every tab of the .xlsx at filePath in workbook order.
func readWorkbook(filePath string) ([]*sheet, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer archive.Close()
	parts := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		parts[file.Name] = file
	}

	var workbook xmlWorkbook
	if err := decodePart(parts, workbookPart, &workbook); err != nil {
		return nil, err
	}
	var relationships xmlRelationships
	if err := decodePart(parts, "xl/_rels/workbook.xml.rels", &relationships); err != nil {
		return nil, err
	}
	targets := make(map[string]string, len(relationships.Relationships))
	for _, relationship := range relationships.Relationships {
		targets[relationship.ID] = partPath(relationship.Target)
	}
	var shared xmlSharedStrings
	if _, ok := parts["xl/sharedStrings.xml"]; ok {
		if err := decodePart(parts, "xl/sharedStrings.xml", &shared); err != nil {
			return nil, err
		}
	}
	sharedTexts := make([]string, len(shared.Items))
	for i, item := range shared.Items {
		sharedTexts[i] = item.text()
	}

	sheets := make([]*sheet, 0, len(workbook.Sheets))
	for _, entry := range workbook.Sheets {
		target, ok := targets[entry.RelID]
		if !ok {
			return nil, fmt.Errorf("workbook tab %q has no part (relationship %q)", entry.Name, entry.RelID)
		}
		var worksheet xmlWorksheet
		if err := decodePart(parts, target, &worksheet); err != nil {
			return nil, fmt.Errorf("tab %q: %w", entry.Name, err)
		}
		loaded, err := loadSheet(entry.Name, worksheet, sharedTexts)
		if err != nil {
			return nil, fmt.Errorf("tab %q: %w", entry.Name, err)
		}
		sheets = append(sheets, loaded)
	}
	return sheets, nil
}

// partPath resolves a relationship target, relative to xl/ unless absolute.
func partPath(target string) string {
	if absolute, ok := strings.CutPrefix(target, "/"); ok {
		return absolute
	}
	return path.Join("xl", target)
}

func decodePart(parts map[string]*zip.File, name string, target any) error {
	file, ok := parts[name]
	if !ok {
		return fmt.Errorf("workbook part %s is missing; is this an .xlsx file?", name)
	}
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer reader.Close()
	if err := xml.NewDecoder(reader).Decode(target); err != nil && err != io.EOF {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	return nil
}

// loadSheet places each cell by its reference, or after the previous row or
// cell when the reference is omitted (both are optional in the format).
func loadSheet(title string, worksheet xmlWorksheet, shared []string) (*sheet, error) {
	loaded := &sheet{title: title}
	rowNumber := 0
	for _, row := range worksheet.Rows {
		rowNumber++
		if row.R > 0 {
			rowNumber = row.R
		}
		colNumber := 0
		for _, xmlCell := range row.Cells {
			colNumber++
			if xmlCell.R != "" {
				col, _, err := parseCellRef(xmlCell.R, xmlCell.R)
				if err != nil {
					return nil, err
				}
				if col == 0 {
					return nil, fmt.Errorf("cell reference %q has no column", xmlCell.R)
				}
				colNumber = col
			}
			value, err := decodeCell(xmlCell.T, xmlCell.V, xmlCell.Inline, shared)
			if err != nil {
				return nil, fmt.Errorf("cell %s: %w", cellName(colNumber, rowNumber), err)
			}
			loaded.set(rowNumber, colNumber, value)
		}
	}
	return loaded, nil
}

// decodeCell types a stored value; t is the cell type attribute.
func decodeCell(t string, v *string, inline *xmlRichText, shared []string) (cell, error) {
	raw := ""
	if v != nil {
		raw = *v
	}
	switch t {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || index < 0 || index >= len(shared) {
			return cell{}, fmt.Errorf("shared string %q out of range", raw)
		}
		return textCell(shared[index]), nil
	case "inlineStr":
		if inline == nil {
			return cell{}, nil
		}
		return textCell(inline.text()), nil
	case "b":
		return cell{kind: cellBool, boolean: strings.TrimSpace(raw) == "1"}, nil
	case "e":
		return cell{kind: cellError, text: raw}, nil
	case "str", "d":
		return textCell(raw), nil
	default:
		if v == nil {
			// A formula with no cached value, or a styled empty cell.
			return cell{}, nil
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return textCell(raw), nil
		}
		return cell{kind: cellNumber, number: number}, nil
	}
}

// cellName is the A1 name of a cell, for error messages.
func cellName(col int, row int) string {
	letters := ""
	for col > 0 {
		col--
		letters = string(rune('A'+col%26)) + letters
		col /= 26
	}
	return letters + strconv.Itoa(row)
}
