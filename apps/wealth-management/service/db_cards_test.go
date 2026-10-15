package service

import (
	"apps/wealth-management/domain"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func cardsFixture() *fakeSheetClient {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts", programsSheet, programRulesSheet, banksSheet, cardsSheet, merchantsSheet}
	client.ranges[programsRange] = [][]string{
		{"Gold", "Acme", "01/01/2026", "599,000", "300000", "Cycle"},
		{},
		{"Gold", "Acme", "45658", "499000", "", "cycle"}, // 2025-01-01, listed after the newer version
		{"Silver", "Acme", "01/01/2025", "", "", "month"},
	}
	client.ranges[programRulesRange] = [][]string{
		{"gold", "01/01/2026", "Groceries", "5411, 5499", "5", "200000", "100,000", "", "", "", " Grocery cap "},
		{"Gold", "46023", "Groceries tier 2", "5411", "10", "200000", "100000", "5000000", "", "", "Grocery cap"}, // 2026-01-01
		{"Gold", "01/01/2026", "Utilities", "4900", "", "", "", "", "", "x"},
		{"Gold", "01/01/2026", "Others", "", "0.5", "", "", "", "x"},
		{"", "", "", ""},
		{"Gold", "01/01/2025", "Others", "", "1", "", "", "", "yes"},
	}
	client.ranges[banksRange] = [][]string{
		{"Acme", "Credit Card Acme", "20,000,000"},
	}
	client.ranges[cardsRange] = [][]string{
		{"Gold Card", "gold", "gold", "47331", "5", "30", "", "", "123"}, // 0123 typed as a number
		{"Silver Card", "Silver", "silver", "09/2030", "5", "30", "x", "01/10/2026", " 4567 "},
	}
	client.ranges[merchantsRange] = [][]string{
		{"Coop Mart", "5411", "", "Confirmed", "seen on the receipt"},
		{},
		{"EVN", "4900"},
		{"Coop Mart", "5499", "46143", "assumed"},
		{"Coop Mart", "5311", "01/06/2026"},
	}
	return client
}

func TestGivenTheCardTabsWhenListingCardConfigThenProgrammesBanksCardsAndMerchantsAreMapped(t *testing.T) {
	client := cardsFixture()
	config, err := newTestDBService(t, client, newFakeCache()).ListCardConfig(true)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(config.Programs) != 2 || config.Programs[0].Name != "Gold" || config.Programs[1].Name != "Silver" {
		t.Fatalf("programs: %+v", config.Programs)
	}
	gold := config.Programs[0]
	if gold.Bank != "Acme" || len(gold.Versions) != 2 {
		t.Fatalf("gold: %+v", gold)
	}
	older, current := gold.Versions[0], gold.Versions[1]
	if !older.From.Equal(date(2025, 1, 1)) || older.AnnualFee != 499_000 || older.Period != domain.CapPeriodCycle || len(older.Rules) != 1 {
		t.Fatalf("versions are sorted by start date: %+v", older)
	}
	if !current.From.Equal(date(2026, 1, 1)) || current.AnnualFee != 599_000 || current.Cap != 300_000 || current.Period != domain.CapPeriodCycle {
		t.Fatalf("current version: %+v", current)
	}
	groceries := domain.CashbackRule{Name: "Groceries", MCCs: []domain.MCCRange{{From: 5411, To: 5411}, {From: 5499, To: 5499}},
		Rate: 5, Cap: 200_000, MinTransaction: 100_000, CapGroup: "Grocery cap"}
	if len(current.Rules) != 4 || !reflect.DeepEqual(current.Rules[0], groceries) || current.Rules[1].MinCycleSpend != 5_000_000 ||
		!current.Rules[2].Exclude || !current.Rules[3].Fallback || current.Rules[3].MCCs == nil {
		t.Fatalf("rules: %+v", current.Rules)
	}
	if silver := config.Programs[1].Versions[0]; silver.Period != domain.CapPeriodMonth || silver.Rules == nil || len(silver.Rules) != 0 {
		t.Fatalf("a version without rules has an empty rule list: %+v", silver)
	}
	if !reflect.DeepEqual(config.Banks, []domain.Bank{{Name: "Acme", SharedAccount: "Credit Card Acme", ReferenceLimit: 20_000_000}}) {
		t.Fatalf("banks: %+v", config.Banks)
	}
	closed := date(2026, 10, 1)
	wantCards := []domain.Card{
		{Name: "Gold Card", Program: "Gold", Keyword: "gold", Expiry: "08/2029", StatementDay: 5, DueDay: 30, Last4: "0123"},
		{Name: "Silver Card", Program: "Silver", Keyword: "silver", Expiry: "09/2030", StatementDay: 5, DueDay: 30, Default: true, ClosedFrom: &closed, Last4: "4567"},
	}
	if !reflect.DeepEqual(config.Cards, wantCards) {
		t.Fatalf("cards: %+v", config.Cards)
	}
	may, june := date(2026, 5, 1), date(2026, 6, 1)
	wantMerchants := []domain.MerchantMCC{
		{Pattern: "Coop Mart", MCC: 5411, Confidence: domain.MCCConfirmed, Note: "seen on the receipt"},
		{Pattern: "EVN", MCC: 4900, Confidence: domain.MCCAssumed},
		{Pattern: "Coop Mart", MCC: 5499, From: &may, Confidence: domain.MCCAssumed},
		{Pattern: "Coop Mart", MCC: 5311, From: &june, Confidence: domain.MCCAssumed},
	}
	if !reflect.DeepEqual(config.Merchants, wantMerchants) {
		t.Fatalf("merchants: %+v", config.Merchants)
	}
	if client.requests != 1 {
		t.Fatalf("the tabs should be read in one request, got %d", client.requests)
	}
}

func TestGivenNoCardTabsWhenListingCardConfigThenNothingIsConfigured(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{"Accounts"}
	config, err := newTestDBService(t, client, newFakeCache()).ListCardConfig(true)
	if err != nil || config.Programs == nil || config.Banks == nil || config.Cards == nil || config.Merchants == nil || len(config.Cards) != 0 {
		t.Fatalf("missing tabs should mean empty lists: %+v %v", config, err)
	}
	if len(client.reads) != 0 {
		t.Fatalf("must not read missing tabs: %v", client.reads)
	}
}

func TestGivenSomeCardTabsMissingWhenListingCardConfigThenOnlyPresentTabsAreRead(t *testing.T) {
	client := cardsFixture()
	client.titles = []string{programsSheet, programRulesSheet, merchantsSheet}
	config, err := newTestDBService(t, client, newFakeCache()).ListCardConfig(true)
	if err != nil || len(config.Programs) != 2 || len(config.Cards) != 0 || len(config.Banks) != 0 || len(config.Merchants) != 4 {
		t.Fatalf("config: %+v %v", config, err)
	}
	for _, read := range client.reads {
		if read == cardsRange || read == banksRange {
			t.Fatal("must not read a tab that does not exist")
		}
	}
}

// cardTabRows is a valid minimal configuration that each case breaks.
type cardTabRows struct {
	programs, rules, banks, cards, merchants [][]string
}

func validCardTabs() cardTabRows {
	return cardTabRows{
		programs: [][]string{{"Gold", "Acme", "01/01/2026", "", "", "cycle"}},
		banks:    [][]string{{"Acme", "Credit Card Acme", ""}},
		cards:    [][]string{{"Gold Card", "Gold", "gold", "", "5", "30"}},
	}
}

func TestGivenInvalidCardRowsWhenListingCardConfigThenTheErrorNamesEachRow(t *testing.T) {
	tests := map[string]struct {
		change func(*cardTabRows)
		want   []string
	}{
		"programme without a bank": {
			change: func(r *cardTabRows) { r.programs[0][1] = "" },
			want:   []string{"Programs row 2", "bank is required"},
		},
		"programme without a start date": {
			change: func(r *cardTabRows) { r.programs[0][2] = "" },
			want:   []string{"Programs row 2", "Version from is required"},
		},
		"programme start that is not a date": {
			change: func(r *cardTabRows) { r.programs[0][2] = "soon" },
			want:   []string{"Programs row 2", "not a date"},
		},
		"unknown cap period": {
			change: func(r *cardTabRows) { r.programs[0][5] = "week" },
			want:   []string{"Programs row 2", "cap period"},
		},
		"missing cap period": {
			change: func(r *cardTabRows) { r.programs[0][5] = "" },
			want:   []string{"Programs row 2", "cap period"},
		},
		"bad annual fee": {
			change: func(r *cardTabRows) { r.programs[0][3] = "lots" },
			want:   []string{"Programs row 2", "Annual fee"},
		},
		"two versions from the same date": {
			change: func(r *cardTabRows) {
				r.programs = append(r.programs, []string{"gold", "Acme", "46023", "", "", "month"})
			},
			want: []string{"Programs row 3", "row 2"},
		},
		"programme moving bank": {
			change: func(r *cardTabRows) {
				r.programs = append(r.programs, []string{"Gold", "Beta", "01/01/2027", "", "", "cycle"})
			},
			want: []string{"Programs row 3", "Beta"},
		},
		"rule for an unknown version": {
			change: func(r *cardTabRows) { r.rules = [][]string{{"Gold", "01/02/2026", "Others", "", "1", "", "", "", "x"}} },
			want:   []string{"ProgramRules row 2", "matches no row"},
		},
		"rule without mccs": {
			change: func(r *cardTabRows) { r.rules = [][]string{{"Gold", "01/01/2026", "Online", "", "5"}} },
			want:   []string{"ProgramRules row 2", "MCCs are required"},
		},
		"fallback with mccs": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "Others", "5411", "1", "", "", "", "x"}}
			},
			want: []string{"ProgramRules row 2", "fallback rule takes no MCCs"},
		},
		"negative rate": {
			change: func(r *cardTabRows) { r.rules = [][]string{{"Gold", "01/01/2026", "Online", "5311", "-1"}} },
			want:   []string{"ProgramRules row 2", "rate must not be negative"},
		},
		"bad mcc list": {
			change: func(r *cardTabRows) { r.rules = [][]string{{"Gold", "01/01/2026", "Groceries", "54x1", "1"}} },
			want:   []string{"ProgramRules row 2", "MCCs"},
		},
		"bad min transaction": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "Groceries", "5411", "1", "", "lots"}}
			},
			want: []string{"ProgramRules row 2", "Min transaction"},
		},
		"exclude with a rate": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "Utilities", "4900", "1", "", "", "", "", "x"}}
			},
			want: []string{"ProgramRules row 2", "earns nothing"},
		},
		"duplicate rule name": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "Online", "5311", "1"}, {"Gold", "01/01/2026", "online", "5399", "2"}}
			},
			want: []string{"ProgramRules row 3", "rule named", "row 2"},
		},
		"two fallbacks with the same minimums": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "A", "", "1", "", "", "", "x"}, {"Gold", "01/01/2026", "B", "", "2", "", "", "", "x"}}
			},
			want: []string{"ProgramRules row 3", "fallback", "row 2"},
		},
		"cap group without a rule cap": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "Online", "5311", "5", "", "", "", "", "", "Shared"}}
			},
			want: []string{"ProgramRules row 2", "cap group needs a rule cap"},
		},
		"exclude in a cap group": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{{"Gold", "01/01/2026", "Utilities", "4900", "", "", "", "", "", "x", "Shared"}}
			},
			want: []string{"ProgramRules row 2", "earns nothing"},
		},
		"cap group with different caps": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{
					{"Gold", "01/01/2026", "Online", "5311", "5", "100000", "", "", "", "", "Shared"},
					{"Gold", "01/01/2026", "Dining", "5812", "3", "200000", "", "", "", "", "shared"},
				}
			},
			want: []string{"ProgramRules row 3", "cap group", "row 2"},
		},
		"cap group named like a rule": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{
					{"Gold", "01/01/2026", "Online", "5311", "5"},
					{"Gold", "01/01/2026", "Dining", "5812", "3", "200000", "", "", "", "", "online"},
				}
			},
			want: []string{"ProgramRules row 3", "cap group", "same name as the rule on row 2"},
		},
		"rule named like a cap group": {
			change: func(r *cardTabRows) {
				r.rules = [][]string{
					{"Gold", "01/01/2026", "Dining", "5812", "3", "200000", "", "", "", "", "Shared"},
					{"Gold", "01/01/2026", "Shared", "5311", "5"},
				}
			},
			want: []string{"ProgramRules row 3", "cap group", "row 2"},
		},
		"bank without a name": {
			change: func(r *cardTabRows) { r.banks = append(r.banks, []string{"", "Card B", "1"}) },
			want:   []string{"Banks row 3", "bank is required"},
		},
		"duplicate bank": {
			change: func(r *cardTabRows) { r.banks = append(r.banks, []string{"ACME", "Card B", ""}) },
			want:   []string{"Banks row 3", "row 2"},
		},
		"negative reference limit": {
			change: func(r *cardTabRows) { r.banks[0][2] = "-1" },
			want:   []string{"Banks row 2", "reference limit"},
		},
		"card for an unknown programme": {
			change: func(r *cardTabRows) { r.cards[0][1] = "Bronze" },
			want:   []string{"Cards row 2", "Bronze", "matches no row"},
		},
		"card whose bank has no row": {
			change: func(r *cardTabRows) { r.banks = nil },
			want:   []string{"Cards row 2", "Acme", "Banks"},
		},
		"day out of range": {
			change: func(r *cardTabRows) { r.cards[0][5] = "32" },
			want:   []string{"Cards row 2", "due day"},
		},
		"fractional day": {
			change: func(r *cardTabRows) { r.cards[0][4] = "5.5" },
			want:   []string{"Cards row 2", "Statement day"},
		},
		"closed from not a date": {
			change: func(r *cardTabRows) { r.cards[0] = append(r.cards[0][:6:6], "", "soon") },
			want:   []string{"Cards row 2", "Closed from", "soon"},
		},
		"last 4 digits too long": {
			change: func(r *cardTabRows) { r.cards[0] = append(r.cards[0][:6:6], "", "", "12345") },
			want:   []string{"Cards row 2", "last 4 digits", "12345"},
		},
		"last 4 digits not digits": {
			change: func(r *cardTabRows) { r.cards[0] = append(r.cards[0][:6:6], "", "", "12ab") },
			want:   []string{"Cards row 2", "last 4 digits", "12ab"},
		},
		"missing keyword": {
			change: func(r *cardTabRows) { r.cards[0][2] = "" },
			want:   []string{"Cards row 2", "keyword is required"},
		},
		"duplicate keyword": {
			change: func(r *cardTabRows) { r.cards = append(r.cards, []string{"Other Card", "Gold", "GOLD", "", "5", "30"}) },
			want:   []string{"Cards row 3", "keyword", "row 2"},
		},
		"duplicate card name": {
			change: func(r *cardTabRows) { r.cards = append(r.cards, []string{"gold card", "Gold", "other", "", "5", "30"}) },
			want:   []string{"Cards row 3", "card name", "row 2"},
		},
		"two default cards in a bank": {
			change: func(r *cardTabRows) {
				r.cards[0] = append(r.cards[0], "x")
				r.cards = append(r.cards, []string{"Other Card", "Gold", "other", "", "5", "30", "yes"})
			},
			want: []string{"Cards row 3", "default card", "row 2"},
		},
		"merchant without mcc": {
			change: func(r *cardTabRows) { r.merchants = [][]string{{"Coop", ""}} },
			want:   []string{"Merchants row 2", "MCC"},
		},
		"merchant with a range": {
			change: func(r *cardTabRows) { r.merchants = [][]string{{"Coop", "5411-5412"}} },
			want:   []string{"Merchants row 2", "one MCC"},
		},
		"unknown confidence": {
			change: func(r *cardTabRows) { r.merchants = [][]string{{"Coop", "5411", "", "maybe"}} },
			want:   []string{"Merchants row 2", "confidence", "maybe"},
		},
		"duplicate merchant pattern and date": {
			change: func(r *cardTabRows) {
				r.merchants = [][]string{{"Coop", "5411", "01/05/2026"}, {"COOP", "5499", "46143"}}
			},
			want: []string{"Merchants row 3", "row 2"},
		},
		"duplicate undated merchant pattern": {
			change: func(r *cardTabRows) { r.merchants = [][]string{{"Coop", "5411"}, {"COOP", "5499"}} },
			want:   []string{"Merchants row 3", "row 2"},
		},
		"bad merchant date": {
			change: func(r *cardTabRows) { r.merchants = [][]string{{"Coop", "5411", "soon"}} },
			want:   []string{"Merchants row 2", "From"},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rows := validCardTabs()
			test.change(&rows)
			client := newFakeSheetClient()
			client.titles = []string{programsSheet, programRulesSheet, banksSheet, cardsSheet, merchantsSheet}
			client.ranges[programsRange] = rows.programs
			client.ranges[programRulesRange] = rows.rules
			client.ranges[banksRange] = rows.banks
			client.ranges[cardsRange] = rows.cards
			client.ranges[merchantsRange] = rows.merchants
			_, err := newTestDBService(t, client, newFakeCache()).ListCardConfig(true)
			if !errors.Is(err, domain.ErrValidation) {
				t.Fatalf("expected a validation error, got %v", err)
			}
			for _, fragment := range test.want {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(fragment)) {
					t.Fatalf("error %q should mention %q", err, fragment)
				}
			}
		})
	}
}

func TestGivenTheValidMinimalTabsWhenListingCardConfigThenTheyLoad(t *testing.T) {
	rows := validCardTabs()
	client := newFakeSheetClient()
	client.titles = []string{programsSheet, banksSheet, cardsSheet}
	client.ranges[programsRange] = rows.programs
	client.ranges[banksRange] = rows.banks
	client.ranges[cardsRange] = rows.cards
	if _, err := newTestDBService(t, client, newFakeCache()).ListCardConfig(true); err != nil {
		t.Fatalf("the base of the invalid-row cases must be valid: %v", err)
	}
}

func TestGivenACachedCardConfigWhenListingWithoutForceThenTheSheetIsNotRead(t *testing.T) {
	client := cardsFixture()
	svc := newTestDBService(t, client, newFakeCache())
	if _, err := svc.ListCardConfig(false); err != nil {
		t.Fatalf("first read: %v", err)
	}
	requests := client.requests
	config, err := svc.ListCardConfig(false)
	if err != nil || len(config.Cards) != 2 || client.requests != requests {
		t.Fatalf("second read should come from the cache: %+v %v (%d -> %d requests)", config, err, requests, client.requests)
	}
}

func TestCardConfigIsClearedWithTheOtherSheetCaches(t *testing.T) {
	for _, pattern := range sheetCachePatterns {
		if pattern == cacheKeyCards {
			return
		}
	}
	t.Fatalf("%s missing from sheetCachePatterns", cacheKeyCards)
}
