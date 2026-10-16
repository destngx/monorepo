package domain

import (
	"fmt"
	"strings"
	"time"
)

// Deterministic facts handed to the AI reports. The model explains and
// recommends; every figure here is computed by the engine (§10.1).

// BriefingFacts are the current-month numbers behind the daily briefing (WM-AI-6).
type BriefingFacts struct {
	NetWorth    float64 `json:"netWorth"`
	Assets      float64 `json:"assets"`
	Liabilities float64 `json:"liabilities"`
	Income      float64 `json:"income"`
	Expense     float64 `json:"expense"`
	CashFlow    float64 `json:"cashFlow"`
	SavingsRate float64 `json:"savingsRate"`
	// Month to date against the same days of last month and the month-end
	// expense pace (WM-DASH-2/9), as the dashboard shows them.
	ComparedDays         int          `json:"comparedDays,omitempty"`
	IncomeChangePercent  *float64     `json:"incomeChangePercent,omitempty"`
	ExpenseChangePercent *float64     `json:"expenseChangePercent,omitempty"`
	ExpensePace          *ExpensePace `json:"expensePace,omitempty"`
	// MoneyLent is what people owe the user and the user owes people (D30).
	MoneyLent MoneyLentFacts `json:"moneyLent"`
}

// WithMonthComparison adds the dashboard's same-days comparison and pace.
func (f BriefingFacts) WithMonthComparison(summary DashboardSummary) BriefingFacts {
	f.ComparedDays = summary.ComparedDays
	f.IncomeChangePercent = summary.IncomeChangePercent
	f.ExpenseChangePercent = summary.ExpenseChangePercent
	f.ExpensePace = summary.ExpensePace
	return f
}

func ComputeBriefingFacts(accounts []Account, transactions []Transaction, today time.Time) BriefingFacts {
	sheet := ComputeBalanceSheet(accounts, transactions)
	flow := ComputeCashFlow(transactions, today)
	return BriefingFacts{
		MoneyLent:   ComputeMoneyLent(transactions, today).MoneyLentFacts,
		NetWorth:    sheet.NetWorth,
		Assets:      sheet.Assets,
		Liabilities: sheet.Liabilities,
		Income:      flow.Income,
		Expense:     flow.Expense,
		CashFlow:    flow.CashFlow,
		SavingsRate: flow.SavingsRate,
	}
}

type BriefingAlert struct {
	Severity InsightSeverity `json:"severity"`
	Title    string          `json:"title"`
	Message  string          `json:"message"`
}

// BriefingPatternKinds are the dashboard pattern cards (WM-DASH-6).
var BriefingPatternKinds = []string{"spending-forecast", "portfolio-health", "what-if", "anomaly", "benchmark", "investment-strategy"}

type BriefingPattern struct {
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Briefing struct {
	Briefing string            `json:"briefing"`
	Alerts   []BriefingAlert   `json:"alerts"`
	Patterns []BriefingPattern `json:"patterns"`
	Facts    BriefingFacts     `json:"facts"`
}

// Normalize validates a briefing decoded from a model; Facts are set by the
// engine afterwards. Unknown pattern kinds and blank items are dropped.
func (b *Briefing) Normalize() error {
	b.Briefing = strings.TrimSpace(b.Briefing)
	if b.Briefing == "" {
		return fmt.Errorf("%w: briefing is empty", ErrAIOutput)
	}
	alerts := make([]BriefingAlert, 0, len(b.Alerts))
	for _, alert := range b.Alerts {
		alert.Title, alert.Message = strings.TrimSpace(alert.Title), strings.TrimSpace(alert.Message)
		if alert.Title == "" && alert.Message == "" {
			continue
		}
		alert.Severity = normalizeSeverity(alert.Severity)
		if alerts = append(alerts, alert); len(alerts) == briefingMaxAlerts {
			break
		}
	}
	b.Alerts = alerts
	patterns := make([]BriefingPattern, 0, len(b.Patterns))
	seen := map[string]bool{}
	for _, pattern := range b.Patterns {
		pattern.Kind = strings.ToLower(strings.TrimSpace(pattern.Kind))
		pattern.Title, pattern.Content = strings.TrimSpace(pattern.Title), strings.TrimSpace(pattern.Content)
		if seen[pattern.Kind] || pattern.Content == "" || oneOf(pattern.Kind, BriefingPatternKinds, "") == "" {
			continue
		}
		seen[pattern.Kind] = true
		if patterns = append(patterns, pattern); len(patterns) == briefingMaxPatterns {
			break
		}
	}
	b.Patterns = patterns
	return nil
}

func normalizeSeverity(severity InsightSeverity) InsightSeverity {
	return InsightSeverity(oneOf(strings.ToLower(strings.TrimSpace(string(severity))), severityNames(), string(InsightInfo)))
}

// BriefingFormatInstruction is the output contract of the daily briefing.
var BriefingFormatInstruction = `IMPORTANT: Respond with ONLY a valid JSON object (no markdown fences, no explanation) in this exact format:
{
  "briefing": "Markdown briefing, 3-6 short paragraphs or bullet groups",
  "alerts": [{"severity": "<info|warning|success|critical>", "title": "Short title", "message": "One or two sentences"}],
  "patterns": [{"kind": "<one of: ` + strings.Join(BriefingPatternKinds, ", ") + `>", "title": "Card title", "content": "1-3 sentences"}]
}

Rules:
- At most 4 alerts; only include real issues or wins found in the data.
- At most one pattern per kind; skip a kind when the data does not support it.
- Quote amounts exactly as given in the FACTS and context; never compute new ledger totals.`
