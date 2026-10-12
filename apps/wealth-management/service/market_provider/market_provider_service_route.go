package market_provider

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

var errCapabilityMissing = errors.New("capability not implemented")

// routeResult is a successful chain answer with the providers that failed
// before it, used for data-source indicators (WM-INV-15).
type routeResult[T any] struct {
	value    T
	provider string
	failures []string
}

// route tries each provider of chain in order until one answers. call gets
// the provider and type-asserts the capability it needs, returning
// errCapabilityMissing when the provider does not have it. Providers without
// the capability or coverage (port.ErrUnsupported) are skipped: they appear
// in the final error but not in the fallback failures.
func route[T any](
	ctx context.Context,
	s *marketProviderService,
	operation string,
	chain []string,
	call func(port.MarketProvider) (T, error),
) (routeResult[T], error) {
	if len(chain) == 0 {
		var zero routeResult[T]
		return zero, fmt.Errorf("no provider chain configured for %s", operation)
	}

	failures := make([]string, 0, len(chain))
	skipped := []string{}
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		value, err := call(provider)
		if err == nil {
			if len(failures) > 0 {
				s.log.LogApplicationEvent(ctx, "market provider fallback used",
					slog.String("operation", operation),
					slog.String("provider", provider.Name()),
					slog.String("failures", strings.Join(failures, "; ")),
				)
			}
			return routeResult[T]{value: value, provider: provider.Name(), failures: failures}, nil
		}
		reason := fmt.Sprintf("%s: %v", provider.Name(), err)
		if errors.Is(err, errCapabilityMissing) || errors.Is(err, port.ErrUnsupported) {
			skipped = append(skipped, reason)
			continue
		}
		failures = append(failures, reason)
	}

	var zero routeResult[T]
	if len(failures) == 0 {
		return zero, fmt.Errorf("%w: no provider covers %s (%s)", domain.ErrValidation, operation, strings.Join(skipped, "; "))
	}
	return zero, fmt.Errorf("%s failed: %s", operation, strings.Join(append(failures, skipped...), "; "))
}

// cachedRoute serves key from the cache unless the request bypasses it, and
// otherwise routes and caches the answer for the routing TTL.
func cachedRoute[T any](
	ctx context.Context,
	s *marketProviderService,
	key string,
	operation string,
	chain []string,
	call func(port.MarketProvider) (T, error),
) (T, error) {
	if cached, ok := getCached[T](ctx, s, key); ok {
		return cached, nil
	}
	result, err := route(ctx, s, operation, chain, call)
	if err != nil {
		return result.value, err
	}
	s.setCache(ctx, key, result.value)
	return result.value, nil
}

func getCached[T any](ctx context.Context, s *marketProviderService, key string) (T, bool) {
	var value T
	if s.cache == nil || domain.ShouldBypassCache(ctx) {
		return value, false
	}
	raw, found, err := s.cache.Get(ctx, key)
	if err != nil {
		s.log.LogApplicationEvent(ctx, "market cache read failed", slog.String("key", key), slog.String("error", err.Error()))
		return value, false
	}
	if !found {
		return value, false
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		s.log.LogApplicationEvent(ctx, "market cache entry unreadable", slog.String("key", key), slog.String("error", err.Error()))
		return value, false
	}
	return value, true
}

func (s *marketProviderService) setCache(ctx context.Context, key string, payload any) {
	if s.cache == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err == nil {
		err = s.cache.Set(ctx, key, string(data), s.routing.CacheTTLSeconds)
	}
	if err != nil {
		s.log.LogApplicationEvent(ctx, "market cache write failed", slog.String("key", key), slog.String("error", err.Error()))
	}
}
