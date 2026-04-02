package fmarket

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	provider *Provider
	cache    port.CacheClient
}

func NewService(provider *Provider, cache port.CacheClient) *Service {
	return &Service{provider: provider, cache: cache}
}

func (s *Service) RunAction(ctx context.Context, request domain.FmarketRequest, forceFresh bool) (any, error) {
	if s.provider == nil {
		return nil, fmt.Errorf("fmarket provider unavailable")
	}
	action := request.Action
	params := request.Params
	if params == nil {
		params = map[string]any{}
	}
	cacheKey := fmt.Sprintf("fmarket:%s:%s", action, stableJSON(params))
	if !forceFresh {
		if cached, ok, err := s.getCached(cacheKey); err == nil && ok {
			return cached, nil
		}
	}
	result, err := s.dispatch(ctx, action, params)
	if err != nil {
		return nil, err
	}
	_ = s.setCache(cacheKey, result)
	return result, nil
}

func (s *Service) dispatch(ctx context.Context, action domain.FmarketAction, params map[string]any) (any, error) {
	switch action {
	case domain.FmarketActionProductsFilterNav:
		return s.getProductsFilterNav(ctx, params)
	case domain.FmarketActionIssuers:
		return s.postJSON(ctx, "/res/issuers", map[string]any{})
	case domain.FmarketActionBankInterestRates:
		return s.provider.GetBankInterestRate(ctx)
	case domain.FmarketActionProductDetails:
		code := argumentString(params, "code")
		if strings.TrimSpace(code) == "" {
			return nil, fmt.Errorf("code is required")
		}
		return s.getJSON(ctx, "/home/product/"+code)
	case domain.FmarketActionNavHistory:
		productID := argumentInt(params, "productId")
		if productID == 0 {
			return nil, fmt.Errorf("productId is required")
		}
		return s.postJSON(ctx, "/res/product/get-nav-history", map[string]any{
			"isAllData": 1,
			"productId": productID,
			"navPeriod": fallbackString(argumentString(params, "navPeriod"), "navToBeginning"),
		})
	case domain.FmarketActionGoldPriceHistory:
		return s.postDateRange(ctx, "/res/get-price-gold-history", params)
	case domain.FmarketActionUsdRateHistory:
		return s.postDateRange(ctx, "/res/get-usd-rate-history", params)
	case domain.FmarketActionGoldProducts:
		return s.postJSON(ctx, "/res/products/filter", map[string]any{
			"types":             []string{"GOLD"},
			"issuerIds":         []string{},
			"page":              1,
			"pageSize":          100,
			"fundAssetTypes":    []string{},
			"bondRemainPeriods": []string{},
			"searchField":       "",
		})
	default:
		return nil, fmt.Errorf("invalid fmarket action: %s", action)
	}
}

func (s *Service) getProductsFilterNav(ctx context.Context, params map[string]any) (any, error) {
	page := argumentInt(params, "page")
	if page == 0 {
		page = 1
	}
	pageSize := argumentInt(params, "pageSize")
	if pageSize == 0 {
		pageSize = 10
	}
	assetTypes := argumentStringSlice(params, "assetTypes")
	if len(assetTypes) == 0 {
		assetTypes = []string{"STOCK"}
	}
	isMMFFund := argumentBool(params, "isMMFFund")
	fundAssetTypes := []string{}
	if !isMMFFund {
		fundAssetTypes = assetTypes
	}
	return s.postJSON(ctx, "/res/products/filter", map[string]any{
		"types":          []string{"NEW_FUND", "TRADING_FUND"},
		"sortOrder":      "DESC",
		"sortField":      "navTo12Months",
		"isIpo":          false,
		"isMMFFund":      isMMFFund,
		"fundAssetTypes": fundAssetTypes,
		"page":           page,
		"pageSize":       pageSize,
	})
}

func (s *Service) postDateRange(ctx context.Context, path string, params map[string]any) (any, error) {
	fromDate := normalizeLegacyDate(argumentString(params, "fromDate"))
	toDate := normalizeLegacyDate(argumentString(params, "toDate"))
	if fromDate == "" || toDate == "" {
		from, to := lastNDaysLegacy(30)
		if fromDate == "" {
			fromDate = from
		}
		if toDate == "" {
			toDate = to
		}
	}
	return s.postJSON(ctx, path, map[string]any{
		"fromDate":  fromDate,
		"toDate":    toDate,
		"isAllData": false,
	})
}

func (s *Service) getJSON(ctx context.Context, path string) (any, error) {
	response, err := s.provider.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Service) postJSON(ctx context.Context, path string, body map[string]any) (any, error) {
	response, err := s.provider.do(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var payload any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func stableJSON(value any) string {
	normalized := normalizeStableValue(value)
	raw, _ := json.Marshal(normalized)
	return string(raw)
}

func normalizeStableValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := slices.Sorted(maps.Keys(typed))
		normalized := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			normalized = append(normalized, map[string]any{"key": key, "value": normalizeStableValue(typed[key])})
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeStableValue(item))
		}
		return normalized
	default:
		return typed
	}
}

func (s *Service) getCached(key string) (any, bool, error) {
	if s.cache == nil {
		return nil, false, nil
	}
	value, found, err := s.cache.Get(context.Background(), key)
	if err != nil || !found {
		return nil, false, err
	}
	var payload any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return nil, false, err
	}
	return payload, true, nil
}

func (s *Service) setCache(key string, payload any) error {
	if s.cache == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.cache.Set(context.Background(), key, string(raw), 60)
}

func argumentString(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func argumentInt(values map[string]any, key string) int {
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
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

func argumentBool(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	result, ok := value.(bool)
	return ok && result
}

func argumentStringSlice(values map[string]any, key string) []string {
	value, ok := values[key]
	if !ok {
		return nil
	}
	rawValues, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(rawValues))
	for _, item := range rawValues {
		result = append(result, fmt.Sprintf("%v", item))
	}
	return result
}

func normalizeLegacyDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) == 8 && !strings.Contains(trimmed, "-") {
		return trimmed
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "01/02/2006"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.Format("20060102")
		}
	}
	return trimmed
}

func lastNDaysLegacy(days int) (string, string) {
	now := time.Now()
	from := now.AddDate(0, 0, -days)
	return from.Format("20060102"), now.Format("20060102")
}
