package domain

import (
	"fmt"
	"math"
	"time"
)

// One allocation (WM-INV-20, WM-ACC-8, WM-HEA-1; ROADMAP C4). Investments,
// Accounts and Health all show this answer, so they agree on every figure.
//
// Classes:
//   - cash: positive balances of every VND account (bank, cash, savings,
//     long holding and investment accounts), as the WM-RULE-8 VND accounts;
//   - funds: fund certificate holdings valued in VND (WM-RULE-8);
//   - crypto: the larger of positive crypto account balances and the crypto
//     holdings value (WM-RULE-8: the two track the same coins);
//   - money lent: what people still owe the user (D30). It is shown as its
//     own line but has no target and stays out of drift.
//
// Card debt is a liability, not a negative asset, so it is not netted
// against cash. Archived (deprecated) accounts are left out.

// AllocationClass is one line of the allocation.
type AllocationClass string

const (
	AllocationCash      AllocationClass = "cash"
	AllocationFunds     AllocationClass = "funds"
	AllocationCrypto    AllocationClass = "crypto"
	AllocationMoneyLent AllocationClass = "moneyLent"
)

// TargetClasses are the classes with a target share, in display order.
var TargetClasses = []AllocationClass{AllocationCash, AllocationFunds, AllocationCrypto}

// DriftThreshold: rebalance when a class is more than this many percentage
// points away from its target (WM-INV-20).
const DriftThreshold = 5.0

var allocationLabels = map[AllocationClass]string{
	AllocationCash:      "Cash and deposits",
	AllocationFunds:     "Fund certificates",
	AllocationCrypto:    "Crypto",
	AllocationMoneyLent: "Money lent",
}

// AllocationStatus says how a line compares with its target.
type AllocationStatus string

const (
	AllocationOnTarget AllocationStatus = "on_target"
	AllocationOver     AllocationStatus = "over"
	AllocationUnder    AllocationStatus = "under"
	// AllocationNoTarget: targets are not set, or the line never has one (money lent).
	AllocationNoTarget AllocationStatus = "no_target"
)

// AllocationLine is one class. Share is the percent of every allocated asset
// (money lent included); Current is the percent of the targeted total, the
// figure drift compares with Target.
type AllocationLine struct {
	Class    AllocationClass  `json:"class"`
	Label    string           `json:"label"`
	Value    float64          `json:"value"`
	Share    float64          `json:"share"`
	Targeted bool             `json:"targeted"`
	Current  *float64         `json:"current"`
	Target   *float64         `json:"target"`
	Drift    *float64         `json:"drift"`
	Status   AllocationStatus `json:"status"`
}

// Allocation is the whole answer. Total covers every line; TargetedTotal the
// target classes only. Rebalance is true when targets are set and a class
// drifts beyond DriftThreshold.
type Allocation struct {
	Lines          []AllocationLine   `json:"lines"`
	Total          float64            `json:"total"`
	TargetedTotal  float64            `json:"targetedTotal"`
	Targets        *AllocationTargets `json:"targets"`
	DriftThreshold float64            `json:"driftThreshold"`
	Rebalance      bool               `json:"rebalance"`
	// Unvalued counts holdings without a VND value, left out of the figures.
	Unvalued    int       `json:"unvalued"`
	Issues      []string  `json:"issues"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Line returns the line of a class.
func (a Allocation) Line(class AllocationClass) AllocationLine {
	for _, line := range a.Lines {
		if line.Class == class {
			return line
		}
	}
	return AllocationLine{Class: class, Label: allocationLabels[class]}
}

// AllocationInputs are the figures the allocation is built from. Funds and
// CryptoHoldings come from the portfolio valuation; without one they are 0.
type AllocationInputs struct {
	Accounts       []Account
	Transactions   []Transaction
	Funds          float64
	CryptoHoldings float64
	Unvalued       int
	Targets        *AllocationTargets
	Issues         []string
	Now            time.Time
}

// AllocationInputsFromValuation takes the holdings figures of a valuation.
func AllocationInputsFromValuation(valuation PortfolioValuation) AllocationInputs {
	return AllocationInputs{
		Funds:          valuation.Totals.Funds,
		CryptoHoldings: valuation.Totals.CryptoHoldings,
		Unvalued:       valuation.Totals.Unvalued,
		Issues:         append([]string(nil), valuation.Issues...),
		Now:            valuation.GeneratedAt,
	}
}

// isCryptoAccount: crypto type, or a USD/USDT (exchange) account (D25).
func isCryptoAccount(account Account) bool {
	return account.Type == AccountTypeCrypto || account.Currency == CurrencyUSD || account.Currency == CurrencyUSDT
}

// ComputeAllocation builds the allocation with targets and drift.
func ComputeAllocation(in AllocationInputs) Allocation {
	var cash, cryptoAccounts float64
	for _, account := range in.Accounts {
		if account.Type == AccountTypeDeprecated || account.Balance <= 0 {
			continue
		}
		if isCryptoAccount(account) {
			cryptoAccounts += account.Balance
		} else {
			cash += account.Balance
		}
	}
	receivables, _ := lendingPositions(in.Transactions)
	values := map[AllocationClass]float64{
		AllocationCash:      cash,
		AllocationFunds:     math.Max(in.Funds, 0),
		AllocationCrypto:    math.Max(math.Max(cryptoAccounts, in.CryptoHoldings), 0),
		AllocationMoneyLent: receivables,
	}

	result := Allocation{
		Lines: []AllocationLine{}, Targets: in.Targets, DriftThreshold: DriftThreshold,
		Unvalued: in.Unvalued, Issues: append([]string{}, in.Issues...), GeneratedAt: in.Now,
	}
	for _, class := range TargetClasses {
		result.TargetedTotal += values[class]
	}
	result.Total = result.TargetedTotal + values[AllocationMoneyLent]

	for _, class := range append(append([]AllocationClass{}, TargetClasses...), AllocationMoneyLent) {
		line := AllocationLine{Class: class, Label: allocationLabels[class], Value: values[class], Status: AllocationNoTarget}
		if result.Total > 0 {
			line.Share = round2(line.Value / result.Total * 100)
		}
		if class != AllocationMoneyLent {
			line.Targeted = true
			current := 0.0
			if result.TargetedTotal > 0 {
				current = line.Value / result.TargetedTotal * 100
			}
			line.Current = percentPointer(round2(current))
			if in.Targets != nil {
				target := in.Targets.For(class)
				// Rounded first, so float noise cannot tip a class at
				// exactly the threshold over it.
				drift := round2(current - target)
				line.Target = percentPointer(target)
				line.Drift = percentPointer(drift)
				line.Status = driftStatus(drift)
				if result.TargetedTotal > 0 && math.Abs(drift) > DriftThreshold {
					result.Rebalance = true
				}
			}
		}
		result.Lines = append(result.Lines, line)
	}
	if result.TargetedTotal <= 0 {
		result.Rebalance = false
	}
	return result
}

func driftStatus(drift float64) AllocationStatus {
	switch {
	case drift > DriftThreshold:
		return AllocationOver
	case drift < -DriftThreshold:
		return AllocationUnder
	default:
		return AllocationOnTarget
	}
}

func percentPointer(value float64) *float64 { return &value }

// liquidBalance is the cash that can be spent now: positive balances of
// active, rarely used, bank and cash accounts. It backs the emergency fund
// (WM-HEA-2) and the budget projection; investment and long holding accounts
// are not liquid.
func liquidBalance(accounts []Account) float64 {
	total := 0.0
	for _, account := range accounts {
		switch account.Type {
		case AccountTypeActiveUse, AccountTypeRarelyUse, AccountTypeBank, AccountTypeCash:
			if account.Balance > 0 {
				total += account.Balance
			}
		}
	}
	return total
}

// diversificationScore returns the HHI over the target classes and its
// score: an even split scores full, a single class scores zero.
func diversificationScore(allocation Allocation) (float64, float64) {
	if allocation.TargetedTotal <= 0 {
		return 1, 0
	}
	hhi := 0.0
	for _, class := range TargetClasses {
		share := allocation.Line(class).Value / allocation.TargetedTotal
		hhi += share * share
	}
	minimum := 1 / float64(len(TargetClasses))
	return hhi, healthDiversificationMax * clampUnit((1-hhi)/(1-minimum))
}

func allocationDetail(allocation Allocation) string {
	detail := ""
	for _, class := range TargetClasses {
		line := allocation.Line(class)
		current := 0.0
		if line.Current != nil {
			current = *line.Current
		}
		if detail != "" {
			detail += ", "
		}
		detail += fmt.Sprintf("%s %.1f%%", line.Label, current)
	}
	return detail
}
