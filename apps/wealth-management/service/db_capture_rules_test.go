package service

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
)

func TestListCaptureRulesWithoutTabIsEmpty(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	rules, err := newTestDBService(t, client, newFakeCache()).ListCaptureRules(true)
	if err != nil || len(rules) != 0 || rules == nil {
		t.Fatalf("missing tab should mean no rules: %v %v", rules, err)
	}
	for _, read := range client.reads {
		if read == captureRulesRange {
			t.Fatal("must not read a tab that does not exist")
		}
	}
}

func TestListCaptureRulesSkipsInvalidRows(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{captureRulesSheet}
	client.ranges[captureRulesRange] = [][]string{
		{"alias", "Sacombank 1234", "Credit Card Sacombank", "2026-10-01 10:00"},
		{"Payee", " Highlands ", "Food"},
		{"other", "x", "y"},
		{"alias", "", "Ví MoMo"},
	}
	rules, err := newTestDBService(t, client, newFakeCache()).ListCaptureRules(true)
	if err != nil || len(rules) != 2 || rules[1].Kind != domain.CaptureRulePayee || rules[1].Pattern != "Highlands" {
		t.Fatalf("rules: %+v %v", rules, err)
	}
}

func TestSaveCaptureRulesCreatesTabAndAppends(t *testing.T) {
	client := newFakeSheetClient()
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)
	result, err := svc.SaveCaptureRules([]domain.CaptureRule{
		{Kind: "Alias", Pattern: "MoMo", Value: "Ví MoMo"},
		{Kind: domain.CaptureRulePayee, Pattern: "Highlands", Value: "Food"},
		{Kind: domain.CaptureRulePayee, Pattern: "highlands", Value: "Coffee"},
	})
	if err != nil || result.Created != 2 {
		t.Fatalf("save: %+v %v", result, err)
	}
	if len(client.created) != 1 || client.created[0] != captureRulesSheet || len(client.appends) != 1 {
		t.Fatalf("expected tab creation then one append: %+v %+v", client.created, client.appends)
	}
	rows := client.appends[0].rows
	if len(rows) != 2 || rows[0][0] != "alias" || rows[1][2] != "Coffee" || rows[1][3] != "2026-10-12 16:00" {
		t.Fatalf("rows: %+v", rows)
	}
	if len(cache.invalidated) == 0 || cache.invalidated[len(cache.invalidated)-1] != cacheKeyCaptureRules {
		t.Fatalf("cache not invalidated: %v", cache.invalidated)
	}
}

func TestSaveCaptureRulesUpdatesChangedRowsOnly(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{captureRulesSheet}
	client.ranges[captureRulesRange] = [][]string{
		{"alias", "MoMo", "Ví MoMo"},
		{"payee", "Highlands", "Food"},
	}
	result, err := newTestDBService(t, client, newFakeCache()).SaveCaptureRules([]domain.CaptureRule{
		{Kind: domain.CaptureRuleAlias, Pattern: "momo", Value: "Ví MoMo"},
		{Kind: domain.CaptureRulePayee, Pattern: "HIGHLANDS", Value: "Coffee"},
		{Kind: domain.CaptureRulePayee, Pattern: "Grab", Value: "Transport"},
	})
	if err != nil || result != (domain.CaptureRulesSaveResult{Created: 1, Updated: 1, Unchanged: 1}) {
		t.Fatalf("save: %+v %v", result, err)
	}
	if len(client.created) != 0 || len(client.updates) != 1 || client.updates[0].rangeValue != "CaptureRules!A3:D3" || client.updates[0].values[2] != "Coffee" {
		t.Fatalf("updates: %+v", client.updates)
	}
}

func TestSaveCaptureRulesValidates(t *testing.T) {
	svc := newTestDBService(t, newFakeSheetClient(), newFakeCache())
	for _, rules := range [][]domain.CaptureRule{nil, {{Kind: "alias", Pattern: "x"}}} {
		if _, err := svc.SaveCaptureRules(rules); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("expected validation error for %+v, got %v", rules, err)
		}
	}
}
