package ai

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeSearch struct {
	mu      sync.Mutex
	err     error
	results []domain.SearchResult
	queries []string
	bypass  []bool
}

func (f *fakeSearch) Search(ctx context.Context, query string) (domain.SearchResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries = append(f.queries, query)
	f.bypass = append(f.bypass, domain.ShouldBypassCache(ctx))
	return domain.SearchResponse{Query: query, Provider: "tavily", Results: f.results}, f.err
}

func (f *fakeSearch) Providers() []string { return []string{"tavily"} }

type fakeAnalysis struct {
	pulseErr error
}

func (f *fakeAnalysis) Pulse(context.Context, domain.Timeframe) (domain.MarketPulse, error) {
	half := 0.5
	return domain.MarketPulse{
		Assets:      []domain.PulseQuote{{PulseAsset: domain.PulseAsset{Symbol: "VNINDEX", Name: "VN-Index"}, Price: 1300, ChangePct: 1.25}},
		Correlation: domain.CorrelationMatrix{Symbols: []string{"VNINDEX", "SJC"}, Values: [][]*float64{{nil, &half}, {&half, nil}}},
		CapitalFlow: domain.CapitalFlow{Signal: "RISK_ON", Summary: "Money flows into equities."},
		Scenario:    domain.Scenario{Name: "Steady climb", Regime: "BULL", Summary: "Index above its averages."},
	}, f.pulseErr
}

func (f *fakeAnalysis) Analysis(_ context.Context, symbol string, timeframe domain.Timeframe) (domain.MarketAnalysis, error) {
	return domain.MarketAnalysis{
		Symbol: strings.ToUpper(symbol), Timeframe: timeframe, Provider: "vnstock", Currency: "VND",
		Technicals: domain.Technicals{Price: 120_000, Trend: "UP", Phase: "MARKUP", Signal: domain.Signal{Action: "BUY"}},
		Seasonality: []domain.SeasonalityBucket{
			{Type: domain.SeasonalityDay, Label: "Mon", Rank: 2},
			{Type: domain.SeasonalityDay, Label: "Tue", Rank: 1},
			{Type: domain.SeasonalityDay, Label: "Wed", Rank: 3},
		},
	}, nil
}

type fakePortfolio struct{}

func (fakePortfolio) Valuation(context.Context, bool) (domain.PortfolioValuation, error) {
	quantity, value := 0.5, 1_500_000_000.0
	return domain.PortfolioValuation{
		Holdings: []domain.ValuedHolding{{Section: domain.PortfolioSectionCrypto, Holding: domain.Holding{Platform: "Binance Nguyen Van A", Symbol: "BTC", Quantity: &quantity}, ValueVND: &value}},
		Rates:    []domain.ExchangeRate{{From: "USDT", To: "VND", Rate: 26_500, Provider: "coingecko"}},
		Totals:   domain.PortfolioTotals{VND: 40_000_000},
	}, nil
}

func marketDatabase() *fakes.Database {
	db := fixtureDatabase()
	for _, key := range marketPrompts {
		db.AIContent.Prompts[key] = "[" + key + "]"
	}
	db.AIContent.Prompts[keyNewsTask] = "Classify news about {{topic}} in {{year}}."
	db.AIContent.Prompts[keyTickerTask] = "Ticker {{symbol}} {{timeframe}} at {{price}}, {{trend}}/{{phase}}/{{action}}."
	db.AIContent.Prompts[phaseThinkTank.taskKey] = "Debate. VND {{totalVND}} crypto {{totalCrypto}} p2p {{p2pRate}} US {{usRegimeName}} VN {{vnRegimeName}} assets {{vnAssets}}\n{{vnCorrelationMatrix}}\nNews:\n{{searchContext}}\nHoldings {{cryptoHoldings}}"
	db.AIContent.Prompts[phaseSynthesis.taskKey] = "Synthesise: {{expertDebateContext}}"
	db.AIContent.Prompts[phaseActions.taskKey] = "Act on: {{synthesisContext}}"
	db.AIContent.Prompts[phaseThinkTank.fallbackTask] = "Short debate for {{totalVND}}."
	return db
}

func newMarketService(t *testing.T, db *fakes.Database, client *scriptedClient, search port.SearchService, cache port.CacheService) *Service {
	t.Helper()
	now := func() time.Time { return testToday }
	options := Options{Now: now, Search: search, Market: &fakeAnalysis{}, Portfolio: fakePortfolio{}}
	return New(client, db, service.NewAnalyticsService(db, domain.CardConfig{}, now), cache, testConfig(), logger.NewTestLogger(t), options)
}

var newsResults = []domain.SearchResult{
	{Title: "Rates cut", URL: "https://a.vn/1", Snippet: "SBV cuts rates", Source: "a.vn"},
	{Title: "Exports up", URL: "https://b.vn/2", Snippet: "Exports grow", Source: "b.vn"},
	{Title: "Fed holds", URL: "https://c.com/3", Snippet: "No change", Source: "c.com"},
}

const newsReply = `{"articles":[
	{"index":0,"sentiment":"positive","confidence":90,"impactScore":80,"summary":"Cheaper credit."},
	{"index":1,"sentiment":"Positive","confidence":70,"impactScore":60,"summary":"Growth."},
	{"index":2,"sentiment":"Neutral","confidence":50,"impactScore":20,"summary":"Wait and see."}]}`

func TestNewsSentimentClassifiesComputesBiasAndCaches(t *testing.T) {
	search := &fakeSearch{results: newsResults}
	client := &scriptedClient{replies: []reply{{content: `{"articles":[{"index":0,"sentiment":"Positive","confidence":90,"impactScore":80,"summary":"x"}]}`}, {content: newsReply}}}
	svc := newMarketService(t, marketDatabase(), client, search, &memoryCache{})

	report, err := svc.NewsSentiment(context.Background(), port.NewsRequest{Topic: "  Vietnam   market "})
	if err != nil {
		t.Fatal(err)
	}
	if report.Topic != "Vietnam market" || report.Bias != domain.NewsBullish || report.Counts.Positive != 2 || len(report.Articles) != 3 {
		t.Fatalf("unexpected report %+v", report)
	}
	if report.Articles[0].Sentiment != domain.NewsPositive || report.Articles[0].URL != "https://a.vn/1" || report.Provider != "tavily" {
		t.Fatalf("unexpected first article %+v", report.Articles[0])
	}
	if strings.Join(client.models(), ",") != "primary,fallback" {
		t.Fatalf("a verdict per article is required; expected a retry, got %v", client.models())
	}
	if search.queries[0] != "Vietnam market latest market news financial impact 2026" {
		t.Fatalf("unexpected query %q", search.queries[0])
	}
	if system := client.requests[0].Messages[0].Content; system != "Classify news about Vietnam market in 2026." {
		t.Fatalf("unexpected system prompt %q", system)
	}
	if user := client.requests[0].Messages[1].Content; !strings.Contains(user, "[2] Title: Fed holds") || !strings.Contains(user, `"impactScore"`) {
		t.Fatalf("articles or format missing: %s", user)
	}

	if _, err := svc.NewsSentiment(context.Background(), port.NewsRequest{Topic: "Vietnam market"}); err != nil || len(client.requests) != 2 {
		t.Fatalf("second call should be cached: %v %d", err, len(client.requests))
	}
	client.replies = []reply{{content: newsReply}}
	if _, err := svc.NewsSentiment(context.Background(), port.NewsRequest{Topic: "Vietnam market", Force: true}); err != nil || len(client.requests) != 3 || !search.bypass[2] {
		t.Fatalf("force should bypass both caches: %v %d %v", err, len(client.requests), search.bypass)
	}
}

func TestNewsSentimentWithoutArticlesSkipsTheModel(t *testing.T) {
	client := &scriptedClient{}
	report, err := newMarketService(t, marketDatabase(), client, &fakeSearch{}, nil).NewsSentiment(context.Background(), port.NewsRequest{Topic: "rates"})
	if err != nil || report.Bias != domain.NewsMixed || len(report.Articles) != 0 || len(client.requests) != 0 {
		t.Fatalf("expected an empty neutral report, got %+v %v", report, err)
	}
}

func TestNewsSentimentNeedsTopicAndSearch(t *testing.T) {
	if _, err := newMarketService(t, marketDatabase(), &scriptedClient{}, &fakeSearch{}, nil).NewsSentiment(context.Background(), port.NewsRequest{Topic: " "}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected a validation error, got %v", err)
	}
	if _, err := newMarketService(t, marketDatabase(), &scriptedClient{}, nil, nil).NewsSentiment(context.Background(), port.NewsRequest{Topic: "rates"}); !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("expected search to be required, got %v", err)
	}
	failing := &fakeSearch{err: errors.New("every web search provider failed")}
	if _, err := newMarketService(t, marketDatabase(), &scriptedClient{}, failing, nil).NewsSentiment(context.Background(), port.NewsRequest{Topic: "rates"}); err == nil || !strings.Contains(err.Error(), "search news") {
		t.Fatalf("expected the search failure, got %v", err)
	}
}

func TestTickerInsightUsesEngineFactsAndNotesSearchFailure(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: validInsight}}}
	search := &fakeSearch{err: errors.New("quota exceeded")}
	insight, err := newMarketService(t, marketDatabase(), client, search, nil).TickerInsight(context.Background(), port.TickerInsightRequest{Symbol: "fpt"})
	if err != nil || insight.Summary != "Healthy month." {
		t.Fatalf("unexpected insight %+v %v", insight, err)
	}
	system := client.requests[0].Messages[0].Content
	if !strings.Contains(system, "Ticker FPT 1d at 120000, UP/MARKUP/BUY.") {
		t.Fatalf("task not rendered: %s", system)
	}
	prompt := system + client.requests[0].Messages[1].Content
	if !strings.Contains(prompt, "web search failed: quota exceeded") || !strings.Contains(prompt, `"Tue"`) || !strings.Contains(prompt, `"Wed"`) || strings.Contains(prompt, `"Mon"`) {
		t.Fatalf("facts should carry the best and worst buckets and the search failure: %s", prompt)
	}
	if search.queries[0] != "FPT stock latest news October 2026" {
		t.Fatalf("unexpected query %q", search.queries[0])
	}
	if _, err := newMarketService(t, marketDatabase(), client, nil, nil).TickerInsight(context.Background(), port.TickerInsightRequest{Symbol: "FPT", Timeframe: "2h"}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected an invalid timeframe, got %v", err)
	}
}

const commandsReply = `{"executable_commands":[{"action":"allocate","exact_target":"VN30 ETF","execution_logic":"Trend is up."},{"action":"OBSERVE","target":"BTC","rationale":"Wait."}]}`

// thinkTankRoute fails the main think-tank prompt on both models so the
// fallback pair runs, and answers every other phase.
func thinkTankRoute(request domain.AIRequest) reply {
	system := request.Messages[0].Content
	switch {
	case strings.Contains(system, "Debate."):
		return reply{err: fmt.Errorf("%w: gateway down", domain.ErrAIUnavailable)}
	case strings.Contains(system, "Short debate"):
		return reply{content: "Experts disagree."}
	case strings.Contains(system, "Synthesise"):
		return reply{content: "Hold cash, add VN30."}
	case strings.Contains(system, "Act on"):
		return reply{content: commandsReply}
	}
	return reply{err: errors.New("unexpected prompt")}
}

func TestInvestmentAnalysisStreamsPhasesWithFallbackAndCommands(t *testing.T) {
	client := &scriptedClient{route: thinkTankRoute}
	search := &fakeSearch{results: newsResults}
	var events []port.InvestmentEvent
	err := newMarketService(t, marketDatabase(), client, search, nil).InvestmentAnalysis(context.Background(), port.InvestmentAnalysisRequest{}, func(event port.InvestmentEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var trail []string
	for _, event := range events {
		trail = append(trail, strings.Trim(event.Type+":"+event.Phase+":"+event.Status, ":"))
	}
	want := []string{
		"phase:context:running", "context", "phase:context:ok",
		"phase:think-tank:running", "phase:think-tank:fallback", "phase:think-tank:ok", "message:think-tank",
		"phase:synthesis:running", "phase:synthesis:ok", "message:synthesis",
		"phase:actions:running", "phase:actions:ok", "actions:actions", "done",
	}
	if !slices.Equal(trail, want) {
		t.Fatalf("unexpected event trail:\n%v\nwant\n%v", trail, want)
	}
	commands := events[12].Commands
	if len(commands) != 2 || commands[0].Action != domain.ActionAllocate || commands[0].Target != "VN30 ETF" || commands[0].Rationale != "Trend is up." {
		t.Fatalf("unexpected commands %+v", commands)
	}
	if events[6].Content != "Experts disagree." || events[9].Content != "Hold cash, add VN30." {
		t.Fatalf("unexpected messages %+v %+v", events[6], events[9])
	}
	if len(search.queries) != 3 || !slices.Contains(search.queries, "crypto forecast BTC October 2026") {
		t.Fatalf("expected the three legacy searches, got %v", search.queries)
	}

	var debate string
	for _, request := range client.requests {
		if strings.Contains(request.Messages[0].Content, "Debate.") {
			debate = request.Messages[0].Content
		}
	}
	for _, fragment := range []string{"VND 40000000", "crypto 0.5 BTC", "p2p 26500", "US N/A: US market coverage", "VN Steady climb", "VN-Index: 1300 (1.25%)", "VNINDEX   : n/a | 0.50", `[1] "Rates cut"`, "Binance [name]"} {
		if !strings.Contains(debate, fragment) {
			t.Errorf("think-tank prompt is missing %q:\n%s", fragment, debate)
		}
	}
	if strings.Contains(debate, "Nguyen Van A") {
		t.Fatalf("names must be masked: %s", debate)
	}
}

func TestInvestmentAnalysisFailsAPhaseWithoutFallbackAndToleratesMissingContext(t *testing.T) {
	db := marketDatabase()
	delete(db.AIContent.Prompts, phaseThinkTank.fallbackTask)
	client := &scriptedClient{route: thinkTankRoute}
	svc := newMarketService(t, db, client, nil, nil)
	svc.market = &fakeAnalysis{pulseErr: errors.New("vnstock down")}
	var events []port.InvestmentEvent
	err := svc.InvestmentAnalysis(context.Background(), port.InvestmentAnalysisRequest{}, func(event port.InvestmentEvent) error {
		events = append(events, event)
		return nil
	})
	if !errors.Is(err, domain.ErrAIUnavailable) || !strings.Contains(err.Error(), "think-tank phase") {
		t.Fatalf("expected the think-tank failure, got %v", err)
	}
	last := events[len(events)-1]
	if last.Status != "error" || last.Phase != "think-tank" {
		t.Fatalf("expected an error phase event, got %+v", last)
	}
	debate := client.requests[0].Messages[0].Content
	if !strings.Contains(debate, "NO WEB RESULTS") || !strings.Contains(debate, "market pulse failed (vnstock down)") {
		t.Fatalf("missing context must be stated: %s", debate)
	}

	svc.portfolio = nil
	if err := svc.InvestmentAnalysis(context.Background(), port.InvestmentAnalysisRequest{}, func(port.InvestmentEvent) error { return nil }); !errors.Is(err, domain.ErrAIUnavailable) {
		t.Fatalf("expected the portfolio to be required, got %v", err)
	}
}

func TestStatusReportsMarketPromptsOnlyWhenMarketIsWired(t *testing.T) {
	db := fixtureDatabase()
	status := newMarketService(t, db, &scriptedClient{}, nil, nil).Status(context.Background())
	if !slices.Contains(status.MissingKeys, keyNewsTask) || !slices.Contains(status.MissingKeys, phaseActions.actionKey) {
		t.Fatalf("expected the market prompts to be reported, got %v", status.MissingKeys)
	}
	if status := newTestService(t, db, &scriptedClient{}, nil).Status(context.Background()); slices.Contains(status.MissingKeys, keyNewsTask) {
		t.Fatalf("market prompts are not required without market data: %v", status.MissingKeys)
	}
}

func TestChatOffersMarketToolsWhenWired(t *testing.T) {
	client := &scriptedClient{replies: []reply{
		{toolCalls: []domain.AIToolCall{{ID: "c1", Name: "getMarketAnalysis", Arguments: `{"symbol":"fpt"}`}, {ID: "c2", Name: "webSearch", Arguments: `{"query":"FPT news"}`}}},
		{content: "FPT is trending up."},
	}}
	search := &fakeSearch{results: newsResults}
	err := newMarketService(t, marketDatabase(), client, search, nil).Chat(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{Role: "user", Content: "How is FPT?"}},
	}, func(port.ChatEvent) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(client.requests[0].Tools) != 8 || len(search.queries) != 1 || search.queries[0] != "FPT news" {
		t.Fatalf("expected both market tools to run, got %d tools and %v", len(client.requests[0].Tools), search.queries)
	}
	results := client.requests[1].Messages
	if !strings.Contains(results[len(results)-2].Content, `"phase":"MARKUP"`) || !strings.Contains(results[len(results)-1].Content, "Rates cut") {
		t.Fatalf("tool results not passed back: %+v", results[len(results)-2:])
	}
}
