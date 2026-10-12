package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Credit card cashback (WM-RULE-7, WM-CARD-1..3). Card programs are data
// (see adapter/config/cards.default.json), not code.

const beginningBalanceCategory = "[beginning balance]"

// CashbackRule pays Rate percent on payments tagged with Tag, up to CapMonthly per cycle.
type CashbackRule struct {
	Name       string  `json:"name"`
	Tag        string  `json:"tag"`
	Rate       float64 `json:"rate"`
	CapMonthly float64 `json:"capMonthly"`
	// Fallback rules apply to the card's spend that matches no other rule tag.
	Fallback bool `json:"fallback,omitempty"`
}

// CardProgram describes one physical card and its cashback rules.
type CardProgram struct {
	Name         string         `json:"name"`
	Bank         string         `json:"bank"`
	Keyword      string         `json:"keyword"`
	DefaultLimit float64        `json:"defaultLimit"`
	Expiry       string         `json:"expiry"`
	AnnualFee    float64        `json:"annualFee"`
	MonthlyCap   float64        `json:"monthlyCap"`
	StatementDay int            `json:"statementDay"`
	DueDay       int            `json:"dueDay"`
	Rules        []CashbackRule `json:"rules"`
}

// CardBank groups programs sharing a bank-level account and cashback cap.
type CardBank struct {
	Name string `json:"name"`
	// SharedAccount holds spend for every card at this bank; untagged spend on
	// it belongs to DefaultCardKeyword.
	SharedAccount      string        `json:"sharedAccount"`
	DefaultCardKeyword string        `json:"defaultCardKeyword"`
	SharedMonthlyCap   float64       `json:"sharedMonthlyCap"`
	Cards              []CardProgram `json:"cards"`
}

// CardConfig is the full cashback configuration.
type CardConfig struct {
	Banks []CardBank `json:"banks"`
}

// CardHistoryMonth is one month of a card's cashback efficiency (WM-CARD-2).
type CardHistoryMonth struct {
	Month             string  `json:"month"`
	Spent             float64 `json:"spent"`
	PotentialCashback float64 `json:"potentialCashback"`
	ActualRefund      float64 `json:"actualRefund"`
	Efficiency        float64 `json:"efficiency"`
	Missed            float64 `json:"missed"`
}

// CardReport is the computed state of one card.
type CardReport struct {
	Name              string             `json:"name"`
	Bank              string             `json:"bank"`
	Keyword           string             `json:"keyword"`
	AccountName       string             `json:"accountName"`
	Expiry            string             `json:"expiry"`
	Limit             float64            `json:"limit"`
	Usage             float64            `json:"usage"`
	Utilization       float64            `json:"utilization"`
	UtilizationBand   UtilizationBand    `json:"utilizationBand"`
	RemainingLimit    float64            `json:"remainingLimit"`
	SpentThisCycle    float64            `json:"spentThisCycle"`
	EstimatedCashback float64            `json:"estimatedCashback"`
	RefundYTD         float64            `json:"refundYtd"`
	AnnualFee         float64            `json:"annualFee"`
	NetEarnYTD        float64            `json:"netEarnYtd"`
	LifetimeFees      float64            `json:"lifetimeFees"`
	TransactionCount  int                `json:"transactionCount"`
	Cycle             CardCycle          `json:"cycle"`
	Rules             []CashbackRule     `json:"rules"`
	History           []CardHistoryMonth `json:"history"`
	CycleTransactions []Transaction      `json:"cycleTransactions"`
}

// CardAccountGroup aggregates the cards billed to one sheet account.
type CardAccountGroup struct {
	Name              string       `json:"name"`
	Shared            bool         `json:"shared"`
	Limit             float64      `json:"limit"`
	Usage             float64      `json:"usage"`
	Utilization       float64      `json:"utilization"`
	RemainingLimit    float64      `json:"remainingLimit"`
	EstimatedCashback float64      `json:"estimatedCashback"`
	RefundYTD         float64      `json:"refundYtd"`
	AnnualFees        float64      `json:"annualFees"`
	NetEarnYTD        float64      `json:"netEarnYtd"`
	Cards             []CardReport `json:"cards"`
}

// CardBankReport groups card accounts by bank.
type CardBankReport struct {
	Name     string             `json:"name"`
	Accounts []CardAccountGroup `json:"accounts"`
}

const cardHistoryMonths = 36

// TransactionCashback returns the rule-based cashback for one payment on a program.
func TransactionCashback(program CardProgram, t Transaction) (float64, string) {
	payment := t.PaymentAmount()
	if payment <= 0 || normalizeCategory(t.Category) == beginningBalanceCategory {
		return 0, ""
	}
	tags := lowerTags(t.Tags)
	var fallback *CashbackRule
	for i := range program.Rules {
		rule := program.Rules[i]
		if tags[strings.ToLower(rule.Tag)] {
			return math.Round(payment * rule.Rate / 100), rule.Name
		}
		if rule.Fallback && fallback == nil {
			fallback = &program.Rules[i]
		}
	}
	if fallback == nil {
		return 0, ""
	}
	return math.Round(payment * fallback.Rate / 100), fallback.Name
}

// IsCashbackRefund matches a refund deposit from the bank's cashback programme.
func IsCashbackRefund(t Transaction) bool {
	return t.DepositAmount() > 0 && strings.Contains(strings.ToLower(t.Payee), "cashback") && IsRefundCategory(t.Category)
}

// ComputeCardReports evaluates every configured card against accounts and transactions.
func ComputeCardReports(config CardConfig, accounts []Account, transactions []Transaction, today time.Time) []CardBankReport {
	reports := make([]CardBankReport, 0, len(config.Banks))
	for _, bank := range config.Banks {
		bankReport := CardBankReport{Name: bank.Name, Accounts: []CardAccountGroup{}}
		groups := map[string]*CardAccountGroup{}
		order := []string{}
		for _, program := range bank.Cards {
			account, found := findCardAccount(bank, program, accounts)
			accountName := bank.SharedAccount
			if found {
				accountName = account.Name
			}
			card := computeCard(bank, program, account, accountName, transactions, today)
			group, ok := groups[accountName]
			if !ok {
				group = &CardAccountGroup{Name: accountName, Shared: accountName == bank.SharedAccount, Limit: card.Limit, Usage: card.Usage, Cards: []CardReport{}}
				groups[accountName] = group
				order = append(order, accountName)
			}
			group.RefundYTD += card.RefundYTD
			group.AnnualFees += card.AnnualFee
			group.EstimatedCashback += card.EstimatedCashback
			group.Cards = append(group.Cards, card)
		}
		for _, name := range order {
			group := groups[name]
			if bank.SharedMonthlyCap > 0 {
				group.EstimatedCashback = math.Min(group.EstimatedCashback, bank.SharedMonthlyCap)
			}
			group.NetEarnYTD = group.RefundYTD - group.AnnualFees
			group.Utilization = percentOf(group.Usage, group.Limit)
			group.RemainingLimit = group.Limit - group.Usage
			bankReport.Accounts = append(bankReport.Accounts, *group)
		}
		reports = append(reports, bankReport)
	}
	return reports
}

// findCardAccount prefers a liability account named after the card, else the bank's shared account.
func findCardAccount(bank CardBank, program CardProgram, accounts []Account) (Account, bool) {
	keyword := strings.ToLower(program.Keyword)
	var shared *Account
	for i := range accounts {
		account := accounts[i]
		if account.Type != AccountTypeNegativeActive {
			continue
		}
		if strings.Contains(strings.ToLower(account.Name), keyword) {
			return account, true
		}
		if account.Name == bank.SharedAccount {
			shared = &accounts[i]
		}
	}
	if shared != nil {
		return *shared, true
	}
	return Account{}, false
}

// belongsToCard applies the membership rules: a card tag wins; on the card's
// own account another card's tag excludes the row; untagged rows on a shared
// account belong to the bank's default card.
func belongsToCard(bank CardBank, program CardProgram, accountName string, t Transaction) bool {
	keyword := strings.ToLower(program.Keyword)
	hasTag := func(word string) bool {
		for _, tag := range t.Tags {
			if strings.Contains(strings.ToLower(tag), word) {
				return true
			}
		}
		return false
	}
	if hasTag(keyword) {
		return true
	}
	if t.AccountName != accountName {
		return false
	}
	for _, other := range bank.Cards {
		if other.Keyword != program.Keyword && hasTag(strings.ToLower(other.Keyword)) {
			return false
		}
	}
	if accountName == bank.SharedAccount {
		return strings.EqualFold(program.Keyword, bank.DefaultCardKeyword)
	}
	return true
}

func computeCard(bank CardBank, program CardProgram, account Account, accountName string, transactions []Transaction, today time.Time) CardReport {
	limit := program.DefaultLimit
	if account.GoalAmount != nil && *account.GoalAmount > 0 {
		limit = *account.GoalAmount
	}
	statementDay, dueDay, ok := ParseCardDays(account.DueDate)
	if !ok {
		statementDay, dueDay = program.StatementDay, program.DueDay
	}
	usage := math.Abs(account.ClearedBalance)
	utilization := percentOf(usage, limit)
	card := CardReport{
		Name: program.Name, Bank: bank.Name, Keyword: program.Keyword, AccountName: accountName, Expiry: program.Expiry,
		Limit: limit, Usage: usage, Utilization: utilization, UtilizationBand: UtilizationBandFor(utilization),
		RemainingLimit: limit - usage, AnnualFee: program.AnnualFee, Rules: program.Rules,
		Cycle:             ComputeCardCycle(statementDay, dueDay, account.Balance, today),
		CycleTransactions: []Transaction{},
	}
	cycleStart := CycleStartFor(statementDay, today)

	ruleTotals := map[string]float64{}
	history := map[string]*CardHistoryMonth{}
	firstSeen := today
	monthEntry := func(date time.Time) *CardHistoryMonth {
		key := MonthKey(date)
		if history[key] == nil {
			history[key] = &CardHistoryMonth{Month: key}
		}
		return history[key]
	}
	for _, t := range transactions {
		if !t.Usable() || !belongsToCard(bank, program, accountName, t) {
			continue
		}
		if t.Date.Before(firstSeen) {
			firstSeen = t.Date
		}
		cashback, ruleName := TransactionCashback(program, t)
		spend := 0.0
		if normalizeCategory(t.Category) != beginningBalanceCategory {
			spend = t.PaymentAmount()
		}
		month := monthEntry(t.Date)
		month.Spent += spend
		month.PotentialCashback += cashback
		if IsCashbackRefund(t) {
			effective := EffectiveDate(t)
			monthEntry(effective).ActualRefund += t.DepositAmount()
			if effective.Year() == today.Year() {
				card.RefundYTD += t.DepositAmount()
			}
		}
		if !t.Date.Before(cycleStart) && !t.Date.After(today) {
			card.SpentThisCycle += spend
			card.TransactionCount++
			card.CycleTransactions = append(card.CycleTransactions, t)
			if ruleName != "" {
				ruleTotals[ruleName] += cashback
			}
		}
	}
	for _, rule := range program.Rules {
		amount := ruleTotals[rule.Name]
		if rule.CapMonthly > 0 {
			amount = math.Min(amount, rule.CapMonthly)
		}
		card.EstimatedCashback += amount
	}
	if program.MonthlyCap > 0 {
		card.EstimatedCashback = math.Min(card.EstimatedCashback, program.MonthlyCap)
	}
	card.NetEarnYTD = card.RefundYTD - card.AnnualFee
	card.LifetimeFees = float64(yearsActive(firstSeen, today)) * card.AnnualFee
	sort.SliceStable(card.CycleTransactions, func(i, j int) bool {
		return card.CycleTransactions[i].Date.After(card.CycleTransactions[j].Date)
	})
	card.History = finishCardHistory(history, program.MonthlyCap)
	return card
}

func finishCardHistory(history map[string]*CardHistoryMonth, monthlyCap float64) []CardHistoryMonth {
	keys := make([]string, 0, len(history))
	for key := range history {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	if len(keys) > cardHistoryMonths {
		keys = keys[:cardHistoryMonths]
	}
	months := make([]CardHistoryMonth, 0, len(keys))
	for _, key := range keys {
		month := *history[key]
		if monthlyCap > 0 {
			month.PotentialCashback = math.Min(month.PotentialCashback, monthlyCap)
		}
		month.Efficiency = percentOf(month.ActualRefund, month.Spent)
		if month.Spent == 0 {
			month.Efficiency = 0
		}
		month.Missed = clampZero(month.PotentialCashback - month.ActualRefund)
		months = append(months, month)
	}
	return months
}

// yearsActive counts started years since the first transaction, at least one.
func yearsActive(first time.Time, today time.Time) int {
	years := int(math.Ceil(today.Sub(first).Hours() / 24 / 365))
	if years < 1 {
		return 1
	}
	return years
}

func lowerTags(tags []string) map[string]bool {
	set := make(map[string]bool, len(tags))
	for _, tag := range tags {
		set[strings.ToLower(strings.TrimSpace(tag))] = true
	}
	return set
}
