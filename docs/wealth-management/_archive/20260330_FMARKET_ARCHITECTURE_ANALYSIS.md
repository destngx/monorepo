# Fmarket Architecture Analysis & Redesign

**Date**: March 30, 2026  
**Status**: Architecture Consultation  
**Audience**: Dev Lead, Architects

---

## Executive Summary

The current market provider architecture treats vnstock and fmarket as **separate proxy services** with isolated health checks. This is **incorrect**.

The proper model is:

- **Market Provider Domain** = A unified abstraction providing **capabilities** (exchange rates, fund data, ticker prices)
- **Data Source Priority** = Multiple providers (fmarket, vnstock) implementing the same capability interfaces, with priority-based routing
- **Fmarket Priority** = Fmarket is the preferred source for: exchange rates (USD/VND), fund performance data, banking rates
- **Vnstock Fallback** = Vnstock provides equity ticker prices, historical data when fmarket unavailable

**Current Gap**: The `port.MarketProvider` interface only defines `Health(ctx)` — it doesn't define **what data the provider can fetch**. This blocks the ability to route requests to the best source.

---

## Current Architecture Analysis

### File Structure (Today)

```
apps/wealth-management-engine/
├── domain/
│   ├── market_provider.go          ❌ Only defines Health response
│   ├── market_data_provider.go     ❌ Only defines BaseURL config
│   └── (NO exchange rate / fund data models)
├── port/
│   └── market_provider.go          ❌ Only Health(ctx) method
├── adapter/market/
│   └── vnstock/
│       ├── provider.go             ✅ Implements MarketProvider
│       └── provider_test.go
├── service/
│   └── market_provider_service.go  ❌ Only knows how to call Health()
└── adapter/fiber/
    └── market_provider_handler.go  ❌ Only exposes /api/external/market/providers/:provider/health
```

### Current Data Flow

```
Client → GET /api/external/vnstock/health
       → MarketProviderHandler.Health(provider="vnstock")
       → MarketProviderService.Health()
       → vnstock.Health()
       → Returns {provider: "vnstock", status: "OK"}
```

**Problem**: This flow can only tell you if the provider is alive. It cannot:

- Fetch exchange rates
- Fetch fund performance data
- Fetch banking rates
- Route based on which provider has the data

### Fmarket API Actions (Current Legacy Implementation)

From `apps/wealth-management/src/shared/api/fmarket.ts`:

```typescript
fmarketApi.getProductsFilterNav(); // → Stocks/bonds/funds
fmarketApi.getIssuers(); // → Fund companies
fmarketApi.getBankInterestRates(); // → Bank deposit rates
fmarketApi.getProductDetails(); // → Fund certificate metadata
fmarketApi.getNavHistory(); // → Fund performance over time
fmarketApi.getUsdRateHistory(); // → USD/VND exchange rates ⭐ CRITICAL
fmarketApi.getGoldProducts(); // → Physical gold prices
```

**What This Maps To (Domain Capabilities)**:

- `getUsdRateHistory()` → **EXCHANGE_RATE_DATA** (Fmarket is authoritative for VN market)
- `getNavHistory()` → **FUND_PERFORMANCE_DATA** (Fmarket is primary)
- `getBankInterestRates()` → **BANKING_RATE_DATA** (Fmarket only source)
- `getProductsFilterNav()` → **FUND_FILTER_DATA** (Fmarket only source)

**Vnstock Provides**:

- Equity ticker prices (HOSE, HNX)
- Equity price history
- Market indices

---

## Domain Model Gaps

### What's Missing

The current domain has **no models for**:

1. **Exchange Rate Data** - Needed by portfolio valuation (convert USD/crypto to VND)
2. **Fund Data** - Needed by investment terminal
3. **Banking Rate Data** - Needed by savings comparison
4. **Provider Capability Declaration** - Needed to know which provider has which data

### What Needs to Exist

```go
// domain/exchange_rate.go
type ExchangeRateProvider interface {
    FetchExchangeRate(ctx context.Context, from, to string) (float64, error)
    FetchHistoricalRates(ctx context.Context, from, to string, startDate, endDate string) ([]ExchangeRate, error)
}

// domain/fund_data.go
type FundDataProvider interface {
    FetchFunds(ctx context.Context, assetTypes []string) ([]Fund, error)
    FetchFundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error)
}

// domain/banking_rate.go
type BankingRateProvider interface {
    FetchBankRates(ctx context.Context) ([]BankRate, error)
}

// port/market_provider.go (REVISED)
type MarketProvider interface {
    Name() string
    Health(ctx context.Context) (MarketProviderHealth, error)
    Capabilities() []string  // ← NEW: returns ["exchange_rate", "fund_data", "equity_ticker"]

    // Capability methods (with fallback chain)
    ExchangeRate(ctx context.Context, from, to string) (float64, error)
    FundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error)
    BankRates(ctx context.Context) ([]BankRate, error)
    EquityTicker(ctx context.Context, symbol string) (TickerData, error)
}
```

---

## Proposed Architecture: Capability-Based Routing

### 1. **Unified Domain Model**

```
domain/
├── market_provider.go          # Defines MarketProviderHealth
├── market_provider_config.go   # Defines provider config + priority
├── exchange_rate.go            # ExchangeRate model
├── fund_data.go                # Fund, NavPoint models
├── banking_rate.go             # BankRate model
└── ticker_data.go              # TickerData model
```

### 2. **Provider Capability Declaration**

```go
// domain/market_provider_config.go
type ProviderCapability string

const (
    CapabilityExchangeRate    ProviderCapability = "exchange_rate"
    CapabilityFundData        ProviderCapability = "fund_data"
    CapabilityBankingRate     ProviderCapability = "banking_rate"
    CapabilityEquityTicker    ProviderCapability = "equity_ticker"
    CapabilityGoldPrice       ProviderCapability = "gold_price"
)

type ProviderPriority struct {
    Provider     string                  // "fmarket", "vnstock"
    Capabilities []ProviderCapability    // What it can do
    Priority     int                     // 1 = highest
}

// Global priority config (in adapter/config/config.go)
var ProviderPriorityChain = []ProviderPriority{
    {Provider: "fmarket", Capabilities: []ProviderCapability{
        CapabilityExchangeRate,    // Fmarket is authoritative for VN rates
        CapabilityFundData,         // Fmarket is primary for fund data
        CapabilityBankingRate,      // Fmarket only source
        CapabilityGoldPrice,        // Fmarket only source
    }, Priority: 1},
    {Provider: "vnstock", Capabilities: []ProviderCapability{
        CapabilityEquityTicker,     // Fallback for equities
        CapabilityExchangeRate,     // Fallback if fmarket down
    }, Priority: 2},
}
```

### 3. **Service Layer Routing**

```go
// service/market_data_service.go (NEW)
type MarketDataService struct {
    providers map[string]port.MarketProvider
    priority  []ProviderPriority
    cache     port.CacheService
}

// ExchangeRate - tries fmarket first, falls back to vnstock
func (s *MarketDataService) ExchangeRate(ctx context.Context, from, to string) (float64, error) {
    cacheKey := fmt.Sprintf("exchange_rate:%s_%s", from, to)

    // Try cache
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        return parseFloat(cached), nil
    }

    // Try providers in priority order
    for _, prio := range s.priority {
        provider, ok := s.providers[prio.Provider]
        if !ok || !prio.hasCapability(CapabilityExchangeRate) {
            continue
        }

        rate, err := provider.ExchangeRate(ctx, from, to)
        if err == nil {
            s.cache.Set(ctx, cacheKey, rate, 300*time.Second) // Cache for 5 min
            return rate, nil
        }

        // Log error but continue to next provider
        log.Printf("ExchangeRate from %s failed: %v, trying next provider", prio.Provider, err)
    }

    return 0, fmt.Errorf("exchange rate unavailable from all providers")
}

// FundHistory - always prefers fmarket
func (s *MarketDataService) FundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error) {
    cacheKey := fmt.Sprintf("fund_history:%d:%s", fundID, navPeriod)

    // Try cache
    if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
        return parseNavPoints(cached), nil
    }

    fmarket := s.providers["fmarket"]
    if fmarket == nil {
        return nil, fmt.Errorf("fmarket provider not available")
    }

    history, err := fmarket.FundHistory(ctx, fundID, navPeriod)
    if err == nil {
        s.cache.Set(ctx, cacheKey, history, 600*time.Second) // Cache for 10 min
    }
    return history, err
}
```

### 4. **Provider Implementation Pattern**

```go
// adapter/market/fmarket/provider.go
type Provider struct {
    baseURL    string
    httpClient *http.Client
    cache      port.CacheService  // For internal caching
}

func (p *Provider) Name() string {
    return "fmarket"
}

func (p *Provider) Capabilities() []string {
    return []string{
        "exchange_rate",
        "fund_data",
        "banking_rate",
        "gold_price",
    }
}

func (p *Provider) ExchangeRate(ctx context.Context, from, to string) (float64, error) {
    // Internal cache check
    cacheKey := fmt.Sprintf("fmarket:exchange_rate:%s_%s", from, to)
    if cached, err := p.cache.Get(ctx, cacheKey); err == nil {
        return parseFloat(cached), nil
    }

    // Fetch from fmarket.vn via POST /api/fmarket (action: getUsdRateHistory)
    // + parse response
    // + cache internally
    // + return
}

func (p *Provider) FundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error) {
    // Relay to fmarket via POST /api/fmarket (action: getNavHistory)
}

func (p *Provider) BankRates(ctx context.Context) ([]BankRate, error) {
    // Relay to fmarket via POST /api/fmarket (action: getBankInterestRates)
}

// Vnstock doesn't implement these; only implements ExchangeRate as fallback
```

---

## HTTP API Layer (Revised Design)

### Current (Wrong)

```
GET /api/external/market/providers/:provider/health
GET /api/external/vnstock/health
GET /api/fmarket (legacy proxy)
```

### Proposed (Right)

```
# Provider health (unchanged)
GET /api/external/market/providers/:provider/health
GET /api/external/vnstock/health

# Market data capability endpoints (NEW - service level, not provider level)
GET /api/market/exchange-rate?from=USD&to=VND            # Routes to best provider
GET /api/market/fund-history?fundId=123&period=nav12m    # Routes to fmarket
GET /api/market/bank-rates                               # Routes to fmarket
GET /api/market/equity-ticker?symbol=ACB                 # Routes to vnstock (or fmarket fallback)

# Legacy compatibility (still works, but calls service layer)
POST /api/fmarket                                         # Relays to fmarket provider
```

---

## Data Flow Examples

### Exchange Rate Fetch (Priority Routing)

```
Client: GET /api/market/exchange-rate?from=USD&to=VND
  ↓
MarketDataHandler.ExchangeRate("USD", "VND")
  ↓
MarketDataService.ExchangeRate("USD", "VND")
  ├─ Check cache: "exchange_rate:USD_VND"
  │  └─ Hit? Return cached value
  │  └─ Miss? Continue
  ├─ Try provider[0] = "fmarket" (priority 1)
  │  └─ fmarket.ExchangeRate("USD", "VND")
  │     └─ Call POST /api/fmarket {action: "getUsdRateHistory", params: {...}}
  │     └─ Cache internally, return
  │        └─ Success? Cache in Redis, return to client ✅
  │        └─ Fail? Continue to next provider
  ├─ Try provider[1] = "vnstock" (priority 2)
  │  └─ vnstock.ExchangeRate("USD", "VND")
  │     └─ Call vnstock service
  │        └─ Success? Cache in Redis, return to client ✅
  │        └─ Fail? Continue
  └─ All failed? Return error 500
```

### Fund History Fetch (Fmarket-Only)

```
Client: GET /api/market/fund-history?fundId=123&period=nav12m
  ↓
MarketDataHandler.FundHistory(123, "nav12m")
  ↓
MarketDataService.FundHistory(123, "nav12m")
  ├─ Check cache: "fund_history:123:nav12m"
  │  └─ Hit? Return cached
  │  └─ Miss? Continue
  ├─ Check capabilities: only "fmarket" has CapabilityFundData
  ├─ fmarket.FundHistory(123, "nav12m")
  │  └─ Call POST /api/fmarket {action: "getNavHistory", params: {productId: 123, navPeriod: "nav12m"}}
  │     └─ Internal fmarket cache check (key: "fmarket:fund_history:123:nav12m")
  │     └─ Hit? Return cached
  │     └─ Miss? Fetch from fmarket.vn, cache internally, return
  └─ Parse response, cache in Redis (10 min TTL), return to client ✅
```

---

## Impact on WM-010 & WM-011 Tasks

### WM-010 Revision (What Actually Needs to Be Implemented)

**Before** (Wrong):

- "Create fmarket provider as a separate proxy"
- "7 actions dispatched independently"

**After** (Right):

1. **Fmarket Provider Implementation**:
   - Implements capability-based `port.MarketProvider` interface
   - Declares capabilities: `[exchange_rate, fund_data, banking_rate, gold_price]`
   - Methods: `ExchangeRate()`, `FundHistory()`, `BankRates()`, `GoldPrice()`
   - Each method: cache check → internal fmarket request → relay response → internal cache → return

2. **Domain Models** (NEW):
   - `ExchangeRate` struct
   - `Fund`, `NavPoint` structs
   - `BankRate` struct
   - `GoldPrice` struct

3. **Market Data Service** (NEW):
   - Replaces/expands `MarketProviderService`
   - Knows about provider priority chain
   - Routes requests to best provider
   - Handles caching at service level

4. **Handler Layer** (REVISED):
   - `POST /api/fmarket` - legacy compatibility (relays to fmarket provider)
   - `GET /api/market/exchange-rate` - service-level routing
   - `GET /api/market/fund-history` - service-level routing

### WM-011 Revision (What Actually Needs to Be Tested)

**Before** (Wrong):

- "Test each of 7 actions independently"

**After** (Right):

1. **Provider-Level Tests**:
   - Fmarket provider health check
   - Each capability method returns correct schema
   - Internal caching works

2. **Service-Level Tests**:
   - Priority routing: fmarket tried first, vnstock fallback
   - Cache hit/miss behavior
   - Error handling per provider

3. **Integration Tests**:
   - E2E: Client request → Service routing → Provider execution → Cache update → Response
   - Multi-provider scenario: fmarket down, vnstock fallback works
   - Capability declaration respected (fund data only from fmarket, never tries vnstock)

---

## Implementation Roadmap

### Phase 1: Domain & Port (Foundational)

- Create domain models: `exchange_rate.go`, `fund_data.go`, `banking_rate.go`, `ticker_data.go`
- Extend `port/market_provider.go` with capability methods
- Create `domain/market_provider_config.go` with priority chain

### Phase 2: Fmarket Provider (Data Source)

- Implement fmarket provider with capability methods
- Each method relays to `POST /api/fmarket` internally
- Internal caching per fmarket action

### Phase 3: Service & Routing (Integration)

- Create `service/market_data_service.go` with priority routing
- Implement cache-aware methods with fallback chain
- Service-level cache management

### Phase 4: HTTP API (Client Facing)

- Create `adapter/fiber/market_data_handler.go`
- Expose `/api/market/*` endpoints
- Keep `POST /api/fmarket` for backward compatibility

### Phase 5: Vnstock Migration (Fallback)

- Update vnstock provider to implement capability-based interface
- Declare capabilities (equity ticker, fallback exchange rate)
- Register in service priority chain

---

## Key Decisions

| Decision                       | Rationale                                                         |
| ------------------------------ | ----------------------------------------------------------------- |
| **Capability-Based Interface** | Allows runtime provider discovery and fallback chains             |
| **Service-Level Caching**      | Single point of cache management across all providers             |
| **Provider-Level Caching**     | Prevents thundering herd from internal relay calls                |
| **Dual-Layer Cache**           | Provider cache for internal relay, service cache for HTTP clients |
| **Priority Chain Config**      | Centralizes provider selection logic, easy to adjust              |
| **Fmarket as Primary**         | Higher data quality, faster response, more reliable for VN market |
| **Backward Compatibility**     | `POST /api/fmarket` still works, calls new provider interface     |

---

## Success Criteria

- [ ] All domain models defined
- [ ] `port.MarketProvider` interface supports capability methods
- [ ] Fmarket provider implements capability methods
- [ ] Service layer routes by capability + priority
- [ ] Cache hit ratio >80% on repeated requests
- [ ] Fallback chain works: fmarket down → vnstock fallback succeeds
- [ ] Tests cover: provider capability, service routing, cache behavior, error scenarios
- [ ] No breaking changes to HTTP API (backward compatible with `/api/fmarket`)
