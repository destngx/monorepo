package domain

import (
	"fmt"
	"strings"
	"time"
)

type NewsSentiment string

const (
	NewsPositive NewsSentiment = "Positive"
	NewsNegative NewsSentiment = "Negative"
	NewsNeutral  NewsSentiment = "Neutral"
)

type NewsBias string

const (
	NewsBullish NewsBias = "BULLISH"
	NewsBearish NewsBias = "BEARISH"
	// NewsMixed is the bias when no sentiment has a strict plurality.
	NewsMixed NewsBias = "NEUTRAL"
)

// DefaultNewsTopics are the WM-INV-17 scanners.
var DefaultNewsTopics = []string{"Vietnam market & policy", "Interest rates & macro", "Global geopolitics"}

// NewsArticle is a search result with the model's reading of it.
type NewsArticle struct {
	SearchResult
	Sentiment  NewsSentiment `json:"sentiment"`
	Confidence int           `json:"confidence"`
	Impact     int           `json:"impact"`
	Summary    string        `json:"summary"`
}

type NewsCounts struct {
	Positive int `json:"positive"`
	Negative int `json:"negative"`
	Neutral  int `json:"neutral"`
}

// NewsReport is WM-INV-17: the bias is computed here, not by the model.
type NewsReport struct {
	Topic       string        `json:"topic"`
	Query       string        `json:"query"`
	Bias        NewsBias      `json:"bias"`
	Counts      NewsCounts    `json:"counts"`
	Articles    []NewsArticle `json:"articles"`
	Provider    string        `json:"provider"`
	Fallback    string        `json:"fallback,omitempty"`
	GeneratedAt time.Time     `json:"generatedAt"`
}

// NewsQuery is the search the legacy scanner ran for a topic.
func NewsQuery(topic string, year int) string {
	return fmt.Sprintf("%s latest market news financial impact %d", strings.TrimSpace(topic), year)
}

// NewsClassification is the model's verdict on the article at Index.
type NewsClassification struct {
	Index      int           `json:"index"`
	Sentiment  NewsSentiment `json:"sentiment"`
	Confidence float64       `json:"confidence"`
	Impact     float64       `json:"impactScore"`
	Summary    string        `json:"summary"`
}

// NewsClassifications is the JSON contract of the news prompt.
type NewsClassifications struct {
	Articles []NewsClassification `json:"articles"`
}

// ClassifyNews joins the model's verdicts to the results. Every result needs
// exactly one verdict with a known sentiment and 0-100 scores; anything else
// is ErrAIOutput so the fallback model gets a turn.
func ClassifyNews(results []SearchResult, verdicts []NewsClassification) ([]NewsArticle, NewsCounts, error) {
	articles := make([]NewsArticle, len(results))
	seen := make([]bool, len(results))
	var counts NewsCounts
	for _, verdict := range verdicts {
		if verdict.Index < 0 || verdict.Index >= len(results) || seen[verdict.Index] {
			return nil, NewsCounts{}, fmt.Errorf("%w: article index %d is unknown or repeated", ErrAIOutput, verdict.Index)
		}
		sentiment := parseNewsSentiment(verdict.Sentiment)
		switch sentiment {
		case NewsPositive:
			counts.Positive++
		case NewsNegative:
			counts.Negative++
		case NewsNeutral:
			counts.Neutral++
		default:
			return nil, NewsCounts{}, fmt.Errorf("%w: unknown sentiment %q", ErrAIOutput, verdict.Sentiment)
		}
		if verdict.Confidence < 0 || verdict.Confidence > 100 || verdict.Impact < 0 || verdict.Impact > 100 {
			return nil, NewsCounts{}, fmt.Errorf("%w: article %d scores must be 0-100", ErrAIOutput, verdict.Index)
		}
		seen[verdict.Index] = true
		articles[verdict.Index] = NewsArticle{
			SearchResult: results[verdict.Index],
			Sentiment:    sentiment,
			Confidence:   int(verdict.Confidence + 0.5),
			Impact:       int(verdict.Impact + 0.5),
			Summary:      strings.TrimSpace(verdict.Summary),
		}
	}
	for index, ok := range seen {
		if !ok {
			return nil, NewsCounts{}, fmt.Errorf("%w: article %d was not classified", ErrAIOutput, index)
		}
	}
	return articles, counts, nil
}

// BiasOf is bullish or bearish only on a strict plurality (WM-INV-17).
func BiasOf(counts NewsCounts) NewsBias {
	switch {
	case counts.Positive > counts.Negative && counts.Positive > counts.Neutral:
		return NewsBullish
	case counts.Negative > counts.Positive && counts.Negative > counts.Neutral:
		return NewsBearish
	default:
		return NewsMixed
	}
}

type InvestmentAction string

const (
	ActionAllocate InvestmentAction = "ALLOCATE"
	ActionHedge    InvestmentAction = "HEDGE"
	ActionObserve  InvestmentAction = "OBSERVE"
)

// InvestmentCommand is one executable command of the think tank (WM-INV-18).
type InvestmentCommand struct {
	Action    InvestmentAction `json:"action"`
	Target    string           `json:"target"`
	Rationale string           `json:"rationale"`
}

// InvestmentCommands is the action prompt contract. The legacy prompts name
// fields exact_target, execution_logic and macro_hedge_reason; they are
// accepted and folded into Target and Rationale.
type InvestmentCommands struct {
	Commands []rawInvestmentCommand `json:"executable_commands"`
}

type rawInvestmentCommand struct {
	Action           string `json:"action"`
	Target           string `json:"target"`
	ExactTarget      string `json:"exact_target"`
	Rationale        string `json:"rationale"`
	ExecutionLogic   string `json:"execution_logic"`
	MacroHedgeReason string `json:"macro_hedge_reason"`
}

// Normalize validates the commands; an unknown action or a command without
// a target is ErrAIOutput.
func (c InvestmentCommands) Normalize() ([]InvestmentCommand, error) {
	commands := make([]InvestmentCommand, 0, len(c.Commands))
	for i, raw := range c.Commands {
		action := InvestmentAction(strings.ToUpper(strings.TrimSpace(raw.Action)))
		if action != ActionAllocate && action != ActionHedge && action != ActionObserve {
			return nil, fmt.Errorf("%w: command %d has unknown action %q", ErrAIOutput, i+1, raw.Action)
		}
		target := firstNonEmpty(raw.Target, raw.ExactTarget)
		if target == "" {
			return nil, fmt.Errorf("%w: command %d has no target", ErrAIOutput, i+1)
		}
		commands = append(commands, InvestmentCommand{
			Action:    action,
			Target:    target,
			Rationale: firstNonEmpty(raw.Rationale, raw.ExecutionLogic, raw.MacroHedgeReason),
		})
	}
	return commands, nil
}

func parseNewsSentiment(value NewsSentiment) NewsSentiment {
	for _, known := range []NewsSentiment{NewsPositive, NewsNegative, NewsNeutral} {
		if strings.EqualFold(strings.TrimSpace(string(value)), string(known)) {
			return known
		}
	}
	return value
}
