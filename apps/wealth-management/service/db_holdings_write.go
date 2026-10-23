package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Holding writes (WM-FILL-10, WM-FILL-11). The fund and crypto tabs are laid
// out by hand, so columns are found by their headers as the reads find them
// (WM-DATA-7), and a new row goes into the first blank row of the table,
// never below a totals row under it.

const (
	fundsSheet       = "InvestmentFundCertificate"
	cryptoSheet      = "Crypto"
	holdingsLastRow  = 100
	cryptoNewHeaders = "Platform|Token|Amount|Note"
)

// holdingTable is a holdings tab read from A1 with its header located.
type holdingTable struct {
	rows      [][]string
	headerRow int // index into rows
	headers   []string
	columns   holdingColumns
	note      int
}

func readHoldingTable(rows [][]string, section holdingSection) (holdingTable, bool) {
	headerRow := findHeaderRow(rows, section.headerKeywords)
	if headerRow < 0 {
		return holdingTable{rows: rows}, false
	}
	headers := normaliseHeaders(rows[headerRow])
	table := holdingTable{rows: rows, headerRow: headerRow, headers: headers, columns: classifyHoldingColumns(headers, section), note: -1}
	for index, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header), "note") {
			table.note = index
		}
	}
	return table, true
}

// sheetRow is the 1-based sheet row of rows[index].
func (table holdingTable) sheetRow(index int) int { return index + 1 }

// firstBlankRow is the first row below the header with no value at all.
func (table holdingTable) firstBlankRow() int {
	for index := table.headerRow + 1; index < len(table.rows); index++ {
		if blankRow(table.rows[index]) {
			return index
		}
	}
	return len(table.rows)
}

func cell(sheet string, column int, row int) string {
	return fmt.Sprintf("'%s'!%s%d", sheet, columnLetter(column), row)
}

// RecordFundTrade appends the trade to the fund's History cell and adds its
// units to Total Unit in one write, or adds a row for a code not held yet.
func (s *dbService) RecordFundTrade(input domain.FundTradeInput) (domain.FundTradeResult, error) {
	input = input.Normalized()
	date, err := input.Validate(s.today())
	if err != nil {
		return domain.FundTradeResult{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, err := s.client.ReadSheet(fundsSheetRange)
	if err != nil {
		return domain.FundTradeResult{}, fmt.Errorf("read %s: %w", fundsSheet, err)
	}
	table, found := readHoldingTable(rows, fundSection)
	if !found || table.columns.symbol < 0 || table.columns.quantity < 0 || table.columns.history < 0 {
		return domain.FundTradeResult{}, fmt.Errorf("%w: the %s tab needs Index, Total Unit and History headers", domain.ErrValidation, fundsSheet)
	}
	entry := domain.FundHistoryEntry(date, input.Units)
	key := domain.FundKey(input.Code)
	for index := table.headerRow + 1; index < len(rows); index++ {
		row := rows[index]
		if domain.FundKey(cellAt(row, table.columns.symbol)) != key || strings.TrimSpace(cellAt(row, table.columns.symbol)) == "" {
			continue
		}
		held := 0.0
		if units := parseOptionalNumber(cellAt(row, table.columns.quantity)); units != nil {
			held = *units
		}
		total := math.Round((held+input.Units)*100) / 100
		if total < 0 {
			return domain.FundTradeResult{}, fmt.Errorf("%w: selling %s units of %s, but %s are held", domain.ErrValidation,
				formatUnits(-input.Units), input.Code, formatUnits(held))
		}
		history := domain.AppendFundHistory(cellAt(row, table.columns.history), entry)
		sheetRow := table.sheetRow(index)
		err := s.client.UpdateCells([]domain.CellUpdate{
			{Range: cell(fundsSheet, table.columns.quantity, sheetRow), Values: []any{total}},
			{Range: cell(fundsSheet, table.columns.history, sheetRow), Values: []any{history}},
		})
		s.invalidate(cacheKeyInvest)
		if err != nil {
			return domain.FundTradeResult{}, fmt.Errorf("record the %s trade: %w", input.Code, err)
		}
		return domain.FundTradeResult{Code: strings.TrimSpace(cellAt(row, table.columns.symbol)), Units: total, Entry: entry, Row: sheetRow}, nil
	}
	if input.Units < 0 {
		return domain.FundTradeResult{}, fmt.Errorf("%w: %s is not held, so it cannot be sold", domain.ErrValidation, input.Code)
	}
	index := table.firstBlankRow()
	sheetRow := table.sheetRow(index)
	if sheetRow > holdingsLastRow {
		return domain.FundTradeResult{}, fmt.Errorf("%w: the %s tab has no free row above row %d", domain.ErrValidation, fundsSheet, holdingsLastRow)
	}
	err = s.client.UpdateCells([]domain.CellUpdate{
		{Range: cell(fundsSheet, table.columns.symbol, sheetRow), Values: []any{input.Code}},
		{Range: cell(fundsSheet, table.columns.quantity, sheetRow), Values: []any{input.Units}},
		{Range: cell(fundsSheet, table.columns.history, sheetRow), Values: []any{entry}},
	})
	s.invalidate(cacheKeyInvest)
	if err != nil {
		return domain.FundTradeResult{}, fmt.Errorf("add %s: %w", input.Code, err)
	}
	return domain.FundTradeResult{Code: input.Code, Units: input.Units, Entry: entry, Row: sheetRow, Created: true}, nil
}

func formatUnits(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// SetCryptoHoldings sets each coin's quantity on its platform in one write,
// adding rows for coins the tab does not list. A tab without a holdings
// header gets one below its last row first (WM-FILL-11).
func (s *dbService) SetCryptoHoldings(input domain.CryptoHoldingsInput) ([]domain.CryptoHoldingResult, error) {
	holdings, err := input.Normalized()
	if err != nil {
		return nil, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	exists, err := s.hasSheet(cryptoSheet)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := s.client.CreateSheet(cryptoSheet, toAny(strings.Split(cryptoNewHeaders, "|"))); err != nil {
			return nil, fmt.Errorf("create the %s tab: %w", cryptoSheet, err)
		}
	}
	rows, err := s.client.ReadSheet(cryptoSheetRange)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cryptoSheet, err)
	}
	var updates []domain.CellUpdate
	table, found := readHoldingTable(rows, cryptoSection)
	if !found {
		// Below the tab's last row, after one blank row.
		headerRow := len(rows) + 1
		if len(rows) == 0 {
			headerRow = 0
		}
		header := strings.Split(cryptoNewHeaders, "|")
		updates = append(updates, domain.CellUpdate{Range: cell(cryptoSheet, 0, headerRow+1), Values: toAny(header)})
		for len(rows) <= headerRow {
			rows = append(rows, nil)
		}
		rows[headerRow] = header
		table, _ = readHoldingTable(rows, cryptoSection)
	}
	if table.columns.symbol < 0 || table.columns.quantity < 0 {
		return nil, fmt.Errorf("%w: the %s tab needs a Token and an Amount header", domain.ErrValidation, cryptoSheet)
	}

	// Rows carry the platform down from the last row that names one.
	located := map[string]int{}
	platform := ""
	for index := table.headerRow + 1; index < len(rows); index++ {
		if name := strings.TrimSpace(cellAt(rows[index], 0)); name != "" {
			platform = name
		}
		if symbol := strings.ToUpper(strings.TrimSpace(cellAt(rows[index], table.columns.symbol))); symbol != "" {
			located[domain.FoldText(platform)+"|"+symbol] = index
		}
	}
	results := make([]domain.CryptoHoldingResult, 0, len(holdings))
	next := table.firstBlankRow()
	for _, holding := range holdings {
		result := domain.CryptoHoldingResult{Platform: holding.Platform, Symbol: holding.Symbol, Quantity: holding.Quantity}
		index, known := located[domain.FoldText(holding.Platform)+"|"+holding.Symbol]
		if !known {
			index = next
			next++
			result.Created = true
			if table.sheetRow(index) > holdingsLastRow {
				return nil, fmt.Errorf("%w: the %s tab has no free row above row %d", domain.ErrValidation, cryptoSheet, holdingsLastRow)
			}
			updates = append(updates,
				domain.CellUpdate{Range: cell(cryptoSheet, 0, table.sheetRow(index)), Values: []any{holding.Platform}},
				domain.CellUpdate{Range: cell(cryptoSheet, table.columns.symbol, table.sheetRow(index)), Values: []any{holding.Symbol}},
			)
		}
		result.Row = table.sheetRow(index)
		updates = append(updates, domain.CellUpdate{Range: cell(cryptoSheet, table.columns.quantity, result.Row), Values: []any{holding.Quantity}})
		if table.note >= 0 && holding.Note != "" {
			updates = append(updates, domain.CellUpdate{Range: cell(cryptoSheet, table.note, result.Row), Values: []any{holding.Note}})
		}
		results = append(results, result)
	}
	err = s.client.UpdateCells(updates)
	s.invalidate(cacheKeyInvest)
	if err != nil {
		return nil, fmt.Errorf("write crypto holdings: %w", err)
	}
	return results, nil
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for index, value := range values {
		out[index] = value
	}
	return out
}
