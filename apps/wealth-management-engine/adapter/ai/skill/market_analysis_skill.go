package skill

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type MarketAnalysisSkill struct {
	skillPath string
}

func NewMarketAnalysisSkill(skillPath string) *MarketAnalysisSkill {
	return &MarketAnalysisSkill{skillPath: skillPath}
}

func (s *MarketAnalysisSkill) Analyze(_ context.Context, ticker string) (string, error) {
	content, err := os.ReadFile(s.skillPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	firstLine := "No guidance"
	if len(lines) > 0 && lines[0] != "" {
		firstLine = lines[0]
	}

	return fmt.Sprintf("%s | ticker=%s", firstLine, strings.ToUpper(ticker)), nil
}
