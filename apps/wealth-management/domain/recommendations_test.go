package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var recToday = time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)

func recItem(id string, source string, title string, status RecommendationStatus) Recommendation {
	return Recommendation{ID: id, Row: 2, Created: "2026-10-01", Source: source, Title: title, Action: "Do it", Status: status, Page: "/budget", ETag: "etag-" + id}
}

func impact(value float64) *float64 { return &value }

func TestGivenSheetRowsWhenParsingThenDatesStatusesAndEvidenceAreRead(t *testing.T) {
	item, problems := ParseRecommendationRow(RecommendationRow{
		ID: " rec-1a2b3c4d ", Created: "46312", Source: "budget-review", Title: " Cut dining ", Action: "Cook twice a week",
		ImpactVND: impact(1_200_000), Evidence: "/budget: Dining 3.000.000 ₫ of 2.000.000 ₫", Status: "", Page: "/budget",
	}, 5, "etag")
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	if item.ID != "rec-1a2b3c4d" || item.Row != 5 || item.Status != RecommendationOpen || item.Title != "Cut dining" || item.Created != "2026-10-17" {
		t.Fatalf("item = %+v", item)
	}
	if item.EvidencePath != "/budget" || item.Evidence != "Dining 3.000.000 ₫ of 2.000.000 ₫" {
		t.Fatalf("evidence = %q %q", item.EvidencePath, item.Evidence)
	}
	if row := item.SheetRow(); row.Evidence != "/budget: Dining 3.000.000 ₫ of 2.000.000 ₫" || row.Status != RecommendationOpen {
		t.Fatalf("row = %+v", row)
	}

	_, problems = ParseRecommendationRow(RecommendationRow{ID: "x", Status: "later", Created: "soon", ImpactVND: impact(-1)}, 6, "")
	joined := strings.Join(problems, "; ")
	for _, want := range []string{"id \"x\"", "title is required", "status \"later\"", "created \"soon\"", "impact must be 0 or more"} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q should mention %q", joined, want)
		}
	}
}

func TestGivenEvidenceTextWhenSplittingThenOnlyALeadingAppPathIsTakenAsTheLink(t *testing.T) {
	cases := map[string][2]string{
		"/accounts/goals: short by 1 ₫": {"/accounts/goals", "short by 1 ₫"},
		"/investments":                  {"/investments", ""},
		"plain facts":                   {"", "plain facts"},
		"/ not a path: x":               {"", "/ not a path: x"},
	}
	for input, want := range cases {
		if path, facts := splitEvidence(input); path != want[0] || facts != want[1] {
			t.Errorf("%q -> %q %q, want %q", input, path, facts, want)
		}
	}
}

func TestGivenItemsWhenBuildingTheListThenAnEndedSnoozeIsOpenAndTheHeadlineSumsOpenImpact(t *testing.T) {
	ended := recItem("rec-a", "budget-review", "Ended snooze", RecommendationSnoozed)
	ended.SnoozedUntil = "2026-10-17"
	ended.ImpactVND = impact(100)
	big := recItem("rec-b", "budget-review", "Big", RecommendationOpen)
	big.ImpactVND = impact(5_000)
	none := recItem("rec-c", "year-review", "No estimate", RecommendationOpen)
	none.Page = "/cash-flow"
	snoozed := recItem("rec-d", "budget-review", "Later", RecommendationSnoozed)
	snoozed.SnoozedUntil = "2026-11-01"
	done := recItem("rec-e", "budget-review", "Done", RecommendationDone)
	dismissed := recItem("rec-f", "budget-review", "Dismissed", RecommendationDismissed)
	sheet := RecommendationSheet{Exists: true, Items: []Recommendation{dismissed, done, snoozed, none, ended, big}}

	list := BuildRecommendationList(sheet, RecommendationFilter{}, recToday)
	ids := []string{}
	for _, item := range list.Items {
		ids = append(ids, item.ID)
	}
	if strings.Join(ids, ",") != "rec-b,rec-a,rec-c,rec-d,rec-e,rec-f" {
		t.Fatalf("order = %v", ids)
	}
	if list.Items[1].Status != RecommendationOpen {
		t.Fatal("a snooze that ended must read as open")
	}
	if s := list.Summary; s.Open != 3 || s.OpenImpactVND != 5_100 || s.OpenWithoutImpact != 1 || s.Snoozed != 1 || s.Done != 1 || s.Dismissed != 1 {
		t.Fatalf("summary = %+v", s)
	}

	page := BuildRecommendationList(sheet, RecommendationFilter{Page: "/cash-flow", Status: RecommendationOpen}, recToday)
	if len(page.Items) != 1 || page.Items[0].ID != "rec-c" || page.Summary.Open != 1 {
		t.Fatalf("page filter = %+v", page)
	}
	if err := (RecommendationFilter{Status: "later"}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown status filter: %v", err)
	}
}

func TestGivenAStatusChangeWhenPlanningThenTheETagIsCheckedAndASnoozeGetsItsDate(t *testing.T) {
	item := recItem("rec-a", "budget-review", "Cut dining", RecommendationOpen)
	item.Outcome = "Not met: still over"
	sheet := RecommendationSheet{Exists: true, Items: []Recommendation{item}}

	changes, err := PlanStatusChange(sheet, "rec-a", RecommendationStatusInput{Status: RecommendationSnoozed, SnoozeDays: 7, ETag: item.ETag}, recToday)
	if err != nil {
		t.Fatal(err)
	}
	update := changes.Update[0]
	if update.Status != RecommendationSnoozed || update.SnoozedUntil != "2026-10-24" || update.StatusChanged != "2026-10-17" || update.Outcome != item.Outcome || update.Row != 2 {
		t.Fatalf("update = %+v", update)
	}
	if _, err := PlanStatusChange(sheet, "rec-a", RecommendationStatusInput{Status: RecommendationDone, ETag: "stale"}, recToday); !errors.Is(err, ErrConflict) {
		t.Fatalf("a stale etag must conflict: %v", err)
	}
	if _, err := PlanStatusChange(sheet, "rec-zz", RecommendationStatusInput{Status: RecommendationDone, ETag: "x"}, recToday); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an unknown id: %v", err)
	}
	for name, input := range map[string]RecommendationStatusInput{
		"past snooze":       {Status: RecommendationSnoozed, SnoozedUntil: "2026-10-17", ETag: item.ETag},
		"no snooze length":  {Status: RecommendationSnoozed, ETag: item.ETag},
		"snooze on done":    {Status: RecommendationDone, SnoozeDays: 7, ETag: item.ETag},
		"unknown status":    {Status: "later", ETag: item.ETag},
		"missing etag":      {Status: RecommendationDone},
		"too long a snooze": {Status: RecommendationSnoozed, SnoozeDays: 400, ETag: item.ETag},
	} {
		if _, err := PlanStatusChange(sheet, "rec-a", input, recToday); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: %v", name, err)
		}
	}
	reopened, err := PlanStatusChange(sheet, "rec-a", RecommendationStatusInput{Status: RecommendationOpen, ETag: item.ETag}, recToday)
	if err != nil || reopened.Update[0].SnoozedUntil != "" || reopened.Update[0].Status != RecommendationOpen {
		t.Fatalf("reopen = %+v %v", reopened, err)
	}
}

func TestGivenAModelReplyWhenDecodingProposalsThenBadItemsAreDroppedNotFatal(t *testing.T) {
	var proposals []RecommendationProposal
	reply := `[
		{"title":"Cut dining","action":"Cook at home twice a week","impactVnd":"1.200.000","evidence":"Dining over limit"},
		{"title":"cut  DINING!","action":"Repeat","impactVnd":null},
		{"title":"","action":"No title"},
		{"title":"No action","action":" "},
		{"title":"Negative","action":"x","impactVnd":-5},
		{"title":"Odd impact","action":"x","impactVnd":{"a":1}},
		{"title":"Second","action":"y","impactVnd":300.4},
		{"title":"Third","action":"z"},
		{"title":"Fourth","action":"w"}
	]`
	if err := json.Unmarshal([]byte(reply), &proposals); err != nil {
		t.Fatal(err)
	}
	normalized := NormalizeProposals(proposals)
	if len(normalized) != 3 || normalized[0].Title != "Cut dining" || *normalized[0].ImpactVND != 1_200_000 || *normalized[1].ImpactVND != 300 || normalized[2].ImpactVND != nil {
		t.Fatalf("normalized = %+v", normalized)
	}
}

func TestGivenAReviewWhenRecordingThenPendingTitlesAreNotRepeatedAndOnlyKnownOutcomesApply(t *testing.T) {
	pending := recItem("rec-a", "budget-review", "Cut dining out", RecommendationOpen)
	pending.Row = 2
	snoozed := recItem("rec-b", "budget-review", "Cancel a subscription", RecommendationSnoozed)
	snoozed.Row, snoozed.SnoozedUntil = 3, "2026-11-01"
	oldDone := recItem("rec-c", "budget-review", "Old", RecommendationDone)
	oldDone.Row, oldDone.StatusChanged = 4, "2026-07-01"
	otherSource := recItem("rec-d", "year-review", "Raise savings", RecommendationOpen)
	otherSource.Row = 5
	dismissed := recItem("rec-e", "budget-review", "Use a cashback card", RecommendationDismissed)
	dismissed.Row = 6
	sheet := RecommendationSheet{Exists: true, Items: []Recommendation{pending, snoozed, oldDone, otherSource, dismissed}}

	if recent := RecentForReview(sheet, "budget-review", recToday); len(recent) != 2 {
		t.Fatalf("recent = %+v", recent)
	}
	changes, duplicates, ignored := PlanReviewRecord(sheet, ReviewRecommendations{
		Source: "budget-review", Page: "/budget",
		Proposals: []RecommendationProposal{
			{Title: "Cut dining-out", Action: "Same as before"},
			{Title: "Use a cashback card", Action: "Dismissed before, may return", ImpactVND: impact(200_000)},
			{Title: "Raise savings", Action: "Pending only in another source"},
		},
		Outcomes: []RecommendationOutcome{
			{ID: "rec-a", Verdict: "met", Note: "Dining is under the limit"},
			{ID: "rec-b", Verdict: "not met"},
			{ID: "rec-c", Verdict: "met"},
			{ID: "rec-d", Verdict: "met"},
			{ID: "rec-a", Verdict: "unclear"},
		},
	}, recToday)
	if duplicates != 1 || ignored != 2 || len(changes.Add) != 2 || len(changes.Update) != 2 {
		t.Fatalf("changes = %+v duplicates %d ignored %d", changes, duplicates, ignored)
	}
	added := changes.Add[0]
	if added.Source != "budget-review" || added.Page != "/budget" || added.EvidencePath != "/budget" || added.Created != "2026-10-17" || added.Status != RecommendationOpen || !strings.HasPrefix(added.ID, "rec-") || len(added.ID) != 12 {
		t.Fatalf("added = %+v", added)
	}
	met, notMet := changes.Update[0], changes.Update[1]
	if met.ID != "rec-a" || met.Status != RecommendationDone || met.StatusChanged != "2026-10-17" || met.Outcome != "Met: Dining is under the limit" {
		t.Fatalf("met = %+v", met)
	}
	if notMet.ID != "rec-b" || notMet.Status != RecommendationSnoozed || notMet.SnoozedUntil != "2026-11-01" || notMet.Outcome != "Not met" {
		t.Fatalf("not met = %+v", notMet)
	}
}

func TestGivenTheSameTitleOnTheSameDayWhenAddingTwiceThenIDsStayUnique(t *testing.T) {
	taken := map[string]bool{}
	first := newRecommendationID("budget-review", "Cut dining", "2026-10-17", taken)
	second := newRecommendationID("budget-review", "Cut dining", "2026-10-17", taken)
	if first == second || first != newRecommendationID("budget-review", "Cut dining", "2026-10-17", map[string]bool{}) {
		t.Fatalf("ids %s %s must be stable and unique", first, second)
	}
}

func TestGivenEngineFiguresWhenEvaluatingRulesThenEachConditionMakesOneFinding(t *testing.T) {
	cryptoNow, cryptoTarget, cryptoDrift := 32.0, 20.0, 12.0
	cashNow, cashTarget, cashDrift := 58.0, 70.0, -12.0
	allocation := Allocation{
		Rebalance: true, TargetedTotal: 100_000_000, DriftThreshold: DriftThreshold,
		Lines: []AllocationLine{
			{Class: AllocationCash, Label: "Cash and deposits", Targeted: true, Current: &cashNow, Target: &cashTarget, Drift: &cashDrift},
			{Class: AllocationCrypto, Label: "Crypto", Targeted: true, Current: &cryptoNow, Target: &cryptoTarget, Drift: &cryptoDrift},
			{Class: AllocationMoneyLent, Label: "Money lent"},
		},
	}
	moveDate := "2028-06"
	plan := GoalPlan{Check: PlanCheck{Verdict: PlanShort, Gap: 2_000_000, RequiredPerMonth: 7_000_000, MonthlySavings: 5_000_000,
		TradeOffs: []GoalTradeOff{{Name: "Goal A", MoveTargetDateTo: &moveDate}}}}
	oldDays, newDays := 120, 30
	lent := MoneyLent{People: []MoneyLentPerson{
		{Name: "Person A", Outstanding: 3_000_000, DaysOutstanding: &oldDays},
		{Name: "Person B", Outstanding: 1_000_000, DaysOutstanding: &newDays},
		{Name: "Person C", Outstanding: -500_000, DaysOutstanding: &oldDays},
	}}

	evaluated, findings := EvaluateRecommendationRules(RuleInputs{Allocation: &allocation, GoalPlan: &plan, MoneyLent: &lent})
	if strings.Join(evaluated, ",") != "rebalance,goal-plan-short,lent-overdue" || len(findings) != 3 {
		t.Fatalf("evaluated %v findings %+v", evaluated, findings)
	}
	rebalance, goals, chase := findings[0], findings[1], findings[2]
	if rebalance.Title != "Rebalance: Cash and deposits is 12 points under target" || !strings.Contains(rebalance.Action, "12.000.000 ₫") || rebalance.Page != "/investments" {
		t.Fatalf("rebalance = %+v", rebalance)
	}
	if goals.Title != "Goal plan is short by 2.000.000 ₫ a month" || *goals.ImpactVND != 24_000_000 || !strings.Contains(goals.Action, "move Goal A to 2028-06") {
		t.Fatalf("goals = %+v", goals)
	}
	if chase.Title != "Chase money lent: 1 person owes 3.000.000 ₫ for over 90 days" || chase.Page != "/accounts/lent" {
		t.Fatalf("chase = %+v", chase)
	}

	idle := IdleCash{Idle: 50_000_000, CurrentRate: 0.5, ExtraPerYear: 2_250_000, Verdict: IdleCashWorthMoving, Best: &DepositRate{Bank: "Bank A", Tenor: "12M", Rate: 5}}
	evaluated, findings = EvaluateRecommendationRules(RuleInputs{IdleCash: &idle})
	if len(evaluated) != 1 || len(findings) != 1 || *findings[0].ImpactVND != 2_250_000 || findings[0].Page != "/bills" || !strings.Contains(findings[0].Action, "Bank A (5.0%, 12M)") {
		t.Fatalf("idle cash: %v %+v", evaluated, findings)
	}
	idle.Verdict = IdleCashNoRate
	if evaluated, _ = EvaluateRecommendationRules(RuleInputs{IdleCash: &idle}); len(evaluated) != 0 {
		t.Fatal("without a rate the idle cash rule cannot judge")
	}
	idle.Verdict = IdleCashNotWorthIt
	if evaluated, findings = EvaluateRecommendationRules(RuleInputs{IdleCash: &idle}); len(evaluated) != 1 || len(findings) != 0 {
		t.Fatalf("a gain too small is no finding: %v %+v", evaluated, findings)
	}

	plan.SetupNeeded = true
	evaluated, findings = EvaluateRecommendationRules(RuleInputs{GoalPlan: &plan})
	if len(evaluated) != 0 || len(findings) != 0 {
		t.Fatalf("a plan that is not set up is not evaluated: %v %v", evaluated, findings)
	}
}

func TestGivenRuleFindingsWhenPlanningThenItemsNeverPileUpAndCloseWhenResolved(t *testing.T) {
	finding := RuleFinding{Rule: RuleLentOverdue, Title: "Chase money lent", Action: "Ask", Evidence: "facts", Page: "/accounts/lent"}
	empty := RecommendationSheet{}

	first := PlanRuleRecommendations(empty, []string{RuleLentOverdue, RuleRebalance}, []RuleFinding{finding}, recToday)
	if len(first.Add) != 1 || first.Add[0].Source != "rule:lent-overdue" || first.Add[0].EvidencePath != "/accounts/lent" || len(first.Update) != 0 {
		t.Fatalf("first = %+v", first)
	}

	open := first.Add[0]
	open.Row = 2
	again := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{open}}, []string{RuleLentOverdue}, []RuleFinding{finding}, recToday)
	if !again.Empty() {
		t.Fatalf("a pending rule item must not be added again: %+v", again)
	}

	notEvaluated := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{open}}, []string{RuleRebalance}, nil, recToday)
	if !notEvaluated.Empty() {
		t.Fatalf("a rule that was not evaluated keeps its items: %+v", notEvaluated)
	}

	resolved := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{open}}, []string{RuleLentOverdue}, nil, recToday)
	if len(resolved.Update) != 1 || resolved.Update[0].Status != RecommendationDone || resolved.Update[0].Outcome != "resolved" || resolved.Update[0].StatusChanged != "2026-10-17" {
		t.Fatalf("resolved = %+v", resolved)
	}

	userDone := open
	userDone.Status, userDone.StatusChanged = RecommendationDone, "2026-10-10"
	if quiet := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{userDone}}, []string{RuleLentOverdue}, []RuleFinding{finding}, recToday); !quiet.Empty() {
		t.Fatalf("an item the user just marked done keeps the rule quiet while the sheet catches up: %+v", quiet)
	}
	userDone.Outcome = "resolved"
	if back := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{userDone}}, []string{RuleLentOverdue}, []RuleFinding{finding}, recToday); len(back.Add) != 1 {
		t.Fatalf("a condition that returns after it resolved adds a new item: %+v", back)
	}

	dismissed := open
	dismissed.Status, dismissed.StatusChanged = RecommendationDismissed, "2026-09-01"
	if quiet := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{dismissed}}, []string{RuleLentOverdue}, []RuleFinding{finding}, recToday); !quiet.Empty() {
		t.Fatalf("a recent dismissal keeps the rule quiet: %+v", quiet)
	}
	dismissed.StatusChanged = "2026-06-01"
	if back := PlanRuleRecommendations(RecommendationSheet{Items: []Recommendation{dismissed}}, []string{RuleLentOverdue}, []RuleFinding{finding}, recToday); len(back.Add) != 1 || back.Add[0].ID == dismissed.ID {
		t.Fatalf("after 90 days the rule may return with a new id: %+v", back)
	}
}

func TestPreviousRecommendationsBlockMasksText(t *testing.T) {
	item := recItem("rec-a", "budget-review", "Ask Person A", RecommendationOpen)
	block := PreviousRecommendationsBlock([]Recommendation{item}, func(text string) string { return strings.ReplaceAll(text, "Person A", "[name]") })
	if !strings.Contains(block, `"id":"rec-a"`) || strings.Contains(block, "Person A") {
		t.Fatalf("block = %s", block)
	}
	if !strings.Contains(PreviousRecommendationsBlock(nil, nil), "none") {
		t.Fatal("an empty list says none")
	}
}
