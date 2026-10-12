package config

import (
	"errors"
	"testing"
)

func TestDefaultCardConfigIsValid(t *testing.T) {
	t.Setenv("CARD_PROGRAMS_FILE", "")
	config, err := LoadCardConfig()
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
	t.Setenv("CARD_PROGRAMS_FILE", "./does-not-exist.json")
	if _, err := LoadCardConfig(); err == nil {
		t.Fatal("expected error for missing file")
	}
}
