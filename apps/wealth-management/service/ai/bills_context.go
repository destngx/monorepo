package ai

import "apps/wealth-management/domain"

// billsContext is the bills page as AI facts (ROADMAP R4): safe-to-spend
// with its parts, the 30-day calendar and the suggestions waiting. Names,
// payees and account names go through the PII masker.
type billsContext struct {
	SafeToSpend      float64              `json:"safeToSpend"`
	ShortBy          float64              `json:"shortBy"`
	Until            string               `json:"until"`
	UntilIncome      bool                 `json:"untilIncome"`
	ReadyMoney       float64              `json:"readyMoney"`
	ExcludedAccounts []excludedContext    `json:"excludedAccounts"`
	BillsDue         float64              `json:"billsDue"`
	CardPayments     float64              `json:"cardPayments"`
	PerDay           float64              `json:"perDay"`
	Rule             string               `json:"rule"`
	ReadyRule        string               `json:"readyRule"`
	Facts            domain.BillsFacts    `json:"facts"`
	Calendar         []billContext        `json:"calendar"`
	Suggestions      []suggestionContext  `json:"suggestions,omitempty"`
	Items            []recurringItemFacts `json:"items"`
}

// excludedContext is an account left out of ready money, name masked.
type excludedContext struct {
	Name    string                 `json:"name"`
	Balance float64                `json:"balance"`
	Reason  domain.ExclusionReason `json:"reason"`
}

type billContext struct {
	Name     string               `json:"name"`
	Kind     domain.RecurringKind `json:"kind"`
	Date     string               `json:"date"`
	Amount   float64              `json:"amount"`
	Status   domain.BillStatus    `json:"status"`
	Paid     float64              `json:"paid,omitempty"`
	Estimate bool                 `json:"estimate,omitempty"`
}

type suggestionContext struct {
	Name       string                      `json:"name"`
	Kind       domain.RecurringKind        `json:"kind"`
	Amount     float64                     `json:"amount"`
	Frequency  domain.RecurringFrequency   `json:"frequency"`
	Day        int                         `json:"day"`
	Confidence domain.SuggestionConfidence `json:"confidence"`
}

type recurringItemFacts struct {
	Name      string                    `json:"name"`
	Kind      domain.RecurringKind      `json:"kind"`
	Amount    *float64                  `json:"amount"`
	Frequency domain.RecurringFrequency `json:"frequency"`
	Day       int                       `json:"day,omitempty"`
	Status    domain.RecurringStatus    `json:"status"`
	NextDate  string                    `json:"nextDate,omitempty"`
}

// safeToSpendRule states the formula for the model (WM-BILL-5, D33).
const safeToSpendRule = "safeToSpend = readyMoney - billsDue - cardPayments, floored at 0 (shortBy is the shortfall). " +
	"Bills and card payments count when unpaid and due before the next income (overdue ones too); one due on payday " +
	"is paid by that wage and not counted. Without an expected income the window is the next 30 days, inclusive. " +
	"Goal money is not subtracted."

func (s *Service) billsContext(bills domain.Bills) billsContext {
	safe := bills.SafeToSpend
	out := billsContext{
		SafeToSpend: safe.SafeToSpend, ShortBy: safe.ShortBy, Until: safe.Until, UntilIncome: safe.UntilIncome,
		ReadyMoney: safe.ReadyMoney, ExcludedAccounts: []excludedContext{}, BillsDue: safe.BillsDue, CardPayments: safe.CardPayments,
		PerDay: safe.PerDay, Rule: safeToSpendRule, ReadyRule: safe.ReadyRule,
		Facts: bills.Facts(), Calendar: []billContext{}, Items: []recurringItemFacts{},
	}
	for _, excluded := range safe.ExcludedAccounts {
		out.ExcludedAccounts = append(out.ExcludedAccounts, excludedContext{Name: s.masker.Mask(excluded.Name), Balance: excluded.Balance, Reason: excluded.Reason})
	}
	for _, o := range bills.Calendar.Occurrences {
		out.Calendar = append(out.Calendar, billContext{
			Name: s.masker.Mask(o.Name), Kind: o.Kind, Date: o.ExpectedDate, Amount: o.ExpectedAmount, Status: o.Status, Paid: o.PaidAmount, Estimate: o.Estimate,
		})
	}
	for _, suggestion := range bills.Suggestions {
		out.Suggestions = append(out.Suggestions, suggestionContext{
			Name: s.masker.Mask(suggestion.Name), Kind: suggestion.Kind, Amount: suggestion.Amount, Frequency: suggestion.Frequency, Day: suggestion.Day, Confidence: suggestion.Confidence,
		})
	}
	for _, item := range bills.Items {
		out.Items = append(out.Items, recurringItemFacts{
			Name: s.masker.Mask(item.Name), Kind: item.Kind, Amount: item.Amount, Frequency: item.Frequency, Day: item.Day, Status: item.Status, NextDate: item.NextDate,
		})
	}
	return out
}
