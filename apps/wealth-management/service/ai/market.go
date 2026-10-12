package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
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

// thinkTankPhase is one step of the investment analysis with its legacy
// prompt keys (WM-AI-9). The fallback task is optional and smaller; its
// action falls back to the main action.
type thinkTankPhase struct {
	name                         string
	title                        string
	taskKey, actionKey           string
	fallbackTask, fallbackAction string
}

var (
	phaseThinkTank = thinkTankPhase{"think-tank", "Think tank", "market/think-tank", "actions/investment-think-tank", "market/fallback-think-tank", "actions/investment-think-tank-fallback"}
	phaseSynthesis = thinkTankPhase{"synthesis", "Synthesis", "market/synthesis", "actions/investment-synthesis", "market/fallback-synthesis", "actions/investment-synthesis-fallback"}
	phaseActions   = thinkTankPhase{"actions", "Actions", "market/action", "actions/investment-action", "market/fallback-action", "actions/investment-action-fallback"}
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

// thinkTankInputs is the raw context of an investment analysis; it is sent
// to the UI as-is for disclosure (WM-INV-18).
type thinkTankInputs struct {
	Valuation          *domain.PortfolioValuation `json:"valuation,omitempty"`
	ValuationError     string                     `json:"valuationError,omitempty"`
	Accounts           []accountContext           `json:"accounts"`
	Loans              []loanContext              `json:"loans"`
	RecentTransactions []txContext                `json:"recentTransactions"`
	Budget             budgetLimits               `json:"budget"`
	USDTRate           *domain.ExchangeRate       `json:"usdtRate,omitempty"`
	Searches           []domain.SearchResponse    `json:"searches"`
	SearchErrors       []string                   `json:"searchErrors,omitempty"`
	SearchContext      string                     `json:"searchContext"`
	Pulse              *domain.MarketPulse        `json:"pulse,omitempty"`
	PulseError         string                     `json:"pulseError,omitempty"`
	News               []domain.NewsReport        `json:"news,omitempty"`
}

// budgetLimits are this month's planned income and expenses (legacy input).
type budgetLimits struct {
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
}

// InvestmentAnalysis streams context → think tank → synthesis → commands.
// Each phase retries with its fallback prompt pair when the main one fails
// (WM-AI-9). Errors after the stream started are sent as an error event.
func (s *Service) InvestmentAnalysis(ctx context.Context, request port.InvestmentAnalysisRequest, emit func(port.InvestmentEvent) error) error {
	if err := errors.Join(s.requireConfigured(), s.requireMarket()); err != nil {
		return err
	}
	if s.portfolio == nil {
		return fmt.Errorf("%w: portfolio valuation is not configured", domain.ErrAIUnavailable)
	}
	ctx, cancel := context.WithTimeout(ctx, thinkTankTimeout)
	defer cancel()
	snap, err := s.loadSnapshot()
	if err != nil {
		return err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: "context", Status: "running"}); err != nil {
		return err
	}
	inputs := s.thinkTankInputs(ctx, snap, request.News)
	if err := emit(port.InvestmentEvent{Type: "context", Context: inputs}); err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: "context", Status: "ok"}); err != nil {
		return err
	}
	vars := s.thinkTankVars(snap, inputs)

	debate, model, err := s.runPhase(ctx, p, snap, phaseThinkTank, vars, map[string]string{"totalVND": vars["totalVND"], "totalCrypto": vars["totalCrypto"]}, request.Model, emit)
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "message", Phase: phaseThinkTank.name, Name: phaseThinkTank.title, Content: debate, Model: model}); err != nil {
		return err
	}

	synthesisVars := pick(vars, "totalVND", "totalCrypto", "income", "expenses", "p2pRate", "accountsSummary", "loans", "recentTransactions")
	synthesisVars["expertDebateContext"] = debate
	synthesis, model, err := s.runPhase(ctx, p, snap, phaseSynthesis, synthesisVars, map[string]string{"accountsSummary": vars["accountsSummary"], "debate": debate}, request.Model, emit)
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "message", Phase: phaseSynthesis.name, Name: phaseSynthesis.title, Content: synthesis, Model: model}); err != nil {
		return err
	}

	actionVars := pick(vars, "accountsSummary", "cryptoHoldings", "ifcHoldings")
	actionVars["synthesisContext"] = synthesis
	var commands []domain.InvestmentCommand
	parse := func(content string) error {
		var raw domain.InvestmentCommands
		if err := domain.ParseAIJSON(content, &raw); err != nil {
			return err
		}
		normalized, err := raw.Normalize()
		commands = normalized
		return err
	}
	_, model, err = s.runPhaseWith(ctx, p, snap, phaseActions, actionVars, map[string]string{"synthesis": synthesis}, request.Model, true, parse, emit)
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "actions", Phase: phaseActions.name, Name: phaseActions.title, Commands: commands, Model: model}); err != nil {
		return err
	}
	return emit(port.InvestmentEvent{Type: "done"})
}

func (s *Service) runPhase(ctx context.Context, p *prompts, snap snapshot, phase thinkTankPhase, vars map[string]string, fallbackVars map[string]string, override string, emit func(port.InvestmentEvent) error) (string, string, error) {
	return s.runPhaseWith(ctx, p, snap, phase, vars, fallbackVars, override, false, func(content string) error {
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("%w: %s reply is empty", domain.ErrAIOutput, phase.name)
		}
		return nil
	}, emit)
}

// runPhaseWith runs the main prompt pair (with the model fallback) and, if
// that fails, the fallback pair. It returns the reply and the model used.
func (s *Service) runPhaseWith(ctx context.Context, p *prompts, snap snapshot, phase thinkTankPhase, vars map[string]string, fallbackVars map[string]string, override string, json bool, validate func(string) error, emit func(port.InvestmentEvent) error) (string, string, error) {
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "running"}); err != nil {
		return "", "", err
	}
	attempt := func(taskKey, actionKey string, vars map[string]string) (domain.AIResponse, error) {
		task, err := p.render(taskKey, vars)
		if err != nil {
			return domain.AIResponse{}, err
		}
		action, err := p.render(actionKey, nil)
		if err != nil {
			return domain.AIResponse{}, err
		}
		if json {
			action += "\n\n" + commandsFormatInstruction
		}
		system, err := s.systemPrompt(p, snap, task)
		if err != nil {
			return domain.AIResponse{}, err
		}
		return s.complete(ctx, call{
			feature:  "think-tank:" + phase.name,
			useCase:  domain.AIUseCaseInsight,
			model:    override,
			prompts:  []string{keyIdentity, taskKey, actionKey},
			messages: []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, {Role: domain.AIRoleUser, Content: action}},
			json:     json,
			validate: validate,
		})
	}
	response, err := attempt(phase.taskKey, phase.actionKey, vars)
	if err != nil && ctx.Err() == nil {
		if _, ok := p.lookup(phase.fallbackTask, nil); ok {
			if emitErr := emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "fallback", Message: phaseMessage(err)}); emitErr != nil {
				return "", "", emitErr
			}
			actionKey := phase.fallbackAction
			if _, ok := p.lookup(actionKey, nil); !ok {
				actionKey = phase.actionKey
			}
			if fallback, fallbackErr := attempt(phase.fallbackTask, actionKey, fallbackVars); fallbackErr == nil {
				response, err = fallback, nil
			} else {
				err = errors.Join(err, fallbackErr)
			}
		}
	}
	if err != nil {
		_ = emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "error", Message: phaseMessage(err)})
		return "", "", fmt.Errorf("%s phase: %w", phase.name, err)
	}
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "ok"}); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(response.Content), firstNonEmpty(response.Model, s.models(domain.AIUseCaseInsight, override)[0]), nil
}

func phaseMessage(err error) string {
	if IsUserFacing(err) {
		return err.Error()
	}
	return "the model call failed"
}

// thinkTankInputs gathers valuation, searches and the market pulse in
// parallel; a failed source is recorded, not fatal.
func (s *Service) thinkTankInputs(ctx context.Context, snap snapshot, news []domain.NewsReport) thinkTankInputs {
	inputs := thinkTankInputs{
		Accounts:           s.accountContexts(snap.accounts),
		Loans:              s.loanContexts(snap.loans),
		RecentTransactions: s.txContexts(recent(snap.transactions, time.Time{}, snap.today, thinkTankRecentTx)),
		Budget:             monthLimits(snap.budget),
		News:               news,
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	wg.Go(func() {
		pulse, err := s.market.Pulse(ctx, domain.Timeframe1D)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			inputs.PulseError = err.Error()
			return
		}
		inputs.Pulse = &pulse
	})
	valuation, err := s.portfolio.Valuation(ctx, false)
	if err != nil {
		inputs.ValuationError = err.Error()
	} else {
		inputs.Valuation = &valuation
		for _, rate := range valuation.Rates {
			if rate.From == "USDT" && rate.To == "VND" {
				inputs.USDTRate = &rate
			}
		}
	}
	queries := thinkTankQueries(inputs.Valuation, snap.accounts, snap.today)
	searches := make([]*domain.SearchResponse, len(queries))
	searchErrors := make([]string, len(queries))
	if s.search != nil {
		for i, query := range queries {
			wg.Go(func() {
				found, err := s.search.Search(ctx, query)
				if err != nil {
					searchErrors[i] = fmt.Sprintf("%s: %v", query, err)
					return
				}
				searches[i] = &found
			})
		}
	} else {
		searchErrors = []string{"web search is not configured"}
	}
	wg.Wait()

	seen := map[string]bool{}
	var results []domain.SearchResult
	for _, found := range searches {
		if found == nil {
			continue
		}
		inputs.Searches = append(inputs.Searches, *found)
		for _, result := range found.Results {
			if !seen[result.URL] && len(results) < thinkTankContextSize {
				seen[result.URL] = true
				results = append(results, result)
			}
		}
	}
	inputs.SearchErrors = slices.DeleteFunc(searchErrors, func(message string) bool { return message == "" })
	inputs.SearchContext = searchContext(results)
	return inputs
}

// thinkTankQueries are the three legacy searches: earn rates, world macro,
// and either the held coins or the SBV policy.
func thinkTankQueries(valuation *domain.PortfolioValuation, accounts []domain.Account, today time.Time) []string {
	month := today.Format("January 2006")
	coins := []string{}
	if valuation != nil {
		for _, holding := range valuation.Holdings {
			if holding.Section == domain.PortfolioSectionCrypto && holding.Symbol != "" && !slices.Contains(coins, holding.Symbol) {
				coins = append(coins, holding.Symbol)
			}
		}
	}
	for _, account := range accounts {
		if code := string(account.Currency); slices.Contains([]string{"BTC", "ETH", "SOL"}, code) && !slices.Contains(coins, code) {
			coins = append(coins, code)
		}
	}
	queries := []string{
		"Binance Flexible Earn APY interest rates " + month + " USDT USDC ETH SOL",
		"World Macro Breaking News Geopolitics FED interest rate " + month,
	}
	if len(coins) == 0 {
		return append(queries, "Vietnam SBV state bank policy status "+month)
	}
	return append(queries, "crypto forecast "+strings.Join(coins[:min(5, len(coins))], " ")+" "+month)
}

func searchContext(results []domain.SearchResult) string {
	if len(results) == 0 {
		return "[NO WEB RESULTS: every search failed or none is configured. Do not cite news; say the news context is missing.]"
	}
	var builder strings.Builder
	for i, result := range results {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		fmt.Fprintf(&builder, "[%d] %q\n    %s\n    Source: %s", i+1, result.Title, result.Snippet, result.URL)
	}
	return builder.String()
}

func monthLimits(report domain.BudgetReport) budgetLimits {
	var limits budgetLimits
	for _, line := range report.Lines {
		switch line.CategoryType {
		case domain.CategoryTypeIncome:
			limits.Income += line.Limit
		case domain.CategoryTypeExpense:
			limits.Expenses += line.Limit
		}
	}
	return limits
}

// thinkTankVars fills the legacy market/think-tank placeholders. Values the
// engine does not have say so instead of being invented.
func (s *Service) thinkTankVars(snap snapshot, inputs thinkTankInputs) map[string]string {
	vars := map[string]string{
		"totalVND":           unavailable,
		"totalCrypto":        "None",
		"income":             amount(inputs.Budget.Income),
		"expenses":           amount(inputs.Budget.Expenses),
		"p2pRate":            unavailable,
		"accountsSummary":    toJSON(inputs.Accounts),
		"loans":              toJSON(inputs.Loans),
		"recentTransactions": toJSON(inputs.RecentTransactions),
		"cryptoHoldings":     "[]",
		"ifcHoldings":        "[]",
		"prices":             "{}",
		"newsAnalysis":       "N/A: No pre-analyzed news available.",
		"searchContext":      inputs.SearchContext,
	}
	if inputs.USDTRate != nil {
		vars["p2pRate"] = amount(inputs.USDTRate.Rate)
	}
	if len(inputs.News) > 0 {
		vars["newsAnalysis"] = toJSON(inputs.News)
	}
	if valuation := inputs.Valuation; valuation != nil {
		vars["totalVND"] = amount(valuation.Totals.VND)
		crypto, funds, coins := []map[string]any{}, []map[string]any{}, []string{}
		prices := map[string]float64{}
		for _, holding := range valuation.Holdings {
			row := map[string]any{"platform": s.masker.Mask(holding.Platform), "symbol": holding.Symbol, "quantity": holding.Quantity, "valueVND": holding.ValueVND}
			if holding.LivePrice != nil {
				prices[holding.Symbol] = holding.LivePrice.Price
			}
			if holding.Section == domain.PortfolioSectionCrypto {
				crypto = append(crypto, row)
				if holding.Quantity != nil {
					coins = append(coins, fmt.Sprintf("%g %s", *holding.Quantity, holding.Symbol))
				}
			} else {
				funds = append(funds, row)
			}
		}
		vars["cryptoHoldings"], vars["ifcHoldings"], vars["prices"] = toJSON(crypto), toJSON(funds), toJSON(prices)
		if len(coins) > 0 {
			vars["totalCrypto"] = strings.Join(coins, ", ")
		}
	}
	for _, key := range []string{"usRegimeName", "usRegimeType", "usRegimeConfidence", "usAssets", "usDrivers", "usCapitalFlowSignal", "usCapitalFlowSummary", "usCorrelationMatrix", "usScenarioSummary"} {
		vars[key] = usUnavailable
	}
	vn := map[string]string{"vnRegimeConfidence": unavailable, "vnDrivers": unavailable}
	if pulse := inputs.Pulse; pulse != nil {
		assets := []string{}
		for _, asset := range pulse.Assets {
			if asset.Error == "" {
				assets = append(assets, fmt.Sprintf("%s: %g (%.2f%%)", asset.Name, asset.Price, asset.ChangePct))
			}
		}
		maps.Copy(vn, map[string]string{
			"vnRegimeName":         pulse.Scenario.Name,
			"vnRegimeType":         pulse.Scenario.Regime,
			"vnAssets":             strings.Join(assets, ", "),
			"vnCapitalFlowSignal":  pulse.CapitalFlow.Signal,
			"vnCapitalFlowSummary": pulse.CapitalFlow.Summary,
			"vnCorrelationMatrix":  correlationText(pulse.Correlation),
			"vnScenarioSummary":    pulse.Scenario.Summary,
		})
	} else {
		for _, key := range []string{"vnRegimeName", "vnRegimeType", "vnAssets", "vnCapitalFlowSignal", "vnCapitalFlowSummary", "vnCorrelationMatrix", "vnScenarioSummary"} {
			vn[key] = "N/A: market pulse failed (" + inputs.PulseError + ")"
		}
	}
	maps.Copy(vars, vn)
	return vars
}

func correlationText(matrix domain.CorrelationMatrix) string {
	if len(matrix.Symbols) == 0 {
		return unavailable
	}
	var builder strings.Builder
	builder.WriteString("ASSET CORRELATION MATRIX (daily returns, n/a = too few shared bars):\nColumns: " + strings.Join(matrix.Symbols, " | ") + "\n")
	for i, symbol := range matrix.Symbols {
		cells := make([]string, len(matrix.Values[i]))
		for j, value := range matrix.Values[i] {
			cells[j] = "n/a"
			if value != nil {
				cells[j] = fmt.Sprintf("%.2f", *value)
			}
		}
		fmt.Fprintf(&builder, "%-10s: %s\n", symbol, strings.Join(cells, " | "))
	}
	return builder.String()
}

func pick(vars map[string]string, keys ...string) map[string]string {
	out := make(map[string]string, len(keys)+1)
	for _, key := range keys {
		out[key] = vars[key]
	}
	return out
}

// marketTools let the chat search the web and read a ticker analysis; each
// exists only when its service is wired.
func (s *Service) marketTools(str func(string) map[string]any) []chatTool {
	var tools []chatTool
	if s.search != nil {
		tools = append(tools, chatTool{
			definition: domain.AITool{Name: "webSearch", Description: "Search the web for recent news, rates or facts. Cite the sources you use.", Parameters: objectSchema(map[string]any{"query": str("Search query")})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var input struct {
					Query string `json:"query"`
				}
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return s.search.Search(ctx, input.Query)
			},
		})
	}
	if s.market != nil {
		tools = append(tools, chatTool{
			definition: domain.AITool{Name: "getMarketAnalysis", Description: "Technicals, multi-timeframe confluence, seasonality and risk for a VN stock, index, gold or FX symbol.", Parameters: objectSchema(map[string]any{
				"symbol":    str("Symbol such as FPT, VNINDEX, SJC or USDVND"),
				"timeframe": str("1h, 4h, 1d or 1w; default 1d"),
			})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var input struct {
					Symbol    string `json:"symbol"`
					Timeframe string `json:"timeframe"`
				}
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				timeframe, err := domain.ParseTimeframe(firstNonEmpty(input.Timeframe, string(domain.Timeframe1D)))
				if err != nil {
					return nil, err
				}
				analysis, err := s.market.Analysis(ctx, input.Symbol, timeframe)
				if err != nil {
					return nil, err
				}
				return tickerFacts(analysis), nil
			},
		})
	}
	return tools
}
