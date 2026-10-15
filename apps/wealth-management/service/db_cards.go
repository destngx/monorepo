package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Credit card configuration (WM-CARD-3, WM-CARD-7, WM-CARD-9, WM-CARD-16)
// lives in the sheet, not the repo (N8): Programs holds one row per
// programme version, ProgramRules one row per rule of a version, Banks one
// row per bank, Cards one row per card held and Merchants one row per
// payee-to-MCC observation.

const (
	programsSheet     = "Programs"
	programsRange     = "Programs!A2:F"
	programRulesSheet = "ProgramRules"
	programRulesRange = "ProgramRules!A2:K"
	banksSheet        = "Banks"
	banksRange        = "Banks!A2:C"
	cardsSheet        = "Cards"
	cardsRange        = "Cards!A2:I"
	merchantsSheet    = "Merchants"
	merchantsRange    = "Merchants!A2:E"
	cardsFirstRow     = 2
	cacheKeyCards     = "cards:config"
)

// Column labels, used in error messages so they match the sheet headers.
const (
	colVersionFrom    = "Version from"
	colAnnualFee      = "Annual fee"
	colCardCap        = "Card cap"
	colRuleCap        = "Rule cap"
	colRate           = "Rate %"
	colMCCs           = "MCCs"
	colMinTransaction = "Min transaction"
	colMinCycleSpend  = "Min cycle spend"
	colReferenceLimit = "Reference limit"
	colStatementDay   = "Statement day"
	colDueDay         = "Due day"
	colMerchantFrom   = "From"
	colClosedFrom     = "Closed from"
	colCapGroup       = "Cap group"
)

// cardTabs are the tabs read, in batch order.
var cardTabs = []struct{ sheet, rng string }{
	{programsSheet, programsRange},
	{programRulesSheet, programRulesRange},
	{banksSheet, banksRange},
	{cardsSheet, cardsRange},
	{merchantsSheet, merchantsRange},
}

// ListCardConfig reads the card configuration from the tabs that exist; no
// tab means no rows. Invalid rows fail the whole read with ErrValidation
// naming each row, so cashback is never computed from a half-read
// configuration.
func (s *dbService) ListCardConfig(forceFresh bool) (domain.CardConfig, error) {
	return cachedRead(s, cacheKeyCards, sheetDataTTL, forceFresh, func() (domain.CardConfig, error) {
		titles, err := s.sheetTitles()
		if err != nil {
			return domain.CardConfig{}, err
		}
		ranges := []string{}
		for _, tab := range cardTabs {
			if titles[tab.sheet] {
				ranges = append(ranges, tab.rng)
			}
		}
		byRange := map[string][][]string{}
		if len(ranges) > 0 {
			rows, err := s.readRanges(ranges...)
			if err != nil {
				return domain.CardConfig{}, err
			}
			for i, rng := range ranges {
				byRange[rng] = rows[i]
			}
		}
		return mapCardConfig(cardSheetRows{
			programs: byRange[programsRange], rules: byRange[programRulesRange], banks: byRange[banksRange],
			cards: byRange[cardsRange], merchants: byRange[merchantsRange],
		})
	})
}

type cardSheetRows struct {
	programs, rules, banks, cards, merchants [][]string
}

type cardRowProblems []string

func (p *cardRowProblems) add(tab string, rowIndex int, problem string) {
	*p = append(*p, fmt.Sprintf("%s row %d: %s", tab, cardsFirstRow+rowIndex, problem))
}

func (p *cardRowProblems) addAll(tab string, rowIndex int, problems []string) {
	for _, problem := range problems {
		p.add(tab, rowIndex, problem)
	}
}

func mapCardConfig(rows cardSheetRows) (domain.CardConfig, error) {
	var problems cardRowProblems
	programs := mapPrograms(rows.programs, rows.rules, &problems)
	banks := mapBanks(rows.banks, &problems)
	cards := mapCards(rows.cards, programs, banks, &problems)
	merchants := mapMerchants(rows.merchants, &problems)
	if len(problems) > 0 {
		return domain.CardConfig{}, fmt.Errorf("%w: %s", domain.ErrValidation, strings.Join(problems, "; "))
	}
	return domain.CardConfig{Programs: programs, Banks: banks, Cards: cards, Merchants: merchants}, nil
}

func versionKey(program string, from time.Time) string {
	return domain.FoldText(program) + "|" + from.Format(time.DateOnly)
}

func mapPrograms(programRows, ruleRows [][]string, problems *cardRowProblems) []domain.Program {
	programs := []domain.Program{}
	programAt := map[string]int{}
	versionRow := map[string]int{}
	// version locates a version by programme and date: programme index, version index.
	type versionRef struct{ program, version int }
	versionAt := map[string]versionRef{}
	for index, row := range programRows {
		if blankRow(row) {
			continue
		}
		parse := cellParser{row: row}
		name := strings.TrimSpace(cellAt(row, 0))
		bank := strings.TrimSpace(cellAt(row, 1))
		version := domain.ProgramVersion{
			From:      parse.date(2, colVersionFrom),
			AnnualFee: parse.number(3, colAnnualFee),
			Cap:       parse.number(4, colCardCap),
			Period:    domain.CapPeriod(strings.ToLower(strings.TrimSpace(cellAt(row, 5)))),
			Rules:     []domain.CashbackRule{},
		}
		rowProblems := parse.problems
		if name == "" {
			rowProblems = append(rowProblems, "programme is required")
		}
		if bank == "" {
			rowProblems = append(rowProblems, "bank is required")
		}
		if !parse.failed {
			rowProblems = append(rowProblems, version.Problems()...)
		}
		position, known := programAt[domain.FoldText(name)]
		if known && name != "" && domain.FoldText(programs[position].Bank) != domain.FoldText(bank) {
			rowProblems = append(rowProblems, fmt.Sprintf("bank %q differs from %q on an earlier row of programme %q", bank, programs[position].Bank, name))
		}
		key := versionKey(name, version.From)
		if previous, seen := versionRow[key]; seen && !parse.failed {
			rowProblems = append(rowProblems, fmt.Sprintf("programme %q already has a version from this date on row %d", name, cardsFirstRow+previous))
		}
		if len(rowProblems) > 0 {
			problems.addAll(programsSheet, index, rowProblems)
			continue
		}
		if !known {
			position = len(programs)
			programAt[domain.FoldText(name)] = position
			programs = append(programs, domain.Program{Name: name, Bank: bank, Versions: []domain.ProgramVersion{}})
		}
		versionRow[key] = index
		versionAt[key] = versionRef{program: position, version: len(programs[position].Versions)}
		programs[position].Versions = append(programs[position].Versions, version)
	}

	// Rules are told apart by name within a version (caps are per rule), and
	// two fallbacks with the same minimums would leave the second unreachable.
	// A cap group shares one cap, so its rules state the same one, and its
	// name is not a rule's, so standings stay told apart.
	ruleRow := map[string]int{}
	fallbackRow := map[string]int{}
	type capGroupRow struct {
		row int
		cap float64
	}
	groupRow := map[string]capGroupRow{}
	for index, row := range ruleRows {
		if blankRow(row) {
			continue
		}
		parse := cellParser{row: row}
		program := strings.TrimSpace(cellAt(row, 0))
		from := parse.date(1, colVersionFrom)
		rule := domain.CashbackRule{
			Name:           strings.TrimSpace(cellAt(row, 2)),
			MCCs:           parse.mccs(3, colMCCs),
			Rate:           parse.number(4, colRate),
			Cap:            parse.number(5, colRuleCap),
			MinTransaction: parse.number(6, colMinTransaction),
			MinCycleSpend:  parse.number(7, colMinCycleSpend),
			Fallback:       parseCleared(cellAt(row, 8)),
			Exclude:        parseCleared(cellAt(row, 9)),
			CapGroup:       strings.TrimSpace(cellAt(row, 10)),
		}
		if rule.MCCs == nil {
			rule.MCCs = []domain.MCCRange{}
		}
		rowProblems := parse.problems
		if !parse.failed {
			rowProblems = append(rowProblems, rule.Problems()...)
		}
		key := versionKey(program, from)
		ref, found := versionAt[key]
		if !found && !parse.failed {
			rowProblems = append(rowProblems, fmt.Sprintf("programme %q from %s matches no row in %s", program, from.Format(time.DateOnly), programsSheet))
		}
		nameKey := key + "|" + domain.FoldText(rule.Name)
		if previous, seen := ruleRow[nameKey]; found && seen && rule.Name != "" {
			rowProblems = append(rowProblems, fmt.Sprintf("this version already has a rule named %q on row %d", rule.Name, cardsFirstRow+previous))
		}
		fallbackKey := fmt.Sprintf("%s|%v|%v", key, rule.MinTransaction, rule.MinCycleSpend)
		if previous, seen := fallbackRow[fallbackKey]; found && seen && rule.Fallback {
			rowProblems = append(rowProblems, fmt.Sprintf("this version already has a fallback rule with the same minimums on row %d", cardsFirstRow+previous))
		}
		groupKey := key + "|" + domain.FoldText(rule.CapGroup)
		if rule.CapGroup != "" && found {
			if previous, seen := groupRow[groupKey]; seen && previous.cap != rule.Cap {
				rowProblems = append(rowProblems, fmt.Sprintf("cap group %q has a different rule cap on row %d: its rules share one cap", rule.CapGroup, cardsFirstRow+previous.row))
			}
			if previous, seen := ruleRow[groupKey]; seen || domain.FoldText(rule.CapGroup) == domain.FoldText(rule.Name) {
				if !seen {
					previous = index
				}
				rowProblems = append(rowProblems, fmt.Sprintf("cap group %q has the same name as the rule on row %d", rule.CapGroup, cardsFirstRow+previous))
			}
		}
		if previous, seen := groupRow[nameKey]; found && seen {
			rowProblems = append(rowProblems, fmt.Sprintf("rule %q has the same name as the cap group on row %d", rule.Name, cardsFirstRow+previous.row))
		}
		if len(rowProblems) > 0 {
			problems.addAll(programRulesSheet, index, rowProblems)
			continue
		}
		ruleRow[nameKey] = index
		if _, seen := groupRow[groupKey]; rule.CapGroup != "" && !seen {
			groupRow[groupKey] = capGroupRow{row: index, cap: rule.Cap}
		}
		if rule.Fallback {
			fallbackRow[fallbackKey] = index
		}
		version := &programs[ref.program].Versions[ref.version]
		version.Rules = append(version.Rules, rule)
	}
	for i := range programs {
		sort.SliceStable(programs[i].Versions, func(a, b int) bool {
			return programs[i].Versions[a].From.Before(programs[i].Versions[b].From)
		})
	}
	return programs
}

func mapBanks(rows [][]string, problems *cardRowProblems) []domain.Bank {
	banks := []domain.Bank{}
	bankRow := map[string]int{}
	for index, row := range rows {
		if blankRow(row) {
			continue
		}
		parse := cellParser{row: row}
		bank := domain.Bank{
			Name:           strings.TrimSpace(cellAt(row, 0)),
			SharedAccount:  strings.TrimSpace(cellAt(row, 1)),
			ReferenceLimit: parse.number(2, colReferenceLimit),
		}
		rowProblems := parse.problems
		if !parse.failed {
			rowProblems = append(rowProblems, bank.Problems()...)
		}
		if previous, seen := bankRow[domain.FoldText(bank.Name)]; seen && bank.Name != "" {
			rowProblems = append(rowProblems, fmt.Sprintf("bank %q is already on row %d", bank.Name, cardsFirstRow+previous))
		}
		if len(rowProblems) > 0 {
			problems.addAll(banksSheet, index, rowProblems)
			continue
		}
		bankRow[domain.FoldText(bank.Name)] = index
		banks = append(banks, bank)
	}
	return banks
}

func mapCards(rows [][]string, programs []domain.Program, banks []domain.Bank, problems *cardRowProblems) []domain.Card {
	cards := []domain.Card{}
	programBank := map[string]domain.Program{}
	for _, program := range programs {
		programBank[domain.FoldText(program.Name)] = program
	}
	bankNames := map[string]bool{}
	for _, bank := range banks {
		bankNames[domain.FoldText(bank.Name)] = true
	}
	nameRow, keywordRow, defaultRow := map[string]int{}, map[string]int{}, map[string]int{}
	for index, row := range rows {
		if blankRow(row) {
			continue
		}
		parse := cellParser{row: row}
		card := domain.Card{
			Name:         strings.TrimSpace(cellAt(row, 0)),
			Program:      strings.TrimSpace(cellAt(row, 1)),
			Keyword:      strings.TrimSpace(cellAt(row, 2)),
			Expiry:       formatCardExpiry(cellAt(row, 3)),
			StatementDay: parse.day(4, colStatementDay),
			DueDay:       parse.day(5, colDueDay),
			Default:      parseCleared(cellAt(row, 6)),
			Last4:        formatCardLast4(cellAt(row, 8)),
		}
		if closed := strings.TrimSpace(cellAt(row, 7)); closed != "" {
			date := parse.date(7, colClosedFrom)
			if !parse.failed {
				card.ClosedFrom = &date
			}
		}
		rowProblems := parse.problems
		if !parse.failed {
			rowProblems = append(rowProblems, card.Problems()...)
		}
		program, known := programBank[domain.FoldText(card.Program)]
		switch {
		case card.Program == "":
		case !known:
			rowProblems = append(rowProblems, fmt.Sprintf("programme %q matches no row in %s", card.Program, programsSheet))
		case !bankNames[domain.FoldText(program.Bank)]:
			rowProblems = append(rowProblems, fmt.Sprintf("bank %q of programme %q has no row in %s", program.Bank, program.Name, banksSheet))
		}
		if previous, seen := nameRow[domain.FoldText(card.Name)]; seen && card.Name != "" {
			rowProblems = append(rowProblems, fmt.Sprintf("card name %q is already used on row %d", card.Name, cardsFirstRow+previous))
		}
		if previous, seen := keywordRow[domain.FoldText(card.Keyword)]; seen && card.Keyword != "" {
			rowProblems = append(rowProblems, fmt.Sprintf("keyword %q is already used on row %d", card.Keyword, cardsFirstRow+previous))
		}
		bankKey := domain.FoldText(program.Bank)
		if previous, seen := defaultRow[bankKey]; seen && known && card.Default {
			rowProblems = append(rowProblems, fmt.Sprintf("bank %q already has its default card on row %d", program.Bank, cardsFirstRow+previous))
		}
		if len(rowProblems) > 0 {
			problems.addAll(cardsSheet, index, rowProblems)
			continue
		}
		card.Program = program.Name
		nameRow[domain.FoldText(card.Name)] = index
		keywordRow[domain.FoldText(card.Keyword)] = index
		if card.Default {
			defaultRow[bankKey] = index
		}
		cards = append(cards, card)
	}
	return cards
}

func mapMerchants(rows [][]string, problems *cardRowProblems) []domain.MerchantMCC {
	merchants := []domain.MerchantMCC{}
	merchantRow := map[string]int{}
	for index, row := range rows {
		if blankRow(row) {
			continue
		}
		pattern := strings.TrimSpace(cellAt(row, 0))
		mccText := strings.TrimSpace(cellAt(row, 1))
		fromText := strings.TrimSpace(cellAt(row, 2))
		confidenceText := strings.ToLower(strings.TrimSpace(cellAt(row, 3)))
		var rowProblems []string
		var from *time.Time
		if fromText != "" {
			date, err := domain.ParseSheetDate(fromText)
			if err != nil {
				rowProblems = append(rowProblems, fmt.Sprintf("%s %q is not a date", colMerchantFrom, fromText))
			} else {
				from = &date
			}
		}
		if domain.FoldText(pattern) == "" {
			rowProblems = append(rowProblems, "payee pattern is required")
		}
		codes, err := domain.ParseMCCList(mccText)
		switch {
		case err != nil:
			rowProblems = append(rowProblems, fmt.Sprintf("MCC: %v", err))
		case len(codes) != 1 || codes[0].From != codes[0].To:
			rowProblems = append(rowProblems, fmt.Sprintf("MCC %q must be exactly one MCC", mccText))
		}
		confidence := domain.MCCAssumed
		switch confidenceText {
		case "", string(domain.MCCAssumed):
		case string(domain.MCCConfirmed):
			confidence = domain.MCCConfirmed
		default:
			rowProblems = append(rowProblems, fmt.Sprintf("confidence must be %q or %q, got %q", domain.MCCAssumed, domain.MCCConfirmed, confidenceText))
		}
		// A pattern may be mapped once per start date: a later row records a
		// change of MCC, the same date twice is ambiguous.
		mappingKey := domain.FoldText(pattern)
		if from != nil {
			mappingKey += "|" + from.Format(time.DateOnly)
		}
		if previous, seen := merchantRow[mappingKey]; seen && pattern != "" {
			rowProblems = append(rowProblems, fmt.Sprintf("payee pattern %q is already mapped from the same date on row %d", pattern, cardsFirstRow+previous))
		}
		if len(rowProblems) > 0 {
			problems.addAll(merchantsSheet, index, rowProblems)
			continue
		}
		merchantRow[mappingKey] = index
		merchants = append(merchants, domain.MerchantMCC{
			Pattern: pattern, MCC: codes[0].From, From: from, Confidence: confidence, Note: strings.TrimSpace(cellAt(row, 4)),
		})
	}
	return merchants
}

func blankRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// formatCardExpiry keeps a text expiry as written; a cell Sheets parsed as
// a date (e.g. 08/2029) arrives as a serial and becomes MM/YYYY.
func formatCardExpiry(value string) string {
	trimmed := strings.TrimSpace(value)
	if serial, err := strconv.ParseFloat(trimmed, 64); err == nil && serial > 1000 {
		date := domain.SerialToDate(serial)
		return fmt.Sprintf("%02d/%d", int(date.Month()), date.Year())
	}
	return trimmed
}

// formatCardLast4 restores leading zeros a number cell drops (0123 reads
// as 123); anything else is left for validation.
func formatCardLast4(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 4 && strings.Trim(trimmed, "0123456789") == "" && trimmed != "" {
		return strings.Repeat("0", 4-len(trimmed)) + trimmed
	}
	return trimmed
}

// cellParser reads strict numbers from a row: an empty cell is 0, anything
// else must parse, and each failure is recorded against its column.
type cellParser struct {
	row      []string
	problems []string
	failed   bool
}

func (p *cellParser) number(index int, column string) float64 {
	raw := strings.TrimSpace(cellAt(p.row, index))
	if raw == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", ""), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		p.fail(fmt.Sprintf("%s %q is not a number", column, raw))
		return 0
	}
	return parsed
}

func (p *cellParser) day(index int, column string) int {
	raw := strings.TrimSpace(cellAt(p.row, index))
	value := p.number(index, column)
	if raw != "" && value != math.Trunc(value) {
		p.fail(fmt.Sprintf("%s %q must be a whole day of the month", column, raw))
		return 0
	}
	return int(value)
}

func (p *cellParser) mccs(index int, column string) []domain.MCCRange {
	ranges, err := domain.ParseMCCList(cellAt(p.row, index))
	if err != nil {
		p.fail(fmt.Sprintf("%s: %v", column, err))
		return nil
	}
	return ranges
}

// date reads a required date cell: a serial or a written date.
func (p *cellParser) date(index int, column string) time.Time {
	raw := strings.TrimSpace(cellAt(p.row, index))
	if raw == "" {
		p.fail(column + " is required")
		return time.Time{}
	}
	date, err := domain.ParseSheetDate(raw)
	if err != nil {
		p.fail(fmt.Sprintf("%s %q is not a date", column, raw))
		return time.Time{}
	}
	return date
}

func (p *cellParser) fail(problem string) {
	p.problems = append(p.problems, problem)
	p.failed = true
}
