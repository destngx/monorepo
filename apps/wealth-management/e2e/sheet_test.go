package e2e

import (
	"apps/wealth-management/domain"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

// Assertions are structural or relative because the workbook holds current real
// data (WM-TEST-5).

// sheetTags are the OpenAPI tags of routes that read the sheet.
var sheetTags = []string{"accounts", "transactions", "budget", "categories", "goals", "lending", "tags", "investments", "capture", "init", "analytics", "preferences"}

func TestE2EEveryReadRouteParsesTheSheet(t *testing.T) {
	h := engine(t)
	var spec struct {
		Paths map[string]map[string]struct {
			Tags []string `json:"tags"`
		} `json:"paths"`
	}
	h.request(t, http.MethodGet, "/api/openapi.json", nil, http.StatusOK, &spec)

	var paths []string
	for path, operations := range spec.Paths {
		get, ok := operations["get"]
		if !ok || strings.Contains(path, "{") || !slices.ContainsFunc(get.Tags, func(tag string) bool { return slices.Contains(sheetTags, tag) }) {
			continue
		}
		paths = append(paths, path)
	}
	slices.Sort(paths)
	if len(paths) < 10 {
		t.Fatalf("expected the sheet read routes in the spec, found %v", paths)
	}
	for _, path := range paths {
		t.Run(strings.TrimPrefix(path, "/api/"), func(t *testing.T) {
			var body any
			h.request(t, http.MethodGet, path+"?force=true", nil, http.StatusOK, &body)
		})
	}
}

func TestE2EReadsAreStructurallySound(t *testing.T) {
	h := engine(t)
	accounts := listAccounts(t, h)
	if len(accounts) == 0 {
		t.Fatal("the workbook has no accounts")
	}
	for _, account := range accounts {
		if account.Name == "" || account.Type == "" {
			t.Fatalf("account without name or type: %+v", account)
		}
	}
	for _, transaction := range listTransactions(t, h) {
		if !strings.HasPrefix(transaction.ID, "row-") || transaction.ETag == "" || transaction.AccountName == "" {
			t.Fatalf("transaction without id, etag or account: %+v", transaction)
		}
	}
	var initResult domain.InitResult
	h.request(t, http.MethodGet, "/api/init", nil, http.StatusOK, &initResult)
	if !initResult.Ready || initResult.Loaded.Prompts == 0 {
		t.Fatalf("the workbook must carry the Prompts tab: %+v", initResult)
	}
}

func TestE2EWritesNeedTheAPIToken(t *testing.T) {
	h := engine(t)
	status, _, err := h.call(http.MethodPost, "/api/accounts", map[string]any{"name": "unauthorized", "type": "Cash"}, false)
	if err != nil || status != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a token, got %d, %v", status, err)
	}
}

func TestE2EAccountRoundTrip(t *testing.T) {
	h := engine(t)
	existing := listAccounts(t, h)
	name := "wm-e2e " + h.runID + " account"
	h.request(t, http.MethodPost, "/api/accounts", map[string]any{"name": name, "type": existing[0].Type, "note": "created by e2e"}, http.StatusCreated, nil)

	created, ok := findAccount(listAccounts(t, h), name)
	if !ok || created.Type != existing[0].Type {
		t.Fatalf("the new account did not read back: %+v", created)
	}
	path := "/api/accounts/" + url.PathEscape(name)
	h.request(t, http.MethodPatch, path, map[string]any{"note": "edited by e2e"}, http.StatusOK, nil)
	if edited, _ := findAccount(listAccounts(t, h), name); edited.Note == nil || *edited.Note != "edited by e2e" {
		t.Fatalf("the note edit did not read back: %+v", edited)
	}
	h.request(t, http.MethodDelete, path, nil, http.StatusOK, nil)
	if _, still := findAccount(listAccounts(t, h), name); still {
		t.Fatal("a soft-deleted account must leave the list")
	}
}

func TestE2ETransactionRoundTrip(t *testing.T) {
	h := engine(t)
	account := listAccounts(t, h)[0].Name
	category := firstCategory(t, h)
	before := len(listTransactions(t, h))
	payee := "wm-e2e " + h.runID + " single"

	var created domain.TransactionWriteResult
	h.request(t, http.MethodPost, "/api/transactions", transactionInput(account, payee, category, 1000), http.StatusCreated, &created)
	transactions := listTransactions(t, h)
	row, ok := findTransaction(transactions, created.ID)
	if len(transactions) != before+1 || !ok || row.Payee != payee || row.Payment == nil || *row.Payment != 1000 {
		t.Fatalf("append did not read back: %d -> %d rows, %+v", before, len(transactions), row)
	}

	update := transactionInput(account, payee, category, 1000)
	update["memo"] = "edited by e2e"
	update["etag"] = row.ETag
	h.request(t, http.MethodPatch, "/api/transactions/"+created.ID, update, http.StatusOK, nil)
	edited, _ := findTransaction(listTransactions(t, h), created.ID)
	if edited.Memo == nil || *edited.Memo != "edited by e2e" {
		t.Fatalf("the edit did not read back: %+v", edited)
	}

	// The pre-edit ETag is stale now.
	h.request(t, http.MethodDelete, "/api/transactions/"+created.ID, nil, http.StatusConflict, nil, "If-Match", row.ETag)
	h.request(t, http.MethodDelete, "/api/transactions/"+created.ID, nil, http.StatusOK, nil, "If-Match", edited.ETag)
	if _, still := findTransaction(listTransactions(t, h), created.ID); still {
		t.Fatal("the deleted transaction is still listed")
	}
}

func TestE2ETransactionBatchIsIdempotent(t *testing.T) {
	h := engine(t)
	account := listAccounts(t, h)[0].Name
	category := firstCategory(t, h)
	before := len(listTransactions(t, h))
	batch := map[string]any{
		"batchId": "wm-e2e-" + h.runID,
		"transactions": []any{
			transactionInput(account, "wm-e2e "+h.runID+" batch 1", category, 2000),
			transactionInput(account, "wm-e2e "+h.runID+" batch 2", category, 3000),
		},
	}
	var first, retry domain.TransactionBatchResult
	h.request(t, http.MethodPost, "/api/transactions/batch", batch, http.StatusOK, &first)
	h.request(t, http.MethodPost, "/api/transactions/batch", batch, http.StatusOK, &retry)
	if first.Created != 2 || first.Failed != 0 || !retry.Replayed {
		t.Fatalf("first = %+v, retry = %+v", first, retry)
	}
	transactions := listTransactions(t, h)
	if len(transactions) != before+2 {
		t.Fatalf("a replayed batch must not write again: %d -> %d rows", before, len(transactions))
	}
	for _, written := range first.Rows {
		row, ok := findTransaction(transactions, written.ID)
		if !ok {
			t.Fatalf("batch row %s did not read back", written.ID)
		}
		h.request(t, http.MethodDelete, "/api/transactions/"+row.ID, nil, http.StatusOK, nil, "If-Match", row.ETag)
	}
}

func TestE2EBudgetLimitRoundTrip(t *testing.T) {
	h := engine(t)
	month := time.Now().Format("2006-01")
	var items []domain.BudgetItem
	h.request(t, http.MethodGet, "/api/budget?force=true", nil, http.StatusOK, &items)
	index := slices.IndexFunc(items, func(item domain.BudgetItem) bool {
		_, ok := item.MonthlyLimits[month]
		return ok
	})
	if index < 0 {
		t.Skipf("the budget tab has no %s column to edit", month)
	}
	item := items[index]
	original := item.MonthlyLimits[month]
	update := map[string]any{"category": item.Category, "month": month, "limit": original + 1000}
	h.request(t, http.MethodPatch, "/api/budget/limits", update, http.StatusOK, nil)
	h.request(t, http.MethodGet, "/api/budget?force=true", nil, http.StatusOK, &items)
	if got := items[slices.IndexFunc(items, func(i domain.BudgetItem) bool { return i.Category == item.Category })].MonthlyLimits[month]; got != original+1000 {
		t.Fatalf("%s %s limit = %v, want %v", item.Category, month, got, original+1000)
	}
	update["limit"] = original
	h.request(t, http.MethodPatch, "/api/budget/limits", update, http.StatusOK, nil)
}

func TestE2ECaptureRulesRoundTrip(t *testing.T) {
	h := engine(t)
	pattern := "wm-e2e-" + h.runID
	rule := domain.CaptureRule{Kind: domain.CaptureRulePayee, Pattern: pattern, Value: firstCategory(t, h)}
	h.request(t, http.MethodPost, "/api/capture/rules", map[string]any{"rules": []domain.CaptureRule{rule}}, http.StatusOK, nil)
	var rules []domain.CaptureRule
	h.request(t, http.MethodGet, "/api/capture/rules?force=true", nil, http.StatusOK, &rules)
	if !slices.ContainsFunc(rules, func(r domain.CaptureRule) bool { return r.Pattern == pattern && r.Value == rule.Value }) {
		t.Fatalf("the saved rule did not read back from %d rules", len(rules))
	}
}

func TestE2ECategoryNatureRoundTrip(t *testing.T) {
	h := engine(t)
	var categories []domain.Category
	h.request(t, http.MethodGet, "/api/categories?force=true", nil, http.StatusOK, &categories)
	if len(categories) == 0 {
		t.Fatal("the workbook has no categories")
	}
	category := categories[0]
	target := domain.NatureFlexible
	if category.Nature == target {
		target = domain.NatureEssential
	}
	original := string(category.Nature)
	if original == "" {
		original = domain.CategoryNatureNone
	}
	update := map[string]any{"category": category.Name, "nature": string(target)}
	status, _, err := h.call(http.MethodPatch, "/api/categories/nature", update, true)
	if err != nil {
		t.Fatal(err)
	}
	if status == http.StatusConflict {
		t.Skipf("%s appears more than once in the Categories tab", category.Name)
	}
	if status != http.StatusOK {
		t.Fatalf("PATCH /api/categories/nature = %d", status)
	}
	defer h.request(t, http.MethodPatch, "/api/categories/nature", map[string]any{"category": category.Name, "nature": original}, http.StatusOK, nil)
	h.request(t, http.MethodGet, "/api/categories?force=true", nil, http.StatusOK, &categories)
	index := slices.IndexFunc(categories, func(c domain.Category) bool { return c.Name == category.Name })
	if index < 0 || categories[index].Nature != target {
		t.Fatalf("%s nature did not read back as %s", category.Name, target)
	}
}

func TestE2EMoneyLentAddsUp(t *testing.T) {
	h := engine(t)
	var ledger domain.MoneyLent
	h.request(t, http.MethodGet, "/api/lending?force=true", nil, http.StatusOK, &ledger)
	var owed float64
	for _, person := range ledger.People {
		if diff := person.Lent - person.Repaid - person.Outstanding; diff > 1 || diff < -1 {
			t.Fatalf("outstanding is not lent - repaid for a person (diff %v)", diff)
		}
		if person.Outstanding > 0 {
			owed += person.Outstanding
		}
	}
	if diff := owed - ledger.OwedToYou; diff > 1 || diff < -1 {
		t.Fatalf("owedToYou does not add up (diff %v)", diff)
	}
}

func TestE2EPreferencesRoundTrip(t *testing.T) {
	h := engine(t)
	var before domain.Preferences
	h.request(t, http.MethodGet, "/api/preferences?force=true", nil, http.StatusOK, &before)
	months := 7.0
	if before.GoalPool.EmergencyMonths == months {
		months = 8
	}
	update := domain.PreferencesUpdate{Values: []domain.PreferenceValue{{Key: domain.PrefGoalEmergencyMonths, Value: &months}}}
	var saved domain.Preferences
	h.request(t, http.MethodPost, "/api/preferences", update, http.StatusOK, &saved)
	if !saved.TabExists || saved.GoalPool.EmergencyMonths != months {
		t.Fatalf("the saved value did not read back: %+v", saved.GoalPool)
	}
	var after domain.Preferences
	h.request(t, http.MethodGet, "/api/preferences?force=true", nil, http.StatusOK, &after)
	if after.GoalPool.EmergencyMonths != months {
		t.Fatalf("emergencyMonths = %v after save", after.GoalPool.EmergencyMonths)
	}
}

func TestE2EAllocationAddsUp(t *testing.T) {
	h := engine(t)
	var allocation domain.Allocation
	h.request(t, http.MethodGet, "/api/investments/allocation?force=true", nil, http.StatusOK, &allocation)
	var targeted, total float64
	for _, line := range allocation.Lines {
		if line.Value < 0 {
			t.Fatalf("%s is negative", line.Class)
		}
		total += line.Value
		if line.Targeted {
			targeted += line.Value
		}
	}
	if diff := total - allocation.Total; diff > 1 || diff < -1 {
		t.Fatalf("lines do not add up to the total (diff %v)", diff)
	}
	if diff := targeted - allocation.TargetedTotal; diff > 1 || diff < -1 {
		t.Fatalf("target lines do not add up to the targeted total (diff %v)", diff)
	}
	var health domain.FinancialHealth
	h.request(t, http.MethodGet, "/api/analytics/health", nil, http.StatusOK, &health)
	if diff := health.Allocation.TargetedTotal - allocation.TargetedTotal; diff > 1 || diff < -1 {
		t.Fatalf("health and investments disagree on the allocation (diff %v)", diff)
	}
}

func listAccounts(t *testing.T, h *harness) []domain.Account {
	t.Helper()
	var accounts []domain.Account
	h.request(t, http.MethodGet, "/api/accounts?force=true", nil, http.StatusOK, &accounts)
	return accounts
}

func listTransactions(t *testing.T, h *harness) []domain.Transaction {
	t.Helper()
	var transactions []domain.Transaction
	h.request(t, http.MethodGet, "/api/transactions?force=true", nil, http.StatusOK, &transactions)
	return transactions
}

func firstCategory(t *testing.T, h *harness) string {
	t.Helper()
	var categories []domain.Category
	h.request(t, http.MethodGet, "/api/categories?force=true", nil, http.StatusOK, &categories)
	for _, category := range categories {
		if category.Name != "" {
			return category.Name
		}
	}
	t.Fatal("the workbook has no categories")
	return ""
}

func findAccount(accounts []domain.Account, name string) (domain.Account, bool) {
	index := slices.IndexFunc(accounts, func(a domain.Account) bool { return a.Name == name })
	if index < 0 {
		return domain.Account{}, false
	}
	return accounts[index], true
}

func findTransaction(transactions []domain.Transaction, id string) (domain.Transaction, bool) {
	index := slices.IndexFunc(transactions, func(tx domain.Transaction) bool { return tx.ID == id })
	if index < 0 {
		return domain.Transaction{}, false
	}
	return transactions[index], true
}

func transactionInput(account string, payee string, category string, payment float64) map[string]any {
	return map[string]any{
		"accountName": account,
		"date":        time.Now().Format("2006-01-02"),
		"payee":       payee,
		"category":    category,
		"payment":     payment,
		"memo":        fmt.Sprintf("e2e %s", payee),
	}
}

func TestE2EFundPerformanceAddsUpAndOnlyPricesReconciledHistory(t *testing.T) {
	h := engine(t)
	var performance domain.FundPerformance
	h.request(t, http.MethodGet, "/api/investments/fund-performance?force=true", nil, http.StatusOK, &performance)
	var value float64
	known := 0
	for _, holding := range performance.Holdings {
		if holding.Value != nil {
			value += *holding.Value
		}
		if holding.CostStatus != domain.FundCostKnown {
			if holding.Return != nil {
				t.Fatalf("%s has a return without a known cost", holding.Symbol)
			}
			continue
		}
		known++
		if holding.Units == nil || holding.HistoryUnits-*holding.Units > 0.01 || *holding.Units-holding.HistoryUnits > 0.01 {
			t.Fatalf("%s is priced but its History does not reconcile with Total Unit", holding.Symbol)
		}
		if holding.Return == nil || len(holding.CashFlows) == 0 {
			t.Fatalf("%s has a known cost without cash flows", holding.Symbol)
		}
	}
	if known != performance.Totals.CostKnown || len(performance.Holdings) != performance.Totals.Holdings {
		t.Fatalf("totals count %d known of %d, holdings say %d of %d", performance.Totals.CostKnown, performance.Totals.Holdings, known, len(performance.Holdings))
	}
	if diff := value - performance.Totals.Value; diff > 1 || diff < -1 {
		t.Fatalf("holdings do not add up to the total value (diff %v)", diff)
	}
}
