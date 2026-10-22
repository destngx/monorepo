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
// that keep the money within reach (no-term or 1-month deposits, and
// flexible savings products such as e-wallet earning balances, D41). The
// advice is about rate only; goal money stays pooled (D31), so it never
// suggests dedicated goal accounts. Emergency accounts hold the emergency
// fund: they cover its target first and are never idle. An account's
// minimum balance to keep (D35) is never idle either: every account adds
// only its balance above its minimum.

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
	"months x average monthly expense) stays ready too. An account's minimum balance to keep (accounts.minimumBalance) " +
	"is never idle nor moved: each ready or emergency account counts only its balance above its minimum. " +
	"Idle = ready money - kept liquid - float."

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

// flexibleRateBasis labels a flexible savings product's rate.
const flexibleRateBasis = "The rate the provider publishes on its product page today, read by the market data server. " +
	"Check the terms in the app before moving money."

// IdleCashProductRule documents linked accounts and the switch check.
const IdleCashProductRule = "An account linked to a flexible savings product (idleCash.accountProducts, e.g. an e-wallet " +
	"earning balance) earns the product's live published rate instead of a note rate. Its day-to-day money (the part " +
	"not idle) is compared only with other flexible products, which stay as spendable; a switch is suggested on the " +
	"same thresholds. A move loses the interest of the days the money earns nothing: from the day it leaves until the " +
	"new place starts paying (the next working day for Zalopay and MoMo, so moving on a Friday costs 3 days)."

// IdleCashComponents is the math:
// Idle = max(0, ReadyMoney - KeepLiquid - DayToDayFloat).
type IdleCashComponents struct {
	// ReadyMoney is safe-to-spend's ready money (WM-BILL-5, D33) without
	// the emergency accounts, each account above its minimum (D35).
	ReadyMoney float64 `json:"readyMoney"`
	// MinimumReserved is the minimum balances kept in the everyday ready
	// accounts, left out of ReadyMoney (D35).
	MinimumReserved float64 `json:"minimumReserved"`
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
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
	// Reserved is the minimum balance kept (D35): never idle, never moved.
	Reserved     float64 `json:"reserved"`
	Rate         float64 `json:"rate"`
	RateFromNote bool    `json:"rateFromNote"`
	// Product is the linked flexible savings product's name when the rate
	// is its live rate (D41).
	Product   string  `json:"product,omitempty"`
	ProductID string  `json:"productId,omitempty"`
	Idle      float64 `json:"idle"`
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
	// Balance is the emergency accounts' positive balance above their
	// minimums (D35).
	Balance float64 `json:"balance"`
	Target  float64 `json:"target"`
	// CurrentRate is weighted over the emergency balances.
	CurrentRate   float64 `json:"currentRate"`
	RateGapPoints float64 `json:"rateGapPoints"`
	// ExtraPerYear = Balance x (best - current) / 100 when positive.
	ExtraPerYear float64 `json:"extraPerYear"`
	// BetterRate: the best instant-access rate beats the emergency
	// accounts by the materiality thresholds.
	BetterRate bool `json:"betterRate"`
	// Moves plan moving each emergency account's money to Best when
	// BetterRate (WM-CASH-9).
	Moves []SavingsMove `json:"moves"`
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
	// Best is the best instant-access rate (no-term or 1-month, a proxy, or
	// a flexible savings product); nil when none is published.
	Best *DepositRate `json:"best"`
	// Options are the instant-access places compared, best first: the best
	// bank no-term and 1-month rates and every flexible savings product.
	Options    []DepositRate `json:"options"`
	RatesIssue string        `json:"ratesIssue,omitempty"`
	// ProductsIssue says why flexible savings products are missing or
	// partial (a provider's page could not be read).
	ProductsIssue string `json:"productsIssue,omitempty"`
	// Ladder is set when Best is a 1-month deposit and the idle cash is
	// big enough to split.
	Ladder        *IdleCashLadder `json:"ladder"`
	RateGapPoints float64         `json:"rateGapPoints"`
	// ExtraPerYear = Idle x (best - current) / 100 when positive, else 0.
	ExtraPerYear float64 `json:"extraPerYear"`
	// Moves plan moving the idle part of each account to Best when the
	// verdict is worth_moving (WM-CASH-9).
	Moves []SavingsMove `json:"moves"`
	// Switches plan moving the day-to-day money of an account linked to a
	// product to a better flexible product (WM-CASH-10).
	Switches       []SavingsMove   `json:"switches"`
	Verdict        IdleCashVerdict `json:"verdict"`
	Reasons        []string        `json:"reasons"`
	Issues         []string        `json:"issues"`
	MinRateGap     float64         `json:"minRateGapPoints"`
	MinExtra       float64         `json:"minExtraPerYear"`
	KeepLiquidRule string          `json:"keepLiquidRule"`
	RateRule       string          `json:"rateRule"`
	ThresholdRule  string          `json:"thresholdRule"`
	ProductRule    string          `json:"productRule"`
}

// IdleCashInput is what ComputeIdleCash needs. Plan may be nil (no goal
// plan: no goal set-aside, float or emergency fund kept). The accounts
// marked not ready come from Bills.SafeToSpend.Readiness (D33). Rates are
// the published bank rates; RatesIssue says why they are missing.
// Products are the flexible savings products; ProductsIssue says why they
// are missing or partial.
type IdleCashInput struct {
	Accounts      []Account
	Bills         Bills
	Plan          *GoalPlan
	Settings      IdleCashSettings
	Rates         []BankRate
	RatesIssue    string
	Products      []FlexibleSaving
	ProductsIssue string
	Today         time.Time
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
		Options:        []DepositRate{},
		Moves:          []SavingsMove{},
		Switches:       []SavingsMove{},
		RatesIssue:     input.RatesIssue,
		ProductsIssue:  input.ProductsIssue,
		MinRateGap:     IdleCashMinRateGapPoints,
		MinExtra:       IdleCashMinExtraPerYear,
		KeepLiquidRule: IdleCashKeepLiquidRule,
		RateRule:       IdleCashRateRule,
		ThresholdRule:  IdleCashThresholdRule,
		ProductRule:    IdleCashProductRule,
	}
	idle.sortAccounts(input)
	idle.Components = idleComponents(input, idle.Accounts, idle.Emergency.Balance)
	idle.Emergency.Target = idle.Components.EmergencyFund
	c := idle.Components
	idle.Idle = math.Max(0, math.Round(c.ReadyMoney-c.KeepLiquid-c.DayToDayFloat))

	left, earning := idle.Idle, 0.0
	for i := range idle.Accounts {
		part := math.Max(0, math.Min(idle.Accounts[i].available(), left))
		idle.Accounts[i].Idle = part
		earning += part * idle.Accounts[i].Rate
		left -= part
	}
	if idle.Idle > 0 {
		idle.CurrentRate = round2(earning / idle.Idle)
	}

	idle.Options = instantAccessOptions(input)
	if len(idle.Options) > 0 {
		rate := idle.Options[0]
		idle.Best = &rate
		idle.RateGapPoints = round2(rate.Rate - idle.CurrentRate)
		idle.ExtraPerYear = math.Max(0, math.Round(idle.Idle*(rate.Rate-idle.CurrentRate)/100))
		idle.Ladder = idleLadder(idle.Idle, rate)
		idle.Emergency.compare(rate)
	}
	if _, ok := InstantAccessRateFromBankRates(input.Rates); !ok && idle.RatesIssue == "" {
		idle.RatesIssue = noInstantRateIssue(input)
	}
	idle.Verdict, idle.Reasons = idleVerdict(idle)
	idle.planMoves(input)
	return idle
}

// instantAccessOptions are the places compared, best first: the best bank
// no-term rate, the best bank 1-month rate and every flexible savings
// product. A tie goes to a deposit, then to quicker access (no-term, a
// product, 1-month).
func instantAccessOptions(input IdleCashInput) []DepositRate {
	options := []DepositRate{}
	// 0 is a no-term deposit.
	for _, months := range []int{0, 1} {
		if rate, ok := InstantAccessRateFromBankRates(onlyTenor(input.Rates, months)); ok {
			rate.Basis = idleCashRateBasis
			options = append(options, rate)
		}
	}
	for _, product := range input.Products {
		if product.Rate > 0 && product.Rate < accountRateCeiling {
			rate := FlexibleDepositRate(product)
			rate.Basis = flexibleRateBasis
			options = append(options, rate)
		}
	}
	rank := map[DepositAccess]int{DepositNoTerm: 0, DepositFlexible: 1, DepositOneMonth: 2}
	slices.SortStableFunc(options, func(a, b DepositRate) int {
		switch {
		case a.Rate != b.Rate:
			return cmpFloat(b.Rate, a.Rate)
		case (a.Backing == SavingsBackingBusinessCooperation) != (b.Backing == SavingsBackingBusinessCooperation):
			if a.Backing == SavingsBackingBusinessCooperation {
				return 1
			}
			return -1
		}
		return rank[a.Access] - rank[b.Access]
	})
	return options
}

// onlyTenor keeps the bank rates of one tenor in months.
func onlyTenor(rates []BankRate, months int) []BankRate {
	return slices.DeleteFunc(slices.Clone(rates), func(rate BankRate) bool {
		tenor, ok := bankRateMonths(rate)
		return !ok || tenor != months
	})
}

// planMoves plans moving the idle part to Best (worth_moving), the
// emergency accounts to Best (BetterRate) and the day-to-day money of
// linked accounts to a better flexible product.
func (idle *IdleCash) planMoves(input IdleCashInput) {
	products := map[string]FlexibleSaving{}
	for _, product := range input.Products {
		products[strings.ToLower(product.ID)] = product
	}
	plan := func(account IdleCashAccount, amount float64, to DepositRate) (SavingsMove, bool) {
		if account.ProductID != "" && strings.EqualFold(account.ProductID, to.ProductID) {
			return SavingsMove{}, false
		}
		in := savingsMoveInput{From: account.Name, FromRate: account.Rate, Amount: amount, To: to, Today: input.Today}
		if product, ok := products[strings.ToLower(account.ProductID)]; ok {
			in.FromProduct = &product
		}
		return planSavingsMove(in)
	}
	if idle.Best != nil && idle.Verdict == IdleCashWorthMoving {
		for _, account := range idle.Accounts {
			if move, ok := plan(account, account.Idle, *idle.Best); ok {
				idle.Moves = append(idle.Moves, move)
			}
		}
	}
	if idle.Best != nil && idle.Emergency.BetterRate {
		for _, account := range idle.Emergency.Accounts {
			if move, ok := plan(account, account.available(), *idle.Best); ok {
				idle.Emergency.Moves = append(idle.Emergency.Moves, move)
			}
		}
	}
	for _, account := range idle.Accounts {
		if account.ProductID == "" {
			continue
		}
		var best *DepositRate
		for i := range idle.Options {
			option := idle.Options[i]
			if option.Access == DepositFlexible && !strings.EqualFold(option.ProductID, account.ProductID) {
				best = &option
				break
			}
		}
		if best == nil {
			continue
		}
		move, ok := plan(account, account.available()-account.Idle, *best)
		if ok && move.RateGapPoints >= IdleCashMinRateGapPoints && move.ExtraPerYear >= IdleCashMinExtraPerYear {
			idle.Switches = append(idle.Switches, move)
		}
	}
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
		Accounts: []IdleCashAccount{}, Unknown: []string{}, Moves: []SavingsMove{},
		Settings: IdleCashSettings{
			DayToDayFloat: input.Settings.DayToDayFloat, EmergencyAccounts: append([]string{}, emergencyNames...), ETag: input.Settings.ETag,
			AccountProducts: AccountProductSettings{Accounts: append([]AccountProduct{}, input.Settings.AccountProducts.Accounts...), ETag: input.Settings.AccountProducts.ETag},
		},
	}
	// An emergency account is set aside whatever its not-ready mark.
	notReady := slices.DeleteFunc(slices.Clone(input.Bills.SafeToSpend.Readiness.NotReadyAccounts), func(name string) bool { return marked(emergencyNames, name) })
	minimums := input.Bills.SafeToSpend.MinimumBalance
	var ready SafeToSpend
	readyMoney(&ready, input.Accounts, notReady, minimums.Accounts)
	idle.Excluded = append(idle.Excluded, ready.ExcludedAccounts...)
	byName := map[string]Account{}
	for _, account := range input.Accounts {
		byName[strings.ToLower(strings.TrimSpace(account.Name))] = account
	}
	productOf := idle.linkedProducts(input, byName)
	for _, spendable := range ready.ReadyAccounts {
		if !marked(emergencyNames, spendable.Name) {
			idle.Accounts = append(idle.Accounts, accountWithRate(byName[strings.ToLower(strings.TrimSpace(spendable.Name))], minimums, productOf(spendable.Name)))
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
			entry := accountWithRate(account, minimums, productOf(account.Name))
			idle.Emergency.Accounts = append(idle.Emergency.Accounts, entry)
			idle.Emergency.Balance += entry.available()
		}
	}
	sortByRate(idle.Emergency.Accounts)
	weighted := 0.0
	for _, account := range idle.Emergency.Accounts {
		weighted += account.available() * account.Rate
	}
	if idle.Emergency.Balance > 0 {
		idle.Emergency.CurrentRate = round2(weighted / idle.Emergency.Balance)
	}
}

// linkedProducts resolves idleCash.accountProducts: the returned lookup
// gives an account's linked product, nil when it has none or the product is
// not listed today (then the note rate stands and an issue says why).
func (idle *IdleCash) linkedProducts(input IdleCashInput, byName map[string]Account) func(string) *FlexibleSaving {
	linked := map[string]*FlexibleSaving{}
	for _, link := range input.Settings.AccountProducts.Accounts {
		if _, ok := byName[strings.ToLower(strings.TrimSpace(link.Account))]; !ok {
			idle.Issues = append(idle.Issues, fmt.Sprintf("Preferences: %s links %q, which is not an account (renamed or deleted?); it is ignored. Unlink it to remove it.", PrefIdleCashAccountProducts, link.Account))
			continue
		}
		index := slices.IndexFunc(input.Products, func(p FlexibleSaving) bool { return strings.EqualFold(p.ID, link.Product) })
		if index < 0 {
			reason := "the market data server does not list it today"
			if input.ProductsIssue != "" {
				reason = input.ProductsIssue
			}
			idle.Issues = append(idle.Issues, fmt.Sprintf("%s is linked to %s but its live rate is unavailable (%s); the rate in its note is used.", link.Account, link.Product, reason))
			continue
		}
		product := input.Products[index]
		if product.Stale {
			idle.Issues = append(idle.Issues, fmt.Sprintf("%s's rate for %s is the last one read (%s); the provider's page could not be read since.", product.Name(), link.Account, product.AsOf))
		}
		linked[strings.ToLower(strings.TrimSpace(link.Account))] = &product
	}
	return func(name string) *FlexibleSaving { return linked[strings.ToLower(strings.TrimSpace(name))] }
}

// isReadyType: the account types that can be ready money.
func isReadyType(account Account) bool {
	switch account.Type {
	case AccountTypeActiveUse, AccountTypeRarelyUse, AccountTypeBank, AccountTypeCash:
		return true
	}
	return false
}

func accountWithRate(account Account, minimums AccountMinimumSettings, product *FlexibleSaving) IdleCashAccount {
	entry := IdleCashAccount{Name: account.Name, Balance: account.Balance}
	switch {
	case product != nil:
		entry.Rate, entry.Product, entry.ProductID = product.Rate, product.Name(), product.ID
	case account.Note != nil:
		entry.Rate, entry.RateFromNote = AccountRateFromNote(*account.Note)
	}
	if minimum, ok := minimums.For(account.Name); ok {
		entry.Reserved = math.Min(math.Max(0, account.Balance), minimum.Minimum)
	}
	return entry
}

// available is the positive balance above the minimum: what the account
// adds to ready money or the emergency fund (D35).
func (a IdleCashAccount) available() float64 {
	return math.Max(0, a.Balance-a.Reserved)
}

// sortByRate lists the accounts lowest rate first, then the larger
// balance above the minimum, then by name.
func sortByRate(accounts []IdleCashAccount) {
	slices.SortStableFunc(accounts, func(a, b IdleCashAccount) int {
		switch {
		case a.Rate != b.Rate:
			return cmpFloat(a.Rate, b.Rate)
		case a.available() != b.available():
			return cmpFloat(b.available(), a.available())
		}
		return strings.Compare(a.Name, b.Name)
	})
}

func idleComponents(input IdleCashInput, ready []IdleCashAccount, emergencyBalance float64) IdleCashComponents {
	c := IdleCashComponents{EmergencyInAccounts: emergencyBalance}
	for _, account := range ready {
		c.ReadyMoney += account.available()
		c.MinimumReserved += account.Reserved
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
// prose: "rolls monthly; withdraw early at the no-term rate", "withdraw
// any time" or, for a product that is not a deposit, "spend or withdraw
// any time; not a bank deposit".
func (rate DepositRate) InstantAccessTerms() string {
	switch {
	case rate.Access == DepositOneMonth:
		return "rolls monthly; withdraw early at the no-term rate"
	case rate.Access == DepositFlexible && rate.Backing == SavingsBackingBusinessCooperation:
		return "spend or withdraw any time; not a bank deposit"
	case rate.Access == DepositFlexible:
		return "spend or withdraw any time"
	}
	return "withdraw any time"
}

// InstantAccessKind is "a 1-month deposit", "a no-term deposit" or "an
// earning balance".
func (rate DepositRate) InstantAccessKind() string {
	switch rate.Access {
	case DepositOneMonth:
		return "a 1-month deposit"
	case DepositFlexible:
		return "an earning balance"
	}
	return "a no-term deposit"
}

// Place names where the money would go: "a no-term deposit at Bank A",
// "Zalopay Số dư sinh lời".
func (rate DepositRate) Place() string {
	if rate.Access == DepositFlexible {
		return strings.TrimSpace(rate.Bank + " " + rate.Product)
	}
	return rate.InstantAccessKind() + " at " + rate.Bank
}

func idleVerdict(idle IdleCash) (IdleCashVerdict, []string) {
	if idle.Idle <= 0 {
		return IdleCashNothingIdle, []string{"The day-to-day float and what must stay liquid cover all ready money, so nothing is idle."}
	}
	if idle.Best == nil {
		return IdleCashNoRate, []string{"No instant-access rate (no-term or 1-month deposit, or a flexible savings product) to compare with: " + idle.RatesIssue + "."}
	}
	reasons := []string{}
	if idle.RateGapPoints < IdleCashMinRateGapPoints {
		reasons = append(reasons, fmt.Sprintf("The best instant-access rate (%s%%, %s) is only %.2f points above the current rate (%.2f%%), under the %.1f-point threshold.",
			formatRate(idle.Best.Rate), strings.TrimPrefix(idle.Best.Place(), "a "), idle.RateGapPoints, idle.CurrentRate, IdleCashMinRateGapPoints))
	}
	if idle.ExtraPerYear < IdleCashMinExtraPerYear {
		reasons = append(reasons, fmt.Sprintf("It would add about %s VND a year, under the %s VND threshold.",
			groupThousands(idle.ExtraPerYear), groupThousands(IdleCashMinExtraPerYear)))
	}
	if len(reasons) > 0 {
		return IdleCashNotWorthIt, reasons
	}
	return IdleCashWorthMoving, []string{fmt.Sprintf("About %s VND above the day-to-day float is idle at %.2f%%; %s (%s%%, %s) would add about %s VND a year and stays reachable.",
		groupThousands(idle.Idle), idle.CurrentRate, idle.Best.Place(), formatRate(idle.Best.Rate), idle.Best.InstantAccessTerms(), groupThousands(idle.ExtraPerYear))}
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
	Idle          float64 `json:"idle"`
	ReadyMoney    float64 `json:"readyMoney"`
	KeepLiquid    float64 `json:"keepLiquid"`
	DayToDayFloat float64 `json:"dayToDayFloat"`
	// MinimumReserved is the minimum balances kept out of ready money (D35).
	MinimumReserved float64       `json:"minimumReserved"`
	CurrentRate     float64       `json:"currentRate"`
	BestRate        float64       `json:"bestRate"`
	BestBank        string        `json:"bestBank,omitempty"`
	BestKind        DepositAccess `json:"bestKind,omitempty"`
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
		DayToDayFloat: idle.Components.DayToDayFloat, MinimumReserved: idle.Components.MinimumReserved, CurrentRate: idle.CurrentRate, ExtraPerYear: idle.ExtraPerYear, Verdict: idle.Verdict,
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
