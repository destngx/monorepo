package e2e

import (
	"apps/wealth-management/domain"
	"net/http"
	"testing"
)

// TestE2ERecommendationsRoundTrip runs the engine rules on the in-memory
// workbook, then snoozes and reopens an open item when there is one, reading
// each change back (WM-REC-1, 3, 5). Assertions are structural: the copy
// holds real data.
func TestE2ERecommendationsRoundTrip(t *testing.T) {
	h := engine(t)
	var refreshed domain.RecommendationList
	h.request(t, http.MethodPost, "/api/recommendations/refresh", nil, http.StatusOK, &refreshed)
	summary := refreshed.Summary
	if summary.Open+summary.Snoozed+summary.Done+summary.Dismissed != len(refreshed.Items) {
		t.Fatalf("the summary does not count the items: %+v", refreshed)
	}
	var again domain.RecommendationList
	h.request(t, http.MethodPost, "/api/recommendations/refresh", nil, http.StatusOK, &again)
	if len(again.Items) != len(refreshed.Items) {
		t.Fatalf("a second refresh must not add items: %d then %d", len(refreshed.Items), len(again.Items))
	}
	var open *domain.Recommendation
	for i, item := range again.Items {
		if item.Status == domain.RecommendationOpen {
			open = &again.Items[i]
			break
		}
	}
	if open == nil {
		t.Skip("no open recommendation in this workbook to change")
	}
	var snoozed domain.Recommendation
	h.request(t, http.MethodPost, "/api/recommendations/"+open.ID+"/status", map[string]any{"status": "snoozed", "snoozeDays": 7, "etag": open.ETag}, http.StatusOK, &snoozed)
	if snoozed.Status != domain.RecommendationSnoozed || snoozed.SnoozedUntil == "" || snoozed.Title != open.Title {
		t.Fatalf("the snooze did not read back: %+v", snoozed)
	}
	h.request(t, http.MethodPost, "/api/recommendations/"+open.ID+"/status", map[string]any{"status": "done", "etag": open.ETag}, http.StatusConflict, nil)
	var reopened domain.Recommendation
	h.request(t, http.MethodPost, "/api/recommendations/"+open.ID+"/status", map[string]any{"status": "open", "etag": snoozed.ETag}, http.StatusOK, &reopened)
	if reopened.Status != domain.RecommendationOpen || reopened.SnoozedUntil != "" {
		t.Fatalf("the reopen did not read back: %+v", reopened)
	}
}
