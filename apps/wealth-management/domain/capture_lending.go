package domain

import (
	"sort"
	"strings"
)

// Capture lending (WM-CAP-22). A screenshot names the person as the bank
// prints it ("LE THU NGHIEM"), the ledger as the user typed it ("Lê Thử
// Nghiệm"), so payee history never meets money lent. Before categorising,
// each payee is matched to the people in money lent: an accent- and
// case-blind match renames it to the ledger spelling, a near one is offered
// and flagged. Lending words in the user's note ("lend", "cho vay") file
// the row under the lending category and confirm a near match.

// CaptureLendingInput is everything ApplyCaptureLending needs.
type CaptureLendingInput struct {
	Transactions []Transaction
	// Categories are the lending-nature categories; the first is the default.
	Categories []string
	// Notes are typed notes that read as no transaction, so they are context
	// for the screenshot's rows.
	Notes []string
}

// lendingPerson is one person in money lent, by the latest spelling.
type lendingPerson struct {
	name     string
	folded   string
	category string
}

// Lending payee match strengths.
const (
	lendingMatchNone = iota
	lendingMatchNear
	lendingMatchSame
)

// lendingMinNameRunes keeps short names out of near matching: "An" is in
// too many payees.
const lendingMinNameRunes = 6

// lendingIntentPhrases are folded word sequences that mark money lent or
// repaid. Single Vietnamese words such as "vay" and "muon" are left out:
// without accents they also mean "skirt" and "want".
var lendingIntentPhrases = []string{
	"lend", "lent", "lending", "loan", "loaned", "borrow", "borrowed", "repay", "repaid", "repayment",
	"cho vay", "cho muon", "vay tien", "muon tien", "tra no",
}

// HasLendingIntent reports whether the text says the money was lent,
// borrowed or repaid, ignoring case and Vietnamese accents.
func HasLendingIntent(text string) bool {
	folded := FoldText(text)
	for _, phrase := range lendingIntentPhrases {
		if containsWords(folded, phrase) {
			return true
		}
	}
	return false
}

// ApplyCaptureLending matches each suggestion's payee to money lent and
// files rows the user called lending under the lending category. Own
// transfers, fee rows and rows already categorised are left alone.
func ApplyCaptureLending(suggestions []CaptureSuggestion, input CaptureLendingInput) {
	people := lendingPeople(input.Transactions, input.Categories)
	noteIntent := false
	for _, note := range input.Notes {
		noteIntent = noteIntent || HasLendingIntent(note)
	}
	// A note sent with several rows says which is lending only through the
	// payee, so it applies alone to a single row.
	open := 0
	for _, suggestion := range suggestions {
		if suggestion.TransferGroup == "" && !suggestion.Fee {
			open++
		}
	}
	for index := range suggestions {
		suggestion := &suggestions[index]
		if suggestion.TransferGroup != "" || suggestion.Fee || suggestion.Transaction.Category != "" {
			continue
		}
		description := ""
		if suggestion.Transaction.Memo != nil {
			description = *suggestion.Transaction.Memo
		}
		ownIntent := HasLendingIntent(suggestion.CategoryHint) || HasLendingIntent(description)
		person, strength := matchLendingPerson(suggestion.Transaction.Payee, people)
		intent := ownIntent || (noteIntent && (open == 1 || strength != lendingMatchNone))

		read := suggestion.Transaction.Payee
		switch {
		case strength == lendingMatchSame:
			suggestion.Transaction.Payee = person.name
			if !strings.EqualFold(strings.TrimSpace(read), person.name) {
				suggestion.Notes = append(suggestion.Notes, "Payee “"+read+"” matched “"+person.name+"” in money lent.")
			}
		case strength == lendingMatchNear && intent:
			suggestion.Transaction.Payee = person.name
			suggestion.Notes = append(suggestion.Notes, "Payee “"+read+"” is close to “"+person.name+"” in money lent; check the name.")
		case strength == lendingMatchNear:
			suggestion.Transaction.Payee = person.name
			suggestion.LowConfidence = appendMissing(suggestion.LowConfidence, CaptureFieldPayee)
			suggestion.Notes = append(suggestion.Notes, "Payee “"+read+"” looks like “"+person.name+"” in money lent; check the name.")
		}
		if !intent || len(input.Categories) == 0 {
			continue
		}
		category := input.Categories[0]
		if strength != lendingMatchNone && person.category != "" {
			category = person.category
		}
		suggestion.Transaction.Category, suggestion.CategorySource = category, CategorySourceManual
		if strength == lendingMatchNone {
			suggestion.LowConfidence = appendMissing(suggestion.LowConfidence, CaptureFieldPayee)
			suggestion.Notes = append(suggestion.Notes, "Noted as lending, but no one in money lent matches “"+read+"”; check the name.")
		}
	}
}

// CaptureFieldPayee flags the payee on the review screen.
const CaptureFieldPayee = "payee"

// lendingPeople lists the people in money lent with their latest spelling
// and latest lending category, newest activity first so ties go to them.
func lendingPeople(transactions []Transaction, categories []string) []lendingPerson {
	people := make([]lendingPerson, 0)
	latest := map[string]Transaction{}
	for key, group := range lendingGroups(transactions) {
		if key == "" {
			continue
		}
		newest := group[0]
		for _, t := range group[1:] {
			if t.Date.After(newest.Date) {
				newest = t
			}
		}
		latest[key] = newest
	}
	for _, t := range latest {
		name := strings.TrimSpace(t.Payee)
		category, _ := ResolveCategory(t.Category, categories)
		people = append(people, lendingPerson{name: name, folded: FoldText(name), category: category})
	}
	sort.Slice(people, func(a, b int) bool {
		left, right := latest[lendingKey(people[a].name)], latest[lendingKey(people[b].name)]
		if !left.Date.Equal(right.Date) {
			return left.Date.After(right.Date)
		}
		return people[a].name < people[b].name
	})
	return people
}

// matchLendingPerson finds the person a payee names. Same: equal ignoring
// case, accents and word order. Near: the person's full name inside a
// longer payee, or a spelling or two off; only when exactly one person is
// that near.
func matchLendingPerson(payee string, people []lendingPerson) (lendingPerson, int) {
	folded := FoldText(payee)
	if folded == "" {
		return lendingPerson{}, lendingMatchNone
	}
	for _, person := range people {
		if person.folded == folded || sameWords(person.folded, folded) {
			return person, lendingMatchSame
		}
	}
	var near []lendingPerson
	for _, person := range people {
		if isNearName(person.folded, folded) {
			near = append(near, person)
		}
	}
	if len(near) == 1 {
		return near[0], lendingMatchNear
	}
	return lendingPerson{}, lendingMatchNone
}

func sameWords(a, b string) bool {
	left, right := strings.Fields(a), strings.Fields(b)
	if len(left) != len(right) {
		return false
	}
	sort.Strings(left)
	sort.Strings(right)
	return strings.Join(left, " ") == strings.Join(right, " ")
}

// isNearName: a multi-word name contained word for word in the payee
// ("chuyen tien le thu nghiem"), or the whole names within one edit per six
// letters ("le thu nghiem" against "le thu ngheim" or a cut-off "le thu nghie").
func isNearName(person, payee string) bool {
	if len([]rune(person)) < lendingMinNameRunes {
		return false
	}
	if strings.Contains(person, " ") && containsWords(payee, person) {
		return true
	}
	allowed := len([]rune(person)) / lendingMinNameRunes
	return editDistance(person, payee) <= allowed
}

// editDistance is the Levenshtein distance in runes.
func editDistance(a, b string) int {
	left, right := []rune(a), []rune(b)
	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(left); i++ {
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(right)]
}

func appendMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
