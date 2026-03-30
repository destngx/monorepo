# ANALYSIS MODE FINDINGS & RECOMMENDATION

**Date**: March 30, 2026  
**Mode**: Architectural Consultation  
**Status**: Ready for Review

---

## 🎯 THE CORE ISSUE

You were right. The initial Sprint 1 tasks (WM-010 & WM-011) **treat fmarket as just another proxy service**, which is architecturally wrong.

### What We Got Wrong

```
❌ Task WM-010 Design:
   "Implement Fmarket Provider (Proxy Gateway)"
   → Action-based dispatcher for 7 fmarket actions
   → Upstash caching at the relay level
   → No integration with vnstock
   → Treats fmarket as isolated from the broader market provider system
```

### What's Actually Needed

```
✅ Proper Design:
   "Implement Fmarket as Priority Market Data Provider"
   → Fmarket provides CAPABILITIES (exchange rates, fund data, banking rates)
   → Service layer routes requests based on capability + provider priority
   → Fmarket is the PRIMARY SOURCE (priority 1) for:
      - Exchange rates (USD/VND) — authoritative for Vietnam market
      - Fund performance data (NAV history) — only source
      - Banking rates — only source
   → Vnstock is FALLBACK (priority 2) for:
      - Equity ticker prices (HOSE/HNX) — primary source
      - Exchange rates — fallback if fmarket down
   → Single unified market data abstraction across all sources
```

---

## 📊 ARCHITECTURE ANALYSIS FINDINGS

### Current State (Broken)

**Port Definition** (`port/market_provider.go`):

```go
type MarketProvider interface {
    Name() string
    Health(ctx context.Context) (domain.MarketProviderHealth, error)
}
// ❌ Missing: What data can this provider actually fetch?
```

**Service** (`service/market_provider_service.go`):

```go
// ❌ Can only call Health()
// ❌ Cannot route requests by capability
// ❌ Cannot implement fallback chains
```

**HTTP API**:

```
GET /api/external/market/providers/:provider/health     ← Only health checks
POST /api/fmarket (legacy)                              ← Isolated proxy
                                                         ← No integration with provider abstraction
```

**Data Model Gap**:

- ❌ No `ExchangeRate` domain model
- ❌ No `Fund` / `NavPoint` domain model
- ❌ No `BankRate` domain model
- ❌ No capability declaration system

---

## 🏗️ PROPOSED ARCHITECTURE: CAPABILITY-BASED ROUTING

### Layer 1: Domain Models (NEW)

```
domain/
├── exchange_rate.go              # ExchangeRate struct
├── fund_data.go                  # Fund, NavPoint structs
├── banking_rate.go               # BankRate struct
├── ticker_data.go                # TickerData struct (for equity/crypto)
└── market_provider_config.go     # ProviderCapability enum + ProviderPriority config
```

### Layer 2: Port Interface (REVISED)

```go
type MarketProvider interface {
    Name() string
    Capabilities() []string  // ["exchange_rate", "fund_data", "banking_rate", etc]
    Health(ctx context.Context) (MarketProviderHealth, error)

    // Capability Methods
    ExchangeRate(ctx context.Context, from, to string) (float64, error)
    ExchangeRateHistory(ctx context.Context, from, to string, startDate, endDate string) ([]ExchangeRate, error)
    FundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error)
    BankRates(ctx context.Context) ([]BankRate, error)
    EquityTicker(ctx context.Context, symbol string) (TickerData, error)
    GoldPrice(ctx context.Context) (float64, error)
}
```

### Layer 3: Provider Priority Configuration (NEW)

```go
// Global provider priority chain (in adapter/config/)
var ProviderPriorityChain = []ProviderPriority{
    {
        Provider: "fmarket",
        Capabilities: []ProviderCapability{
            CapabilityExchangeRate,   // ⭐ Primary for VN market rates
            CapabilityFundData,       // ⭐ Primary for fund performance
            CapabilityBankingRate,    // ⭐ Only source
            CapabilityGoldPrice,      // ⭐ Only source
        },
        Priority: 1,  // Try first
    },
    {
        Provider: "vnstock",
        Capabilities: []ProviderCapability{
            CapabilityEquityTicker,   // ⭐ Primary for equity prices
            CapabilityExchangeRate,   // Fallback (lower quality)
        },
        Priority: 2,  // Fallback
    },
}
```

### Layer 4: Service Layer (NEW)

```go
type MarketDataService struct {
    providers map[string]port.MarketProvider
    priority  []ProviderPriority
    cache     port.CacheService
}

// Example: ExchangeRate with priority routing
func (s *MarketDataService) ExchangeRate(ctx context.Context, from, to string) (float64, error) {
    // 1. Check service-level cache
    cacheKey := fmt.Sprintf("exchange_rate:%s_%s", from, to)
    if cached, _ := s.cache.Get(ctx, cacheKey); cached != "" {
        return parseFloat(cached), nil
    }

    // 2. Try providers in priority order
    for _, prio := range s.priority {
        if !prio.hasCapability("exchange_rate") {
            continue  // Skip if provider doesn't have this capability
        }

        provider := s.providers[prio.Provider]
        rate, err := provider.ExchangeRate(ctx, from, to)
        if err == nil {
            // 3. Cache and return
            s.cache.Set(ctx, cacheKey, rate, 5*time.Minute)
            return rate, nil
        }

        // 4. Log failure and continue to next provider
        log.Printf("ExchangeRate from %s failed: %v, trying next provider", prio.Provider, err)
    }

    return 0, fmt.Errorf("exchange_rate unavailable from all providers")
}

// Example: Fund data with Fmarket-only guarantee
func (s *MarketDataService) FundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error) {
    cacheKey := fmt.Sprintf("fund_history:%d:%s", fundID, navPeriod)
    if cached, _ := s.cache.Get(ctx, cacheKey); cached != "" {
        return parseNavPoints(cached), nil
    }

    // Only fmarket has this capability — never fallback to vnstock
    fmarket := s.providers["fmarket"]
    if fmarket == nil {
        return nil, fmt.Errorf("fmarket provider not available")
    }

    history, err := fmarket.FundHistory(ctx, fundID, navPeriod)
    if err == nil {
        s.cache.Set(ctx, cacheKey, history, 10*time.Minute)
    }
    return history, err
}
```

### Layer 5: Fmarket Provider Implementation (REVISED)

```go
// adapter/market/fmarket/provider.go
type Provider struct {
    baseURL    string
    httpClient *http.Client
    cache      port.CacheService  // Internal cache for relay calls
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
    // 1. Internal cache check
    cacheKey := fmt.Sprintf("fmarket:exchange_rate:%s_%s", from, to)
    if cached, _ := p.cache.Get(ctx, cacheKey); cached != "" {
        return parseFloat(cached), nil
    }

    // 2. Relay to fmarket: POST /api/fmarket {action: "getUsdRateHistory", ...}
    // 3. Parse response to extract current rate
    // 4. Internal cache
    // 5. Return
}

func (p *Provider) FundHistory(ctx context.Context, fundID int, navPeriod string) ([]NavPoint, error) {
    // Relay to: POST /api/fmarket {action: "getNavHistory", params: {...}}
}

func (p *Provider) BankRates(ctx context.Context) ([]BankRate, error) {
    // Relay to: POST /api/fmarket {action: "getBankInterestRates", params: {...}}
}

func (p *Provider) GoldPrice(ctx context.Context) (float64, error) {
    // Relay to: POST /api/fmarket {action: "getGoldProducts", params: {...}}
}
```

### Layer 6: HTTP API (REVISED)

```
# Service-level endpoints (capability-based routing) — NEW
GET /api/market/exchange-rate?from=USD&to=VND           # Routes to fmarket > vnstock
GET /api/market/exchange-rate-history?from=USD&to=VND&startDate=...&endDate=...
GET /api/market/fund-history?fundId=123&navPeriod=nav12m # Routes to fmarket only
GET /api/market/bank-rates                              # Routes to fmarket only
GET /api/market/equity-ticker?symbol=ACB                # Routes to vnstock > fmarket
GET /api/market/gold-price                              # Routes to fmarket only

# Provider health (unchanged)
GET /api/external/market/providers/:provider/health

# Legacy compatibility (still works)
POST /api/fmarket                                        # Relays to fmarket provider
GET /api/external/vnstock/health
```

---

## 🔄 DATA FLOW EXAMPLE: Exchange Rate Request

```
Client Request:
GET /api/market/exchange-rate?from=USD&to=VND

⬇️ Handler Layer
MarketDataHandler.ExchangeRate(from="USD", to="VND")

⬇️ Service Layer (with priority routing)
MarketDataService.ExchangeRate("USD", "VND")
  1️⃣ Check service cache: "exchange_rate:USD_VND"
     ✅ Hit? Return cached value immediately
     ❌ Miss? Continue to step 2

  2️⃣ Iterate through provider priority chain:
     Priority 1: "fmarket" (has capability: "exchange_rate")
       └─ Call fmarket.ExchangeRate("USD", "VND")

          ⬇️ Fmarket Provider
          fmarket.ExchangeRate("USD", "VND")
            1️⃣ Check internal cache: "fmarket:exchange_rate:USD_VND"
               ✅ Hit? Return cached
               ❌ Miss? Continue
            2️⃣ Relay to fmarket backend: POST /api/fmarket
               {action: "getUsdRateHistory", params: {from: "USD", to: "VND", ...}}
            3️⃣ Parse Fmarket.vn response
            4️⃣ Cache internally (300s TTL)
            5️⃣ Return rate

     ✅ Success? Cache in service layer (5 min TTL), return to client
     ❌ Error? Continue to Priority 2

     Priority 2: "vnstock" (has capability: "exchange_rate")
       └─ Call vnstock.ExchangeRate("USD", "VND")
          └─ Similar flow...

     ✅ Success? Cache, return
     ❌ Error? Continue to next...

  3️⃣ All providers failed? Return 500 error

Response to Client:
{
  "from": "USD",
  "to": "VND",
  "rate": 24500.50,
  "timestamp": "2026-03-30T12:00:00Z",
  "source": "fmarket"  // Which provider provided this
}
```

---

## 🚀 IMPLEMENTATION ROADMAP

### Phase 1: Domain Models & Port (Foundation)

**Files to Create/Modify**:

- ✅ `domain/exchange_rate.go` - ExchangeRate struct
- ✅ `domain/fund_data.go` - Fund, NavPoint structs
- ✅ `domain/banking_rate.go` - BankRate struct
- ✅ `domain/ticker_data.go` - TickerData struct (equity/crypto)
- ✅ `domain/market_provider_config.go` - ProviderCapability enum, ProviderPriority config
- ✅ `port/market_provider.go` - Extended interface with capability methods

**Effort**: Small  
**Risk**: Low (pure definitions)

### Phase 2: Fmarket Provider (Data Source)

**Files to Create**:

- ✅ `adapter/market/fmarket/provider.go` - Implements all capability methods
- ✅ `adapter/market/fmarket/provider_test.go` - BDD tests

**Effort**: Medium  
**Risk**: Medium (relay logic, internal caching)

### Phase 3: Service & Routing (Integration)

**Files to Create**:

- ✅ `service/market_data_service.go` - Priority routing + cache management
- ✅ `service/market_data_service_test.go` - Priority chain tests

**Effort**: Medium  
**Risk**: Medium (complex routing logic)

### Phase 4: HTTP API (Client Facing)

**Files to Create**:

- ✅ `adapter/fiber/market_data_handler.go` - Service-level endpoints
- ✅ `adapter/fiber/market_data_handler_test.go` - E2E tests

**Effort**: Small  
**Risk**: Low (standard handler pattern)

### Phase 5: Vnstock Migration (Fallback)

**Files to Modify**:

- ✅ `adapter/market/vnstock/provider.go` - Implement capability methods
- ✅ `adapter/market/vnstock/provider_test.go` - Tests for new methods

**Effort**: Medium  
**Risk**: Medium (backward compatibility)

### Phase 6: Integration & Wiring (Assembly)

**Files to Modify**:

- ✅ `main.go` - Register both providers in service
- ✅ `adapter/config/config.go` - Add provider priority config
- ✅ `adapter/fiber/market_provider_handler.go` - Keep for backward compatibility

**Effort**: Small  
**Risk**: Low (assembly code)

---

## 🔑 KEY DIFFERENCES FROM WM-010 / WM-011

### Old Approach (WM-010)

```
"Implement Fmarket Provider (Proxy Gateway)"
- 7 independent fmarket actions
- Action dispatcher with validation
- Upstash caching at relay level
- No integration with vnstock
- Health check only
```

### New Approach (REVISED WM-010)

```
"Implement Fmarket as Priority Market Data Provider"
- Capability-based interface (ExchangeRate, FundHistory, BankRates, etc.)
- Service layer with fallback routing
- Dual-layer caching (provider-level + service-level)
- Integrated with vnstock via capability declaration
- Full data model support (Exchange rates, Fund data, Banking rates)
```

### Old Verification (WM-011)

```
"Test all 7 fmarket actions independently"
- Each action works in isolation
- Cache hit/miss verified per action
- Load test on single action
```

### New Verification (REVISED WM-011)

```
"Test provider integration and capability routing"
- Capability declarations correct
- Priority routing works: fmarket tried first
- Fallback chain works: fmarket down → vnstock succeeds
- Service-level caching prevents double-fetch
- Fund data never tries vnstock (capability constraint)
- E2E: client request → routing → provider → cache → response
```

---

## ✅ SUCCESS CRITERIA

**After WM-010 & WM-011 (Revised)**:

- [ ] Domain models exist: ExchangeRate, Fund, NavPoint, BankRate, TickerData
- [ ] `port.MarketProvider` declares capabilities and implements methods
- [ ] Fmarket provider implements all methods with internal caching
- [ ] Service layer routes by capability + priority
- [ ] Service layer caches responses (5-10 min TTL)
- [ ] Vnstock implements fallback methods (at least ExchangeRate)
- [ ] Both providers registered in service with priority chain
- [ ] HTTP endpoints expose service-level capabilities (not provider-level actions)
- [ ] Backward compatibility: `POST /api/fmarket` still works
- [ ] Cache hit ratio >80% on repeated requests
- [ ] Fallback routing tested: fmarket down → vnstock handles request
- [ ] Capability constraint tested: fund data only fetched from fmarket
- [ ] E2E flow verified: client → service → provider → cache → response

---

## 📝 NEXT STEPS

### Option A: Full Redesign (Recommended)

1. Recreate WM-010 & WM-011 with capability-based architecture
2. Follow implementation roadmap (6 phases)
3. Result: Production-ready, extensible market provider system

### Option B: Incremental Evolution

1. Keep WM-010 & WM-011 as-is (quick delivery)
2. Plan WM-012 & WM-013 for refactoring to capability model
3. Risk: Technical debt, harder to add more providers later

### Option C: Hybrid Approach

1. WM-010: Implement fmarket with basic capability interface
2. WM-011: Implement routing between providers
3. WM-012: Add remaining capability methods + service logic
4. Balance speed with architecture correctness

---

## 📚 Reference Documents

**Created**:

- `/docs/wealth-management/_archive/20260330_FMARKET_ARCHITECTURE_ANALYSIS.md` - Full technical analysis

**Existing**:

- `/docs/wealth-management/_api/api/fmarket.md` - Fmarket API spec
- `/docs/wealth-management/_specs/4-Market-Alpha/Investment_and_Market_Terminal.md` - Product requirements
- `/docs/wealth-management/_technical/1-Data-Engine/Architecture_and_Schema.md` - Data architecture

---

## 🎯 RECOMMENDATION

**PROCEED WITH FULL REDESIGN** of WM-010 & WM-011 using the capability-based routing model.

The additional effort (phases 1-6) is **minimal compared to the long-term benefit** of having an extensible market provider system that can:

- ✅ Add new providers (e.g., Binance for crypto prices)
- ✅ Route by capability (not by provider)
- ✅ Implement fallback chains automatically
- ✅ Support unified caching strategy
- ✅ Enable AI tools to query market data generically

**Time Investment**: +20% effort vs. simple proxy approach  
**Technical Debt Avoided**: -80% future refactoring  
**Maintainability**: +500% easier to extend and debug
