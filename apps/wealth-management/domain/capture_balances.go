package domain

import (
	"sort"
	"strings"
)

// CaptureBalances reports, per matched account, the latest balance shown on
// a material. Credit cards are skipped: their apps show available credit,
// not the ledger balance.
func CaptureBalances(items []CaptureItem, accounts []Account) []CaptureBalance {
	byName := map[string]Account{}
	for _, account := range accounts {
		byName[account.Name] = account
	}
	latest := map[string]CaptureItem{}
	for _, item := range items {
		account, ok := byName[item.Account.Name]
		if !ok || item.Event.BalanceAfter == nil || account.Type == AccountTypeNegativeActive {
			continue
		}
		if current, seen := latest[account.Name]; !seen || item.Event.At.After(current.Event.At) {
			latest[account.Name] = item
		}
	}
	balances := make([]CaptureBalance, 0, len(latest))
	for name, item := range latest {
		asOf := item.Event.At.Format(captureTimeLayout)
		if !item.Event.HasTime {
			asOf = item.Event.At.Format("2006-01-02")
		}
		balances = append(balances, CaptureBalance{
			Account:      name,
			Currency:     item.Event.Currency,
			BalanceAfter: *item.Event.BalanceAfter,
			AsOf:         asOf,
			MaterialID:   item.Event.MaterialID,
			SheetBalance: byName[name].Balance,
		})
	}
	sort.Slice(balances, func(a, b int) bool { return balances[a].Account < balances[b].Account })
	return balances
}

// TransferCategory picks the non-budget category used for own-account
// transfers: one named like "transfer" first, else none.
func TransferCategory(categories []Category) (string, bool) {
	for _, category := range categories {
		folded := FoldText(category.Name)
		if category.Type == CategoryTypeNonBudget && (strings.Contains(folded, "transfer") || strings.Contains(folded, "chuyen")) {
			return category.Name, true
		}
	}
	return "", false
}
