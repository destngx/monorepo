package ai

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
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
	prompts["tasks/intelligence-briefing"] = "Money lent: {{moneyLentJson}}"
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
			{AccountName: "Bank", Date: testToday.AddDate(0, 0, -30), Payee: "Person B", Category: "Lending / Repayment", CategoryType: kind(domain.CategoryTypeNonBudget), CategoryNature: domain.NatureLending, Payment: money(20_000_000)},
		},
		Categories: []domain.Category{
			{Name: "Coffee", Type: domain.CategoryTypeExpense},
			{Name: "Food", Type: domain.CategoryTypeExpense},
			{Name: "Transport", Type: domain.CategoryTypeExpense},
			{Name: "Uncategorized", Type: domain.CategoryTypeExpense},
		},
		Budget: []domain.BudgetItem{{Category: "Food", CategoryType: kind(domain.CategoryTypeExpense), MonthlyLimit: 3_000_000}},
		GoalPlan: domain.GoalPlanSheet{Exists: true, Goals: []domain.PlannedGoal{
			{ID: "goal-2", Row: 2, Name: "Trip", Target: money(30_000_000), TargetDate: "2027-06", Priority: 1, Status: domain.GoalActive},
		}},
		AIContent: domain.AIContent{Prompts: allPrompts()},
	}
}

func testConfig() domain.AIConfig {
	return domain.AIConfig{
		BaseURL:       "http://user:secret@gateway.local:8080/v1",
		Models:        map[domain.AIUseCase]string{domain.AIUseCaseInsight: "primary", domain.AIUseCaseSuggest: "primary", domain.AIUseCaseChat: "primary", domain.AIUseCaseVision: "primary"},
		FallbackModel: "fallback",
		Timeout:       time.Second,
		PIINames:      []string{"Nguyen Van A", "Person B"},
	}
}

func newTestService(t *testing.T, db *fakes.Database, client *scriptedClient, cache port.CacheService) *Service {
	t.Helper()
	now := func() time.Time { return testToday }
	analytics := service.NewAnalyticsService(db, now)
	return New(client, db, analytics, cache, testConfig(), logger.NewTestLogger(t), Options{Now: now})
}

const validInsight = `{"summary":"Healthy month.","sections":[{"icon":"wallet","title":"Cash","content":"Cash is fine.","severity":"success"}],"actionItems":["Keep going"]}`

func TestStatusReportsMissingPromptsAndHidesCredentials(t *testing.T) {
	db := fixtureDatabase()
	delete(db.AIContent.Prompts, "tasks/year-review")
	status := newTestService(t, db, &scriptedClient{}, nil).Status(context.Background())

	if !status.Configured || status.Endpoint != "http://gateway.local:8080" {
		t.Fatalf("unexpected status %+v", status)
	}
	if len(status.MissingKeys) != 1 || status.MissingKeys[0] != "tasks/year-review" {
		t.Fatalf("expected the year review prompt to be missing, got %v", status.MissingKeys)
	}
}

func TestProviderUsageAsksTheClient(t *testing.T) {
	want := domain.AIProviderUsage{Provider: "openai", Windows: []domain.AIUsageWindow{{Name: "5h", UsedPercent: 12}}}
	client := &scriptedClient{usage: want}

	usage, err := newTestService(t, fixtureDatabase(), client, nil).ProviderUsage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if client.usageCalls != 1 || usage.Provider != "openai" || len(usage.Windows) != 1 || usage.Windows[0].UsedPercent != 12 {
		t.Fatalf("calls %d, usage %+v", client.usageCalls, usage)
	}
}

func TestProviderUsageNeedsOnlyTheEndpoint(t *testing.T) {
	config := testConfig()
	config.Models = nil
	client := &scriptedClient{}
	svc := New(client, fixtureDatabase(), nil, nil, config, logger.NewTestLogger(t), Options{})
	if _, err := svc.ProviderUsage(context.Background()); err != nil {
		t.Fatalf("usage without models: %v", err)
	}

	config.BaseURL = ""
	svc = New(client, fixtureDatabase(), nil, nil, config, logger.NewTestLogger(t), Options{})
	if _, err := svc.ProviderUsage(context.Background()); !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("want ErrAIUnavailable without AI_BASE_URL, got %v", err)
	}
	if client.usageCalls != 1 {
		t.Fatalf("client called %d times, want 1", client.usageCalls)
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
	if !strings.Contains(system, "Net worth 55000000.") || !strings.Contains(system, "You are the wealth assistant.") || !strings.Contains(system, "OPERATIONAL DIRECTIVE") {
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
	for _, reviewKind := range []port.ReviewKind{port.ReviewBudget, port.ReviewTransactions, port.ReviewYear} {
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
	delete(db.AIContent.Prompts, "tasks/transaction-review")
	_, err := newTestService(t, db, &scriptedClient{}, nil).Review(context.Background(), port.ReviewRequest{Kind: port.ReviewTransactions})
	if !errors.Is(err, domain.ErrPromptMissing) || !strings.Contains(err.Error(), "tasks/transaction-review") {
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
	// Accounts 35M plus 20M lent out (D30).
	if briefing.Facts.NetWorth != 55_000_000 || len(briefing.Patterns) != 1 {
		t.Fatalf("unexpected briefing %+v", briefing)
	}
	// Day 10: the September row on the 20th is outside the same days.
	pace := briefing.Facts.ExpensePace
	if briefing.Facts.ComparedDays != 10 || briefing.Facts.ExpenseChangePercent != nil {
		t.Fatalf("same-days comparison = %+v", briefing.Facts)
	}
	if pace == nil || pace.Spent != 235_000 || pace.Projected == nil || *pace.Projected != 728_500 || pace.LastMonthExpense != 80_000 {
		t.Fatalf("pace = %+v", pace)
	}
	if briefing.Facts.MoneyLent.OwedToYou != 20_000_000 || briefing.Facts.MoneyLent.PeopleOwing != 1 {
		t.Fatalf("money lent facts %+v", briefing.Facts.MoneyLent)
	}
	if bills := briefing.Facts.Bills; bills == nil || bills.Until == "" || bills.SafeToSpend < 0 {
		t.Fatalf("bills facts = %+v", bills)
	}
	if !strings.Contains(client.requests[0].Messages[1].Content, `"upcoming30Days"`) {
		t.Fatal("bills facts missing from the prompt")
	}
	system := client.requests[0].Messages[0].Content
	if !strings.Contains(system, `"owedToYou":20000000`) || !strings.Contains(system, "owed to you 20.000.000 ₫ by 1 person (oldest open 30 days)") {
		t.Fatalf("money lent not in the prompt: %s", system)
	}
	if strings.Contains(system, "Person B") {
		t.Fatalf("lending payee not masked: %s", system)
	}
}

func TestBriefingWithoutGoalsOrBillsStillRunsAndSaysWhatIsMissing(t *testing.T) {
	db := fixtureDatabase()
	db.GoalPlanErr = errors.New("goal plan tab unreadable")
	db.RecurringErr = errors.New("recurring tab unreadable")
	client := &scriptedClient{replies: []reply{{content: `{"briefing":"All good.","alerts":[],"patterns":[]}`}}}

	briefing, err := newTestService(t, db, client, nil).Briefing(context.Background(), false, "")
	if err != nil {
		t.Fatalf("optional context must not fail the briefing: %v", err)
	}
	if briefing.Facts.Goals != nil || briefing.Facts.Bills != nil {
		t.Fatalf("no goal or bills facts expected: %+v", briefing.Facts)
	}
	if !reflect.DeepEqual(briefing.Facts.Unavailable, []string{"goals", "bills"}) {
		t.Fatalf("unavailable = %v", briefing.Facts.Unavailable)
	}
	if !strings.Contains(client.requests[0].Messages[1].Content, `"unavailable":["goals","bills"]`) {
		t.Fatal("the model is not told which facts are missing")
	}
}

func TestBriefingSendsTheSameDaysComparisonAndPaceAsFacts(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"briefing":"All good.","alerts":[],"patterns":[]}`}}}
	if _, err := newTestService(t, fixtureDatabase(), client, nil).Briefing(context.Background(), false, ""); err != nil {
		t.Fatal(err)
	}
	prompt := client.requests[0].Messages[1].Content
	for _, fact := range []string{`"comparedDays":10`, `"expensePace":`, `"projected":728500`} {
		if !strings.Contains(prompt, fact) {
			t.Fatalf("prompt lacks %s", fact)
		}
	}
}

func TestBriefingSendsTheExposureAsFactsWhenHoldingsAreValued(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"briefing":"All good.","alerts":[],"patterns":[]}`}}}
	now := func() time.Time { return testToday }
	db := fixtureDatabase()
	svc := New(client, db, service.NewAnalyticsService(db, now), nil, testConfig(), logger.NewTestLogger(t), Options{Now: now, Portfolio: fakePortfolio{}})
	briefing, err := svc.Briefing(context.Background(), false, "")
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Facts.Exposure == nil || briefing.Facts.Exposure.VolatileSharePct != 100 {
		t.Fatalf("exposure facts = %+v", briefing.Facts.Exposure)
	}
	prompt := client.requests[0].Messages[1].Content
	for _, fact := range []string{`"volatileSharePct":100`, `"cryptoDrawdowns":`} {
		if !strings.Contains(prompt, fact) {
			t.Fatalf("prompt lacks %s", fact)
		}
	}
}

// failingExposure is a portfolio whose exposure cannot be read.
type failingExposure struct{ fakePortfolio }

func (failingExposure) Exposure(context.Context, bool) (domain.Exposure, error) {
	return domain.Exposure{}, errors.New("sheet unavailable")
}

func TestBriefingRunsAndSaysSoWhenTheExposureIsUnavailable(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"briefing":"All good.","alerts":[],"patterns":[]}`}}}
	now := func() time.Time { return testToday }
	db := fixtureDatabase()
	svc := New(client, db, service.NewAnalyticsService(db, now), nil, testConfig(), logger.NewTestLogger(t), Options{Now: now, Portfolio: failingExposure{}})
	briefing, err := svc.Briefing(context.Background(), false, "")
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Facts.Exposure != nil || !slices.Contains(briefing.Facts.Unavailable, "exposure") {
		t.Fatalf("facts = %+v", briefing.Facts)
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

func TestProviderModelsNeedsOnlyTheEndpoint(t *testing.T) {
	config := testConfig()
	config.Models = nil
	client := &scriptedClient{providerModels: []string{"gpt-5"}}
	models, err := New(client, fixtureDatabase(), nil, nil, config, logger.NewTestLogger(t), Options{}).ProviderModels(context.Background())
	if err != nil || len(models) != 1 || models[0] != "gpt-5" {
		t.Fatalf("models %v, err %v", models, err)
	}

	config.BaseURL = ""
	svc := New(client, fixtureDatabase(), nil, nil, config, logger.NewTestLogger(t), Options{})
	if _, err := svc.ProviderModels(context.Background()); !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("want ErrAIUnavailable without AI_BASE_URL, got %v", err)
	}
	if client.modelCalls != 1 {
		t.Fatalf("client called %d times, want 1", client.modelCalls)
	}
}

func TestChatBillsToolReturnsSafeToSpendWithMaskedNames(t *testing.T) {
	db := fixtureDatabase()
	amount := 1_000_000.0
	db.Recurring = domain.RecurringSheet{Exists: true, Items: []domain.RecurringItem{
		{ID: "recurring-2", Name: "Person B rent", Kind: domain.RecurringBill, Amount: &amount, Frequency: domain.FrequencyMonthly, Day: 28, Match: "Landlord", Status: domain.RecurringActive},
	}}
	svc := newTestService(t, db, &scriptedClient{}, nil)
	result, err := svc.chatTools()["getBills"].run(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	payload := toJSON(result)
	if !strings.Contains(payload, `"billsDue":1000000`) || !strings.Contains(payload, `"rule":"safeToSpend = spendableCash`) {
		t.Fatalf("bills context = %s", payload)
	}
	if strings.Contains(payload, "Person B") {
		t.Fatalf("item names must be masked: %s", payload)
	}
}
