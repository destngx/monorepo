package config

import (
	"apps/wealth-management/domain"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Card programs ship as data (WM-CARD-3). CARD_PROGRAMS_FILE overrides the embedded default.

//go:embed cards.default.json
var defaultCardConfig []byte

var ErrInvalidCardConfig = errors.New("invalid card programs configuration")

// LoadCardConfig reads CARD_PROGRAMS_FILE, a JSON file replacing the
// built-in card programs.
func LoadCardConfig(values Values) (domain.CardConfig, error) {
	raw := defaultCardConfig
	if path := values.get("CARD_PROGRAMS_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return domain.CardConfig{}, fmt.Errorf("read CARD_PROGRAMS_FILE: %w", err)
		}
		raw = data
	}
	return ParseCardConfig(raw)
}

func ParseCardConfig(raw []byte) (domain.CardConfig, error) {
	var config domain.CardConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return domain.CardConfig{}, fmt.Errorf("%w: %v", ErrInvalidCardConfig, err)
	}
	if err := validateCardConfig(config); err != nil {
		return domain.CardConfig{}, err
	}
	return config, nil
}

func validateCardConfig(config domain.CardConfig) error {
	var problems []string
	for _, bank := range config.Banks {
		if bank.Name == "" {
			problems = append(problems, "bank without name")
		}
		for _, card := range bank.Cards {
			if card.Name == "" || card.Keyword == "" {
				problems = append(problems, fmt.Sprintf("%s: card needs name and keyword", bank.Name))
			}
			if card.StatementDay < 1 || card.StatementDay > 31 || card.DueDay < 1 || card.DueDay > 31 {
				problems = append(problems, fmt.Sprintf("%s: statementDay and dueDay must be 1-31", card.Name))
			}
			for _, rule := range card.Rules {
				if rule.Name == "" || rule.Tag == "" || rule.Rate < 0 || rule.CapMonthly < 0 {
					problems = append(problems, fmt.Sprintf("%s: rule %q needs name, tag and non-negative rate and cap", card.Name, rule.Name))
				}
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalidCardConfig, strings.Join(problems, "; "))
	}
	return nil
}
