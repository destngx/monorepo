package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Credit card cashback engine (WM-RULE-7, WM-CARD-8, WM-CARD-10). A payment
// earns under the programme version in effect on its date; caps and spend
// tiers count per that version's period; every figure splits into the part
// confirmed by known MCCs and the estimated rest (C5).

const beginningBalanceCategory = "[beginning balance]"

// cardPayment is a card's payment ready for the engine; index is its
// position in the transactions it was read from.
type cardPayment struct {
	index  int
	tx     Transaction
	amount float64
	mcc    PaymentMCC
}

// spendAmount is what a transaction adds to a card's spend: payments, but
// not the beginning balance.
func spendAmount(t Transaction) float64 {
	if normalizeCategory(t.Category) == beginningBalanceCategory || IsCardFee(t) {
		return 0
	}
	return math.Max(0, t.PaymentAmount())
}

func (r CashbackRule) matches(mcc PaymentMCC) bool {
	return !r.Fallback && mcc.Known() && mccListContains(r.MCCs, mcc.Code)
}

func (v ProgramVersion) excludes(mcc PaymentMCC) (CashbackRule, bool) {
	for _, rule := range v.Rules {
		if rule.Exclude && rule.matches(mcc) {
			return rule, true
		}
	}
	return CashbackRule{}, false
}

// RuleFor returns the rule deciding a payment of amount: an exclusion
// matching its MCC wins; else the most demanding qualifying rule matching
// it; else the most demanding qualifying fallback (WM-CARD-8). periodSpend
// is the card's qualifying spend in the payment's period.
func (v ProgramVersion) RuleFor(amount float64, mcc PaymentMCC, periodSpend float64) (CashbackRule, bool) {
	if rule, excluded := v.excludes(mcc); excluded {
		return rule, true
	}
	qualifies := func(rule CashbackRule) bool {
		return amount >= rule.MinTransaction && periodSpend >= rule.MinCycleSpend
	}
	rule, found := mostDemanding(v.Rules, func(rule CashbackRule) bool {
		return !rule.Exclude && rule.matches(mcc) && qualifies(rule)
	})
	if !found {
		rule, found = mostDemanding(v.Rules, func(rule CashbackRule) bool {
			return rule.Fallback && qualifies(rule)
		})
	}
	return rule, found
}

// PaymentCashback is round(amount × rate / 100) under the rule deciding the
// payment, before caps, and that rule's name ("" when none applies).
func (v ProgramVersion) PaymentCashback(amount float64, mcc PaymentMCC, periodSpend float64) (float64, string) {
	if amount <= 0 {
		return 0, ""
	}
	rule, found := v.RuleFor(amount, mcc, periodSpend)
	if !found {
		return 0, ""
	}
	return math.Round(amount * rule.Rate / 100), rule.Name
}

// mostDemanding picks, among the rules accepted, the highest spend tier,
// then the highest minimum transaction, then the first row.
func mostDemanding(rules []CashbackRule, accept func(CashbackRule) bool) (CashbackRule, bool) {
	var best *CashbackRule
	for i := range rules {
		rule := &rules[i]
		if !accept(*rule) {
			continue
		}
		if best == nil || rule.MinCycleSpend > best.MinCycleSpend ||
			(rule.MinCycleSpend == best.MinCycleSpend && rule.MinTransaction > best.MinTransaction) {
			best = rule
		}
	}
	if best == nil {
		return CashbackRule{}, false
	}
	return *best, true
}

// periodGroup is the payments one programme version counts together: a
// period of that version. A version starting mid-period counts its part of
// the period separately, under its own caps. Payments with no version in
// effect fall in a group with the zero version, which earns nothing.
type periodGroup struct {
	version    ProgramVersion
	start, end time.Time // end is exclusive
	payments   []cardPayment
}

// periodResult is a period group evaluated.
type periodResult struct {
	periodGroup
	spent           float64
	qualifyingSpend float64
	// cashback per payment before caps, aligned with payments.
	paymentCashback []float64
	paymentRules    []string
	rules           []RuleStanding
	cashback        CashbackAmount
	unconfirmed     int
}

type versionLookup func(date time.Time) (ProgramVersion, bool)

// groupByPeriod puts each payment in its version's period, ordered by
// period start, then version.
func groupByPeriod(payments []cardPayment, versionOn versionLookup, statementDay int) []periodGroup {
	type key struct{ version, start time.Time }
	groups := map[key]*periodGroup{}
	keys := []key{}
	for _, payment := range payments {
		version, found := versionOn(payment.tx.Date)
		period := version.Period
		if !found {
			period = CapPeriodCycle
		}
		start, end := PeriodBounds(period, statementDay, payment.tx.Date)
		k := key{version.From, start}
		if groups[k] == nil {
			groups[k] = &periodGroup{version: version, start: start, end: end}
			keys = append(keys, k)
		}
		groups[k].payments = append(groups[k].payments, payment)
	}
	sort.Slice(keys, func(i, j int) bool {
		if !keys[i].start.Equal(keys[j].start) {
			return keys[i].start.Before(keys[j].start)
		}
		return keys[i].version.Before(keys[j].version)
	})
	result := make([]periodGroup, 0, len(keys))
	for _, k := range keys {
		result = append(result, *groups[k])
	}
	return result
}

// evaluatePeriod works out each payment's cashback and the period's capped
// totals: rule (or cap group) caps first, then the card cap, each applied to
// the confirmed part and to the total, so the confirmed part never exceeds
// what caps allow.
func evaluatePeriod(group periodGroup) periodResult {
	version := group.version
	result := periodResult{periodGroup: group, paymentCashback: make([]float64, len(group.payments)), paymentRules: make([]string, len(group.payments)), rules: []RuleStanding{}}
	for _, payment := range group.payments {
		result.spent += payment.amount
		if _, excluded := version.excludes(payment.mcc); !excluded {
			result.qualifyingSpend += payment.amount
		}
		if payment.mcc.Status != MCCConfirmed {
			result.unconfirmed++
		}
	}
	// Rules in a cap group are capped together, so earnings are summed per
	// cap key: the group, else the rule.
	capKey := map[string]string{}
	for _, rule := range version.Rules {
		capKey[rule.Name] = "rule:" + rule.Name
		if rule.CapGroup != "" {
			capKey[rule.Name] = "group:" + rule.CapGroup
		}
	}
	type sums struct{ confirmed, total float64 }
	byKey := map[string]*sums{}
	for i, payment := range group.payments {
		cashback, rule := version.PaymentCashback(payment.amount, payment.mcc, result.qualifyingSpend)
		result.paymentCashback[i], result.paymentRules[i] = cashback, rule
		if rule == "" {
			continue
		}
		key := capKey[rule]
		if byKey[key] == nil {
			byKey[key] = &sums{}
		}
		byKey[key].total += cashback
		if payment.mcc.Status == MCCConfirmed {
			byKey[key].confirmed += cashback
		}
	}
	var confirmed, total float64
	standingAt := map[string]int{}
	for _, rule := range version.Rules {
		if rule.Exclude {
			continue
		}
		key := capKey[rule.Name]
		if at, seen := standingAt[key]; seen {
			result.rules[at].Rules = append(result.rules[at].Rules, rule.Name)
			continue
		}
		earned := CashbackAmount{}
		if s := byKey[key]; s != nil {
			earned = newCashbackAmount(capAt(s.confirmed, rule.Cap), capAt(s.total, rule.Cap))
		}
		confirmed += earned.Confirmed
		total += earned.Total
		standing := RuleStanding{Name: rule.Name, Earned: earned, Cap: rule.Cap}
		if rule.CapGroup != "" {
			standing.Name, standing.Rules = rule.CapGroup, []string{rule.Name}
		}
		if rule.Cap > 0 {
			standing.CapLeft = math.Max(0, rule.Cap-earned.Total)
		}
		standingAt[key] = len(result.rules)
		result.rules = append(result.rules, standing)
	}
	result.cashback = newCashbackAmount(capAt(confirmed, version.Cap), capAt(total, version.Cap))
	return result
}

func capAt(value, limit float64) float64 {
	if limit > 0 {
		return math.Min(value, limit)
	}
	return value
}

// nextTier is the nearest spend tier above the period's qualifying spend.
func (r periodResult) nextTier() *NextTier {
	var tier *NextTier
	for _, rule := range r.version.Rules {
		if rule.Exclude || rule.MinCycleSpend <= r.qualifyingSpend {
			continue
		}
		switch {
		case tier == nil || rule.MinCycleSpend < tier.MinCycleSpend:
			tier = &NextTier{MinCycleSpend: rule.MinCycleSpend, SpendNeeded: rule.MinCycleSpend - r.qualifyingSpend, Rules: []string{rule.Name}}
		case rule.MinCycleSpend == tier.MinCycleSpend:
			tier.Rules = append(tier.Rules, rule.Name)
		}
	}
	return tier
}

// IsCashbackRefund matches a refund deposit from the bank's cashback programme.
// IsCardFee reports whether t is a card fee charged by the bank (or given
// back), marked by "bankfee" in its memo. A fee is not spend.
func IsCardFee(t Transaction) bool {
	return t.Memo != nil && strings.Contains(strings.ToLower(*t.Memo), cardFeeMemo)
}

const cardFeeMemo = "bankfee"

func IsCashbackRefund(t Transaction) bool {
	return t.DepositAmount() > 0 && strings.Contains(strings.ToLower(t.Payee), "cashback") && IsRefundCategory(t.Category)
}
