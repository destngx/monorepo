package mock

import (
	"context"
	"strings"
)

type BalanceTool struct{}

func NewBalanceTool() *BalanceTool {
	return &BalanceTool{}
}

func (t *BalanceTool) GetBalance(_ context.Context, account string) (float64, error) {
	switch strings.ToLower(account) {
	case "cash":
		return 373000, nil
	case "zalopay":
		return 68783606, nil
	default:
		return 1000000, nil
	}
}
