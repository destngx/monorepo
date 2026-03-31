package fmarket

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"net/http"
)

func (p *Provider) Health(ctx context.Context) (domain.MarketProviderHealth, error) {
	response, err := p.do(ctx, http.MethodGet, "/res/bank-interest-rate", nil)
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return domain.MarketProviderHealth{}, err
	}
	status := "ok"
	if statusCode, ok := parseEnvelopeStatusCode(envelope.Status); ok && statusCode >= http.StatusBadRequest {
		status = "degraded"
	}

	return domain.MarketProviderHealth{
		Provider: p.Name(),
		Status:   status,
	}, nil
}
