package marketdata

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var _ port.FundNAVSource = (*Client)(nil)

// GetFundNAVHistory reads a fund's full NAV history by its code from the
// server (fmarket getNavHistory). Rows are fmarket's own: `nav` and
// `navDate` (yyyy-MM-dd, or epoch milliseconds); unusable rows are dropped.
func (c *Client) GetFundNAVHistory(ctx context.Context, code string) (domain.FundNAVHistory, error) {
	symbol := domain.FundKey(code)
	if symbol == "" {
		return domain.FundNAVHistory{}, fmt.Errorf("%w: fund code is required", domain.ErrValidation)
	}
	rows, meta, err := get[[]map[string]any](ctx, c, "/api/v2/funds/"+url.PathEscape(symbol)+"/nav", nil)
	if err != nil {
		return domain.FundNAVHistory{}, err
	}
	history := domain.FundNAVHistory{Symbol: symbol, Points: []domain.NavPoint{}, Source: meta.source()}
	for _, row := range rows {
		nav, ok := number(row["nav"])
		date := navDate(row["navDate"])
		if ok && nav > 0 && date != "" {
			history.Points = append(history.Points, domain.NavPoint{Date: date, NAV: nav})
		}
	}
	return history, nil
}

func number(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

// navDate is a NAV day in Vietnam time.
func navDate(value any) string {
	if ms, ok := value.(float64); ok && ms > 0 {
		return time.UnixMilli(int64(ms)).In(domain.AppLocation).Format(time.DateOnly)
	}
	text, _ := value.(string)
	text = strings.TrimSpace(text)
	if len(text) >= 10 {
		if _, err := time.Parse(time.DateOnly, text[:10]); err == nil {
			return text[:10]
		}
	}
	return ""
}
