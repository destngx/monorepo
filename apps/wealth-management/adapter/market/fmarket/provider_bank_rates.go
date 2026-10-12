package fmarket

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// GetBankInterestRate reads fmarket's savings rate table. fmarket publishes
// one rate per bank without a tenor, so Tenor stays empty rather than
// guessed; UpdatedAt is the table's publication date.

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
		UpdateAt string `json:"updateAt"`
		BankList []struct {
			Name  string `json:"name"`
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"bankList"`
	}
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return nil, err
	}

	updatedAt := ""
	if published, err := time.Parse("02/01/2006", payload.UpdateAt); err == nil {
		updatedAt = published.Format(time.DateOnly)
	}
	rates := make([]domain.BankRate, 0, len(payload.BankList))
	for _, item := range payload.BankList {
		rate, ok := parseStringFloat(item.Value)
		if !ok || rate <= 0 {
			continue
		}
		rates = append(rates, domain.BankRate{
			Bank:      fallbackString(item.Name, item.Key),
			Rate:      rate,
			Currency:  "VND",
			UpdatedAt: updatedAt,
		})
	}

	return rates, nil
}
