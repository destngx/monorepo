package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Card configuration writes (WM-FILL-12..14). The tabs are read strictly:
// one bad row fails every cashback figure (WM-CARD-3). So each write builds
// the tabs as they would read after it and runs the same mapping; a write
// that would leave a bad row is refused before anything is written.

var (
	cardsHeader     = []any{"Card", "Programme", "Keyword", "Expiry", "Statement day", "Due day", "Default", "Closed from", "Last 4"}
	programsHeader  = []any{"Programme", "Bank", "Version from", "Annual fee", "Card cap", "Cap period"}
	rulesHeader     = []any{"Programme", "Version from", "Rule", "MCCs", "Rate %", "Rule cap", "Min transaction", "Min cycle spend", "Fallback", "Exclude", "Cap group"}
	banksHeader     = []any{"Bank", "Shared account", "Reference limit"}
	merchantsHeader = []any{"Payee pattern", "MCC", "From", "Confidence", "Source"}
)

// readCardRows reads the card tabs fresh, a missing tab as no rows.
func (s *dbService) readCardRows() (cardSheetRows, map[string]bool, error) {
	titles, err := s.sheetTitles()
	if err != nil {
		return cardSheetRows{}, nil, err
	}
	var ranges []string
	for _, tab := range cardTabs {
		if titles[tab.sheet] {
			ranges = append(ranges, tab.rng)
		}
	}
	byRange := map[string][][]string{}
	if len(ranges) > 0 {
		read, err := s.readRanges(ranges...)
		if err != nil {
			return cardSheetRows{}, nil, err
		}
		for index, rng := range ranges {
			byRange[rng] = read[index]
		}
	}
	return cardSheetRows{
		programs: byRange[programsRange], rules: byRange[programRulesRange], banks: byRange[banksRange],
		cards: byRange[cardsRange], merchants: byRange[merchantsRange],
	}, titles, nil
}

// asRead turns written values into the strings a read returns: a leading
// apostrophe only marks text.
func asRead(values []any) []string {
	row := make([]string, len(values))
	for index, value := range values {
		switch typed := value.(type) {
		case string:
			row[index] = strings.TrimPrefix(typed, "'")
		case float64:
			row[index] = strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			row[index] = strconv.Itoa(typed)
		default:
			row[index] = fmt.Sprint(typed)
		}
	}
	return row
}

func withRow(rows [][]string, index int, row []string) [][]string {
	out := make([][]string, max(len(rows), index+1))
	copy(out, rows)
	out[index] = row
	return out
}

// lastUsedRow is the number of rows up to the last non-blank one.
func lastUsedRow(rows [][]string) int {
	for index := len(rows) - 1; index >= 0; index-- {
		if !blankRow(rows[index]) {
			return index + 1
		}
	}
	return 0
}

func (s *dbService) ensureTab(titles map[string]bool, sheet string, header []any) error {
	if titles[sheet] {
		return nil
	}
	if err := s.client.CreateSheet(sheet, header); err != nil {
		return fmt.Errorf("create the %s tab: %w", sheet, err)
	}
	return nil
}

// CreateCard adds a Cards row (WM-FILL-12).
func (s *dbService) CreateCard(input domain.CardInput) (domain.Card, error) {
	return s.writeCard("", input)
}

// UpdateCard replaces the card's row; ErrNotFound for an unknown name.
func (s *dbService) UpdateCard(name string, input domain.CardInput) (domain.Card, error) {
	if strings.TrimSpace(name) == "" {
		return domain.Card{}, fmt.Errorf("%w: card name is required", domain.ErrValidation)
	}
	return s.writeCard(name, input)
}

func (s *dbService) writeCard(existing string, input domain.CardInput) (domain.Card, error) {
	input = input.Normalized()
	if err := input.Validate(); err != nil {
		return domain.Card{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, titles, err := s.readCardRows()
	if err != nil {
		return domain.Card{}, err
	}
	index := lastUsedRow(rows.cards)
	if existing != "" {
		index = -1
		for at, row := range rows.cards {
			if domain.FoldText(cellAt(row, 0)) == domain.FoldText(existing) {
				index = at
				break
			}
		}
		if index < 0 {
			return domain.Card{}, fmt.Errorf("%w: card %q", domain.ErrNotFound, existing)
		}
	}
	values := input.SheetRow()
	proposed := rows
	proposed.cards = withRow(rows.cards, index, asRead(values))
	config, err := mapCardConfig(proposed)
	if err != nil {
		return domain.Card{}, err
	}
	card, _ := findCard(config.Cards, input.Name)

	sheetRow := cardsFirstRow + index
	updates := []domain.CellUpdate{{Range: fmt.Sprintf("'%s'!A%d:I%d", cardsSheet, sheetRow, sheetRow), Values: values}}
	if input.Limit != nil {
		limit, err := s.creditLimitUpdate(config, card, *input.Limit)
		if err != nil {
			return domain.Card{}, err
		}
		updates = append(updates, limit)
	}
	if err := s.ensureTab(titles, cardsSheet, cardsHeader); err != nil {
		return domain.Card{}, err
	}
	err = s.client.UpdateCells(updates)
	s.invalidate(cacheKeyCards, "accounts:*")
	if err != nil {
		return domain.Card{}, fmt.Errorf("write card %s: %w", input.Name, err)
	}
	return card, nil
}

func findCard(cards []domain.Card, name string) (domain.Card, bool) {
	for _, card := range cards {
		if domain.FoldText(card.Name) == domain.FoldText(name) {
			return card, true
		}
	}
	return domain.Card{}, false
}

// creditLimitUpdate writes the limit to the shared account of the card's
// bank (Accounts C), where the cards' usage is measured (WM-CARD-9).
func (s *dbService) creditLimitUpdate(config domain.CardConfig, card domain.Card, limit float64) (domain.CellUpdate, error) {
	bank := ""
	for _, program := range config.Programs {
		if program.Name == card.Program {
			bank = program.Bank
		}
	}
	shared := ""
	for _, candidate := range config.Banks {
		if domain.FoldText(candidate.Name) == domain.FoldText(bank) {
			shared = candidate.SharedAccount
		}
	}
	if shared == "" {
		return domain.CellUpdate{}, fmt.Errorf("%w: bank %q has no shared account in %s, so the limit has nowhere to go", domain.ErrValidation, bank, banksSheet)
	}
	accounts, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return domain.CellUpdate{}, fmt.Errorf("read accounts: %w", err)
	}
	index, found := findAccountRow(accounts, shared)
	if !found {
		return domain.CellUpdate{}, fmt.Errorf("%w: shared account %q is not an account", domain.ErrValidation, shared)
	}
	return domain.CellUpdate{Range: fmt.Sprintf("Accounts!C%d", accountsFirstRow+index), Values: []any{limit}}, nil
}

// CreateCardProgram appends a programme version and its rules
// (WM-FILL-13). ErrConflict when the programme already has that version.
func (s *dbService) CreateCardProgram(input domain.ProgramInput) (domain.Program, error) {
	input = input.Normalized()
	from, err := input.Validate()
	if err != nil {
		return domain.Program{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, titles, err := s.readCardRows()
	if err != nil {
		return domain.Program{}, err
	}
	current, err := mapCardConfig(rows)
	if err != nil {
		return domain.Program{}, err
	}
	for _, program := range current.Programs {
		if domain.FoldText(program.Name) != domain.FoldText(input.Program) {
			continue
		}
		for _, version := range program.Versions {
			if version.From.Equal(from) {
				return domain.Program{}, fmt.Errorf("%w: %s already has terms from %s", domain.ErrConflict, program.Name, input.From)
			}
		}
	}
	bankUpdate, err := s.newBankRow(current, rows.banks, input)
	if err != nil {
		return domain.Program{}, err
	}
	programIndex := lastUsedRow(rows.programs)
	ruleIndex := lastUsedRow(rows.rules)
	proposed := rows
	if bankUpdate != nil {
		proposed.banks = withRow(rows.banks, lastUsedRow(rows.banks), asRead(bankUpdate.Values))
	}
	proposed.programs = withRow(rows.programs, programIndex, asRead(input.ProgramRow()))
	ruleRows := input.RuleRows()
	for offset, rule := range ruleRows {
		proposed.rules = withRow(proposed.rules, ruleIndex+offset, asRead(rule))
	}
	config, err := mapCardConfig(proposed)
	if err != nil {
		return domain.Program{}, err
	}
	if err := s.ensureTab(titles, programsSheet, programsHeader); err != nil {
		return domain.Program{}, err
	}
	if err := s.ensureTab(titles, programRulesSheet, rulesHeader); err != nil {
		return domain.Program{}, err
	}
	if bankUpdate != nil {
		if err := s.ensureTab(titles, banksSheet, banksHeader); err != nil {
			return domain.Program{}, err
		}
	}
	programRow := cardsFirstRow + programIndex
	updates := []domain.CellUpdate{{Range: fmt.Sprintf("'%s'!A%d:F%d", programsSheet, programRow, programRow), Values: input.ProgramRow()}}
	for offset, rule := range ruleRows {
		row := cardsFirstRow + ruleIndex + offset
		updates = append(updates, domain.CellUpdate{Range: fmt.Sprintf("'%s'!A%d:K%d", programRulesSheet, row, row), Values: rule})
	}
	if bankUpdate != nil {
		updates = append(updates, *bankUpdate)
	}
	err = s.client.UpdateCells(updates)
	s.invalidate(cacheKeyCards)
	if err != nil {
		return domain.Program{}, fmt.Errorf("write programme %s: %w", input.Program, err)
	}
	for _, program := range config.Programs {
		if domain.FoldText(program.Name) == domain.FoldText(input.Program) {
			return program, nil
		}
	}
	return domain.Program{}, nil
}

// newBankRow is the Banks row a programme of a new bank adds, nil when the
// bank has a row. A shared account given for a known bank that bills
// elsewhere is refused rather than ignored.
func (s *dbService) newBankRow(config domain.CardConfig, bankRows [][]string, input domain.ProgramInput) (*domain.CellUpdate, error) {
	for _, bank := range config.Banks {
		if domain.FoldText(bank.Name) != domain.FoldText(input.Bank) {
			continue
		}
		if input.SharedAccount != "" && domain.FoldText(bank.SharedAccount) != domain.FoldText(input.SharedAccount) {
			return nil, fmt.Errorf("%w: %s cards already bill to %s; change that in the %s tab", domain.ErrValidation, bank.Name, bank.SharedAccount, banksSheet)
		}
		return nil, nil
	}
	if input.SharedAccount == "" {
		return nil, nil
	}
	accounts, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return nil, fmt.Errorf("read accounts: %w", err)
	}
	index, found := findAccountRow(accounts, input.SharedAccount)
	if !found {
		return nil, fmt.Errorf("%w: shared account %q is not an account", domain.ErrValidation, input.SharedAccount)
	}
	account := strings.TrimSpace(cellAt(accounts[index], 0))
	row := cardsFirstRow + lastUsedRow(bankRows)
	return &domain.CellUpdate{Range: fmt.Sprintf("'%s'!A%d:B%d", banksSheet, row, row), Values: []any{input.Bank, account}}, nil
}

// SaveMerchant adds a Merchants row, or replaces the row with the same
// pattern and start date (WM-FILL-14).
func (s *dbService) SaveMerchant(input domain.MerchantInput) (domain.MerchantMCC, error) {
	input = input.Normalized()
	if err := input.Validate(); err != nil {
		return domain.MerchantMCC{}, err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	rows, titles, err := s.readCardRows()
	if err != nil {
		return domain.MerchantMCC{}, err
	}
	index := lastUsedRow(rows.merchants)
	for at, row := range rows.merchants {
		if domain.FoldText(cellAt(row, 0)) == domain.FoldText(input.Pattern) && sameSheetDate(cellAt(row, 2), input.From) {
			index = at
			break
		}
	}
	values := input.SheetRow()
	proposed := rows
	proposed.merchants = withRow(rows.merchants, index, asRead(values))
	config, err := mapCardConfig(proposed)
	if err != nil {
		return domain.MerchantMCC{}, err
	}
	if err := s.ensureTab(titles, merchantsSheet, merchantsHeader); err != nil {
		return domain.MerchantMCC{}, err
	}
	sheetRow := cardsFirstRow + index
	err = s.client.UpdateCells([]domain.CellUpdate{{Range: fmt.Sprintf("'%s'!A%d:E%d", merchantsSheet, sheetRow, sheetRow), Values: values}})
	s.invalidate(cacheKeyCards)
	if err != nil {
		return domain.MerchantMCC{}, fmt.Errorf("write merchant %s: %w", input.Pattern, err)
	}
	for _, merchant := range config.Merchants {
		if domain.FoldText(merchant.Pattern) == domain.FoldText(input.Pattern) && sameMerchantFrom(merchant.From, input.From) {
			return merchant, nil
		}
	}
	return domain.MerchantMCC{}, nil
}

// sameSheetDate compares a read date cell (a serial or written date) with
// a yyyy-mm-dd input; both blank are the same.
func sameSheetDate(cell string, value string) bool {
	cell = strings.TrimSpace(cell)
	if cell == "" || value == "" {
		return cell == "" && value == ""
	}
	date, err := domain.ParseSheetDate(cell)
	return err == nil && date.Format(time.DateOnly) == value
}

func sameMerchantFrom(from *time.Time, value string) bool {
	if from == nil {
		return value == ""
	}
	return from.Format(time.DateOnly) == value
}
