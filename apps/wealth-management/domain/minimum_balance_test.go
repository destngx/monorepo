package domain

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// Minimum balance to keep (D35): a bank rewards an account whose average
// daily balance over the month stays at or above a minimum. Names are
// invented.

func minimumTxn(account string, date int, payment, deposit float64) Transaction {
	t := txn(day(2026, 10, date), "Transfer", CategoryTypeNonBudget, payment, deposit)
	t.AccountName = account
	return t
}

func TestGivenAMinimumObjectWhenParsingThenEntriesAreTrimmedSortedAndBadOnesReported(t *testing.T) {
	raw := `{" Reward Bank ":{"minimum":5000000,"note":" monthly points reward "},"Alpha Bank":2000000,` +
		`"reward bank":{"minimum":1},"Broken":{"minimum":-5},"Text":"lots","":{"minimum":1}}`
	minimums, problems, err := ParseAccountMinimums(raw)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(minimums) != 2 || minimums[0] != (AccountMinimum{Account: "Alpha Bank", Minimum: 2_000_000}) ||
		minimums[1].Account != "Reward Bank" || minimums[1].Minimum != 5_000_000 || minimums[1].Note != "monthly points reward" {
		t.Fatalf("minimums = %+v", minimums)
	}
	// The case-insensitive repeat, the negative, the text and the blank name.
	if len(problems) != 4 {
		t.Fatalf("problems = %v", problems)
	}
	if empty, problems, err := ParseAccountMinimums("  "); err != nil || len(empty) != 0 || len(problems) != 0 {
		t.Fatalf("blank = %v %v %v", empty, problems, err)
	}
	if _, _, err := ParseAccountMinimums(`["Reward Bank"]`); err == nil {
		t.Fatalf("a list is not a minimum object")
	}
}

func TestGivenMinimumsWhenEncodingThenTheJSONIsSortedAndReadsBack(t *testing.T) {
	encoded, err := EncodeAccountMinimums([]AccountMinimum{
		{Account: "Reward Bank", Minimum: 5_000_000, Note: "monthly points reward"},
		{Account: "alpha bank", Minimum: 2_000_000},
	})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := `{"alpha bank":{"minimum":2000000},"Reward Bank":{"minimum":5000000,"note":"monthly points reward"}}`
	if encoded != want {
		t.Fatalf("encoded = %s", encoded)
	}
	back, problems, err := ParseAccountMinimums(encoded)
	if err != nil || len(problems) != 0 || len(back) != 2 || back[1].Note != "monthly points reward" {
		t.Fatalf("read back = %+v %v %v", back, problems, err)
	}
	if empty, err := EncodeAccountMinimums(nil); err != nil || empty != "" {
		t.Fatalf("no minimums clear the cell: %q %v", empty, err)
	}
}

func TestGivenTheMinimumRowWhenParsingPreferencesThenTheSettingCarriesItsETag(t *testing.T) {
	prefs := ParsePreferences([][]string{
		{"accounts.minimumBalance", `{"Reward Bank":{"minimum":5000000,"note":"points"},"Bad":{"minimum":"x"}}`, "note"},
	}, 2)
	settings := prefs.AccountMinimums
	if len(settings.Accounts) != 1 || settings.Accounts[0].Minimum != 5_000_000 || settings.ETag == "" || settings.ETag == preferenceETag("") {
		t.Fatalf("settings = %+v", settings)
	}
	if len(prefs.Issues) != 1 || !strings.Contains(prefs.Issues[0], `"Bad"`) {
		t.Fatalf("issues = %v", prefs.Issues)
	}
	if minimum, ok := settings.For(" reward bank "); !ok || minimum.Minimum != 5_000_000 {
		t.Fatalf("lookup ignores case and spaces: %+v %v", minimum, ok)
	}
	broken := ParsePreferences([][]string{{"accounts.minimumBalance", "5000000"}}, 2)
	if len(broken.AccountMinimums.Accounts) != 0 || len(broken.Issues) != 1 {
		t.Fatalf("a value that is not an object sets nothing: %+v", broken)
	}
}

func TestGivenAMinimumWhenSettingItThenTheRowIsAddedChangedAndRemovedWithTheETag(t *testing.T) {
	current := ParsePreferences(nil, 2)
	next, err := ApplyAccountMinimum(current, AccountMinimumInput{Account: " Reward Bank ", Minimum: money(5_000_000), Note: " points ", ETag: current.AccountMinimums.ETag})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(next.Entries) != 1 || next.Entries[0].Row != 0 || next.Entries[0].Note == "" ||
		len(next.AccountMinimums.Accounts) != 1 || next.AccountMinimums.Accounts[0] != (AccountMinimum{Account: "Reward Bank", Minimum: 5_000_000, Note: "points"}) {
		t.Fatalf("next = %+v", next)
	}
	encoded, _ := EncodeAccountMinimums(next.Entries[0].Minimums)
	saved := ParsePreferences([][]string{{string(PrefAccountsMinimumBalance), encoded}}, 2)

	changed, err := ApplyAccountMinimum(saved, AccountMinimumInput{Account: "reward bank", Minimum: money(6_000_000), ETag: saved.AccountMinimums.ETag})
	if err != nil || len(changed.AccountMinimums.Accounts) != 1 || changed.AccountMinimums.Accounts[0].Minimum != 6_000_000 || changed.AccountMinimums.Accounts[0].Account != "reward bank" {
		t.Fatalf("changed = %+v %v", changed.AccountMinimums, err)
	}
	for _, clear := range []*float64{nil, money(0)} {
		cleared, err := ApplyAccountMinimum(saved, AccountMinimumInput{Account: "REWARD BANK", Minimum: clear, ETag: saved.AccountMinimums.ETag})
		if err != nil || len(cleared.AccountMinimums.Accounts) != 0 {
			t.Fatalf("blank or 0 removes the entry: %+v %v", cleared.AccountMinimums, err)
		}
	}
	if _, err := ApplyAccountMinimum(saved, AccountMinimumInput{Account: "Reward Bank", Minimum: money(1), ETag: current.AccountMinimums.ETag}); !errors.Is(err, ErrConflict) {
		t.Fatalf("a stale etag is a conflict: %v", err)
	}
	for _, bad := range []AccountMinimumInput{
		{Account: "Reward Bank", Minimum: money(1)},
		{Minimum: money(1), ETag: "x"},
		{Account: "Reward Bank", Minimum: money(-1), ETag: "x"},
		{Account: "Reward Bank", Minimum: money(math.Inf(1)), ETag: "x"},
		{Account: "Reward Bank", Minimum: money(1), Note: strings.Repeat("n", maxAccountMinimumNote+1), ETag: "x"},
	} {
		if _, err := ApplyAccountMinimum(saved, bad); !errors.Is(err, ErrValidation) {
			t.Fatalf("%+v: err = %v", bad, err)
		}
	}
}

func TestGivenTheMinimumKeyWhenSavingItAsANumberThenTheRouteIsNamed(t *testing.T) {
	err := PreferencesUpdate{Values: []PreferenceValue{{Key: PrefAccountsMinimumBalance, Value: money(1)}}}.Validate()
	if !errors.Is(err, ErrValidation) || !strings.Contains(err.Error(), "/api/bills/safe-to-spend/minimum-balance") {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenDepositsAndWithdrawalsMidMonthWhenTrackingTheAverageThenDailyBalancesAreRebuiltBackwards(t *testing.T) {
	// 17 October, 31 days. Today 8M. A 6M withdrawal on the 10th, a 10M
	// deposit on the 5th, a 1M payment today (counted in today's balance,
	// not in the days so far). Another account's flow is ignored.
	account := Account{Name: "Reward Bank", Type: AccountTypeBank, Balance: 8_000_000}
	transactions := []Transaction{
		minimumTxn("Reward Bank", 5, 0, 10_000_000),
		minimumTxn("Reward Bank", 10, 6_000_000, 0),
		minimumTxn("Reward Bank", 17, 1_000_000, 0),
		minimumTxn("Other Bank", 3, 50_000_000, 0),
		minimumTxn("Reward Bank", 18, 0, 99_000_000),
	}
	track := TrackMinimumBalance(account, AccountMinimum{Account: "Reward Bank", Minimum: 5_000_000, Note: "points"}, transactions, billsToday)
	// End of day 16 = 9M, days 10-16 = 9M, days 5-9 = 15M, days 1-4 = 5M.
	// Sum of days 1-16 = 4x5 + 5x15 + 7x9 = 158M.
	if track.Month != "2026-10" || track.DaysInMonth != 31 || track.DaysCounted != 16 || track.DaysLeft != 15 {
		t.Fatalf("days = %+v", track)
	}
	if track.SumSoFar != 158_000_000 || track.AverageSoFar != 9_875_000 {
		t.Fatalf("so far = %+v", track)
	}
	// (158M + 8M x 15) / 31.
	if track.ProjectedAverage != math.Round(278_000_000.0/31) {
		t.Fatalf("projected = %v", track.ProjectedAverage)
	}
	// (5M x 31 - 158M) / 15 is negative: nothing more is needed.
	if track.RequiredBalance != 0 || track.ExtraNeeded != 0 || track.Status != MinimumOnTrack || track.Reserved != 5_000_000 || track.Note != "points" {
		t.Fatalf("track = %+v", track)
	}
}

func TestGivenALowBalanceWhenTrackingTheAverageThenTheRequiredBalanceForTheRestOfTheMonthIsShown(t *testing.T) {
	// 4M all month (no transactions) against a 5M minimum on 17 October.
	account := Account{Name: "Reward Bank", Type: AccountTypeBank, Balance: 4_000_000}
	track := TrackMinimumBalance(account, AccountMinimum{Account: "Reward Bank", Minimum: 5_000_000}, nil, billsToday)
	// required = (5M x 31 - 4M x 16) / 15 = 91M / 15.
	if track.SumSoFar != 64_000_000 || track.RequiredBalance != math.Ceil(91_000_000.0/15) {
		t.Fatalf("track = %+v", track)
	}
	if track.Status != MinimumAtRisk || track.ExtraNeeded != track.RequiredBalance-4_000_000 || track.ProjectedAverage != 4_000_000 || track.Reserved != 4_000_000 {
		t.Fatalf("track = %+v", track)
	}
}

func TestGivenTheLastDayOfTheMonthWhenTrackingTheAverageThenTodayAloneMustCoverTheGap(t *testing.T) {
	account := Account{Name: "Reward Bank", Type: AccountTypeBank, Balance: 4_000_000}
	track := TrackMinimumBalance(account, AccountMinimum{Account: "Reward Bank", Minimum: 5_000_000}, nil, day(2026, 10, 31))
	// required = 5M x 31 - 4M x 30 = 35M for the one day left.
	if track.DaysCounted != 30 || track.DaysLeft != 1 || track.RequiredBalance != 35_000_000 || track.ExtraNeeded != 31_000_000 || track.Status != MinimumAtRisk {
		t.Fatalf("track = %+v", track)
	}
	first := TrackMinimumBalance(account, AccountMinimum{Account: "Reward Bank", Minimum: 5_000_000}, nil, day(2026, 10, 1))
	// The first day: nothing counted yet, the average so far is today's balance.
	if first.DaysCounted != 0 || first.DaysLeft != 31 || first.AverageSoFar != 4_000_000 || first.RequiredBalance != 5_000_000 {
		t.Fatalf("first = %+v", first)
	}
}

func TestGivenAnAccountOpenedThisMonthWhenTrackingTheAverageThenTheDaysBeforeCountAsEmpty(t *testing.T) {
	// Opened with a 9M deposit on the 9th: days 1-8 rebuild to 0.
	account := Account{Name: "New Bank", Type: AccountTypeBank, Balance: 9_000_000}
	track := TrackMinimumBalance(account, AccountMinimum{Account: "New Bank", Minimum: 5_000_000}, []Transaction{minimumTxn("New Bank", 9, 0, 9_000_000)}, billsToday)
	// Sum = 8 x 9M = 72M; required = (155M - 72M) / 15.
	if track.SumSoFar != 72_000_000 || track.RequiredBalance != math.Ceil(83_000_000.0/15) || track.Status != MinimumOnTrack {
		t.Fatalf("track = %+v", track)
	}
}
