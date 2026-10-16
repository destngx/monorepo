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
	"testing"
	"time"
)

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
	for _, fragment := range []string{"VND 40000000", "crypto 0.5 BTC", "p2p 26500", "US N/A: US market coverage", "VN Steady climb", "VN-Index: 1300 (1.25%)", "VNINDEX   : n/a | 0.50", `[1] "Rates cut"`, "Binance [name]", `"symbol":"FUND-A"`, `"annualisedReturnPct":10`, `"vsDeposit":"ahead"`, `"volatileSharePct":100`, `"cryptoShareOfNetWorthPct":75`, `"fallPct":30`} {
		if !strings.Contains(debate, fragment) {
			t.Errorf("think-tank prompt is missing %q:\n%s", fragment, debate)
		}
	}
	if strings.Contains(debate, "Nguyen Van A") || strings.Contains(debate, "Platform") {
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

const recommendingCommandsReply = `{"executable_commands":[{"action":"OBSERVE","target":"BTC","rationale":"Wait."}],
	"recommendations":[{"title":"Cap crypto at 20%","action":"Move 5.000.000 to a deposit","impactVnd":300000,"evidence":"Crypto is 75% of net worth"},{"title":"","action":"dropped"}],
	"outcomes":[]}`

// recommendingThinkTankRoute answers every phase; the actions phase
// returns commands with recommendations.
func recommendingThinkTankRoute(actions string) func(domain.AIRequest) reply {
	return func(request domain.AIRequest) reply {
		system := request.Messages[0].Content
		switch {
		case strings.Contains(system, "Debate."):
			return reply{content: "Experts disagree."}
		case strings.Contains(system, "Synthesise"):
			return reply{content: "Hold cash."}
		case strings.Contains(system, "Act on"):
			return reply{content: actions}
		}
		return reply{err: errors.New("unexpected prompt")}
	}
}

func newThinkTankService(t *testing.T, db *fakes.Database, client *scriptedClient, search port.SearchService, recommendations port.RecommendationService) *Service {
	t.Helper()
	now := func() time.Time { return testToday }
	analytics := service.NewAnalyticsService(db, now)
	if recommendations == nil {
		recommendations = service.NewRecommendationService(db, analytics, nil, logger.NewTestLogger(t), now)
	}
	options := Options{Now: now, Search: search, Market: &fakeAnalysis{}, Portfolio: fakePortfolio{}, Recommendations: recommendations}
	return New(client, db, analytics, nil, testConfig(), logger.NewTestLogger(t), options)
}

func runThinkTank(t *testing.T, svc *Service, request port.InvestmentAnalysisRequest) ([]port.InvestmentEvent, error) {
	t.Helper()
	var events []port.InvestmentEvent
	err := svc.InvestmentAnalysis(context.Background(), request, func(event port.InvestmentEvent) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

func eventOf(events []port.InvestmentEvent, kind string) (port.InvestmentEvent, bool) {
	for _, event := range events {
		if event.Type == kind {
			return event, true
		}
	}
	return port.InvestmentEvent{}, false
}

func userMessages(client *scriptedClient) []string {
	client.mu.Lock()
	defer client.mu.Unlock()
	out := []string{}
	for _, request := range client.requests {
		out = append(out, request.Messages[1].Content)
	}
	return out
}

func TestGivenNoTopicWhenTheThinkTankRunsThenItFocusesOnTheHoldingsBySymbolOnly(t *testing.T) {
	client := &scriptedClient{route: recommendingThinkTankRoute(commandsReply)}
	events, err := runThinkTank(t, newThinkTankService(t, marketDatabase(), client, &fakeSearch{results: newsResults}, nil), port.InvestmentAnalysisRequest{})
	if err != nil {
		t.Fatal(err)
	}
	disclosed, ok := eventOf(events, "context")
	if !ok {
		t.Fatal("no context event")
	}
	inputs := disclosed.Context.(thinkTankInputs)
	if inputs.Focus.Source != domain.ThinkTankFocusHoldings || !slices.Equal(inputs.Focus.Symbols, []string{"BTC"}) {
		t.Fatalf("focus = %+v", inputs.Focus)
	}
	messages := userMessages(client)
	if len(messages) != 3 {
		t.Fatalf("expected three phases, got %d", len(messages))
	}
	for _, message := range messages {
		if !strings.Contains(message, "current holdings: BTC.") {
			t.Fatalf("every phase must get the focus: %s", message)
		}
		if strings.Contains(message, "Nguyen Van A") {
			t.Fatalf("the focus names symbols only: %s", message)
		}
	}
}

func TestGivenRequestedSymbolsWhenTheThinkTankRunsThenTheyAreTheFocusAndTheSearch(t *testing.T) {
	client := &scriptedClient{route: recommendingThinkTankRoute(commandsReply)}
	search := &fakeSearch{results: newsResults}
	events, err := runThinkTank(t, newThinkTankService(t, marketDatabase(), client, search, nil), port.InvestmentAnalysisRequest{Symbols: []string{" fpt", "VCB"}})
	if err != nil {
		t.Fatal(err)
	}
	disclosed, _ := eventOf(events, "context")
	if focus := disclosed.Context.(thinkTankInputs).Focus; focus.Source != domain.ThinkTankFocusRequest || !slices.Equal(focus.Symbols, []string{"FPT", "VCB"}) {
		t.Fatalf("focus = %+v", focus)
	}
	if !slices.Contains(search.queries, "market outlook FPT VCB October 2026") {
		t.Fatalf("the search must follow the focus: %v", search.queries)
	}
	if !strings.Contains(userMessages(client)[0], "the user asked about: FPT, VCB") {
		t.Fatal("the debate must get the requested focus")
	}
}

func TestGivenABadSymbolWhenTheThinkTankStartsThenItIsRejectedBeforeAnyModelCall(t *testing.T) {
	client := &scriptedClient{route: recommendingThinkTankRoute(commandsReply)}
	events, err := runThinkTank(t, newThinkTankService(t, marketDatabase(), client, nil, nil), port.InvestmentAnalysisRequest{Symbols: []string{"BTC; DROP"}})
	if !errors.Is(err, domain.ErrValidation) || len(events) != 0 || len(client.requests) != 0 {
		t.Fatalf("err = %v, events = %v, requests = %d", err, events, len(client.requests))
	}
}

func TestGivenTheThinkTankRecommendsWhenItRunsThenTheListGetsItsItemsAndTheNextRunChecksThem(t *testing.T) {
	db := marketDatabase()
	client := &scriptedClient{route: recommendingThinkTankRoute(recommendingCommandsReply)}
	svc := newThinkTankService(t, db, client, nil, nil)

	events, err := runThinkTank(t, svc, port.InvestmentAnalysisRequest{})
	if err != nil {
		t.Fatal(err)
	}
	actions, _ := eventOf(events, "actions")
	if len(actions.Commands) != 1 || actions.Recorded == nil || actions.Recorded.Error != "" || len(actions.Recorded.Created) != 1 {
		t.Fatalf("actions = %+v recorded = %+v", actions, actions.Recorded)
	}
	created := actions.Recorded.Created[0]
	if created.Source != domain.RecommendationSourceThinkTank || created.Page != domain.ThinkTankPage || *created.ImpactVND != 300_000 {
		t.Fatalf("created = %+v", created)
	}
	act := userMessages(client)[2]
	if !strings.Contains(act, `"recommendations": [{"title"`) || !strings.Contains(act, "PREVIOUS RECOMMENDATIONS: none.") || !strings.Contains(act, "executable_commands") {
		t.Fatalf("the actions phase must ask for recommendations: %s", act)
	}
	for _, message := range userMessages(client)[:2] {
		if strings.Contains(message, "PREVIOUS RECOMMENDATIONS") {
			t.Fatal("only the actions phase gets the recommendation contract")
		}
	}

	client.route = recommendingThinkTankRoute(strings.Replace(recommendingCommandsReply, `"outcomes":[]`, `"outcomes":[{"id":"`+created.ID+`","verdict":"met","note":"Crypto is under 20%"}]`, 1))
	again, err := runThinkTank(t, svc, port.InvestmentAnalysisRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if act := userMessages(client)[5]; !strings.Contains(act, `"id":"`+created.ID+`"`) {
		t.Fatalf("the next run must see the earlier item: %s", act)
	}
	record, _ := eventOf(again, "actions")
	if record.Recorded == nil || len(record.Recorded.Created) != 0 || record.Recorded.Duplicates != 1 || len(record.Recorded.Checked) != 1 || record.Recorded.Checked[0].Status != domain.RecommendationDone {
		t.Fatalf("second record = %+v", record.Recorded)
	}
}

// unreadableRecommendations cannot read the earlier items.
type unreadableRecommendations struct{ port.RecommendationService }

func (unreadableRecommendations) RecentForReview(context.Context, string) ([]domain.Recommendation, error) {
	return nil, errors.New("sheet read failed")
}

func TestGivenTheListFailsWhenTheThinkTankRunsThenTheRunFinishesAndSaysSo(t *testing.T) {
	for name, list := range map[string]port.RecommendationService{
		"write fails": failingRecommendations{},
		"read fails":  unreadableRecommendations{},
	} {
		t.Run(name, func(t *testing.T) {
			client := &scriptedClient{route: recommendingThinkTankRoute(recommendingCommandsReply)}
			events, err := runThinkTank(t, newThinkTankService(t, marketDatabase(), client, nil, list), port.InvestmentAnalysisRequest{})
			if err != nil {
				t.Fatal(err)
			}
			actions, _ := eventOf(events, "actions")
			if actions.Recorded == nil || !strings.Contains(actions.Recorded.Error, "sheet") || len(actions.Commands) != 1 {
				t.Fatalf("actions = %+v recorded = %+v", actions, actions.Recorded)
			}
			if last := events[len(events)-1]; last.Type != "done" {
				t.Fatalf("the run must finish, last event %+v", last)
			}
		})
	}
}

func TestGivenNoRecommendationsServiceWhenTheThinkTankRunsThenTheActionsContractIsUnchanged(t *testing.T) {
	client := &scriptedClient{route: recommendingThinkTankRoute(commandsReply)}
	events, err := runThinkTank(t, newMarketService(t, marketDatabase(), client, nil, nil), port.InvestmentAnalysisRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if actions, _ := eventOf(events, "actions"); actions.Recorded != nil {
		t.Fatalf("recorded = %+v", actions.Recorded)
	}
	if strings.Contains(userMessages(client)[2], "PREVIOUS RECOMMENDATIONS") {
		t.Fatal("without the list the think tank is not asked for recommendations")
	}
}
