package domain

import (
	"strings"
	"testing"
)

func memoTxn(id string, date string, payee string, payment float64, deposit float64, memo string) Transaction {
	t := lendingTxn(id, date, payee, payment, deposit)
	t.Memo = &memo
	return t
}

func TestGivenAMemoWhenReadingTheRepaymentDateThenTheEarliestLaterDateIsUsed(t *testing.T) {
	cases := []struct {
		date, memo, want string
	}{
		{"2026-10-01", "repay 2026-11-30", "2026-11-30"},
		{"2026-10-01", "hẹn trả 15/11", "2026-11-15"},
		{"2026-10-01", "trả 15/11/2026", "2026-11-15"},
		{"2026-10-01", "trả 5/1/27", "2027-01-05"},
		{"2026-10-01", "trả 15-11-2026", "2026-11-15"},
		{"2026-10-01", "trả 15.11.2026", "2026-11-15"},
		{"2026-12-20", "trả 05/01", "2027-01-05"},
		{"2026-10-01", "trả 1 nửa 20/10, còn lại 30/12", "2026-10-20"},
		{"2026-10-12", "mượn 12/10, trả 2026-01-12 hoặc 2026-11-02", "2026-11-02"},
	}
	for _, c := range cases {
		got, ok := RepaymentDate(memoTxn("row-2", c.date, "Person A", 1_000_000, 0, c.memo))
		if !ok || got.Format("2006-01-02") != c.want {
			t.Fatalf("RepaymentDate(%q on %s) = %v %v, want %s", c.memo, c.date, got, ok, c.want)
		}
	}
}

func TestGivenAMemoWithoutARepaymentDateWhenReadingThenThereIsNone(t *testing.T) {
	for _, memo := range []string{"", "lãi 1.5%", "trả 2-3 triệu", "trả 31/02", "lent on 12/10", "lent on 05/10", "trả 15-11/2026"} {
		if got, ok := RepaymentDate(memoTxn("row-2", "2026-10-12", "Person A", 1_000_000, 0, memo)); ok {
			t.Fatalf("RepaymentDate(%q) = %v, want none", memo, got)
		}
	}
	if _, ok := RepaymentDate(lendingTxn("row-2", "2026-10-12", "Person A", 1_000_000, 0)); ok {
		t.Fatal("no memo, no date")
	}
}

func TestGivenRepaymentDatesInMemosWhenComputingMoneyLentThenTheNewestNoteOfTheOpenBalanceCounts(t *testing.T) {
	today := day(2026, 10, 19)
	transactions := []Transaction{
		memoTxn("row-2", "2026-01-10", "Person A", 1_000_000, 0, "repay 20/01"),
		lendingTxn("row-3", "2026-01-20", "Person A", 0, 1_000_000),
		memoTxn("row-4", "2026-09-01", "Person A", 5_000_000, 0, "repay 01/10"),
		memoTxn("row-5", "2026-10-01", "Person A", 0, 2_000_000, "paid part, rest by 30/10"),
		memoTxn("row-6", "2026-09-01", "Person B", 1_000_000, 0, "repay 15/10/2026"),
		lendingTxn("row-7", "2026-10-02", "Person B", 0, 1_000_000),
		memoTxn("row-8", "2026-09-01", "Person C", 0, 500_000, "I repay 30/10"),
	}

	lent := ComputeMoneyLent(transactions, today)

	a, c, b := lent.People[0], lent.People[1], lent.People[2]
	if a.Name != "Person A" || a.RepayBy == nil || *a.RepayBy != "2026-10-30" || *a.DaysUntilRepay != 11 {
		t.Fatalf("the newest note replaces the earlier one: %+v", a)
	}
	if b.Name != "Person B" || b.RepayBy != nil || b.DaysUntilRepay != nil {
		t.Fatalf("a settled person has no repayment date: %+v", b)
	}
	if c.Name != "Person C" || c.RepayBy != nil {
		t.Fatalf("money the user owes has no repayment date to chase: %+v", c)
	}
}

func lentPerson(name string, outstanding float64, daysOutstanding int, repayBy string, daysUntil int) MoneyLentPerson {
	person := MoneyLentPerson{Name: name, Outstanding: outstanding, DaysOutstanding: &daysOutstanding}
	if repayBy != "" {
		person.RepayBy, person.DaysUntilRepay = &repayBy, &daysUntil
	}
	return person
}

func TestGivenRepaymentDatesWhenChasingMoneyLentThenOnlyCloseOrPassedDatesAreChased(t *testing.T) {
	later := MoneyLent{People: []MoneyLentPerson{lentPerson("Person A", 3_000_000, 200, "2026-10-23", 4)}}
	if _, findings := EvaluateRecommendationRules(RuleInputs{MoneyLent: &later}); len(findings) != 0 {
		t.Fatalf("a repayment date more than %d days out is not chased, even after 90 days: %+v", LentDueSoonDays, findings)
	}

	soon := MoneyLent{People: []MoneyLentPerson{lentPerson("Person A", 3_000_000, 10, "2026-10-22", 3)}}
	_, findings := EvaluateRecommendationRules(RuleInputs{MoneyLent: &soon})
	if len(findings) != 1 {
		t.Fatalf("findings = %+v", findings)
	}
	chase := findings[0]
	if chase.Title != "Chase money lent: 1 person owes 3.000.000 ₫, repayment due in 3 days" || chase.DueDate != "2026-10-22" || chase.Urgent {
		t.Fatalf("due soon = %+v", chase)
	}
	if !strings.Contains(chase.Action, "due on 2026-10-22") || chase.Evidence != "3.000.000 ₫ due back by 2026-10-22" {
		t.Fatalf("due soon = %+v", chase)
	}

	mixed := MoneyLent{People: []MoneyLentPerson{
		lentPerson("Person A", 3_000_000, 10, "2026-10-15", -4),
		lentPerson("Person B", 1_000_000, 5, "2026-10-19", 0),
		lentPerson("Person C", 2_000_000, 120, "", 0),
		lentPerson("Person D", 4_000_000, 30, "", 0),
	}}
	_, findings = EvaluateRecommendationRules(RuleInputs{MoneyLent: &mixed})
	chase = findings[0]
	if chase.Title != "Chase money lent: 3 people owe 6.000.000 ₫, repayment overdue since 2026-10-15" || chase.DueDate != "2026-10-15" || !chase.Urgent || *chase.AmountVND != 6_000_000 {
		t.Fatalf("overdue = %+v", chase)
	}
	want := "3.000.000 ₫ past the repayment date (due 2026-10-15, 4 days ago); 1.000.000 ₫ due back by 2026-10-19; " +
		"2.000.000 ₫ open longer than 90 days with no repayment date; oldest 120 days"
	if chase.Evidence != want {
		t.Fatalf("evidence = %q", chase.Evidence)
	}

	item := ruleItem(chase, "")
	if item.Status != NeedsYouSerious || item.DueDate != "2026-10-15" {
		t.Fatalf("an overdue repayment is serious in the inbox: %+v", item)
	}
	if item := ruleItem(findings[0], ""); item.Kind != NeedsYouLentOverdue {
		t.Fatalf("kind = %+v", item)
	}
}

func TestGivenARepaymentDateWhenClosingTheMonthThenMoneyLentIsChasedOnlyNearTheDate(t *testing.T) {
	lent := MoneyLent{People: []MoneyLentPerson{
		lentPerson("Person A", 3_000_000, 60, "2026-11-30", 42),
		lentPerson("Person B", 1_000_000, 5, "2026-10-18", -1),
		lentPerson("Person C", 2_000_000, 40, "", 0),
	}}
	lent.OwedToYou = 6_000_000

	item := moneyLentItem(CloseInputs{MoneyLent: &lent})

	if item.State != CloseAction || *item.Count != 2 || *item.Amount != 3_000_000 {
		t.Fatalf("close item = %+v", item)
	}
}
