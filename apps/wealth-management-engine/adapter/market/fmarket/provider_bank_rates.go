package fmarket

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"net/http"
)

func (p *Provider) GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error) {
	response, err := p.do(ctx, http.MethodGet, "/res/bank-interest-rate", nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var envelope fmarketEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return nil, err
	}

	var payload struct {
		BankList []struct {
			Name  string `json:"name"`
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"bankList"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, err
	}

	rates := make([]domain.BankRate, 0, len(payload.BankList))
	for _, item := range payload.BankList {
		rate, _ := parseStringFloat(item.Value)
		rates = append(rates, domain.BankRate{
			Bank:     fallbackString(item.Name, item.Key),
			Tenor:    "",
			Rate:     rate,
			Currency: "VND",
		})
	}

	return rates, nil
}
