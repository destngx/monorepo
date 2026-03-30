# Narrative: Market Provider Architecture Evolution

**Author**: System Architect  
**Audience**: Engineering Team & Future Maintainers  
**Purpose**: Explain the design rationale behind capability-based market provider architecture  
**Context**: Supporting multiple market data sources (Fmarket, Vnstock, future Binance, etc.)

---

## The Problem We Solved

### Initial Approach: Action-Based Proxy

When integrating Fmarket, the initial approach was to build a **relay handler**:

```
User Request
  ↓
HTTP Handler (POST /api/fmarket)
  ├─ Dispatcher: action = "getProductsFilterNav"?
  ├─ Dispatcher: action = "getIssuers"?
  ├─ Dispatcher: action = "getBankInterestRates"?
  └─ ...7 more actions
  ↓
Single Upstream: api.fmarket.vn
```

**Problems with this approach:**

1. **Not extensible**: Adding Vnstock or Binance requires new handlers or massive refactoring
   - `POST /api/fmarket` (7 actions)
   - `POST /api/vnstock` (different 5 actions)
   - `POST /api/binance` (different 3 actions)
   - Client code must know which action maps to which provider

2. **No unified routing**: No intelligent selection of "best provider for this data type"
   - If Fmarket is down, client still tries `/api/fmarket` first (fails)
   - Client must implement fallback logic (violates separation of concerns)

3. **Duplicate caching logic**: Each provider handler manages its own cache
   - If both Fmarket and Vnstock support "exchange rate", both are cached separately
   - User sees slight differences if querying both

4. **Tight coupling**: Business logic scattered across handlers
   - Handler decides what to cache and for how long
   - Hard to change routing without modifying code

---

## The Solution: Capability-Based Routing

We redesigned around **capabilities** (domain abilities) rather than **actions** (provider-specific operations):

```
User Request: "Get IFC Ticker for FUND001"
  ↓
Generic Handler (GET /api/market/ticker?symbol=FUND001&type=ifc)
  ↓
Service Layer: Look up routing config
  ├─ Capability: getTicker
  ├─ Type: ifc
  ├─ Config says: [Fmarket (primary), Vnstock (fallback)]
  ↓
Try Fmarket: GetTicker("FUND001", ifc)
  ├─ Success → Cache & return
  └─ Failure → Try fallback
Try Vnstock: GetTicker("FUND001", ifc)
  ├─ Success → Cache & return
  └─ Error → Return error
```

### Why This is Better

#### 1. **Extensible to N Providers**

Adding Binance for crypto tickers:

**Old approach** (action-based):

- Create new `adapter/fiber/binance_handler.go`
- Implement 10+ handler methods
- Update client to distinguish between `/api/fmarket`, `/api/vnstock`, `/api/binance`
- Client code must try multiple endpoints on failure

**New approach** (capability-based):

- Create `adapter/market/binance/provider.go` implementing `port.MarketProvider`
- Add one line to routing config: `getTicker.crypto: [binance]`
- Register in `main.go`: `providers["binance"] = NewBinanceProvider(...)`
- **No client changes needed.** The same endpoint works:
  ```bash
  curl /api/market/ticker?symbol=BTCUSDT&type=crypto  # Binance (config-driven)
  ```

#### 2. **Intelligent Routing Per Data Type**

Same endpoint, different providers based on **what you're asking for**:

```bash
# IFC ticker → Fmarket (specialist)
curl /api/market/ticker?symbol=FUND001&type=ifc

# Equity ticker → Vnstock (has HOSE data natively)
curl /api/market/ticker?symbol=ACB&type=equity

# Crypto ticker → Binance (future provider, config-driven)
curl /api/market/ticker?symbol=BTCUSDT&type=crypto
```

All use the same endpoint (`/api/market/ticker`). The **service layer routes internally** based on type.

#### 3. **Fallback Chains Without Client Logic**

If Fmarket is temporarily down:

```bash
# User requests IFC ticker
curl /api/market/ticker?symbol=FUND001&type=ifc

# Service tries:
#   1. Fmarket → timeout error
#   2. Vnstock → success
# User gets response (from fallback, no explicit error for transient failure)
```

**No client retry logic needed.** The service handles it transparently.

#### 4. **Unified Caching**

Cache key is **capability-based**, not provider-based:

```go
cacheKey := "market:getTicker:ifc:FUND001"
```

If both Fmarket and Vnstock support IFC tickers (hypothetically), they **share the same cache**:

- Request 1 (Fmarket): cache miss → fetch → cache with TTL
- Request 2 (Vnstock, if Fmarket down): cache hit → serve from cache (no fetch)

Result: **Fewer upstream calls, faster responses, better resilience.**

---

## Architectural Implications

### Port Interface is the Contract

The `port.MarketProvider` interface is **the contract**. It doesn't change when adding new providers:

```go
type MarketProvider interface {
    Name() string
    Health(ctx context.Context) error
    GetTicker(ctx context.Context, symbol string, tickerType TickerType) (*Ticker, error)
    GetExchangeRate(ctx context.Context, from, to Currency) (*ExchangeRate, error)
    GetFundNavHistory(ctx context.Context, symbol string) ([]*NavPoint, error)
    GetGoldPrice(ctx context.Context) (*Price, error)
    GetBankInterestRate(ctx context.Context) ([]*BankRate, error)
}
```

Each provider implements these methods. Some return `ErrUnsupported` if they don't handle the type (service tries fallback).

### Service Layer is the Orchestrator

The service layer is **single-responsibility**: route by capability + manage fallback chains.

```go
// Service: "Get me a ticker of type 'ifc'"
// 1. Look up routing config: getTicker.ifc → [Fmarket, Vnstock]
// 2. Try Fmarket.GetTicker(..., ifc)
// 3. If fails, try Vnstock.GetTicker(..., ifc)
// 4. Cache at capability level
// 5. Return response
```

### Routing Config is Injected

The routing config is **not hardcoded**. It's injected at startup, enabling:

- **Different environments**: Dev uses mock providers, prod uses real providers
- **A/B testing**: Swap provider priorities without code changes
- **Feature flags**: Enable/disable providers per user or region
- **Circuit breakers**: If Fmarket is broken, config can temporarily exclude it

```yaml
# Easy to change without code changes:
capabilities:
  getTicker:
    ifc: [fmarket, vnstock] # Primary-fallback chain
    equity: [vnstock, fmarket] # Different order for equities
    crypto: [binance] # Only Binance supports
```

---

## Historical Comparison: Why We Didn't Do Action-Based

If we had implemented the initial action-based proxy approach, **adding a second provider would be painful**:

**WM-010 (Action-Based Proxy)**:

- Fmarket: 7 actions (getProductsFilterNav, getIssuers, getNavHistory, etc.)
- Handler logic: `if action == "getProductsFilterNav" { ... }`
- Caching: per-action cache keys

**WM-012 (Hypothetical Vnstock Integration)**:

- Vnstock: 5 different actions (getEquityList, getEquityPrice, getFxRate, etc.)
- New handler: `POST /api/vnstock` with different action names
- Caching: duplicate keys, duplicate TTL logic
- Client: must call `/api/fmarket` for funds, `/api/vnstock` for equities
- **No intelligent routing**: Client decides which provider, not the service

**Total lines of code to add Vnstock**: ~1000+ lines across handlers, caching, client code

**With capability-based approach**:

- One provider file: `adapter/market/vnstock/provider.go` (~300 lines)
- Register in `main.go`: ~5 lines
- Update routing config: ~5 lines
- **No handler changes needed**

**Total lines of code to add Vnstock**: ~310 lines (97% reduction)

---

## When This Pattern Shines

### Use Capability-Based Routing When:

✅ **Multiple providers** (2+) offering overlapping or complementary services  
✅ **Smart selection needed**: Different providers for different types/asset classes  
✅ **Fallback resilience needed**: Automatic failover if primary provider fails  
✅ **Extensibility critical**: Plan to add more providers (Binance, other exchanges, etc.)  
✅ **Config-driven behavior**: Want to change routing without code changes

### When Action-Based Proxy Suffices:

❌ Only one provider forever (but rare)  
❌ Provider actions are completely unrelated (not overlapping)  
❌ No fallback needed (if provider down, service is down)  
❌ Extremely simple integration (single endpoint, no logic)

**Our case**: Clearly needs capability-based (Fmarket + Vnstock, plan for Binance, overlapping tickers/rates).

---

## Testing Philosophy

The capability-based architecture enables **deterministic testing without complexity**:

```go
// Mock providers independently
mockFmarket := &MockProvider{
    getTicker: func(sym, type) { ... }
}
mockVnstock := &MockProvider{
    getTicker: func(sym, type) { ... }
}

// Test routing logic (service picks correct provider)
func TestRoutingSelectsFmarket(t *testing.T) {
    routing := RoutingConfig{
        getTicker.ifc: [Fmarket, Vnstock],
    }
    // Service should try Fmarket first for IFC
}

// Test fallback logic (service retries on failure)
func TestFallbackOnPrimaryFailure(t *testing.T) {
    mockFmarket.getTicker = returns error
    mockVnstock.getTicker = returns success
    // Service should try Fmarket, then Vnstock, return Vnstock result
}

// Test caching (unified at capability level)
func TestCacheKeyPerCapability(t *testing.T) {
    key := service.cacheKey("getTicker", "ifc", "FUND001")
    // Same key whether Fmarket or Vnstock fetches (unified cache)
}
```

**Result**: All behavior tested without mocking internal service logic. Tests are stable and easy to understand.

---

## Cost-Benefit Analysis

### Development Cost

| Aspect                             | Action-Based Proxy                  | Capability-Based                      |
| ---------------------------------- | ----------------------------------- | ------------------------------------- |
| **Initial WM-010**                 | 500 lines (handler + caching)       | 1000 lines (port + service + routing) |
| **Adding Provider 2 (Vnstock)**    | 1000 lines (new handler, dup logic) | 300 lines (new provider)              |
| **Adding Provider 3 (Binance)**    | 1000+ lines                         | 300 lines                             |
| **Changing priorities (A/B test)** | Code change + redeploy              | Config change + restart               |
| **Testing**                        | Complex (mocks at multiple layers)  | Simple (mock at port layer)           |

### Long-Term Value

| Scenario                | Proxy Approach                              | Capability-Based                               |
| ----------------------- | ------------------------------------------- | ---------------------------------------------- |
| **After 3 providers**   | 2500+ lines of duplication                  | 1300 lines (extensible)                        |
| **Fallback on failure** | Client implements retry logic               | Service handles transparently                  |
| **Routing change**      | Code review + testing + deploy              | Config file change                             |
| **New developer**       | "Which provider has field X?" → Search code | "Which provider for X?" → Check routing config |

---

## Conclusion

The **capability-based market provider architecture** is the right pattern for this system because:

1. **Multiple providers exist and overlap** (Fmarket + Vnstock + future Binance)
2. **Different providers are best for different data types** (fund vs equity vs crypto)
3. **Extensibility is critical** (easy to add Provider 4, 5, 6 in the future)
4. **Config-driven routing enables operational flexibility** (no code changes for priority swaps)
5. **Unified caching and fallback reduce operational complexity** (service, not client, handles failures)

This pattern aligns with **Hexagonal Architecture** (clean boundaries between business logic and providers) and **SOLID principles** (single responsibility, dependency injection, open/closed for extension).

---

**Last Updated**: March 30, 2026  
**Status**: Reference Documentation (explains decision rationale for WM-010/WM-011)
