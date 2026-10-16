package domain

import (
	"cmp"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Home "Needs you" (ROADMAP R5, WM-HOME-1..): one ranked list of what the
// user should act on, built from figures the engine already computes. Each
// item says what it is, how urgent, how much money it is about and which
// page to open. Headlines never carry the names of people or goals.
//
// Rule-backed kinds (rebalance, lent-overdue, goal-plan-short, idle-cash)
// come from the live rule findings (EvaluateRecommendationRules), never from
// the Recommendations tab, so the inbox is current without a refresh; the
// tab holds the user's decision: a rule item snoozed, dismissed in the last
// 90 days or marked done in the last 30 keeps its inbox item quiet. Open
// recommendations from the reviews (not rules) join as their own kind.

// NeedsYouKind is what an inbox item is about.
type NeedsYouKind string

const (
	NeedsYouDrafts         NeedsYouKind = "drafts"
	NeedsYouBillOverdue    NeedsYouKind = "bill-overdue"
	NeedsYouCardDue        NeedsYouKind = "card-due"
	NeedsYouBudgetOverPace NeedsYouKind = "budget-over-pace"
	NeedsYouLentOverdue    NeedsYouKind = "lent-overdue"
	NeedsYouGoalPlanShort  NeedsYouKind = "goal-plan-short"
	NeedsYouGoalBehind     NeedsYouKind = "goal-behind"
	NeedsYouRebalance      NeedsYouKind = "rebalance"
	NeedsYouUncategorized  NeedsYouKind = "uncategorized"
	NeedsYouRecommendation NeedsYouKind = "recommendation"
	NeedsYouIdleCash       NeedsYouKind = "idle-cash"
	NeedsYouMonthClose     NeedsYouKind = "month-close"
)

// needsYouKindOrder breaks ties after severity, weight and due date.
var needsYouKindOrder = []NeedsYouKind{
	NeedsYouBillOverdue, NeedsYouCardDue, NeedsYouBudgetOverPace, NeedsYouMonthClose, NeedsYouDrafts, NeedsYouLentOverdue,
	NeedsYouGoalPlanShort, NeedsYouGoalBehind, NeedsYouRebalance, NeedsYouUncategorized, NeedsYouRecommendation,
	NeedsYouIdleCash,
}

// NeedsYouStatus is the shared status token of an item (UI status scale).
type NeedsYouStatus string

const (
	NeedsYouCritical NeedsYouStatus = "critical"
	NeedsYouSerious  NeedsYouStatus = "serious"
	NeedsYouWarning  NeedsYouStatus = "warning"
	NeedsYouInfo     NeedsYouStatus = "info"
)

// severity ranks a status: lower is more urgent.
func (s NeedsYouStatus) severity() int {
	switch s {
	case NeedsYouCritical:
		return 0
	case NeedsYouSerious:
		return 1
	case NeedsYouWarning:
		return 2
	default:
		return 3
	}
}

const (
	// NeedsYouCardDueDays: card payments due this soon are listed.
	NeedsYouCardDueDays = 7
	// needsYouCardUrgentDays: a card payment due this soon is serious.
	needsYouCardUrgentDays = 2
	// NeedsYouUncategorizedDays: uncategorized rows this recent are listed.
	NeedsYouUncategorizedDays = 60
	// NeedsYouRecommendations caps the review recommendations listed.
	NeedsYouRecommendations = 3

	// NeedsYouRanking states the order for clients.
	NeedsYouRanking = "Severity first (critical, serious, warning, info), then the larger weight (the estimated VND a year when known, " +
		"else the amount the item is about), then the earlier due date, then kind."
)

// NeedsYouItem is one action in the inbox.
type NeedsYouItem struct {
	// ID is stable while the item stands: the kind, or the recommendation id.
	ID       string         `json:"id"`
	Kind     NeedsYouKind   `json:"kind"`
	Status   NeedsYouStatus `json:"status"`
	Headline string         `json:"headline"`
	Detail   string         `json:"detail,omitempty"`
	// AmountVND is the money the item is about (to pay, over budget, owed,
	// to move); nil when the item has no amount.
	AmountVND *float64 `json:"amountVnd"`
	// ImpactVND is the estimated VND a year, when known.
	ImpactVND *float64 `json:"impactVnd"`
	// DueDate (yyyy-mm-dd) is when it is due or runs out; empty when undated.
	DueDate string `json:"dueDate,omitempty"`
	// Count is how many things the item groups (drafts, bills, categories,
	// people, goals, rows); 0 when it is one thing.
	Count int `json:"count,omitempty"`
	// Route is the app page where the user acts on it.
	Route string `json:"route"`
	// RecommendationID links the open recommendation behind the item, if any.
	RecommendationID string `json:"recommendationId,omitempty"`
}

// weight orders items of the same severity.
func (item NeedsYouItem) weight() float64 {
	switch {
	case item.ImpactVND != nil:
		return *item.ImpactVND
	case item.AmountVND != nil:
		return *item.AmountVND
	default:
		return 0
	}
}

// NeedsYouPart names one source of the inbox.
type NeedsYouPart string

const (
	NeedsYouPartDrafts          NeedsYouPart = "drafts"
	NeedsYouPartBills           NeedsYouPart = "bills"
	NeedsYouPartBudget          NeedsYouPart = "budget"
	NeedsYouPartMoneyLent       NeedsYouPart = "moneyLent"
	NeedsYouPartGoals           NeedsYouPart = "goals"
	NeedsYouPartAllocation      NeedsYouPart = "allocation"
	NeedsYouPartTransactions    NeedsYouPart = "transactions"
	NeedsYouPartRecommendations NeedsYouPart = "recommendations"
	NeedsYouPartIdleCash        NeedsYouPart = "idleCash"
	NeedsYouPartClose           NeedsYouPart = "close"
)

var needsYouPartLabels = map[NeedsYouPart]string{
	NeedsYouPartDrafts:          "Drafts to approve",
	NeedsYouPartBills:           "Bills and card payments",
	NeedsYouPartBudget:          "Budget",
	NeedsYouPartMoneyLent:       "Money lent",
	NeedsYouPartGoals:           "Goals",
	NeedsYouPartAllocation:      "Allocation",
	NeedsYouPartTransactions:    "Uncategorized transactions",
	NeedsYouPartRecommendations: "Recommendations",
	NeedsYouPartIdleCash:        "Idle cash",
	NeedsYouPartClose:           "Monthly close",
}

// NeedsYouUnavailable is a source that could not be loaded: its items are
// missing, so an empty inbox does not mean all is well.
type NeedsYouUnavailable struct {
	Part  NeedsYouPart `json:"part"`
	Label string       `json:"label"`
}

// NeedsYouMissing makes the entry for a part that failed.
func NeedsYouMissing(part NeedsYouPart) NeedsYouUnavailable {
	return NeedsYouUnavailable{Part: part, Label: needsYouPartLabels[part]}
}

// NeedsYou is the ranked inbox.
type NeedsYou struct {
	AsOf        string                `json:"asOf"`
	Items       []NeedsYouItem        `json:"items"`
	Unavailable []NeedsYouUnavailable `json:"unavailable"`
	Ranking     string                `json:"ranking"`
}

// NeedsYouInputs are the loaded sources; nil (or a nil count) means the
// source is not configured or failed, and adds no items.
type NeedsYouInputs struct {
	Today         time.Time
	PendingDrafts *int
	Bills         *Bills
	Budget        *BudgetReport
	Categories    []Category
	MoneyLent     *MoneyLent
	GoalPlan      *GoalPlan
	Allocation    *Allocation
	IdleCash      *IdleCash
	// Transactions are read for uncategorized rows when TransactionsLoaded.
	Transactions       []Transaction
	TransactionsLoaded bool
	Recommendations    *RecommendationList
	// Close is the default month-close checklist (last month during the
	// first CloseGraceDays days, else this month).
	Close       *MonthlyClose
	Unavailable []NeedsYouUnavailable
}

// BuildNeedsYou makes and ranks the inbox items.
func BuildNeedsYou(in NeedsYouInputs) NeedsYou {
	today := Today(in.Today)
	result := NeedsYou{AsOf: today.Format(time.DateOnly), Items: []NeedsYouItem{}, Unavailable: in.Unavailable, Ranking: NeedsYouRanking}
	if result.Unavailable == nil {
		result.Unavailable = []NeedsYouUnavailable{}
	}
	add := func(item NeedsYouItem, ok bool) {
		if ok {
			result.Items = append(result.Items, item)
		}
	}
	if in.PendingDrafts != nil {
		add(needsDraftsItem(*in.PendingDrafts))
	}
	if in.Bills != nil {
		add(billsOverdueItem(*in.Bills))
		add(cardsDueItem(*in.Bills))
	}
	if in.Budget != nil {
		add(budgetOverPaceItem(*in.Budget, in.Categories, today))
	}
	if in.TransactionsLoaded {
		add(uncategorizedItem(in.Transactions, today))
	}
	if in.Close != nil {
		add(monthCloseItem(*in.Close, today))
	}

	_, findings := EvaluateRecommendationRules(RuleInputs{Allocation: in.Allocation, GoalPlan: in.GoalPlan, MoneyLent: in.MoneyLent, IdleCash: in.IdleCash})
	var items []Recommendation
	if in.Recommendations != nil {
		items = in.Recommendations.Items
	}
	planShort := false
	for _, finding := range findings {
		if finding.Rule == RuleGoalPlanShort {
			planShort = true
		}
		if in.Recommendations != nil && ruleSilenced(items, finding.Rule, today) {
			continue
		}
		result.Items = append(result.Items, ruleItem(finding, pendingRuleID(items, finding.Rule)))
	}
	// A short plan already explains goals falling behind.
	if in.GoalPlan != nil && !in.GoalPlan.SetupNeeded && !planShort {
		add(goalsBehindItem(*in.GoalPlan))
	}
	if in.Recommendations != nil {
		result.Items = append(result.Items, recommendationItems(items)...)
	}
	slices.SortStableFunc(result.Items, compareNeedsYou)
	return result
}

func compareNeedsYou(a, b NeedsYouItem) int {
	if c := cmp.Compare(a.Status.severity(), b.Status.severity()); c != 0 {
		return c
	}
	if c := cmp.Compare(b.weight(), a.weight()); c != 0 {
		return c
	}
	// Dated items before undated ones, the earlier date first.
	switch {
	case a.DueDate != "" && b.DueDate == "":
		return -1
	case a.DueDate == "" && b.DueDate != "":
		return 1
	}
	if c := cmp.Compare(a.DueDate, b.DueDate); c != 0 {
		return c
	}
	return cmp.Compare(slices.Index(needsYouKindOrder, a.Kind), slices.Index(needsYouKindOrder, b.Kind))
}

func needsDraftsItem(pending int) (NeedsYouItem, bool) {
	if pending <= 0 {
		return NeedsYouItem{}, false
	}
	return NeedsYouItem{
		ID: string(NeedsYouDrafts), Kind: NeedsYouDrafts, Status: NeedsYouWarning, Count: pending,
		Headline: plural(pending, "captured transaction", "captured transactions") + " to approve",
		Detail:   "Approve or reject the drafts read from your captures so the ledger stays complete.",
		Route:    "/transactions/drafts",
	}, true
}

// billsOverdueItem: unpaid bills and card payments whose date has passed
// are critical: they cost late fees or card interest.
func billsOverdueItem(bills Bills) (NeedsYouItem, bool) {
	count, total, earliest := 0, 0.0, ""
	for _, o := range bills.Calendar.Occurrences {
		if o.IsIncome() || o.Status != BillOverdue {
			continue
		}
		count++
		total += o.Remaining
		if earliest == "" || o.ExpectedDate < earliest {
			earliest = o.ExpectedDate
		}
	}
	if count == 0 {
		return NeedsYouItem{}, false
	}
	return NeedsYouItem{
		ID: string(NeedsYouBillOverdue), Kind: NeedsYouBillOverdue, Status: NeedsYouCritical, Count: count,
		Headline:  plural(count, "bill is", "bills are") + " overdue",
		Detail:    "Pay them, or record the payment if it was made, so safe-to-spend stays right.",
		AmountVND: vndAmount(total), DueDate: earliest, Route: "/bills",
	}, true
}

// cardsDueItem: unpaid card payments due within a week; serious within two days.
func cardsDueItem(bills Bills) (NeedsYouItem, bool) {
	count, total, earliest, soonest := 0, 0.0, "", NeedsYouCardDueDays+1
	for _, o := range bills.Calendar.Occurrences {
		if o.Source != BillFromCard || o.Status != BillDue || o.DaysUntil > NeedsYouCardDueDays || o.Remaining <= 0 {
			continue
		}
		count++
		total += o.Remaining
		soonest = min(soonest, o.DaysUntil)
		if earliest == "" || o.ExpectedDate < earliest {
			earliest = o.ExpectedDate
		}
	}
	if count == 0 {
		return NeedsYouItem{}, false
	}
	status := NeedsYouWarning
	if soonest <= needsYouCardUrgentDays {
		status = NeedsYouSerious
	}
	when := fmt.Sprintf("in %d days", soonest)
	switch soonest {
	case 0:
		when = "today"
	case 1:
		when = "tomorrow"
	}
	return NeedsYouItem{
		ID: string(NeedsYouCardDue), Kind: NeedsYouCardDue, Status: status, Count: count,
		Headline:  plural(count, "card payment", "card payments") + " due, the first " + when,
		Detail:    "Pay the statement balance in full by the due date to avoid interest.",
		AmountVND: vndAmount(total), DueDate: earliest, Route: "/bills",
	}, true
}

// budgetOverPaceItem: expense categories over their limit, or on pace to
// pass it by the month end. The pace extrapolates spend per day from day 7
// (as WM-DASH-9); fixed categories are paid once, so only their actual
// spend counts.
func budgetOverPaceItem(report BudgetReport, categories []Category, today time.Time) (NeedsYouItem, bool) {
	if report.Month != MonthKey(today) || report.DaysElapsed <= 0 {
		return NeedsYouItem{}, false
	}
	fixed := map[string]bool{}
	for _, category := range categories {
		if category.Nature == NatureFixed {
			fixed[categoryKey(category.Name)] = true
		}
	}
	type overLine struct {
		name  string
		over  float64
		spent bool
	}
	var lines []overLine
	total, runsOut := 0.0, ""
	anyOver := false
	for _, line := range report.Lines {
		if !line.CountsAsExpense() || line.Limit <= 0 || line.Spent <= 0 {
			continue
		}
		projected := line.Spent
		if !fixed[categoryKey(line.Category)] && report.DaysElapsed >= ExpensePaceMinDays {
			projected = line.Spent / float64(report.DaysElapsed) * float64(report.DaysInMonth)
		}
		if line.Spent <= line.Limit && projected <= line.Limit {
			continue
		}
		over := math.Max(projected, line.Spent) - line.Limit
		overNow := line.Spent > line.Limit
		anyOver = anyOver || overNow
		lines = append(lines, overLine{name: line.Category, over: over, spent: overNow})
		total += over
		if line.BurnOutDate != nil && (runsOut == "" || *line.BurnOutDate < runsOut) {
			runsOut = *line.BurnOutDate
		}
	}
	if len(lines) == 0 {
		return NeedsYouItem{}, false
	}
	slices.SortStableFunc(lines, func(a, b overLine) int { return cmp.Compare(b.over, a.over) })
	names := make([]string, 0, 3)
	for i, line := range lines {
		if i == 3 {
			names = append(names, fmt.Sprintf("+%d more", len(lines)-3))
			break
		}
		names = append(names, line.name)
	}
	status, headline := NeedsYouWarning, ""
	if anyOver {
		status = NeedsYouSerious
	}
	if len(lines) == 1 {
		headline = lines[0].name + " is on pace to pass its budget"
		if lines[0].spent {
			headline = lines[0].name + " is over its budget"
		}
	} else {
		headline = fmt.Sprintf("%d budget categories over pace: %s", len(lines), strings.Join(names, ", "))
	}
	return NeedsYouItem{
		ID: string(NeedsYouBudgetOverPace), Kind: NeedsYouBudgetOverPace, Status: status, Count: len(lines),
		Headline:  headline,
		Detail:    "About this much over the limits by the month end at the current pace; slow down or move limit between categories.",
		AmountVND: vndAmount(total), DueDate: runsOut, Route: "/budget",
	}, true
}

func uncategorizedItem(transactions []Transaction, today time.Time) (NeedsYouItem, bool) {
	since := today.AddDate(0, 0, -NeedsYouUncategorizedDays)
	count := 0
	for _, t := range transactions {
		if t.Usable() && !t.Date.Before(since) && !t.Date.After(today) && NeedsCategory(t) {
			count++
		}
	}
	if count == 0 {
		return NeedsYouItem{}, false
	}
	return NeedsYouItem{
		ID: string(NeedsYouUncategorized), Kind: NeedsYouUncategorized, Status: NeedsYouInfo, Count: count,
		Headline: plural(count, "transaction has", "transactions have") + fmt.Sprintf(" no category (last %d days)", NeedsYouUncategorizedDays),
		Detail:   "Budget, cash flow and the pace leave them out until they have a category.",
		Route:    "/transactions?uncategorized=1",
	}, true
}

var ruleKinds = map[string]struct {
	kind   NeedsYouKind
	status NeedsYouStatus
}{
	RuleRebalance:     {NeedsYouRebalance, NeedsYouWarning},
	RuleLentOverdue:   {NeedsYouLentOverdue, NeedsYouWarning},
	RuleGoalPlanShort: {NeedsYouGoalPlanShort, NeedsYouWarning},
	RuleIdleCash:      {NeedsYouIdleCash, NeedsYouInfo},
}

func ruleItem(finding RuleFinding, recommendationID string) NeedsYouItem {
	meta := ruleKinds[finding.Rule]
	return NeedsYouItem{
		ID: string(meta.kind), Kind: meta.kind, Status: meta.status, Headline: finding.Title, Detail: finding.Action,
		AmountVND: finding.AmountVND, ImpactVND: finding.ImpactVND, Route: finding.Page, RecommendationID: recommendationID,
	}
}

// ruleSilenced: the user snoozed the rule's item, dismissed it in the last
// 90 days or marked it done in the last 30 (as PlanRuleRecommendations).
// Items are as listed: a snooze that ended reads as open.
func ruleSilenced(items []Recommendation, rule string, today time.Time) bool {
	source := RecommendationRuleSource(rule)
	dismissedSince := today.AddDate(0, 0, -RecommendationDismissQuietDays).Format(time.DateOnly)
	doneSince := today.AddDate(0, 0, -RecommendationDoneQuietDays).Format(time.DateOnly)
	for _, item := range items {
		if item.Source != source {
			continue
		}
		switch {
		case item.Status == RecommendationSnoozed:
			return true
		case item.Status == RecommendationDismissed && item.StatusChanged >= dismissedSince:
			return true
		case item.Status == RecommendationDone && item.Outcome != ruleResolvedOutcome && item.StatusChanged >= doneSince:
			return true
		}
	}
	return false
}

func pendingRuleID(items []Recommendation, rule string) string {
	source := RecommendationRuleSource(rule)
	for _, item := range items {
		if item.Source == source && item.Status == RecommendationOpen {
			return item.ID
		}
	}
	return ""
}

func goalsBehindItem(plan GoalPlan) (NeedsYouItem, bool) {
	behind, needed := 0, 0.0
	for _, goal := range plan.Goals {
		if goal.Status != GoalActive || goal.Pace != PaceBehind {
			continue
		}
		behind++
		if goal.RequiredPerMonth != nil {
			needed += *goal.RequiredPerMonth
		}
	}
	if behind == 0 {
		return NeedsYouItem{}, false
	}
	item := NeedsYouItem{
		ID: string(NeedsYouGoalBehind), Kind: NeedsYouGoalBehind, Status: NeedsYouWarning, Count: behind,
		Headline: plural(behind, "goal is", "goals are") + " behind pace",
		Detail:   "At the current savings they finish after their target dates; reorder the goals, push a date or lower a target.",
		Route:    "/accounts/goals",
	}
	if needed > 0 {
		item.AmountVND = vndAmount(needed)
	}
	return item, true
}

// recommendationItems: the open review recommendations (rules have their
// live items), largest impact first, at most NeedsYouRecommendations.
func recommendationItems(items []Recommendation) []NeedsYouItem {
	open := []Recommendation{}
	for _, item := range items {
		if item.Status == RecommendationOpen && !item.IsRule() {
			open = append(open, item)
		}
	}
	slices.SortStableFunc(open, func(a, b Recommendation) int {
		return cmp.Compare(impactOf(b), impactOf(a))
	})
	result := []NeedsYouItem{}
	for _, item := range open[:min(len(open), NeedsYouRecommendations)] {
		route := "/recommendations"
		if item.Page != "" {
			route += "?page=" + url.QueryEscape(item.Page)
		}
		result = append(result, NeedsYouItem{
			ID: item.ID, Kind: NeedsYouRecommendation, Status: NeedsYouInfo, Headline: item.Title, Detail: item.Action,
			ImpactVND: item.ImpactVND, Route: route, RecommendationID: item.ID,
		})
	}
	return result
}

// monthCloseItem: last month has ended and its snapshot is not written
// (WM-CLOSE). Only the snapshot is new here: drafts, categories, budget,
// goals, money lent and recommendations have their own items, so the
// checklist's other lines are counted, not repeated.
func monthCloseItem(checklist MonthlyClose, today time.Time) (NeedsYouItem, bool) {
	if !checklist.MonthEnded {
		return NeedsYouItem{}, false
	}
	snapshotDue, open := false, 0
	for _, item := range checklist.Items {
		if item.State != CloseAction {
			continue
		}
		open++
		snapshotDue = snapshotDue || item.Key == CloseSnapshot
	}
	if !snapshotDue {
		return NeedsYouItem{}, false
	}
	label := checklist.Month
	if month, err := ParseMonthKey(checklist.Month); err == nil {
		label = month.Format("January 2006")
	}
	due := firstOfMonthOf(today).AddDate(0, 0, CloseGraceDays-1)
	return NeedsYouItem{
		ID: string(NeedsYouMonthClose), Kind: NeedsYouMonthClose, Status: NeedsYouWarning, Count: open,
		Headline: "Close " + label,
		Detail:   fmt.Sprintf("Go through the month-close checklist (%s left) and write the month's snapshot.", plural(open, "step", "steps")),
		DueDate:  due.Format(time.DateOnly), Route: "/close",
	}, true
}
