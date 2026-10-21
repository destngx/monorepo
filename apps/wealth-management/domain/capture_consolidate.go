package domain

import (
	"math"
	"sort"
	"time"
)

// CaptureEvidence ties a suggestion to the material it was read from.
type CaptureEvidence struct {
	MaterialID string `json:"materialId"`
	Quote      string `json:"quote,omitempty"`
}

// CaptureItem is an event with its account and evidence, possibly merged
// from several materials.
type CaptureItem struct {
	Event    CaptureEvent
	Account  AccountMatch
	Evidence []CaptureEvidence
}

func NewCaptureItem(event CaptureEvent, account AccountMatch) CaptureItem {
	return CaptureItem{Event: event, Account: account, Evidence: []CaptureEvidence{{MaterialID: event.MaterialID, Quote: event.Quote}}}
}

const (
	captureSameEventWindow = 15 * time.Minute
	captureTransferWindow  = 60 * time.Minute
	captureAmountTolerance = 0.005
)

// ConsolidateCaptureItems merges items from different materials that
// describe the same money movement, e.g. a bank SMS and an app screenshot
// of one transfer (WM-CAP-3). Items from the same material are never
// merged: a statement may list two identical purchases.
func ConsolidateCaptureItems(items []CaptureItem) []CaptureItem {
	out := make([]CaptureItem, 0, len(items))
	for _, item := range items {
		merged := false
		for index := range out {
			if sameMovement(out[index], item) {
				out[index] = mergeItems(out[index], item)
				merged = true
				break
			}
		}
		if !merged {
			out = append(out, item)
		}
	}
	return out
}

func sameMovement(a, b CaptureItem) bool {
	for _, left := range a.Evidence {
		for _, right := range b.Evidence {
			if left.MaterialID == right.MaterialID {
				return false
			}
		}
	}
	x, y := a.Event, b.Event
	if x.Direction != y.Direction || x.Currency != y.Currency || math.Abs(x.Amount-y.Amount) > captureAmountTolerance {
		return false
	}
	if a.Account.Name != "" && b.Account.Name != "" && a.Account.Name != b.Account.Name {
		return false
	}
	if x.Reference != "" && y.Reference != "" && FoldText(x.Reference) != FoldText(y.Reference) {
		return false
	}
	return closeInTime(x, y, captureSameEventWindow, 0)
}

// closeInTime compares times when both events have one, else calendar days.
func closeInTime(x, y CaptureEvent, window time.Duration, days int) bool {
	if x.HasTime && y.HasTime {
		return absDuration(x.At.Sub(y.At)) <= window
	}
	return dayDistance(x.Date(), y.Date()) <= days
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func dayDistance(a, b time.Time) int {
	days := int(math.Round(a.Sub(b).Hours() / 24))
	if days < 0 {
		return -days
	}
	return days
}

// mergeItems keeps the more confident event and fills its blanks from the other.
func mergeItems(a, b CaptureItem) CaptureItem {
	base, other := a, b
	if b.Event.Confidence > a.Event.Confidence {
		base, other = b, a
	}
	event := base.Event
	fill := func(target *string, value string) {
		if *target == "" {
			*target = value
		}
	}
	fill(&event.Counterparty, other.Event.Counterparty)
	fill(&event.Description, other.Event.Description)
	fill(&event.Reference, other.Event.Reference)
	fill(&event.SourceApp, other.Event.SourceApp)
	fill(&event.AccountHint, other.Event.AccountHint)
	fill(&event.Category, other.Event.Category)
	if len(event.ScreenCues) == 0 {
		event.ScreenCues = other.Event.ScreenCues
	}
	if event.BalanceAfter == nil {
		event.BalanceAfter = other.Event.BalanceAfter
	}
	if event.Fee == nil {
		event.Fee = other.Event.Fee
	}
	if !event.HasTime && other.Event.HasTime {
		event.At, event.HasTime = other.Event.At, true
	}
	event.DateAssumed = event.DateAssumed && other.Event.DateAssumed
	account := base.Account
	if account.Name == "" && other.Account.Name != "" {
		account = other.Account
	}
	evidence := append(append([]CaptureEvidence{}, base.Evidence...), other.Evidence...)
	return CaptureItem{Event: event, Account: account, Evidence: evidence}
}

// PairCaptureTransfers finds own-account transfers: money out of account
// A and into account B, same amount and currency, close in time (WM-CAP-3).
// It returns (out index, in index) pairs, closest first, each item used once.
func PairCaptureTransfers(items []CaptureItem) [][2]int {
	type candidate struct {
		out, in  int
		distance time.Duration
	}
	var candidates []candidate
	for o, out := range items {
		if out.Event.Direction != CaptureOut || out.Account.Name == "" {
			continue
		}
		for i, in := range items {
			if in.Event.Direction != CaptureIn || in.Account.Name == "" || in.Account.Name == out.Account.Name {
				continue
			}
			if in.Event.Currency != out.Event.Currency || math.Abs(in.Event.Amount-out.Event.Amount) > captureAmountTolerance {
				continue
			}
			if !closeInTime(out.Event, in.Event, captureTransferWindow, 1) {
				continue
			}
			candidates = append(candidates, candidate{out: o, in: i, distance: absDuration(out.Event.At.Sub(in.Event.At))})
		}
	}
	sort.SliceStable(candidates, func(a, b int) bool { return candidates[a].distance < candidates[b].distance })
	used := map[int]bool{}
	var pairs [][2]int
	for _, c := range candidates {
		if used[c.out] || used[c.in] {
			continue
		}
		used[c.out], used[c.in] = true, true
		pairs = append(pairs, [2]int{c.out, c.in})
	}
	return pairs
}
