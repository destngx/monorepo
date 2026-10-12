package ai

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var testToday = time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

func money(value float64) *float64 { return &value }

func kind(value domain.CategoryType) *domain.CategoryType { return &value }

func expense(date time.Time, payee string, category string, amount float64) domain.Transaction {
	return domain.Transaction{AccountName: "Bank", Date: date, Payee: payee, Category: category, CategoryType: kind(domain.CategoryTypeExpense), Payment: money(amount)}
}

// allPrompts echoes each key and its placeholders so tests can assert rendering.
func allPrompts() map[string]string {
	prompts := map[string]string{}
	for _, key := range append(append([]string{}, RequiredPrompts...), capturePrompts...) {
		prompts[key] = "[" + key + "]"
	}
	prompts[keyIdentity] = "You are the wealth assistant."
	prompts["tasks/account-review"] = "Net worth {{totalNetWorth}}. Accounts: {{accountsJson}}"
	prompts["tasks/suggest-category"] = "Pick one of: {{categories}}"
	prompts["actions/suggest-category"] = "Payee: {{payee}}"
	return prompts
}

func fixtureDatabase() *fakes.Database {
	return &fakes.Database{
		Accounts: []domain.Account{
			{Name: "Bank Nguyen Van A", Type: domain.AccountTypeBank, Balance: 40_000_000},
			{Name: "Card", Type: domain.AccountTypeNegativeActive, Balance: -5_000_000},
		},
		Transactions: []domain.Transaction{
			expense(testToday.AddDate(0, 0, -2), "Highlands Coffee", "Coffee", 60_000),
			expense(testToday.AddDate(0, 0, -9), "Highlands Coffee", "Coffee", 55_000),
			expense(testToday.AddDate(0, 0, -20), "Highlands Coffee", "Food", 80_000),
			expense(testToday.AddDate(0, 0, -3), "Grab", "Transport", 120_000),
		},
		Categories: []domain.Category{
			{Name: "Coffee", Type: domain.CategoryTypeExpense},
			{Name: "Food", Type: domain.CategoryTypeExpense},
			{Name: "Transport", Type: domain.CategoryTypeExpense},
			{Name: "Uncategorized", Type: domain.CategoryTypeExpense},
		},
		Budget:    []domain.BudgetItem{{Category: "Food", CategoryType: kind(domain.CategoryTypeExpense), MonthlyLimit: 3_000_000}},
		Loans:     []domain.Loan{{Name: "Car", MonthlyDebt: 2_000_000, YearlyRemaining: 20_000_000}},
		Goals:     []domain.Goal{{Name: "Trip", TargetAmount: 30_000_000, CurrentAmount: 10_000_000}},
		AIContent: domain.AIContent{Prompts: allPrompts()},
	}
}

func testConfig() domain.AIConfig {
	return domain.AIConfig{
		BaseURL:       "http://user:secret@gateway.local:8080/v1",
		Provider:      "openai",
		Models:        map[domain.AIUseCase]string{domain.AIUseCaseInsight: "primary", domain.AIUseCaseSuggest: "primary", domain.AIUseCaseChat: "primary", domain.AIUseCaseVision: "primary"},
		FallbackModel: "fallback",
		Timeout:       time.Second,
		PIINames:      []string{"Nguyen Van A"},
	}
}

func newTestService(t *testing.T, db *fakes.Database, client *scriptedClient, cache port.CacheService) *Service {
	t.Helper()
	now := func() time.Time { return testToday }
	analytics := service.NewAnalyticsService(db, domain.CardConfig{}, now)
	return New(client, db, analytics, cache, testConfig(), logger.NewTestLogger(t), Options{Now: now})
}

const validInsight = `{"summary":"Healthy month.","sections":[{"icon":"wallet","title":"Cash","content":"Cash is fine.","severity":"success"}],"actionItems":["Keep going"]}`

func TestStatusReportsMissingPromptsAndHidesCredentials(t *testing.T) {
	db := fixtureDatabase()
	delete(db.AIContent.Prompts, "tasks/loan-review")
	status := newTestService(t, db, &scriptedClient{}, nil).Status(context.Background())

	if !status.Configured || status.Endpoint != "http://gateway.local:8080" {
		t.Fatalf("unexpected status %+v", status)
	}
	if len(status.MissingKeys) != 1 || status.MissingKeys[0] != "tasks/loan-review" {
		t.Fatalf("expected the loan prompt to be missing, got %v", status.MissingKeys)
	}
}

func TestReviewRendersPromptsMasksNamesAndCaches(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: "```json\n" + validInsight + "\n```"}, {content: validInsight}}}
	svc := newTestService(t, fixtureDatabase(), client, &memoryCache{})
	ctx := context.Background()

	insight, err := svc.Review(ctx, port.ReviewRequest{Kind: port.ReviewAccounts})
	if err != nil {
		t.Fatal(err)
	}
	if insight.Summary != "Healthy month." || insight.Sections[0].Icon != "wallet" {
		t.Fatalf("unexpected insight %+v", insight)
	}
	system := client.requests[0].Messages[0].Content
	if !strings.Contains(system, "Net worth 15000000.") || !strings.Contains(system, "You are the wealth assistant.") || !strings.Contains(system, "OPERATIONAL DIRECTIVE") {
		t.Fatalf("system prompt not composed: %s", system)
	}
	if strings.Contains(system, "Nguyen Van A") || !strings.Contains(system, "Bank [name]") {
		t.Fatalf("configured name not masked: %s", system)
	}
	if !client.requests[0].JSON || !strings.Contains(client.requests[0].Messages[1].Content, "[actions/account-review]") {
		t.Fatalf("unexpected request %+v", client.requests[0])
	}

	if _, err := svc.Review(ctx, port.ReviewRequest{Kind: port.ReviewAccounts}); err != nil || len(client.requests) != 1 {
		t.Fatalf("second review should be cached: err=%v calls=%d", err, len(client.requests))
	}
	if _, err := svc.Review(ctx, port.ReviewRequest{Kind: port.ReviewAccounts, Force: true}); err != nil || len(client.requests) != 2 {
		t.Fatalf("force should bypass the cache: err=%v calls=%d", err, len(client.requests))
	}
}

func TestEveryReviewKindRenders(t *testing.T) {
	for _, reviewKind := range []port.ReviewKind{port.ReviewBudget, port.ReviewLoans, port.ReviewTransactions} {
		client := &scriptedClient{replies: []reply{{content: validInsight}}}
		if _, err := newTestService(t, fixtureDatabase(), client, nil).Review(context.Background(), port.ReviewRequest{Kind: reviewKind}); err != nil {
			t.Fatalf("%s: %v", reviewKind, err)
		}
	}
	_, err := newTestService(t, fixtureDatabase(), &scriptedClient{}, nil).Review(context.Background(), port.ReviewRequest{Kind: "nope"})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown review kind should be a validation error, got %v", err)
	}
}

func TestInvalidOutputFallsBackToSecondModel(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: "Sure! Here is your review."}, {content: validInsight}}}
	if _, err := newTestService(t, fixtureDatabase(), client, nil).HealthCommentary(context.Background(), false, ""); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(client.models(), ","); got != "primary,fallback" {
		t.Fatalf("expected primary then fallback, got %s", got)
	}

	client = &scriptedClient{replies: []reply{{err: domain.ErrAIUnavailable}, {content: "still not json"}}}
	_, err := newTestService(t, fixtureDatabase(), client, nil).HealthCommentary(context.Background(), false, "override")
	if !errors.Is(err, domain.ErrAIOutput) || strings.Join(client.models(), ",") != "override,fallback" {
		t.Fatalf("expected an output error after override and fallback, got %v %v", err, client.models())
	}
}

func TestMissingPromptAndMissingConfig(t *testing.T) {
	db := fixtureDatabase()
	delete(db.AIContent.Prompts, "tasks/loan-review")
	_, err := newTestService(t, db, &scriptedClient{}, nil).Review(context.Background(), port.ReviewRequest{Kind: port.ReviewLoans})
	if !errors.Is(err, domain.ErrPromptMissing) || !strings.Contains(err.Error(), "tasks/loan-review") {
		t.Fatalf("expected a missing prompt error, got %v", err)
	}

	unconfigured := New(&scriptedClient{}, fixtureDatabase(), nil, nil, domain.AIConfig{}, logger.NewTestLogger(t), Options{})
	if _, err := unconfigured.SuggestCategory(context.Background(), "Some new shop", ""); !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("expected AI unavailable, got %v", err)
	}
}

func TestBudgetAdviceKeepsEngineFacts(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{
		"briefing": "Food is on track.",
		"suggestedLimits": [{"category": "food", "suggestedLimit": 2500000, "rationale": "Below average"}, {"category": "Yachts", "suggestedLimit": 1}],
		"alerts": [{"type": "weird", "severity": "loud", "title": "Heads up", "message": "Check coffee"}],
		"goalImpact": [{"goal": "Trip", "impact": "On track"}],
		"projection": [{"date": "2026-10-11", "balance": 1}],
		"forecastedSavings": 999
	}`}}}
	advice, err := newTestService(t, fixtureDatabase(), client, nil).BudgetAdvice(context.Background(), false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(advice.SuggestedLimits) != 1 || advice.SuggestedLimits[0].Category != "Food" || advice.SuggestedLimits[0].CurrentLimit != 3_000_000 {
		t.Fatalf("unexpected limits %+v", advice.SuggestedLimits)
	}
	if advice.Alerts[0].Type != "opportunity" || advice.Alerts[0].Severity != domain.InsightInfo {
		t.Fatalf("alert not normalized %+v", advice.Alerts[0])
	}
	if advice.Month != "2026-10" || len(advice.Projection) != 30 || advice.ForecastedSavings == 999 {
		t.Fatalf("model must not override engine facts: %+v", advice)
	}
	if !strings.Contains(client.requests[0].Messages[1].Content, "FACTS (computed by the engine") {
		t.Fatal("facts block missing from the action prompt")
	}
}

func TestBriefingAttachesFacts(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"briefing":"All good.","alerts":[],"patterns":[{"kind":"anomaly","title":"Coffee","content":"More coffee."},{"kind":"horoscope","title":"x","content":"y"}]}`}}}
	briefing, err := newTestService(t, fixtureDatabase(), client, nil).Briefing(context.Background(), false, "")
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Facts.NetWorth != 15_000_000 || len(briefing.Patterns) != 1 {
		t.Fatalf("unexpected briefing %+v", briefing)
	}
}

func TestChartInsightReturnsMarkdown(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: "  **Spending** is flat.  "}}}
	text, err := newTestService(t, fixtureDatabase(), client, nil).ChartInsight(context.Background(), port.ChartInsightRequest{ChartType: "line", Title: "Cash flow", Data: []int{1, 2, 3}})
	if err != nil || text != "**Spending** is flat." {
		t.Fatalf("unexpected insight %q %v", text, err)
	}
	if _, err := newTestService(t, fixtureDatabase(), client, nil).ChartInsight(context.Background(), port.ChartInsightRequest{ChartType: "line"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("missing data should be a validation error, got %v", err)
	}
}
