package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestYearReviewSendsTheEngineFactsForTheYear(t *testing.T) {
	db := fixtureDatabase()
	db.AIContent.Prompts["tasks/year-review"] = "Year {{year}}, verdict {{verdict}} at {{savingsRate}}%. {{yearJson}}"
	client := &scriptedClient{replies: []reply{{content: validInsight}}}

	if _, err := newTestService(t, db, client, nil).Review(context.Background(), port.ReviewRequest{Kind: port.ReviewYear}); err != nil {
		t.Fatal(err)
	}
	system := client.requests[0].Messages[0].Content
	year := testToday.Format("2006")
	if !strings.Contains(system, "Year "+year+", verdict noData at 0.0%.") || !strings.Contains(system, `"expenseByCategory":[{"category":"Transport"`) {
		t.Fatalf("task prompt not rendered with the year: %s", system)
	}
	user := client.requests[0].Messages[1].Content
	if !strings.Contains(user, "[actions/year-review]") || !strings.Contains(user, `"comparedWith":"the same months of `) || !strings.Contains(user, `"yearEndProjection"`) ||
		!strings.Contains(user, `"needsVsWants":{"fixed":0,"essential":0,"flexible":0,"untagged":`) {
		t.Fatalf("facts not attached: %s", user)
	}
}

func TestYearReviewOfAYearNotStartedIsAValidationError(t *testing.T) {
	_, err := newTestService(t, fixtureDatabase(), &scriptedClient{}, nil).Review(context.Background(), port.ReviewRequest{Kind: port.ReviewYear, Year: testToday.Year() + 1})
	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v", err)
	}
}

func TestTopCategoriesFoldsTheTailIntoOther(t *testing.T) {
	rows := []domain.CategoryMonths{
		{Category: "Rent", Values: []float64{5, 5, 0}, Total: 10},
		{Category: "Food", Values: []float64{2, 3, 0}, Total: 5},
		{Category: "Coffee", Values: []float64{1, 0, 0}, Total: 1},
		{Category: "Books", Values: []float64{0, 1, 0}, Total: 1},
	}
	got := topCategories(rows, 2, 2)
	want := []domain.CategoryMonths{
		{Category: "Rent", Values: []float64{5, 5}, Total: 10},
		{Category: "Food", Values: []float64{2, 3}, Total: 5},
		{Category: "Other", Values: []float64{1, 1}, Total: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if short := topCategories(rows[:1], 2, 3); len(short) != 1 || short[0].Category != "Rent" {
		t.Fatalf("no tail, no Other: %+v", short)
	}
}
