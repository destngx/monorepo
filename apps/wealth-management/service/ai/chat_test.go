package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestChatRunsToolsAndStreams(t *testing.T) {
	client := &scriptedClient{replies: []reply{
		{toolCalls: []domain.AIToolCall{{ID: "c1", Name: "getRecentTransactions", Arguments: `{"payee":"grab"}`}}},
		{content: "You spent 120.000 ₫ on Grab."},
	}}
	var events []port.ChatEvent
	err := newTestService(t, fixtureDatabase(), client, nil).Chat(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{Role: "user", Content: "How much on Grab?"}},
		Page:     &port.ChatPageContext{Path: "/transactions", Title: "Transactions"},
	}, func(event port.ChatEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var text strings.Builder
	kinds := []string{}
	for _, event := range events {
		if event.Type == "text" {
			text.WriteString(event.Delta)
			continue
		}
		kinds = append(kinds, event.Type+":"+event.Status)
	}
	if strings.Join(kinds, ",") != "tool:running,tool:ok,done:" || text.String() != "You spent 120.000 ₫ on Grab." {
		t.Fatalf("unexpected events %v %q", kinds, text.String())
	}
	if len(client.requests[0].Tools) != 9 || !strings.Contains(client.requests[0].Messages[0].Content, "/transactions") {
		t.Fatalf("first turn missing tools or page context")
	}
	second := client.requests[1].Messages
	toolResult := second[len(second)-1]
	if toolResult.Role != domain.AIRoleTool || toolResult.ToolCallID != "c1" || !strings.Contains(toolResult.Content, `"total":1`) {
		t.Fatalf("tool result not passed back: %+v", toolResult)
	}

	err = newTestService(t, fixtureDatabase(), client, nil).Chat(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: "assistant", Content: "hi"}}}, func(port.ChatEvent) error { return nil })
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("chat must end with a user message, got %v", err)
	}
}

func TestChatFallsBackOnlyBeforeStreaming(t *testing.T) {
	client := &scriptedClient{replies: []reply{{err: domain.ErrAIUnavailable}, {content: "Hello there"}}}
	err := newTestService(t, fixtureDatabase(), client, nil).Chat(context.Background(), port.ChatRequest{Messages: []port.ChatMessage{{Role: "user", Content: "hi"}}}, func(port.ChatEvent) error { return nil })
	if err != nil || strings.Join(client.models(), ",") != "primary,fallback" {
		t.Fatalf("expected fallback before streaming: %v %v", err, client.models())
	}
}

func TestGivenAGoalPlanWhenTheChatReadsGoalsThenItGetsThePlanWithMaskedNames(t *testing.T) {
	db := fixtureDatabase()
	db.GoalPlan.Goals[0].Name = "Trip with Person B"
	db.GoalPlan.Goals[0].Target = money(300_000_000)
	tools := newTestService(t, db, &scriptedClient{}, nil).chatTools()

	result, err := tools["getGoals"].run(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := result.(goalPlanContext)
	if !ok || len(plan.Goals) != 1 || plan.Goals[0].Pace != domain.PaceBehind || plan.Goals[0].Target != 300_000_000 {
		t.Fatalf("plan = %+v", result)
	}
	if strings.Contains(plan.Goals[0].Name, "Person B") || plan.Verdict != domain.PlanShort {
		t.Fatalf("names must be masked and the check included: %+v", plan)
	}
}

func TestGivenAChatAskingForAGoalWhenFillingThenTheStreamSendsTheFormAndNothingIsSaved(t *testing.T) {
	client := &scriptedClient{replies: []reply{
		{toolCalls: []domain.AIToolCall{{ID: "c1", Name: "fillForm", Arguments: `{"form":"goal","text":"save 50M for Japan by June 2027"}`}}},
		{content: `{"values": {"name": "Japan", "target": 50000000, "targetDate": "2027-06"}}`},
		{content: "Open the form to check it."},
	}}
	db := fixtureDatabase()
	var forms []*domain.FillChatForm
	err := newTestService(t, db, client, nil).Chat(context.Background(), port.ChatRequest{
		Messages: []port.ChatMessage{{Role: "user", Content: "Add a goal: 50M for Japan by June 2027"}},
	}, func(event port.ChatEvent) error {
		if event.Type == "form" {
			forms = append(forms, event.Form)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(forms) != 1 || forms[0].Form != "goal" || forms[0].Page != "/accounts/goals" || forms[0].Result.Values["name"] != "Japan" {
		t.Fatalf("form events = %+v", forms)
	}
	if len(db.CreatedGoals) != 0 {
		t.Fatalf("the chat must not save: %+v", db.CreatedGoals)
	}
	toolResult := client.requests[2].Messages[len(client.requests[2].Messages)-1]
	if !strings.Contains(toolResult.Content, `"filled"`) || !strings.Contains(toolResult.Content, "Open the form") {
		t.Fatalf("the model is told what was filled: %s", toolResult.Content)
	}
}

func TestGivenAFormThatEditsAChosenItemWhenTheChatAsksThenTheToolRefusesIt(t *testing.T) {
	tools := newTestService(t, fixtureDatabase(), &scriptedClient{}, nil).chatTools()
	if _, err := tools["fillForm"].run(context.Background(), []byte(`{"form":"category-definition","text":"groceries"}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v", err)
	}
}
