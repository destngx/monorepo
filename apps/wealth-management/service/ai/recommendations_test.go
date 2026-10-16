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

func newRecommendingService(t *testing.T, db *fakes.Database, client *scriptedClient, recommendations port.RecommendationService) *Service {
	t.Helper()
	now := func() time.Time { return testToday }
	analytics := service.NewAnalyticsService(db, now)
	if recommendations == nil {
		recommendations = service.NewRecommendationService(db, analytics, nil, logger.NewTestLogger(t), now)
	}
	return New(client, db, analytics, nil, testConfig(), logger.NewTestLogger(t), Options{Now: now, Recommendations: recommendations})
}

const recommendingInsight = `{"summary":"Spending is up.","sections":[{"icon":"wallet","title":"Food","content":"Food is over.","severity":"warning"}],"actionItems":["Cook"],
	"recommendations":[{"title":"Cook at home","action":"Cook twice a week","impactVnd":1200000,"evidence":"Food over its limit"},{"title":"","action":"dropped"}],
	"outcomes":[]}`

func TestGivenABudgetReviewWhenItRecommendsThenTheItemsAreAddedAndTheNextReviewChecksThem(t *testing.T) {
	db := fixtureDatabase()
	client := &scriptedClient{replies: []reply{{content: recommendingInsight}}}
	svc := newRecommendingService(t, db, client, nil)
	ctx := context.Background()

	insight, err := svc.Review(ctx, port.ReviewRequest{Kind: port.ReviewBudget})
	if err != nil {
		t.Fatal(err)
	}
	user := client.requests[0].Messages[1].Content
	if !strings.Contains(user, `"recommendations": [{"title"`) || !strings.Contains(user, "PREVIOUS RECOMMENDATIONS: none.") {
		t.Fatalf("the review must ask for recommendations: %s", user)
	}
	if insight.Recorded == nil || len(insight.Recorded.Created) != 1 || insight.Recorded.Error != "" {
		t.Fatalf("recorded = %+v", insight.Recorded)
	}
	created := insight.Recorded.Created[0]
	if created.Source != "budget-review" || created.Page != "/budget" || *created.ImpactVND != 1_200_000 || created.Status != domain.RecommendationOpen {
		t.Fatalf("created = %+v", created)
	}

	client.replies = []reply{{content: strings.Replace(recommendingInsight, `"outcomes":[]`, `"outcomes":[{"id":"`+created.ID+`","verdict":"met","note":"Food is under the limit"},{"id":"rec-other","verdict":"met"}]`, 1)}}
	again, err := svc.Review(ctx, port.ReviewRequest{Kind: port.ReviewBudget, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	user = client.requests[1].Messages[1].Content
	if !strings.Contains(user, `"id":"`+created.ID+`"`) || !strings.Contains(user, `"title":"Cook at home"`) {
		t.Fatalf("the next review must see the earlier item: %s", user)
	}
	record := again.Recorded
	if len(record.Created) != 0 || record.Duplicates != 1 || len(record.Checked) != 1 || record.IgnoredOutcomes != 1 {
		t.Fatalf("second record = %+v", record)
	}
	if checked := record.Checked[0]; checked.Status != domain.RecommendationDone || checked.Outcome != "Met: Food is under the limit" {
		t.Fatalf("checked = %+v", checked)
	}
}

// failingRecommendations lists nothing and fails to record.
type failingRecommendations struct{ port.RecommendationService }

func (failingRecommendations) RecentForReview(context.Context, string) ([]domain.Recommendation, error) {
	return []domain.Recommendation{}, nil
}

func (failingRecommendations) Record(context.Context, domain.ReviewRecommendations) (domain.RecommendationRecord, error) {
	return domain.RecommendationRecord{}, errors.New("sheet write failed")
}

func TestGivenTheListCannotBeWrittenWhenAReviewRunsThenTheReviewStandsAndSaysSo(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: recommendingInsight}}}
	insight, err := newRecommendingService(t, fixtureDatabase(), client, failingRecommendations{}).HealthCommentary(context.Background(), false, "")
	if err != nil {
		t.Fatal(err)
	}
	if insight.Summary == "" || insight.Recorded == nil || !strings.Contains(insight.Recorded.Error, "sheet write failed") {
		t.Fatalf("insight = %+v", insight)
	}
}

func TestGivenTheBriefingWhenItRecommendsThenItFeedsTheListForTheHomePage(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: `{"briefing":"All good.","alerts":[],"patterns":[],"recommendations":[{"title":"Ask Person B to repay","action":"Agree a date","impactVnd":null,"evidence":"Lent 30 days ago"}]}`}}}
	briefing, err := newRecommendingService(t, fixtureDatabase(), client, nil).Briefing(context.Background(), false, "")
	if err != nil {
		t.Fatal(err)
	}
	if briefing.Recorded == nil || len(briefing.Recorded.Created) != 1 || briefing.Recorded.Created[0].Source != "intelligence-briefing" || briefing.Recorded.Created[0].Page != "/" {
		t.Fatalf("recorded = %+v", briefing.Recorded)
	}
	if !strings.Contains(client.requests[0].Messages[1].Content, "PREVIOUS RECOMMENDATIONS") {
		t.Fatal("the briefing must get the recommendation contract")
	}
}

func TestGivenNoRecommendationsServiceWhenReviewingThenTheContractIsUnchanged(t *testing.T) {
	client := &scriptedClient{replies: []reply{{content: recommendingInsight}}}
	insight, err := newTestService(t, fixtureDatabase(), client, nil).Review(context.Background(), port.ReviewRequest{Kind: port.ReviewAccounts})
	if err != nil || insight.Recorded != nil {
		t.Fatalf("insight = %+v %v", insight, err)
	}
	if strings.Contains(client.requests[0].Messages[1].Content, "PREVIOUS RECOMMENDATIONS") {
		t.Fatal("without the list a review is not asked for recommendations")
	}
}

func TestTheChatCanReadTheRecommendationsWithNamesMasked(t *testing.T) {
	db := fixtureDatabase()
	db.Recommendations = domain.RecommendationSheet{Exists: true, Items: []domain.Recommendation{
		{ID: "rec-1", Created: "2026-10-01", Source: "intelligence-briefing", Title: "Ask Person B to repay", Action: "Agree a date", Status: domain.RecommendationOpen, Page: "/"},
	}}
	svc := newRecommendingService(t, db, &scriptedClient{}, nil)
	tool, ok := svc.chatTools()["getRecommendations"]
	if !ok {
		t.Fatal("getRecommendations is missing")
	}
	result, err := tool.run(context.Background(), []byte(`{"status":"open"}`))
	if err != nil {
		t.Fatal(err)
	}
	text := toJSON(result)
	if !strings.Contains(text, `"open":1`) || strings.Contains(text, "Person B") || !strings.Contains(text, "Ask [name] to repay") {
		t.Fatalf("result = %s", text)
	}
}
