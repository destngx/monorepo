package domain

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
)

// SpendTrend compares a category's last full months with the months before
// them this year (WM-BUD-9).
type SpendTrend string

const (
	SpendTrendRising  SpendTrend = "rising"
	SpendTrendFalling SpendTrend = "falling"
	SpendTrendSteady  SpendTrend = "steady"
	// SpendTrendUnknown: no earlier full month this year to compare with.
	SpendTrendUnknown SpendTrend = "unknown"
)

// SpendPattern is how a category's spend is spread over its months (WM-BUD-9).
type SpendPattern string

const (
	SpendPatternSteady     SpendPattern = "steady"
	SpendPatternVariable   SpendPattern = "variable"
	SpendPatternOccasional SpendPattern = "occasional"
)

// MonthSpend is a category's spend in one full month.
type MonthSpend struct {
	Month string  `json:"month"`
	Spent float64 `json:"spent"`
}

// BudgetCategoryStats are per-category numbers behind the budget advisor
// (WM-BUD-5, WM-BUD-9). AverageSpend covers the last advisorLookbackMonths
// full months; YearToDateAverage the full months since January.
type BudgetCategoryStats struct {
	Category          string       `json:"category"`
	CurrentLimit      float64      `json:"currentLimit"`
	SpentSoFar        float64      `json:"spentSoFar"`
	AverageSpend      float64      `json:"averageSpend"`
	YearToDateAverage float64      `json:"yearToDateAverage"`
	MonthlySpend      []MonthSpend `json:"monthlySpend"`
	Trend             SpendTrend   `json:"trend"`
	Pattern           SpendPattern `json:"pattern"`
	// BaselineLimit is the engine's suggested monthly limit; 0 without history.
	BaselineLimit float64 `json:"baselineLimit"`
}

// VelocityFlag marks a category spending much faster than its own average.
type VelocityFlag struct {
	Category     string  `json:"category"`
	DailyNow     float64 `json:"dailyNow"`
	DailyAverage float64 `json:"dailyAverage"`
	Ratio        float64 `json:"ratio"`
}

type ProjectionPoint struct {
	Date    string  `json:"date"`
	Balance float64 `json:"balance"`
	Note    string  `json:"note,omitempty"`
}

type AdvisorFacts struct {
	Month string `json:"month"`
	// YearToDateMonths is the number of full months this year behind
	// each category's YearToDateAverage.
	YearToDateMonths  int                   `json:"yearToDateMonths"`
	Categories        []BudgetCategoryStats `json:"categories"`
	UnusualVelocity   []VelocityFlag        `json:"unusualVelocity"`
	Projection        []ProjectionPoint     `json:"projection"`
	AverageIncome     float64               `json:"averageIncome"`
	ProjectedSpend    float64               `json:"projectedSpend"`
	ForecastedSavings float64               `json:"forecastedSavings"`
}

const (
	advisorLookbackMonths = 3
	projectionDays        = 30
	velocityRatio         = 1.5
	averageDaysPerMonth   = 365.25 / 12
	// trendChange is the relative move of the recent average against the
	// earlier months that counts as rising or falling.
	trendChange = 0.15
	// variableSpread is the coefficient of variation above which monthly
	// spend counts as variable.
	variableSpread = 0.5
	// baselineStep rounds baseline limits up (VND).
	baselineStep = 10_000
	// suggestionChange is the relative gap between a baseline and the
	// current limit worth suggesting when the model left the category out.
	suggestionChange = 0.10
)

// ComputeAdvisorFacts derives the budget advisor inputs for the month of
// today: per-category spend over the previous full months (the recent
// window and the year to date) with its trend, pattern and a baseline limit,
// categories burning faster than usual, and a 30-day liquid balance
// projection that extends the average monthly cash flow from today's cash
// balance.
func ComputeAdvisorFacts(items []BudgetItem, accounts []Account, transactions []Transaction, today time.Time) AdvisorFacts {
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	report := ComputeBudgetReport(items, transactions, month, today)

	// History runs over full months: the recent window, extended back to
	// January when the year has more full months than the window.
	yearToDateMonths := int(month.Month()) - 1
	historyMonths := max(advisorLookbackMonths, yearToDateMonths)
	history := map[string][]float64{}
	names := map[string]string{}
	order := []string{}
	for i := historyMonths; i >= 1; i-- {
		for _, line := range ComputeBudgetReport(items, transactions, month.AddDate(0, -i, 0), today).Lines {
			if !line.CountsAsExpense() {
				continue
			}
			key := normalizeCategory(line.Category)
			if _, ok := history[key]; !ok {
				history[key] = make([]float64, historyMonths)
				names[key] = line.Category
				order = append(order, key)
			}
			history[key][historyMonths-i] = line.Spent
		}
	}

	facts := AdvisorFacts{Month: report.Month, YearToDateMonths: yearToDateMonths, Categories: []BudgetCategoryStats{}, UnusualVelocity: []VelocityFlag{}, ProjectedSpend: report.ProjectedSpend}
	listed := map[string]bool{}
	for _, line := range report.Lines {
		if !line.CountsAsExpense() {
			continue
		}
		key := normalizeCategory(line.Category)
		listed[key] = true
		stats := categoryStats(line.Category, line.Limit, line.Spent, history[key], month, yearToDateMonths)
		facts.Categories = append(facts.Categories, stats)
		average := stats.AverageSpend
		if report.DaysElapsed == 0 || average <= 0 || line.Spent <= 0 {
			continue
		}
		dailyNow := line.Spent / float64(report.DaysElapsed)
		dailyAverage := average / averageDaysPerMonth
		if ratio := dailyNow / dailyAverage; ratio >= velocityRatio {
			facts.UnusualVelocity = append(facts.UnusualVelocity, VelocityFlag{Category: line.Category, DailyNow: math.Round(dailyNow), DailyAverage: math.Round(dailyAverage), Ratio: round2(ratio)})
		}
	}
	// Categories spent on earlier but not yet this month, and without a
	// limit, still need one planned.
	for _, key := range order {
		if listed[key] || mean(history[key]) <= 0 {
			continue
		}
		facts.Categories = append(facts.Categories, categoryStats(names[key], 0, 0, history[key], month, yearToDateMonths))
	}
	sort.Slice(facts.UnusualVelocity, func(a, b int) bool { return facts.UnusualVelocity[a].Ratio > facts.UnusualVelocity[b].Ratio })

	income, expense := averageIncomeExpense(transactions, today, advisorLookbackMonths)
	facts.AverageIncome = income
	facts.ForecastedSavings = math.Round(income - report.ProjectedSpend)
	daily := (income - expense) / averageDaysPerMonth
	balance := liquidBalance(accounts)
	facts.Projection = make([]ProjectionPoint, projectionDays)
	for day := 1; day <= projectionDays; day++ {
		facts.Projection[day-1] = ProjectionPoint{Date: today.AddDate(0, 0, day).Format(time.DateOnly), Balance: math.Round(balance + daily*float64(day))}
	}
	return facts
}

// categoryStats reads one category's full-month spend, oldest first and
// ending last month. The recent window is its last advisorLookbackMonths
// months; the year to date its last yearToDateMonths months; the trend
// compares the recent window with the months before it this year.
func categoryStats(category string, limit, spentSoFar float64, spend []float64, month time.Time, yearToDateMonths int) BudgetCategoryStats {
	months := max(advisorLookbackMonths, yearToDateMonths)
	if spend == nil {
		spend = make([]float64, months)
	}
	recent := spend[months-advisorLookbackMonths:]
	yearToDate := spend[months-yearToDateMonths:]
	stats := BudgetCategoryStats{
		Category:          category,
		CurrentLimit:      limit,
		SpentSoFar:        spentSoFar,
		AverageSpend:      math.Round(mean(recent)),
		YearToDateAverage: math.Round(mean(yearToDate)),
		MonthlySpend:      make([]MonthSpend, months),
		Trend:             SpendTrendUnknown,
		Pattern:           spendPattern(spend),
	}
	for i, spent := range spend {
		stats.MonthlySpend[i] = MonthSpend{Month: MonthKey(month.AddDate(0, i-months, 0)), Spent: spent}
	}
	if earlier := yearToDateMonths - advisorLookbackMonths; earlier > 0 {
		stats.Trend = spendTrend(mean(recent), mean(yearToDate[:earlier]))
	}
	baseline := mean(recent)
	switch stats.Pattern {
	case SpendPatternOccasional:
		baseline = mean(spend)
	case SpendPatternVariable:
		baseline = max(baseline, mean(yearToDate))
	}
	stats.BaselineLimit = math.Ceil(math.Round(baseline)/baselineStep) * baselineStep
	return stats
}

func spendTrend(recent, earlier float64) SpendTrend {
	switch {
	case earlier <= 0 && recent <= 0:
		return SpendTrendSteady
	case earlier <= 0 || recent >= earlier*(1+trendChange):
		return SpendTrendRising
	case recent <= earlier*(1-trendChange):
		return SpendTrendFalling
	default:
		return SpendTrendSteady
	}
}

// spendPattern looks at the months from the first one with spend: spend in
// fewer than half of them is occasional, a wide spread is variable.
func spendPattern(spend []float64) SpendPattern {
	first := slices.IndexFunc(spend, func(spent float64) bool { return spent > 0 })
	if first < 0 {
		return SpendPatternSteady
	}
	active := spend[first:]
	withSpend := 0
	for _, spent := range active {
		if spent > 0 {
			withSpend++
		}
	}
	if withSpend*2 < len(active) {
		return SpendPatternOccasional
	}
	if average := mean(active); stdDev(active, average)/average > variableSpread {
		return SpendPatternVariable
	}
	return SpendPatternSteady
}

// SuggestedLimit is a limit proposal: the model's, or the engine's baseline
// for a category the model left out (WM-BUD-9). The figures beside the
// suggestion always come from the engine.
type SuggestedLimit struct {
	Category          string       `json:"category"`
	CurrentLimit      float64      `json:"currentLimit"`
	AverageSpend      float64      `json:"averageSpend"`
	YearToDateAverage float64      `json:"yearToDateAverage"`
	Trend             SpendTrend   `json:"trend"`
	Pattern           SpendPattern `json:"pattern"`
	BaselineLimit     float64      `json:"baselineLimit"`
	SuggestedLimit    float64      `json:"suggestedLimit"`
	Rationale         string       `json:"rationale"`
}

func newSuggestedLimit(stat BudgetCategoryStats, limit float64, rationale string) SuggestedLimit {
	return SuggestedLimit{
		Category:          stat.Category,
		CurrentLimit:      stat.CurrentLimit,
		AverageSpend:      stat.AverageSpend,
		YearToDateAverage: stat.YearToDateAverage,
		Trend:             stat.Trend,
		Pattern:           stat.Pattern,
		BaselineLimit:     stat.BaselineLimit,
		SuggestedLimit:    math.Round(limit),
		Rationale:         rationale,
	}
}

// baselineSuggestions proposes the baseline for categories the model left
// out whose baseline is at least suggestionChange away from the current
// limit, the largest relative change first.
func baselineSuggestions(categories []BudgetCategoryStats, suggested map[string]bool) []SuggestedLimit {
	type candidate struct {
		stat   BudgetCategoryStats
		change float64
	}
	candidates := []candidate{}
	for _, stat := range categories {
		if suggested[strings.ToLower(stat.Category)] || stat.BaselineLimit <= 0 {
			continue
		}
		change := math.Abs(stat.BaselineLimit-stat.CurrentLimit) / max(stat.BaselineLimit, stat.CurrentLimit)
		if change >= suggestionChange {
			candidates = append(candidates, candidate{stat, change})
		}
	}
	sort.SliceStable(candidates, func(a, b int) bool { return candidates[a].change > candidates[b].change })
	limits := make([]SuggestedLimit, len(candidates))
	for i, c := range candidates {
		limits[i] = newSuggestedLimit(c.stat, c.stat.BaselineLimit, baselineRationale(c.stat))
	}
	return limits
}

// baselineRationale explains a baseline without amounts, so the privacy
// mask has nothing to hide.
func baselineRationale(stat BudgetCategoryStats) string {
	switch {
	case stat.Pattern == SpendPatternOccasional:
		return "Spent in only some months; sets aside its monthly average so the big months are covered."
	case stat.Pattern == SpendPatternVariable:
		return "Swings month to month; uses the higher of the 3-month and year-to-date averages."
	case stat.Trend == SpendTrendRising:
		return "Rising: the last 3 months ran above earlier in the year; follows the 3-month average."
	case stat.Trend == SpendTrendFalling:
		return "Falling: the last 3 months ran below earlier in the year; keeps the lower 3-month average."
	default:
		return "Steady spend; follows the 3-month average."
	}
}

// AdvisorAlertTypes are the budget advisor alert kinds (WM-BUD-5).
var AdvisorAlertTypes = []string{"overspend", "spike", "bill", "opportunity", "goal"}

type AdvisorAlert struct {
	Type     string          `json:"type"`
	Severity InsightSeverity `json:"severity"`
	Title    string          `json:"title"`
	Message  string          `json:"message"`
	Category string          `json:"category,omitempty"`
}

type GoalImpact struct {
	Goal   string `json:"goal"`
	Impact string `json:"impact"`
}

type BudgetAdvice struct {
	Month             string            `json:"month"`
	Briefing          string            `json:"briefing"`
	SuggestedLimits   []SuggestedLimit  `json:"suggestedLimits"`
	Projection        []ProjectionPoint `json:"projection"`
	Alerts            []AdvisorAlert    `json:"alerts"`
	GoalImpact        []GoalImpact      `json:"goalImpact"`
	ForecastedSavings float64           `json:"forecastedSavings"`
	UnusualVelocity   []VelocityFlag    `json:"unusualVelocity"`
}

const (
	briefingMaxAlerts   = 4
	briefingMaxPatterns = 6
	advisorMaxAlerts    = 6
	advisorMaxGoals     = 5
)

// ApplyFacts validates advice decoded from a model against the engine facts:
// suggested limits must name a known expense category and carry the engine's
// current limit, averages, trend, pattern and baseline; categories the model
// left out get their baseline when it moves the limit enough; projection,
// savings and velocity come from facts.
func (a *BudgetAdvice) ApplyFacts(facts AdvisorFacts) error {
	a.Briefing = strings.TrimSpace(a.Briefing)
	if a.Briefing == "" {
		return fmt.Errorf("%w: advice has no briefing", ErrAIOutput)
	}
	stats := map[string]BudgetCategoryStats{}
	for _, category := range facts.Categories {
		stats[strings.ToLower(category.Category)] = category
	}
	limits := make([]SuggestedLimit, 0, len(a.SuggestedLimits))
	seen := map[string]bool{}
	for _, limit := range a.SuggestedLimits {
		key := strings.ToLower(strings.TrimSpace(limit.Category))
		stat, ok := stats[key]
		if !ok || seen[key] || limit.SuggestedLimit < 0 {
			continue
		}
		seen[key] = true
		limits = append(limits, newSuggestedLimit(stat, limit.SuggestedLimit, strings.TrimSpace(limit.Rationale)))
	}
	a.SuggestedLimits = append(limits, baselineSuggestions(facts.Categories, seen)...)
	alerts := make([]AdvisorAlert, 0, len(a.Alerts))
	for _, alert := range a.Alerts {
		alert.Title, alert.Message = strings.TrimSpace(alert.Title), strings.TrimSpace(alert.Message)
		if alert.Title == "" && alert.Message == "" {
			continue
		}
		alert.Type = oneOf(strings.ToLower(strings.TrimSpace(alert.Type)), AdvisorAlertTypes, "opportunity")
		alert.Severity = normalizeSeverity(alert.Severity)
		alert.Category = strings.TrimSpace(alert.Category)
		if alerts = append(alerts, alert); len(alerts) == advisorMaxAlerts {
			break
		}
	}
	a.Alerts = alerts
	goals := make([]GoalImpact, 0, len(a.GoalImpact))
	for _, goal := range a.GoalImpact {
		goal.Goal, goal.Impact = strings.TrimSpace(goal.Goal), strings.TrimSpace(goal.Impact)
		if goal.Goal == "" || goal.Impact == "" {
			continue
		}
		if goals = append(goals, goal); len(goals) == advisorMaxGoals {
			break
		}
	}
	a.GoalImpact = goals
	a.Month = facts.Month
	a.Projection = facts.Projection
	a.ForecastedSavings = facts.ForecastedSavings
	a.UnusualVelocity = facts.UnusualVelocity
	return nil
}

// AdvisorFormatInstruction is the output contract of the budget advisor.
var AdvisorFormatInstruction = `IMPORTANT: Respond with ONLY a valid JSON object (no markdown fences, no explanation) in this exact format:
{
  "briefing": "Markdown summary of the month so far and what to do next",
  "suggestedLimits": [{"category": "Exact category name from FACTS", "suggestedLimit": 1500000, "rationale": "One sentence"}],
  "alerts": [{"type": "<one of: ` + strings.Join(AdvisorAlertTypes, ", ") + `>", "severity": "<info|warning|success|critical>", "title": "Short title", "message": "One or two sentences", "category": "Optional category"}],
  "goalImpact": [{"goal": "Goal name", "impact": "How this month's spending affects it"}]
}

Rules:
- Suggest limits only for categories listed in FACTS.categories, using their exact names.
- Each category has averageSpend (last 3 full months), yearToDateAverage (full months since January; FACTS.yearToDateMonths of them), monthlySpend, trend (last 3 months against earlier this year: rising, falling, steady or unknown), pattern (steady, variable or occasional) and baselineLimit (the engine's limit from them).
- Start each suggested limit from baselineLimit. Move away from it only for a reason in the data (spentSoFar, recent transactions, goals) and say why in the rationale.
- Name the trend or pattern in the rationale when it drives the limit. For occasional spend, plan a monthly set-aside rather than a cut.
- Round suggested limits to thousands of VND.
- Mention every category in FACTS.unusualVelocity in an alert of type "spike".
- Use FACTS.projection and FACTS.forecastedSavings as given; never recompute them.`
