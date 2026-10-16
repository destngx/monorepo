package e2e

import (
	"apps/wealth-management/domain"
	"net/http"
	"slices"
	"testing"
)

// TestE2ENeedsYou reads the home inbox from the in-memory workbook
// (WM-HOME-1). Assertions are structural: the copy holds real data.
func TestE2ENeedsYou(t *testing.T) {
	h := engine(t)
	var inbox domain.NeedsYou
	h.request(t, http.MethodGet, "/api/home/needs-you?force=true", nil, http.StatusOK, &inbox)
	if inbox.AsOf == "" || inbox.Items == nil || inbox.Unavailable == nil {
		t.Fatalf("inbox: %+v", inbox)
	}
	severity := map[domain.NeedsYouStatus]int{domain.NeedsYouCritical: 0, domain.NeedsYouSerious: 1, domain.NeedsYouWarning: 2, domain.NeedsYouInfo: 3}
	if !slices.IsSortedFunc(inbox.Items, func(a, b domain.NeedsYouItem) int { return severity[a.Status] - severity[b.Status] }) {
		t.Fatalf("items must be ranked by severity: %+v", inbox.Items)
	}
	for _, item := range inbox.Items {
		if item.ID == "" || item.Headline == "" || item.Route == "" || item.Route[0] != '/' {
			t.Fatalf("item: %+v", item)
		}
	}
}
