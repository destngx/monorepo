package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

// CaptureCandidate is what dedup compares against the ledger.
type CaptureCandidate struct {
	Account   string
	Direction CaptureDirection
	Amount    float64
	Date      time.Time
	Reference string
}

// FindCaptureDuplicates marks candidates likely already recorded: same
// account, same amount, date within ±2 days and no conflicting reference
// (WM-CAP-6). Each ledger row explains at most one candidate; a matching
// reference beats a closer date. The result maps candidate index to row.
func FindCaptureDuplicates(candidates []CaptureCandidate, ledger []Transaction) map[int]Transaction {
	type match struct {
		candidate, row int
		score          int
	}
	var matches []match
	for c, candidate := range candidates {
		if candidate.Account == "" {
			continue
		}
		for r, row := range ledger {
			if !row.Usable() || row.AccountName != candidate.Account {
				continue
			}
			amount := row.PaymentAmount()
			if candidate.Direction == CaptureIn {
				amount = row.DepositAmount()
			}
			if math.Abs(amount-candidate.Amount) > captureAmountTolerance {
				continue
			}
			days := dayDistance(row.Date, candidate.Date)
			if days > CaptureDuplicateDays {
				continue
			}
			score := days
			if reference := referenceOf(row); reference != "" && candidate.Reference != "" {
				if FoldText(reference) != FoldText(candidate.Reference) {
					continue
				}
				score = -1
			}
			matches = append(matches, match{candidate: c, row: r, score: score})
		}
	}
	sort.SliceStable(matches, func(a, b int) bool { return matches[a].score < matches[b].score })
	result := map[int]Transaction{}
	usedRows := map[int]bool{}
	for _, m := range matches {
		if _, done := result[m.candidate]; done || usedRows[m.row] {
			continue
		}
		usedRows[m.row] = true
		result[m.candidate] = ledger[m.row]
	}
	return result
}

func referenceOf(t Transaction) string {
	if t.ReferenceNumber == nil {
		return ""
	}
	return strings.TrimSpace(*t.ReferenceNumber)
}
