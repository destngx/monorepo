package marketdata

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

// fakeServer answers every request with handler and records the requests.
func fakeServer(t *testing.T, handler func(r *http.Request) (int, any)) (*Client, *[]*http.Request) {
	t.Helper()
	var requests []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r)
		status, body := handler(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	return client, &requests
}

func answer(data any, meta map[string]any) map[string]any {
	full := map[string]any{"provider": "vnstock", "fallback": []string{}, "cached": false, "stale": false, "as_of": "2026-10-14T01:00:00Z"}
	for key, value := range meta {
		full[key] = value
	}
	return map[string]any{"data": data, "meta": full}
}

func TestNewClientRequiresAnAbsoluteURL(t *testing.T) {
	for _, raw := range []string{"", "localhost:8000", "/api"} {
		if _, err := NewClient(raw); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%q: err %v, want ErrValidation", raw, err)
		}
	}
}

func TestGetTickerDecodesTheEnvelope(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer(map[string]any{"symbol": "FPT", "type": "equity", "provider": "vnstock", "currency": "VND", "price": 57.1}, nil)
	})
	ticker, err := client.GetTicker(context.Background(), "fpt", domain.TickerTypeEquity)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Ticker{Symbol: "FPT", Type: domain.TickerTypeEquity, Provider: "vnstock", Currency: "VND", Price: 57.1,
		Source: &domain.DataSource{Provider: "vnstock", AsOf: "2026-10-14T01:00:00Z"}}
	if !reflect.DeepEqual(ticker, want) {
		t.Fatalf("ticker = %+v", ticker)
	}
	request := (*requests)[0]
	if request.URL.Path != "/api/v2/ticker" || request.URL.Query().Get("symbol") != "fpt" || request.URL.Query().Get("type") != "equity" {
		t.Fatalf("request = %s", request.URL)
	}
	if request.URL.Query().Has("skip_cache") {
		t.Fatal("skip_cache sent without a bypass")
	}
}

func TestBypassCacheBecomesSkipCache(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer(map[string]any{"from": "USD", "to": "VND", "rate": 26080, "provider": "fmarket", "basis": "bank_sell"}, nil)
	})
	rate, err := client.GetExchangeRate(domain.WithBypassCache(context.Background(), true), "USD", "VND")
	if err != nil || rate.Rate != 26080 || rate.From != "USD" || rate.Basis != domain.RateBasisBankSell || rate.Source.Provider != "vnstock" {
		t.Fatalf("rate = %+v, err %v", rate, err)
	}
	query := (*requests)[0].URL.Query()
	if query.Get("skip_cache") != "true" || query.Get("from") != "USD" || query.Get("to") != "VND" {
		t.Fatalf("query = %v", query)
	}
}

func TestStaleAnswersAreMarkedOnTheRequestAndTheSource(t *testing.T) {
	client, _ := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer([]any{}, map[string]any{"provider": "fmarket", "stale": true, "as_of": "2026-10-13T08:00:00Z"})
	})
	recorder := &domain.StaleRecorder{}
	rates, err := client.GetBankInterestRate(domain.WithStaleRecorder(context.Background(), recorder))
	if err != nil {
		t.Fatal(err)
	}
	asOf, stale := recorder.AsOf()
	if !stale || !asOf.Equal(time.Date(2026, 10, 13, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("stale = %v as of %v", stale, asOf)
	}
	want := &domain.DataSource{Provider: "fmarket", Stale: true, AsOf: "2026-10-13T08:00:00Z"}
	if !reflect.DeepEqual(rates.Source, want) || len(rates.Rates) != 0 {
		t.Fatalf("rates = %+v", rates)
	}
}

func TestEveryAnswerCarriesItsSource(t *testing.T) {
	client, _ := fakeServer(t, func(r *http.Request) (int, any) {
		return 200, answer(map[string]any{"provider": "yahoo"}, map[string]any{"provider": "yahoo", "fallback": []string{"fmarket: down"}})
	})
	want := &domain.DataSource{Provider: "yahoo", Fallback: []string{"fmarket: down"}, AsOf: "2026-10-14T01:00:00Z"}
	series, err := client.GetPriceSeries(context.Background(), "USD_VND", domain.SeriesTypeForex)
	if err != nil || !reflect.DeepEqual(series.Source, want) {
		t.Fatalf("series = %+v, err %v", series, err)
	}
	rate, err := client.GetExchangeRate(context.Background(), "USD", "VND")
	if err != nil || !reflect.DeepEqual(rate.Source, want) {
		t.Fatalf("rate = %+v, err %v", rate, err)
	}
}

func TestErrorAnswersMapToDomainErrors(t *testing.T) {
	cases := []struct {
		status int
		code   string
		want   error
	}{
		{400, "validation", domain.ErrValidation},
		{404, "not_found", domain.ErrNotFound},
		{502, "upstream", nil},
	}
	for _, tc := range cases {
		client, _ := fakeServer(t, func(*http.Request) (int, any) {
			return tc.status, map[string]any{"error": map[string]any{"code": tc.code, "message": "get ticker failed: vnstock: down"}}
		})
		_, err := client.GetTicker(context.Background(), "FPT", domain.TickerTypeEquity)
		var serverErr *ServerError
		if !errors.As(err, &serverErr) || serverErr.Status != tc.status || serverErr.Code != tc.code {
			t.Fatalf("%d: err = %v", tc.status, err)
		}
		if tc.want != nil && !errors.Is(err, tc.want) {
			t.Fatalf("%d: err %v does not wrap %v", tc.status, err, tc.want)
		}
		if tc.want == nil && (errors.Is(err, domain.ErrValidation) || errors.Is(err, domain.ErrNotFound)) {
			t.Fatalf("%d: an upstream failure must not be a client error", tc.status)
		}
		if err.Error() != "market data server: get ticker failed: vnstock: down" {
			t.Fatalf("message = %q", err.Error())
		}
	}
}

func TestUnreachableServerIsAnError(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetPriceSeries(context.Background(), "USD_VND", domain.SeriesTypeForex); err == nil {
		t.Fatal("expected an error")
	}
}

func TestGetCandlesCarriesTheFallbackIndicator(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		series := map[string]any{"symbol": "FPT", "timeframe": "4h", "provider": "yahoo", "currency": "VND", "candles": []any{
			map[string]any{"time": "2026-10-12T02:00:00Z", "open": 1, "high": 2, "low": 0.5, "close": 1.5, "volume": 10},
		}}
		return 200, answer(series, map[string]any{"provider": "yahoo", "fallback": []string{"vnstock: quota", "fmarket: down"}})
	})
	series, err := client.GetCandles(context.Background(), "FPT", domain.Timeframe4H, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(series.Source.Fallback, []string{"vnstock: quota", "fmarket: down"}) || series.Source.Provider != "yahoo" || series.Provider != "yahoo" || series.Timeframe != domain.Timeframe4H {
		t.Fatalf("series = %+v", series)
	}
	if len(series.Candles) != 1 || !series.Candles[0].Time.Equal(time.Date(2026, 10, 12, 2, 0, 0, 0, time.UTC)) || series.Candles[0].Close != 1.5 {
		t.Fatalf("candles = %+v", series.Candles)
	}
	query := (*requests)[0].URL.Query()
	if query.Get("timeframe") != "4h" || query.Get("days") != "30" {
		t.Fatalf("query = %v", query)
	}
}

func TestHealthNamesTheProvider(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, map[string]any{"data": map[string]any{"provider": "fmarket", "status": "ok"}}
	})
	health, err := client.Health(context.Background(), " FMarket ")
	if err != nil || health != (domain.MarketProviderHealth{Provider: "fmarket", Status: "ok"}) {
		t.Fatalf("health = %+v, err %v", health, err)
	}
	if (*requests)[0].URL.Path != "/api/v2/providers/fmarket/health" {
		t.Fatalf("path = %s", (*requests)[0].URL.Path)
	}
	if _, err := client.Health(context.Background(), ""); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("empty provider: err %v", err)
	}
}

func TestFmarketActionsMapToServerEndpoints(t *testing.T) {
	cases := []struct {
		name   string
		action domain.FmarketAction
		params map[string]any
		path   string
		query  url.Values
	}{
		{"funds", domain.FmarketActionProductsFilterNav, map[string]any{"page": float64(1), "pageSize": float64(100), "assetTypes": []any{"BOND"}, "isMMFFund": false},
			"/api/v2/funds", url.Values{"page": {"1"}, "page_size": {"100"}, "asset_type": {"BOND"}}},
		{"money market funds", domain.FmarketActionProductsFilterNav, map[string]any{"assetTypes": []any{}, "isMMFFund": true},
			"/api/v2/funds", url.Values{"mmf": {"true"}}},
		{"issuers", domain.FmarketActionIssuers, nil, "/api/v2/funds/issuers", url.Values{}},
		{"product", domain.FmarketActionProductDetails, map[string]any{"code": "DCDS"}, "/api/v2/funds/DCDS", url.Values{}},
		{"nav history", domain.FmarketActionNavHistory, map[string]any{"productId": float64(28)}, "/api/v2/funds/28/nav", url.Values{}},
		{"usd history", domain.FmarketActionUsdRateHistory, map[string]any{"fromDate": "2026-09-01", "toDate": "14/10/2026"},
			"/api/v2/fx/history", url.Values{"from": {"20260901"}, "to": {"20261014"}}},
		{"gold products", domain.FmarketActionGoldProducts, nil, "/api/v2/gold/products", url.Values{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, requests := fakeServer(t, func(*http.Request) (int, any) {
				return 200, answer(map[string]any{"rows": []any{}}, nil)
			})
			result, err := client.RunAction(context.Background(), domain.FmarketRequest{Action: tc.action, Params: tc.params}, false)
			if err != nil {
				t.Fatal(err)
			}
			want := fmarketPayload{Data: map[string]any{"rows": []any{}}, Source: &domain.DataSource{Provider: "vnstock", AsOf: "2026-10-14T01:00:00Z"}}
			if !reflect.DeepEqual(result, want) {
				t.Fatalf("result = %#v", result)
			}
			request := (*requests)[0]
			if request.URL.Path != tc.path || !reflect.DeepEqual(request.URL.Query(), tc.query) {
				t.Fatalf("request = %s, want %s?%s", request.URL, tc.path, tc.query.Encode())
			}
		})
	}
}

func TestGoldHistoryRestoresFmarketsEnvelope(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer(map[string]any{"rows": []any{map[string]any{"price": 4150.26}}, "summary": map[string]any{"goldRingPriceBuy": 142000000}}, map[string]any{"provider": "fmarket"})
	})
	result, err := client.RunAction(context.Background(), domain.FmarketRequest{Action: domain.FmarketActionGoldPriceHistory}, true)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"data":[{"price":4150.26}],"extra":{"goldRingPriceBuy":142000000},"source":{"provider":"fmarket","asOf":"2026-10-14T01:00:00Z"}}`
	if string(encoded) != want {
		t.Fatalf("result = %s", encoded)
	}
	if query := (*requests)[0].URL.Query(); !reflect.DeepEqual(query, url.Values{"skip_cache": {"true"}}) {
		t.Fatalf("forceFresh must bypass the server cache, query = %v", query)
	}
}

func TestBankRatesActionStaysTyped(t *testing.T) {
	client, _ := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer([]any{map[string]any{"bank": "VCB", "tenor": "", "rate": 4.6, "currency": "VND", "updatedAt": "2026-10-10"}}, nil)
	})
	result, err := client.RunAction(context.Background(), domain.FmarketRequest{Action: domain.FmarketActionBankInterestRates}, false)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.BankRates{
		Rates:  []domain.BankRate{{Bank: "VCB", Rate: 4.6, Currency: "VND", UpdatedAt: "2026-10-10"}},
		Source: &domain.DataSource{Provider: "vnstock", AsOf: "2026-10-14T01:00:00Z"},
	}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("result = %#v", result)
	}
}

func TestFmarketActionValidation(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) { return 200, answer(nil, nil) })
	for _, request := range []domain.FmarketRequest{
		{Action: "nope"},
		{Action: domain.FmarketActionProductDetails, Params: map[string]any{"code": " "}},
		{Action: domain.FmarketActionNavHistory, Params: map[string]any{"productId": "x"}},
	} {
		if _, err := client.RunAction(context.Background(), request, false); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%+v: err %v", request, err)
		}
	}
	if len(*requests) != 0 {
		t.Fatalf("invalid actions reached the server: %d requests", len(*requests))
	}
}

func TestLegacyDate(t *testing.T) {
	for raw, want := range map[string]string{"": "", "20261014": "20261014", "2026-10-14": "20261014", "14/10/2026": "20261014", "10/31/2026": "20261031", "soon": "soon"} {
		if got := legacyDate(raw); got != want {
			t.Fatalf("legacyDate(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestGetFundNAVHistoryReadsFmarketRowsByFundCode(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer([]map[string]any{
			{"nav": 12_000.5, "navDate": "2026-10-12"},
			{"nav": 11_000, "navDate": float64(time.Date(2026, 10, 9, 17, 30, 0, 0, time.UTC).UnixMilli())},
			{"nav": 0, "navDate": "2026-10-08"},
			{"nav": "x", "navDate": "2026-10-07"},
			{"nav": 10_000, "navDate": "n/a"},
		}, map[string]any{"provider": "fmarket"})
	})

	history, err := client.GetFundNAVHistory(context.Background(), " fund-a ")

	if err != nil {
		t.Fatal(err)
	}
	want := []domain.NavPoint{{Date: "2026-10-12", NAV: 12_000.5}, {Date: "2026-10-10", NAV: 11_000}}
	if history.Symbol != "FUND-A" || !reflect.DeepEqual(history.Points, want) || history.Source.Provider != "fmarket" {
		t.Fatalf("history = %+v", history)
	}
	if path := (*requests)[0].URL.Path; path != "/api/v2/funds/FUND-A/nav" {
		t.Fatalf("path = %s", path)
	}
	if _, err := client.GetFundNAVHistory(context.Background(), " "); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenTenoredBankRatesWhenReadingThemThenTenorMonthsInstantAccessAndChannelAreParsed(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer([]any{
			map[string]any{"bank": "Alpha Bank", "tenor": "no-term", "rate": 0.2, "currency": "VND", "tenorMonths": 0, "channel": "counter", "instantAccess": true},
			map[string]any{"bank": "Alpha Bank", "tenor": "1m", "rate": 3.9, "currency": "VND", "tenorMonths": 1, "channel": "counter", "instantAccess": false},
			map[string]any{"bank": "Beta Bank", "tenor": "", "rate": 5.1, "currency": "VND", "updatedAt": "2026-10-10"},
		}, map[string]any{"provider": "cafef"})
	})
	rates, err := client.GetBankInterestRate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if (*requests)[0].URL.Path != "/api/v2/bank-rates" {
		t.Fatalf("request = %s", (*requests)[0].URL)
	}
	zero, one := 0, 1
	want := []domain.BankRate{
		{Bank: "Alpha Bank", Tenor: "no-term", TenorMonths: &zero, InstantAccess: true, Channel: "counter", Rate: 0.2, Currency: "VND"},
		{Bank: "Alpha Bank", Tenor: "1m", TenorMonths: &one, Channel: "counter", Rate: 3.9, Currency: "VND"},
		{Bank: "Beta Bank", Rate: 5.1, Currency: "VND", UpdatedAt: "2026-10-10"},
	}
	if !reflect.DeepEqual(rates.Rates, want) {
		t.Fatalf("rates = %#v", rates.Rates)
	}
	if rates.Rates[2].TenorMonths != nil {
		t.Fatal("an untenored row must keep TenorMonths absent, not 0 (no-term)")
	}
}

func TestGivenFlexibleSavingsWhenReadingThemThenTermsFailuresAndSourceAreKept(t *testing.T) {
	client, requests := fakeServer(t, func(*http.Request) (int, any) {
		return 200, answer(map[string]any{
			"products": []any{map[string]any{
				"id": "wallet:earn", "provider": "Wallet", "product": "Earn", "rate": 4.5, "backing": "deposit",
				"earningStarts": "next_working_day", "withdrawalMaxWorkingHours": 16, "currency": "VND", "url": "https://wallet.example/earn",
				"asOf": "2026-10-22T02:00:00Z", "stale": true,
			}},
			"failed": []any{"get flexible savings from other failed: other: down"},
		}, map[string]any{"provider": "wallet"})
	})
	savings, err := client.GetFlexibleSavings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if (*requests)[0].URL.Path != "/api/v2/flexible-savings" {
		t.Fatalf("request = %s", (*requests)[0].URL)
	}
	hours := 16
	want := domain.FlexibleSaving{
		ID: "wallet:earn", Provider: "Wallet", Product: "Earn", Rate: 4.5, Backing: domain.SavingsBackingDeposit,
		EarningStarts: domain.EarningStartsNextWorkingDay, WithdrawalMaxWorkingHours: &hours, Currency: "VND", URL: "https://wallet.example/earn",
		AsOf: "2026-10-22T02:00:00Z", Stale: true,
	}
	if len(savings.Products) != 1 || !reflect.DeepEqual(savings.Products[0], want) || len(savings.Failed) != 1 || savings.Source == nil || savings.Source.Provider != "wallet" {
		t.Fatalf("savings = %#v", savings)
	}
}
