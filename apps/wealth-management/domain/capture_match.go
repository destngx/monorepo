package domain

import (
	"regexp"
	"strings"
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

// Account match sources, strongest first.
const (
	MatchSourceAlias  = "alias"
	MatchSourceDigits = "digits"
	MatchSourceName   = "name"
	// MatchSourceLedger is an account filled from the sheet row a capture
	// with no account likely duplicates (WM-CAP-6).
	MatchSourceLedger = "ledger"
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

// CaptureCard is an open card the user holds: the account it bills to, the
// keyword tag that files a row to it and its last four digits, if known.
type CaptureCard struct {
	Name    string
	Account string
	Keyword string
	Last4   string
}

// MatchCaptureCard finds the one card whose last four digits end a digit
// run in the hint ("Acme *0123"). Two cards sharing the digits match none.
func MatchCaptureCard(hint string, cards []CaptureCard) (CaptureCard, bool) {
	wanted := map[string]bool{}
	for _, run := range digitRunPattern.FindAllString(hint, -1) {
		wanted[run[len(run)-4:]] = true
	}
	var found []CaptureCard
	for _, card := range cards {
		if card.Last4 != "" && wanted[card.Last4] {
			found = append(found, card)
		}
	}
	if len(found) != 1 {
		return CaptureCard{}, false
	}
	return found[0], true
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
