package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	// CaptureDuplicateDays is the ledger dedup window (WM-CAP-6): a week,
	// since rows are sometimes entered once a week with the entry day.
	CaptureDuplicateDays = 7
	// CaptureLikelyDuplicateScore is the score a match must beat to count as
	// likely recorded and be unselected.
	CaptureLikelyDuplicateScore = 80

	captureReferenceScore  = 100
	captureSameDayScore    = 90
	captureScorePerDay     = 5
	captureTextScore       = 30
	captureMaxScoreWithout = 99
	captureMinTokenLength  = 2
	captureMinPrefixLength = 3
)

// Words that say how money moved rather than to whom, plus the words
// Smart Capture puts in its memos. They never make two rows look alike.
var captureGenericWords = map[string]bool{
	"chuyen": true, "tien": true, "thanh": true, "toan": true, "den": true, "cho": true, "tu": true,
	"ck": true, "ct": true, "tt": true, "nap": true, "rut": true, "giao": true, "dich": true, "qua": true, "va": true,
	"transfer": true, "payment": true, "pay": true, "paid": true, "to": true, "from": true, "for": true,
	"the": true, "and": true, "capture": true, "screenshot": true, "sms": true, "text": true, "note": true,
}

// CaptureCandidate is what dedup compares against the ledger.
type CaptureCandidate struct {
	Account   string
	Direction CaptureDirection
	Amount    float64
	Date      time.Time
	Reference string
	// Text is the payee, counterparty and memo, compared with the row's
	// payee and memo.
	Text string
	// SourceApp is the app the material came from; like the account name,
	// its words say nothing about the payee.
	SourceApp string
}

// CaptureMatch is the ledger row that likely explains a candidate, with how
// sure the match is (0-100) and how many days apart the two are.
type CaptureMatch struct {
	Row   Transaction
	Score int
	Days  int
}

// FindCaptureDuplicates matches candidates to ledger rows on the same
// account with the same amount and direction, dated up to a week apart,
// with no conflicting reference (WM-CAP-6). A shared reference scores 100.
// Otherwise the score starts at 90 on the same day, loses 5 a day, and
// gains up to 30 for payee and memo words in common, capped at 99. Each
// row explains at most one candidate; the strongest pairs are taken first.
// The result maps candidate index to its match.
func FindCaptureDuplicates(candidates []CaptureCandidate, ledger []Transaction) map[int]CaptureMatch {
	type pair struct {
		candidate, row int
		match          CaptureMatch
	}
	var pairs []pair
	for c, candidate := range candidates {
		if candidate.Account == "" {
			continue
		}
		ignored := candidate.Account + " " + candidate.SourceApp
		words := captureWords(candidate.Text, ignored)
		for r, row := range ledger {
			if match, ok := scoreCaptureMatch(candidate, words, ignored, row); ok {
				pairs = append(pairs, pair{candidate: c, row: r, match: match})
			}
		}
	}
	sort.SliceStable(pairs, func(a, b int) bool {
		if pairs[a].match.Score != pairs[b].match.Score {
			return pairs[a].match.Score > pairs[b].match.Score
		}
		return pairs[a].match.Days < pairs[b].match.Days
	})
	result := map[int]CaptureMatch{}
	usedRows := map[int]bool{}
	for _, p := range pairs {
		if _, done := result[p.candidate]; done || usedRows[p.row] {
			continue
		}
		usedRows[p.row] = true
		result[p.candidate] = p.match
	}
	return result
}

func scoreCaptureMatch(candidate CaptureCandidate, words []string, ignored string, row Transaction) (CaptureMatch, bool) {
	if !row.Usable() || row.AccountName != candidate.Account {
		return CaptureMatch{}, false
	}
	amount := row.PaymentAmount()
	if candidate.Direction == CaptureIn {
		amount = row.DepositAmount()
	}
	if math.Abs(amount-candidate.Amount) > captureAmountTolerance {
		return CaptureMatch{}, false
	}
	days := dayDistance(row.Date, candidate.Date)
	if days > CaptureDuplicateDays {
		return CaptureMatch{}, false
	}
	match := CaptureMatch{Row: row, Days: days}
	if reference := referenceOf(row); reference != "" && candidate.Reference != "" {
		if FoldText(reference) != FoldText(candidate.Reference) {
			return CaptureMatch{}, false
		}
		match.Score = captureReferenceScore
		return match, true
	}
	memo := ""
	if row.Memo != nil {
		memo = *row.Memo
	}
	similarity := wordSimilarity(words, captureWords(row.Payee+" "+memo, ignored))
	score := captureSameDayScore - captureScorePerDay*days + int(math.Round(captureTextScore*similarity))
	match.Score = min(score, captureMaxScoreWithout)
	return match, true
}

// captureWords are the folded words of a payee or memo that can tell two
// rows apart: no generic transfer words, no ignored words (the account and
// source app names) and no single letters.
func captureWords(text, ignored string) []string {
	ignoredWords := map[string]bool{}
	for _, word := range strings.Fields(FoldText(ignored)) {
		ignoredWords[word] = true
	}
	var words []string
	for _, word := range strings.Fields(FoldText(text)) {
		if len(word) >= captureMinTokenLength && !captureGenericWords[word] && !ignoredWords[word] {
			words = append(words, word)
		}
	}
	return uniqueStrings(words)
}

// wordSimilarity is the share of the shorter word list found in the other.
// A word matches the same word or, for apps that cut payees short
// ("Highla..."), a word it starts.
func wordSimilarity(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	found := 0
	for _, left := range a {
		for _, right := range b {
			if sameOrPrefix(left, right) {
				found++
				break
			}
		}
	}
	return float64(found) / float64(len(a))
}

func sameOrPrefix(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	return len(a) >= captureMinPrefixLength && strings.HasPrefix(b, a)
}

// LastRecordedDates maps each account to its newest usable ledger date on or
// before today. Rows dated later (entered ahead, like a planned bill) are
// ignored so they do not hide new rows.
func LastRecordedDates(ledger []Transaction, today time.Time) map[string]time.Time {
	last := map[string]time.Time{}
	for _, row := range ledger {
		if !row.Usable() || row.AccountName == "" || row.Date.After(today) {
			continue
		}
		if current, seen := last[row.AccountName]; !seen || row.Date.After(current) {
			last[row.AccountName] = row.Date
		}
	}
	return last
}

func referenceOf(t Transaction) string {
	if t.ReferenceNumber == nil {
		return ""
	}
	return strings.TrimSpace(*t.ReferenceNumber)
}
