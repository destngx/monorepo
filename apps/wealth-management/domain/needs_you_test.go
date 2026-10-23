package domain

import (
	"strings"
	"testing"
)

var homeToday = day(2026, 10, 17)

func homeBudget(lines ...BudgetLine) *BudgetReport {
	return &BudgetReport{Month: "2026-10", DaysInMonth: 31, DaysElapsed: 17, Lines: lines}
}

func budgetLine(category string, limit, spent float64) BudgetLine {
	return BudgetLine{Category: category, CategoryType: CategoryTypeExpense, Limit: limit, Spent: spent}
}

func kinds(items []NeedsYouItem) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = string(item.Kind)
	}
	return strings.Join(parts, ",")
}

func findItem(t *testing.T, result NeedsYou, kind NeedsYouKind) NeedsYouItem {
	t.Helper()
	for _, item := range result.Items {
		if item.Kind == kind {
			return item
		}
	}
	t.Fatalf("no %s item in %s", kind, kinds(result.Items))
	return NeedsYouItem{}
}

func TestGivenEverySourceWithAnActionWhenBuildingThenItemsRankBySeverityThenWeightThenDueDate(t *testing.T) {
	drafts := 4
	oldDays := 120
	bills := &Bills{Calendar: BillCalendar{Occurrences: []BillOccurrence{
		{Name: "Card A", Kind: RecurringCard, Source: BillFromCard, ExpectedDate: "2026-10-18", Status: BillDue, DaysUntil: 1, Remaining: 6_000_000},
		{Name: "Card B", Kind: RecurringCard, Source: BillFromCard, ExpectedDate: "2026-10-22", Status: BillDue, DaysUntil: 5, Remaining: 2_000_000},
		{Name: "Card C", Kind: RecurringCard, Source: BillFromCard, ExpectedDate: "2026-10-29", Status: BillDue, DaysUntil: 12, Remaining: 9_000_000},
		{Name: "Internet", Kind: RecurringBill, Source: BillFromRecurring, ExpectedDate: "2026-10-05", Status: BillOverdue, Remaining: 300_000},
		{Name: "Salary", Kind: RecurringIncome, Source: BillFromRecurring, ExpectedDate: "2026-10-10", Status: BillOverdue, Remaining: 30_000_000},
	}}}
	lent := &MoneyLent{People: []MoneyLentPerson{lentPerson("Person A", 3_000_000, oldDays, "", 0)}}
	idle := &IdleCash{Idle: 50_000_000, ExtraPerYear: 2_250_000, Verdict: IdleCashWorthMoving, Best: &DepositRate{Bank: "Bank A", Tenor: "12M", Rate: 5}}
	recommendations := &RecommendationList{Items: []Recommendation{
		{ID: "rec-small", Source: "budget-review", Title: "Cook at home", Status: RecommendationOpen, ImpactVND: impact(1_000_000), Page: "/budget"},
		{ID: "rec-big", Source: "year-review", Title: "Raise the savings target", Status: RecommendationOpen, ImpactVND: impact(9_000_000), Page: "/cash-flow"},
		{ID: "rec-none", Source: "account-review", Title: "Close an unused account", Status: RecommendationOpen, Page: "/accounts"},
		{ID: "rec-four", Source: "budget-review", Title: "Cancel a subscription", Status: RecommendationOpen, ImpactVND: impact(500_000), Page: "/budget"},
		{ID: "rec-done", Source: "budget-review", Title: "Done already", Status: RecommendationDone, ImpactVND: impact(99_000_000)},
	}}

	result := BuildNeedsYou(NeedsYouInputs{
		Today: homeToday, PendingDrafts: &drafts, Bills: bills,
		Budget:    homeBudget(budgetLine("Dining", 2_000_000, 3_000_000)),
		MoneyLent: lent, IdleCash: idle, Recommendations: recommendations,
		Transactions: []Transaction{
			{Date: day(2026, 10, 3), Payee: "Shop", Payment: impact(100_000)},
			txn(day(2026, 10, 4), "Groceries", CategoryTypeExpense, 200_000, 0),
		},
		TransactionsLoaded: true,
	})

	want := "bill-overdue,card-due,budget-over-pace,lent-overdue,drafts,recommendation,idle-cash,recommendation,recommendation,uncategorized"
	if got := kinds(result.Items); got != want {
		t.Fatalf("order = %s\nwant    %s", got, want)
	}
	overdue := result.Items[0]
	if overdue.Status != NeedsYouCritical || *overdue.AmountVND != 300_000 || overdue.Count != 1 || overdue.Route != "/bills" || overdue.DueDate != "2026-10-05" {
		t.Fatalf("overdue = %+v (late income is not a bill)", overdue)
	}
	cards := result.Items[1]
	if cards.Status != NeedsYouSerious || cards.Count != 2 || *cards.AmountVND != 8_000_000 || cards.DueDate != "2026-10-18" || !strings.Contains(cards.Headline, "tomorrow") {
		t.Fatalf("cards = %+v (only payments due within 7 days)", cards)
	}
	if drafts := result.Items[4]; drafts.Count != 4 || drafts.Route != "/transactions/drafts" || drafts.Headline != "4 captured transactions to approve" {
		t.Fatalf("drafts = %+v", drafts)
	}
	if uncategorized := findItem(t, result, NeedsYouUncategorized); uncategorized.Count != 1 || uncategorized.Route != "/transactions?uncategorized=1" {
		t.Fatalf("uncategorized = %+v", uncategorized)
	}
	// Info items: the larger yearly impact first; the three largest open
	// recommendations, done ones never.
	if result.Items[5].ID != "rec-big" || result.Items[6].Kind != NeedsYouIdleCash || result.Items[7].ID != "rec-small" || result.Items[8].ID != "rec-four" {
		t.Fatalf("info order = %+v", result.Items[5:])
	}
	if result.Items[5].Route != "/recommendations?page=%2Fcash-flow" || result.Items[5].RecommendationID != "rec-big" {
		t.Fatalf("recommendation = %+v", result.Items[5])
	}
	if result.AsOf != "2026-10-17" || len(result.Unavailable) != 0 || result.Ranking == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestGivenCardPaymentsLaterThanAWeekOrPaidWhenBuildingThenNoCardItem(t *testing.T) {
	bills := &Bills{Calendar: BillCalendar{Occurrences: []BillOccurrence{
		{Name: "Card A", Kind: RecurringCard, Source: BillFromCard, ExpectedDate: "2026-10-29", Status: BillDue, DaysUntil: 12, Remaining: 4_000_000},
		{Name: "Card B", Kind: RecurringCard, Source: BillFromCard, ExpectedDate: "2026-10-20", Status: BillPaid, DaysUntil: 3},
		{Name: "Gym", Kind: RecurringBill, Source: BillFromRecurring, ExpectedDate: "2026-10-20", Status: BillDue, DaysUntil: 3, Remaining: 500_000},
	}}}
	if result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}); len(result.Items) != 0 {
		t.Fatalf("items = %+v", result.Items)
	}
	bills.Calendar.Occurrences[0].DaysUntil = 6
	if item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}), NeedsYouCardDue); item.Status != NeedsYouWarning || item.Headline != "1 card payment due, the first in 6 days" {
		t.Fatalf("item = %+v", item)
	}
}

func TestGivenBudgetLinesWhenCheckingPaceThenFixedCategoriesCountAsSpentAndPaceStartsOnDaySeven(t *testing.T) {
	categories := []Category{{Name: "Rent", Type: CategoryTypeExpense, Nature: NatureFixed}}
	// Day 17 of 31: Dining spent 1.2M of 2M projects to about 2.19M; Rent is paid once.
	report := homeBudget(
		budgetLine("Dining", 2_000_000, 1_200_000),
		budgetLine("Rent", 10_000_000, 10_000_000),
		budgetLine("Groceries", 5_000_000, 1_000_000),
		BudgetLine{Category: "Lent", CategoryType: CategoryTypeExpense, Lending: true, Limit: 1, Spent: 5_000_000},
	)
	result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, Budget: report, Categories: categories})
	item := findItem(t, result, NeedsYouBudgetOverPace)
	if item.Count != 1 || item.Status != NeedsYouWarning || item.Headline != "Dining is on pace to pass its budget" || *item.AmountVND != 188_235 {
		t.Fatalf("item = %+v", item)
	}

	// Before day 7 only spend already over the limit counts.
	early := &BudgetReport{Month: "2026-10", DaysInMonth: 31, DaysElapsed: 3, Lines: []BudgetLine{
		budgetLine("Dining", 2_000_000, 1_000_000), budgetLine("Shopping", 1_000_000, 1_500_000),
	}}
	result = BuildNeedsYou(NeedsYouInputs{Today: day(2026, 10, 3), Budget: early})
	item = findItem(t, result, NeedsYouBudgetOverPace)
	if item.Count != 1 || item.Status != NeedsYouSerious || item.Headline != "Shopping is over its budget" || *item.AmountVND != 500_000 {
		t.Fatalf("early item = %+v", item)
	}

	// Several categories name the three largest overruns.
	many := homeBudget(
		budgetLine("A", 1_000_000, 2_000_000), budgetLine("B", 1_000_000, 3_000_000),
		budgetLine("C", 1_000_000, 4_000_000), budgetLine("D", 1_000_000, 5_000_000),
	)
	result = BuildNeedsYou(NeedsYouInputs{Today: homeToday, Budget: many})
	if item = findItem(t, result, NeedsYouBudgetOverPace); item.Headline != "4 budget categories over pace: D, C, B, +1 more" {
		t.Fatalf("headline = %q", item.Headline)
	}

	// Another month's report is not this month's pace.
	past := homeBudget(budgetLine("Dining", 1, 5))
	past.Month = "2026-09"
	if result = BuildNeedsYou(NeedsYouInputs{Today: homeToday, Budget: past}); len(result.Items) != 0 {
		t.Fatalf("items = %+v", result.Items)
	}
}

func TestGivenRuleConditionsWhenBuildingThenLiveFindingsAreTheSourceAndTheUserDecisionKeepsThemQuiet(t *testing.T) {
	oldDays := 120
	cashNow, cashTarget, cashDrift := 58.0, 70.0, -12.0
	cryptoNow, cryptoTarget, cryptoDrift := 32.0, 20.0, 12.0
	allocation := &Allocation{Rebalance: true, TargetedTotal: 100_000_000, DriftThreshold: DriftThreshold, Lines: []AllocationLine{
		{Class: AllocationCash, Label: "Cash and deposits", Targeted: true, Current: &cashNow, Target: &cashTarget, Drift: &cashDrift},
		{Class: AllocationCrypto, Label: "Crypto", Targeted: true, Current: &cryptoNow, Target: &cryptoTarget, Drift: &cryptoDrift},
	}}
	lent := &MoneyLent{People: []MoneyLentPerson{lentPerson("Person A", 3_000_000, oldDays, "", 0)}}
	plan := &GoalPlan{Check: PlanCheck{Verdict: PlanShort, Gap: 2_000_000, RequiredPerMonth: 7_000_000, MonthlySavings: 5_000_000},
		Goals: []GoalProgress{{PlannedGoal: PlannedGoal{Name: "Goal A", Status: GoalActive}, Pace: PaceBehind}}}
	inputs := NeedsYouInputs{Today: homeToday, Allocation: allocation, MoneyLent: lent, GoalPlan: plan}

	// Without the recommendations list, every finding shows.
	result := BuildNeedsYou(inputs)
	if got := kinds(result.Items); got != "goal-plan-short,rebalance,lent-overdue" {
		t.Fatalf("kinds = %s (a short plan stands for the goals behind)", got)
	}
	short := result.Items[0]
	if *short.ImpactVND != 24_000_000 || *short.AmountVND != 2_000_000 || short.Route != "/accounts/goals" {
		t.Fatalf("short = %+v", short)
	}
	if rebalance := result.Items[1]; *rebalance.AmountVND != 12_000_000 || rebalance.Route != "/investments" {
		t.Fatalf("rebalance = %+v", rebalance)
	}
	for _, item := range result.Items {
		if strings.Contains(item.Headline+item.Detail, "Person A") || strings.Contains(item.Headline, "Goal A") {
			t.Fatalf("names leak into %+v", item)
		}
	}

	// Rule items in the tab: the open one links, never repeats; snoozed,
	// recently dismissed and user-done ones keep their finding quiet.
	inputs.Recommendations = &RecommendationList{Items: []Recommendation{
		{ID: "rec-rebal", Source: "rule:rebalance", Title: "Rebalance (stale words)", Status: RecommendationOpen, ImpactVND: impact(1)},
		{ID: "rec-lent", Source: "rule:lent-overdue", Title: "Chase", Status: RecommendationDismissed, StatusChanged: "2026-09-01"},
		{ID: "rec-goal", Source: "rule:goal-plan-short", Title: "Short", Status: RecommendationSnoozed, SnoozedUntil: "2026-11-01"},
	}}
	result = BuildNeedsYou(inputs)
	if got := kinds(result.Items); got != "rebalance" {
		t.Fatalf("kinds = %s (the goals behind stay quiet with the short plan)", got)
	}
	if result.Items[0].RecommendationID != "rec-rebal" || !strings.HasPrefix(result.Items[0].Headline, "Rebalance: Cash") {
		t.Fatalf("rebalance = %+v", result.Items[0])
	}

	// An old dismissal and a rule-resolved done item no longer silence.
	inputs.Recommendations = &RecommendationList{Items: []Recommendation{
		{ID: "rec-lent", Source: "rule:lent-overdue", Title: "Chase", Status: RecommendationDismissed, StatusChanged: "2026-06-01"},
		{ID: "rec-goal", Source: "rule:goal-plan-short", Title: "Short", Status: RecommendationDone, StatusChanged: "2026-10-10", Outcome: "resolved"},
	}}
	if got := kinds(BuildNeedsYou(inputs).Items); got != "goal-plan-short,rebalance,lent-overdue" {
		t.Fatalf("kinds = %s", got)
	}
}

func TestGivenGoalsBehindWithAPlanThatFitsWhenBuildingThenOneGoalItemCountsThemWithoutNames(t *testing.T) {
	required := 1_500_000.0
	plan := &GoalPlan{Check: PlanCheck{Verdict: PlanFits}, Goals: []GoalProgress{
		{PlannedGoal: PlannedGoal{Name: "Goal A", Status: GoalActive}, Pace: PaceBehind, RequiredPerMonth: &required},
		{PlannedGoal: PlannedGoal{Name: "Goal B", Status: GoalActive}, Pace: PaceBehind},
		{PlannedGoal: PlannedGoal{Name: "Goal C", Status: GoalPaused}, Pace: PaceBehind},
		{PlannedGoal: PlannedGoal{Name: "Goal D", Status: GoalActive}, Pace: PaceOnTrack},
	}}
	result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, GoalPlan: plan})
	item := findItem(t, result, NeedsYouGoalBehind)
	if item.Count != 2 || item.Headline != "2 goals are behind pace" || *item.AmountVND != 1_500_000 || item.Route != "/accounts/goals" {
		t.Fatalf("item = %+v", item)
	}

	plan.SetupNeeded = true
	if result = BuildNeedsYou(NeedsYouInputs{Today: homeToday, GoalPlan: plan}); len(result.Items) != 0 {
		t.Fatalf("a plan not set up adds nothing: %+v", result.Items)
	}
}

func TestGivenUncategorizedRowsWhenCountingThenOnlyRecentUsableRowsWithoutAKnownCategoryCount(t *testing.T) {
	unknown := txn(day(2026, 10, 1), "Mystery", CategoryTypeExpense, 1, 0)
	unknown.CategoryType = nil
	named := txn(day(2026, 10, 2), " uncategorized ", CategoryTypeExpense, 1, 0)
	broken := Transaction{Date: day(2026, 10, 3), Issues: []string{"bad amount"}}
	old := Transaction{Date: day(2026, 7, 1)}
	future := Transaction{Date: day(2026, 10, 30)}
	blank := Transaction{Date: day(2026, 10, 5)}
	known := txn(day(2026, 10, 6), "Transfer", CategoryTypeNonBudget, 1, 0)

	result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, TransactionsLoaded: true,
		Transactions: []Transaction{unknown, named, broken, old, future, blank, known}})
	if item := findItem(t, result, NeedsYouUncategorized); item.Count != 3 || item.Headline != "3 transactions have no category (last 60 days)" {
		t.Fatalf("item = %+v", item)
	}
}

func TestGivenNothingToDoWhenBuildingThenTheInboxIsEmptyAndUnavailablePartsAreListed(t *testing.T) {
	none := 0
	result := BuildNeedsYou(NeedsYouInputs{
		Today: homeToday, PendingDrafts: &none, Bills: &Bills{}, Budget: homeBudget(budgetLine("Dining", 2_000_000, 100_000)),
		Recommendations: &RecommendationList{}, TransactionsLoaded: true,
		Unavailable: []NeedsYouUnavailable{NeedsYouMissing(NeedsYouPartMoneyLent)},
	})
	if len(result.Items) != 0 {
		t.Fatalf("items = %+v", result.Items)
	}
	if len(result.Unavailable) != 1 || result.Unavailable[0].Part != NeedsYouPartMoneyLent || result.Unavailable[0].Label != "Money lent" {
		t.Fatalf("unavailable = %+v", result.Unavailable)
	}
	if empty := BuildNeedsYou(NeedsYouInputs{Today: homeToday}); empty.Items == nil || empty.Unavailable == nil {
		t.Fatal("lists must encode as [] not null")
	}
}

func TestGivenLastMonthEndedWithoutASnapshotWhenBuildingThenOneCloseItemCountsTheChecklistWithoutRepeatingIt(t *testing.T) {
	close := &MonthlyClose{Month: "2026-09", MonthEnded: true, Items: []CloseItem{
		{Key: CloseDrafts, State: CloseAction},
		{Key: CloseCategorize, State: CloseDone},
		{Key: CloseBudget, State: CloseUnavailable},
		{Key: CloseSnapshot, State: CloseAction},
	}}
	today := day(2026, 10, 3)
	result := BuildNeedsYou(NeedsYouInputs{Today: today, Close: close})
	if len(result.Items) != 1 {
		t.Fatalf("items = %+v", result.Items)
	}
	item := result.Items[0]
	if item.Kind != NeedsYouMonthClose || item.Headline != "Close September 2026" || item.Count != 2 || item.Route != "/close" ||
		item.DueDate != "2026-10-10" || item.Status != NeedsYouWarning || !strings.Contains(item.Detail, "2 steps left") {
		t.Fatalf("item = %+v", item)
	}

	// Snapshot written: the close is done enough; the other lines have their own items.
	close.Items[3].State = CloseDone
	if result = BuildNeedsYou(NeedsYouInputs{Today: today, Close: close}); len(result.Items) != 0 {
		t.Fatalf("items = %+v", result.Items)
	}
	// This month's close (not ended yet) is not due.
	close.Items[3].State, close.MonthEnded, close.Month = CloseAction, false, "2026-10"
	if result = BuildNeedsYou(NeedsYouInputs{Today: day(2026, 10, 17), Close: close}); len(result.Items) != 0 {
		t.Fatalf("items = %+v", result.Items)
	}
}

func TestGivenReadyMoneyShortOfTheBillsBeforePaydayWhenBuildingThenOneSeriousItemSaysByHowMuch(t *testing.T) {
	bills := &Bills{SafeToSpend: SafeToSpend{Until: "2026-10-25", UntilIncome: true, IncomeName: "Salary", ShortBy: 2_500_000, Amount: -2_500_000}}
	item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}), NeedsYouShortBeforePayday)
	if item.Status != NeedsYouSerious || item.AmountVND == nil || *item.AmountVND != 2_500_000 || item.DueDate != "2026-10-25" || item.Route != "/bills" {
		t.Fatalf("item = %+v", item)
	}
	if item.Headline != "Ready money does not cover what must be paid before payday" || strings.Contains(item.Headline+item.Detail, "Salary") {
		t.Fatalf("headline = %q detail = %q", item.Headline, item.Detail)
	}

	bills.SafeToSpend.UntilIncome = false
	if item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}), NeedsYouShortBeforePayday); item.Headline != "Ready money does not cover what must be paid in the next 30 days" {
		t.Fatalf("headline without income = %q", item.Headline)
	}

	bills.SafeToSpend.ShortBy = 0
	if result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}); len(result.Items) != 0 {
		t.Fatalf("covered: %+v", result.Items)
	}
}

func TestGivenEssentialsNotCoveredBeforePaydayWhenBuildingThenOneWarningSaysByHowMuch(t *testing.T) {
	bills := &Bills{SafeToSpend: SafeToSpend{Until: "2026-10-25", UntilIncome: true, IncomeName: "Salary", Amount: 400_000, Essentials: 800_000, TightBy: 400_000}}
	result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills})
	item := findItem(t, result, NeedsYouTightBeforePayday)
	if item.Status != NeedsYouWarning || item.AmountVND == nil || *item.AmountVND != 400_000 || item.DueDate != "2026-10-25" || item.Route != "/bills" {
		t.Fatalf("item = %+v", item)
	}
	if item.Headline != "Money for essentials runs short before payday" || strings.Contains(item.Headline+item.Detail, "Salary") {
		t.Fatalf("headline = %q detail = %q", item.Headline, item.Detail)
	}
	if len(result.Items) != 1 {
		t.Fatalf("covered commitments are not short: %+v", result.Items)
	}

	bills.SafeToSpend.UntilIncome = false
	if item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}), NeedsYouTightBeforePayday); item.Headline != "Money for essentials runs short in the next 30 days" {
		t.Fatalf("headline without income = %q", item.Headline)
	}

	// Short already says it: no second item.
	bills.SafeToSpend.Amount, bills.SafeToSpend.ShortBy, bills.SafeToSpend.TightBy = -100_000, 100_000, 800_000
	result = BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills})
	if len(result.Items) != 1 || result.Items[0].Kind != NeedsYouShortBeforePayday {
		t.Fatalf("short = %+v", result.Items)
	}
}

func TestGivenAnEmergencyFundBelowTheBestInstantRateWhenBuildingThenItIsItsOwnInfoItem(t *testing.T) {
	input := idleInput()
	input.Accounts = append(input.Accounts, Account{Name: "Rainy Day Bank", Type: AccountTypeBank, Balance: 20_000_000, Note: note("lãi 4%")})
	input.Settings.EmergencyAccounts = []string{"Rainy Day Bank"}
	idle := ComputeIdleCash(input)

	result := BuildNeedsYou(NeedsYouInputs{Today: homeToday, IdleCash: &idle})
	emergency := findItem(t, result, NeedsYouEmergencyFundRate)
	idleItem := findItem(t, result, NeedsYouIdleCash)
	if emergency.ID == idleItem.ID || emergency.Status != NeedsYouInfo || *emergency.ImpactVND != 300_000 || emergency.Route != "/bills" {
		t.Fatalf("emergency = %+v idle = %+v", emergency, idleItem)
	}
	if strings.Contains(emergency.Headline, "Rainy") || strings.Contains(emergency.Detail, "Rainy") {
		t.Fatalf("no account names in the inbox: %+v", emergency)
	}
}

func TestGivenAnAccountProjectedBelowItsMinimumAverageWhenBuildingThenOneNamelessItemSaysHowMuchMore(t *testing.T) {
	input := billsInput()
	input.Accounts = append(input.Accounts, Account{Name: "Reward Bank", Type: AccountTypeBank, Balance: 4_000_000})
	input.Minimums = AccountMinimumSettings{Accounts: []AccountMinimum{{Account: "Reward Bank", Minimum: 5_000_000}, {Account: "Main Bank", Minimum: 5_000_000}}}
	bills := ComputeBills(input)

	item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: &bills}), NeedsYouMinimumBalance)
	track := bills.SafeToSpend.MinimumBalances
	extra := 0.0
	for _, account := range track {
		extra += account.ExtraNeeded
	}
	if item.Status != NeedsYouWarning || item.Count != 0 || item.AmountVND == nil || *item.AmountVND != extra || extra <= 0 ||
		item.DueDate != "2026-10-31" || item.Route != "/bills" {
		t.Fatalf("item = %+v", item)
	}
	if item.Headline != "An account is at risk of missing its minimum average balance" || strings.Contains(item.Headline+item.Detail, "Reward") {
		t.Fatalf("no account names in the inbox: %+v", item)
	}

	// The last week is serious.
	input.Today = day(2026, 10, 26)
	late := ComputeBills(input)
	if item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: input.Today, Bills: &late}), NeedsYouMinimumBalance); item.Status != NeedsYouSerious {
		t.Fatalf("last week = %+v", item)
	}

	// On track: no item.
	input.Today = homeToday
	input.Accounts[len(input.Accounts)-1].Balance = 9_000_000
	covered := ComputeBills(input)
	for _, item := range BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: &covered}).Items {
		if item.Kind == NeedsYouMinimumBalance {
			t.Fatalf("on track: %+v", item)
		}
	}
}

func TestGivenTwoAccountsAtRiskWhenBuildingThenTheItemCountsThem(t *testing.T) {
	bills := &Bills{SafeToSpend: SafeToSpend{MinimumBalances: []AccountMinimumBalance{
		{Account: "A", Status: MinimumAtRisk, ExtraNeeded: 1_000_000, DaysLeft: 10, MonthEnd: "2026-10-31"},
		{Account: "B", Status: MinimumAtRisk, ExtraNeeded: 2_000_000, DaysLeft: 10, MonthEnd: "2026-10-31"},
	}}}
	item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, Bills: bills}), NeedsYouMinimumBalance)
	if item.Count != 2 || *item.AmountVND != 3_000_000 || item.Headline != "2 accounts are at risk of missing their minimum average balance" {
		t.Fatalf("item = %+v", item)
	}
}

func TestGivenABetterWalletProductWhenBuildingThenTheSwitchIsANamelessInfoItem(t *testing.T) {
	idle := ComputeIdleCash(productInput())

	item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, IdleCash: &idle}), NeedsYouWalletSwitch)
	if item.Status != NeedsYouInfo || item.Route != "/bills" || item.ImpactVND == nil || *item.ImpactVND != idle.Switches[0].ExtraPerYear {
		t.Fatalf("item = %+v", item)
	}
	if !strings.Contains(item.Detail, "Wallet Z Earn pays 4.5% now; Wallet M Earn pays 6%") || !strings.Contains(item.Detail, "not a bank deposit") {
		t.Fatalf("detail = %q", item.Detail)
	}
	// "Wallet Z Earn" is the product; the account is "Wallet Z" alone.
	if strings.Contains(strings.ReplaceAll(item.Headline+item.Detail, "Wallet Z Earn", ""), "Wallet Z") {
		t.Fatalf("no account names in the inbox: %+v", item)
	}

	idle.Switches = nil
	if _, ok := walletSwitchFinding(idle); ok {
		t.Fatal("no switch, no finding")
	}
}

func TestGivenTheBestPlaceIsNotADepositWhenTheEmergencyFundCouldEarnMoreThenTheItemSaysSo(t *testing.T) {
	input := productInput()
	input.Settings.EmergencyAccounts = []string{"Wallet Z"}
	idle := ComputeIdleCash(input)

	item := findItem(t, BuildNeedsYou(NeedsYouInputs{Today: homeToday, IdleCash: &idle}), NeedsYouEmergencyFundRate)
	if !strings.Contains(item.Detail, "Wallet M Earn is not a bank deposit") {
		t.Fatalf("detail = %q", item.Detail)
	}
}
