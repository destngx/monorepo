package ai

import "apps/wealth-management/domain"

// billsContext is the bills page as AI facts (ROADMAP R4): safe-to-spend
// with its parts, the 30-day calendar and the suggestions waiting. Names,
// payees and account names go through the PII masker.
type billsContext struct {
	SafeToSpend      float64           `json:"safeToSpend"`
	ShortBy          float64           `json:"shortBy"`
	Until            string            `json:"until"`
	UntilIncome      bool              `json:"untilIncome"`
	ReadyMoney       float64           `json:"readyMoney"`
	ExcludedAccounts []excludedContext `json:"excludedAccounts"`
	// Reserved is the minimum balances kept out of ready money (D35).
	Reserved        float64              `json:"reserved"`
	MinimumBalances []minimumContext     `json:"minimumBalances"`
	MinimumRule     string               `json:"minimumRule"`
	BillsDue        float64              `json:"billsDue"`
	CardPayments    float64              `json:"cardPayments"`
	PerDay          float64              `json:"perDay"`
	Rule            string               `json:"rule"`
	ReadyRule       string               `json:"readyRule"`
	Facts           domain.BillsFacts    `json:"facts"`
	Calendar        []billContext        `json:"calendar"`
	Suggestions     []suggestionContext  `json:"suggestions,omitempty"`
	Items           []recurringItemFacts `json:"items"`
}

// excludedContext is an account left out of ready money, name masked.
type excludedContext struct {
	Name    string                 `json:"name"`
	Balance float64                `json:"balance"`
	Reason  domain.ExclusionReason `json:"reason"`
}

// minimumContext is an account's minimum balance and this month's average
// against it (D35), name masked; the free-text note is left out.
type minimumContext struct {
	Name             string                      `json:"name"`
	Minimum          float64                     `json:"minimum"`
	Balance          float64                     `json:"balance"`
	AverageSoFar     float64                     `json:"averageSoFar"`
	ProjectedAverage float64                     `json:"projectedAverage"`
	RequiredBalance  float64                     `json:"requiredBalance"`
	ExtraNeeded      float64                     `json:"extraNeeded"`
	DaysLeft         int                         `json:"daysLeft"`
	Status           domain.MinimumBalanceStatus `json:"status"`
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
	"Goal money is not subtracted. readyMoney counts each account above its minimum balance to keep only " +
	"(reserved is what is kept back; a hard reserve, since a bank reward on the month's average cannot be made up late)."

func (s *Service) billsContext(bills domain.Bills) billsContext {
	safe := bills.SafeToSpend
	out := billsContext{
		SafeToSpend: safe.SafeToSpend, ShortBy: safe.ShortBy, Until: safe.Until, UntilIncome: safe.UntilIncome,
		ReadyMoney: safe.ReadyMoney, ExcludedAccounts: []excludedContext{}, BillsDue: safe.BillsDue, CardPayments: safe.CardPayments,
		PerDay: safe.PerDay, Rule: safeToSpendRule, ReadyRule: safe.ReadyRule,
		Reserved: safe.Reserved, MinimumBalances: []minimumContext{}, MinimumRule: safe.MinimumRule,
		Facts: bills.Facts(), Calendar: []billContext{}, Items: []recurringItemFacts{},
	}
	for _, excluded := range safe.ExcludedAccounts {
		out.ExcludedAccounts = append(out.ExcludedAccounts, excludedContext{Name: s.masker.Mask(excluded.Name), Balance: excluded.Balance, Reason: excluded.Reason})
	}
	for _, track := range safe.MinimumBalances {
		out.MinimumBalances = append(out.MinimumBalances, minimumContext{
			Name: s.masker.Mask(track.Account), Minimum: track.Minimum, Balance: track.Balance, AverageSoFar: track.AverageSoFar,
			ProjectedAverage: track.ProjectedAverage, RequiredBalance: track.RequiredBalance, ExtraNeeded: track.ExtraNeeded,
			DaysLeft: track.DaysLeft, Status: track.Status,
		})
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
