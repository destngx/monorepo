package domain

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Idle cash check (ROADMAP R6, WM-CASH-1..5): spendable cash beyond what
// must stay liquid, compared with the best published deposit rate. The
// advice is about rate only; goal money stays pooled (D31), so it never
// suggests dedicated goal accounts.

// IdleCashVerdict answers "is money sitting idle worth moving?".
type IdleCashVerdict string

const (
	// IdleCashWorthMoving: the rate gap and the extra interest a year both
	// pass the materiality thresholds.
	IdleCashWorthMoving IdleCashVerdict = "worth_moving"
	// IdleCashNotWorthIt: money is idle but the gain is too small.
	IdleCashNotWorthIt IdleCashVerdict = "not_worth_it"
	// IdleCashNothingIdle: what must stay liquid covers all spendable cash.
	IdleCashNothingIdle IdleCashVerdict = "nothing_idle"
	// IdleCashNoRate: money is idle but no published rate is available.
	IdleCashNoRate IdleCashVerdict = "no_rate"
)

// Materiality thresholds: moving money is worth it only when the best rate
// beats the current weighted rate by at least IdleCashMinRateGapPoints and
// the move earns at least IdleCashMinExtraPerYear VND a year.
const (
	IdleCashMinRateGapPoints = 0.5
	IdleCashMinExtraPerYear  = 200_000
	// accountRateCeiling rejects a parsed note rate this high or more as a
	// misreading (no deposit pays 100% a year).
	accountRateCeiling = 100
)

// IdleCashKeepLiquidRule documents what stays liquid.
const IdleCashKeepLiquidRule = "Kept liquid: unpaid bills and card payments due in the next 30 days (overdue ones too), " +
	"this month's goal set-aside (the full monthly amount the goal plan requires, not prorated), the spending buffer (spending buffer months x average " +
	"monthly expense, as the goal pool uses it) and the emergency fund target (kept liquid). The parts overlap a " +
	"little on purpose, so the idle figure errs low."

// IdleCashRateRule documents the current-rate reading.
const IdleCashRateRule = "An account's current rate is read from its note: a keyword (earn rate, interest rate, " +
	"interest, lãi suất, lãi, rate, apy, yield) followed by a number and %, case and accents ignored, decimal point or " +
	"comma (\"earn rate 4%\", \"lãi 4%\", \"rate 4.5%\", \"lãi suất 4,5%/năm\"); else 0%. The idle cash is assumed to " +
	"sit in the lowest-yield spendable accounts first, so the current rate is weighted over those portions."

// IdleCashThresholdRule documents the verdict thresholds.
var IdleCashThresholdRule = fmt.Sprintf("Worth moving when the best rate beats the current weighted rate by at least "+
	"%.1f points and the extra interest is at least %s VND a year.", IdleCashMinRateGapPoints, groupThousands(IdleCashMinExtraPerYear))

// idleCashRateBasis labels the comparison rate as a proxy.
const idleCashRateBasis = "A proxy: the best savings rate published today by the bank-rates source (one rate per bank; " +
	"the tenor is not always given). Check the bank's own terms before moving money."

// IdleCashComponents is the math: Idle = max(0, SpendableCash - KeepLiquid).
type IdleCashComponents struct {
	// SpendableCash is the liquid balance (WM-HEA-2): positive balances of
	// active, rarely used, bank and cash accounts. Unlike safe-to-spend's
	// ready money it keeps crypto and accounts marked not ready.
	SpendableCash float64 `json:"spendableCash"`
	// BillsDue and CardPayments are unpaid in the 30-day bills calendar.
	BillsDue     float64 `json:"billsDue"`
	CardPayments float64 `json:"cardPayments"`
	// GoalSetAside is this month's goal money: the goal plan's full
	// required per month (dated active goals); 0 before the plan is set up.
	GoalSetAside float64 `json:"goalSetAside"`
	// SpendingBuffer = BufferMonths x AverageMonthlyExpense (WM-GOAL-4).
	SpendingBuffer        float64 `json:"spendingBuffer"`
	BufferMonths          float64 `json:"bufferMonths"`
	AverageMonthlyExpense float64 `json:"averageMonthlyExpense"`
	// EmergencyFund is the emergency goal's target; without that goal,
	// emergency months x monthly needs (WM-GOAL-5).
	EmergencyFund     float64 `json:"emergencyFund"`
	EmergencyFromGoal bool    `json:"emergencyFromGoal"`
	KeepLiquid        float64 `json:"keepLiquid"`
}

// IdleCashAccount is a spendable cash account with its rate and the part of
// the idle cash assumed to sit in it.
type IdleCashAccount struct {
	Name         string  `json:"name"`
	Balance      float64 `json:"balance"`
	Rate         float64 `json:"rate"`
	RateFromNote bool    `json:"rateFromNote"`
	Idle         float64 `json:"idle"`
}

// IdleCash is the idle cash check.
type IdleCash struct {
	AsOf       string             `json:"asOf"`
	Idle       float64            `json:"idle"`
	Components IdleCashComponents `json:"components"`
	// Accounts are the spendable cash accounts, lowest rate first.
	Accounts []IdleCashAccount `json:"accounts"`
	// CurrentRate is the idle cash's weighted rate, percent a year.
	CurrentRate float64 `json:"currentRate"`
	// Best is the comparison rate (a proxy); nil when none is published.
	Best          *DepositRate `json:"best"`
	RatesIssue    string       `json:"ratesIssue,omitempty"`
	RateGapPoints float64      `json:"rateGapPoints"`
	// ExtraPerYear = Idle x (best - current) / 100 when positive, else 0.
	ExtraPerYear   float64         `json:"extraPerYear"`
	Verdict        IdleCashVerdict `json:"verdict"`
	Reasons        []string        `json:"reasons"`
	MinRateGap     float64         `json:"minRateGapPoints"`
	MinExtra       float64         `json:"minExtraPerYear"`
	KeepLiquidRule string          `json:"keepLiquidRule"`
	RateRule       string          `json:"rateRule"`
	ThresholdRule  string          `json:"thresholdRule"`
}

// IdleCashInput is what ComputeIdleCash needs. Plan may be nil (no goal
// plan: no goal set-aside, buffer or emergency fund kept). Rates are the published bank
// rates; RatesIssue says why they are missing.
type IdleCashInput struct {
	Accounts   []Account
	Bills      Bills
	Plan       *GoalPlan
	Rates      []BankRate
	RatesIssue string
	Today      time.Time
}

// ComputeIdleCash finds the idle cash, its current rate and what a deposit
// at the best published rate would add a year.
func ComputeIdleCash(input IdleCashInput) IdleCash {
	idle := IdleCash{
		AsOf:           dateOnly(input.Today).Format(time.DateOnly),
		Accounts:       []IdleCashAccount{},
		Reasons:        []string{},
		RatesIssue:     input.RatesIssue,
		MinRateGap:     IdleCashMinRateGapPoints,
		MinExtra:       IdleCashMinExtraPerYear,
		KeepLiquidRule: IdleCashKeepLiquidRule,
		RateRule:       IdleCashRateRule,
		ThresholdRule:  IdleCashThresholdRule,
	}
	idle.Components = keepLiquid(input)
	idle.Idle = math.Max(0, math.Round(idle.Components.SpendableCash-idle.Components.KeepLiquid))

	idle.Accounts = spendableAccounts(input.Accounts)
	left, earning := idle.Idle, 0.0
	for i := range idle.Accounts {
		part := math.Min(idle.Accounts[i].Balance, left)
		idle.Accounts[i].Idle = part
		earning += part * idle.Accounts[i].Rate
		left -= part
	}
	if idle.Idle > 0 {
		idle.CurrentRate = round2(earning / idle.Idle)
	}

	if rate, ok := DepositRateFromBankRates(input.Rates); ok {
		rate.Basis = idleCashRateBasis
		idle.Best = &rate
		idle.RateGapPoints = round2(rate.Rate - idle.CurrentRate)
		idle.ExtraPerYear = math.Max(0, math.Round(idle.Idle*(rate.Rate-idle.CurrentRate)/100))
	} else if idle.RatesIssue == "" {
		idle.RatesIssue = "no bank published a savings rate"
	}
	idle.Verdict, idle.Reasons = idleVerdict(idle)
	return idle
}

func keepLiquid(input IdleCashInput) IdleCashComponents {
	c := IdleCashComponents{SpendableCash: liquidBalance(input.Accounts)}
	for _, o := range input.Bills.Calendar.Occurrences {
		if o.IsIncome() || o.Status == BillPaid {
			continue
		}
		if o.Source == BillFromCard {
			c.CardPayments += o.Remaining
		} else {
			c.BillsDue += o.Remaining
		}
	}
	if plan := input.Plan; plan != nil {
		if !plan.SetupNeeded {
			c.GoalSetAside = math.Round(plan.Check.RequiredPerMonth)
		}
		c.SpendingBuffer = plan.Pool.SpendingBuffer
		c.BufferMonths = plan.Pool.BufferMonths
		c.AverageMonthlyExpense = plan.Pool.AverageMonthlyExpense
		c.EmergencyFund = math.Round(plan.Settings.EmergencyMonths * plan.Needs.PerMonth)
		for _, goal := range plan.Goals {
			if goal.EmergencyFund {
				c.EmergencyFund, c.EmergencyFromGoal = goal.TargetAmount, true
				break
			}
		}
	}
	c.KeepLiquid = math.Round(c.BillsDue + c.CardPayments + c.GoalSetAside + c.SpendingBuffer + c.EmergencyFund)
	return c
}

// spendableAccounts lists the accounts of the liquid balance (positive
// balances of active, rarely used, bank and cash accounts), lowest rate
// first, then the larger balance, then by name.
func spendableAccounts(accounts []Account) []IdleCashAccount {
	out := []IdleCashAccount{}
	for _, account := range accounts {
		if account.Balance <= 0 || liquidBalance([]Account{account}) == 0 {
			continue
		}
		entry := IdleCashAccount{Name: account.Name, Balance: account.Balance}
		if account.Note != nil {
			entry.Rate, entry.RateFromNote = AccountRateFromNote(*account.Note)
		}
		out = append(out, entry)
	}
	slices.SortStableFunc(out, func(a, b IdleCashAccount) int {
		switch {
		case a.Rate != b.Rate:
			return cmpFloat(a.Rate, b.Rate)
		case a.Balance != b.Balance:
			return cmpFloat(b.Balance, a.Balance)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return out
}

func cmpFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func idleVerdict(idle IdleCash) (IdleCashVerdict, []string) {
	if idle.Idle <= 0 {
		return IdleCashNothingIdle, []string{"What must stay liquid covers all spendable cash, so nothing is idle."}
	}
	if idle.Best == nil {
		return IdleCashNoRate, []string{"No published deposit rate to compare with: " + idle.RatesIssue + "."}
	}
	reasons := []string{}
	if idle.RateGapPoints < IdleCashMinRateGapPoints {
		reasons = append(reasons, fmt.Sprintf("The best published rate (%.2f%%) is only %.2f points above the current rate (%.2f%%), under the %.1f-point threshold.",
			idle.Best.Rate, idle.RateGapPoints, idle.CurrentRate, IdleCashMinRateGapPoints))
	}
	if idle.ExtraPerYear < IdleCashMinExtraPerYear {
		reasons = append(reasons, fmt.Sprintf("A deposit would add about %s VND a year, under the %s VND threshold.",
			groupThousands(idle.ExtraPerYear), groupThousands(IdleCashMinExtraPerYear)))
	}
	if len(reasons) > 0 {
		return IdleCashNotWorthIt, reasons
	}
	return IdleCashWorthMoving, []string{fmt.Sprintf("About %s VND is idle at %.2f%%; a deposit at %.2f%% (%s) would add about %s VND a year.",
		groupThousands(idle.Idle), idle.CurrentRate, idle.Best.Rate, idle.Best.Bank, groupThousands(idle.ExtraPerYear))}
}

// accountRatePattern reads a rate from a note folded by foldAccents.
var accountRatePattern = regexp.MustCompile(`\b(?:earn rate|interest rate|interest|lai suat|lai|rate|apy|yield)\s*[:=]?\s*(\d+(?:[.,]\d+)?)\s*%`)

// AccountRateFromNote reads an account's rate (percent a year) from its
// note, per IdleCashRateRule; false when the note states none.
func AccountRateFromNote(note string) (float64, bool) {
	match := accountRatePattern.FindStringSubmatch(foldAccents(note))
	if match == nil {
		return 0, false
	}
	rate, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
	if err != nil || rate < 0 || rate >= accountRateCeiling {
		return 0, false
	}
	return rate, true
}

// foldAccents lowercases and strips Vietnamese diacritics (đ included) but,
// unlike FoldText, keeps punctuation so "4,5%" survives.
func foldAccents(value string) string {
	stripped, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), value)
	if err != nil {
		stripped = value
	}
	return strings.NewReplacer("đ", "d", "Đ", "d").Replace(strings.ToLower(stripped))
}

// groupThousands formats a VND amount the Vietnamese way: 1.234.567.
func groupThousands(value float64) string {
	digits := strconv.FormatInt(int64(math.Round(math.Abs(value))), 10)
	var b strings.Builder
	if value < 0 {
		b.WriteByte('-')
	}
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// IdleCashFacts is the nameless summary for the AI briefing.
type IdleCashFacts struct {
	Idle                float64         `json:"idle"`
	SpendableCash       float64         `json:"spendableCash"`
	KeepLiquid          float64         `json:"keepLiquid"`
	CurrentRate         float64         `json:"currentRate"`
	BestRate            float64         `json:"bestRate"`
	BestBank            string          `json:"bestBank,omitempty"`
	ExtraPerYear        float64         `json:"extraPerYear"`
	Verdict             IdleCashVerdict `json:"verdict"`
	AccountsHoldingIdle int             `json:"accountsHoldingIdle"`
}

// Facts summarises the check without account names (the bank of the
// published rate is public data).
func (idle IdleCash) Facts() IdleCashFacts {
	facts := IdleCashFacts{
		Idle: idle.Idle, SpendableCash: idle.Components.SpendableCash, KeepLiquid: idle.Components.KeepLiquid,
		CurrentRate: idle.CurrentRate, ExtraPerYear: idle.ExtraPerYear, Verdict: idle.Verdict,
	}
	if idle.Best != nil {
		facts.BestRate, facts.BestBank = idle.Best.Rate, idle.Best.Bank
	}
	for _, account := range idle.Accounts {
		if account.Idle > 0 {
			facts.AccountsHoldingIdle++
		}
	}
	return facts
}
