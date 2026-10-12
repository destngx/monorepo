package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// BudgetCategoryStats are per-category numbers behind the budget advisor (WM-BUD-5).
type BudgetCategoryStats struct {
	Category     string  `json:"category"`
	CurrentLimit float64 `json:"currentLimit"`
	SpentSoFar   float64 `json:"spentSoFar"`
	AverageSpend float64 `json:"averageSpend"`
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
	Month             string                `json:"month"`
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
)

// ComputeAdvisorFacts derives the budget advisor inputs for the month of
// today: per-category average spend over the previous full months, categories
// burning faster than usual, and a 30-day liquid balance projection that
// extends the average monthly cash flow from today's cash balance.
func ComputeAdvisorFacts(items []BudgetItem, accounts []Account, transactions []Transaction, today time.Time) AdvisorFacts {
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	report := ComputeBudgetReport(items, transactions, month, today)

	averages := map[string]float64{}
	for i := 1; i <= advisorLookbackMonths; i++ {
		past := month.AddDate(0, -i, 0)
		for _, line := range ComputeBudgetReport(items, transactions, past, today).Lines {
			if line.CategoryType == CategoryTypeExpense {
				averages[line.Category] += line.Spent / advisorLookbackMonths
			}
		}
	}

	facts := AdvisorFacts{Month: report.Month, Categories: []BudgetCategoryStats{}, UnusualVelocity: []VelocityFlag{}, ProjectedSpend: report.ProjectedSpend}
	for _, line := range report.Lines {
		if line.CategoryType != CategoryTypeExpense {
			continue
		}
		average := math.Round(averages[line.Category])
		facts.Categories = append(facts.Categories, BudgetCategoryStats{Category: line.Category, CurrentLimit: line.Limit, SpentSoFar: line.Spent, AverageSpend: average})
		if report.DaysElapsed == 0 || average <= 0 || line.Spent <= 0 {
			continue
		}
		dailyNow := line.Spent / float64(report.DaysElapsed)
		dailyAverage := average / averageDaysPerMonth
		if ratio := dailyNow / dailyAverage; ratio >= velocityRatio {
			facts.UnusualVelocity = append(facts.UnusualVelocity, VelocityFlag{Category: line.Category, DailyNow: math.Round(dailyNow), DailyAverage: math.Round(dailyAverage), Ratio: round2(ratio)})
		}
	}
	sort.Slice(facts.UnusualVelocity, func(a, b int) bool { return facts.UnusualVelocity[a].Ratio > facts.UnusualVelocity[b].Ratio })

	income, expense := averageIncomeExpense(transactions, today, advisorLookbackMonths)
	facts.AverageIncome = income
	facts.ForecastedSavings = math.Round(income - report.ProjectedSpend)
	daily := (income - expense) / averageDaysPerMonth
	balance := AssetAllocation(accounts)[AssetClassCash]
	facts.Projection = make([]ProjectionPoint, projectionDays)
	for day := 1; day <= projectionDays; day++ {
		facts.Projection[day-1] = ProjectionPoint{Date: today.AddDate(0, 0, day).Format(time.DateOnly), Balance: math.Round(balance + daily*float64(day))}
	}
	return facts
}

type SuggestedLimit struct {
	Category       string  `json:"category"`
	CurrentLimit   float64 `json:"currentLimit"`
	AverageSpend   float64 `json:"averageSpend"`
	SuggestedLimit float64 `json:"suggestedLimit"`
	Rationale      string  `json:"rationale"`
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
// current limit and average; projection, savings and velocity come from facts.
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
		limits = append(limits, SuggestedLimit{
			Category:       stat.Category,
			CurrentLimit:   stat.CurrentLimit,
			AverageSpend:   stat.AverageSpend,
			SuggestedLimit: math.Round(limit.SuggestedLimit),
			Rationale:      strings.TrimSpace(limit.Rationale),
		})
	}
	a.SuggestedLimits = limits
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
- Base suggested limits on averageSpend and spentSoFar; round to thousands of VND.
- Mention every category in FACTS.unusualVelocity in an alert of type "spike".
- Use FACTS.projection and FACTS.forecastedSavings as given; never recompute them.`
