package config

import (
	"apps/wealth-management/domain"
	"errors"
	"testing"
	"time"
)

func TestDefaultCardConfigIsValid(t *testing.T) {
	config, err := LoadCardConfig(Values{})
	if err != nil {
		t.Fatalf("default config: %v", err)
	}
	cards := config.Banks[0].Cards
	if len(cards) != 2 || cards[0].Keyword != "uniq" {
		t.Fatalf("UNIQ must be checked before Platinum, got %+v", cards)
	}
}

func TestParseCardConfigRejectsInvalidData(t *testing.T) {
	tests := map[string]string{
		"not json":     `{`,
		"bad due day":  `{"banks":[{"name":"B","cards":[{"name":"C","keyword":"c","statementDay":5,"dueDay":40}]}]}`,
		"rule no tag":  `{"banks":[{"name":"B","cards":[{"name":"C","keyword":"c","statementDay":5,"dueDay":5,"rules":[{"name":"R","rate":1}]}]}]}`,
		"card no name": `{"banks":[{"name":"B","cards":[{"keyword":"c","statementDay":5,"dueDay":5}]}]}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseCardConfig([]byte(raw)); !errors.Is(err, ErrInvalidCardConfig) {
				t.Fatalf("expected ErrInvalidCardConfig, got %v", err)
			}
		})
	}
}

func TestLoadCardConfigFromMissingFileFails(t *testing.T) {
	if _, err := LoadCardConfig(Values{"CARD_PROGRAMS_FILE": "./does-not-exist.json"}); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// WM-RULE-7 decision: UNIQ pays at most 300k per cycle across all its rules.
func TestGivenDefaultCardsWhenUNIQRulesEachHitTheirCapThenTheCardTotalIsCappedAt300k(t *testing.T) {
	config, err := LoadCardConfig(Values{})
	if err != nil {
		t.Fatalf("default config: %v", err)
	}
	if uniq := config.Banks[0].Cards[0]; uniq.MonthlyCap != 300_000 {
		t.Fatalf("UNIQ monthly cap = %v, want 300000", uniq.MonthlyCap)
	}
	today := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	spend := func(amount float64, tag string) domain.Transaction {
		return domain.Transaction{AccountName: "Sacombank UNIQ", Date: today, Category: "Shopping", Payment: &amount, Tags: []string{tag}}
	}
	accounts := []domain.Account{{Name: "Sacombank UNIQ", Type: domain.AccountTypeNegativeActive}}
	transactions := []domain.Transaction{spend(2_000_000, "uniq supermarket and transport"), spend(80_000_000, "uniq others")}

	reports := domain.ComputeCardReports(config, accounts, transactions, today)
	for _, group := range reports[0].Accounts {
		for _, card := range group.Cards {
			if card.Keyword == "uniq" && card.EstimatedCashback != 300_000 {
				t.Fatalf("UNIQ estimated cashback = %v, want 300000", card.EstimatedCashback)
			}
		}
	}
}
