package service

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	accountsSheetRange      = "Accounts!A2:I"
	transactionsSheetRange  = "Transactions!A2:M"
	transactionsAnchorRange = "Transactions!A2:A"
	budgetSheetRange        = "Budget_2026!A1:N200"
	categoriesSheetRange    = "Categories!A1:A153"
	goalsSheetRange         = "Goals!A1:Z50"
	loansSheetRange         = "'Loan / Dept'!A10:H50"
	notificationsSheetRange = "EmailNotifications!A3:F"
	cryptoSheetRange        = "Crypto!A1:Z100"
	fundsSheetRange         = "InvestmentFundCertificate!A1:Z100"
	promptsSheetRange       = "Prompts!A2:C"
	knowledgeSheetRange     = "Knowledge!A2:B"
)

type dbService struct {
	client port.DatabaseClient
	cache  port.CacheService
	log    *logger.Logger
}

func NewDatabaseService(client port.DatabaseClient, cache port.CacheService, log *logger.Logger) port.DatabaseService {
	log.LogApplicationEvent(context.Background(), "initializing database service")
	return &dbService{client: client, cache: cache, log: log}
}

func (s *dbService) ReadAccounts() (domain.Accounts, error) {
	rows, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		s.log.LogError(context.Background(), "failed to read accounts sheet", err,
			slog.String("range", accountsSheetRange),
		)
		return domain.Accounts{}, err
	}
	s.log.LogApplicationEvent(context.Background(), "read accounts sheet",
		slog.Int("row_count", len(rows)),
		slog.String("range", accountsSheetRange),
	)
	return domain.Accounts{Range: accountsSheetRange, Rows: rows}, nil
}

func (s *dbService) ListAccounts() ([]domain.Account, error) {
	const cacheKey = "accounts:all"
	if cached, ok, err := readCachedJSON[[]domain.Account](s.cache, cacheKey); err != nil {
		return nil, err
	} else if ok {
		return cached, nil
	}
	rows, err := s.client.ReadSheet(accountsSheetRange)
	if err != nil {
		return nil, err
	}
	accounts := make([]domain.Account, 0, len(rows))
	for _, row := range rows {
		account, ok := mapAccount(row)
		if ok {
			accounts = append(accounts, account)
		}
	}
	if err := writeCachedJSON(s.cache, cacheKey, accounts, 300); err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *dbService) ListTransactions(forceFresh bool) ([]domain.Transaction, error) {
	const cacheKey = "transactions:all"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[[]domain.Transaction](s.cache, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}
	rows, err := s.client.ReadSheet(transactionsSheetRange)
	if err != nil {
		return nil, err
	}
	categories, err := s.ListCategories(forceFresh)
	if err != nil {
		return nil, err
	}
	categoryIndex := buildCategoryIndex(categories)
	transactions := make([]domain.Transaction, 0, len(rows))
	for idx, row := range rows {
		transaction, ok := mapTransaction(row, idx+2)
		if !ok {
			continue
		}
		if categoryType, ok := categoryIndex[normalizeKey(transaction.Category)]; ok {
			transaction.CategoryType = categoryType
		}
		transactions = append(transactions, transaction)
	}
	if err := writeCachedJSON(s.cache, cacheKey, transactions, 300); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (s *dbService) CreateTransaction(input domain.TransactionCreateInput) error {
	date := parseFlexibleDate(input.Date)
	values := []any{
		strings.TrimSpace(input.AccountName),
		date.Format("02/01/2006"),
		nilToString(input.ReferenceNumber),
		strings.TrimSpace(input.Payee),
		strings.Join(cleanStrings(input.Tags), ","),
		nilToString(input.Memo),
		strings.TrimSpace(input.Category),
		boolToCheckmark(input.Cleared),
		formatTransactionAmount(input.Payment, input.AccountName),
		formatTransactionAmount(input.Deposit, input.AccountName),
	}
	if err := s.client.WriteToFirstEmptyRow("Transactions", transactionsAnchorRange, values); err != nil {
		return err
	}
	return s.invalidatePatterns("transactions:*", "budget:*")
}

func (s *dbService) ListCategories(forceFresh bool) ([]domain.Category, error) {
	const cacheKey = "categories:all"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[[]domain.Category](s.cache, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}
	rows, err := s.client.ReadSheet(categoriesSheetRange)
	if err != nil {
		return nil, err
	}
	categories := make([]domain.Category, 0)
	seen := map[string]struct{}{}
	var currentGroup *domain.CategoryType
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		cell := strings.TrimSpace(row[0])
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
	if err := writeCachedJSON(s.cache, cacheKey, categories, 300); err != nil {
		return nil, err
	}
	return categories, nil
}

func (s *dbService) ListBudget(forceFresh bool) ([]domain.BudgetItem, error) {
	const cacheKey = "budget:all"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[[]domain.BudgetItem](s.cache, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}
	rows, err := s.client.ReadSheet(budgetSheetRange)
	if err != nil {
		return nil, err
	}
	items := make([]domain.BudgetItem, 0)
	headerRow := rowAt(rows, 9)
	colToMonth := map[int]time.Time{}
	for c := 3; c < len(headerRow); c++ {
		serial := parseNumber(headerRow[c])
		if serial > 1000 {
			colToMonth[c] = googleSerialToDate(serial)
		}
	}
	currentType := domain.CategoryTypeNonBudget
	for r := 11; r < len(rows); r++ {
		row := rows[r]
		category := strings.TrimSpace(cellAt(row, 0))
		if category == "" {
			continue
		}
		lower := strings.ToLower(category)
		switch lower {
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
		yearlyLimit := parseNumber(cellAt(row, 2))
		monthlyLimits := map[string]float64{}
		monthlyLimitSum := 0.0
		for col, date := range colToMonth {
			value := parseNumber(cellAt(row, col))
			if value == 0 {
				continue
			}
			key := fmt.Sprintf("%04d-%02d", date.Year(), int(date.Month()))
			monthlyLimits[key] = value
			monthlyLimitSum += value
		}
		resolvedYearlyLimit := yearlyLimit
		if resolvedYearlyLimit == 0 {
			resolvedYearlyLimit = monthlyLimitSum
		}
		monthlyLimit := 0.0
		if resolvedYearlyLimit > 0 {
			monthlyLimit = float64(int(resolvedYearlyLimit / 12.0))
		}
		categoryType := currentType
		items = append(items, domain.BudgetItem{
			Category:         category,
			CategoryType:     &categoryType,
			MonthlyLimit:     monthlyLimit,
			YearlyLimit:      resolvedYearlyLimit,
			MonthlyLimits:    monthlyLimits,
			MonthlySpent:     0,
			YearlySpent:      0,
			MonthlyRemaining: monthlyLimit,
			YearlyRemaining:  resolvedYearlyLimit,
			Note:             strings.TrimSpace(cellAt(row, 1)),
		})
	}
	if err := writeCachedJSON(s.cache, cacheKey, items, 300); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *dbService) ListGoals(forceFresh bool) ([]domain.Goal, error) {
	const cacheKey = "goals"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[[]domain.Goal](s.cache, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}
	rows, err := s.client.ReadSheet(goalsSheetRange)
	if err != nil {
		return nil, err
	}
	goals := make([]domain.Goal, 0)
	for idx, row := range rows {
		goal, ok := mapGoal(row, idx)
		if ok {
			goals = append(goals, goal)
		}
	}
	if err := writeCachedJSON(s.cache, cacheKey, goals, 300); err != nil {
		return nil, err
	}
	return goals, nil
}

func (s *dbService) ListLoans(forceFresh bool) ([]domain.Loan, error) {
	const cacheKey = "loans"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[[]domain.Loan](s.cache, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}
	rows, err := s.client.ReadSheet(loansSheetRange)
	if err != nil {
		return nil, err
	}
	loans := make([]domain.Loan, 0)
	for _, row := range rows {
		name := strings.TrimSpace(cellAt(row, 0))
		if name == "" {
			continue
		}
		loans = append(loans, domain.Loan{
			Name:             name,
			MonthlyDebt:      parseNumber(cellAt(row, 1)),
			MonthlyPaid:      parseNumber(cellAt(row, 2)),
			MonthlyRemaining: parseNumber(cellAt(row, 3)),
			Extra:            cellAt(row, 4),
			YearlyDebt:       parseNumber(cellAt(row, 5)),
			YearlyPaid:       parseNumber(cellAt(row, 6)),
			YearlyRemaining:  parseNumber(cellAt(row, 7)),
		})
	}
	if err := writeCachedJSON(s.cache, cacheKey, loans, 300); err != nil {
		return nil, err
	}
	return loans, nil
}

func (s *dbService) ListNotifications(forceFresh bool) ([]domain.EmailNotification, error) {
	const cacheKey = "notifications"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[[]domain.EmailNotification](s.cache, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
	}
	rows, err := s.client.ReadSheet(notificationsSheetRange)
	if err != nil {
		return nil, err
	}
	notifications := make([]domain.EmailNotification, 0)
	for i, row := range rows {
		id := strings.TrimSpace(cellAt(row, 0))
		if id == "" {
			continue
		}
		status := domain.NotificationPending
		if strings.EqualFold(strings.TrimSpace(cellAt(row, 5)), "done") {
			status = domain.NotificationDone
		}
		notifications = append(notifications, domain.EmailNotification{
			ID:        id,
			Timestamp: cellAt(row, 1),
			From:      cellAt(row, 2),
			Subject:   cellAt(row, 3),
			Content:   cellAt(row, 4),
			Status:    status,
			RowNumber: i + 3,
		})
	}
	if err := writeCachedJSON(s.cache, cacheKey, notifications, 300); err != nil {
		return nil, err
	}
	return notifications, nil
}

func (s *dbService) ListPendingNotifications(forceFresh bool) ([]domain.EmailNotification, error) {
	notifications, err := s.ListNotifications(forceFresh)
	if err != nil {
		return nil, err
	}
	pending := make([]domain.EmailNotification, 0)
	for _, item := range notifications {
		if item.Status == domain.NotificationPending {
			pending = append(pending, item)
		}
	}
	return pending, nil
}

func (s *dbService) MarkNotificationDone(rowNumber int) error {
	if err := s.client.UpdateRow(fmt.Sprintf("EmailNotifications!F%d", rowNumber), []any{"done"}); err != nil {
		return err
	}
	return s.invalidatePatterns("notifications", "notifications:*")
}

func (s *dbService) ListTags(forceFresh bool) ([]string, error) {
	transactions, err := s.ListTransactions(forceFresh)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, txn := range transactions {
		for _, tag := range txn.Tags {
			normalized := normalizeKey(tag)
			if normalized == "" {
				continue
			}
			set[normalized] = struct{}{}
		}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

func (s *dbService) GetInvestmentAssets(forceFresh bool) (domain.InvestmentAssets, error) {
	const cacheKey = "investments:assets"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[domain.InvestmentAssets](s.cache, cacheKey); err != nil {
			return domain.InvestmentAssets{}, err
		} else if ok {
			return cached, nil
		}
	}
	cryptoRows, err := s.client.ReadSheet(cryptoSheetRange)
	if err != nil {
		return domain.InvestmentAssets{}, err
	}
	fundRows, err := s.client.ReadSheet(fundsSheetRange)
	if err != nil {
		return domain.InvestmentAssets{}, err
	}
	assets := domain.InvestmentAssets{
		Crypto: parseHoldings(cryptoRows, []string{"Currency", "Spot/Fund", "Token"}),
		Funds:  parseHoldings(fundRows, []string{"Index", "Total Unit", "Certificate"}),
	}
	if err := writeCachedJSON(s.cache, cacheKey, assets, 300); err != nil {
		return domain.InvestmentAssets{}, err
	}
	return assets, nil
}

func (s *dbService) WarmAIContent(forceFresh bool) (domain.InitResult, error) {
	const cacheKey = "sheets:init"
	if !forceFresh {
		if cached, ok, err := readCachedJSON[domain.InitResult](s.cache, cacheKey); err != nil {
			return domain.InitResult{}, err
		} else if ok {
			return cached, nil
		}
	}
	prompts, err := s.client.ReadSheet(promptsSheetRange)
	if err != nil {
		return domain.InitResult{}, err
	}
	knowledge, err := s.client.ReadSheet(knowledgeSheetRange)
	if err != nil {
		return domain.InitResult{}, err
	}
	result := domain.InitResult{Ready: true}
	result.Loaded.Prompts = countNonEmptyRows(prompts, 3)
	result.Loaded.Knowledge = countNonEmptyRows(knowledge, 2)
	if err := writeCachedJSON(s.cache, cacheKey, result, 3600); err != nil {
		return domain.InitResult{}, err
	}
	return result, nil
}

func (s *dbService) SyncCache() (domain.SyncResult, error) {
	result := domain.SyncResult{Success: true, Message: "Cache cleared successfully"}
	if s.cache == nil {
		return result, nil
	}
	patterns := []string{"accounts*", "transactions*", "budget*", "categories*", "goals*", "loans*", "notifications*", "sheets:*", "prompts*", "knowledge*"}
	patterns = []string{"accounts:*", "transactions:*", "budget:*", "categories:*", "goals", "loans", "notifications", "notifications:*", "sheets:init", "investments:assets"}
	for _, pattern := range patterns {
		deleted, err := s.cache.Invalidate(context.Background(), pattern)
		if err != nil {
			return domain.SyncResult{}, err
		}
		result.Deleted += deleted
	}
	return result, nil
}

func mapAccount(row []string) (domain.Account, bool) {
	name := strings.TrimSpace(cellAt(row, 0))
	if isJunkAccountName(name) {
		return domain.Account{}, false
	}
	typeRaw := strings.ToLower(strings.TrimSpace(cellAt(row, 6)))
	if typeRaw == "type" || typeRaw == "date to pay" {
		return domain.Account{}, false
	}
	var dueDate *string
	if value := formatDueDate(cellAt(row, 1)); value != "" {
		dueDate = &value
	}
	goalAmount := nullableNumber(cellAt(row, 2))
	goalProgress := nullableGoalProgress(cellAt(row, 3))
	return domain.Account{
		Name:           name,
		DueDate:        dueDate,
		GoalAmount:     goalAmount,
		GoalProgress:   goalProgress,
		ClearedBalance: parseNumber(cellAt(row, 4)),
		Balance:        parseNumber(cellAt(row, 5)),
		Type:           domain.AccountType(typeRaw),
		Currency:       inferCurrency(name),
		Note:           nullableString(cellAt(row, 7)),
	}, true
}

func mapTransaction(row []string, rowIndex int) (domain.Transaction, bool) {
	accountName := strings.TrimSpace(cellAt(row, 0))
	if isJunkAccountName(accountName) {
		return domain.Transaction{}, false
	}
	category := strings.TrimSpace(cellAt(row, 6))
	if category == "" {
		category = "Uncategorized"
	}
	if category == "Category" || accountName == "Account" {
		return domain.Transaction{}, false
	}
	return domain.Transaction{
		ID:              fmt.Sprintf("row-%d", rowIndex),
		AccountName:     accountName,
		Date:            parseFlexibleDate(cellAt(row, 1)),
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
	}, true
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
		goalType = domain.GoalTypeSavingsTarget
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
		StreakCount:      0,
		Milestones:       []domain.Milestone{},
		History:          []domain.Contribution{},
	}, true
}

func parseHoldings(rows [][]string, keywords []string) domain.InvestmentHoldingSection {
	headerIdx := -1
	for i, row := range rows {
		for _, cell := range row {
			trimmed := strings.TrimSpace(cell)
			for _, keyword := range keywords {
				if trimmed == keyword {
					headerIdx = i
					break
				}
			}
			if headerIdx >= 0 {
				break
			}
		}
		if headerIdx >= 0 {
			break
		}
	}
	if headerIdx < 0 {
		return domain.InvestmentHoldingSection{Headers: []string{}, Holdings: []map[string]any{}}
	}
	headersRaw := rows[headerIdx]
	headers := make([]string, len(headersRaw))
	for i, header := range headersRaw {
		if i == 0 && strings.TrimSpace(header) == "" {
			headers[i] = "Platform"
			continue
		}
		if strings.TrimSpace(header) == "" {
			headers[i] = fmt.Sprintf("Col%d", i)
			continue
		}
		headers[i] = header
	}
	holdings := make([]map[string]any, 0)
	lastPlatform := ""
	for _, row := range rows[headerIdx+1:] {
		item := map[string]any{}
		if strings.TrimSpace(cellAt(row, 0)) != "" {
			lastPlatform = cellAt(row, 0)
		}
		hasData := false
		for index, header := range headers {
			value := cellAt(row, index)
			if index == 0 {
				item[header] = lastPlatform
				continue
			}
			item[header] = value
			if strings.TrimSpace(value) != "" {
				hasData = true
			}
		}
		if hasData {
			holdings = append(holdings, item)
		}
	}
	return domain.InvestmentHoldingSection{Headers: headers, Holdings: holdings}
}

func buildCategoryIndex(categories []domain.Category) map[string]*domain.CategoryType {
	index := make(map[string]*domain.CategoryType, len(categories))
	for _, category := range categories {
		categoryType := category.Type
		index[normalizeKey(category.Name)] = &categoryType
	}
	return index
}

func cellAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func rowAt(rows [][]string, index int) []string {
	if index < 0 || index >= len(rows) {
		return nil
	}
	return rows[index]
}

func parseNumber(value string) float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	trimmed = strings.ReplaceAll(trimmed, ",", "")
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func nullableNumber(value string) *float64 {
	parsed := parseNumber(value)
	if parsed == 0 {
		return nil
	}
	return &parsed
}

func nullableGoalProgress(value string) *float64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parsed := parseNumber(trimmed) * 100
	return &parsed
}

func nullableString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func splitTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

func parseCleared(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "y" || lower == "yes" || value == "✓"
}

func parseFlexibleDate(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Now()
	}
	if serial, err := strconv.ParseFloat(trimmed, 64); err == nil && serial > 1000 && serial < 100000 {
		return googleSerialToDate(serial)
	}
	formats := []string{"01/02/2006", "1/2/2006", "02/01/2006", "2/1/2006", "01/02/06", "1/2/06", "02/01/06", "2/1/06", "2006-01-02", time.RFC3339}
	for _, layout := range formats {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			if parsed.Year() < 100 {
				return parsed.AddDate(2000, 0, 0)
			}
			return parsed
		}
	}
	if parsed, err := time.Parse(time.DateOnly, trimmed); err == nil {
		return parsed
	}
	return time.Now()
}

func googleSerialToDate(serial float64) time.Time {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	return epoch.AddDate(0, 0, int(serial))
}

func formatDueDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if serial, err := strconv.Atoi(trimmed); err == nil && serial > 1000 {
		date := googleSerialToDate(float64(serial))
		return fmt.Sprintf("%d/%d/%d", date.Day(), int(date.Month()), date.Year())
	}
	return trimmed
}

func parseVND(value string) float64 {
	clean := strings.NewReplacer(".", "", "₫", "", " ", "", ",", "").Replace(value)
	parsed, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func inferCurrency(name string) domain.Currency {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "binance") {
		return domain.CurrencyUSDT
	}
	return domain.CurrencyVND
}

func isJunkBudgetCategory(name string) bool {
	if strings.TrimSpace(name) == "" {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	sectionHeaders := map[string]struct{}{"help": {}, "income categories": {}, "expense categories": {}, "transfer categories": {}}
	if _, ok := sectionHeaders[lower]; ok {
		return true
	}
	for _, prefix := range []string{"total ", "net ", "projected", "[", "starting balance"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	if onlyDigits(lower) {
		return true
	}
	return false
}

func onlyDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isJunkAccountName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, exact := range []string{"help", "accounts", "account"} {
		if lower == exact {
			return true
		}
	}
	for _, prefix := range []string{"you can track", "accounts column", "to get started"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func nilToString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func boolToCheckmark(value bool) string {
	if value {
		return "✓"
	}
	return ""
}

func formatTransactionAmount(amount *float64, accountName string) any {
	if amount == nil {
		return ""
	}
	if strings.Contains(strings.ToLower(accountName), "binance") {
		return fmt.Sprintf(`=%v * IF(INDIRECT("A" & ROW())="%s"; GOOGLEFINANCE("CURRENCY:USDVND"); 1)`, *amount, accountName)
	}
	return *amount
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func countNonEmptyRows(rows [][]string, requiredColumns int) int {
	count := 0
	for _, row := range rows {
		nonEmpty := 0
		limit := requiredColumns
		if len(row) < limit {
			limit = len(row)
		}
		for i := 0; i < limit; i++ {
			if strings.TrimSpace(row[i]) != "" {
				nonEmpty++
			}
		}
		if nonEmpty == requiredColumns {
			count++
		}
	}
	return count
}

func (s *dbService) invalidatePatterns(patterns ...string) error {
	if s.cache == nil {
		return nil
	}
	for _, pattern := range patterns {
		if _, err := s.cache.Invalidate(context.Background(), pattern); err != nil {
			return err
		}
	}
	return nil
}

func readCachedJSON[T any](cache port.CacheService, key string) (T, bool, error) {
	var zero T
	if cache == nil {
		return zero, false, nil
	}
	entry, err := cache.Get(context.Background(), key)
	if err != nil {
		return zero, false, err
	}
	if !entry.Found || strings.TrimSpace(entry.Value) == "" {
		return zero, false, nil
	}
	var value T
	if err := json.Unmarshal([]byte(entry.Value), &value); err != nil {
		return zero, false, err
	}
	return value, true, nil
}

func writeCachedJSON(cache port.CacheService, key string, value any, ttlSeconds int) error {
	if cache == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cache.Set(context.Background(), key, string(payload), ttlSeconds)
}
