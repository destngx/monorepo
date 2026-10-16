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
	SpendableCash    float64              `json:"spendableCash"`
	BillsDue         float64              `json:"billsDue"`
	CardPayments     float64              `json:"cardPayments"`
	GoalSetAside     float64              `json:"goalSetAside"`
	PerDay           float64              `json:"perDay"`
	Rule             string               `json:"rule"`
	Facts            domain.BillsFacts    `json:"facts"`
	Calendar         []billContext        `json:"calendar"`
	Suggestions      []suggestionContext  `json:"suggestions,omitempty"`
	Items            []recurringItemFacts `json:"items"`
	GoalSetAsideRule string               `json:"goalSetAsideRule"`
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

func (s *Service) billsContext(bills domain.Bills) billsContext {
	safe := bills.SafeToSpend
	out := billsContext{
		SafeToSpend: safe.SafeToSpend, ShortBy: safe.ShortBy, Until: safe.Until, UntilIncome: safe.UntilIncome,
		SpendableCash: safe.SpendableCash, BillsDue: safe.BillsDue, CardPayments: safe.CardPayments, GoalSetAside: safe.GoalSetAside,
		PerDay: safe.PerDay, Rule: "safeToSpend = spendableCash - billsDue - cardPayments - goalSetAside, floored at 0 (shortBy is the shortfall)",
		Facts: bills.Facts(), Calendar: []billContext{}, Items: []recurringItemFacts{}, GoalSetAsideRule: safe.GoalSetAsideRule,
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
