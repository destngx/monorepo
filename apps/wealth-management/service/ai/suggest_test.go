package ai

import (
	"apps/wealth-management/domain"
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
