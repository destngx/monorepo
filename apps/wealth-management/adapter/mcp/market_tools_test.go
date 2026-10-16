package mcp

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func newMarketTestServer(t *testing.T) (*Server, *fakes.Market, *fakes.AI) {
	t.Helper()
	server := newTestServer(t, nil)
	market := &fakes.Market{}
	ai := &fakes.AI{}
	server.SetMarketService(market)
	server.SetMarketAnalysisService(market)
	server.SetPortfolioService(market)
	server.SetAIService(ai)
	return server, market, ai
}

func TestGivenNoAnalysisOrPortfolioServiceWhenListingToolsThenTheirToolsAreHidden(t *testing.T) {
	server := newTestServer(t, nil)
	server.SetMarketService(&fakes.Market{})
	names := map[string]bool{}
	for _, definition := range server.ListTools() {
		names[definition.Name] = true
	}
	if !names["GetMarketCandles"] || names["GetMarketPulse"] || names["GetMarketAnalysis"] || names["GetInvestmentValuation"] {
		t.Fatalf("tools = %v", names)
	}
}

func TestGivenCandleArgsWhenCallingGetMarketCandlesThenArgsAndBypassAreForwarded(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	if _, err := server.CallTool(context.Background(), "GetMarketCandles", json.RawMessage(`{"symbol":"FPT","timeframe":"4h","days":30,"force":true}`)); err != nil {
		t.Fatal(err)
	}
	call := market.Calls[0]
	if call.Symbol != "FPT" || call.Kind != "4h" || call.Days != 30 || !call.Bypass {
		t.Fatalf("call = %+v", call)
	}
}

func TestGivenNoTimeframeWhenCallingMarketPulseThenItDefaultsToDailyWithCache(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	if _, err := server.CallTool(context.Background(), "GetMarketPulse", nil); err != nil {
		t.Fatal(err)
	}
	if call := market.Calls[0]; call.Kind != "pulse:1d" || call.Bypass {
		t.Fatalf("call = %+v", call)
	}
}

func TestGivenAnalysisArgsWhenCallingGetMarketAnalysisThenSymbolTimeframeAndForceAreForwarded(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	if _, err := server.CallTool(context.Background(), "GetMarketAnalysis", json.RawMessage(`{"symbol":"VNINDEX","timeframe":"1w","force":true}`)); err != nil {
		t.Fatal(err)
	}
	if call := market.Calls[0]; call.Symbol != "VNINDEX" || call.Kind != "analysis:1w" || !call.Bypass {
		t.Fatalf("call = %+v", call)
	}
}

func TestGivenInvalidMarketArgsWhenCallingThenValidationErrors(t *testing.T) {
	server, _, _ := newMarketTestServer(t)
	cases := map[string]string{
		"GetMarketPulse":         `{"timeframe":"5m"}`,
		"GetMarketAnalysis":      `{"symbol":"FPT","bogus":1}`,
		"GetMarketCandles":       `{"symbol":"FPT","days":"ten"}`,
		"GetInvestmentValuation": `{"force":true,"extra":1}`,
		"GetNewsSentiment":       `{"topic":"VN30","bogus":true}`,
		"GetTickerInsight":       `{"symbol":"FPT","unknown":1}`,
		"RunInvestmentAnalysis":  `{"news":"not-an-array"}`,
	}
	for name, arguments := range cases {
		if _, err := server.CallTool(context.Background(), name, json.RawMessage(arguments)); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%s(%s) err = %v, want validation", name, arguments, err)
		}
	}
}

func TestGivenForceWhenCallingGetInvestmentValuationThenForceIsForwarded(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	if _, err := server.CallTool(context.Background(), "GetInvestmentValuation", json.RawMessage(`{"force":true}`)); err != nil {
		t.Fatal(err)
	}
	if call := market.Calls[0]; call.Kind != "valuation" || !call.ForceRead {
		t.Fatalf("call = %+v", call)
	}
}

func TestGivenWiderTickerTypesWhenCallingGetMarketTickerThenEquityAndCryptoAreAccepted(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	for _, tickerType := range []string{"equity", "crypto"} {
		if _, err := server.CallTool(context.Background(), "GetMarketTicker", json.RawMessage(`{"symbol":"X","type":"`+tickerType+`"}`)); err != nil {
			t.Fatal(err)
		}
	}
	if len(market.Calls) != 2 || market.Calls[1].Kind != "crypto" {
		t.Fatalf("calls = %+v", market.Calls)
	}
}

func TestGivenNewsAndTickerArgsWhenCallingAIMarketToolsThenRequestsAreForwarded(t *testing.T) {
	server, _, ai := newMarketTestServer(t)
	result, err := server.CallTool(context.Background(), "GetNewsSentiment", json.RawMessage(`{"topic":"VN30","force":true,"model":"m"}`))
	if err != nil || !strings.Contains(result.Content, `"topic":"VN30"`) {
		t.Fatalf("news: %v %s", err, result.Content)
	}
	if len(ai.News) != 1 || !ai.News[0].Force || ai.News[0].Model != "m" {
		t.Fatalf("news requests = %+v", ai.News)
	}
	if _, err := server.CallTool(context.Background(), "GetTickerInsight", json.RawMessage(`{"symbol":"FPT","timeframe":"1h"}`)); err != nil {
		t.Fatal(err)
	}
	if len(ai.Tickers) != 1 || ai.Tickers[0].Symbol != "FPT" || ai.Tickers[0].Timeframe != "1h" {
		t.Fatalf("ticker requests = %+v", ai.Tickers)
	}
}

func TestGivenAThinkTankStreamWhenCallingRunInvestmentAnalysisThenEventsAreCollected(t *testing.T) {
	server, _, _ := newMarketTestServer(t)
	result, err := server.CallTool(context.Background(), "RunInvestmentAnalysis", json.RawMessage(`{"model":"m","symbols":["FPT"]}`))
	if err != nil {
		t.Fatal(err)
	}
	var collected investmentAnalysisResult
	if err := json.Unmarshal([]byte(result.Content), &collected); err != nil {
		t.Fatal(err)
	}
	if len(collected.Phases) != 1 || len(collected.Commands) != 1 || collected.Commands[0].Target != "VN30" {
		t.Fatalf("collected = %+v", collected)
	}
	if collected.Recorded == nil || collected.Recorded.Duplicates != 1 {
		t.Fatalf("the recommendations record must be collected: %+v", collected.Recorded)
	}
}

// erroringThinkTank emits an error event instead of returning an error,
// as the streaming service does for late failures.
type erroringThinkTank struct{ fakes.AI }

func (f *erroringThinkTank) InvestmentAnalysis(_ context.Context, _ port.InvestmentAnalysisRequest, emit func(port.InvestmentEvent) error) error {
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: "synthesis", Status: "error"}); err != nil {
		return err
	}
	return emit(port.InvestmentEvent{Type: "error", Message: "synthesis model unavailable"})
}

func TestGivenAnErrorEventWhenCallingRunInvestmentAnalysisThenTheToolFails(t *testing.T) {
	server := newTestServer(t, nil)
	server.SetAIService(&erroringThinkTank{})
	_, err := server.CallTool(context.Background(), "RunInvestmentAnalysis", nil)
	if !errors.Is(err, errInvestmentAnalysis) || !strings.Contains(err.Error(), "synthesis model unavailable") {
		t.Fatalf("err = %v", err)
	}
	response := callTool(t, server, "RunInvestmentAnalysis", `{}`)
	if !response.Result.IsError {
		t.Fatalf("response = %+v", response)
	}
}

func TestGivenTheServiceFailsWhenCallingRunInvestmentAnalysisThenTheErrorIsReturned(t *testing.T) {
	server := newTestServer(t, nil)
	failure := errors.New("boom")
	server.SetAIService(&fakes.AI{Err: failure})
	if _, err := server.CallTool(context.Background(), "RunInvestmentAnalysis", nil); !errors.Is(err, failure) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenForceWhenCallingGetAssetAllocationThenTheAllocationIsReturned(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	result, err := server.CallTool(context.Background(), "GetAssetAllocation", json.RawMessage(`{"force":true}`))
	if err != nil || !strings.Contains(result.Content, `"driftThreshold":5`) {
		t.Fatalf("result %v err %v", result.Content, err)
	}
	if call := market.Calls[0]; call.Kind != "allocation" || !call.ForceRead {
		t.Fatalf("call = %+v", call)
	}
	if _, err := server.CallTool(context.Background(), "GetAssetAllocation", json.RawMessage(`{"extra":1}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown arguments must be rejected: %v", err)
	}
}

func TestGivenForceWhenCallingGetFundPerformanceThenThePerformanceIsReturned(t *testing.T) {
	server, market, _ := newMarketTestServer(t)
	result, err := server.CallTool(context.Background(), "GetFundPerformance", json.RawMessage(`{"force":true}`))
	if err != nil || !strings.Contains(result.Content, `"totals"`) {
		t.Fatalf("result %v err %v", result.Content, err)
	}
	if call := market.Calls[0]; call.Kind != "fund-performance" || !call.ForceRead {
		t.Fatalf("call = %+v", call)
	}
	if _, err := server.CallTool(context.Background(), "GetFundPerformance", json.RawMessage(`{"extra":1}`)); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("unknown arguments must be rejected: %v", err)
	}
}

func TestGivenForceWhenCallingTheExposureToolsThenTheServiceIsCalledFresh(t *testing.T) {
	for tool, want := range map[string]struct{ kind, content string }{
		"GetExposure": {"exposure", `"volatileShare"`},
		"GetHedges":   {"hedges", `"cashShare"`},
	} {
		server, market, _ := newMarketTestServer(t)
		result, err := server.CallTool(context.Background(), tool, json.RawMessage(`{"force":true}`))
		if err != nil || !strings.Contains(result.Content, want.content) {
			t.Fatalf("%s: result %v err %v", tool, result.Content, err)
		}
		if call := market.Calls[0]; call.Kind != want.kind || !call.ForceRead {
			t.Fatalf("%s: call = %+v", tool, call)
		}
		if _, err := server.CallTool(context.Background(), tool, json.RawMessage(`{"extra":1}`)); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%s: unknown arguments must be rejected: %v", tool, err)
		}
	}
}
