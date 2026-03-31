package vnstock

import (
	"apps/wealth-management-engine/domain"
	"context"
	"errors"
)

func (p *Provider) GetBankInterestRate(_ context.Context) ([]domain.BankRate, error) {
	return nil, errors.New("vnstock does not support bank interest rates")
}
