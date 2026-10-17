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

// Idle cash check (ROADMAP R6, WM-CASH-1..5, D34): ready money beyond the
// day-to-day float and what must stay liquid, compared only with options
// that keep the money within reach (no-term or 1-month deposits). The
// advice is about rate only; goal money stays pooled (D31), so it never
// suggests dedicated goal accounts. Emergency accounts hold the emergency
// fund: they cover its target first and are never idle.

// IdleCashVerdict answers "is money sitting idle worth moving?".
type IdleCashVerdict string

const (
	// IdleCashWorthMoving: the rate gap and the extra interest a year both
	// pass the materiality thresholds.
	IdleCashWorthMoving IdleCashVerdict = "worth_moving"
	// IdleCashNotWorthIt: money is idle but the gain is too small.
	IdleCashNotWorthIt IdleCashVerdict = "not_worth_it"
	// IdleCashNothingIdle: the float and what must stay liquid cover all
	// ready money.
	IdleCashNothingIdle IdleCashVerdict = "nothing_idle"
	// IdleCashNoRate: money is idle but no instant-access rate (no-term or
	// 1-month) is published.
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
	// idleLadderMinPart is the smallest part worth its own 1-month deposit;
	// the idle cash is split into 2 or 3 parts only when each is this big.
	idleLadderMinPart = 10_000_000
	idleLadderMaxPart = 3
	// idleLadderCycleDays is the 1-month term the parts are spread over.
	idleLadderCycleDays = 30
)

// IdleCashKeepLiquidRule documents what stays liquid.
const IdleCashKeepLiquidRule = "Ready money is safe to spend's (bank, cash, active and rarely used accounts; not crypto, " +
	"not accounts marked not ready to use) without the emergency accounts. Kept liquid: unpaid bills and card payments " +
	"due in the next 30 days (overdue ones too), this month's goal set-aside (the full monthly amount the goal plan " +
	"requires, not prorated) and the part of the emergency fund target the emergency accounts do not cover. The " +
	"day-to-day float (idleCash.dayToDayFloat in the Preferences tab; unset = the spending buffer, spending buffer " +
	"months x average monthly expense) stays ready too. Idle = ready money - kept liquid - float."

// IdleCashRateRule documents the current-rate reading.
const IdleCashRateRule = "An account's current rate is read from its note: a keyword (earn rate, interest rate, " +
	"interest, lãi suất, lãi, rate, apy, yield) followed by a number and %, case and accents ignored, decimal point or " +
	"comma (\"earn rate 4%\", \"lãi 4%\", \"rate 4.5%\", \"lãi suất 4,5%/năm\"); else 0%. The idle cash is assumed to " +
	"sit in the lowest-yield ready accounts first, so the current rate is weighted over those portions."

// IdleCashThresholdRule documents the verdict thresholds.
var IdleCashThresholdRule = fmt.Sprintf("Worth moving when the best instant-access rate beats the current weighted rate by at "+
	"least %.1f points and the extra interest is at least %s VND a year. The same thresholds decide whether the "+
	"emergency fund has a better instant-access place.", IdleCashMinRateGapPoints, groupThousands(IdleCashMinExtraPerYear))

// idleCashRateBasis labels the comparison rate as a proxy.
const idleCashRateBasis = "A proxy: the best no-term or 1-month savings rate published today by the bank-rates source " +
	"(longer terms and rates without a tenor are never used, so the money stays within reach). Check the bank's own " +
	"terms before moving money."

// IdleCashComponents is the math:
// Idle = max(0, ReadyMoney - KeepLiquid - DayToDayFloat).
type IdleCashComponents struct {
	// ReadyMoney is safe-to-spend's ready money (WM-BILL-5, D33) without
	// the emergency accounts.
	ReadyMoney float64 `json:"readyMoney"`
	// BillsDue and CardPayments are unpaid in the 30-day bills calendar.
	BillsDue     float64 `json:"billsDue"`
	CardPayments float64 `json:"cardPayments"`
	// GoalSetAside is this month's goal money: the goal plan's full
	// required per month (dated active goals); 0 before the plan is set up.
	GoalSetAside float64 `json:"goalSetAside"`
	// EmergencyFund is the emergency goal's target; without that goal,
	// emergency months x monthly needs (WM-GOAL-5).
	EmergencyFund     float64 `json:"emergencyFund"`
	EmergencyFromGoal bool    `json:"emergencyFromGoal"`
	// EmergencyInAccounts is the balance of the emergency accounts;
	// EmergencyCovered = min(EmergencyInAccounts, EmergencyFund) and
	// EmergencyFromReady = EmergencyFund - EmergencyCovered, kept liquid.
	EmergencyInAccounts float64 `json:"emergencyInAccounts"`
	EmergencyCovered    float64 `json:"emergencyCovered"`
	EmergencyFromReady  float64 `json:"emergencyFromReady"`
	// KeepLiquid = BillsDue + CardPayments + GoalSetAside + EmergencyFromReady.
	KeepLiquid float64 `json:"keepLiquid"`
	// DayToDayFloat is the money kept ready for everyday spending: the
	// preference when set, else BufferMonths x AverageMonthlyExpense.
	DayToDayFloat         float64 `json:"dayToDayFloat"`
	FloatFromPreference   bool    `json:"floatFromPreference"`
	BufferMonths          float64 `json:"bufferMonths"`
	AverageMonthlyExpense float64 `json:"averageMonthlyExpense"`
}

// IdleCashAccount is a ready (or emergency) account with its rate and the
// part of the idle cash assumed to sit in it.
type IdleCashAccount struct {
	Name         string  `json:"name"`
	Balance      float64 `json:"balance"`
	Rate         float64 `json:"rate"`
	RateFromNote bool    `json:"rateFromNote"`
	Idle         float64 `json:"idle"`
}

// IdleCashLadder splits the idle cash into rolling 1-month deposits opened
// SpacingDays apart, so one part is always close to maturity.
type IdleCashLadder struct {
	Parts       int     `json:"parts"`
	PartAmount  float64 `json:"partAmount"`
	SpacingDays int     `json:"spacingDays"`
}

// IdleCashEmergency is the emergency fund held in emergency accounts and
// whether a better instant-access place exists for it.
type IdleCashEmergency struct {
	// Accounts are the emergency accounts (liquid, not crypto), listed
	// whatever the balance so they can be unmarked.
	Accounts []IdleCashAccount `json:"accounts"`
	Balance  float64           `json:"balance"`
	Target   float64           `json:"target"`
	// CurrentRate is weighted over the emergency balances.
	CurrentRate   float64 `json:"currentRate"`
	RateGapPoints float64 `json:"rateGapPoints"`
	// ExtraPerYear = Balance x (best - current) / 100 when positive.
	ExtraPerYear float64 `json:"extraPerYear"`
	// BetterRate: the best instant-access rate beats the emergency
	// accounts by the materiality thresholds.
	BetterRate bool `json:"betterRate"`
	// Settings are the saved names with the ETag a change must send.
	Settings IdleCashSettings `json:"settings"`
	// Unknown are marked names that match no account (ignored).
	Unknown []string `json:"unknown"`
}

// IdleCash is the idle cash check.
type IdleCash struct {
	AsOf       string             `json:"asOf"`
	Idle       float64            `json:"idle"`
	Components IdleCashComponents `json:"components"`
	// Accounts are the everyday ready accounts, lowest rate first.
	Accounts []IdleCashAccount `json:"accounts"`
	// Excluded are the crypto and not-ready accounts left out (never idle).
	Excluded  []ExcludedAccount `json:"excluded"`
	Emergency IdleCashEmergency `json:"emergency"`
	// CurrentRate is the idle cash's weighted rate, percent a year.
	CurrentRate float64 `json:"currentRate"`
	// Best is the best instant-access rate (no-term or 1-month, a proxy);
	// nil when none is published.
	Best       *DepositRate `json:"best"`
	RatesIssue string       `json:"ratesIssue,omitempty"`
	// Ladder is set when Best is a 1-month deposit and the idle cash is
	// big enough to split.
	Ladder        *IdleCashLadder `json:"ladder"`
	RateGapPoints float64         `json:"rateGapPoints"`
	// ExtraPerYear = Idle x (best - current) / 100 when positive, else 0.
	ExtraPerYear   float64         `json:"extraPerYear"`
	Verdict        IdleCashVerdict `json:"verdict"`
	Reasons        []string        `json:"reasons"`
	Issues         []string        `json:"issues"`
	MinRateGap     float64         `json:"minRateGapPoints"`
	MinExtra       float64         `json:"minExtraPerYear"`
	KeepLiquidRule string          `json:"keepLiquidRule"`
	RateRule       string          `json:"rateRule"`
	ThresholdRule  string          `json:"thresholdRule"`
}

// IdleCashInput is what ComputeIdleCash needs. Plan may be nil (no goal
// plan: no goal set-aside, float or emergency fund kept). The accounts
// marked not ready come from Bills.SafeToSpend.Readiness (D33). Rates are
// the published bank rates; RatesIssue says why they are missing.
type IdleCashInput struct {
	Accounts   []Account
	Bills      Bills
	Plan       *GoalPlan
	Settings   IdleCashSettings
	Rates      []BankRate
	RatesIssue string
	Today      time.Time
}

// ComputeIdleCash finds the idle cash, its current rate and what the best
// instant-access deposit would add a year.
func ComputeIdleCash(input IdleCashInput) IdleCash {
	idle := IdleCash{
		AsOf:           dateOnly(input.Today).Format(time.DateOnly),
		Accounts:       []IdleCashAccount{},
		Excluded:       []ExcludedAccount{},
		Reasons:        []string{},
		Issues:         []string{},
		RatesIssue:     input.RatesIssue,
		MinRateGap:     IdleCashMinRateGapPoints,
		MinExtra:       IdleCashMinExtraPerYear,
		KeepLiquidRule: IdleCashKeepLiquidRule,
		RateRule:       IdleCashRateRule,
		ThresholdRule:  IdleCashThresholdRule,
	}
	idle.sortAccounts(input)
	idle.Components = idleComponents(input, idle.Accounts, idle.Emergency.Balance)
	idle.Emergency.Target = idle.Components.EmergencyFund
	c := idle.Components
	idle.Idle = math.Max(0, math.Round(c.ReadyMoney-c.KeepLiquid-c.DayToDayFloat))

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

	if rate, ok := InstantAccessRateFromBankRates(input.Rates); ok {
		rate.Basis = idleCashRateBasis
		idle.Best = &rate
		idle.RateGapPoints = round2(rate.Rate - idle.CurrentRate)
		idle.ExtraPerYear = math.Max(0, math.Round(idle.Idle*(rate.Rate-idle.CurrentRate)/100))
		idle.Ladder = idleLadder(idle.Idle, rate)
		idle.Emergency.compare(rate)
	} else {
		idle.RatesIssue = noInstantRateIssue(input)
	}
	idle.Verdict, idle.Reasons = idleVerdict(idle)
	return idle
}

// sortAccounts splits the accounts into everyday ready money (the
// candidates for idle cash), emergency accounts and the accounts left out
// (crypto, not ready), using safe-to-spend's ready money rule (D33).
func (idle *IdleCash) sortAccounts(input IdleCashInput) {
	marked := func(names []string, name string) bool {
		return slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(strings.TrimSpace(n), strings.TrimSpace(name)) })
	}
	emergencyNames := input.Settings.EmergencyAccounts
	idle.Emergency = IdleCashEmergency{
		Accounts: []IdleCashAccount{}, Unknown: []string{},
		Settings: IdleCashSettings{DayToDayFloat: input.Settings.DayToDayFloat, EmergencyAccounts: append([]string{}, emergencyNames...), ETag: input.Settings.ETag},
	}
	// An emergency account is set aside whatever its not-ready mark.
	notReady := slices.DeleteFunc(slices.Clone(input.Bills.SafeToSpend.Readiness.NotReadyAccounts), func(name string) bool { return marked(emergencyNames, name) })
	var ready SafeToSpend
	readyMoney(&ready, input.Accounts, notReady)
	idle.Excluded = append(idle.Excluded, ready.ExcludedAccounts...)
	byName := map[string]Account{}
	for _, account := range input.Accounts {
		byName[strings.ToLower(strings.TrimSpace(account.Name))] = account
	}
	for _, spendable := range ready.ReadyAccounts {
		if !marked(emergencyNames, spendable.Name) {
			idle.Accounts = append(idle.Accounts, accountWithRate(byName[strings.ToLower(strings.TrimSpace(spendable.Name))]))
		}
	}
	sortByRate(idle.Accounts)

	for _, name := range emergencyNames {
		account, ok := byName[strings.ToLower(strings.TrimSpace(name))]
		switch {
		case !ok:
			idle.Emergency.Unknown = append(idle.Emergency.Unknown, name)
			idle.Issues = append(idle.Issues, fmt.Sprintf("Preferences: %s names %q, which is not an account (renamed or deleted?); it is ignored. Unmark it to remove it.", PrefIdleCashEmergencyAccounts, name))
		case isCryptoAccount(account):
			idle.Issues = append(idle.Issues, fmt.Sprintf("%s is marked as an emergency account but is crypto, which never counts as emergency money (D32); it is ignored.", account.Name))
		case !isReadyType(account):
			idle.Issues = append(idle.Issues, fmt.Sprintf("%s is marked as an emergency account but is not a bank, cash, active or rarely used account; it is ignored.", account.Name))
		default:
			entry := accountWithRate(account)
			idle.Emergency.Accounts = append(idle.Emergency.Accounts, entry)
			if entry.Balance > 0 {
				idle.Emergency.Balance += entry.Balance
			}
		}
	}
	sortByRate(idle.Emergency.Accounts)
	weighted := 0.0
	for _, account := range idle.Emergency.Accounts {
		weighted += math.Max(0, account.Balance) * account.Rate
	}
	if idle.Emergency.Balance > 0 {
		idle.Emergency.CurrentRate = round2(weighted / idle.Emergency.Balance)
	}
}

// isReadyType: the account types that can be ready money.
func isReadyType(account Account) bool {
	switch account.Type {
	case AccountTypeActiveUse, AccountTypeRarelyUse, AccountTypeBank, AccountTypeCash:
		return true
	}
	return false
}

func accountWithRate(account Account) IdleCashAccount {
	entry := IdleCashAccount{Name: account.Name, Balance: account.Balance}
	if account.Note != nil {
		entry.Rate, entry.RateFromNote = AccountRateFromNote(*account.Note)
	}
	return entry
}

// sortByRate lists the accounts lowest rate first, then the larger
// balance, then by name.
func sortByRate(accounts []IdleCashAccount) {
	slices.SortStableFunc(accounts, func(a, b IdleCashAccount) int {
		switch {
		case a.Rate != b.Rate:
			return cmpFloat(a.Rate, b.Rate)
		case a.Balance != b.Balance:
			return cmpFloat(b.Balance, a.Balance)
		}
		return strings.Compare(a.Name, b.Name)
	})
}

func idleComponents(input IdleCashInput, ready []IdleCashAccount, emergencyBalance float64) IdleCashComponents {
	c := IdleCashComponents{EmergencyInAccounts: emergencyBalance}
	for _, account := range ready {
		c.ReadyMoney += account.Balance
	}
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
		c.BufferMonths = plan.Pool.BufferMonths
		c.AverageMonthlyExpense = plan.Pool.AverageMonthlyExpense
		c.DayToDayFloat = math.Round(plan.Pool.SpendingBuffer)
		c.EmergencyFund = math.Round(plan.Settings.EmergencyMonths * plan.Needs.PerMonth)
		for _, goal := range plan.Goals {
			if goal.EmergencyFund {
				c.EmergencyFund, c.EmergencyFromGoal = goal.TargetAmount, true
				break
			}
		}
	}
	if input.Settings.DayToDayFloat != nil {
		c.DayToDayFloat, c.FloatFromPreference = math.Round(*input.Settings.DayToDayFloat), true
	}
	c.EmergencyCovered = math.Min(c.EmergencyInAccounts, c.EmergencyFund)
	c.EmergencyFromReady = math.Max(0, c.EmergencyFund-c.EmergencyCovered)
	c.ReadyMoney = math.Round(c.ReadyMoney)
	c.KeepLiquid = math.Round(c.BillsDue + c.CardPayments + c.GoalSetAside + c.EmergencyFromReady)
	return c
}

// idleLadder splits idle cash for a 1-month best rate into 2-3 parts of
// at least idleLadderMinPart, opened a few days apart.
func idleLadder(idleAmount float64, best DepositRate) *IdleCashLadder {
	if best.Access != DepositOneMonth {
		return nil
	}
	parts := min(idleLadderMaxPart, int(idleAmount/idleLadderMinPart))
	if parts < 2 {
		return nil
	}
	return &IdleCashLadder{Parts: parts, PartAmount: math.Round(idleAmount / float64(parts)), SpacingDays: idleLadderCycleDays / parts}
}

// compare weighs the emergency accounts against the best rate.
func (e *IdleCashEmergency) compare(best DepositRate) {
	if e.Balance <= 0 {
		return
	}
	e.RateGapPoints = round2(best.Rate - e.CurrentRate)
	e.ExtraPerYear = math.Max(0, math.Round(e.Balance*(best.Rate-e.CurrentRate)/100))
	e.BetterRate = e.RateGapPoints >= IdleCashMinRateGapPoints && e.ExtraPerYear >= IdleCashMinExtraPerYear
}

// noInstantRateIssue says why no instant-access rate is available.
func noInstantRateIssue(input IdleCashInput) string {
	if input.RatesIssue != "" {
		return input.RatesIssue
	}
	if len(input.Rates) == 0 {
		return "no bank published a savings rate"
	}
	return "no no-term or 1-month rate is published today (only longer terms or rates without a tenor, which are " +
		"never used for money kept within reach)"
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

// formatRate prints a rate without trailing zeros: 5.5, 4.75, 6.
func formatRate(rate float64) string {
	return strconv.FormatFloat(round2(rate), 'f', -1, 64)
}

// InstantAccessTerms is how the best instant-access option behaves, for
// prose: "rolls monthly; withdraw early at the no-term rate" or
// "withdraw any time".
func (rate DepositRate) InstantAccessTerms() string {
	if rate.Access == DepositOneMonth {
		return "rolls monthly; withdraw early at the no-term rate"
	}
	return "withdraw any time"
}

// InstantAccessKind is "a 1-month deposit" or "a no-term deposit".
func (rate DepositRate) InstantAccessKind() string {
	if rate.Access == DepositOneMonth {
		return "a 1-month deposit"
	}
	return "a no-term deposit"
}

func idleVerdict(idle IdleCash) (IdleCashVerdict, []string) {
	if idle.Idle <= 0 {
		return IdleCashNothingIdle, []string{"The day-to-day float and what must stay liquid cover all ready money, so nothing is idle."}
	}
	if idle.Best == nil {
		return IdleCashNoRate, []string{"No instant-access rate (no-term or 1-month deposit) to compare with: " + idle.RatesIssue + "."}
	}
	reasons := []string{}
	if idle.RateGapPoints < IdleCashMinRateGapPoints {
		reasons = append(reasons, fmt.Sprintf("The best instant-access rate (%s%%, %s at %s) is only %.2f points above the current rate (%.2f%%), under the %.1f-point threshold.",
			formatRate(idle.Best.Rate), strings.TrimPrefix(idle.Best.InstantAccessKind(), "a "), idle.Best.Bank, idle.RateGapPoints, idle.CurrentRate, IdleCashMinRateGapPoints))
	}
	if idle.ExtraPerYear < IdleCashMinExtraPerYear {
		reasons = append(reasons, fmt.Sprintf("It would add about %s VND a year, under the %s VND threshold.",
			groupThousands(idle.ExtraPerYear), groupThousands(IdleCashMinExtraPerYear)))
	}
	if len(reasons) > 0 {
		return IdleCashNotWorthIt, reasons
	}
	return IdleCashWorthMoving, []string{fmt.Sprintf("About %s VND above the day-to-day float is idle at %.2f%%; %s at %s (%s%%, %s) would add about %s VND a year and stays reachable.",
		groupThousands(idle.Idle), idle.CurrentRate, idle.Best.InstantAccessKind(), idle.Best.Bank, formatRate(idle.Best.Rate), idle.Best.InstantAccessTerms(), groupThousands(idle.ExtraPerYear))}
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
	Idle          float64       `json:"idle"`
	ReadyMoney    float64       `json:"readyMoney"`
	KeepLiquid    float64       `json:"keepLiquid"`
	DayToDayFloat float64       `json:"dayToDayFloat"`
	CurrentRate   float64       `json:"currentRate"`
	BestRate      float64       `json:"bestRate"`
	BestBank      string        `json:"bestBank,omitempty"`
	BestKind      DepositAccess `json:"bestKind,omitempty"`
	// LadderParts is how many rolling 1-month parts are suggested (0: none).
	LadderParts         int             `json:"ladderParts,omitempty"`
	ExtraPerYear        float64         `json:"extraPerYear"`
	Verdict             IdleCashVerdict `json:"verdict"`
	AccountsHoldingIdle int             `json:"accountsHoldingIdle"`
	EmergencyTarget     float64         `json:"emergencyTarget"`
	EmergencyAccounts   int             `json:"emergencyAccounts"`
	EmergencyInAccounts float64         `json:"emergencyInAccounts"`
	EmergencyRate       float64         `json:"emergencyRate"`
	EmergencyBetterRate bool            `json:"emergencyBetterRate"`
}

// Facts summarises the check without account names (the bank of the
// published rate is public data).
func (idle IdleCash) Facts() IdleCashFacts {
	facts := IdleCashFacts{
		Idle: idle.Idle, ReadyMoney: idle.Components.ReadyMoney, KeepLiquid: idle.Components.KeepLiquid,
		DayToDayFloat: idle.Components.DayToDayFloat, CurrentRate: idle.CurrentRate, ExtraPerYear: idle.ExtraPerYear, Verdict: idle.Verdict,
		EmergencyTarget: idle.Components.EmergencyFund, EmergencyAccounts: len(idle.Emergency.Accounts),
		EmergencyInAccounts: idle.Emergency.Balance, EmergencyRate: idle.Emergency.CurrentRate, EmergencyBetterRate: idle.Emergency.BetterRate,
	}
	if idle.Best != nil {
		facts.BestRate, facts.BestBank, facts.BestKind = idle.Best.Rate, idle.Best.Bank, idle.Best.Access
	}
	if idle.Ladder != nil {
		facts.LadderParts = idle.Ladder.Parts
	}
	for _, account := range idle.Accounts {
		if account.Idle > 0 {
			facts.AccountsHoldingIdle++
		}
	}
	return facts
}
