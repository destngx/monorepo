package domain

import "slices"

// The forms the engine can fill (WM-FILL-15). Field names are the keys of
// the UI form state, so an answer merges into the form as it is.

const FillBanks FillList = "banks"

func fillBound(value float64) *float64 { return &value }

var (
	fillZero = fillBound(0)
	fillOne  = fillBound(1)
)

var fillForms = []FillForm{
	{
		Key: "transaction", Title: "Transaction", Images: true, Page: "/transactions",
		Fields: []FillField{
			{Name: "accountName", Label: "Account", Type: FillName, List: FillAccounts, Required: true},
			{Name: "date", Label: "Date", Type: FillDate, Required: true},
			{Name: "payee", Label: "Payee", Type: FillName, List: FillPayees, New: true, Required: true, Hint: "who was paid or who paid"},
			{Name: "category", Label: "Category", Type: FillName, List: FillCategories},
			{Name: "tags", Label: "Tags", Type: FillNames, List: FillTags, New: true},
			{Name: "memo", Label: "Memo", Type: FillText},
			{Name: "referenceNumber", Label: "Reference", Type: FillText, Hint: "the transaction code, when shown"},
			{Name: "payment", Label: "Payment", Type: FillMoney, Min: fillZero, Hint: "money out"},
			{Name: "deposit", Label: "Deposit", Type: FillMoney, Min: fillZero, Hint: "money in"},
		},
		Guide: "Set either payment (money out) or deposit (money in), never both.",
	},
	{
		Key: "goal", Title: "Goal", Page: "/accounts/goals",
		Fields: []FillField{
			{Name: "name", Label: "Name", Type: FillText, Required: true, Hint: "short, e.g. Trip to Japan"},
			{Name: "target", Label: "Target", Type: FillMoney, Min: fillOne, Hint: "the amount the goal needs in total"},
			{Name: "targetDate", Label: "Target date", Type: FillMonth},
			{Name: "priority", Label: "Priority", Type: FillNumber, Min: fillOne, Hint: "1 is filled first; place it among the goals the app lists"},
			{Name: "startingAmount", Label: "Starting amount", Type: FillMoney, Min: fillZero, Hint: "money already set aside outside the accounts"},
			{Name: "status", Label: "Status", Type: FillChoice, Choices: []string{"active", "paused", "done"}},
			{Name: "note", Label: "Note", Type: FillText},
		},
	},
	{
		Key: "recurring", Title: "Bill or recurring item", Images: true, Page: "/bills",
		Fields: []FillField{
			{Name: "name", Label: "Name", Type: FillText, Required: true},
			{Name: "kind", Label: "Kind", Type: FillChoice, Choices: fillRecurringKinds()},
			{Name: "amount", Label: "Amount", Type: FillMoney, Min: fillZero},
			{Name: "frequency", Label: "How often", Type: FillChoice, Choices: []string{"monthly", "quarterly", "yearly"}},
			{Name: "day", Label: "Day of month", Type: FillDay},
			{Name: "month", Label: "Month", Type: FillNumber, Min: fillOne, Max: fillBound(12), Hint: "1-12, the month a quarterly or yearly item falls in"},
			{Name: "account", Label: "Account", Type: FillName, List: FillAccounts, Hint: "the account it is paid from or into"},
			{Name: "category", Label: "Category", Type: FillName, List: FillCategories},
			{Name: "match", Label: "Payee contains", Type: FillText, Hint: "a word of the payee as the bank shows it, e.g. NETFLIX"},
			{Name: "graceDays", Label: "Grace days", Type: FillNumber, Min: fillZero, Max: fillBound(21), Hint: "days it may be paid late"},
			{Name: "status", Label: "Status", Type: FillChoice, Choices: []string{"active", "paused"}},
			{Name: "note", Label: "Note", Type: FillText},
		},
		Guide: "For a card row (kind card), account is the credit card account and amount stays out.",
	},
	{
		Key: "budget-limits", Title: "Budget limits", RowsLabel: "limit", Page: "/budget",
		Rows: []FillField{
			{Name: "category", Label: "Category", Type: FillName, List: FillCategories, Required: true},
			{Name: "limit", Label: "Limit", Type: FillMoney, Min: fillZero, Required: true},
			{Name: "scope", Label: "Applies to", Type: FillChoice, Choices: []string{"month", "year"}, Hint: "month for this month only, year for every month of the year; month unless the request says otherwise"},
		},
		Guide: "One row per category the request sets. A change such as \"cut shopping by 1M\" uses the limit the app lists.",
	},
	{
		Key: "budget-move", Title: "Move money between categories", Page: "/budget",
		Fields: []FillField{
			{Name: "from", Label: "From", Type: FillName, List: FillCategories, Required: true},
			{Name: "to", Label: "To", Type: FillName, List: FillCategories, Required: true},
			{Name: "amount", Label: "Amount", Type: FillMoney, Min: fillOne, Required: true},
		},
	},
	{
		Key: "category-definition", Title: "Category definition",
		Fields: []FillField{
			{Name: "definition", Label: "What belongs here", Type: FillText, Hint: "one or two plain sentences, what belongs and what does not"},
			{Name: "splitTags", Label: "Split tags", Type: FillNames, List: FillTags, New: true, Hint: "tags that split the category, without #"},
		},
	},
	{
		Key: "allocation", Title: "Allocation targets", Page: "/investments",
		Fields: []FillField{
			{Name: "cash", Label: "Cash", Type: FillPercent, Min: fillZero, Max: fillBound(100)},
			{Name: "funds", Label: "Funds", Type: FillPercent, Min: fillZero, Max: fillBound(100)},
			{Name: "crypto", Label: "Crypto", Type: FillPercent, Min: fillZero, Max: fillBound(100)},
		},
		Guide: "The three targets add up to 100. When the request gives two, set the third to the rest.",
	},
	{
		Key: "account", Title: "Account", Images: true, Page: "/accounts",
		Fields: []FillField{
			{Name: "name", Label: "Name", Type: FillText, Required: true, Hint: "bank or wallet and what the account is, e.g. Bank B Savings"},
			{Name: "type", Label: "Type", Type: FillChoice, Choices: fillAccountTypes},
			{Name: "currency", Label: "Entry currency", Type: FillChoice, Choices: []string{"VND", "USD", "USDT"}},
			{Name: "balance", Label: "Opening balance", Type: FillMoney, Hint: "for a credit card, the available credit"},
			{Name: "note", Label: "Note", Type: FillText},
		},
	},
	{
		Key: "minimum-balance", Title: "Minimum balance", Images: true,
		Fields: []FillField{
			{Name: "account", Label: "Account", Type: FillName, List: FillAccounts, Required: true},
			{Name: "minimum", Label: "Minimum", Type: FillMoney, Min: fillZero, Hint: "the balance to keep, 0 removes it"},
			{Name: "note", Label: "Note", Type: FillText, Hint: "why, e.g. fee waiver"},
		},
	},
	{
		Key: "account-savings", Title: "Savings product and emergency money", Images: true,
		Fields: []FillField{
			{Name: "account", Label: "Account", Type: FillName, List: FillAccounts, Required: true},
			{Name: "product", Label: "Savings product", Type: FillName, List: FillProducts, Hint: "the product id the account holds"},
			{Name: "emergency", Label: "Emergency money", Type: FillFlag, Hint: "true when the account holds emergency money"},
		},
	},
	{
		Key: "best-card", Title: "Best card for a payment", Images: true, Page: "/accounts/credit-cards",
		Fields: []FillField{
			{Name: "payee", Label: "Payee", Type: FillName, List: FillPayees, New: true, Required: true},
			{Name: "amount", Label: "Amount", Type: FillMoney, Min: fillOne, Required: true},
			{Name: "mcc", Label: "MCC", Type: FillNumber, Min: fillZero, Max: fillBound(9999), Hint: "only when the request or image shows it"},
		},
	},
	{
		Key: "stablecoin", Title: "Stablecoin switch", Images: true, Page: "/investments/stablecoin",
		Fields: []FillField{
			{Name: "principal", Label: "Amount", Type: FillMoney, Min: fillZero, Hint: "in USD or a USD stablecoin"},
			{Name: "currentRate", Label: "Current rate", Type: FillPercent, Min: fillZero, Hint: "% a year"},
			{Name: "newRate", Label: "New rate", Type: FillPercent, Min: fillZero, Hint: "% a year"},
			{Name: "days", Label: "Days held", Type: FillNumber, Min: fillOne},
			{Name: "entryFeeKind", Label: "Entry fee as", Type: FillChoice, Choices: []string{"percent", "amount"}},
			{Name: "entryFee", Label: "Entry fee", Type: FillNumber, Min: fillZero, Hint: "a percent or a USDT amount, as entryFeeKind says"},
			{Name: "exitFeeKind", Label: "Exit fee as", Type: FillChoice, Choices: []string{"percent", "amount"}},
			{Name: "exitFee", Label: "Exit fee", Type: FillNumber, Min: fillZero},
		},
		Guide: "\"0.1% fee each way\" sets both fees.",
	},
	{
		Key: "fund-trade", Title: "Fund trade", Images: true, Page: "/investments",
		Fields: []FillField{
			{Name: "code", Label: "Fund code", Type: FillName, List: FillFunds, New: true, Required: true, Hint: "the fund's short code, e.g. VESAF"},
			{Name: "date", Label: "Date", Type: FillDate, Required: true, Hint: "the day the order was matched"},
			{Name: "units", Label: "Units", Type: FillNumber, Required: true, Hint: "units bought, negative for a sale"},
			{Name: "amount", Label: "Amount", Type: FillMoney, Min: fillZero, Hint: "money paid or received, for the note only"},
		},
		Guide: "When only the money is given, leave units out and say so in notes.",
	},
	{
		Key: "crypto-holdings", Title: "Crypto holdings", Images: true, RowsLabel: "coin", Page: "/investments",
		Rows: []FillField{
			{Name: "platform", Label: "Platform", Type: FillName, List: FillPlatforms, New: true, Required: true, Hint: "exchange or wallet"},
			{Name: "symbol", Label: "Coin", Type: FillText, Required: true, Hint: "the ticker in capitals, e.g. BTC"},
			{Name: "quantity", Label: "Quantity", Type: FillNumber, Min: fillZero, Required: true, Hint: "the total held now, not the change"},
			{Name: "note", Label: "Note", Type: FillText},
		},
		Guide: "One row per coin shown, with the platform the request or screen names.",
	},
	{
		Key: "card", Title: "Credit card", Images: true, Page: "/accounts/credit-cards",
		Fields: []FillField{
			{Name: "name", Label: "Card", Type: FillName, List: FillCards, New: true, Required: true},
			{Name: "program", Label: "Programme", Type: FillName, List: FillPrograms},
			{Name: "keyword", Label: "Tag keyword", Type: FillText, Hint: "the word in a tag that marks a payment with this card"},
			{Name: "expiry", Label: "Expiry", Type: FillText, Hint: "MM/YYYY"},
			{Name: "statementDay", Label: "Statement day", Type: FillDay},
			{Name: "dueDay", Label: "Due day", Type: FillDay},
			{Name: "last4", Label: "Last 4 digits", Type: FillText},
			{Name: "default", Label: "Default card", Type: FillFlag, Hint: "takes the untagged spending on the bank's shared account"},
			{Name: "closedFrom", Label: "Closed from", Type: FillDate},
			{Name: "limit", Label: "Credit limit", Type: FillMoney, Min: fillZero},
		},
		Guide: "A statement shows the statement day, the due date (use its day) and the limit. Mention the statement balance and minimum payment in notes.",
	},
	{
		Key: "cashback-program", Title: "Cashback programme", Images: true, RowsLabel: "rule", Page: "/accounts/credit-cards",
		Fields: []FillField{
			{Name: "program", Label: "Programme", Type: FillName, List: FillPrograms, New: true, Required: true},
			{Name: "bank", Label: "Bank", Type: FillName, List: FillBanks, New: true, Required: true},
			{Name: "sharedAccount", Label: "Shared account", Type: FillName, List: FillAccounts, Hint: "only for a bank not listed yet: the account its cards bill to"},
			{Name: "from", Label: "From", Type: FillDate, Required: true, Hint: "the day these terms start"},
			{Name: "annualFee", Label: "Annual fee", Type: FillMoney, Min: fillZero},
			{Name: "cap", Label: "Card cashback cap", Type: FillMoney, Min: fillZero},
			{Name: "capPeriod", Label: "Cap period", Type: FillChoice, Choices: []string{"cycle", "month"}, Hint: "cycle for each statement cycle, month for each calendar month"},
		},
		Rows: []FillField{
			{Name: "rule", Label: "Rule", Type: FillText, Required: true, Hint: "a short name, e.g. Supermarkets"},
			{Name: "mccs", Label: "MCCs", Type: FillText, Hint: "codes and ranges as the bank lists them, e.g. 5411, 5812-5814"},
			{Name: "rate", Label: "Rate", Type: FillPercent, Min: fillZero, Max: fillBound(100), Required: true},
			{Name: "cap", Label: "Rule cap", Type: FillMoney, Min: fillZero},
			{Name: "minTransaction", Label: "Minimum transaction", Type: FillMoney, Min: fillZero},
			{Name: "minCycleSpend", Label: "Minimum cycle spend", Type: FillMoney, Min: fillZero},
			{Name: "fallback", Label: "Fallback", Type: FillFlag, Hint: "the rate for everything no other rule covers"},
			{Name: "exclude", Label: "Exclude", Type: FillFlag, Hint: "spending that earns nothing"},
			{Name: "capGroup", Label: "Cap group", Type: FillText, Hint: "rules sharing one cap get the same group name"},
		},
	},
	{
		Key: "merchant", Title: "Merchant MCC", Images: true, Page: "/accounts/credit-cards",
		Fields: []FillField{
			{Name: "pattern", Label: "Payee pattern", Type: FillName, List: FillPayees, New: true, Required: true, Hint: "the payee as the ledger writes it"},
			{Name: "mcc", Label: "MCC", Type: FillNumber, Min: fillZero, Max: fillBound(9999), Required: true},
			{Name: "from", Label: "From", Type: FillDate, Hint: "only when the MCC changed on a date"},
			{Name: "confidence", Label: "Confidence", Type: FillChoice, Choices: []string{"assumed", "confirmed"}, Hint: "confirmed when a receipt or statement shows the MCC"},
			{Name: "note", Label: "Source", Type: FillText, Hint: "where the MCC comes from"},
		},
	},
}

// ChatFillForms are the forms the chat can fill (WM-FILL-7).
func ChatFillForms() []FillForm {
	var forms []FillForm
	for _, form := range fillForms {
		if form.Page != "" {
			forms = append(forms, form)
		}
	}
	return forms
}

// fillAccountTypes is the type list in the UI's order.
var fillAccountTypes = []string{
	string(AccountTypeActiveUse), string(AccountTypeRarelyUse), string(AccountTypeLongHolding), string(AccountTypeNegativeActive),
	string(AccountTypeDeprecated), string(AccountTypeBank), string(AccountTypeCrypto), string(AccountTypeCash), string(AccountTypeInvestment),
}

func fillRecurringKinds() []string {
	kinds := make([]string, len(RecurringKinds))
	for index, kind := range RecurringKinds {
		kinds[index] = string(kind)
	}
	return kinds
}

// FillForms lists every form that can be filled.
func FillForms() []FillForm { return slices.Clone(fillForms) }

// FillFormFor returns the form with the key.
func FillFormFor(key string) (FillForm, bool) {
	for _, form := range fillForms {
		if form.Key == key {
			return form, true
		}
	}
	return FillForm{}, false
}
