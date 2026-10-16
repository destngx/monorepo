package domain

import "time"

// Credit card cashback types (WM-RULE-7, §5.2 of the requirements). A
// programme is a bank's published terms, held or not (C1); a card is one the
// user holds; a bank owns the credit limit its cards share (C9).

// CapPeriod is the period a programme version counts caps and spend tiers
// in (C7): the card's statement cycle or the calendar month.
type CapPeriod string

const (
	CapPeriodCycle CapPeriod = "cycle"
	CapPeriodMonth CapPeriod = "month"
)

// CashbackRule pays Rate percent on payments whose MCC is in MCCs, up to Cap
// per period. It qualifies only when the payment is at least MinTransaction
// and the card's qualifying spend in the period is at least MinCycleSpend.
type CashbackRule struct {
	Name           string     `json:"name"`
	MCCs           []MCCRange `json:"mccs"`
	Rate           float64    `json:"rate"`
	Cap            float64    `json:"cap"`
	MinTransaction float64    `json:"minTransaction"`
	MinCycleSpend  float64    `json:"minCycleSpend"`
	// Fallback rules apply to spend that matches no other rule, including
	// payments whose MCC is unknown.
	Fallback bool `json:"fallback,omitempty"`
	// Exclude rules earn nothing on what they match, ahead of every other
	// rule, and that spend does not count toward MinCycleSpend.
	Exclude bool `json:"exclude,omitempty"`
	// CapGroup names rules that share one cap: what they earn together is
	// bounded by Cap, which each of them states.
	CapGroup string `json:"capGroup,omitempty"`
}

// ProgramVersion is a full copy of a programme's terms from a date (C2).
type ProgramVersion struct {
	From      time.Time `json:"from"`
	AnnualFee float64   `json:"annualFee"`
	// Cap bounds the card's total cashback per period; 0 means none.
	Cap    float64        `json:"cap"`
	Period CapPeriod      `json:"period"`
	Rules  []CashbackRule `json:"rules"`
}

// Program is a bank's cashback programme; Versions are sorted by From.
type Program struct {
	Name     string           `json:"name"`
	Bank     string           `json:"bank"`
	Versions []ProgramVersion `json:"versions"`
}

// Bank owns the credit limit and usage its cards share (F8, C9).
type Bank struct {
	Name string `json:"name"`
	// SharedAccount is the ledger account the bank's cards bill to; untagged
	// spend on it belongs to the bank's default card.
	SharedAccount string `json:"sharedAccount"`
	// ReferenceLimit is the limit to keep usage under (N9); the shared
	// account's credit limit (Accounts C), when set, overrides it.
	ReferenceLimit float64 `json:"referenceLimit"`
}

// Card is a card the user holds.
type Card struct {
	Name    string `json:"name"`
	Program string `json:"program"`
	// Keyword in a tag marks a payment as made with this card.
	Keyword      string `json:"keyword"`
	Expiry       string `json:"expiry"`
	StatementDay int    `json:"statementDay"`
	// DueDay is the bank's due day, shown for reference; payments follow
	// the pay-by date (C8).
	DueDay int `json:"dueDay"`
	// Default takes the untagged spend on the bank's shared account.
	Default bool `json:"default,omitempty"`
	// ClosedFrom is the first day the card is no longer used: payments from
	// then on are not its, while its history and later refunds stay.
	ClosedFrom *time.Time `json:"closedFrom,omitempty"`
	// Last4 is the card number's last four digits, matched against the
	// account shown on captured material.
	Last4 string `json:"last4,omitempty"`
}

// ClosedOn reports whether the card is closed on date.
func (c Card) ClosedOn(date time.Time) bool {
	return c.ClosedFrom != nil && !date.Before(*c.ClosedFrom)
}

// CardConfig is the full cashback configuration from the sheet.
type CardConfig struct {
	Programs  []Program     `json:"programs"`
	Banks     []Bank        `json:"banks"`
	Cards     []Card        `json:"cards"`
	Merchants []MerchantMCC `json:"merchants"`
}

// CashbackAmount splits cashback into the part every MCC involved is
// confirmed for and the estimated rest (C5).
type CashbackAmount struct {
	Confirmed float64 `json:"confirmed"`
	Estimated float64 `json:"estimated"`
	Total     float64 `json:"total"`
}

func newCashbackAmount(confirmed, total float64) CashbackAmount {
	return CashbackAmount{Confirmed: confirmed, Estimated: total - confirmed, Total: total}
}

func (a CashbackAmount) plus(other CashbackAmount) CashbackAmount {
	return newCashbackAmount(a.Confirmed+other.Confirmed, a.Total+other.Total)
}

// ReconciliationStatus compares a period's refund with what it should earn
// (WM-CARD-11, C6).
type ReconciliationStatus string

const (
	// ReconciliationOpen: the period has not ended.
	ReconciliationOpen ReconciliationStatus = "open"
	// ReconciliationAwaiting: the period ended, cashback is due, no refund yet.
	ReconciliationAwaiting ReconciliationStatus = "awaiting"
	// ReconciliationMatch: the refund lies between the confirmed part and the total.
	ReconciliationMatch ReconciliationStatus = "match"
	// ReconciliationBelow: the refund is less than the confirmed part.
	ReconciliationBelow ReconciliationStatus = "below"
	// ReconciliationAbove: the refund exceeds the total, so some payment
	// earned more than estimated (an assumed or unknown MCC is wrong).
	ReconciliationAbove ReconciliationStatus = "above"
)

// CardPeriodHistory is one counting period of a card. Month is the month
// the period starts in; a refund counts for the period whose Month is its
// effective month (WM-RULE-4), so write that month in the refund's memo.
type CardPeriodHistory struct {
	Month      string               `json:"month"`
	Start      string               `json:"start,omitempty"`
	End        string               `json:"end,omitempty"`
	Spent      float64              `json:"spent"`
	Cashback   CashbackAmount       `json:"cashback"`
	Refund     float64              `json:"refund"`
	Status     ReconciliationStatus `json:"status"`
	Efficiency float64              `json:"efficiency"`
	Missed     float64              `json:"missed"`
}

// RuleStanding is one rule's cashback so far in the period.
type RuleStanding struct {
	// Name is the rule's, or its cap group's.
	Name string `json:"name"`
	// Rules are a cap group's rules, in sheet order; absent for a single rule.
	Rules  []string       `json:"rules,omitempty"`
	Earned CashbackAmount `json:"earned"`
	// Cap is the rule's (or group's) cap; 0 means none and CapLeft is then 0.
	Cap     float64 `json:"cap"`
	CapLeft float64 `json:"capLeft"`
}

// NextTier is the nearest spend tier not reached yet in the period.
type NextTier struct {
	MinCycleSpend float64  `json:"minCycleSpend"`
	SpendNeeded   float64  `json:"spendNeeded"`
	Rules         []string `json:"rules"`
}

// PeriodStanding is a card's position in its current counting period
// (WM-CARD-12).
type PeriodStanding struct {
	Start string  `json:"start"`
	End   string  `json:"end"`
	Spent float64 `json:"spent"`
	// QualifyingSpend is what counts toward spend tiers: excluded spend
	// does not.
	QualifyingSpend float64        `json:"qualifyingSpend"`
	Cashback        CashbackAmount `json:"cashback"`
	Cap             float64        `json:"cap"`
	CapLeft         float64        `json:"capLeft"`
	Rules           []RuleStanding `json:"rules"`
	NextTier        *NextTier      `json:"nextTier,omitempty"`
	PaymentCount    int            `json:"paymentCount"`
	// UnconfirmedPayments counts payments whose MCC is assumed or unknown.
	UnconfirmedPayments int `json:"unconfirmedPayments"`
}

// CardTransaction is a card's transaction with the cashback it earns before
// period caps.
type CardTransaction struct {
	Transaction Transaction `json:"transaction"`
	MCC         PaymentMCC  `json:"mcc"`
	Rule        string      `json:"rule,omitempty"`
	Cashback    float64     `json:"cashback"`
}

// CardReport is the computed state of one held card.
type CardReport struct {
	Name        string `json:"name"`
	Bank        string `json:"bank"`
	Program     string `json:"program"`
	Keyword     string `json:"keyword"`
	AccountName string `json:"accountName"`
	Expiry      string `json:"expiry"`
	// ClosedFrom is the card's closing date, when set.
	ClosedFrom string `json:"closedFrom,omitempty"`
	Last4      string `json:"last4,omitempty"`
	// VersionFrom is the start of the programme version in effect today;
	// empty when none is.
	VersionFrom string         `json:"versionFrom,omitempty"`
	AnnualFee   float64        `json:"annualFee"`
	Cap         float64        `json:"cap"`
	CapPeriod   CapPeriod      `json:"capPeriod,omitempty"`
	Rules       []CashbackRule `json:"rules"`
	Cycle       CardCycle      `json:"cycle"`
	// Statement is what the pay-by date settles; nil for a closed card.
	Statement *CardStatement `json:"statement,omitempty"`
	Standing  PeriodStanding `json:"standing"`
	RefundYTD float64        `json:"refundYtd"`
	// FeesPaidYTD are this year's fee rows (memo "bankfee"), net of any
	// given back; the fee counts against net earn only once paid.
	FeesPaidYTD  float64             `json:"feesPaidYtd"`
	NetEarnYTD   float64             `json:"netEarnYtd"`
	LifetimeFees float64             `json:"lifetimeFees"`
	History      []CardPeriodHistory `json:"history"`
	// Transactions are the card's transactions in the current period, newest first.
	Transactions []CardTransaction `json:"transactions"`
}

// CardStatement estimates a statement from the ledger: the card's spend and
// fees in the statement cycle less the cashback credited in it. Payments
// toward it are not deducted: the shared account cannot tell whose they are.
type CardStatement struct {
	Start string `json:"start"`
	// End is inclusive: the day before the next statement day.
	End string `json:"end"`
	// Open is true while the cycle has not ended, so the amount can grow.
	Open     bool    `json:"open,omitempty"`
	Spent    float64 `json:"spent"`
	Fees     float64 `json:"fees"`
	Cashback float64 `json:"cashback"`
	Amount   float64 `json:"amount"`
	// Refunds are the cashback credits behind Cashback.
	Refunds []StatementRefund `json:"refunds,omitempty"`
}

// StatementRefund is a cashback credit on a statement: it lowers the
// statement it is credited in, while For (the month in its memo, else the
// month it is credited in) is the period it was earned in (WM-RULE-4).
type StatementRefund struct {
	Date   string  `json:"date"`
	For    string  `json:"for"`
	Amount float64 `json:"amount"`
}

// CardBankReport is a bank's shared limit and its held cards (WM-CARD-16).
type CardBankReport struct {
	Name            string          `json:"name"`
	SharedAccount   string          `json:"sharedAccount"`
	Limit           float64         `json:"limit"`
	Usage           float64         `json:"usage"`
	Utilization     float64         `json:"utilization"`
	UtilizationBand UtilizationBand `json:"utilizationBand"`
	RemainingLimit  float64         `json:"remainingLimit"`
	Cashback        CashbackAmount  `json:"cashback"`
	RefundYTD       float64         `json:"refundYtd"`
	FeesPaidYTD     float64         `json:"feesPaidYtd"`
	NetEarnYTD      float64         `json:"netEarnYtd"`
	Cards           []CardReport    `json:"cards"`
}

// CardSuggestion is what one held card would earn on a payment now
// (WM-CARD-13).
type CardSuggestion struct {
	Card    string     `json:"card"`
	Bank    string     `json:"bank"`
	Program string     `json:"program"`
	MCC     PaymentMCC `json:"mcc"`
	Rule    string     `json:"rule,omitempty"`
	// Cashback is the change in the period's capped total, so a payment
	// that reaches a spend tier also counts what the tier adds elsewhere.
	Cashback CashbackAmount `json:"cashback"`
	// FitsLimit is false when the payment would exceed the bank's remaining limit.
	FitsLimit      bool    `json:"fitsLimit"`
	RemainingLimit float64 `json:"remainingLimit"`
}

// ProgramComparison is a programme replayed over the last year of card
// payments (WM-CARD-14).
type ProgramComparison struct {
	Program     string         `json:"program"`
	Bank        string         `json:"bank"`
	Held        bool           `json:"held"`
	VersionFrom string         `json:"versionFrom"`
	Spent       float64        `json:"spent"`
	Cashback    CashbackAmount `json:"cashback"`
	AnnualFee   float64        `json:"annualFee"`
	// Net is the cashback total less the annual fee.
	Net float64 `json:"net"`
}

// ProgramComparisonReport ranks programmes by Net, best first.
type ProgramComparisonReport struct {
	From     string              `json:"from"`
	To       string              `json:"to"`
	Payments int                 `json:"payments"`
	Programs []ProgramComparison `json:"programs"`
}
