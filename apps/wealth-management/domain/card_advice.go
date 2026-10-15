package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Card advice: which held card to pay with (WM-CARD-13, N4) and which
// programme suits the spending pattern (WM-CARD-14, N5, N6).

const comparisonMonths = 12

// CardPaymentQuery describes a payment about to be made. MCC, when set, is
// a code the user knows (confirmed); otherwise the payee's mapping applies.
type CardPaymentQuery struct {
	Payee  string
	Amount float64
	MCC    *int
}

// Validate requires a positive amount and something to find the MCC by: a
// payee or the MCC itself.
func (q CardPaymentQuery) Validate() error {
	switch {
	case q.Amount <= 0:
		return fmt.Errorf("%w: amount must be positive", ErrValidation)
	case q.MCC != nil && (*q.MCC < 0 || *q.MCC > 9999):
		return fmt.Errorf("%w: MCC must be 0-9999, got %d", ErrValidation, *q.MCC)
	case q.MCC == nil && strings.TrimSpace(q.Payee) == "":
		return fmt.Errorf("%w: payee or MCC is required", ErrValidation)
	}
	return nil
}

// SuggestCards ranks held cards by what the payment would add to their
// current period's cashback: cards whose bank has room for it first, then
// by cashback, then by the confirmed part.
func SuggestCards(config CardConfig, accounts []Account, transactions []Transaction, query CardPaymentQuery, today time.Time) []CardSuggestion {
	today = dateOnly(today)
	payment := Transaction{Payee: query.Payee, Date: today, Payment: &query.Amount}
	mcc := TransactionMCC(payment, config.Merchants)
	if query.MCC != nil {
		mcc = PaymentMCC{Code: *query.MCC, Status: MCCConfirmed}
	}
	suggestions := []CardSuggestion{}
	for _, bank := range config.Banks {
		cards := heldCards(config, bank)
		billed := map[string]bool{}
		ledgers := make([]cardLedger, 0, len(cards))
		for _, card := range cards {
			ledger := newCardLedger(config, bank, cards, card, accounts, transactions)
			billed[ledger.accountName] = true
			ledgers = append(ledgers, ledger)
		}
		limit, usage := bankLimit(bank, billed, accounts)
		for _, ledger := range ledgers {
			if ledger.card.ClosedOn(today) {
				continue
			}
			before, found := ledger.currentPeriod(today)
			if !found {
				continue
			}
			after, _ := ledger.currentPeriod(today, cardPayment{index: -1, tx: payment, amount: query.Amount, mcc: mcc})
			suggestion := CardSuggestion{
				Card: ledger.card.Name, Bank: bank.Name, Program: ledger.program.Name, MCC: mcc,
				Rule:     after.paymentRules[len(after.paymentRules)-1],
				Cashback: newCashbackAmount(after.cashback.Confirmed-before.cashback.Confirmed, after.cashback.Total-before.cashback.Total),
				// A bank without a limit set has no room to check.
				FitsLimit:      limit == 0 || query.Amount <= limit-usage,
				RemainingLimit: limit - usage,
			}
			suggestions = append(suggestions, suggestion)
		}
	}
	sort.SliceStable(suggestions, func(i, j int) bool {
		a, b := suggestions[i], suggestions[j]
		if a.FitsLimit != b.FitsLimit {
			return a.FitsLimit
		}
		if a.Cashback.Total != b.Cashback.Total {
			return a.Cashback.Total > b.Cashback.Total
		}
		return a.Cashback.Confirmed > b.Cashback.Confirmed
	})
	return suggestions
}

// ComparePrograms replays the held cards' payments of the last 12 months
// under each programme's current version, held or not, and ranks them by
// cashback less the annual fee. A programme counts per its own period: the
// statement day of a card holding it, else the 1st for a programme never held.
func ComparePrograms(config CardConfig, accounts []Account, transactions []Transaction, today time.Time) ProgramComparisonReport {
	today = dateOnly(today)
	from := today.AddDate(0, -comparisonMonths, 0)
	payments := []cardPayment{}
	seen := map[int]bool{}
	for _, bank := range config.Banks {
		cards := heldCards(config, bank)
		for _, card := range cards {
			for _, payment := range newCardLedger(config, bank, cards, card, accounts, transactions).payments {
				if seen[payment.index] || !payment.tx.Date.After(from) || payment.tx.Date.After(today) {
					continue
				}
				seen[payment.index] = true
				payments = append(payments, payment)
			}
		}
	}
	report := ProgramComparisonReport{From: from.AddDate(0, 0, 1).Format(time.DateOnly), To: today.Format(time.DateOnly), Payments: len(payments), Programs: []ProgramComparison{}}
	for _, program := range config.Programs {
		version, found := program.VersionOn(today)
		if !found {
			if len(program.Versions) == 0 {
				continue
			}
			// Terms not in effect yet are still worth comparing.
			version = program.Versions[0]
		}
		// Held means held by an open card; a closed one still lends its statement day.
		statementDay, held, holder := 1, false, false
		for _, card := range config.Cards {
			if !strings.EqualFold(card.Program, program.Name) {
				continue
			}
			if !holder || !card.ClosedOn(today) {
				statementDay, holder = card.StatementDay, true
			}
			if !card.ClosedOn(today) {
				held = true
				break
			}
		}
		comparison := ProgramComparison{Program: program.Name, Bank: program.Bank, Held: held, VersionFrom: version.From.Format(time.DateOnly), AnnualFee: version.AnnualFee}
		always := func(time.Time) (ProgramVersion, bool) { return version, true }
		for _, group := range groupByPeriod(payments, always, statementDay) {
			result := evaluatePeriod(group)
			comparison.Spent += result.spent
			comparison.Cashback = comparison.Cashback.plus(result.cashback)
		}
		comparison.Net = comparison.Cashback.Total - comparison.AnnualFee
		report.Programs = append(report.Programs, comparison)
	}
	sort.SliceStable(report.Programs, func(i, j int) bool {
		return report.Programs[i].Net > report.Programs[j].Net
	})
	return report
}
