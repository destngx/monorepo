package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ExplainCashback walks through how a card period's cashback estimate was
// reached, payment by payment (WM-CARD-17); markdown.
func (s *Service) ExplainCashback(ctx context.Context, request port.CashbackExplainRequest) (string, error) {
	if strings.TrimSpace(request.Card) == "" {
		return "", fmt.Errorf("%w: card is required", domain.ErrValidation)
	}
	if err := s.requireConfigured(); err != nil {
		return "", err
	}
	period, err := s.analytics.CardPeriod(request.Card, request.Month, false)
	if err != nil {
		return "", err
	}
	snap, err := s.loadSnapshot()
	if err != nil {
		return "", err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return "", err
	}
	vars := map[string]string{
		"card": period.Card, "program": period.Program, "month": period.Month,
		"start": period.Start, "end": period.End, "open": strconv.FormatBool(period.Open),
		"periodJson": s.masker.Mask(toJSON(explainPeriod(period))),
	}
	task, err := p.render("tasks/cashback-explain", vars)
	if err != nil {
		return "", err
	}
	action, err := p.render("actions/cashback-explain", vars)
	if err != nil {
		return "", err
	}
	system, err := s.systemPrompt(p, snap, task)
	if err != nil {
		return "", err
	}
	messages := []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, {Role: domain.AIRoleUser, Content: action + "\n\nAnswer in concise markdown."}}
	model := s.models(domain.AIUseCaseInsight, request.Model)[0]
	return cached(ctx, s, "cashback-explain", request.Force, []any{model, messages}, func(ctx context.Context) (string, error) {
		response, err := s.complete(ctx, call{feature: "cashback-explain", class: classExplain, useCase: domain.AIUseCaseInsight, model: request.Model, prompts: p.used, messages: messages, validate: nonEmpty})
		return strings.TrimSpace(response.Content), err
	})
}

type explainedPayment struct {
	Date      string           `json:"date"`
	Payee     string           `json:"payee"`
	Amount    float64          `json:"amount"`
	MCC       int              `json:"mcc,omitempty"`
	MCCStatus domain.MCCStatus `json:"mccStatus"`
	// Rule is empty when the payment earns nothing (below a minimum,
	// excluded or no rule matched).
	Rule     string  `json:"rule,omitempty"`
	Cashback float64 `json:"cashbackBeforeCaps"`
}

type explainedPart struct {
	VersionFrom     string                `json:"programVersionFrom,omitempty"`
	Start           string                `json:"start"`
	End             string                `json:"end"`
	CardCap         float64               `json:"cardCap,omitempty"`
	CapPeriod       domain.CapPeriod      `json:"capPeriod,omitempty"`
	Rules           []string              `json:"rules"`
	Spent           float64               `json:"spent"`
	QualifyingSpend float64               `json:"qualifyingSpend"`
	Cashback        domain.CashbackAmount `json:"cashbackAfterCaps"`
	Standings       []domain.RuleStanding `json:"perRuleAfterCaps"`
	NextTier        *domain.NextTier      `json:"nextTier,omitempty"`
	Payments        []explainedPayment    `json:"payments"`
}

type explainedPeriod struct {
	Card           string                      `json:"card"`
	Program        string                      `json:"program"`
	Start          string                      `json:"start"`
	End            string                      `json:"end"`
	Open           bool                        `json:"open"`
	Spent          float64                     `json:"spent"`
	Cashback       domain.CashbackAmount       `json:"cashbackEstimate"`
	Refund         float64                     `json:"actualRefund,omitempty"`
	Reconciliation domain.ReconciliationStatus `json:"reconciliation"`
	Parts          []explainedPart             `json:"parts"`
}

// explainPeriod trims the period to what the explanation needs.
func explainPeriod(period domain.CardPeriodDetail) explainedPeriod {
	out := explainedPeriod{
		Card: period.Card, Program: period.Program, Start: period.Start, End: period.End, Open: period.Open,
		Spent: period.Spent, Cashback: period.Cashback, Refund: period.Refund, Reconciliation: period.Status,
		Parts: make([]explainedPart, 0, len(period.Parts)),
	}
	for _, part := range period.Parts {
		explained := explainedPart{
			VersionFrom: part.VersionFrom, Start: part.Start, End: part.End, CardCap: part.CardCap, CapPeriod: part.CapPeriod,
			Rules: part.Rules, Spent: part.Spent, QualifyingSpend: part.QualifyingSpend, Cashback: part.Cashback,
			Standings: part.Standings, NextTier: part.NextTier, Payments: make([]explainedPayment, 0, len(part.Payments)),
		}
		for _, payment := range part.Payments {
			explained.Payments = append(explained.Payments, explainedPayment{
				Date: payment.Transaction.Date.Format(time.DateOnly), Payee: payment.Transaction.Payee,
				Amount: payment.Transaction.PaymentAmount(), MCC: payment.MCC.Code, MCCStatus: payment.MCC.Status,
				Rule: payment.Rule, Cashback: payment.Cashback,
			})
		}
		out.Parts = append(out.Parts, explained)
	}
	return out
}
