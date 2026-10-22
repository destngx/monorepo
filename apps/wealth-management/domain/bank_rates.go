package domain

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// DepositAccess says how quickly an instant-access comparison deposit can
// be reached (idle cash, WM-CASH-3).
type DepositAccess string

const (
	// DepositNoTerm: a no-term deposit, withdrawable any time.
	DepositNoTerm DepositAccess = "no_term"
	// DepositOneMonth: a 1-month term deposit that rolls over monthly; an
	// early withdrawal earns the no-term rate.
	DepositOneMonth DepositAccess = "one_month"
	// DepositFlexible: a flexible savings product such as an e-wallet's
	// earning balance, spendable and withdrawable any time (D41).
	DepositFlexible DepositAccess = "flexible"
)

// Labels of the instant-access kinds.
const (
	DepositNoTermLabel   = "no-term, withdraw any time"
	DepositOneMonthLabel = "1-month term, rolls monthly; early withdrawal pays the no-term rate"
	DepositFlexibleLabel = "earning balance, spend or withdraw any time"
)

// FlexibleDepositRate is a flexible savings product as an instant-access
// option.
func FlexibleDepositRate(product FlexibleSaving) DepositRate {
	return DepositRate{
		Bank: product.Provider, Product: product.Product, ProductID: product.ID, Rate: product.Rate,
		Access: DepositFlexible, AccessLabel: DepositFlexibleLabel, UpdatedAt: product.UpdatedAt,
		Backing: product.Backing, EarningStarts: product.EarningStarts, WithdrawalMaxWorkingHours: product.WithdrawalMaxWorkingHours,
		URL: product.URL, Stale: product.Stale,
	}
}

var tenorDigits = regexp.MustCompile(`(\d+)`)

// noTermWords are the tenor texts of a no-term deposit (folded).
var noTermWords = []string{"no-term", "no term", "khong ky han", "kkh", "demand"}

// bankRateMonths is the rate's tenor in months; false when the source
// publishes none. TenorMonths wins; else the tenor text is read ("12M",
// "12 tháng", "no-term", "không kỳ hạn").
func bankRateMonths(rate BankRate) (int, bool) {
	if rate.TenorMonths != nil {
		return *rate.TenorMonths, true
	}
	text := foldAccents(strings.TrimSpace(rate.Tenor))
	if text == "" {
		return 0, false
	}
	for _, word := range noTermWords {
		if strings.Contains(text, word) {
			return 0, true
		}
	}
	if match := tenorDigits.FindString(text); match != "" {
		if months, err := strconv.Atoi(match); err == nil {
			return months, true
		}
	}
	return 0, false
}

// usableRate rejects zero, negative and implausible rates.
func usableRate(rate BankRate) bool {
	return rate.Rate > 0 && rate.Rate < accountRateCeiling
}

func depositRateOf(rate BankRate, months *int) DepositRate {
	updated := rate.UpdatedAt
	if updated == "" {
		updated = rate.Timestamp
	}
	return DepositRate{Bank: rate.Bank, Tenor: rate.Tenor, TenorMonths: months, Channel: rate.Channel, Rate: rate.Rate, UpdatedAt: updated}
}

// DepositRateFromBankRates picks the fund performance comparison rate: the
// best published 12-month rate, else the best rate of the term tenor
// nearest 12 months, else (the source publishes no tenors) the best
// published rate. No-term rows are never a 12-month comparison.
func DepositRateFromBankRates(rates []BankRate) (DepositRate, bool) {
	const untenored = math.MaxInt - 1
	var best *BankRate
	var bestMonths *int
	bestDistance := math.MaxInt
	for i := range rates {
		rate := &rates[i]
		if !usableRate(*rate) {
			continue
		}
		distance := untenored
		var monthsOf *int
		if months, ok := bankRateMonths(*rate); ok {
			if months <= 0 || rate.InstantAccess {
				continue
			}
			distance, monthsOf = abs(months-12), &months
		}
		if best == nil || distance < bestDistance || (distance == bestDistance && rate.Rate > best.Rate) {
			best, bestDistance, bestMonths = rate, distance, monthsOf
		}
	}
	if best == nil {
		return DepositRate{}, false
	}
	basis := "best published 12-month rate today"
	switch {
	case bestDistance == untenored:
		basis = "best published savings rate today (the source publishes no tenor)"
	case bestDistance != 0:
		basis = fmt.Sprintf("best published %s rate today (nearest tenor to 12 months)", tenorText(best.Tenor, bestMonths))
	}
	basis += "; historical deposit rates are not available, so today's rate stands in for the whole period"
	deposit := depositRateOf(*best, best.TenorMonths)
	if deposit.TenorMonths == nil {
		deposit.TenorMonths = bestMonths
	}
	deposit.Basis = basis
	return deposit, true
}

// InstantAccessRateFromBankRates picks the idle cash comparison rate: the
// best rate among no-term and 1-month deposits (a tie goes to no-term).
// Rows without a tenor and longer terms never count, so false means no
// instant-access rate is published.
func InstantAccessRateFromBankRates(rates []BankRate) (DepositRate, bool) {
	var best *BankRate
	var bestMonths int
	for i := range rates {
		rate := &rates[i]
		months, ok := bankRateMonths(*rate)
		if !usableRate(*rate) || !ok || months < 0 || months > 1 {
			continue
		}
		if best == nil || rate.Rate > best.Rate || (rate.Rate == best.Rate && months < bestMonths) {
			best, bestMonths = rate, months
		}
	}
	if best == nil {
		return DepositRate{}, false
	}
	months := bestMonths
	deposit := depositRateOf(*best, &months)
	deposit.Access, deposit.AccessLabel = DepositNoTerm, DepositNoTermLabel
	if months == 1 {
		deposit.Access, deposit.AccessLabel = DepositOneMonth, DepositOneMonthLabel
	}
	return deposit, true
}

// tenorText is the tenor for prose: "no-term", "12-month", else the
// source's text; empty when unknown.
func tenorText(tenor string, months *int) string {
	switch {
	case months == nil:
		return strings.TrimSpace(tenor)
	case *months == 0:
		return "no-term"
	}
	return fmt.Sprintf("%d-month", *months)
}

// TenorText is the rate's tenor for prose, empty when the source gave none.
func (rate DepositRate) TenorText() string { return tenorText(rate.Tenor, rate.TenorMonths) }

// Label is "7.20%, 12-month", or "7.20%" when the tenor is unknown (never
// an empty part).
func (rate DepositRate) Label() string {
	label := fmt.Sprintf("%.2f%%", rate.Rate)
	if tenor := rate.TenorText(); tenor != "" {
		label += ", " + tenor
	}
	return label
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
