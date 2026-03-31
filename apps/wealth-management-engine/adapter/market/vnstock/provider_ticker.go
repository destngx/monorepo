package vnstock

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (p *Provider) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	if tickerType != domain.TickerTypeEquity {
		return domain.Ticker{}, fmt.Errorf("vnstock does not support ticker type %s", tickerType)
	}

	response, err := p.do(ctx, "/api/v1/stocks/quote?symbol="+url.QueryEscape(symbol))
	if err != nil {
		return domain.Ticker{}, err
	}
	defer response.Body.Close()

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Symbol string      `json:"symbol"`
			Price  interface{} `json:"price"`
			Time   string      `json:"time"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return domain.Ticker{}, err
	}
	if !payload.Success {
		return domain.Ticker{}, fmt.Errorf("vnstock quote request did not succeed")
	}

	return domain.Ticker{
		Symbol:    fallbackSymbol(payload.Data.Symbol, symbol),
		Type:      tickerType,
		Provider:  p.Name(),
		Currency:  "VND",
		Price:     parseFloat(payload.Data.Price),
		Timestamp: payload.Data.Time,
	}, nil
}
