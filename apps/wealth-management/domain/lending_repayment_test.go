package domain

import (
	"reflect"
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
		{"2026-03-01", "trả 31/10", "2026-10-31"},
		{"2026-10-01", "trả 29/02", "2028-02-29"},
		{"2026-03-01", "investment vào vườn chanh dây trên lâm đồng dự kiến cuối tháng 11 hoàn trả", "2026-11-30"},
		{"2026-10-01", "cuối tháng 2", "2027-02-28"},
		{"2026-10-01", "Giữa Tháng 11", "2026-11-15"},
		{"2026-10-01", "đầu tháng 12", "2026-12-01"},
		{"2026-10-01", "hẹn tháng 11 trả", "2026-11-30"},
		{"2026-10-01", "cuoi thang 1", "2027-01-31"},
		{"2026-10-01", "trả cuối tháng 3/2027", "2027-03-31"},
		{"2026-10-01", "trả cuối tháng 3/27", "2027-03-31"},
		{"2026-10-01", "trả đầu tháng 1 năm 2027", "2027-01-01"},
		{"2026-10-01", "(cuối tháng 11), lãi 1.5%", "2026-11-30"},
		{"2026-10-12", "hẹn trả khi có lương ngày 24", "2026-10-24"},
		{"2026-10-12", "trả ngày 5", "2026-11-05"},
		{"2026-11-05", "ngay 31", "2026-12-31"},
		{"2026-10-12", "mùng 5", "2026-11-05"},
		{"2026-10-01", "trả ngày 15 tháng 11", "2026-11-15"},
		{"2026-10-01", "mùng 5 tháng 1 năm 2027", "2027-01-05"},
		{"2026-10-01", "ngay 20 thang 12/26", "2026-12-20"},
		{"2026-10-01", "trả ngày 24/11", "2026-11-24"},
	}
	for _, c := range cases {
		got, ok := RepaymentDate(memoTxn("row-2", c.date, "Person A", 1_000_000, 0, c.memo))
		if !ok || got.Format("2006-01-02") != c.want {
			t.Fatalf("RepaymentDate(%q on %s) = %v %v, want %s", c.memo, c.date, got, ok, c.want)
		}
	}
}

func TestGivenAMemoWithoutARepaymentDateWhenReadingThenThereIsNone(t *testing.T) {
	for _, memo := range []string{"", "lãi 1.5%", "trả 2-3 triệu", "trả 31/02", "lent on 12/10", "lent on 05/10", "trả 15-11/2026",
		"đầu tháng 10", "cuối tháng", "trả trong 3 tháng", "tháng 13", "tháng 0", "cuối tháng 9/2026",
		"trả ngay 2 triệu", "trả ngay 5k", "chuyển ngày 12", "ngày 32", "ngày 24h", "ngày 30 tháng 2"} {
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

// lentPerson owes one open loan.
func TestGivenSeveralLoansToOnePersonWhenComputingMoneyLentThenEachKeepsItsDateAndRepaymentsPayTheOldestFirst(t *testing.T) {
	today := day(2026, 10, 23)
	transactions := []Transaction{
		memoTxn("row-2", "2026-10-01", "Person A", 2_000_000, 0, "repay 31/10"),
		memoTxn("row-3", "2026-10-05", "Person A", 3_000_000, 0, "repay 30/11"),
		lendingTxn("row-4", "2026-10-08", "Person A", 1_000_000, 0),
		lendingTxn("row-5", "2026-10-10", "Person A", 0, 500_000),
	}

	person := ComputeMoneyLent(transactions, today).People[0]

	if person.Outstanding != 5_500_000 || len(person.Loans) != 3 {
		t.Fatalf("person = %+v", person)
	}
	first, second, third := person.Loans[0], person.Loans[1], person.Loans[2]
	if first.TransactionID != "row-2" || first.Repaid != 500_000 || first.Outstanding != 1_500_000 || first.DaysOutstanding != 22 || *first.RepayBy != "2026-10-31" {
		t.Fatalf("the repayment pays the oldest loan: %+v", first)
	}
	if second.Outstanding != 3_000_000 || *second.RepayBy != "2026-11-30" || *second.DaysUntilRepay != 38 {
		t.Fatalf("the second loan keeps its own date: %+v", second)
	}
	if third.Outstanding != 1_000_000 || third.RepayBy != nil {
		t.Fatalf("a loan without a memo date has none: %+v", third)
	}
	if *person.RepayBy != "2026-10-31" || *person.DaysUntilRepay != 8 || person.RepayByAmount != 1_500_000 {
		t.Fatalf("the person shows the soonest date and what is due then: %+v", person)
	}
	if len(first.Repayments) != 1 || first.Repayments[0] != (LoanRepayment{TransactionID: "row-5", Date: "2026-10-10", Amount: 500_000}) || len(second.Repayments) != 0 {
		t.Fatalf("each loan lists the repayments that paid it: %+v %+v", first.Repayments, second.Repayments)
	}

	transactions = append(transactions, memoTxn("row-6", "2026-10-20", "Person A", 0, 2_000_000, "paid the first, rest of the next by 15/12"))
	person = ComputeMoneyLent(transactions, today).People[0]
	first, second = person.Loans[0], person.Loans[1]
	if first.Outstanding != 0 || first.Repaid != 2_000_000 || first.RepayBy != nil || first.DaysUntilRepay != nil {
		t.Fatalf("a repaid loan has no date to chase: %+v", first)
	}
	if second.Repaid != 500_000 || *second.RepayBy != "2026-12-15" {
		t.Fatalf("a dated repayment memo moves the oldest loan it leaves open: %+v", second)
	}
	if len(first.Repayments) != 2 || first.Repayments[1] != (LoanRepayment{TransactionID: "row-6", Date: "2026-10-20", Amount: 1_500_000}) ||
		len(second.Repayments) != 1 || second.Repayments[0] != (LoanRepayment{TransactionID: "row-6", Date: "2026-10-20", Amount: 500_000}) {
		t.Fatalf("a repayment is split over the loans it paid: %+v %+v", first.Repayments, second.Repayments)
	}
	if *person.RepayBy != "2026-12-15" || person.RepayByAmount != 2_500_000 {
		t.Fatalf("person = %+v", person)
	}
}

func TestGivenASettledOrPrepaidHistoryWhenComputingMoneyLentThenOnlyTheOpenBalanceLoansShow(t *testing.T) {
	today := day(2026, 10, 23)
	transactions := []Transaction{
		memoTxn("row-2", "2026-01-10", "Person A", 1_000_000, 0, "repay 20/01"),
		lendingTxn("row-3", "2026-01-20", "Person A", 0, 1_000_000),
		lendingTxn("row-4", "2026-09-01", "Person A", 0, 400_000),
		memoTxn("row-5", "2026-10-01", "Person A", 1_000_000, 0, "repay 30/10"),
		lendingTxn("row-6", "2026-10-02", "Person B", 0, 700_000),
	}

	lent := ComputeMoneyLent(transactions, today)

	a, b := lent.People[0], lent.People[1]
	if len(a.Loans) != 1 || a.Loans[0].TransactionID != "row-5" || a.Loans[0].Repaid != 400_000 || a.Loans[0].Outstanding != 600_000 {
		t.Fatalf("money repaid ahead offsets the next loan; settled loans before the balance opened are left out: %+v", a.Loans)
	}
	if want := []LoanRepayment{{TransactionID: "row-4", Date: "2026-09-01", Amount: 400_000}}; !reflect.DeepEqual(a.Loans[0].Repayments, want) {
		t.Fatalf("money repaid ahead is listed on the loan it offsets: %+v", a.Loans[0].Repayments)
	}
	if b.Outstanding != -700_000 || len(b.Loans) != 0 || b.RepayBy != nil {
		t.Fatalf("money the user owes has no loans to chase: %+v", b)
	}
}

func TestGivenOneOverdueLoanAndOneLaterWhenChasingThenOnlyTheOverdueLoanIsCounted(t *testing.T) {
	lent := MoneyLent{People: []MoneyLentPerson{owingPerson("Person A",
		openLoan(1_000_000, 30, "2026-10-20", -3),
		openLoan(5_000_000, 10, "2026-12-30", 68),
	)}}

	_, findings := EvaluateRecommendationRules(RuleInputs{MoneyLent: &lent})

	if len(findings) != 1 || *findings[0].AmountVND != 1_000_000 || findings[0].DueDate != "2026-10-20" || !findings[0].Urgent {
		t.Fatalf("findings = %+v", findings)
	}
	item := moneyLentItem(CloseInputs{MoneyLent: &lent})
	if *item.Count != 1 || *item.Amount != 1_000_000 {
		t.Fatalf("close item = %+v", item)
	}
}

func refTxn(t Transaction, reference string) Transaction {
	t.ReferenceNumber = &reference
	return t
}

func TestGivenARepaymentWithALoansRefWhenComputingMoneyLentThenItPaysThatLoanFirst(t *testing.T) {
	today := day(2026, 10, 23)
	transactions := []Transaction{
		refTxn(memoTxn("row-2", "2026-10-01", "Person A", 1_000_000, 0, "repay 31/10"), "L1"),
		refTxn(memoTxn("row-3", "2026-10-05", "Person A", 2_000_000, 0, "repay 30/11"), " l2 "),
		refTxn(lendingTxn("row-4", "2026-10-10", "Person A", 0, 2_500_000), "L2"),
		refTxn(memoTxn("row-5", "2026-10-12", "Person A", 0, 100_000, "rest by 15/12"), "L2"),
		refTxn(lendingTxn("row-6", "2026-10-14", "Person A", 0, 100_000), "unknown"),
	}

	loans := ComputeMoneyLent(transactions, today).People[0].Loans

	first, second := loans[0], loans[1]
	if *first.Reference != "L1" || *second.Reference != "l2" {
		t.Fatalf("loans carry their trimmed Ref #: %+v %+v", first, second)
	}
	if second.Outstanding != 0 || !reflect.DeepEqual(second.Repayments, []LoanRepayment{{TransactionID: "row-4", Date: "2026-10-10", Amount: 2_000_000}}) {
		t.Fatalf("a repayment pays the loan with its Ref # first, ignoring case: %+v", second)
	}
	want := []LoanRepayment{
		{TransactionID: "row-4", Date: "2026-10-10", Amount: 500_000},
		{TransactionID: "row-5", Date: "2026-10-12", Amount: 100_000},
		{TransactionID: "row-6", Date: "2026-10-14", Amount: 100_000},
	}
	if first.Outstanding != 300_000 || !reflect.DeepEqual(first.Repayments, want) {
		t.Fatalf("what is left, and repayments with another Ref #, go to the oldest loan: %+v", first)
	}
	if *first.RepayBy != "2026-12-15" {
		t.Fatalf("a dated memo moves the first loan the repayment would pay that is still open: %+v", first)
	}
}

func TestGivenMoneyRepaidAheadWithARefWhenALoanWithThatRefFollowsThenItTakesThatMoneyFirst(t *testing.T) {
	transactions := []Transaction{
		refTxn(lendingTxn("row-2", "2026-09-01", "Person A", 0, 300_000), "OTHER"),
		refTxn(lendingTxn("row-3", "2026-09-02", "Person A", 0, 200_000), "L5"),
		refTxn(lendingTxn("row-4", "2026-10-01", "Person A", 1_000_000, 0), "L5"),
	}

	loan := ComputeMoneyLent(transactions, day(2026, 10, 23)).People[0].Loans[0]

	want := []LoanRepayment{{TransactionID: "row-3", Date: "2026-09-02", Amount: 200_000}, {TransactionID: "row-2", Date: "2026-09-01", Amount: 300_000}}
	if loan.Outstanding != 500_000 || !reflect.DeepEqual(loan.Repayments, want) {
		t.Fatalf("loan = %+v", loan)
	}
}

func lentPerson(name string, outstanding float64, daysOutstanding int, repayBy string, daysUntil int) MoneyLentPerson {
	return owingPerson(name, openLoan(outstanding, daysOutstanding, repayBy, daysUntil))
}

func openLoan(outstanding float64, daysOutstanding int, repayBy string, daysUntil int) LentLoan {
	loan := LentLoan{Amount: outstanding, Outstanding: outstanding, DaysOutstanding: daysOutstanding}
	if repayBy != "" {
		loan.RepayBy, loan.DaysUntilRepay = &repayBy, &daysUntil
	}
	return loan
}

func owingPerson(name string, loans ...LentLoan) MoneyLentPerson {
	person := MoneyLentPerson{Name: name, Loans: loans}
	oldest := 0
	for _, loan := range loans {
		person.Outstanding += loan.Outstanding
		oldest = max(oldest, loan.DaysOutstanding)
	}
	person.DaysOutstanding = &oldest
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
