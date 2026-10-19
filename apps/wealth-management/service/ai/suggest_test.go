package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSuggestCategory(t *testing.T) {
	ctx := context.Background()

	client := &scriptedClient{}
	suggestion, err := newTestService(t, fixtureDatabase(), client, nil).SuggestCategory(ctx, "highlands coffee", "")
	if err != nil || suggestion != (port.CategorySuggestion{Category: "Coffee", Source: "history"}) || len(client.requests) != 0 {
		t.Fatalf("history should win without a model call: %+v %v %d", suggestion, err, len(client.requests))
	}

	db := fixtureDatabase()
	db.CaptureRules = []domain.CaptureRule{{Kind: domain.CaptureRulePayee, Pattern: "Highlands  Coffee", Value: "food"}}
	suggestion, err = newTestService(t, db, client, nil).SuggestCategory(ctx, "highlands coffee", "")
	if err != nil || suggestion != (port.CategorySuggestion{Category: "Food", Source: "rule"}) || len(client.requests) != 0 {
		t.Fatalf("a confirmed payee rule should outrank history: %+v %v", suggestion, err)
	}

	client = &scriptedClient{replies: []reply{{content: "transport."}}}
	suggestion, err = newTestService(t, fixtureDatabase(), client, nil).SuggestCategory(ctx, "Be Taxi", "")
	if err != nil || suggestion != (port.CategorySuggestion{Category: "Transport", Source: "ai"}) {
		t.Fatalf("unexpected AI suggestion %+v %v", suggestion, err)
	}
	if !strings.Contains(client.requests[0].Messages[0].Content, "Coffee, Food, Transport, Uncategorized") {
		t.Fatalf("categories not rendered: %s", client.requests[0].Messages[0].Content)
	}

	client = &scriptedClient{replies: []reply{{content: "Groceries"}, {content: "Shopping"}}}
	suggestion, err = newTestService(t, fixtureDatabase(), client, nil).SuggestCategory(ctx, "Be Taxi", "")
	if err != nil || suggestion.Category != "Uncategorized" || suggestion.Source != "default" {
		t.Fatalf("unknown answers should fall back to Uncategorized: %+v %v", suggestion, err)
	}

	if _, err := newTestService(t, fixtureDatabase(), client, nil).SuggestCategory(ctx, "ab", ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("short payee should be rejected, got %v", err)
	}
}

// guidedDatabase gives Food a definition and an eating-out split tag.
func guidedDatabase() *fakes.Database {
	db := fixtureDatabase()
	for i := range db.Categories {
		if db.Categories[i].Name == "Food" {
			db.Categories[i].Definition = "Food bought for home, and everyday meals out."
			db.Categories[i].SplitTags = []string{"eating-out"}
		}
	}
	return db
}

func TestGivenCategoryDefinitionsWhenTheModelSuggestsThenItSeesTheGuideAndMayAddASplitTag(t *testing.T) {
	ctx := context.Background()
	client := &scriptedClient{replies: []reply{{content: `{"category":"food","splitTag":"Eating-Out"}`}}}

	suggestion, err := newTestService(t, guidedDatabase(), client, nil).SuggestCategory(ctx, "Pho Corner", "")

	if err != nil || suggestion != (port.CategorySuggestion{Category: "Food", Source: "ai", SplitTag: "eating-out"}) {
		t.Fatalf("suggestion = %+v %v", suggestion, err)
	}
	user := client.requests[0].Messages[1].Content
	if !strings.Contains(user, "- Food: Food bought for home, and everyday meals out. Split tags: eating-out.") || !strings.Contains(user, `"splitTag"`) {
		t.Fatalf("the guide and the split tag contract must be in the request: %s", user)
	}
}

func TestGivenASplitTagThatIsNotTheCategorysWhenTheModelSuggestsThenItIsDropped(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"category":"Transport","splitTag":"eating-out"}`}}}

	suggestion, err := newTestService(t, guidedDatabase(), client, nil).SuggestCategory(context.Background(), "Be Taxi", "")

	if err != nil || suggestion != (port.CategorySuggestion{Category: "Transport", Source: "ai"}) {
		t.Fatalf("suggestion = %+v %v", suggestion, err)
	}
}

func TestGivenAPayeeUsuallyTaggedInItsCategoryWhenSuggestingFromHistoryThenTheSplitTagComesToo(t *testing.T) {
	db := guidedDatabase()
	db.Transactions = append(db.Transactions,
		domain.Transaction{AccountName: "Bank", Date: testToday.AddDate(0, 0, -5), Payee: "Pho Corner", Category: "Food", Tags: []string{"eating-out"}, Payment: money(60_000)},
		domain.Transaction{AccountName: "Bank", Date: testToday.AddDate(0, 0, -4), Payee: "pho corner", Category: "Food", Tags: []string{"silver", "Eating-Out"}, Payment: money(70_000)},
		domain.Transaction{AccountName: "Bank", Date: testToday.AddDate(0, 0, -3), Payee: "Pho Corner", Category: "Food", Payment: money(65_000)},
	)
	client := &scriptedClient{}

	suggestion, err := newTestService(t, db, client, nil).SuggestCategory(context.Background(), "Pho Corner", "")

	if err != nil || suggestion != (port.CategorySuggestion{Category: "Food", Source: "history", SplitTag: "eating-out"}) || len(client.requests) != 0 {
		t.Fatalf("suggestion = %+v %v calls=%d", suggestion, err, len(client.requests))
	}
}
