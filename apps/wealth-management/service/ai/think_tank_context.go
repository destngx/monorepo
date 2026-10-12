package ai

import (
	"apps/wealth-management/domain"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"
)

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
