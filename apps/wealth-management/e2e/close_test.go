package e2e

import (
	"apps/wealth-management/domain"
	"encoding/json"
	"math"
	"net/http"
	"testing"
)

// TestE2EMonthlyClose reads the checklist from the in-memory workbook,
// writes this month's snapshot twice and reads the history back
// (WM-CLOSE-1..4, WM-NW-1..6). Assertions are structural: the copy holds
// real data, and its writes are never saved.
func TestE2EMonthlyClose(t *testing.T) {
	h := engine(t)
	var close domain.MonthlyClose
	h.request(t, http.MethodGet, "/api/close?force=true", nil, http.StatusOK, &close)
	if len(close.Items) != 7 || close.Checked != 6 || close.Done > close.Checked {
		t.Fatalf("checklist: %+v", close)
	}

	var first, second domain.SnapshotSaveResult
	// The copy may already hold a row for the month: 201 adds it, 200 updates it.
	status, raw, err := h.call(http.MethodPost, "/api/close/snapshot", map[string]any{"month": ""}, true)
	if err != nil || (status != http.StatusCreated && status != http.StatusOK) || json.Unmarshal(raw, &first) != nil {
		t.Fatalf("first write: %d %v %s", status, err, raw)
	}
	h.request(t, http.MethodPost, "/api/close/snapshot", map[string]any{"month": first.Snapshot.Month}, http.StatusOK, &second)
	if second.Created || second.Snapshot.Row != first.Snapshot.Row || second.Snapshot.Month != first.Snapshot.Month {
		t.Fatalf("the second write must update the first row: %+v then %+v", first, second)
	}
	s := second.Snapshot
	if s.Assets == nil || s.Liabilities == nil || math.Abs(*s.Assets-*s.Liabilities-s.NetWorth) > 1 || s.HealthScore == nil {
		t.Fatalf("snapshot figures do not add up: %+v", s)
	}

	var history domain.NetWorthHistory
	h.request(t, http.MethodGet, "/api/net-worth/history?force=true", nil, http.StatusOK, &history)
	if len(history.Points) == 0 || !history.TabExists || history.Points[len(history.Points)-1].Source != domain.NetWorthLive {
		t.Fatalf("history: %+v", history)
	}
}
