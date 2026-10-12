package domain

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Deterministic Smart Capture steps (WM-CAP-3, 4, 6, 11).

// FoldText lowercases, strips Vietnamese diacritics (đ included) and
// collapses everything but letters and digits to single spaces, so
// "Ví MoMo" and "vi momo" compare equal.
func FoldText(value string) string {
	stripped, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), value)
	if err != nil {
		stripped = value
	}
	stripped = strings.NewReplacer("đ", "d", "Đ", "d").Replace(strings.ToLower(stripped))
	return strings.Join(strings.FieldsFunc(stripped, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

type CaptureRuleKind string

const (
	// CaptureRuleAlias maps an account hint (Pattern) to an account name (Value).
	CaptureRuleAlias CaptureRuleKind = "alias"
	// CaptureRulePayee maps a payee (Pattern) to a category (Value).
	CaptureRulePayee CaptureRuleKind = "payee"
)

// CaptureRule is a correction confirmed during review (WM-CAP-11).
type CaptureRule struct {
	Kind      CaptureRuleKind `json:"kind"`
	Pattern   string          `json:"pattern"`
	Value     string          `json:"value"`
	UpdatedAt string          `json:"updatedAt,omitempty"`
}

// Key identifies a rule for upserts: kind plus folded pattern.
func (r CaptureRule) Key() string {
	return string(r.Kind) + ":" + FoldText(r.Pattern)
}

// CaptureRulesSaveResult counts what a rules upsert changed.
type CaptureRulesSaveResult struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

func (r CaptureRule) Validate() error {
	var problems []string
	if r.Kind != CaptureRuleAlias && r.Kind != CaptureRulePayee {
		problems = append(problems, "kind must be alias or payee")
	}
	if FoldText(r.Pattern) == "" {
		problems = append(problems, "pattern is required")
	}
	if strings.TrimSpace(r.Value) == "" {
		problems = append(problems, "value is required")
	}
	return validationError(problems)
}

// Account match sources, strongest first.
const (
	MatchSourceAlias  = "alias"
	MatchSourceDigits = "digits"
	MatchSourceName   = "name"
)

// AccountMatch is the sheet account for an event, or the candidates when
// the hint is ambiguous ("ask the user", WM-CAP-4).
type AccountMatch struct {
	Name       string   `json:"name"`
	Source     string   `json:"source,omitempty"`
	Candidates []string `json:"candidates,omitempty"`
}

var (
	digitRunPattern = regexp.MustCompile(`\d{4,}`)
	// Words that say what kind of account it is rather than which one.
	genericAccountWords = map[string]bool{
		"bank": true, "ngan": true, "hang": true, "card": true, "credit": true, "debit": true, "the": true,
		"tai": true, "khoan": true, "tk": true, "vi": true, "wallet": true, "account": true, "app": true,
		"visa": true, "master": true, "mastercard": true, "tin": true, "dung": true, "thanh": true, "toan": true,
		"so": true, "du": true, "ck": true, "of": true, "my": true, "cash": true,
	}
	cardWords = []string{"card", "credit", "the", "visa", "master", "mastercard", "tin dung"}
)

// MatchCaptureAccount maps an event's account hint to a sheet account:
// confirmed aliases first, then the last four digits, then distinctive
// name words (WM-CAP-4). Deprecated accounts are never matched.
func MatchCaptureAccount(hint string, sourceApp string, accounts []Account, rules []CaptureRule) AccountMatch {
	active := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Type != AccountTypeDeprecated {
			active = append(active, account)
		}
	}
	text := FoldText(hint + " " + sourceApp)
	if text == "" {
		return AccountMatch{}
	}
	if name, ok := matchAlias(FoldText(hint), active, rules); ok {
		return AccountMatch{Name: name, Source: MatchSourceAlias}
	}
	if name, ok := matchAlias(FoldText(sourceApp), active, rules); ok {
		return AccountMatch{Name: name, Source: MatchSourceAlias}
	}
	if name, ok := matchDigits(hint, active); ok {
		return AccountMatch{Name: name, Source: MatchSourceDigits}
	}
	return matchName(text, active)
}

// matchAlias picks the longest alias pattern contained in the hint whose
// account still exists.
func matchAlias(hint string, accounts []Account, rules []CaptureRule) (string, bool) {
	if hint == "" {
		return "", false
	}
	best, bestLength := "", 0
	padded := " " + hint + " "
	for _, rule := range rules {
		pattern := FoldText(rule.Pattern)
		if rule.Kind != CaptureRuleAlias || pattern == "" || !strings.Contains(padded, " "+pattern+" ") || len(pattern) <= bestLength {
			continue
		}
		if name, ok := accountNamed(rule.Value, accounts); ok {
			best, bestLength = name, len(pattern)
		}
	}
	return best, best != ""
}

func accountNamed(name string, accounts []Account) (string, bool) {
	for _, account := range accounts {
		if strings.EqualFold(strings.TrimSpace(account.Name), strings.TrimSpace(name)) {
			return account.Name, true
		}
	}
	return "", false
}

// matchDigits matches a unique account whose name or note carries the
// same last four digits as the hint.
func matchDigits(hint string, accounts []Account) (string, bool) {
	wanted := map[string]bool{}
	for _, run := range digitRunPattern.FindAllString(hint, -1) {
		wanted[run[len(run)-4:]] = true
	}
	if len(wanted) == 0 {
		return "", false
	}
	found := ""
	for _, account := range accounts {
		text := account.Name
		if account.Note != nil {
			text += " " + *account.Note
		}
		for _, run := range digitRunPattern.FindAllString(text, -1) {
			if wanted[run[len(run)-4:]] {
				if found != "" && found != account.Name {
					return "", false
				}
				found = account.Name
			}
		}
	}
	return found, found != ""
}

// matchName scores accounts by distinctive name words found in the hint.
// A tie is broken towards credit cards when the hint talks about a card;
// a remaining tie is returned as candidates.
func matchName(text string, accounts []Account) AccountMatch {
	words := map[string]bool{}
	for _, word := range strings.Fields(text) {
		words[word] = true
	}
	scores := map[string]int{}
	best := 0
	for _, account := range accounts {
		score := 0
		for _, word := range strings.Fields(FoldText(account.Name)) {
			if !genericAccountWords[word] && len(word) > 1 && words[word] {
				score++
			}
		}
		if score > 0 {
			scores[account.Name] = score
			best = max(best, score)
		}
	}
	if best == 0 {
		return AccountMatch{}
	}
	var top []Account
	for _, account := range accounts {
		if scores[account.Name] == best {
			top = append(top, account)
		}
	}
	if len(top) > 1 {
		mentionsCard := false
		padded := " " + text + " "
		for _, word := range cardWords {
			mentionsCard = mentionsCard || strings.Contains(padded, " "+word+" ")
		}
		var preferred []Account
		for _, account := range top {
			if (account.Type == AccountTypeNegativeActive) == mentionsCard {
				preferred = append(preferred, account)
			}
		}
		if len(preferred) > 0 {
			top = preferred
		}
	}
	if len(top) == 1 {
		return AccountMatch{Name: top[0].Name, Source: MatchSourceName}
	}
	candidates := make([]string, len(top))
	for index, account := range top {
		candidates[index] = account.Name
	}
	return AccountMatch{Candidates: candidates}
}

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
	// CaptureDuplicateDays is the ledger dedup window (WM-CAP-6).
	CaptureDuplicateDays = 2
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
	if event.BalanceAfter == nil {
		event.BalanceAfter = other.Event.BalanceAfter
	}
	if event.Fee == nil {
		event.Fee = other.Event.Fee
	}
	if !event.HasTime && other.Event.HasTime {
		event.At, event.HasTime = other.Event.At, true
	}
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
