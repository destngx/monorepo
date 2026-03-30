# Technical Specification: Capability-Based Market Provider Architecture

**Author**: System Architect / Platform Lead  
**Intended Audience**: Backend Engineering (Go Engine)  
**Status**: Approved for Implementation  
**Keywords**: Hexagonal Architecture, Dependency Injection, Multi-Provider Routing, Capability-Based Design

---

## 1. Strategic Context

### 1.1 Problem Statement

The wealth management platform integrates multiple market data sources:

- **Fmarket.vn**: Specializes in Vietnamese funds, IFC tickers, banking rates, gold prices, USD/VND rates
- **Vnstock**: Specializes in Vietnamese stock market tickers (equity), alternative FX sources
- **Future providers**: Binance (crypto), potentially other regional exchanges

Each provider offers **overlapping but different capabilities**. A naive approach treats each provider as a separate service with isolated endpoints. This creates:

- Code duplication across handlers (fetch from Provider A, fetch from Provider B, merge results)
- Tight coupling to specific providers (hard to add Provider 3)
- No unified routing logic
- Duplicate caching per provider

### 1.2 Solution: Capability-Based Routing

Instead of treating providers as isolated silos, we model the **capability** as the fundamental unit:

- **Capability**: A domain-specific ability (e.g., "Get Ticker", "Get Exchange Rate", "Get Gold Price")
- **Capability Type**: A parameter that modifies the capability (e.g., "GetTicker" with type "equity" vs. type "ifc")
- **Provider Priority Chain**: For each capability+type, a list of providers in priority order

**Benefits**:

- ✅ Single stable port interface regardless of provider count
- ✅ Config-driven routing (no code changes for new providers or priority swaps)
- ✅ Extensible to 3+ providers without refactoring
- ✅ Unified caching at capability level
- ✅ Clean fallback chains built into service layer
- ✅ Hexagonal architecture (business logic isolated from provider implementations)

---

## 2. Architecture Overview

### 2.1 Layering

```
┌─────────────────────────────────────────────────────┐
│ HTTP Handler Layer (adapter/fiber/)                 │
│ ├─ Exposes generic capability endpoints              │
│ └─ Maps HTTP params to capability requests           │
├─────────────────────────────────────────────────────┤
│ Service Layer (service/)                            │
│ ├─ Orchestrates capability routing                   │
│ ├─ Manages fallback chains                           │
│ ├─ Handles unified caching                           │
│ └─ Logs/monitors provider selection                  │
├─────────────────────────────────────────────────────┤
│ Port Layer (port/)                                  │
│ ├─ Defines MarketProvider interface (stable)         │
│ └─ Capability methods (generic, provider-agnostic)   │
├─────────────────────────────────────────────────────┤
│ Provider Layer (adapter/market/*)                   │
│ ├─ Fmarket provider                                  │
│ ├─ Vnstock provider                                  │
│ └─ Future providers (Binance, etc.)                  │
├─────────────────────────────────────────────────────┤
│ External Layer                                       │
│ ├─ api.fmarket.vn (HTTP)                            │
│ ├─ Vnstock API (HTTP)                               │
│ └─ Upstash Redis (cache)                            │
└─────────────────────────────────────────────────────┘
```

### 2.2 Port Interface (The Stable Contract)

The `port.MarketProvider` interface is the contract all providers must implement. It is **stable** and doesn't change when adding new providers.

```go
// port/market_provider.go
type MarketProvider interface {
    // Provider identification
    Name() string

    // Health check
    Health(ctx context.Context) error

    // Core capabilities (domain-specific, not action-based)
    GetTicker(ctx context.Context, symbol string, tickerType TickerType) (*Ticker, error)
    GetExchangeRate(ctx context.Context, from, to Currency) (*ExchangeRate, error)
    GetPriceSeries(ctx context.Context, symbol string, seriesType SeriesType) (interface{}, error)
    GetBankInterestRate(ctx context.Context) ([]*BankRate, error)
}

// Domain types (stable across all providers)
type TickerType string
const (
    TickerTypeEquity TickerType = "equity"
    TickerTypeIFC    TickerType = "ifc"
)

type Currency string
const (
    CurrencyUSD Currency = "USD"
    CurrencyVND Currency = "VND"
    CurrencyEUR Currency = "EUR"
)

type SeriesType string
const (
    SeriesTypeEquityOHLC SeriesType = "equity"        // OHLC data
    SeriesTypeGoldUSD    SeriesType = "gold_usd"      // Gold world price (USD)
    SeriesTypeForex      SeriesType = "forex"         // FX rate history
)
```

**Key insight**: Gold **SJC bid/ask is a ticker type** (`GetTicker(symbol, "gold")`), not a time-series. It's returned as a normalized `Ticker` struct just like equity or IFC tickers. World gold price in USD is a **time-series** (`GetPriceSeries(symbol, "gold_usd")`) because it's historical price data.

### 2.3 Routing Configuration (Injected)

The routing config defines **which provider handles which capability+type**. This is **injected** at startup, enabling different configurations per environment or user.

```yaml
# config/routing.yml
capabilities:
  getTicker:
    equity:
      - fmarket # Primary: Fmarket has equity tickers
      - vnstock # Fallback: Vnstock also supports
    ifc:
      - fmarket # Primary: Fmarket specializes in IFC
      - vnstock # Fallback: Vnstock can attempt

  getExchangeRate:
    USD_VND:
      - fmarket # Primary: Fmarket faster for VN rates
      - vnstock # Fallback: Vnstock alternative
    EUR_USD:
      - vnstock # Vnstock for other pairs (Fmarket doesn't support)

  getFundNavHistory:
    - fmarket # Only Fmarket supports

  getGoldPrice:
    - fmarket # Only Fmarket supports

  getBankInterestRate:
    - fmarket # Only Fmarket supports
```

### 2.4 Service Layer (Routing Logic)

The service layer **orchestrates** capability dispatch using the injected routing config.

```go
// service/market_provider_service.go
type MarketProviderService struct {
    providers      map[string]port.MarketProvider
    routingConfig  *RoutingConfig
    cache          port.CacheClient
    logger         *Logger
}

func (s *MarketProviderService) GetTicker(ctx context.Context, symbol string, tickerType TickerType) (*Ticker, error) {
    // 1. Look up provider chain from routing config
    chain := s.routingConfig.ChainFor("getTicker", string(tickerType))
    if len(chain) == 0 {
        return nil, ErrUnsupportedCapability
    }

    // 2. Check cache at capability level
    cacheKey := s.cacheKey("getTicker", tickerType, symbol)
    if cached, hit := s.cache.Get(ctx, cacheKey); hit {
        s.logger.Info("cache hit", cacheKey)
        return cached.(*Ticker), nil
    }

    // 3. Try each provider in chain
    var lastErr error
    for _, providerName := range chain {
        provider := s.providers[providerName]

        ticker, err := provider.GetTicker(ctx, symbol, tickerType)
        if err == nil {
            // Cache successful response
            s.cache.Set(ctx, cacheKey, ticker, MARKET_DATA_TTL)
            s.logger.Info("cache miss -> upstream", providerName, cacheKey)
            return ticker, nil
        }

        // Log fallback attempt
        lastErr = err
        s.logger.Warn("provider failed, trying fallback", providerName, err)
    }

    return nil, fmt.Errorf("all providers failed: %w (tried: %v)", lastErr, chain)
}

func (s *MarketProviderService) cacheKey(capability, paramType, symbol string) string {
    return fmt.Sprintf("market:%s:%s:%s", capability, paramType, symbol)
}
```

---

## 3. Provider Implementation Pattern

Each provider (Fmarket, Vnstock, etc.) implements the same `port.MarketProvider` interface but with **different internal logic**.

### 3.1 Fmarket Provider Example

```go
// adapter/market/fmarket/provider.go
type FmarketProvider struct {
    httpClient *http.Client
    baseURL    string
    config     *FmarketConfig
}

func (p *FmarketProvider) Name() string {
    return "fmarket"
}

func (p *FmarketProvider) GetTicker(ctx context.Context, symbol string, tickerType TickerType) (*Ticker, error) {
    // Only Fmarket supports IFC tickers directly
    if tickerType != TickerTypeIFC {
        return nil, ErrUnsupported // Service will try fallback
    }

    // Map capability to Fmarket action
    resp, err := p.getProductDetails(ctx, symbol) // Fmarket API call
    if err != nil {
        return nil, fmt.Errorf("fmarket getProductDetails failed: %w", err)
    }

    // Transform Fmarket response to domain model
    return p.mapProductToTicker(resp), nil
}

func (p *FmarketProvider) GetExchangeRate(ctx context.Context, from, to Currency) (*ExchangeRate, error) {
    if from != CurrencyUSD || to != CurrencyVND {
        return nil, ErrUnsupported // Only USD/VND
    }

    resp, err := p.getUsdRateHistory(ctx) // Fmarket API call
    if err != nil {
        return nil, fmt.Errorf("fmarket getUsdRateHistory failed: %w", err)
    }

    return p.mapRateToExchangeRate(resp), nil
}

// Internal: Fmarket-specific API method (not part of port interface)
func (p *FmarketProvider) getProductDetails(ctx context.Context, symbol string) (*FmarketProduct, error) {
    // Implement Fmarket-specific request logic
    // Include institutional headers (User-Agent, Origin, Referer)
    // Handle timeouts, retries, etc.
}

// Helper: Map Fmarket response to domain model
func (p *FmarketProvider) mapProductToTicker(product *FmarketProduct) *Ticker {
    return &Ticker{
        Symbol:    product.Code,
        Type:      TickerTypeIFC,
        Price:     product.NAV,
        Currency:  CurrencyVND,
        Timestamp: product.UpdatedAt,
    }
}
```

### 3.2 Vnstock Provider Example

```go
// adapter/market/vnstock/provider.go
type VnstockProvider struct {
    httpClient *http.Client
    baseURL    string
}

func (p *VnstockProvider) Name() string {
    return "vnstock"
}

func (p *VnstockProvider) GetTicker(ctx context.Context, symbol string, tickerType TickerType) (*Ticker, error) {
    // Vnstock specializes in equity tickers
    if tickerType == TickerTypeIFC {
        return nil, ErrUnsupported // Service will try fallback
    }

    resp, err := p.getEquityTicker(ctx, symbol) // Vnstock API call
    if err != nil {
        return nil, fmt.Errorf("vnstock getEquityTicker failed: %w", err)
    }

    return p.mapTickerResponse(resp), nil
}

func (p *VnstockProvider) GetExchangeRate(ctx context.Context, from, to Currency) (*ExchangeRate, error) {
    // Vnstock can provide alternative FX rates (fallback)
    resp, err := p.getFxRate(ctx, from, to)
    if err != nil {
        return nil, fmt.Errorf("vnstock getFxRate failed: %w", err)
    }

    return p.mapFxResponse(resp), nil
}

func (p *VnstockProvider) GetFundNavHistory(ctx context.Context, symbol string) ([]*NavPoint, error) {
    return nil, ErrUnsupported // Vnstock doesn't support funds
}

func (p *VnstockProvider) GetGoldPrice(ctx context.Context) (*Price, error) {
    return nil, ErrUnsupported // Vnstock doesn't support gold
}

func (p *VnstockProvider) GetBankInterestRate(ctx context.Context) ([]*BankRate, error) {
    return nil, ErrUnsupported // Vnstock doesn't support bank rates
}
```

---

## 4. HTTP Handler Layer (API Endpoints)

The HTTP layer exposes **generic capability endpoints**, not action-based endpoints.

```go
// adapter/fiber/market_handler.go
type MarketHandler struct {
    service port.MarketProviderService
}

// GET /api/market/ticker?symbol=ACB&type=equity
func (h *MarketHandler) GetTicker(c *fiber.Ctx) error {
    symbol := c.Query("symbol")
    tickerType := c.Query("type", "equity")

    ticker, err := h.service.GetTicker(c.Context(), symbol, TickerType(tickerType))
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(ticker)
}

// GET /api/market/exchange-rate?from=USD&to=VND
func (h *MarketHandler) GetExchangeRate(c *fiber.Ctx) error {
    from := c.Query("from")
    to := c.Query("to")

    rate, err := h.service.GetExchangeRate(c.Context(), Currency(from), Currency(to))
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(rate)
}

// GET /api/market/fund-nav-history?symbol=FUND001
func (h *MarketHandler) GetFundNavHistory(c *fiber.Ctx) error {
    symbol := c.Query("symbol")

    history, err := h.service.GetFundNavHistory(c.Context(), symbol)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(history)
}

// GET /api/market/gold-price
func (h *MarketHandler) GetGoldPrice(c *fiber.Ctx) error {
    price, err := h.service.GetGoldPrice(c.Context())
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(price)
}

// GET /api/market/bank-rates
func (h *MarketHandler) GetBankInterestRate(c *fiber.Ctx) error {
    rates, err := h.service.GetBankInterestRate(c.Context())
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(rates)
}

// Health check (lists all providers)
// GET /api/external/market/health
func (h *MarketHandler) HealthMulti(c *fiber.Ctx) error {
    // Aggregate health from all providers
    status := map[string]interface{}{}
    for _, provider := range h.service.Providers() {
        err := provider.Health(c.Context())
        status[provider.Name()] = err == nil
    }
    return c.JSON(status)
}
```

---

## 5. Dependency Injection & Initialization

Providers and routing config are injected at startup, enabling testability and multi-environment configuration.

```go
// main.go
func init() {
    // 1. Load routing config from file/env
    routingConfig := LoadRoutingConfig("config/routing.yml")

    // 2. Instantiate providers
    fmarketProvider := adapter.NewFmarketProvider(
        httpClient,
        os.Getenv("FMARKET_BASE_URL"),
    )

    vnstockProvider := adapter.NewVnstockProvider(
        httpClient,
        os.Getenv("VNSTOCK_BASE_URL"),
    )

    // 3. Register providers in service
    providers := map[string]port.MarketProvider{
        "fmarket": fmarketProvider,
        "vnstock": vnstockProvider,
    }

    // 4. Instantiate service with DI
    marketService := service.NewMarketProviderService(
        providers,
        routingConfig,
        cacheClient,
        logger,
    )

    // 5. Register HTTP handler
    marketHandler := adapter.NewMarketHandler(marketService)
    router.Get("/api/market/ticker", marketHandler.GetTicker)
    router.Get("/api/market/exchange-rate", marketHandler.GetExchangeRate)
    // ... other endpoints
}
```

---

## 6. Extensibility Examples

### 6.1 Adding a Third Provider (Binance for Crypto)

1. **Implement the interface** (no code changes to port, service, or handler):

   ```go
   type BinanceProvider struct { ... }

   func (p *BinanceProvider) Name() string { return "binance" }
   func (p *BinanceProvider) GetTicker(ctx, symbol, type) (*Ticker, error) { ... }
   func (p *BinanceProvider) GetExchangeRate(ctx, from, to) (*ExchangeRate, error) { ... }
   // ... other methods
   ```

2. **Update routing config**:

   ```yaml
   capabilities:
     getTicker:
       crypto:
         - binance
       equity:
         - vnstock
         - fmarket
   ```

3. **Register in main.go**:

   ```go
   binanceProvider := adapter.NewBinanceProvider(...)
   providers["binance"] = binanceProvider
   ```

4. **Endpoints work immediately**:
   ```bash
   curl http://localhost:8080/api/market/ticker?symbol=BTCUSDT&type=crypto
   ```

### 6.2 Swapping Provider Priority (A/B Testing)

Change routing config without code changes:

```yaml
# Before: Fmarket primary for USD/VND
getExchangeRate:
  USD_VND:
    - fmarket
    - vnstock

# After: Vnstock primary (A/B test)
getExchangeRate:
  USD_VND:
    - vnstock
    - fmarket
```

Restart service (or hot-reload if implemented): priority is swapped instantly.

---

## 7. Caching Strategy

Caching is **unified at the capability level**, not per-provider. This prevents redundant upstream calls even if multiple providers support the same capability.

```go
// Single cache key per capability
const (
    MARKET_DATA_TTL = 300 * time.Second // 5 minutes
)

// Cache key generation
cacheKey := fmt.Sprintf("market:%s:%s:%s",
    "getTicker",           // capability
    "equity",              // type
    "ACB",                 // symbol
)

// First call: cache miss → Vnstock upstream
// Second call (within TTL): cache hit → no upstream call
// Third call (after TTL): cache miss → Vnstock upstream again
```

---

## 8. Error Handling & Fallback

When a provider fails, the service automatically tries the next provider in the chain.

```go
// Example: GetTicker("FUND001", "ifc") with chain [fmarket, vnstock]
// 1. Try Fmarket
//    - Success → Return, cache, done
//    - Error (timeout, network, etc.) → Try fallback
// 2. Try Vnstock
//    - Success → Return, cache, done
//    - Error → Return combined error
//
// Error message: "all providers failed: vnstock error (tried: [fmarket, vnstock])"
```

### 8.1 Graceful Degradation

Some capabilities may only have one provider. If that fails:

```go
// GetGoldPrice() with chain [fmarket]
// 1. Try Fmarket
//    - Success → Return
//    - Error → Return error (no fallback available)
```

---

## 9. Monitoring & Logging

The service logs provider selection, fallback attempts, and cache behavior for observability.

```go
// Per-request log example
2026-03-30T15:23:45.123Z INFO  service=market_provider_service
  capability=getTicker
  type=equity
  symbol=ACB
  primary=vnstock
  cache_hit=false
  provider_attempt=1
  status=success
  latency_ms=542
  cache_key=market:getTicker:equity:ACB

2026-03-30T15:23:50.456Z WARN  service=market_provider_service
  capability=getTicker
  type=ifc
  symbol=FUND001
  primary=fmarket
  cache_hit=false
  provider_attempt=1
  status=failed
  error=timeout
  fallback=true
  next_provider=vnstock
  latency_ms=10234
```

---

## 10. Testing Strategy

Tests validate routing logic, fallback behavior, caching, and provider implementations independently.

```go
// Test: Routing selects correct provider
func TestCapabilityRouting(t *testing.T) {
    cases := []struct {
        capability string
        paramType  string
        expected   string
    }{
        {"getTicker", "equity", "vnstock"},
        {"getTicker", "ifc", "fmarket"},
        {"getExchangeRate", "USD_VND", "fmarket"},
    }

    for _, tc := range cases {
        chain := service.routingConfig.ChainFor(tc.capability, tc.paramType)
        assert.Equal(t, tc.expected, chain[0])
    }
}

// Test: Fallback behavior
func TestFallbackChain(t *testing.T) {
    // Mock primary provider to fail
    mockFmarket.On("GetTicker").Return(nil, ErrTimeout)

    // Service should try Vnstock
    mockVnstock.On("GetTicker").Return(&Ticker{Symbol: "ACB"}, nil)

    ticker, err := service.GetTicker(ctx, "ACB", TickerTypeEquity)
    assert.NoError(t, err)
    assert.Equal(t, "ACB", ticker.Symbol)
    mockFmarket.AssertCalled(t, "GetTicker")
    mockVnstock.AssertCalled(t, "GetTicker")
}

// Test: Cache behavior
func TestCacheKey(t *testing.T) {
    key1 := service.cacheKey("getTicker", "equity", "ACB")
    key2 := service.cacheKey("getTicker", "ifc", "FUND001")

    assert.Equal(t, "market:getTicker:equity:ACB", key1)
    assert.Equal(t, "market:getTicker:ifc:FUND001", key2)
    assert.NotEqual(t, key1, key2) // Different types → different cache keys
}
```

---

## Summary

This architecture provides:

- ✅ **Stability**: Port interface is stable across all provider counts
- ✅ **Extensibility**: Add new providers without code changes
- ✅ **Flexibility**: Config-driven routing for A/B testing and priority swaps
- ✅ **Simplicity**: Service layer is single-responsibility (routing + fallback)
- ✅ **Maintainability**: Each provider is independent; changes don't affect others
- ✅ **Observability**: Logging shows provider selection and fallback attempts
- ✅ **Testability**: DI enables mock providers for deterministic tests

---

**Last Updated**: March 30, 2026  
**Status**: Approved for Implementation
