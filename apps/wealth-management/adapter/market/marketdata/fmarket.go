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

var _ port.FmarketService = (*Client)(nil)

// RunAction serves the fmarket actions of /api/fmarket/action and MCP from
// the server's fund and gold endpoints. The UI parses fmarket's own payloads,
// so each answer is wrapped as `{data}` like fmarket's envelope (gold
// history adds `extra`); getBankInterestRates stays typed. forceFresh
// bypasses the server cache.
func (c *Client) RunAction(ctx context.Context, request domain.FmarketRequest, forceFresh bool) (any, error) {
	ctx = domain.WithBypassCache(ctx, forceFresh || domain.ShouldBypassCache(ctx))
	params := request.Params
	if params == nil {
		params = map[string]any{}
	}

	switch request.Action {
	case domain.FmarketActionProductsFilterNav:
		return c.raw(ctx, "/api/v2/funds", fundsQuery(params))
	case domain.FmarketActionIssuers:
		return c.raw(ctx, "/api/v2/funds/issuers", nil)
	case domain.FmarketActionBankInterestRates:
		return c.GetBankInterestRate(ctx)
	case domain.FmarketActionProductDetails:
		code := strings.TrimSpace(paramString(params, "code"))
		if code == "" {
			return nil, fmt.Errorf("%w: code is required", domain.ErrValidation)
		}
		return c.raw(ctx, "/api/v2/funds/"+url.PathEscape(code), nil)
	case domain.FmarketActionNavHistory:
		productID := paramInt(params, "productId")
		if productID <= 0 {
			return nil, fmt.Errorf("%w: productId is required", domain.ErrValidation)
		}
		return c.raw(ctx, "/api/v2/funds/"+strconv.Itoa(productID)+"/nav", nil)
	case domain.FmarketActionGoldPriceHistory:
		history, _, err := get[struct {
			Rows    any `json:"rows"`
			Summary any `json:"summary"`
		}](ctx, c, "/api/v2/gold/history", dateRange(params))
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": history.Rows, "extra": history.Summary}, nil
	case domain.FmarketActionUsdRateHistory:
		return c.raw(ctx, "/api/v2/fx/history", dateRange(params))
	case domain.FmarketActionGoldProducts:
		return c.raw(ctx, "/api/v2/gold/products", nil)
	default:
		return nil, fmt.Errorf("%w: invalid fmarket action: %s", domain.ErrValidation, request.Action)
	}
}

// raw returns the server's fmarket payload in fmarket's `{data}` envelope.
func (c *Client) raw(ctx context.Context, path string, query url.Values) (any, error) {
	data, _, err := get[any](ctx, c, path, query)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": data}, nil
}

// fundsQuery maps getProductsFilterNav params: page, pageSize, assetTypes
// (default STOCK) and isMMFFund, which ignores the asset types.
func fundsQuery(params map[string]any) url.Values {
	query := url.Values{}
	if page := paramInt(params, "page"); page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	if pageSize := paramInt(params, "pageSize"); pageSize > 0 {
		query.Set("page_size", strconv.Itoa(pageSize))
	}
	for _, assetType := range paramStrings(params, "assetTypes") {
		query.Add("asset_type", assetType)
	}
	if paramBool(params, "isMMFFund") {
		query.Set("mmf", "true")
	}
	return query
}

// dateRange maps fromDate and toDate (yyyyMMdd, yyyy-MM-dd, dd/MM/yyyy or
// MM/dd/yyyy); the server defaults a missing end to today and a missing
// start to 30 days before the end, in Vietnam time.
func dateRange(params map[string]any) url.Values {
	query := url.Values{}
	if from := legacyDate(paramString(params, "fromDate")); from != "" {
		query.Set("from", from)
	}
	if to := legacyDate(paramString(params, "toDate")); to != "" {
		query.Set("to", to)
	}
	return query
}

// legacyDate normalises a date to yyyyMMdd; anything unreadable is passed
// through for the server to reject.
func legacyDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || (len(trimmed) == 8 && !strings.ContainsAny(trimmed, "-/")) {
		return trimmed
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "01/02/2006"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed.Format("20060102")
		}
	}
	return trimmed
}

func paramString(params map[string]any, key string) string {
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func paramInt(params map[string]any, key string) int {
	switch typed := params[key].(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func paramBool(params map[string]any, key string) bool {
	value, ok := params[key].(bool)
	return ok && value
}

func paramStrings(params map[string]any, key string) []string {
	var raw []any
	switch typed := params[key].(type) {
	case []any:
		raw = typed
	case []string:
		for _, item := range typed {
			raw = append(raw, item)
		}
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
			values = append(values, text)
		}
	}
	return values
}
