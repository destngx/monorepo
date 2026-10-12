// Package yahoo reads the Yahoo Finance chart API: the candle fallback for
// vnstock-server (WM-INV-15) and a USD/VND market rate and series.
package yahoo

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/adapter/market/httpjson"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the public chart API host.
const DefaultBaseURL = "https://query1.finance.yahoo.com"

// Yahoo rejects requests without a browser-like user agent.
const userAgent = "Mozilla/5.0 (compatible; wealth-management/1.0)"

var intervals = map[domain.Timeframe]string{
	domain.Timeframe1H: "1h",
	domain.Timeframe1D: "1d",
	domain.Timeframe1W: "1wk",
}

type Provider struct {
	baseURL    string
	httpClient *http.Client
}

var (
	_ port.CandleProvider       = (*Provider)(nil)
	_ port.ExchangeRateProvider = (*Provider)(nil)
	_ port.PriceSeriesProvider  = (*Provider)(nil)
)

func NewProvider(config domain.MarketDataProviderConfig, log *logger.Logger) (*Provider, error) {
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("missing Yahoo base URL")
	}
	log.LogApplicationEvent(context.Background(), "initializing yahoo market provider",
		slog.String("base_url", config.BaseURL),
		slog.String("component", "yahoo"),
	)
	return &Provider{baseURL: strings.TrimRight(config.BaseURL, "/"), httpClient: httpjson.NewClient()}, nil
}

func (p *Provider) Name() string {
	return "yahoo"
}

func (p *Provider) Health(ctx context.Context) (domain.MarketProviderHealth, error) {
	now := time.Now()
	_, err := p.GetCandles(ctx, "VNINDEX", domain.CandleQuery{Timeframe: domain.Timeframe1D, From: now.AddDate(0, 0, -7), To: now})
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}
	return domain.MarketProviderHealth{Provider: p.Name(), Status: "ok"}, nil
}

// globalSymbols are the non-VN instruments the market pulse reads.
var globalSymbols = map[string]string{
	domain.MarketSymbolVIX:    "^VIX",
	domain.MarketSymbolUSDVND: "USDVND=X",
	domain.MarketSymbolXAUUSD: "GC=F",
}

// yahooSymbol maps a VN ticker to Yahoo: FPT → FPT.VN, VNINDEX → ^VNINDEX.VN.
// Symbols that already carry a Yahoo suffix or pair (BTC-USD) pass through.
func yahooSymbol(symbol string) (string, error) {
	if remote, ok := globalSymbols[symbol]; ok {
		return remote, nil
	}
	switch {
	case symbol == domain.MarketSymbolVNIndex:
		return "^VNINDEX.VN", nil
	case symbol == domain.MarketSymbolVN30 || symbol == "HNXINDEX" || symbol == "UPCOMINDEX" || symbol == domain.MarketSymbolSJC:
		return "", fmt.Errorf("%w: yahoo has no %s series", port.ErrUnsupported, symbol)
	case strings.ContainsAny(symbol, ".-=^"):
		return symbol, nil
	default:
		return symbol + ".VN", nil
	}
}

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// chart is one chart API answer: the quote metadata and its bars.
type chart struct {
	currency    string
	marketPrice float64
	marketTime  time.Time
	candles     []domain.Candle
}

// vietnamTime aligns index bars built from hourly data to HOSE trading days.
var vietnamTime = time.FixedZone("ICT", 7*60*60)

// maxHourlyHistory is how far back Yahoo serves 1h bars.
const maxHourlyHistory = 729 * 24 * time.Hour

func isVNIndex(remote string) bool {
	return strings.HasPrefix(remote, "^") && strings.HasSuffix(remote, ".VN")
}

func (p *Provider) GetCandles(ctx context.Context, symbol string, query domain.CandleQuery) (domain.CandleSeries, error) {
	interval, ok := intervals[query.Timeframe]
	if !ok {
		return domain.CandleSeries{}, fmt.Errorf("%w: yahoo does not serve %s candles", port.ErrUnsupported, query.Timeframe)
	}
	remote, err := yahooSymbol(symbol)
	if err != nil {
		return domain.CandleSeries{}, err
	}

	// Yahoo keeps only the latest daily and weekly bar for VN indices, but a
	// full hourly history: build those bars from 1h data.
	from := query.From
	aggregate := isVNIndex(remote) && query.Timeframe != domain.Timeframe1H
	if aggregate {
		interval = intervals[domain.Timeframe1H]
		from = maxTime(from, query.To.Add(-maxHourlyHistory))
	}
	result, err := p.chart(ctx, remote, interval, from, query.To)
	if err != nil {
		return domain.CandleSeries{}, err
	}
	candles := result.candles
	if aggregate {
		candles = domain.AggregateCandles(domain.CleanCandles(candles), query.Timeframe, vietnamTime)
	}
	return domain.CandleSeries{
		Symbol:    symbol,
		Timeframe: query.Timeframe,
		Provider:  p.Name(),
		Currency:  result.currency,
		Candles:   candles,
	}, nil
}

// GetExchangeRate quotes a fiat pair from the FX market, e.g. USDVND=X.
func (p *Provider) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	from, to = strings.ToUpper(strings.TrimSpace(from)), strings.ToUpper(strings.TrimSpace(to))
	if len(from) != 3 || len(to) != 3 {
		return domain.ExchangeRate{}, fmt.Errorf("yahoo quotes only ISO currency pairs")
	}
	now := time.Now()
	result, err := p.chart(ctx, from+to+"=X", "1d", now.AddDate(0, 0, -7), now)
	if err != nil {
		return domain.ExchangeRate{}, err
	}
	if result.marketPrice <= 0 {
		return domain.ExchangeRate{}, fmt.Errorf("yahoo has no %s/%s quote", from, to)
	}
	return domain.ExchangeRate{
		From:      from,
		To:        to,
		Rate:      result.marketPrice,
		Provider:  p.Name(),
		Timestamp: result.marketTime.Format(time.RFC3339),
	}, nil
}

// GetPriceSeries serves the daily close of a fiat pair over 30 days
// (WM-MKT-3); symbol is a pair such as USD_VND.
func (p *Provider) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	if seriesType != domain.SeriesTypeForex {
		return domain.PriceSeries{}, fmt.Errorf("yahoo serves only %s price series", domain.SeriesTypeForex)
	}
	pair := strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(symbol)), "/", ""), "_", "")
	if len(pair) != 6 {
		return domain.PriceSeries{}, fmt.Errorf("%w: forex symbol must be a pair such as USD_VND", domain.ErrValidation)
	}
	now := time.Now()
	result, err := p.chart(ctx, pair+"=X", "1d", now.AddDate(0, 0, -30), now)
	if err != nil {
		return domain.PriceSeries{}, err
	}
	points := make([]domain.PricePoint, 0, len(result.candles))
	for _, candle := range domain.CleanCandles(result.candles) {
		points = append(points, domain.PricePoint{Price: candle.Close, Timestamp: candle.Time.Format(time.RFC3339)})
	}
	if len(points) == 0 {
		return domain.PriceSeries{}, fmt.Errorf("yahoo returned an empty %s series", pair)
	}
	return domain.PriceSeries{
		Provider:   p.Name(),
		SeriesType: seriesType,
		Currency:   result.currency,
		Data:       map[string]any{"pair": pair[:3] + "_" + pair[3:], "points": points},
	}, nil
}

func (p *Provider) chart(ctx context.Context, remote string, interval string, from time.Time, to time.Time) (chart, error) {
	params := url.Values{
		"interval": {interval},
		"period1":  {strconv.FormatInt(from.Unix(), 10)},
		"period2":  {strconv.FormatInt(to.Unix(), 10)},
	}
	endpoint := fmt.Sprintf("%s/v8/finance/chart/%s?%s", p.baseURL, url.PathEscape(remote), params.Encode())

	var payload chartResponse
	if err := httpjson.Get(ctx, p.httpClient, "yahoo", endpoint, map[string]string{"User-Agent": userAgent}, &payload); err != nil {
		return chart{}, err
	}
	if payload.Chart.Error != nil {
		return chart{}, fmt.Errorf("yahoo %s: %s", payload.Chart.Error.Code, payload.Chart.Error.Description)
	}
	if len(payload.Chart.Result) == 0 || len(payload.Chart.Result[0].Indicators.Quote) == 0 {
		return chart{}, fmt.Errorf("yahoo returned no chart for %s", remote)
	}
	result := payload.Chart.Result[0]
	quote := result.Indicators.Quote[0]

	candles := make([]domain.Candle, 0, len(result.Timestamp))
	for i, at := range result.Timestamp {
		candles = append(candles, domain.Candle{
			Time:   time.Unix(at, 0).UTC(),
			Open:   valueAt(quote.Open, i),
			High:   valueAt(quote.High, i),
			Low:    valueAt(quote.Low, i),
			Close:  valueAt(quote.Close, i),
			Volume: valueAt(quote.Volume, i),
		})
	}
	return chart{
		currency:    result.Meta.Currency,
		marketPrice: result.Meta.RegularMarketPrice,
		marketTime:  time.Unix(result.Meta.RegularMarketTime, 0).UTC(),
		candles:     candles,
	}, nil
}

func maxTime(a time.Time, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// valueAt reads a nullable column; Yahoo sends null for bars without trades,
// which CleanCandles then drops.
func valueAt(values []*float64, index int) float64 {
	if index >= len(values) || values[index] == nil {
		return 0
	}
	return *values[index]
}
