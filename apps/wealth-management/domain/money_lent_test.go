package domain

import (
	"errors"
	"testing"
	"time"
)

// lendingTxn is a transaction in a lending category: payment = money lent
// out, deposit = money repaid (WM-LEND-1).
func lendingTxn(id string, date string, payee string, payment float64, deposit float64) Transaction {
	parsed, err := time.Parse(time.DateOnly, date)
	if err != nil {
		panic(err)
	}
	t := txn(parsed, "Lending / Repayment", CategoryTypeNonBudget, payment, deposit)
	t.ID = id
	t.Payee = payee
	t.CategoryNature = NatureLending
	return t
}

func TestGivenANatureCellWhenParsingThenLendingIsRecognised(t *testing.T) {
	for cell, want := range map[string]CategoryNature{"lending": NatureLending, " Lending ": NatureLending, "LENDING": NatureLending, "loan": ""} {
		if got := ParseCategoryNature(cell); got != want {
			t.Fatalf("ParseCategoryNature(%q) = %q, want %q", cell, got, want)
		}
	}
}

func TestGivenLendingTransactionsWhenComputingMoneyLentThenEachPersonHasLentRepaidAndOutstanding(t *testing.T) {
	today := day(2026, 10, 17)
	transactions := []Transaction{
		lendingTxn("row-2", "2026-01-10", "person a", 10_000_000, 0),
		lendingTxn("row-3", "2026-03-01", "Person A", 0, 4_000_000),
		lendingTxn("row-4", "2026-09-17", " Person A ", 2_000_000, 0),
		lendingTxn("row-5", "2026-02-01", "Person B", 5_000_000, 0),
		lendingTxn("row-6", "2026-04-01", "Person B", 0, 5_000_000),
		lendingTxn("row-7", "2026-06-01", "Person C", 0, 3_000_000),
		txn(day(2026, 5, 1), "Food", CategoryTypeExpense, 1_000_000, 0),
	}

	lent := ComputeMoneyLent(transactions, today)

	if len(lent.People) != 3 {
		t.Fatalf("people = %+v", lent.People)
	}
	a := lent.People[0]
	if a.Name != "Person A" || a.Lent != 12_000_000 || a.Repaid != 4_000_000 || a.Outstanding != 8_000_000 {
		t.Fatalf("person A = %+v", a)
	}
	if a.LastActivity != "2026-09-17" || a.OpenSince == nil || *a.OpenSince != "2026-01-10" {
		t.Fatalf("person A dates = %+v", a)
	}
	if a.DaysOutstanding == nil || *a.DaysOutstanding != 280 {
		t.Fatalf("person A days outstanding = %v", a.DaysOutstanding)
	}
	if len(a.Transactions) != 3 || a.Transactions[0].ID != "row-4" {
		t.Fatalf("person A transactions should be newest first: %+v", a.Transactions)
	}
	c := lent.People[1]
	if c.Name != "Person C" || c.Outstanding != -3_000_000 || c.OpenSince == nil || *c.OpenSince != "2026-06-01" {
		t.Fatalf("a deposit before any lending is money the user owes: %+v", c)
	}
	b := lent.People[2]
	if b.Name != "Person B" || b.Outstanding != 0 || b.OpenSince != nil || b.DaysOutstanding != nil {
		t.Fatalf("settled person = %+v", b)
	}
	if lent.OwedToYou != 8_000_000 || lent.YouOwe != 3_000_000 || lent.PeopleOwing != 1 {
		t.Fatalf("totals = %+v", lent.MoneyLentFacts)
	}
	if lent.OldestDaysOutstanding == nil || *lent.OldestDaysOutstanding != 280 {
		t.Fatalf("oldest days outstanding = %v", lent.OldestDaysOutstanding)
	}
	if lent.AsOf != "2026-10-17" {
		t.Fatalf("as of = %q", lent.AsOf)
	}
}

func TestGivenABalanceThatWasSettledWhenLendingAgainThenTheOpenBalanceStartsAtTheNewLoan(t *testing.T) {
	transactions := []Transaction{
		lendingTxn("row-2", "2026-01-01", "Person A", 1_000_000, 0),
		lendingTxn("row-3", "2026-02-01", "Person A", 0, 1_000_000),
		lendingTxn("row-4", "2026-05-01", "Person A", 3_000_000, 0),
		lendingTxn("row-5", "2026-06-01", "Person A", 0, 1_000_000),
	}

	person := ComputeMoneyLent(transactions, day(2026, 6, 11)).People[0]

	if person.Outstanding != 2_000_000 || person.OpenSince == nil || *person.OpenSince != "2026-05-01" || *person.DaysOutstanding != 41 {
		t.Fatalf("person = %+v", person)
	}
}

func TestGivenNoLendingTransactionsWhenComputingMoneyLentThenItIsEmpty(t *testing.T) {
	lent := ComputeMoneyLent([]Transaction{txn(day(2026, 5, 1), "Food", CategoryTypeExpense, 1_000_000, 0)}, day(2026, 6, 1))
	if len(lent.People) != 0 || lent.People == nil || lent.OwedToYou != 0 || lent.OldestDaysOutstanding != nil {
		t.Fatalf("lent = %+v", lent)
	}
}

func TestGivenRowsWithIssuesWhenComputingMoneyLentThenTheyAreIgnored(t *testing.T) {
	broken := lendingTxn("row-2", "2026-01-01", "Person A", 1_000_000, 0)
	broken.Issues = []string{"unparseable date"}
	if lent := ComputeMoneyLent([]Transaction{broken}, day(2026, 6, 1)); len(lent.People) != 0 {
		t.Fatalf("lent = %+v", lent)
	}
}

func TestGivenLendingTransactionsWhenComputingCashFlowThenTheyAreNeitherIncomeNorExpense(t *testing.T) {
	month := day(2026, 3, 1)
	lentOut := lendingTxn("row-2", "2026-03-02", "Person A", 5_000_000, 0)
	lentOut.CategoryType = categoryType(CategoryTypeExpense)
	repaid := lendingTxn("row-3", "2026-03-03", "Person A", 0, 2_000_000)
	repaid.CategoryType = categoryType(CategoryTypeIncome)
	transactions := []Transaction{
		lentOut, repaid,
		txn(day(2026, 3, 5), "Salary", CategoryTypeIncome, 0, 10_000_000),
		txn(day(2026, 3, 6), "Food", CategoryTypeExpense, 4_000_000, 0),
	}

	flow := ComputeCashFlow(transactions, month)

	if flow.Income != 10_000_000 || flow.Expense != 4_000_000 || flow.SavingsRate != 60 {
		t.Fatalf("flow = %+v", flow)
	}
	if lentOut.CountsAsExpense() || repaid.CountsAsIncome() {
		t.Fatal("lending transactions must not count as income or expense")
	}
}

func TestGivenMoneyLentWhenComputingTheBalanceSheetThenAmountsOwedToYouAreAssetsAndAmountsYouOweAreLiabilities(t *testing.T) {
	accounts := []Account{{Name: "Bank", Balance: 10_000_000}, {Name: "Card", Balance: -2_000_000}}
	transactions := []Transaction{
		lendingTxn("row-2", "2026-01-01", "Person A", 3_000_000, 0),
		lendingTxn("row-3", "2026-01-02", "Person B", 0, 1_000_000),
		lendingTxn("row-4", "2026-01-03", "Person C", 500_000, 500_000),
	}

	sheet := ComputeBalanceSheet(accounts, transactions)

	want := BalanceSheet{Assets: 13_000_000, Liabilities: 3_000_000, AccountsNet: 8_000_000, Receivables: 3_000_000, Payables: 1_000_000, NetWorth: 10_000_000}
	if sheet != want {
		t.Fatalf("got %+v want %+v", sheet, want)
	}
}

func TestGivenALendingCategoryWhenComputingTheBudgetThenItIsLeftOutOfTheExpenseTotals(t *testing.T) {
	month := day(2026, 3, 1)
	lentOut := lendingTxn("row-2", "2026-03-02", "Person A", 5_000_000, 0)
	lentOut.CategoryType = categoryType(CategoryTypeExpense)
	items := []BudgetItem{
		{Category: "Food", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 4_000_000},
		{Category: "Lending / Repayment", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 1_000_000},
	}
	transactions := []Transaction{lentOut, txn(day(2026, 3, 6), "Food", CategoryTypeExpense, 2_000_000, 0)}

	report := ComputeBudgetReport(items, transactions, month, day(2026, 3, 10))

	if report.TotalSpent != 2_000_000 || report.TotalLimit != 4_000_000 {
		t.Fatalf("totals = %v / %v", report.TotalSpent, report.TotalLimit)
	}
	for _, line := range report.Lines {
		if line.Category == "Lending / Repayment" && (!line.Lending || line.CountsAsExpense() || line.BurnOutDate != nil) {
			t.Fatalf("lending line = %+v", line)
		}
	}
	history := ComputeBudgetHistory(map[int][]BudgetItem{2026: items}, transactions, []time.Time{month}, day(2026, 3, 10))
	for _, row := range history.Rows {
		if row.Category == "Lending / Repayment" {
			t.Fatalf("lending in budget history: %+v", row)
		}
	}
}

func TestGivenUnbudgetedLendingWhenComputingTheBudgetThenNoUnbudgetedLineAppears(t *testing.T) {
	lentOut := lendingTxn("row-2", "2026-03-02", "Person A", 5_000_000, 0)
	lentOut.CategoryType = categoryType(CategoryTypeExpense)

	report := ComputeBudgetReport(nil, []Transaction{lentOut}, day(2026, 3, 1), day(2026, 3, 10))

	if len(report.Lines) != 0 || report.TotalSpent != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestGivenLendingWhenComputingTheYearlyCashFlowThenItIsNotInCategoriesSpikesOrNeeds(t *testing.T) {
	transactions := []Transaction{}
	for month := time.January; month <= time.June; month++ {
		transactions = append(transactions,
			txn(day(2026, month, 5), "Salary", CategoryTypeIncome, 0, 20_000_000),
			txn(day(2026, month, 6), "Food", CategoryTypeExpense, 5_000_000, 0),
		)
	}
	big := lendingTxn("row-2", "2026-04-07", "Person A", 50_000_000, 0)
	big.CategoryType = categoryType(CategoryTypeExpense)
	repaid := lendingTxn("row-3", "2026-05-07", "Person A", 0, 50_000_000)
	repaid.CategoryType = categoryType(CategoryTypeIncome)
	transactions = append(transactions, big, repaid)
	categories := []Category{{Name: "Food", Type: CategoryTypeExpense, Nature: NatureEssential}, {Name: "Lending / Repayment", Type: CategoryTypeExpense, Nature: NatureLending}}

	flow := ComputeYearlyCashFlow(transactions, categories, 2026, day(2026, 7, 15))

	if flow.Total.Income != 120_000_000 || flow.Total.Expense != 30_000_000 {
		t.Fatalf("total = %+v", flow.Total)
	}
	if len(flow.Spikes) != 0 || len(flow.Expense) != 1 || len(flow.Income) != 1 {
		t.Fatalf("spikes %+v expense %+v income %+v", flow.Spikes, flow.Expense, flow.Income)
	}
	if flow.ExpenseSplit.Untagged.Total != 0 || len(flow.ExpenseSplit.UntaggedCategories) != 0 {
		t.Fatalf("split = %+v", flow.ExpenseSplit)
	}
}

func TestGivenANatureUpdateWhenValidatingThenKnownNaturesAndNonePass(t *testing.T) {
	for _, nature := range []string{"fixed", "Essential", "flexible", "lending", "none", ""} {
		if err := (CategoryNatureUpdate{Category: "Food", Nature: nature}).Validate(); err != nil {
			t.Fatalf("%q: %v", nature, err)
		}
	}
	if nature, ok := (CategoryNatureUpdate{Category: "Food", Nature: "None"}).ParsedNature(); !ok || nature != "" {
		t.Fatalf("none should clear, got %q %v", nature, ok)
	}
	for _, input := range []CategoryNatureUpdate{{Category: "Food", Nature: "loan"}, {Category: " ", Nature: "fixed"}} {
		if err := input.Validate(); !errors.Is(err, ErrValidation) {
			t.Fatalf("%+v: want validation error, got %v", input, err)
		}
	}
}

func TestGivenLendingThisMonthWhenComparingAndPacingThenItIsNotExpense(t *testing.T) {
	today := day(2026, 10, 12)
	lentOut := lendingTxn("row-2", "2026-10-03", "Person A", 9_000_000, 0)
	lentOut.CategoryType = categoryType(CategoryTypeExpense)
	transactions := []Transaction{lentOut, txn(day(2026, 10, 4), "Food", CategoryTypeExpense, 1_200_000, 0)}

	pace := ComputeExpensePace(transactions, nil, today)
	toDay := CashFlowToDay(transactions, day(2026, 10, 1), today.Day())

	if pace.Spent != 1_200_000 || toDay.Expense != 1_200_000 {
		t.Fatalf("pace spent %v, month to date %v", pace.Spent, toDay.Expense)
	}
}
