package domain

import (
	"fmt"
	"strings"
)

// Structured insight (WM-AI-3): the shape of every review and commentary.

type InsightSeverity string

const (
	InsightInfo     InsightSeverity = "info"
	InsightWarning  InsightSeverity = "warning"
	InsightSuccess  InsightSeverity = "success"
	InsightCritical InsightSeverity = "critical"
)

var insightSeverities = []InsightSeverity{InsightInfo, InsightWarning, InsightSuccess, InsightCritical}

var InsightIcons = []string{"alert", "trend-up", "trend-down", "savings", "tip", "target", "shield", "chart", "wallet", "clock", "zap", "star"}

const (
	insightMaxSections    = 5
	insightMaxActionItems = 4
)

type InsightSection struct {
	Icon     string          `json:"icon"`
	Title    string          `json:"title"`
	Content  string          `json:"content"`
	Severity InsightSeverity `json:"severity"`
}

type Insight struct {
	Summary     string           `json:"summary"`
	Sections    []InsightSection `json:"sections"`
	ActionItems []string         `json:"actionItems"`
}

// Normalize validates an insight decoded from a model. Cosmetic fields are
// coerced (unknown icon → "tip", unknown severity → "info", extra items
// dropped); a missing summary or no usable section is an error.
func (i *Insight) Normalize() error {
	i.Summary = strings.TrimSpace(i.Summary)
	if i.Summary == "" {
		return fmt.Errorf("%w: insight has no summary", ErrAIOutput)
	}
	sections := make([]InsightSection, 0, len(i.Sections))
	for _, section := range i.Sections {
		section.Title = strings.TrimSpace(section.Title)
		section.Content = strings.TrimSpace(section.Content)
		if section.Title == "" && section.Content == "" {
			continue
		}
		section.Icon = oneOf(strings.ToLower(strings.TrimSpace(section.Icon)), InsightIcons, "tip")
		section.Severity = InsightSeverity(oneOf(strings.ToLower(strings.TrimSpace(string(section.Severity))), severityNames(), string(InsightInfo)))
		sections = append(sections, section)
		if len(sections) == insightMaxSections {
			break
		}
	}
	if len(sections) == 0 {
		return fmt.Errorf("%w: insight has no sections", ErrAIOutput)
	}
	i.Sections = sections
	i.ActionItems = compactStrings(i.ActionItems, insightMaxActionItems)
	return nil
}

func severityNames() []string {
	names := make([]string, len(insightSeverities))
	for index, severity := range insightSeverities {
		names[index] = string(severity)
	}
	return names
}

func oneOf(value string, allowed []string, fallback string) string {
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return fallback
}

// compactStrings trims, drops blanks and keeps at most limit items.
func compactStrings(values []string, limit int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
		if len(out) == limit {
			break
		}
	}
	return out
}

// InsightFormatInstruction is the output contract appended to every
// structured-insight prompt. It is engine code, not prompt content: the
// parser below depends on it.
var InsightFormatInstruction = `IMPORTANT: Respond with ONLY a valid JSON object (no markdown, no backticks, no explanation) in this exact format:
{
  "summary": "One concise sentence summarizing the key takeaway",
  "sections": [
    {
      "icon": "<one of: ` + strings.Join(InsightIcons, ", ") + `>",
      "title": "Section Title",
      "content": "1-3 sentences focused on ONE specific recommendation or finding.",
      "severity": "<one of: info, warning, success, critical>"
    }
  ],
  "actionItems": ["Action item 1", "Action item 2"]
}

Rules:
- Provide 3-5 sections, each on a different aspect.
- Provide 2-4 actionable items.
- State each fact once: no section repeats another, the summary does not repeat a section's figures, and action items are concrete next steps, not restatements of a section.
- severity "critical" = needs immediate attention, "warning" = should address soon, "success" = positive finding, "info" = informational.
- Keep the summary under 20 words.
- Quote amounts exactly as given in the data; never compute new ledger totals.`
