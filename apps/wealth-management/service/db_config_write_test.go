package service

import (
	"apps/wealth-management/domain"
	"errors"
	"reflect"
	"testing"
)

func fundsFixture() *fakeSheetClient {
	client := newFakeSheetClient()
	client.titles = []string{fundsSheet}
	client.ranges[fundsSheetRange] = [][]string{
		{"Fund certificates"},
		{},
		{"", "Index", "Total Unit", "Note", "History"},
		{"", "FUNDA", "100.5", "", "1-2-2026,100,5"},
		{"", "FUNDB", "20", "", "1-3-2026,20"},
		{},
		{"", "Total", "", "", ""},
	}
	return client
}

func updatesOf(client *fakeSheetClient) map[string][]any {
	out := map[string][]any{}
	for _, update := range client.updates {
		out[update.rangeValue] = update.values
	}
	return out
}

func TestGivenAHeldFundWhenRecordingATradeThenHistoryAndTotalUnitAreWrittenTogether(t *testing.T) {
	client := fundsFixture()
	result, err := newTestDBService(t, client, newFakeCache()).RecordFundTrade(domain.FundTradeInput{Code: " funda ", Date: "2026-10-01", Units: 10.25})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	want := map[string][]any{
		"'InvestmentFundCertificate'!C4": {110.75},
		"'InvestmentFundCertificate'!E4": {"1-2-2026,100,5;1-10-2026,10,25"},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
	if result.Units != 110.75 || result.Row != 4 || result.Created {
		t.Fatalf("result: %+v", result)
	}
}

func TestGivenAFundWhenSellingMoreThanHeldThenNothingIsWritten(t *testing.T) {
	client := fundsFixture()
	_, err := newTestDBService(t, client, newFakeCache()).RecordFundTrade(domain.FundTradeInput{Code: "FUNDB", Date: "2026-10-01", Units: -25})
	if !errors.Is(err, domain.ErrValidation) || len(client.updates) != 0 {
		t.Fatalf("err = %v, updates = %v", err, client.updates)
	}
}

func TestGivenANewFundCodeWhenBuyingThenItTakesTheFirstBlankRowAboveTheTotals(t *testing.T) {
	client := fundsFixture()
	result, err := newTestDBService(t, client, newFakeCache()).RecordFundTrade(domain.FundTradeInput{Code: "FUNDC", Date: "2026-10-01", Units: 5})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	want := map[string][]any{
		"'InvestmentFundCertificate'!B6": {"FUNDC"},
		"'InvestmentFundCertificate'!C6": {5.0},
		"'InvestmentFundCertificate'!E6": {"1-10-2026,5"},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) || !result.Created {
		t.Fatalf("updates = %v, result = %+v", got, result)
	}
}

func TestGivenAFundNotHeldWhenSellingThenTheTradeIsRefused(t *testing.T) {
	client := fundsFixture()
	_, err := newTestDBService(t, client, newFakeCache()).RecordFundTrade(domain.FundTradeInput{Code: "FUNDC", Date: "2026-10-01", Units: -5})
	if !errors.Is(err, domain.ErrValidation) || len(client.updates) != 0 {
		t.Fatalf("err = %v, updates = %v", err, client.updates)
	}
}

func TestGivenNoCryptoTabWhenSettingHoldingsThenTheTabIsCreatedAndRowsFollowTheHeader(t *testing.T) {
	client := newFakeSheetClient()
	client.ranges[cryptoSheetRange] = [][]string{{"Platform", "Token", "Amount", "Note"}}
	results, err := newTestDBService(t, client, newFakeCache()).SetCryptoHoldings(domain.CryptoHoldingsInput{Holdings: []domain.CryptoHoldingInput{
		{Platform: "Exchange A", Symbol: "btc", Quantity: 0.5, Note: "cold wallet"},
	}})
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if !reflect.DeepEqual(client.created, []string{cryptoSheet}) {
		t.Fatalf("created: %v", client.created)
	}
	want := map[string][]any{
		"Crypto!A1":   {"Platform", "Token", "Amount", "Note"},
		"'Crypto'!A2": {"Exchange A"},
		"'Crypto'!B2": {"BTC"},
		"'Crypto'!C2": {0.5},
		"'Crypto'!D2": {"cold wallet"},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
	if len(results) != 1 || results[0].Row != 2 || !results[0].Created {
		t.Fatalf("results: %+v", results)
	}
}

func TestGivenACryptoTabWithoutATableWhenSettingHoldingsThenAHeaderGoesBelowItsRows(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{cryptoSheet}
	client.ranges[cryptoSheetRange] = [][]string{{"Crypto"}, {"prices in USD"}}
	_, err := newTestDBService(t, client, newFakeCache()).SetCryptoHoldings(domain.CryptoHoldingsInput{Holdings: []domain.CryptoHoldingInput{
		{Platform: "Exchange A", Symbol: "ETH", Quantity: 2},
	}})
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	want := map[string][]any{
		"'Crypto'!A4": {"Platform", "Token", "Amount", "Note"},
		"'Crypto'!A5": {"Exchange A"},
		"'Crypto'!B5": {"ETH"},
		"'Crypto'!C5": {2.0},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
}

func TestGivenListedCoinsWhenSettingHoldingsThenQuantitiesUpdateAndNewCoinsAreAdded(t *testing.T) {
	client := newFakeSheetClient()
	client.titles = []string{cryptoSheet}
	client.ranges[cryptoSheetRange] = [][]string{
		{"Platform", "Token", "Amount"},
		{"Exchange A", "BTC", "0.1"},
		{"", "ETH", "1"}, // platform carried down
		{"Wallet B", "ETH", "3"},
	}
	results, err := newTestDBService(t, client, newFakeCache()).SetCryptoHoldings(domain.CryptoHoldingsInput{Holdings: []domain.CryptoHoldingInput{
		{Platform: "exchange a", Symbol: "eth", Quantity: 1.5},
		{Platform: "Exchange A", Symbol: "SOL", Quantity: 10},
	}})
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	want := map[string][]any{
		"'Crypto'!C3": {1.5},
		"'Crypto'!A5": {"Exchange A"},
		"'Crypto'!B5": {"SOL"},
		"'Crypto'!C5": {10.0},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
	if results[0].Row != 3 || results[0].Created || results[1].Row != 5 || !results[1].Created {
		t.Fatalf("results: %+v", results)
	}
}

func TestGivenACoinListedTwiceWhenSettingHoldingsThenNothingIsWritten(t *testing.T) {
	client := newFakeSheetClient()
	_, err := newTestDBService(t, client, newFakeCache()).SetCryptoHoldings(domain.CryptoHoldingsInput{Holdings: []domain.CryptoHoldingInput{
		{Platform: "Exchange A", Symbol: "BTC", Quantity: 1},
		{Platform: "exchange a", Symbol: "btc", Quantity: 2},
	}})
	if !errors.Is(err, domain.ErrValidation) || len(client.updates) != 0 {
		t.Fatalf("err = %v, updates = %v", err, client.updates)
	}
}

func cardWritesFixture() *fakeSheetClient {
	client := cardsFixture()
	client.ranges[accountsSheetRange] = [][]string{
		{"Cash", "Active Use"},
		{"Credit Card Acme", "Negative Active Use"},
	}
	return client
}

func TestGivenANewCardWithALimitWhenCreatingThenTheRowAndTheSharedLimitAreWritten(t *testing.T) {
	client := cardWritesFixture()
	limit := 30_000_000.0
	card, err := newTestDBService(t, client, newFakeCache()).CreateCard(domain.CardInput{
		Name: "Gold Plus", Program: "Gold", Keyword: "goldplus", Expiry: "08/29", StatementDay: 10, DueDay: 25, Last4: "0123", Limit: &limit,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	want := map[string][]any{
		"'Cards'!A4:I4": {"Gold Plus", "Gold", "goldplus", "'08/29", 10, 25, "", "", "'0123"},
		"Accounts!C3":   {limit},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
	if card.Name != "Gold Plus" || card.Last4 != "0123" {
		t.Fatalf("card: %+v", card)
	}
}

func TestGivenAnUnknownProgrammeWhenCreatingACardThenNothingIsWritten(t *testing.T) {
	client := cardWritesFixture()
	_, err := newTestDBService(t, client, newFakeCache()).CreateCard(domain.CardInput{Name: "Platinum", Program: "Platinum"})
	if !errors.Is(err, domain.ErrValidation) || len(client.updates) != 0 {
		t.Fatalf("err = %v, updates = %v", err, client.updates)
	}
}

func TestGivenACardWhenUpdatingThenItsRowIsReplacedAndAnUnknownCardIsNotFound(t *testing.T) {
	client := cardWritesFixture()
	db := newTestDBService(t, client, newFakeCache())
	if _, err := db.UpdateCard("silver card", domain.CardInput{Name: "Silver Card", Program: "Silver", Keyword: "silver", StatementDay: 6, DueDay: 1}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := updatesOf(client)["'Cards'!A3:I3"]; !reflect.DeepEqual(got, []any{"Silver Card", "Silver", "silver", "", 6, 1, "", "", ""}) {
		t.Fatalf("row: %v", got)
	}
	if _, err := db.UpdateCard("Bronze Card", domain.CardInput{Name: "Bronze Card", Program: "Silver"}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown card: %v", err)
	}
}

func TestGivenAProgrammeVersionWhenCreatingThenItsRowAndRulesAreAppended(t *testing.T) {
	client := cardWritesFixture()
	program, err := newTestDBService(t, client, newFakeCache()).CreateCardProgram(domain.ProgramInput{
		Program: "Gold", Bank: "Acme", From: "2027-01-01", CapPeriod: "cycle",
		Rules: []domain.ProgramRuleInput{{Rule: "Dining", MCCs: "5812-5814", Rate: 3, Cap: 100_000}, {Rule: "Others", Rate: 0.5, Fallback: true}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	want := map[string][]any{
		"'Programs'!A6:F6":     {"Gold", "Acme", "2027-01-01", 0.0, 0.0, "cycle"},
		"'ProgramRules'!A8:K8": {"Gold", "2027-01-01", "Dining", "'5812-5814", 3.0, 100_000.0, "", "", "", "", ""},
		"'ProgramRules'!A9:K9": {"Gold", "2027-01-01", "Others", "", 0.5, "", "", "", "x", "", ""},
	}
	if got := updatesOf(client); !reflect.DeepEqual(got, want) {
		t.Fatalf("updates = %v, want %v", got, want)
	}
	if program.Name != "Gold" || len(program.Versions) != 3 {
		t.Fatalf("program: %+v", program)
	}
}

func TestGivenAnExistingVersionWhenCreatingAProgrammeThenItConflicts(t *testing.T) {
	client := cardWritesFixture()
	_, err := newTestDBService(t, client, newFakeCache()).CreateCardProgram(domain.ProgramInput{
		Program: "gold", Bank: "Acme", From: "2026-01-01", CapPeriod: "cycle", Rules: []domain.ProgramRuleInput{{Rule: "Others", Rate: 1, Fallback: true}},
	})
	if !errors.Is(err, domain.ErrConflict) || len(client.updates) != 0 {
		t.Fatalf("err = %v, updates = %v", err, client.updates)
	}
}

func TestGivenAMerchantWhenSavingThenTheSamePatternAndStartIsReplacedAndOthersAppended(t *testing.T) {
	client := cardWritesFixture()
	db := newTestDBService(t, client, newFakeCache())
	merchant, err := db.SaveMerchant(domain.MerchantInput{Pattern: "evn", MCC: 4900, Confidence: "confirmed", Note: "bill"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := updatesOf(client)["'Merchants'!A4:E4"]; !reflect.DeepEqual(got, []any{"evn", "'4900", "", "confirmed", "bill"}) {
		t.Fatalf("replaced row: %v", got)
	}
	if merchant.MCC != 4900 || merchant.Confidence != domain.MCCConfirmed {
		t.Fatalf("merchant: %+v", merchant)
	}
	client.updates = nil
	if _, err := db.SaveMerchant(domain.MerchantInput{Pattern: "Bakery", MCC: 742}); err != nil {
		t.Fatalf("save new: %v", err)
	}
	if got := updatesOf(client)["'Merchants'!A7:E7"]; !reflect.DeepEqual(got, []any{"Bakery", "'0742", "", "assumed", ""}) {
		t.Fatalf("appended row: %v", client.updates)
	}
}

func TestGivenANewBankWhenCreatingAProgrammeThenItsBanksRowIsAddedWithTheSharedAccount(t *testing.T) {
	client := cardWritesFixture()
	db := newTestDBService(t, client, newFakeCache())
	input := domain.ProgramInput{
		Program: "Blue", Bank: "Beta", SharedAccount: "Cash", From: "2026-01-01", CapPeriod: "month",
		Rules: []domain.ProgramRuleInput{{Rule: "Others", Rate: 1, Fallback: true}},
	}
	if _, err := db.CreateCardProgram(input); err != nil {
		t.Fatalf("create: %v", err)
	}
	if got := updatesOf(client)["'Banks'!A3:B3"]; !reflect.DeepEqual(got, []any{"Beta", "Cash"}) {
		t.Fatalf("bank row: %v", client.updates)
	}

	client.updates = nil
	input.Program, input.Bank, input.SharedAccount = "Gold Max", "Acme", "Cash"
	if _, err := db.CreateCardProgram(input); !errors.Is(err, domain.ErrValidation) || len(client.updates) != 0 {
		t.Fatalf("a known bank billing elsewhere: %v %v", err, client.updates)
	}
	input.Bank, input.SharedAccount = "Gamma", "Savings"
	if _, err := db.CreateCardProgram(input); !errors.Is(err, domain.ErrValidation) || len(client.updates) != 0 {
		t.Fatalf("an unknown shared account: %v %v", err, client.updates)
	}
}
