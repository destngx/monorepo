package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	keyNewsTask          = "market/news-sentiment"
	keyTickerTask        = "tasks/ticker-insight"
	keyTickerAction      = "actions/ticker-insight"
	thinkTankContextSize = 8
	thinkTankTimeout     = 5 * time.Minute
	thinkTankRecentTx    = 15
	unavailable          = "N/A: not available"
	usUnavailable        = "N/A: US market coverage is not available (WM-INV-19)"
)

// marketPrompts are reported by Status when the market features are wired.
var marketPrompts = []string{
	keyNewsTask, keyTickerTask, keyTickerAction,
	phaseThinkTank.taskKey, phaseThinkTank.actionKey,
	phaseSynthesis.taskKey, phaseSynthesis.actionKey,
	phaseActions.taskKey, phaseActions.actionKey,
}

// newsFormatInstruction is the engine's output contract for news: the bias
// is computed from the per-article verdicts, never asked of the model.
const newsFormatInstruction = `Return only JSON, no prose:
{"articles":[{"index":0,"sentiment":"Positive","confidence":90,"impactScore":80,"summary":"One sentence."}]}
- Exactly one entry per article index above.
- sentiment is Positive (bullish), Negative (bearish) or Neutral for the topic.
- confidence and impactScore are 0-100: how sure you are, and how much the news can move the market.`

// commandsFormatInstruction is the action-phase contract (WM-INV-18).
const commandsFormatInstruction = `Return only JSON, no prose:
{"executable_commands":[{"action":"ALLOCATE","target":"...","rationale":"..."}]}
action is ALLOCATE, HEDGE or OBSERVE; every command needs a target.`

func (s *Service) requireSearch() error {
	if s.search == nil {
		return fmt.Errorf("%w: web search is not configured", domain.ErrAIUnavailable)
	}
	return nil
}

func (s *Service) requireMarket() error {
	if s.market == nil {
		return fmt.Errorf("%w: market data is not configured", domain.ErrAIUnavailable)
	}
	return nil
}

// NewsSentiment searches the topic, has the model read each article and
// derives the bias by strict plurality (WM-INV-17).
func (s *Service) NewsSentiment(ctx context.Context, request port.NewsRequest) (domain.NewsReport, error) {
	topic := strings.Join(strings.Fields(request.Topic), " ")
	if topic == "" {
		return domain.NewsReport{}, fmt.Errorf("%w: topic is required", domain.ErrValidation)
	}
	if err := errors.Join(s.requireConfigured(), s.requireSearch()); err != nil {
		return domain.NewsReport{}, err
	}
	query := domain.NewsQuery(topic, s.today().Year())
	found, err := s.search.Search(domain.WithBypassCache(ctx, request.Force), query)
	if err != nil {
		return domain.NewsReport{}, fmt.Errorf("search news: %w", err)
	}
	if len(found.Results) == 0 {
		return domain.NewsReport{
			Topic: topic, Query: query, Bias: domain.BiasOf(domain.NewsCounts{}), Articles: []domain.NewsArticle{},
			Provider: found.Provider, Fallback: found.Fallback, GeneratedAt: s.now(),
		}, nil
	}
	p, err := s.loadPrompts()
	if err != nil {
		return domain.NewsReport{}, err
	}
	system, err := p.render(keyNewsTask, map[string]string{"topic": topic, "year": fmt.Sprint(s.today().Year())})
	if err != nil {
		return domain.NewsReport{}, err
	}
	var articles strings.Builder
	fmt.Fprintf(&articles, "Articles about %q:\n", topic)
	for i, result := range found.Results {
		fmt.Fprintf(&articles, "\n[%d] Title: %s\nSnippet: %s\nSource: %s\n", i, result.Title, result.Snippet, result.Source)
	}
	messages := []domain.AIMessage{
		{Role: domain.AIRoleSystem, Content: system},
		{Role: domain.AIRoleUser, Content: articles.String() + "\n" + newsFormatInstruction},
	}
	model := s.models(domain.AIUseCaseInsight, request.Model)[0]
	return cached(ctx, s, "news-sentiment", request.Force, []any{model, messages}, func() (domain.NewsReport, error) {
		report := domain.NewsReport{Topic: topic, Query: query, Provider: found.Provider, Fallback: found.Fallback}
		_, err := s.complete(ctx, call{
			feature:  "news-sentiment",
			useCase:  domain.AIUseCaseInsight,
			model:    request.Model,
			prompts:  p.used,
			messages: messages,
			json:     true,
			validate: func(content string) error {
				var verdicts domain.NewsClassifications
				if err := domain.ParseAIJSON(content, &verdicts); err != nil {
					return err
				}
				var err error
				report.Articles, report.Counts, err = domain.ClassifyNews(found.Results, verdicts.Articles)
				return err
			},
		})
		report.Bias = domain.BiasOf(report.Counts)
		report.GeneratedAt = s.now()
		return report, err
	})
}

// TickerInsight comments on the deterministic analysis of one symbol; the
// figures come from the engine, the model only interprets them (WM-AI-14).
func (s *Service) TickerInsight(ctx context.Context, request port.TickerInsightRequest) (domain.Insight, error) {
	if err := s.requireMarket(); err != nil {
		return domain.Insight{}, err
	}
	timeframe, err := domain.ParseTimeframe(firstNonEmpty(request.Timeframe, string(domain.Timeframe1D)))
	if err != nil {
		return domain.Insight{}, err
	}
	analysis, err := s.market.Analysis(domain.WithBypassCache(ctx, request.Force), request.Symbol, timeframe)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("analyse %s: %w", strings.TrimSpace(request.Symbol), err)
	}
	snap, err := s.loadSnapshot()
	if err != nil {
		return domain.Insight{}, err
	}
	facts := map[string]any{"analysis": tickerFacts(analysis)}
	if s.search != nil {
		query := fmt.Sprintf("%s stock latest news %s", analysis.Symbol, s.today().Format("January 2006"))
		if found, err := s.search.Search(domain.WithBypassCache(ctx, request.Force), query); err != nil {
			facts["newsError"] = "web search failed: " + err.Error()
		} else {
			facts["news"] = found.Results
		}
	}
	technicals := analysis.Technicals
	return s.structuredInsight(ctx, snap, structuredRequest{
		feature: "ticker-insight",
		taskKey: keyTickerTask,
		taskVars: map[string]string{
			"symbol":    analysis.Symbol,
			"timeframe": string(analysis.Timeframe),
			"price":     fmt.Sprint(technicals.Price),
			"currency":  analysis.Currency,
			"trend":     technicals.Trend,
			"phase":     technicals.Phase,
			"action":    technicals.Signal.Action,
		},
		actionKey: keyTickerAction,
		facts:     facts,
		model:     request.Model,
		force:     request.Force,
	})
}

// tickerFacts trims the analysis to what the model needs: the best and
// worst ranked seasonality buckets per type instead of all of them.
func tickerFacts(analysis domain.MarketAnalysis) map[string]any {
	seasonality := []domain.SeasonalityBucket{}
	for _, bucketType := range []string{domain.SeasonalityDay, domain.SeasonalityWeek, domain.SeasonalityMonth} {
		var buckets []domain.SeasonalityBucket
		for _, bucket := range analysis.Seasonality {
			if bucket.Type == bucketType {
				buckets = append(buckets, bucket)
			}
		}
		if len(buckets) == 0 {
			continue
		}
		byRank := func(a, b domain.SeasonalityBucket) int { return a.Rank - b.Rank }
		best, worst := slices.MinFunc(buckets, byRank), slices.MaxFunc(buckets, byRank)
		seasonality = append(seasonality, best)
		if worst != best {
			seasonality = append(seasonality, worst)
		}
	}
	return map[string]any{
		"symbol":      analysis.Symbol,
		"timeframe":   analysis.Timeframe,
		"currency":    analysis.Currency,
		"provider":    analysis.Provider,
		"technicals":  analysis.Technicals,
		"confluence":  analysis.Confluence,
		"seasonality": seasonality,
		"risk":        analysis.Risk,
		"notes":       analysis.Notes,
	}
}
