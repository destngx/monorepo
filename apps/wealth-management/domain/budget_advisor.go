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
	// Nature is the category's tag (fixed, essential, flexible); empty when untagged.
	Nature CategoryNature `json:"nature,omitempty"`
	// RecommendedLimit, Direction and Reason are the engine's quality-of-life
	// recommendation (WM-BUD-10); Reason carries no amounts.
	RecommendedLimit float64        `json:"recommendedLimit"`
	Direction        LimitDirection `json:"direction"`
	Reason           string         `json:"reason"`
	// history is the full-month spend behind the averages, oldest first.
	history          []float64
	yearToDateMonths int
}

// LimitDirection says which way a recommendation moves a limit (WM-BUD-10).
type LimitDirection string

const (
	LimitIncrease LimitDirection = "increase"
	LimitLower    LimitDirection = "lower"
	LimitKeep     LimitDirection = "keep"
)

// SavingsCheck decides whether wants keep their usual level (WM-BUD-10):
// average income less the limits with wants at their usual level must cover
// Required, the higher of what dated goals need a month and the
// advisorSavingsRate share of income.
type SavingsCheck struct {
	AverageIncome float64 `json:"averageIncome"`
	GoalNeed      float64 `json:"goalNeed"`
	Required      float64 `json:"required"`
	// UsualSpend is the sum of the limits with wants at their usual level.
	UsualSpend float64 `json:"usualSpend"`
	OnTrack    bool    `json:"onTrack"`
	// RecommendedSpend is the sum of the recommended limits.
	RecommendedSpend float64 `json:"recommendedSpend"`
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
	SavingsCheck      SavingsCheck          `json:"savingsCheck"`
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
	// limitStep rounds recommended limits up (VND).
	limitStep = 10_000
	// limitCushion is the headroom above the usual cost a limit may keep
	// before it is worth lowering; wants get none while savings are short.
	limitCushion = 0.10
	// increaseTolerance is how far a limit may sit below the usual cost
	// before it is worth raising.
	increaseTolerance = 0.05
	// advisorSavingsRate is the savings-rate target of the health score
	// (WM-HEA-2) that wants must leave room for.
	advisorSavingsRate = 0.30
)

// ComputeAdvisorFacts derives the budget advisor inputs for the month of
// today: per-category spend over the previous full months (the recent
// window and the year to date) with its trend, pattern and a
// quality-of-life limit recommendation, the savings check behind it,
// categories burning faster than usual, and a 30-day liquid balance
// projection that extends the average monthly cash flow from today's cash
// balance. goalNeed is what dated goals need saved a month.
func ComputeAdvisorFacts(items []BudgetItem, accounts []Account, transactions []Transaction, today time.Time, goalNeed float64) AdvisorFacts {
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
	natures := map[string]CategoryNature{}
	for _, t := range transactions {
		if t.CategoryNature != "" {
			natures[normalizeCategory(t.Category)] = t.CategoryNature
		}
	}
	for i := range facts.Categories {
		facts.Categories[i].Nature = natures[normalizeCategory(facts.Categories[i].Category)]
	}
	facts.SavingsCheck = recommendLimits(facts.Categories, income, goalNeed)
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
	stats.history, stats.yearToDateMonths = spend, yearToDateMonths
	return stats
}

// recommendLimits applies the quality-of-life rules (WM-BUD-10): needs are
// budgeted at what they really cost and never cut below it; wants keep
// their usual level while savings are on track and fall back to their
// lower average only when savings are short and they are rising or
// swinging; limits well above the usual cost are lowered to free money.
func recommendLimits(categories []BudgetCategoryStats, income, goalNeed float64) SavingsCheck {
	check := SavingsCheck{AverageIncome: math.Round(income), GoalNeed: math.Round(goalNeed)}
	check.Required = math.Round(max(goalNeed, income*advisorSavingsRate))
	for _, stat := range categories {
		check.UsualSpend += recommendLimit(stat, true).RecommendedLimit
	}
	check.OnTrack = income > 0 && income-check.UsualSpend >= check.Required
	for i := range categories {
		categories[i] = recommendLimit(categories[i], check.OnTrack)
		check.RecommendedSpend += categories[i].RecommendedLimit
	}
	return check
}

// recommendLimit sets one category's recommended limit, direction and reason.
func recommendLimit(stat BudgetCategoryStats, savingsOnTrack bool) BudgetCategoryStats {
	stat.RecommendedLimit, stat.Direction = stat.CurrentLimit, LimitKeep
	if mean(stat.history) <= 0 {
		stat.Reason = "No spend in the last full months yet; nothing to base a change on."
		if stat.CurrentLimit > 0 {
			stat.Reason = "No spend in the last full months; check the limit is still needed."
		}
		return stat
	}
	recent, yearToDate := stat.AverageSpend, stat.YearToDateAverage
	if stat.yearToDateMonths == 0 {
		yearToDate = recent
	}
	usual, lean := max(recent, yearToDate), min(recent, yearToDate)
	if stat.Pattern == SpendPatternOccasional {
		usual, lean = mean(stat.history), mean(stat.history)
	}
	target, cushion := usual, limitCushion
	switch stat.Nature {
	case NatureFixed:
		target = recent
		if stat.Pattern == SpendPatternOccasional {
			target = usual
		}
	case NatureFlexible:
		if !savingsOnTrack {
			cushion = 0
			if stat.Trend == SpendTrendRising || stat.Pattern == SpendPatternVariable {
				target = lean
			}
		}
	case NatureEssential:
	default:
		if stat.Pattern != SpendPatternVariable && stat.Pattern != SpendPatternOccasional {
			target = recent
		}
	}
	target = roundUpLimit(target)
	ceiling := roundUpLimit(target * (1 + cushion))
	switch {
	case stat.CurrentLimit < target*(1-increaseTolerance):
		stat.RecommendedLimit, stat.Direction = target, LimitIncrease
	case stat.CurrentLimit > ceiling:
		stat.RecommendedLimit, stat.Direction = ceiling, LimitLower
	}
	stat.Reason = recommendationReason(stat, savingsOnTrack)
	return stat
}

func roundUpLimit(value float64) float64 {
	return math.Ceil(math.Round(value)/limitStep) * limitStep
}

// recommendationReason explains a recommendation without amounts, so the
// privacy mask has nothing to hide.
func recommendationReason(stat BudgetCategoryStats, savingsOnTrack bool) string {
	var reason string
	switch {
	case stat.Pattern == SpendPatternOccasional && stat.Direction != LimitKeep:
		reason = "Spent in only some months; sets aside its monthly average so the big months are covered."
	case stat.Direction == LimitKeep:
		reason = "The limit fits what you usually spend."
	case stat.Nature == NatureFixed && stat.Direction == LimitIncrease:
		reason = "A fixed cost above its limit; budget what it really costs."
	case stat.Nature == NatureFixed:
		reason = "The fixed cost came down; the limit can follow."
	case stat.Nature == NatureEssential && stat.Direction == LimitIncrease:
		reason = "A need that costs more than its limit; needs are budgeted at their usual cost, not cut."
	case stat.Nature == NatureFlexible && stat.Direction == LimitIncrease && savingsOnTrack:
		reason = "Savings are on track, so this want keeps its usual level."
	case stat.Nature == NatureFlexible && stat.Direction == LimitIncrease:
		reason = "Below what you usually spend even with savings short; a limit you will break helps no one."
	case stat.Nature == NatureFlexible && !savingsOnTrack && stat.Trend == SpendTrendRising:
		reason = "Savings are short and this want is rising; back to its lower average."
	case stat.Nature == NatureFlexible && !savingsOnTrack && stat.Pattern == SpendPatternVariable:
		reason = "Savings are short and this want swings month to month; back to its lower average."
	case stat.Direction == LimitIncrease && stat.Trend == SpendTrendRising:
		reason = "Rising: the last 3 months ran above earlier in the year."
	case stat.Direction == LimitIncrease:
		reason = "Runs above its limit most months."
	default:
		reason = "The limit sits well above what you usually spend; the slack can go to goals."
	}
	if stat.Nature == "" {
		reason += " Tag it fixed, essential or flexible for needs-and-wants advice."
	}
	return reason
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

// SuggestedLimit is a limit proposal: the model's, or the engine's
// recommendation for a category the model left out (WM-BUD-9, WM-BUD-10).
// The figures beside the suggestion always come from the engine; Direction
// compares the suggestion with the current limit.
type SuggestedLimit struct {
	Category          string         `json:"category"`
	Nature            CategoryNature `json:"nature,omitempty"`
	CurrentLimit      float64        `json:"currentLimit"`
	AverageSpend      float64        `json:"averageSpend"`
	YearToDateAverage float64        `json:"yearToDateAverage"`
	Trend             SpendTrend     `json:"trend"`
	Pattern           SpendPattern   `json:"pattern"`
	RecommendedLimit  float64        `json:"recommendedLimit"`
	SuggestedLimit    float64        `json:"suggestedLimit"`
	Direction         LimitDirection `json:"direction"`
	Rationale         string         `json:"rationale"`
}

func newSuggestedLimit(stat BudgetCategoryStats, limit float64, rationale string) SuggestedLimit {
	suggestion := SuggestedLimit{
		Category:          stat.Category,
		Nature:            stat.Nature,
		CurrentLimit:      stat.CurrentLimit,
		AverageSpend:      stat.AverageSpend,
		YearToDateAverage: stat.YearToDateAverage,
		Trend:             stat.Trend,
		Pattern:           stat.Pattern,
		RecommendedLimit:  stat.RecommendedLimit,
		SuggestedLimit:    math.Round(limit),
		Direction:         LimitKeep,
		Rationale:         rationale,
	}
	switch {
	case suggestion.SuggestedLimit > stat.CurrentLimit:
		suggestion.Direction = LimitIncrease
	case suggestion.SuggestedLimit < stat.CurrentLimit:
		suggestion.Direction = LimitLower
	}
	return suggestion
}

// engineSuggestions proposes the recommendation for every category the
// model left out whose limit should move, the largest relative change first.
func engineSuggestions(categories []BudgetCategoryStats, suggested map[string]bool) []SuggestedLimit {
	moves := []BudgetCategoryStats{}
	for _, stat := range categories {
		if !suggested[strings.ToLower(stat.Category)] && stat.Direction != LimitKeep {
			moves = append(moves, stat)
		}
	}
	change := func(stat BudgetCategoryStats) float64 {
		return math.Abs(stat.RecommendedLimit-stat.CurrentLimit) / max(stat.RecommendedLimit, stat.CurrentLimit)
	}
	sort.SliceStable(moves, func(a, b int) bool { return change(moves[a]) > change(moves[b]) })
	limits := make([]SuggestedLimit, len(moves))
	for i, stat := range moves {
		limits[i] = newSuggestedLimit(stat, stat.RecommendedLimit, stat.Reason)
	}
	return limits
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
	SavingsCheck      SavingsCheck      `json:"savingsCheck"`
	SuggestedLimits   []SuggestedLimit  `json:"suggestedLimits"`
	Projection        []ProjectionPoint `json:"projection"`
	Alerts            []AdvisorAlert    `json:"alerts"`
	GoalImpact        []GoalImpact      `json:"goalImpact"`
	ForecastedSavings float64           `json:"forecastedSavings"`
	UnusualVelocity   []VelocityFlag    `json:"unusualVelocity"`
	// Recommendations, Outcomes and Recorded feed the recommendations list
	// for the Budget page (WM-BUD-11, WM-REC-2).
	Recommendations []RecommendationProposal `json:"recommendations,omitempty"`
	Outcomes        []RecommendationOutcome  `json:"outcomes,omitempty"`
	Recorded        *RecommendationRecord    `json:"recorded,omitempty"`
}

const (
	briefingMaxAlerts   = 4
	briefingMaxPatterns = 6
	advisorMaxAlerts    = 6
	advisorMaxGoals     = 5
)

// ApplyFacts validates advice decoded from a model against the engine facts:
// suggested limits must name a known expense category and carry the engine's
// current limit, averages, trend, pattern and recommendation; categories the
// model left out get the engine's recommendation when it moves the limit;
// projection, savings and velocity come from facts.
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
	a.SuggestedLimits = append(limits, engineSuggestions(facts.Categories, seen)...)
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
	a.Recommendations = NormalizeProposals(a.Recommendations)
	a.Outcomes = NormalizeOutcomes(a.Outcomes)
	a.Recorded = nil
	a.Month = facts.Month
	a.SavingsCheck = facts.SavingsCheck
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
- Each category has averageSpend (last 3 full months), yearToDateAverage (full months since January; FACTS.yearToDateMonths of them), monthlySpend, trend (last 3 months against earlier this year: rising, falling, steady or unknown), pattern (steady, variable or occasional), nature (fixed, essential, flexible; empty when untagged) and the engine's recommendedLimit, direction (increase, lower, keep) and reason.
- Quality of life is a principle: never suggest a fixed or essential limit below what it usually costs; let wants (flexible) keep their usual level while FACTS.savingsCheck.onTrack is true; trim wants only when savings are short, starting with rising or swinging ones; lower limits that sit well above usual spend so the slack goes to goals.
- Start each suggested limit from recommendedLimit. Move away from it only for a reason in the data (spentSoFar, recent transactions, goals) and say why in the rationale. Include categories whose direction is increase or lower.
- Name the trend, pattern or nature in the rationale when it drives the limit. For occasional spend, plan a monthly set-aside rather than a cut.
- Recommendations are for actions worth tracking over the coming weeks that a limit alone does not do (cook at home more while a want is rising and savings are short, renegotiate or switch a fixed cost, cancel a subscription behind an unused limit); never repeat a suggested limit as a recommendation.
- Round suggested limits to thousands of VND.
- Mention every category in FACTS.unusualVelocity in an alert of type "spike".
- Use FACTS.projection and FACTS.forecastedSavings as given; never recompute them.`
