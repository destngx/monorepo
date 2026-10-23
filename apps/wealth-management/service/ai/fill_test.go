package ai

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestFillGoalFromASentence(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"values": {"name": "Japan", "target": 50000000, "targetDate": "2027-06", "priority": 2}, "notes": []}`}}}
	result, err := newTestService(t, fixtureDatabase(), client, nil).Fill(context.Background(), domain.FillRequest{
		Form: "goal", Text: "save 50M for Japan by June 2027, after the trip",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "Japan", "target": float64(50_000_000), "targetDate": "2027-06", "priority": float64(2)}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v", result.Values)
	}
	request := client.requests[0]
	if request.Model != "primary" || !request.JSON {
		t.Errorf("text fills use the suggest model with JSON: %+v", request)
	}
	system, user := request.Messages[0].Content, request.Messages[1].Content
	if !strings.Contains(system, "- priority 1: Trip (active, target 30000000, by 2027-06)") {
		t.Errorf("the goals are not given to the model:\n%s", system)
	}
	if !strings.Contains(user, "save 50M for Japan by June 2027") {
		t.Errorf("the request is not sent: %s", user)
	}
}

func TestFillUsesTheVisionModelForImages(t *testing.T) {
	config := testConfig()
	config.Models[domain.AIUseCaseVision] = "vision"
	client := &scriptedClient{replies: []reply{{content: `{"values": {"name": "Electricity", "kind": "bill", "amount": "1.250.000", "day": 12, "category": "food"}}`}}}
	svc := newTestService(t, fixtureDatabase(), client, nil)
	svc.config = config
	result, err := svc.Fill(context.Background(), domain.FillRequest{
		Form: "recurring", Images: []domain.FillImage{{MediaType: "image/png", Data: []byte("png")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.requests[0].Model != "vision" || len(client.requests[0].Messages[1].Images) != 1 {
		t.Fatalf("an image goes to the vision model: %+v", client.requests[0])
	}
	if result.Values["amount"] != float64(1_250_000) || result.Values["category"] != "Food" || result.Values["day"] != float64(12) {
		t.Fatalf("values = %#v", result.Values)
	}
}

func TestFillEditSendsWhatTheFormHolds(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"values": {"targetDate": "2028-06"}}`}}}
	result, err := newTestService(t, fixtureDatabase(), client, nil).Fill(context.Background(), domain.FillRequest{
		Form: "goal", Text: "push it a year", Current: map[string]any{"name": "Trip", "targetDate": "2027-06"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Filled, []string{"targetDate"}) {
		t.Errorf("filled = %v", result.Filled)
	}
	if !strings.Contains(client.requests[0].Messages[0].Content, `{"name":"Trip","targetDate":"2027-06"}`) {
		t.Errorf("the current values are not sent:\n%s", client.requests[0].Messages[0].Content)
	}
}

func TestFillRetriesTheFallbackModelOnABadAnswer(t *testing.T) {
	config := testConfig()
	config.FallbackModel = "fallback"
	client := &scriptedClient{replies: []reply{{content: "Here you go"}, {content: `{"values": {"from": "Food", "to": "Transport", "amount": "500k"}}`}}}
	svc := newTestService(t, fixtureDatabase(), client, nil)
	svc.config = config
	result, err := svc.Fill(context.Background(), domain.FillRequest{Form: "budget-move", Text: "move 500k from food to transport"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(client.models(), []string{"primary", "fallback"}) || result.Values["amount"] != float64(500_000) {
		t.Fatalf("models %v, values %#v", client.models(), result.Values)
	}
	if !strings.Contains(client.requests[0].Messages[0].Content, "- Food: 3000000") {
		t.Errorf("the month's limits are not given:\n%s", client.requests[0].Messages[0].Content)
	}
}

func TestFillGivesTheGuidancePrompt(t *testing.T) {
	db := fixtureDatabase()
	db.AIContent.Prompts[keyFormFill] = "Amounts I write without a unit are thousands."
	client := &scriptedClient{replies: []reply{{content: `{"values": {}}`}}}
	result, err := newTestService(t, db, client, nil).Fill(context.Background(), domain.FillRequest{Form: "allocation", Text: "half funds"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(client.requests[0].Messages[0].Content, "Amounts I write without a unit are thousands.") {
		t.Error("the tasks/form-fill prompt is not given")
	}
	if len(result.Filled) != 0 {
		t.Errorf("an answer without values fills nothing: %v", result.Filled)
	}
}

func TestFillRejectsBadRequestsWithoutCallingTheModel(t *testing.T) {
	client := &scriptedClient{}
	_, err := newTestService(t, fixtureDatabase(), client, nil).Fill(context.Background(), domain.FillRequest{Form: "taxes", Text: "x"})
	if !errors.Is(err, domain.ErrValidation) || len(client.requests) != 0 {
		t.Fatalf("err = %v, calls = %d", err, len(client.requests))
	}
}

func TestFillPayeesAreMostUsedFirst(t *testing.T) {
	db := fixtureDatabase()
	got := frequentPayees(db.Transactions, 2)
	if !reflect.DeepEqual(got, []string{"Highlands Coffee", "Grab"}) {
		t.Fatalf("payees = %v", got)
	}
}
