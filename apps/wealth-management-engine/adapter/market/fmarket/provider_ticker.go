package fmarket

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (p *Provider) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	switch tickerType {
	case domain.TickerTypeIFC:
		response, err := p.do(ctx, http.MethodGet, "/home/product/"+url.PathEscape(symbol), nil)
		if err != nil {
			return domain.Ticker{}, err
		}
		defer response.Body.Close()

		var envelope fmarketEnvelope
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return domain.Ticker{}, err
		}
		var product struct {
			Code string  `json:"code"`
			Nav  float64 `json:"nav"`
		}
		if err := json.Unmarshal(envelope.Data, &product); err != nil {
			return domain.Ticker{}, err
		}

		return domain.Ticker{
			Symbol:   fallbackString(product.Code, symbol),
			Type:     tickerType,
			Provider: p.Name(),
			Currency: "VND",
			Price:    product.Nav,
		}, nil
	case domain.TickerTypeGold:
		if strings.EqualFold(symbol, "XAU") {
			series, err := p.GetPriceSeries(ctx, symbol, domain.SeriesTypeGoldUSD)
			if err != nil {
				return domain.Ticker{}, err
			}
			payload, ok := series.Data.(map[string]any)
			if !ok {
				return domain.Ticker{}, fmt.Errorf("unexpected gold series payload")
			}
			price, _ := payload["price"].(float64)
			timestamp, _ := payload["timestamp"].(string)
			return domain.Ticker{
				Symbol:    "XAU",
				Type:      tickerType,
				Provider:  p.Name(),
				Currency:  "USD",
				Price:     price,
				Timestamp: timestamp,
			}, nil
		}
		series, err := p.GetPriceSeries(ctx, symbol, domain.SeriesTypeGoldUSD)
		if err != nil {
			return domain.Ticker{}, err
		}
		payload, ok := series.Data.(map[string]any)
		if !ok {
			return domain.Ticker{}, fmt.Errorf("unexpected gold series payload")
		}

		bid, _ := payload["bidSjc"].(float64)
		ask, _ := payload["askSjc"].(float64)
		timestamp, _ := payload["timestamp"].(string)
		return domain.Ticker{
			Symbol:    symbol,
			Type:      tickerType,
			Provider:  p.Name(),
			Currency:  "VND",
			Bid:       bid,
			Ask:       ask,
			Timestamp: timestamp,
		}, nil
	default:
		return domain.Ticker{}, fmt.Errorf("fmarket does not support ticker type %s", tickerType)
	}
}
