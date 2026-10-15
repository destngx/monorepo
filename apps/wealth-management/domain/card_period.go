package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CardPeriodDetail is one counting period of a card laid out payment by
// payment, so its cashback estimate can be explained (WM-CARD-17).
type CardPeriodDetail struct {
	Card    string `json:"card"`
	Program string `json:"program"`
	// Month is the month the period starts in, as in the card's history.
	Month string `json:"month"`
	Start string `json:"start"`
	// End is inclusive.
	End string `json:"end"`
	// Open is true while the period has not ended.
	Open     bool                 `json:"open,omitempty"`
	Spent    float64              `json:"spent"`
	Cashback CashbackAmount       `json:"cashback"`
	Refund   float64              `json:"refund"`
	Status   ReconciliationStatus `json:"status"`
	// Parts are the period under each programme version in effect in it:
	// one, unless a version starts mid-period (I2).
	Parts []CardPeriodPart `json:"parts"`
}

// CardPeriodPart is a period under one programme version: its terms, each
// payment with the rule it fell under, and the caps applied.
type CardPeriodPart struct {
	VersionFrom string    `json:"versionFrom,omitempty"`
	Start       string    `json:"start"`
	End         string    `json:"end"`
	CardCap     float64   `json:"cardCap"`
	CapPeriod   CapPeriod `json:"capPeriod,omitempty"`
	// Rules describe the version's rules in words.
	Rules           []string       `json:"rules"`
	Spent           float64        `json:"spent"`
	QualifyingSpend float64        `json:"qualifyingSpend"`
	Cashback        CashbackAmount `json:"cashback"`
	Standings       []RuleStanding `json:"standings"`
	NextTier        *NextTier      `json:"nextTier,omitempty"`
	// Payments are oldest first; their cashback is before caps.
	Payments []CardTransaction `json:"payments"`
}

// ComputeCardPeriod lays out the card's period starting in month ("YYYY-MM";
// empty is the current period). Unknown cards and past months without
// payments are ErrNotFound; a closed card has no current period.
func ComputeCardPeriod(config CardConfig, accounts []Account, transactions []Transaction, cardName string, month string, today time.Time) (CardPeriodDetail, error) {
	today = dateOnly(today)
	card, bank, found := findHeldCard(config, cardName)
	if !found {
		return CardPeriodDetail{}, fmt.Errorf("%w: no card named %q", ErrNotFound, cardName)
	}
	ledger := newCardLedger(config, bank, heldCards(config, bank), card, accounts, transactions)
	detail := CardPeriodDetail{Card: card.Name, Program: ledger.program.Name, Parts: []CardPeriodPart{}}
	var currentStart, currentEnd time.Time
	if month == "" {
		if card.ClosedOn(today) {
			return CardPeriodDetail{}, fmt.Errorf("%w: %s is closed and has no current period", ErrValidation, card.Name)
		}
		version, ok := ledger.program.VersionOn(today)
		if !ok {
			return CardPeriodDetail{}, fmt.Errorf("%w: %s has no programme version in effect today", ErrNotFound, card.Name)
		}
		currentStart, currentEnd = PeriodBounds(version.Period, card.StatementDay, today)
		month = MonthKey(currentStart)
	} else if _, err := ParseMonthKey(month); err != nil {
		return CardPeriodDetail{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	detail.Month = month

	var end time.Time
	for _, group := range groupByPeriod(ledger.payments, ledger.program.VersionOn, card.StatementDay) {
		if MonthKey(group.start) != month {
			continue
		}
		result := evaluatePeriod(group)
		part := CardPeriodPart{
			Start: group.start.Format(time.DateOnly), End: group.end.AddDate(0, 0, -1).Format(time.DateOnly),
			CardCap: group.version.Cap, CapPeriod: group.version.Period, Rules: []string{},
			Spent: result.spent, QualifyingSpend: result.qualifyingSpend, Cashback: result.cashback,
			Standings: result.rules, NextTier: result.nextTier(), Payments: []CardTransaction{},
		}
		if !group.version.From.IsZero() {
			part.VersionFrom = group.version.From.Format(time.DateOnly)
		}
		for _, rule := range group.version.Rules {
			part.Rules = append(part.Rules, rule.Describe())
		}
		for i, payment := range result.payments {
			part.Payments = append(part.Payments, CardTransaction{Transaction: payment.tx, MCC: payment.mcc, Rule: result.paymentRules[i], Cashback: result.paymentCashback[i]})
		}
		sort.SliceStable(part.Payments, func(i, j int) bool {
			return part.Payments[i].Transaction.Date.Before(part.Payments[j].Transaction.Date)
		})
		if detail.Start == "" || part.Start < detail.Start {
			detail.Start = part.Start
		}
		if part.End > detail.End {
			detail.End, end = part.End, group.end
		}
		detail.Spent += part.Spent
		detail.Cashback = detail.Cashback.plus(part.Cashback)
		detail.Parts = append(detail.Parts, part)
	}
	if len(detail.Parts) == 0 {
		if currentStart.IsZero() {
			return CardPeriodDetail{}, fmt.Errorf("%w: %s has no payments in the period starting %s", ErrNotFound, card.Name, month)
		}
		// The current period before its first payment.
		detail.Start, detail.End, end = currentStart.Format(time.DateOnly), currentEnd.AddDate(0, 0, -1).Format(time.DateOnly), currentEnd
	}
	detail.Open = today.Before(end)
	detail.Status = ReconciliationOpen
	for _, history := range cardHistory(ledger, today) {
		if history.Month == month {
			detail.Refund, detail.Status = history.Refund, history.Status
		}
	}
	return detail, nil
}

// findHeldCard finds a held card by name, ignoring case, and its bank.
func findHeldCard(config CardConfig, name string) (Card, Bank, bool) {
	for _, card := range config.Cards {
		if !strings.EqualFold(strings.TrimSpace(card.Name), strings.TrimSpace(name)) {
			continue
		}
		for _, bank := range config.Banks {
			if strings.EqualFold(bank.Name, config.BankOf(card)) {
				return card, bank, true
			}
		}
	}
	return Card{}, Bank{}, false
}
