package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"errors"
	"slices"
	"testing"
	"time"
)

func categoryTabRows() [][]string {
	return [][]string{
		{"Categories"},
		{"Income Categories"},
		{"Salary"},
		{"Expense Categories"},
		{"Food", "essential"},
		{"Rent", "fixed"},
		{"Non-budget Categories"},
		{"Lending / Repayment", "Lending"},
		{"Transfer"},
	}
}

func TestGivenALendingCategoryWhenReadingTransactionsThenEachCarriesItsCategoryNature(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges[categoriesSheetRange] = categoryTabRows()
	client.ranges[transactionsSheetRange] = [][]string{
		{"Bank", "01/10/2026", "", "Person A", "", "", "lending / repayment", "", "5000000"},
		{"Bank", "02/10/2026", "", "Shop", "", "", "Food", "", "50000"},
	}
	svc := newTestDBService(t, client, newFakeCache())

	transactions, err := svc.ListTransactions(false)

	if err != nil {
		t.Fatal(err)
	}
	natures := map[string]domain.CategoryNature{}
	for _, transaction := range transactions {
		natures[transaction.Payee] = transaction.CategoryNature
	}
	if natures["Person A"] != domain.NatureLending || natures["Shop"] != domain.NatureEssential {
		t.Fatalf("natures = %v", natures)
	}
}

func TestGivenACategoryWhenSettingItsNatureThenColumnBOfItsRowIsWritten(t *testing.T) {
	cases := []struct {
		name  string
		input domain.CategoryNatureUpdate
		cell  string
		value string
	}{
		{"lending", domain.CategoryNatureUpdate{Category: "Transfer", Nature: "lending"}, "Categories!B9", "lending"},
		{"case-insensitive nature", domain.CategoryNatureUpdate{Category: " Rent ", Nature: "Flexible"}, "Categories!B6", "flexible"},
		{"none clears", domain.CategoryNatureUpdate{Category: "Food", Nature: "none"}, "Categories!B5", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeSheetClient()
			client.ranges[categoriesSheetRange] = categoryTabRows()
			cache := newFakeCache()
			svc := newTestDBService(t, client, cache)

			category, err := svc.UpdateCategoryNature(tc.input)

			if err != nil {
				t.Fatal(err)
			}
			if len(client.updates) != 1 || client.updates[0].rangeValue != tc.cell || client.updates[0].values[0] != tc.value {
				t.Fatalf("updates = %+v", client.updates)
			}
			if string(category.Nature) != tc.value {
				t.Fatalf("category = %+v", category)
			}
			if !slices.Contains(cache.invalidated, cacheKeyCats) || !slices.Contains(cache.invalidated, "transactions:*") {
				t.Fatalf("invalidated = %v", cache.invalidated)
			}
		})
	}
}

func TestGivenAMissingDuplicatedOrBadInputWhenSettingANatureThenNothingIsWritten(t *testing.T) {
	duplicated := append(categoryTabRows(), []string{"Food"})
	cases := []struct {
		name  string
		rows  [][]string
		input domain.CategoryNatureUpdate
		err   error
	}{
		{"missing", categoryTabRows(), domain.CategoryNatureUpdate{Category: "Travel", Nature: "fixed"}, domain.ErrNotFound},
		{"name differs in case", categoryTabRows(), domain.CategoryNatureUpdate{Category: "food", Nature: "fixed"}, domain.ErrNotFound},
		{"section header", categoryTabRows(), domain.CategoryNatureUpdate{Category: "Expense Categories", Nature: "fixed"}, domain.ErrNotFound},
		{"duplicated", duplicated, domain.CategoryNatureUpdate{Category: "Food", Nature: "fixed"}, domain.ErrConflict},
		{"unknown nature", categoryTabRows(), domain.CategoryNatureUpdate{Category: "Food", Nature: "loan"}, domain.ErrValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeSheetClient()
			client.ranges[categoriesSheetRange] = tc.rows
			svc := newTestDBService(t, client, newFakeCache())

			if _, err := svc.UpdateCategoryNature(tc.input); !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			if len(client.updates) != 0 {
				t.Fatalf("updates = %+v", client.updates)
			}
		})
	}
}

func TestGivenLendingTransactionsWhenReadingMoneyLentThenTheLedgerIsGroupedPerPerson(t *testing.T) {
	lent, repaid := 10_000_000.0, 4_000_000.0
	nonBudget := domain.CategoryTypeNonBudget
	row := func(id string, date time.Time, payment *float64, deposit *float64) domain.Transaction {
		return domain.Transaction{ID: id, AccountName: "Bank", Date: date, Payee: "Person A", Category: "Lending / Repayment", CategoryType: &nonBudget, CategoryNature: domain.NatureLending, Payment: payment, Deposit: deposit}
	}
	db := &fakes.Database{Transactions: []domain.Transaction{
		row("row-2", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), &lent, nil),
		row("row-3", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), nil, &repaid),
	}}

	ledger, err := newAnalytics(db).MoneyLent(false)

	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.People) != 1 || ledger.People[0].Outstanding != 6_000_000 || ledger.OwedToYou != 6_000_000 {
		t.Fatalf("ledger = %+v", ledger)
	}
	if _, err := newAnalytics(&fakes.Database{Err: errors.New("down")}).MoneyLent(false); err == nil {
		t.Fatal("a read failure must be returned")
	}
}

func TestGivenDefinitionAndSplitTagColumnsWhenReadingCategoriesThenTheyAreParsedAndTheNoteColumnIsIgnored(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges[categoriesSheetRange] = [][]string{
		{"Categories"},
		{"Expense Categories", "Type", "Note", "Definition", "Split tags"},
		{"Groceries", "essential", "a private note", " Food for home, and everyday meals out. ", "Eating-Out, ,takeaway"},
		{"Rent", "fixed"},
	}
	svc := newTestDBService(t, client, newFakeCache())

	categories, err := svc.ListCategories(false)

	if err != nil {
		t.Fatal(err)
	}
	groceries := categories[0]
	if groceries.Name != "Groceries" || groceries.Definition != "Food for home, and everyday meals out." || !slices.Equal(groceries.SplitTags, []string{"eating-out", "takeaway"}) {
		t.Fatalf("groceries = %+v", groceries)
	}
	if rent := categories[1]; rent.Definition != "" || len(rent.SplitTags) != 0 {
		t.Fatalf("rent = %+v", rent)
	}
}

func TestGivenACategoryWhenSettingItsDefinitionThenColumnsDAndEOfItsRowAreWritten(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges[categoriesSheetRange] = categoryTabRows()
	cache := newFakeCache()
	svc := newTestDBService(t, client, cache)

	category, err := svc.UpdateCategoryDefinition(domain.CategoryDefinitionUpdate{Category: "Food", Definition: " Food for home. ", SplitTags: []string{"Eating-Out", "eating-out", "takeaway"}})

	if err != nil {
		t.Fatal(err)
	}
	if len(client.updates) != 1 || client.updates[0].rangeValue != "Categories!D5" || !slices.Equal(client.updates[0].values, []any{"Food for home.", "eating-out,takeaway"}) {
		t.Fatalf("updates = %+v", client.updates)
	}
	if category.Definition != "Food for home." || !slices.Equal(category.SplitTags, []string{"eating-out", "takeaway"}) || category.Nature != domain.NatureEssential {
		t.Fatalf("category = %+v", category)
	}
	if !slices.Contains(cache.invalidated, cacheKeyCats) {
		t.Fatalf("invalidated = %v", cache.invalidated)
	}
}

func TestGivenAMissingCategoryOrBadInputWhenSettingADefinitionThenNothingIsWritten(t *testing.T) {
	cases := []struct {
		name  string
		input domain.CategoryDefinitionUpdate
		err   error
	}{
		{"missing", domain.CategoryDefinitionUpdate{Category: "Travel", Definition: "x"}, domain.ErrNotFound},
		{"tag with a comma", domain.CategoryDefinitionUpdate{Category: "Food", SplitTags: []string{"a,b"}}, domain.ErrValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeSheetClient()
			client.ranges[categoriesSheetRange] = categoryTabRows()
			svc := newTestDBService(t, client, newFakeCache())

			if _, err := svc.UpdateCategoryDefinition(tc.input); !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
			if len(client.updates) != 0 {
				t.Fatalf("updates = %+v", client.updates)
			}
		})
	}
}
