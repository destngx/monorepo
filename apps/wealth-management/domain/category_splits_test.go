package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSplitTagsTrimsLowersAndDropsRepeats(t *testing.T) {
	got := ParseSplitTags(" Eating-Out, ,takeaway,eating-out ")
	if strings.Join(got, "|") != "eating-out|takeaway" {
		t.Fatalf("tags = %q", got)
	}
	if got := ParseSplitTags(""); len(got) != 0 {
		t.Fatalf("empty cell gave %q", got)
	}
}

func TestCategoryDefinitionUpdateValidates(t *testing.T) {
	long := strings.Repeat("x", maxCategoryDefinitionRunes+1)
	cases := []struct {
		name  string
		input CategoryDefinitionUpdate
		ok    bool
	}{
		{"definition and tags", CategoryDefinitionUpdate{Category: "Groceries", Definition: "Food for home.", SplitTags: []string{"Eating-Out"}}, true},
		{"clearing both", CategoryDefinitionUpdate{Category: "Groceries"}, true},
		{"no category", CategoryDefinitionUpdate{Definition: "x"}, false},
		{"definition too long", CategoryDefinitionUpdate{Category: "Groceries", Definition: long}, false},
		{"tag with a comma", CategoryDefinitionUpdate{Category: "Groceries", SplitTags: []string{"a,b"}}, false},
		{"tag too long", CategoryDefinitionUpdate{Category: "Groceries", SplitTags: []string{strings.Repeat("t", maxSplitTagRunes+1)}}, false},
		{"too many tags", CategoryDefinitionUpdate{Category: "Groceries", SplitTags: []string{"a", "b", "c", "d", "e", "f"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.ok != (err == nil) || (err != nil && !errors.Is(err, ErrValidation)) {
				t.Fatalf("err = %v, want ok=%v", err, tc.ok)
			}
		})
	}
	tags := CategoryDefinitionUpdate{Category: "Groceries", SplitTags: []string{" Eating-Out ", "", "eating-out"}}.NormalizedSplitTags()
	if strings.Join(tags, "|") != "eating-out" {
		t.Fatalf("normalized = %q", tags)
	}
}

func tagged(t Transaction, tags ...string) Transaction {
	t.Tags = tags
	return t
}

func TestApplySplitsSharesACategoryMonthByItsSplitTags(t *testing.T) {
	// Given a groceries category split by eating-out, with a cashback tag that is not a split
	categories := []Category{{Name: "Groceries", Type: CategoryTypeExpense, SplitTags: []string{"eating-out"}}, {Name: "Going out", Type: CategoryTypeExpense}}
	transactions := []Transaction{
		txn(day(2026, 10, 2), "Groceries", CategoryTypeExpense, 1_000_000, 0),
		tagged(txn(day(2026, 10, 3), "groceries", CategoryTypeExpense, 300_000, 0), "Eating-Out", "card-cashback"),
		tagged(txn(day(2026, 10, 4), "Groceries", CategoryTypeExpense, 100_000, 0), "eating-out"),
		// A refund on a tagged meal lowers the split; last month and other categories do not count
		tagged(txn(day(2026, 10, 5), "Groceries", CategoryTypeExpense, 0, 50_000), "eating-out"),
		tagged(txn(day(2026, 9, 5), "Groceries", CategoryTypeExpense, 999_000, 0), "eating-out"),
		tagged(txn(day(2026, 10, 5), "Going out", CategoryTypeExpense, 500_000, 0), "eating-out"),
	}
	items := []BudgetItem{{Category: "Groceries", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 2_000_000}, {Category: "Going out", CategoryType: categoryType(CategoryTypeExpense), MonthlyLimit: 1_000_000}}
	report := ComputeBudgetReport(items, transactions, day(2026, 10, 1), day(2026, 10, 19))

	ApplySplits(&report, categories, transactions)

	var groceries, goingOut BudgetLine
	for _, line := range report.Lines {
		switch line.Category {
		case "Groceries":
			groceries = line
		case "Going out":
			goingOut = line
		}
	}
	// Then the split is the tagged share of the category's spend
	if len(groceries.Splits) != 1 || groceries.Splits[0] != (SplitSpend{Tag: "eating-out", Spent: 350_000, PercentOfSpent: 25.93}) {
		t.Fatalf("groceries splits = %+v (spent %v)", groceries.Splits, groceries.Spent)
	}
	// And a category without split tags has none
	if len(goingOut.Splits) != 0 {
		t.Fatalf("going out splits = %+v", goingOut.Splits)
	}
}

func TestCategoryGuideListsOnlyCategoriesWithADefinitionOrSplitTags(t *testing.T) {
	categories := []Category{
		{Name: "Groceries", Definition: "Food bought to eat at home, and everyday meals out.", SplitTags: []string{"eating-out"}},
		{Name: "Going out", Definition: "Social outings with friends."},
		{Name: "Rent"},
	}
	guide := CategoryGuide(categories)
	want := "- Groceries: Food bought to eat at home, and everyday meals out. Split tags: eating-out.\n- Going out: Social outings with friends."
	if guide != want {
		t.Fatalf("guide =\n%s\nwant\n%s", guide, want)
	}
	if CategoryGuide([]Category{{Name: "Rent"}}) != "" {
		t.Fatal("no definitions should give an empty guide")
	}
}

func TestHistorySplitTagDecidesOnceThePayeeHasATaggedRowAndNeedsTheTagOnHalf(t *testing.T) {
	row := func(category string, tags ...string) Transaction {
		return Transaction{Payee: "Pho Corner", Category: category, Tags: tags}
	}
	tags := []string{"eating-out"}
	cases := []struct {
		name         string
		transactions []Transaction
		tag          string
		decided      bool
	}{
		{"no history", nil, "", false},
		{"other category only", []Transaction{row("Going out", "eating-out")}, "", false},
		{"history from before the tags existed", []Transaction{row("Groceries"), row("Groceries", "card-cashback")}, "", false},
		{"half tagged", []Transaction{row("Groceries", "Eating-Out"), row("groceries")}, "eating-out", true},
		{"rarely tagged", []Transaction{row("Groceries", "eating-out"), row("Groceries"), row("Groceries")}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tag, decided := HistorySplitTag(" pho corner ", "Groceries", tags, tc.transactions)
			if tag != tc.tag || decided != tc.decided {
				t.Fatalf("tag=%q decided=%v, want %q %v", tag, decided, tc.tag, tc.decided)
			}
		})
	}
}
