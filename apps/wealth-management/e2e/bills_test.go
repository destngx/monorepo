package e2e

import (
	"apps/wealth-management/domain"
	"math"
	"net/http"
	"testing"
)

// TestE2EBillsRoundTrip reads the bills from the in-memory workbook, then
// creates, edits, pauses and deletes a recurring item and reads each change
// back (WM-BILL-1..7). Assertions are structural: the copy holds real data.
func TestE2EBillsRoundTrip(t *testing.T) {
	h := engine(t)
	var bills domain.Bills
	h.request(t, http.MethodGet, "/api/bills?force=true", nil, http.StatusOK, &bills)
	safe := bills.SafeToSpend
	parts := safe.ReadyMoney - safe.BillsDue - safe.CardPayments
	if math.Abs(safe.Amount-parts) > 1 || safe.SafeToSpend < 0 || safe.Until == "" || safe.Readiness.ETag == "" {
		t.Fatalf("safe-to-spend parts do not add up: %+v", safe)
	}
	for _, o := range safe.Counted {
		if safe.UntilIncome && o.ExpectedDate >= safe.Until {
			t.Fatalf("a payment due on or after payday is counted: %+v", o)
		}
	}
	// Marking a ready account not ready moves it out of ready money, and back.
	if len(safe.ReadyAccounts) > 0 {
		account := safe.ReadyAccounts[0]
		var prefs domain.Preferences
		h.request(t, http.MethodPost, "/api/bills/safe-to-spend/accounts", map[string]any{"account": account.Name, "ready": false, "etag": safe.Readiness.ETag}, http.StatusOK, &prefs)
		h.request(t, http.MethodPost, "/api/bills/safe-to-spend/accounts", map[string]any{"account": account.Name, "ready": true, "etag": safe.Readiness.ETag}, http.StatusConflict, nil)
		var marked domain.Bills
		h.request(t, http.MethodGet, "/api/bills?force=true", nil, http.StatusOK, &marked)
		if math.Abs(marked.SafeToSpend.ReadyMoney-(safe.ReadyMoney-account.Balance)) > 1 {
			t.Fatalf("the marked account still counts: %+v", marked.SafeToSpend)
		}
		h.request(t, http.MethodPost, "/api/bills/safe-to-spend/accounts", map[string]any{"account": account.Name, "ready": true, "etag": prefs.SafeToSpend.ETag}, http.StatusOK, nil)
	}
	for _, suggestion := range bills.Suggestions {
		if suggestion.Key == "" || suggestion.Amount <= 0 || suggestion.Seen < 2 {
			t.Fatalf("suggestion without evidence: %+v", suggestion)
		}
	}

	name := "wm-e2e " + h.runID + " bill"
	var created domain.RecurringItem
	h.request(t, http.MethodPost, "/api/bills/items", map[string]any{"name": name, "kind": "bill", "amount": 100_000, "day": 28, "match": name}, http.StatusCreated, &created)
	if created.ETag == "" || created.Day != 28 || created.Amount == nil || *created.Amount != 100_000 {
		t.Fatalf("the new item did not read back: %+v", created)
	}
	var edited domain.RecurringItem
	h.request(t, http.MethodPatch, "/api/bills/items/"+created.ID, map[string]any{"name": name, "kind": "subscription", "amount": 120_000, "day": 3, "match": name, "etag": created.ETag}, http.StatusOK, &edited)
	if edited.Kind != domain.RecurringSubscription || *edited.Amount != 120_000 {
		t.Fatalf("the edit did not read back: %+v", edited)
	}
	h.request(t, http.MethodPatch, "/api/bills/items/"+created.ID, map[string]any{"name": name, "kind": "bill", "amount": 1, "day": 1, "match": name, "etag": created.ETag}, http.StatusConflict, nil)
	var paused domain.RecurringItem
	h.request(t, http.MethodPost, "/api/bills/items/"+created.ID+"/status", map[string]any{"status": "paused", "etag": edited.ETag}, http.StatusOK, &paused)
	if paused.Status != domain.RecurringPaused {
		t.Fatalf("the pause did not read back: %+v", paused)
	}
	h.request(t, http.MethodDelete, "/api/bills/items/"+created.ID, nil, http.StatusOK, nil, "If-Match", paused.ETag)
	h.request(t, http.MethodGet, "/api/bills?force=true", nil, http.StatusOK, &bills)
	for _, item := range bills.Items {
		if item.ID == created.ID {
			t.Fatalf("the deleted item is still listed: %+v", item)
		}
	}
}

// TestE2EIdleCash reads the idle cash check from the in-memory workbook
// (WM-CASH-1..5). Assertions are structural: the copy holds real data.
func TestE2EIdleCash(t *testing.T) {
	h := engine(t)
	var idle domain.IdleCash
	h.request(t, http.MethodGet, "/api/bills/idle-cash?force=true", nil, http.StatusOK, &idle)
	c := idle.Components
	keep := c.BillsDue + c.CardPayments + c.GoalSetAside + c.SpendingBuffer + c.EmergencyFund
	if math.Abs(c.KeepLiquid-keep) > 1 || math.Abs(idle.Idle-math.Max(0, c.SpendableCash-c.KeepLiquid)) > 1 {
		t.Fatalf("idle cash parts do not add up: %+v", idle)
	}
	held := 0.0
	for _, account := range idle.Accounts {
		held += account.Idle
	}
	if math.Abs(held-idle.Idle) > 1 || idle.Verdict == "" || len(idle.Reasons) == 0 {
		t.Fatalf("idle cash is not spread over the accounts or has no verdict: %+v", idle)
	}
}
