package domain

import (
	"fmt"
)

type AssetClass string

const (
	AssetClassCash       AssetClass = "cash"
	AssetClassInvestment AssetClass = "investment"
	AssetClassCrypto     AssetClass = "crypto"
)

var assetClasses = []AssetClass{AssetClassCash, AssetClassInvestment, AssetClassCrypto}

// AssetClassFor maps an account type to an asset class; liabilities and deprecated accounts have none.
func AssetClassFor(accountType AccountType) (AssetClass, bool) {
	switch accountType {
	case AccountTypeActiveUse, AccountTypeRarelyUse, AccountTypeBank, AccountTypeCash:
		return AssetClassCash, true
	case AccountTypeInvestment, AccountTypeLongHolding:
		return AssetClassInvestment, true
	case AccountTypeCrypto:
		return AssetClassCrypto, true
	default:
		return "", false
	}
}

// AssetAllocation sums positive balances per asset class.
func AssetAllocation(accounts []Account) map[AssetClass]float64 {
	allocation := map[AssetClass]float64{}
	for _, account := range accounts {
		class, ok := AssetClassFor(account.Type)
		if ok && account.Balance > 0 {
			allocation[class] += account.Balance
		}
	}
	return allocation
}

// diversificationScore returns the HHI and its score: 1/N (even) scores full, 1 (single class) scores zero.
func diversificationScore(allocation map[AssetClass]float64) (float64, float64) {
	total := 0.0
	for _, value := range allocation {
		total += value
	}
	if total <= 0 {
		return 1, 0
	}
	hhi := 0.0
	for _, value := range allocation {
		share := value / total
		hhi += share * share
	}
	minimum := 1 / float64(len(assetClasses))
	return hhi, healthDiversificationMax * clampUnit((1-hhi)/(1-minimum))
}

func allocationDetail(allocation map[AssetClass]float64) string {
	total := 0.0
	for _, value := range allocation {
		total += value
	}
	detail := ""
	for _, class := range assetClasses {
		share := 0.0
		if total > 0 {
			share = allocation[class] / total * 100
		}
		if detail != "" {
			detail += ", "
		}
		detail += fmt.Sprintf("%s %.1f%%", class, share)
	}
	return detail
}
