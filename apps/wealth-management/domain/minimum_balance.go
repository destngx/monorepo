package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// Minimum balance to keep (D35, WM-SET-11, WM-BILL-9..10): a bank may
// grant a monthly reward (points, a fee waiver) when an account's average
// daily balance over the calendar month stays at or above a minimum. The
// user saves that minimum per account in accounts.minimumBalance; the
// engine treats it as a hard reserve (never ready to spend, never idle)
// and tracks the month-to-date average against it.
//
// Why a hard reserve rather than "dip below now, top up later": the reward
// depends on the month's average, so money spent below the minimum late in
// the month cannot be made up in the days left, and a plain reserve errs
// on the safe side while staying simple to explain.

const (
	// maxAccountMinimum bounds a minimum (VND), like the day-to-day float.
	maxAccountMinimum = maxDayToDayFloat
	// maxAccountMinimumNote bounds the free-text note.
	maxAccountMinimumNote = 200
	// MinimumBalanceLastWeekDays: with this many days left (today
	// included), an account at risk is serious.
	MinimumBalanceLastWeekDays = 7
)

// MinimumBalanceStatus says whether an account will reach its minimum
// average this month if its balance stays as it is today.
type MinimumBalanceStatus string

const (
	MinimumOnTrack MinimumBalanceStatus = "on_track"
	MinimumAtRisk  MinimumBalanceStatus = "at_risk"
)

// MinimumBalanceRule documents the tracking for clients.
const MinimumBalanceRule = "The month-to-date average daily balance rebuilds each day's end-of-day balance backwards " +
	"from today's balance with the account's own transactions (booking dates; a transfer is a payment on one account " +
	"and a deposit on the other). The days counted so far are the 1st to yesterday; today and the rest of the month " +
	"are still open. projectedAverage = (sumSoFar + balance x daysLeft) / daysInMonth, if the balance stays as it is " +
	"today. requiredBalance = (minimum x daysInMonth - sumSoFar) / daysLeft, the balance to hold from today to the " +
	"month end (0 when already covered); extraNeeded = requiredBalance - balance when positive (status at_risk)."

// AccountMinimumBalance is one account's minimum and how this month's
// average stands against it.
type AccountMinimumBalance struct {
	Account string  `json:"account"`
	Minimum float64 `json:"minimum"`
	Note    string  `json:"note,omitempty"`
	Balance float64 `json:"balance"`
	// Reserved is the part of the balance kept: min(balance, minimum), 0
	// when the balance is not positive.
	Reserved float64 `json:"reserved"`
	// Month is yyyy-mm; MonthEnd its last day, yyyy-mm-dd.
	Month       string `json:"month"`
	MonthEnd    string `json:"monthEnd"`
	DaysInMonth int    `json:"daysInMonth"`
	// DaysCounted are the finished days (1st to yesterday); DaysLeft the
	// rest, today included (at least 1).
	DaysCounted int     `json:"daysCounted"`
	DaysLeft    int     `json:"daysLeft"`
	SumSoFar    float64 `json:"sumSoFar"`
	// AverageSoFar averages the finished days; on the 1st it is today's balance.
	AverageSoFar     float64              `json:"averageSoFar"`
	ProjectedAverage float64              `json:"projectedAverage"`
	RequiredBalance  float64              `json:"requiredBalance"`
	ExtraNeeded      float64              `json:"extraNeeded"`
	Status           MinimumBalanceStatus `json:"status"`
}

// TrackMinimumBalance computes the account's month-to-date average daily
// balance and what it needs for the rest of the month (MinimumBalanceRule).
// Only the account's own usable transactions count; a day before an
// account opened rebuilds to what it held then (0 for an opening deposit).
func TrackMinimumBalance(account Account, minimum AccountMinimum, transactions []Transaction, today time.Time) AccountMinimumBalance {
	today = dateOnly(today)
	daysInMonth := DaysInMonth(today)
	counted := today.Day() - 1
	left := daysInMonth - counted
	monthEnd := time.Date(today.Year(), today.Month(), daysInMonth, 0, 0, 0, 0, time.UTC)
	track := AccountMinimumBalance{
		Account: account.Name, Minimum: minimum.Minimum, Note: minimum.Note, Balance: account.Balance,
		Reserved: math.Min(math.Max(0, account.Balance), minimum.Minimum),
		Month:    MonthKey(today), MonthEnd: monthEnd.Format(time.DateOnly), DaysInMonth: daysInMonth,
		DaysCounted: counted, DaysLeft: left, AverageSoFar: account.Balance,
	}
	own := []Transaction{}
	for _, t := range transactions {
		if sameAccountName(t.AccountName, account.Name) {
			own = append(own, dayOnlyTransaction(t))
		}
	}
	// Points run from the 1st to today; today's point is the balance now.
	points := BackCalculateSeries(account.Balance, own, today, today.Day())
	for _, point := range points[:counted] {
		track.SumSoFar += point.Value
	}
	track.SumSoFar = math.Round(track.SumSoFar)
	if counted > 0 {
		track.AverageSoFar = math.Round(track.SumSoFar / float64(counted))
	}
	track.ProjectedAverage = math.Round((track.SumSoFar + account.Balance*float64(left)) / float64(daysInMonth))
	track.RequiredBalance = math.Max(0, math.Ceil((minimum.Minimum*float64(daysInMonth)-track.SumSoFar)/float64(left)))
	track.ExtraNeeded = math.Max(0, track.RequiredBalance-math.Floor(account.Balance))
	track.Status = MinimumOnTrack
	if track.ExtraNeeded > 0 {
		track.Status = MinimumAtRisk
	}
	return track
}

// dayOnlyTransaction drops a time of day, so a flow lands on its date.
func dayOnlyTransaction(t Transaction) Transaction {
	if !t.Date.IsZero() {
		t.Date = dateOnly(t.Date)
	}
	return t
}

// accountMinimumValue is one entry of the stored object.
type accountMinimumValue struct {
	Minimum float64 `json:"minimum"`
	Note    string  `json:"note,omitempty"`
}

// ParseAccountMinimums reads accounts.minimumBalance: blank is none,
// otherwise a JSON object of account name to {"minimum", "note"} (a bare
// number is read as the minimum, for hand edits). Bad entries (no name, a
// minimum that is not a positive number in range, a name repeated ignoring
// case) are returned as problems and skipped; a value that is not an
// object is an error. The result is sorted by name ignoring case.
func ParseAccountMinimums(raw string) ([]AccountMinimum, []string, error) {
	minimums, problems := []AccountMinimum{}, []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return minimums, problems, nil
	}
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil || parsed == nil {
		return minimums, problems, fmt.Errorf(`value is not a JSON object such as {"Bank A":{"minimum":5000000,"note":"monthly reward"}}`)
	}
	names := make([]string, 0, len(parsed))
	for name := range parsed {
		names = append(names, name)
	}
	slices.SortFunc(names, compareAccountNames)
	for _, rawName := range names {
		name := strings.TrimSpace(rawName)
		if name == "" {
			problems = append(problems, "has an entry without an account name")
			continue
		}
		value, err := parseAccountMinimumValue(parsed[rawName])
		switch {
		case err != nil:
			problems = append(problems, fmt.Sprintf("%q %v", name, err))
		case slices.ContainsFunc(minimums, func(kept AccountMinimum) bool { return sameAccountName(kept.Account, name) }):
			problems = append(problems, fmt.Sprintf("%q is set more than once (names ignore case); the first is used", name))
		default:
			minimums = append(minimums, AccountMinimum{Account: name, Minimum: value.Minimum, Note: strings.TrimSpace(value.Note)})
		}
	}
	return minimums, problems, nil
}

func parseAccountMinimumValue(raw json.RawMessage) (accountMinimumValue, error) {
	var value accountMinimumValue
	trimmed := bytes.TrimSpace(raw)
	var err error
	if len(trimmed) > 0 && trimmed[0] == '{' {
		err = json.Unmarshal(trimmed, &value)
	} else {
		err = json.Unmarshal(trimmed, &value.Minimum)
	}
	if err != nil {
		return value, fmt.Errorf("needs a minimum in VND such as {\"minimum\":5000000}")
	}
	if math.IsNaN(value.Minimum) || value.Minimum <= 0 || value.Minimum > maxAccountMinimum {
		return value, fmt.Errorf("minimum must be more than 0 and at most %s", formatPreferenceNumber(maxAccountMinimum))
	}
	return value, nil
}

// EncodeAccountMinimums writes accounts.minimumBalance for column B: a
// JSON object sorted by name ignoring case, or "" for none (the cell is
// cleared).
func EncodeAccountMinimums(minimums []AccountMinimum) (string, error) {
	sorted := slices.Clone(minimums)
	slices.SortStableFunc(sorted, func(a, b AccountMinimum) int { return compareAccountNames(a.Account, b.Account) })
	if len(sorted) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteByte('{')
	for i, minimum := range sorted {
		name, err := json.Marshal(minimum.Account)
		if err != nil {
			return "", fmt.Errorf("encode account minimums: %w", err)
		}
		value, err := json.Marshal(accountMinimumValue{Minimum: minimum.Minimum, Note: minimum.Note})
		if err != nil {
			return "", fmt.Errorf("encode account minimums: %w", err)
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(name)
		b.WriteByte(':')
		b.Write(value)
	}
	b.WriteByte('}')
	return b.String(), nil
}

func compareAccountNames(a, b string) int {
	if c := strings.Compare(strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// AccountMinimumInput sets one account's minimum balance (D35). A nil or
// 0 Minimum removes the entry. ETag is the AccountMinimumSettings ETag
// read with the setting.
type AccountMinimumInput struct {
	Account string   `json:"account"`
	Minimum *float64 `json:"minimum"`
	Note    string   `json:"note"`
	ETag    string   `json:"etag"`
}

// Removes reports whether the input clears the account's minimum.
func (in AccountMinimumInput) Removes() bool {
	return in.Minimum == nil || *in.Minimum == 0
}

// Validate checks the account, the ETag, the minimum and the note.
func (in AccountMinimumInput) Validate() error {
	var problems []string
	if strings.TrimSpace(in.Account) == "" {
		problems = append(problems, "account is required")
	}
	if strings.TrimSpace(in.ETag) == "" {
		problems = append(problems, "etag is required")
	}
	if in.Minimum != nil {
		if value := *in.Minimum; math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > maxAccountMinimum {
			problems = append(problems, fmt.Sprintf("minimum must be between 0 and %s VND (0 or blank removes it)", formatPreferenceNumber(maxAccountMinimum)))
		}
	}
	if len([]rune(strings.TrimSpace(in.Note))) > maxAccountMinimumNote {
		problems = append(problems, fmt.Sprintf("note must be at most %d characters", maxAccountMinimumNote))
	}
	return validationError(problems)
}

// ApplyAccountMinimum returns the preferences after setting (or, for a nil
// or 0 minimum, removing) the account's minimum. A name matching an entry
// ignoring case replaces it. It fails with ErrConflict when the setting
// changed since the ETag was read. A row the change adds has Row 0.
func ApplyAccountMinimum(current Preferences, input AccountMinimumInput) (Preferences, error) {
	if err := input.Validate(); err != nil {
		return Preferences{}, err
	}
	if strings.TrimSpace(input.ETag) != current.AccountMinimums.ETag {
		return Preferences{}, fmt.Errorf("%w: the minimum balances changed since they were read; reload and try again", ErrConflict)
	}
	account := strings.TrimSpace(input.Account)
	minimums := slices.DeleteFunc(slices.Clone(current.AccountMinimums.Accounts), func(m AccountMinimum) bool { return sameAccountName(m.Account, account) })
	if !input.Removes() {
		minimums = append(minimums, AccountMinimum{Account: account, Minimum: *input.Minimum, Note: strings.TrimSpace(input.Note)})
	}
	slices.SortStableFunc(minimums, func(a, b AccountMinimum) int { return compareAccountNames(a.Account, b.Account) })

	next := Preferences{TabExists: current.TabExists, Entries: slices.Clone(current.Entries), Issues: []string{}}
	index := slices.IndexFunc(next.Entries, func(entry PreferenceEntry) bool { return entry.Key == PrefAccountsMinimumBalance })
	if index >= 0 {
		next.Entries[index].Minimums = minimums
	} else {
		spec, _ := PreferenceSpecFor(PrefAccountsMinimumBalance)
		next.Entries = append(next.Entries, PreferenceEntry{Key: PrefAccountsMinimumBalance, Minimums: minimums, Note: spec.Note})
	}
	next.resolve()
	return next, nil
}
