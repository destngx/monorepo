package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Financial health score (WM-HEA-1..3) and freedom months (WM-RULE-12).
// Everything here is deterministic so the UI, AI and MCP agree.

// Pillar weights (D44): they sum to 100.
const (
	healthSavingsMax         = 20
	healthConsistencyMax     = 10
	healthEmergencyMax       = 20
	healthNeedsMax           = 10
	healthDebtMax            = 10
	healthSpendingTrendMax   = 10
	healthLendingMax         = 10
	healthDiversificationMax = 10
)

const (
	TargetSavingsRate     = 30.0
	TargetEmergencyMonths = 6.0
	ReferenceDebtToIncome = 35.0

	// The emergency pillar scores nothing below one month of expense.
	healthEmergencyFloorMonths = 1.0
	// Needs (fixed, essential and untagged expense) as a share of income:
	// full at the 50% rule of thumb, nothing from 80%.
	healthNeedsTarget = 50.0
	healthNeedsZero   = 80.0
	// Card minimums over income: full up to 5%, nothing from 35%.
	healthDebtFree = 5.0
	// Recent 3-month expense against the 3 months before: full when flat
	// or falling, nothing from 20% growth.
	healthSpendingGrowthZero = 20.0
	// Money lent as a share of assets: full up to 10%, nothing from 40%.
	healthLendingTarget = 10.0
	healthLendingZero   = 40.0
	// Crypto above a quarter of cash, funds and crypto keeps the score out
	// of the green band.
	healthCryptoCapShare = 25.0

	healthLookbackMonths = 6

	// Score caps (D44): a critical rule holds the score in the red band,
	// a serious one keeps it out of the green band.
	HealthCapCritical = 49.0
	HealthCapSerious  = 75.0

	// CardMinimumPaymentPercent is the assumed monthly minimum payment on a
	// card balance owed, used for debt to income (WM-HEA-2).
	CardMinimumPaymentPercent = 5.0
)

type HealthBand string

const (
	HealthBandRed    HealthBand = "red"
	HealthBandYellow HealthBand = "yellow"
	HealthBandGreen  HealthBand = "green"
)

// HealthBandFor: red < 50, yellow 50-75, green > 75.
func HealthBandFor(score float64) HealthBand {
	switch {
	case score < 50:
		return HealthBandRed
	case score <= 75:
		return HealthBandYellow
	default:
		return HealthBandGreen
	}
}

// HealthComponent is one scored pillar with its "show your math" breakdown.
type HealthComponent struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Value    float64 `json:"value"`
	Target   float64 `json:"target"`
	Unit     string  `json:"unit"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"maxScore"`
	Formula  string  `json:"formula"`
	Detail   string  `json:"detail"`
}

// HealthCap is a strict rule that holds the score down whatever the
// pillars add up to (D44): the score is at most Limit.
type HealthCap struct {
	Key    string  `json:"key"`
	Limit  float64 `json:"limit"`
	Reason string  `json:"reason"`
}

// HealthRecommendation is one of the "Doctor's Orders".
type HealthRecommendation struct {
	Component string `json:"component"`
	Title     string `json:"title"`
	Action    string `json:"action"`
}

type FinancialHealth struct {
	Score float64 `json:"score"`
	// UncappedScore is the sum of the pillars before the caps.
	UncappedScore     float64                `json:"uncappedScore"`
	Band              HealthBand             `json:"band"`
	Components        []HealthComponent      `json:"components"`
	Caps              []HealthCap            `json:"caps"`
	Recommendations   []HealthRecommendation `json:"recommendations"`
	LiquidAssets      float64                `json:"liquidAssets"`
	AvgMonthlyIncome  float64                `json:"avgMonthlyIncome"`
	AvgMonthlyExpense float64                `json:"avgMonthlyExpense"`
	NetWorth          float64                `json:"netWorth"`
	FreedomMonths     *float64               `json:"freedomMonths"`
	// Allocation is the one engine allocation (ROADMAP C4); the
	// diversification pillar scores its target classes.
	Allocation         Allocation `json:"allocation"`
	LookbackMonths     int        `json:"lookbackMonths"`
	InsufficientIncome bool       `json:"insufficientIncome"`
	// MonthlyDebtPayments is the card minimums (D16).
	MonthlyDebtPayments float64              `json:"monthlyDebtPayments"`
	Projection          []NetWorthProjection `json:"projection"`
}

// healthFacts are the figures the pillars and caps are read from.
type healthFacts struct {
	income, expense, needs  float64
	savingsRate             float64
	surplusMonths           int
	liquid, emergencyMonths float64
	debtPayments, dti       float64
	needsRatio              float64
	recentExpense           float64
	earlierExpense          float64
	spendingGrowth          float64
	spendingKnown           bool
	lent, lentShare         float64
	assets                  float64
	cryptoShare             float64
	netWorth                float64
}

// ComputeFinancialHealth scores the last six full months of activity
// against strict targets (WM-HEA-1, D44). Missing evidence scores nothing:
// a pillar that cannot be measured earns no points. allocation is the
// engine allocation (ComputeAllocation).
func ComputeFinancialHealth(accounts []Account, transactions []Transaction, allocation Allocation, today time.Time) FinancialHealth {
	sheet := ComputeBalanceSheet(accounts, transactions)
	f := healthFactsOf(accounts, transactions, allocation, sheet, today)
	hhi, diversification := diversificationScore(allocation)

	components := []HealthComponent{
		{
			Key: "savings", Label: "Savings rate", Value: f.savingsRate, Target: TargetSavingsRate, Unit: "%", MaxScore: healthSavingsMax,
			Score:   scaled(healthSavingsMax, f.savingsRate/TargetSavingsRate),
			Formula: "(avg income - avg expense) / avg income x 100 over 6 full months; nothing at 0% or below, full at 30%",
			Detail:  fmt.Sprintf("(%.0f - %.0f) / %.0f = %.2f%%", f.income, f.expense, f.income, f.savingsRate),
		},
		{
			Key: "consistency", Label: "Months in surplus", Value: float64(f.surplusMonths), Target: healthLookbackMonths, Unit: "of 6 months", MaxScore: healthConsistencyMax,
			Score:   scaled(healthConsistencyMax, float64(f.surplusMonths)/healthLookbackMonths),
			Formula: "full months of the last 6 where income beat expense, out of 6; a month without records counts as missed",
			Detail:  fmt.Sprintf("%d of 6 months in surplus", f.surplusMonths),
		},
		{
			Key: "emergency", Label: "Emergency fund", Value: f.emergencyMonths, Target: TargetEmergencyMonths, Unit: "months", MaxScore: healthEmergencyMax,
			Score:   scaled(healthEmergencyMax, (f.emergencyMonths-healthEmergencyFloorMonths)/(TargetEmergencyMonths-healthEmergencyFloorMonths)),
			Formula: "liquid assets / avg monthly expense; nothing below 1 month, full at 6 months",
			Detail:  fmt.Sprintf("%.0f / %.0f = %.2f months", f.liquid, f.expense, f.emergencyMonths),
		},
		{
			Key: "needs", Label: "Needs to income", Value: f.needsRatio, Target: healthNeedsTarget, Unit: "%", MaxScore: healthNeedsMax,
			Score:   needsScore(f),
			Formula: "fixed, essential and untagged expense / avg income x 100; full at 50% or less, nothing from 80%",
			Detail:  fmt.Sprintf("%.0f / %.0f = %.2f%%", f.needs, f.income, f.needsRatio),
		},
		{
			Key: "debt", Label: "Debt to income", Value: f.dti, Target: healthDebtFree, Unit: "%", MaxScore: healthDebtMax,
			Score:   scaled(healthDebtMax, (ReferenceDebtToIncome-f.dti)/(ReferenceDebtToIncome-healthDebtFree)),
			Formula: "5% of each negative account balance (card minimum payments) / avg monthly income x 100; full at 5% or less, nothing from 35%",
			Detail:  fmt.Sprintf("%.0f / %.0f = %.2f%%", f.debtPayments, f.income, f.dti),
		},
		{
			Key: "spendingTrend", Label: "Spending trend", Value: f.spendingGrowth, Target: 0, Unit: "%", MaxScore: healthSpendingTrendMax,
			Score:   spendingTrendScore(f),
			Formula: "avg expense of the last 3 full months against the 3 before; full when flat or falling, nothing from +20%; needs both periods",
			Detail:  spendingTrendDetail(f),
		},
		{
			Key: "lending", Label: "Money lent", Value: f.lentShare, Target: healthLendingTarget, Unit: "%", MaxScore: healthLendingMax,
			Score:   lendingScore(f),
			Formula: "money lent / (cash, funds, crypto and money lent) x 100; full at 10% or less, nothing from 40%",
			Detail:  fmt.Sprintf("%.0f / %.0f = %.2f%%", f.lent, f.assets, f.lentShare),
		},
		{
			Key: "diversification", Label: "Diversification", Value: round2(hhi), Target: 1 / float64(len(TargetClasses)), Unit: "HHI", MaxScore: healthDiversificationMax,
			Score:   round2(diversification),
			Formula: "Herfindahl index over cash and deposits, fund certificates and crypto (money lent left out); full when evenly split, nothing in one class",
			Detail:  allocationDetail(allocation),
		},
	}
	total := 0.0
	for _, component := range components {
		total += component.Score
	}
	total = round2(total)
	caps := healthCaps(f)
	score := total
	for _, c := range caps {
		score = math.Min(score, c.Limit)
	}

	health := FinancialHealth{
		Score: score, UncappedScore: total, Band: HealthBandFor(score), Components: components, Caps: caps,
		LiquidAssets: f.liquid, AvgMonthlyIncome: f.income, AvgMonthlyExpense: f.expense, NetWorth: sheet.NetWorth,
		FreedomMonths: FreedomMonths(sheet.NetWorth, f.expense), Allocation: allocation,
		LookbackMonths: healthLookbackMonths, InsufficientIncome: f.income == 0,
		MonthlyDebtPayments: f.debtPayments, Projection: ProjectNetWorth(sheet.NetWorth),
	}
	health.Recommendations = doctorsOrders(components, caps, health)
	return health
}

func healthFactsOf(accounts []Account, transactions []Transaction, allocation Allocation, sheet BalanceSheet, today time.Time) healthFacts {
	current := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	flows := MonthlyCashFlows(transactions, current.AddDate(0, -healthLookbackMonths, 0), current.AddDate(0, -1, 0))
	f := healthFacts{netWorth: sheet.NetWorth, liquid: liquidBalance(accounts)}
	for i, flow := range flows {
		f.income += flow.Income
		f.expense += flow.Expense
		if flow.Income > flow.Expense {
			f.surplusMonths++
		}
		if i < healthLookbackMonths/2 {
			f.earlierExpense += flow.Expense
		} else {
			f.recentExpense += flow.Expense
		}
	}
	f.income = math.Round(f.income / healthLookbackMonths)
	f.expense = math.Round(f.expense / healthLookbackMonths)
	half := float64(healthLookbackMonths / 2)
	f.earlierExpense = math.Round(f.earlierExpense / half)
	f.recentExpense = math.Round(f.recentExpense / half)

	first := current.AddDate(0, -healthLookbackMonths, 0)
	needs := 0.0
	for _, t := range transactions {
		if !t.Usable() || !t.CountsAsExpense() {
			continue
		}
		if date := EffectiveDate(t); date.Before(first) || !date.Before(current) {
			continue
		}
		// Untagged expense counts as a need: what is not known to be a want is not assumed to be one.
		if t.CategoryNature != NatureFlexible {
			needs += t.PaymentAmount()
		}
	}
	f.needs = math.Round(needs / healthLookbackMonths)

	if f.income > 0 {
		f.savingsRate = round2((f.income - f.expense) / f.income * 100)
		f.needsRatio = round2(f.needs / f.income * 100)
	}
	if f.expense > 0 {
		f.emergencyMonths = round2(f.liquid / f.expense)
	}
	f.debtPayments = cardMinimumPayments(accounts)
	switch {
	case f.income > 0:
		f.dti = round2(f.debtPayments / f.income * 100)
	case f.debtPayments > 0:
		f.dti = 100
	}
	if f.earlierExpense > 0 && f.recentExpense > 0 {
		f.spendingKnown = true
		f.spendingGrowth = round2((f.recentExpense - f.earlierExpense) / f.earlierExpense * 100)
	}
	f.lent = allocation.Line(AllocationMoneyLent).Value
	f.assets = allocation.Total
	if f.assets > 0 {
		f.lentShare = round2(f.lent / f.assets * 100)
	}
	if allocation.TargetedTotal > 0 {
		f.cryptoShare = round2(allocation.Line(AllocationCrypto).Value / allocation.TargetedTotal * 100)
	}
	return f
}

// scaled is max x fraction, the fraction clamped to 0..1, rounded.
func scaled(max float64, fraction float64) float64 {
	return round2(max * clampUnit(fraction))
}

func needsScore(f healthFacts) float64 {
	if f.income <= 0 {
		return 0
	}
	return scaled(healthNeedsMax, (healthNeedsZero-f.needsRatio)/(healthNeedsZero-healthNeedsTarget))
}

func spendingTrendScore(f healthFacts) float64 {
	if !f.spendingKnown {
		return 0
	}
	return scaled(healthSpendingTrendMax, (healthSpendingGrowthZero-f.spendingGrowth)/healthSpendingGrowthZero)
}

func spendingTrendDetail(f healthFacts) string {
	if !f.spendingKnown {
		return "Not scored: needs expense in both 3-month periods"
	}
	return fmt.Sprintf("(%.0f - %.0f) / %.0f = %+.2f%%", f.recentExpense, f.earlierExpense, f.earlierExpense, f.spendingGrowth)
}

// lendingScore: nothing to lend from scores nothing; no money lent scores full.
func lendingScore(f healthFacts) float64 {
	if f.assets <= 0 {
		return 0
	}
	return scaled(healthLendingMax, (healthLendingZero-f.lentShare)/(healthLendingZero-healthLendingTarget))
}

// healthCaps are the strict rules (D44), critical first. Each holds the
// score at its limit, whatever the pillars add up to.
func healthCaps(f healthFacts) []HealthCap {
	caps := []HealthCap{}
	add := func(key string, limit float64, reason string) {
		caps = append(caps, HealthCap{Key: key, Limit: limit, Reason: reason})
	}
	if f.income <= 0 {
		add("noIncome", HealthCapCritical, "No income recorded in the last 6 full months.")
	} else if f.savingsRate < 0 {
		add("deficit", HealthCapCritical, fmt.Sprintf("You spent more than you earned over the last 6 months (savings rate %.1f%%).", f.savingsRate))
	}
	if f.emergencyMonths < healthEmergencyFloorMonths {
		add("noBuffer", HealthCapCritical, fmt.Sprintf("Liquid cash covers %.1f months of expense, under 1 month.", f.emergencyMonths))
	}
	if f.netWorth < 0 {
		add("negativeNetWorth", HealthCapCritical, "You owe more than you own.")
	}
	if f.income > 0 && f.savingsRate >= 0 && f.savingsRate < 10 {
		add("lowSavings", HealthCapSerious, fmt.Sprintf("Savings rate %.1f%% is under 10%%.", f.savingsRate))
	}
	if f.emergencyMonths >= healthEmergencyFloorMonths && f.emergencyMonths < 3 {
		add("thinBuffer", HealthCapSerious, fmt.Sprintf("Liquid cash covers %.1f months of expense, under 3 months.", f.emergencyMonths))
	}
	if missed := healthLookbackMonths - f.surplusMonths; missed >= 2 {
		add("deficitMonths", HealthCapSerious, fmt.Sprintf("%d of the last 6 months ended without a surplus.", missed))
	}
	if f.dti >= ReferenceDebtToIncome {
		add("highDebt", HealthCapSerious, fmt.Sprintf("Card minimums take %.1f%% of income, 35%% or more.", f.dti))
	}
	if f.spendingKnown && f.spendingGrowth >= healthSpendingGrowthZero {
		add("spendingGrowth", HealthCapSerious, fmt.Sprintf("Spending grew %.1f%% on the 3 months before.", f.spendingGrowth))
	}
	if f.lentShare >= healthLendingZero {
		add("lentTooMuch", HealthCapSerious, fmt.Sprintf("Money lent is %.1f%% of your assets, 40%% or more.", f.lentShare))
	}
	if f.cryptoShare > healthCryptoCapShare {
		add("cryptoHeavy", HealthCapSerious, fmt.Sprintf("Crypto is %.1f%% of cash, funds and crypto, over a quarter.", f.cryptoShare))
	}
	return caps
}

// cardMinimumPayments is 5% of every negative account balance, liabilities
// being decided by balance sign as in WM-RULE-2 (D16). Money the user owes
// people (D30) has no payment schedule, so it is not a monthly payment.
func cardMinimumPayments(accounts []Account) float64 {
	var cardMinimums float64
	for _, account := range accounts {
		if account.Balance < 0 {
			cardMinimums += -account.Balance * CardMinimumPaymentPercent / 100
		}
	}
	return math.Round(cardMinimums)
}

func averageIncomeExpense(transactions []Transaction, today time.Time, months int) (float64, float64) {
	current := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	flows := MonthlyCashFlows(transactions, current.AddDate(0, -months, 0), current.AddDate(0, -1, 0))
	var income, expense float64
	for _, flow := range flows {
		income += flow.Income
		expense += flow.Expense
	}
	return math.Round(income / float64(months)), math.Round(expense / float64(months))
}

// doctorsOrders returns three actions: the pillars behind a cap first
// (critical caps before serious ones), then the weakest pillars by score
// share.
func doctorsOrders(components []HealthComponent, caps []HealthCap, health FinancialHealth) []HealthRecommendation {
	byKey := map[string]HealthComponent{}
	for _, component := range components {
		byKey[component.Key] = component
	}
	ranked := []HealthComponent{}
	seen := map[string]bool{}
	push := func(component HealthComponent) {
		if !seen[component.Key] {
			seen[component.Key] = true
			ranked = append(ranked, component)
		}
	}
	for _, c := range caps {
		if component, ok := byKey[capPillar[c.Key]]; ok {
			push(component)
		}
	}
	rest := append([]HealthComponent(nil), components...)
	sort.SliceStable(rest, func(i, j int) bool {
		return rest[i].Score/rest[i].MaxScore < rest[j].Score/rest[j].MaxScore
	})
	for _, component := range rest {
		push(component)
	}
	orders := make([]HealthRecommendation, 0, 3)
	for _, component := range ranked[:3] {
		orders = append(orders, recommendationFor(component, health))
	}
	return orders
}

// capPillar is the pillar that fixes each cap.
var capPillar = map[string]string{
	"noIncome": "savings", "deficit": "savings", "lowSavings": "savings",
	"noBuffer": "emergency", "thinBuffer": "emergency",
	"negativeNetWorth": "debt", "highDebt": "debt",
	"deficitMonths":  "consistency",
	"spendingGrowth": "spendingTrend",
	"lentTooMuch":    "lending",
	"cryptoHeavy":    "diversification",
}

func recommendationFor(component HealthComponent, health FinancialHealth) HealthRecommendation {
	full := component.Score >= component.MaxScore
	switch component.Key {
	case "savings":
		if full {
			return HealthRecommendation{component.Key, "Keep saving", fmt.Sprintf("Savings rate %.1f%% meets the 30%% target. Route the surplus to goals or investments.", component.Value)}
		}
		gap := math.Max(0, TargetSavingsRate/100*health.AvgMonthlyIncome-(health.AvgMonthlyIncome-health.AvgMonthlyExpense))
		return HealthRecommendation{component.Key, "Raise your savings rate", fmt.Sprintf("Savings rate is %.1f%%. Cut about %.0f per month from expenses to reach 30%%.", component.Value, gap)}
	case "consistency":
		if full {
			return HealthRecommendation{component.Key, "Every month in surplus", "Income beat expense in each of the last 6 months. Keep large buys planned ahead."}
		}
		return HealthRecommendation{component.Key, "Close every month in surplus", fmt.Sprintf("Only %.0f of the last 6 months ended in surplus. Plan large buys and annual bills ahead so no month runs a deficit.", component.Value)}
	case "emergency":
		if full {
			return HealthRecommendation{component.Key, "Emergency fund is ready", fmt.Sprintf("%.1f months of expenses are liquid. Extra cash can be invested.", component.Value)}
		}
		gap := math.Max(0, TargetEmergencyMonths*health.AvgMonthlyExpense-health.LiquidAssets)
		return HealthRecommendation{component.Key, "Build the emergency fund", fmt.Sprintf("Liquid cash covers %.1f months. Add %.0f to reach 6 months.", component.Value, gap)}
	case "needs":
		if full {
			return HealthRecommendation{component.Key, "Needs are in proportion", fmt.Sprintf("Needs take %.1f%% of income, within the 50%% rule.", component.Value)}
		}
		return HealthRecommendation{component.Key, "Lower fixed and essential costs", fmt.Sprintf("Needs take %.1f%% of income (50%% or less is healthy). Renegotiate fixed costs, trim essentials, and tag untagged categories, which count as needs.", component.Value)}
	case "debt":
		if full {
			return HealthRecommendation{component.Key, "Stay debt free", "Card balances are small against income. Keep paying cards in full each cycle."}
		}
		return HealthRecommendation{component.Key, "Pay down debt", fmt.Sprintf("Card minimums take %.1f%% of monthly income (5%% or less scores full). Pay card balances in full each cycle.", component.Value)}
	case "spendingTrend":
		if component.Score == 0 && component.Value == 0 {
			return HealthRecommendation{component.Key, "Record six months of spending", "The spending trend needs expense in each half of the last 6 months."}
		}
		if full {
			return HealthRecommendation{component.Key, "Spending is steady", fmt.Sprintf("Spending moved %+.1f%% on the 3 months before.", component.Value)}
		}
		return HealthRecommendation{component.Key, "Rein in rising spending", fmt.Sprintf("Spending rose %.1f%% on the 3 months before. Find the categories that grew on Budget and set limits.", component.Value)}
	case "lending":
		if full {
			return HealthRecommendation{component.Key, "Lending is in proportion", fmt.Sprintf("Money lent is %.1f%% of your assets.", component.Value)}
		}
		return HealthRecommendation{component.Key, "Collect money lent", fmt.Sprintf("Money lent is %.1f%% of your assets (10%% or less scores full). Chase repayments before lending more.", component.Value)}
	default:
		if full {
			return HealthRecommendation{component.Key, "Well diversified", "Assets are spread evenly across cash, fund certificates and crypto."}
		}
		return HealthRecommendation{component.Key, "Diversify assets", fmt.Sprintf("Allocation is %s. Move surplus cash beyond the emergency fund into under-weighted classes.", component.Detail)}
	}
}

func clampUnit(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
