package vnstock

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
)

func (p *Provider) GetBankInterestRate(_ context.Context) ([]domain.BankRate, error) {
	return nil, errors.New("vnstock does not support bank interest rates")
}
