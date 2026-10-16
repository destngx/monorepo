package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Card reports (WM-CARD-1, 4, 10-12, 16): per bank, the shared limit and
// usage; per held card, the current period's standing, the reconciled
// history and the period's transactions.

const cardHistoryMonths = 36

// cardLedger is a held card with its billing account and its transactions.
type cardLedger struct {
	card        Card
	program     Program
	account     Account
	accountName string
	members     []Transaction
	payments    []cardPayment
}

// heldCards returns the bank's held cards, in sheet order.
func heldCards(config CardConfig, bank Bank) []Card {
	cards := []Card{}
	for _, card := range config.Cards {
		if strings.EqualFold(config.BankOf(card), bank.Name) {
			cards = append(cards, card)
		}
	}
	return cards
}

func newCardLedger(config CardConfig, bank Bank, bankCards []Card, card Card, accounts []Account, transactions []Transaction) cardLedger {
	program, _ := config.Program(card.Program)
	ledger := cardLedger{card: card, program: program, accountName: bank.SharedAccount}
	if account, found := findCardAccount(bank, card, accounts); found {
		ledger.account, ledger.accountName = account, account.Name
	}
	for i, t := range transactions {
		if !t.Usable() || !belongsToCard(bank, bankCards, card, ledger.accountName, t) {
			continue
		}
		ledger.members = append(ledger.members, t)
		if amount := spendAmount(t); amount > 0 {
			ledger.payments = append(ledger.payments, cardPayment{index: i, tx: t, amount: amount, mcc: TransactionMCC(t, config.Merchants)})
		}
	}
	return ledger
}

// currentPeriod evaluates the period containing today under the version in
// effect today, plus extra payments; ok is false when no version is.
func (l cardLedger) currentPeriod(today time.Time, extra ...cardPayment) (periodResult, bool) {
	version, found := l.program.VersionOn(today)
	if !found {
		return periodResult{}, false
	}
	start, end := PeriodBounds(version.Period, l.card.StatementDay, today)
	group := periodGroup{version: version, start: start, end: end}
	for _, payment := range l.payments {
		inPeriod := !payment.tx.Date.Before(start) && payment.tx.Date.Before(end)
		if current, ok := l.program.VersionOn(payment.tx.Date); inPeriod && ok && current.From.Equal(version.From) {
			group.payments = append(group.payments, payment)
		}
	}
	group.payments = append(group.payments, extra...)
	return evaluatePeriod(group), true
}

// ComputeCardReports evaluates every held card, grouped by bank.
func ComputeCardReports(config CardConfig, accounts []Account, transactions []Transaction, today time.Time) []CardBankReport {
	today = dateOnly(today)
	reports := make([]CardBankReport, 0, len(config.Banks))
	for _, bank := range config.Banks {
		cards := heldCards(config, bank)
		if len(cards) == 0 {
			continue
		}
		report := CardBankReport{Name: bank.Name, SharedAccount: bank.SharedAccount, Cards: make([]CardReport, 0, len(cards))}
		billed := map[string]bool{}
		for _, card := range cards {
			ledger := newCardLedger(config, bank, cards, card, accounts, transactions)
			billed[ledger.accountName] = true
			cardReport := computeCard(ledger, today)
			report.Cashback = report.Cashback.plus(cardReport.Standing.Cashback)
			report.RefundYTD += cardReport.RefundYTD
			report.FeesPaidYTD += cardReport.FeesPaidYTD
			report.Cards = append(report.Cards, cardReport)
		}
		report.Limit, report.Usage = bankLimit(bank, billed, accounts)
		report.Utilization = percentOf(report.Usage, report.Limit)
		report.UtilizationBand = UtilizationBandFor(report.Utilization)
		report.RemainingLimit = report.Limit - report.Usage
		report.NetEarnYTD = report.RefundYTD - report.FeesPaidYTD
		reports = append(reports, report)
	}
	return reports
}

// bankLimit is the bank's limit (the shared account's credit limit from
// Accounts C, else the reference limit) and its usage: what the accounts its cards bill to owe,
// money paid in ahead counting as room (C9).
func bankLimit(bank Bank, billed map[string]bool, accounts []Account) (float64, float64) {
	limit, usage := bank.ReferenceLimit, 0.0
	for _, account := range accounts {
		if account.Type != AccountTypeNegativeActive || !billed[account.Name] {
			continue
		}
		usage += owedAmount(account.ClearedBalance)
		if account.Name == bank.SharedAccount && account.CreditLimit != nil && *account.CreditLimit > 0 {
			limit = *account.CreditLimit
		}
	}
	return limit, usage
}

// findCardAccount prefers a liability account named after the card, else the bank's shared account.
func findCardAccount(bank Bank, card Card, accounts []Account) (Account, bool) {
	keyword := strings.ToLower(card.Keyword)
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
func belongsToCard(bank Bank, bankCards []Card, card Card, accountName string, t Transaction) bool {
	hasTag := func(word string) bool {
		for _, tag := range t.Tags {
			if strings.Contains(strings.ToLower(tag), word) {
				return true
			}
		}
		return false
	}
	if spendAmount(t) > 0 && card.ClosedOn(t.Date) {
		return false
	}
	if hasTag(strings.ToLower(card.Keyword)) {
		return true
	}
	if t.AccountName != accountName {
		return false
	}
	for _, other := range bankCards {
		if !strings.EqualFold(other.Keyword, card.Keyword) && hasTag(strings.ToLower(other.Keyword)) {
			return false
		}
	}
	if accountName == bank.SharedAccount {
		return card.Default
	}
	return true
}

func computeCard(ledger cardLedger, today time.Time) CardReport {
	card := ledger.card
	report := CardReport{
		Name: card.Name, Bank: ledger.program.Bank, Program: ledger.program.Name, Keyword: card.Keyword,
		AccountName: ledger.accountName, Expiry: card.Expiry, Last4: card.Last4, Rules: []CashbackRule{},
		Cycle:        ComputeCardCycle(card.StatementDay, card.DueDay, today),
		Transactions: []CardTransaction{},
	}
	periodStart, periodEnd := PeriodBounds(CapPeriodCycle, card.StatementDay, today)
	standing := PeriodStanding{Rules: []RuleStanding{}}
	closed := card.ClosedOn(today)
	if closed {
		// A closed card has no current period; it shows its last terms.
		report.ClosedFrom = card.ClosedFrom.Format(time.DateOnly)
		if version, found := ledger.program.VersionOn(card.ClosedFrom.AddDate(0, 0, -1)); found {
			report.VersionFrom = version.From.Format(time.DateOnly)
			report.AnnualFee, report.Cap, report.CapPeriod, report.Rules = version.AnnualFee, version.Cap, version.Period, version.Rules
		}
	} else if current, found := ledger.currentPeriod(today); found {
		version := current.version
		report.VersionFrom = version.From.Format(time.DateOnly)
		report.AnnualFee, report.Cap, report.CapPeriod, report.Rules = version.AnnualFee, version.Cap, version.Period, version.Rules
		periodStart, periodEnd = current.start, current.end
		standing = PeriodStanding{
			Spent: current.spent, QualifyingSpend: current.qualifyingSpend, Cashback: current.cashback,
			Cap: version.Cap, Rules: current.rules, NextTier: current.nextTier(),
			PaymentCount: len(current.payments), UnconfirmedPayments: current.unconfirmed,
		}
		if version.Cap > 0 {
			standing.CapLeft = math.Max(0, version.Cap-current.cashback.Total)
		}
		for i, payment := range current.payments {
			report.Transactions = append(report.Transactions, CardTransaction{Transaction: payment.tx, MCC: payment.mcc, Rule: current.paymentRules[i], Cashback: current.paymentCashback[i]})
		}
	}
	standing.Start = periodStart.Format(time.DateOnly)
	standing.End = periodEnd.AddDate(0, 0, -1).Format(time.DateOnly)
	report.Standing = standing
	if !closed {
		report.Statement = ledger.statement(today, PayByDate(today))
		report.Cycle.PayByUrgent = report.Statement.Amount > 0 && report.Cycle.DaysUntilPayBy <= dueUrgentDays
	}

	firstSeen := today
	for _, t := range ledger.members {
		if t.Date.Before(firstSeen) {
			firstSeen = t.Date
		}
		// Payments came from the period evaluation; list the rest too.
		inPeriod := !t.Date.Before(periodStart) && t.Date.Before(periodEnd)
		if inPeriod && !closed && spendAmount(t) == 0 {
			report.Transactions = append(report.Transactions, CardTransaction{Transaction: t, MCC: PaymentMCC{Status: MCCUnknown}})
		}
	}
	sort.SliceStable(report.Transactions, func(i, j int) bool {
		return report.Transactions[i].Transaction.Date.After(report.Transactions[j].Transaction.Date)
	})

	report.History = cardHistory(ledger, today)
	for _, t := range ledger.members {
		if IsCashbackRefund(t) && EffectiveDate(t).Year() == today.Year() {
			report.RefundYTD += t.DepositAmount()
		}
		if IsCardFee(t) && t.Date.Year() == today.Year() {
			report.FeesPaidYTD += t.PaymentAmount() - t.DepositAmount()
		}
	}
	report.NetEarnYTD = report.RefundYTD - report.FeesPaidYTD
	feesUntil := today
	if closed {
		feesUntil = *card.ClosedFrom
	}
	report.LifetimeFees = float64(yearsActive(firstSeen, feesUntil)) * report.AnnualFee
	return report
}

// statement estimates the statement the pay-by date settles (WM-CARD-15).
func (l cardLedger) statement(today, payBy time.Time) *CardStatement {
	start, end := StatementBounds(l.card.StatementDay, payBy)
	statement := &CardStatement{Start: start.Format(time.DateOnly), End: end.AddDate(0, 0, -1).Format(time.DateOnly), Open: today.Before(end)}
	for _, t := range l.members {
		if t.Date.Before(start) || !t.Date.Before(end) {
			continue
		}
		switch {
		case IsCardFee(t):
			statement.Fees += t.PaymentAmount() - t.DepositAmount()
		case IsCashbackRefund(t):
			statement.Cashback += t.DepositAmount()
			statement.Refunds = append(statement.Refunds, StatementRefund{Date: t.Date.Format(time.DateOnly), For: MonthKey(EffectiveDate(t)), Amount: t.DepositAmount()})
		default:
			statement.Spent += spendAmount(t)
		}
	}
	statement.Amount = clampZero(statement.Spent + statement.Fees - statement.Cashback)
	return statement
}

// cardHistory reconciles each period's cashback with the refund received
// for it (WM-CARD-11), newest first.
func cardHistory(ledger cardLedger, today time.Time) []CardPeriodHistory {
	type entry struct {
		history  CardPeriodHistory
		end      time.Time
		payments int
	}
	entries := map[string]*entry{}
	at := func(month string) *entry {
		if entries[month] == nil {
			entries[month] = &entry{history: CardPeriodHistory{Month: month}}
		}
		return entries[month]
	}
	for _, group := range groupByPeriod(ledger.payments, ledger.program.VersionOn, ledger.card.StatementDay) {
		result := evaluatePeriod(group)
		e := at(MonthKey(group.start))
		start, last := group.start.Format(time.DateOnly), group.end.AddDate(0, 0, -1).Format(time.DateOnly)
		if e.history.Start == "" || start < e.history.Start {
			e.history.Start = start
		}
		if last > e.history.End {
			e.history.End, e.end = last, group.end
		}
		e.history.Spent += result.spent
		e.history.Cashback = e.history.Cashback.plus(result.cashback)
		e.payments += len(group.payments)
	}
	for _, t := range ledger.members {
		if IsCashbackRefund(t) {
			at(MonthKey(EffectiveDate(t))).history.Refund += t.DepositAmount()
		}
	}
	months := make([]string, 0, len(entries))
	for month := range entries {
		months = append(months, month)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(months)))
	if len(months) > cardHistoryMonths {
		months = months[:cardHistoryMonths]
	}
	history := make([]CardPeriodHistory, 0, len(months))
	for _, month := range months {
		e := entries[month]
		h := e.history
		h.Status = reconcile(h, e.end, e.payments, today)
		if h.Spent > 0 {
			h.Efficiency = percentOf(h.Refund, h.Spent)
		}
		h.Missed = clampZero(h.Cashback.Total - h.Refund)
		history = append(history, h)
	}
	return history
}

// reconcile compares a period's refund with its cashback (C6). Rounding
// allows 1 VND per payment either way. end is the exclusive end of the
// period; zero for a month with a refund but no payments.
func reconcile(h CardPeriodHistory, end time.Time, payments int, today time.Time) ReconciliationStatus {
	if !end.IsZero() && today.Before(end) {
		return ReconciliationOpen
	}
	tolerance := math.Max(1, float64(payments))
	switch {
	case h.Refund == 0 && h.Cashback.Total > 0:
		return ReconciliationAwaiting
	case h.Refund < h.Cashback.Confirmed-tolerance:
		return ReconciliationBelow
	case h.Refund > h.Cashback.Total+tolerance:
		return ReconciliationAbove
	default:
		return ReconciliationMatch
	}
}

// yearsActive counts started years since the first transaction, at least one.
func yearsActive(first time.Time, today time.Time) int {
	years := int(math.Ceil(today.Sub(first).Hours() / 24 / 365))
	if years < 1 {
		return 1
	}
	return years
}
