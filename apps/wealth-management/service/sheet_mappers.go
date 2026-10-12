package service

import (
	"apps/wealth-management/domain"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Row mappers for each sheet tab (REQUIREMENTS §2).

const (
	transactionWriteColumns = 10 // A-J are user data; K-M are sheet formulas
	budgetHeaderRowIndex    = 9  // row 10: month serials from column D
	budgetFirstDataRowIndex = 11 // row 12
	budgetFirstMonthColumn  = 3  // column D
	budgetYearlyColumn      = 2  // column C
)

func mapAccount(row []string) (domain.Account, bool) {
	name := strings.TrimSpace(cellAt(row, 0))
	if isJunkAccountName(name) || strings.HasPrefix(name, domain.DeletedAccountPrefix) {
		return domain.Account{}, false
	}
	typeRaw := strings.ToLower(strings.TrimSpace(cellAt(row, 6)))
	if typeRaw == "type" || typeRaw == "date to pay" {
		return domain.Account{}, false
	}
	if typeRaw == "" {
		typeRaw = string(domain.AccountTypeBank)
	}
	var dueDate *string
	if value := formatDueDate(cellAt(row, 1)); value != "" {
		dueDate = &value
	}
	return domain.Account{
		Name:           name,
		DueDate:        dueDate,
		GoalAmount:     nullableNumber(cellAt(row, 2)),
		GoalProgress:   nullableGoalProgress(cellAt(row, 3)),
		ClearedBalance: parseNumber(cellAt(row, 4)),
		Balance:        parseNumber(cellAt(row, 5)),
		Type:           domain.AccountType(typeRaw),
		Currency:       resolveCurrency(cellAt(row, 8), name),
		Note:           nullableString(cellAt(row, 7)),
	}, true
}

// mapTransaction maps a Transactions row; rowNumber is the 1-based sheet row.
// An unparseable date is flagged in Issues instead of becoming "now" (WM-DATA-3).
func mapTransaction(row []string, rowNumber int) (domain.Transaction, bool) {
	accountName := strings.TrimSpace(cellAt(row, 0))
	if isJunkAccountName(accountName) {
		return domain.Transaction{}, false
	}
	category := strings.TrimSpace(cellAt(row, 6))
	if category == "Category" || accountName == "Account" {
		return domain.Transaction{}, false
	}
	if category == "" {
		category = "Uncategorized"
	}
	transaction := domain.Transaction{
		ID:              transactionID(rowNumber),
		AccountName:     accountName,
		ReferenceNumber: nullableString(cellAt(row, 2)),
		Payee:           strings.TrimSpace(cellAt(row, 3)),
		Tags:            splitTags(cellAt(row, 4)),
		Memo:            nullableString(cellAt(row, 5)),
		Category:        category,
		Cleared:         parseCleared(cellAt(row, 7)),
		Payment:         nullableNumber(cellAt(row, 8)),
		Deposit:         nullableNumber(cellAt(row, 9)),
		AccountBalance:  parseNumber(cellAt(row, 10)),
		ClearedBalance:  parseNumber(cellAt(row, 11)),
		RunningBalance:  parseNumber(cellAt(row, 12)),
		ETag:            rowETag(row, transactionWriteColumns),
	}
	date, err := domain.ParseSheetDate(cellAt(row, 1))
	if err != nil {
		transaction.Issues = append(transaction.Issues, fmt.Sprintf("unparseable date %q", strings.TrimSpace(cellAt(row, 1))))
	} else {
		transaction.Date = date
	}
	return transaction, true
}

func transactionID(rowNumber int) string {
	return fmt.Sprintf("row-%d", rowNumber)
}

// parseTransactionID returns the sheet row of a "row-N" id.
func parseTransactionID(id string) (int, error) {
	var row int
	if _, err := fmt.Sscanf(strings.TrimSpace(id), "row-%d", &row); err != nil || row < 2 || transactionID(row) != strings.TrimSpace(id) {
		return 0, fmt.Errorf("%w: transaction id %q must look like row-N", domain.ErrValidation, id)
	}
	return row, nil
}

// transactionRowValues renders columns A-J in the sheet's write format;
// currency is the account's, which decides USD conversion.
func transactionRowValues(input domain.TransactionCreateInput, currency domain.Currency) ([]any, error) {
	date, err := domain.ParseSheetDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	return []any{
		strings.TrimSpace(input.AccountName),
		date.Format(domain.SheetDateLayout),
		nilToString(input.ReferenceNumber),
		strings.TrimSpace(input.Payee),
		strings.Join(cleanStrings(input.Tags), ","),
		nilToString(input.Memo),
		strings.TrimSpace(input.Category),
		boolToCheckmark(input.Cleared),
		formatTransactionAmount(input.Payment, input.AccountName, currency),
		formatTransactionAmount(input.Deposit, input.AccountName, currency),
	}, nil
}

func parseCategories(rows [][]string) []domain.Category {
	categories := make([]domain.Category, 0)
	seen := map[string]struct{}{}
	var currentGroup *domain.CategoryType
	for _, row := range rows {
		cell := strings.TrimSpace(cellAt(row, 0))
		if cell == "" {
			continue
		}
		lower := strings.ToLower(cell)
		switch {
		case strings.Contains(lower, "income categories"):
			group := domain.CategoryTypeIncome
			currentGroup = &group
			continue
		case strings.Contains(lower, "expense categories"):
			group := domain.CategoryTypeExpense
			currentGroup = &group
			continue
		case strings.Contains(lower, "non-budget categories"):
			group := domain.CategoryTypeNonBudget
			currentGroup = &group
			continue
		case lower == "categories":
			continue
		}
		if currentGroup == nil {
			continue
		}
		key := normalizeKey(cell)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		categories = append(categories, domain.Category{Name: cell, Type: *currentGroup})
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	return categories
}

// budgetLayout locates the cells of a budget tab, for reads and limit edits.
type budgetLayout struct {
	monthColumns  map[string]int // "YYYY-MM" -> column index
	categoryRows  map[string]int // normalised category -> 1-based sheet row
	categoryNames map[string]string
}

func parseBudget(rows [][]string) ([]domain.BudgetItem, budgetLayout) {
	layout := budgetLayout{monthColumns: map[string]int{}, categoryRows: map[string]int{}, categoryNames: map[string]string{}}
	headerRow := rowAt(rows, budgetHeaderRowIndex)
	for c := budgetFirstMonthColumn; c < len(headerRow); c++ {
		if serial := parseNumber(headerRow[c]); serial > 1000 {
			layout.monthColumns[domain.MonthKey(domain.SerialToDate(serial))] = c
		}
	}
	items := make([]domain.BudgetItem, 0)
	currentType := domain.CategoryTypeNonBudget
	for r := budgetFirstDataRowIndex; r < len(rows); r++ {
		row := rows[r]
		category := strings.TrimSpace(cellAt(row, 0))
		if category == "" {
			continue
		}
		switch strings.ToLower(category) {
		case "income categories":
			currentType = domain.CategoryTypeIncome
			continue
		case "expense categories":
			currentType = domain.CategoryTypeExpense
			continue
		case "transfer categories":
			currentType = domain.CategoryTypeNonBudget
			continue
		}
		if isJunkBudgetCategory(category) {
			continue
		}
		layout.categoryRows[normalizeKey(category)] = r + 1
		layout.categoryNames[normalizeKey(category)] = category

		yearlyLimit := parseNumber(cellAt(row, budgetYearlyColumn))
		monthlyLimits := map[string]float64{}
		monthlyLimitSum := 0.0
		for month, col := range layout.monthColumns {
			value := parseNumber(cellAt(row, col))
			if value == 0 {
				continue
			}
			monthlyLimits[month] = value
			monthlyLimitSum += value
		}
		if yearlyLimit == 0 {
			yearlyLimit = monthlyLimitSum
		}
		monthlyLimit := 0.0
		if yearlyLimit > 0 {
			monthlyLimit = float64(int(yearlyLimit / 12.0))
		}
		categoryType := currentType
		items = append(items, domain.BudgetItem{
			Category:         category,
			CategoryType:     &categoryType,
			MonthlyLimit:     monthlyLimit,
			YearlyLimit:      yearlyLimit,
			MonthlyLimits:    monthlyLimits,
			MonthlyRemaining: monthlyLimit,
			YearlyRemaining:  yearlyLimit,
			Note:             strings.TrimSpace(cellAt(row, 1)),
		})
	}
	return items, layout
}

func mapGoal(row []string, index int) (domain.Goal, bool) {
	name := strings.TrimSpace(cellAt(row, 4))
	if name == "" || name == "Instructions" || strings.Contains(name, "Insert more rows") {
		return domain.Goal{}, false
	}
	targetAmount := parseVND(cellAt(row, 6))
	currentAmount := parseVND(cellAt(row, 8))
	if targetAmount == 0 && currentAmount == 0 {
		return domain.Goal{}, false
	}
	goalType := domain.GoalTypeSavingsTarget
	emoji := "🎯"
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "xe"), strings.Contains(lower, "car"):
		goalType = domain.GoalTypePurchaseGoal
		emoji = "🚗"
	case strings.Contains(lower, "nhà"), strings.Contains(lower, "house"), strings.Contains(lower, "đất"):
		goalType = domain.GoalTypePurchaseGoal
		emoji = "🏠"
	case strings.Contains(lower, "học"), strings.Contains(lower, "school"):
		emoji = "🎓"
	case strings.Contains(lower, "du lịch"), strings.Contains(lower, "vacation"):
		goalType = domain.GoalTypePurchaseGoal
		emoji = "✈️"
	}
	status := domain.GoalStatusAtRisk
	if targetAmount > 0 && currentAmount/targetAmount >= 0.5 {
		status = domain.GoalStatusOnTrack
	}
	return domain.Goal{
		ID:               fmt.Sprintf("goal-%d", index),
		Name:             name,
		Type:             goalType,
		Emoji:            emoji,
		TargetAmount:     targetAmount,
		CurrentAmount:    currentAmount,
		Deadline:         "TBD",
		Status:           status,
		LinkedAccountID:  "sheet-goals",
		ContributionType: "MANUAL",
		Milestones:       []domain.Milestone{},
		History:          []domain.Contribution{},
	}, true
}

func mapLoan(row []string) (domain.Loan, bool) {
	name := strings.TrimSpace(cellAt(row, 0))
	if name == "" {
		return domain.Loan{}, false
	}
	return domain.Loan{
		Name:             name,
		MonthlyDebt:      parseNumber(cellAt(row, 1)),
		MonthlyPaid:      parseNumber(cellAt(row, 2)),
		MonthlyRemaining: parseNumber(cellAt(row, 3)),
		Extra:            cellAt(row, 4),
		YearlyDebt:       parseNumber(cellAt(row, 5)),
		YearlyPaid:       parseNumber(cellAt(row, 6)),
		YearlyRemaining:  parseNumber(cellAt(row, 7)),
	}, true
}

func isJunkBudgetCategory(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return true
	}
	switch lower {
	case "help", "income categories", "expense categories", "transfer categories":
		return true
	}
	for _, prefix := range []string{"total ", "net ", "projected", "[", "starting balance"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return onlyDigits(lower)
}

func isJunkAccountName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return true
	}
	switch lower {
	case "help", "accounts", "account":
		return true
	}
	for _, prefix := range []string{"you can track", "accounts column", "to get started"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func buildCategoryIndex(categories []domain.Category) map[string]*domain.CategoryType {
	index := make(map[string]*domain.CategoryType, len(categories))
	for _, category := range categories {
		categoryType := category.Type
		index[normalizeKey(category.Name)] = &categoryType
	}
	return index
}

// sortTransactionsNewestFirst orders by date, later sheet rows first on the
// same day (WM-TX-1). Rows arrive in sheet order.
func sortTransactionsNewestFirst(transactions []domain.Transaction) {
	slices.Reverse(transactions)
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].Date.After(transactions[j].Date)
	})
}
