# Specification: Investment Terminal & Market Alpha

**Author**: Product Owner / Financial Analyst  
**Intended Audience**: Engineering (Market Data & Frontend)  
**Status**: Approved for Development  
**Keywords**: Market Pulse, Ticker Deep-Dive, Technical Confluence, Alpha Signals

---

## 1. Product Intent

To provide high-fidelity financial data and AI-driven market analysis for ticker-specific deep-dives. This terminal is the primary interface for active stock and crypto tracking.

---

## 2. Functional Specification (Market Pulse)

**BA Requirement**: The user must immediately identify "Money Flow" (Top Gainers/Losers/Liquidity).

### 2.1 Integrated Market Telemetry

- **Visual**: Data table with Ticker, Price, 1D % Change, and Total Value.
- **Logic**: Sources from HOSE (VNStock) and Crypto (Binance).
- **AC**: Tickers MUST be categorized into "Bullish" (Green RSI/MA) and "Bearish" (Red RSI/MA).

### 2.2 Global Sentiment Score

- **Logic**: AI-generated score (from -1 to +1) based on current news headlines.
- **AC**: Must provide 3 bullet points of "Sentiment Reasoning" (e.g., "Positive Q3 Earnings," "Rate Cut Sentiment").

---

## 3. Ticker Deep-Dive (Technical Alpha)

### 3.1 Multi-Timeframe Confluence (M-T-F)

- **Visual**: 1W, 1D, 4H trends on a single view.
- **AC**: Matrix must show MA Crossover status and RSI levels simultaneously.

### 3.2 Historical Seasonality (Heatmap)

- **Logic**: "Win Rate" of a ticker over the last 10 years per specific month.
- **AC**: Returns a heatmap of monthly returns (%) for the selected asset.

---

## 4. Valuation & Simulation (The Fair Value)

- **Objective**: Inform the user if a price is "Cheap" or "Expensive" relative to fundamentals.
- **Logic**: Support for DDM (Dividend Discount Model) or Earnings Multiple (P/E) models.
- **AC**: Show "% Gap" between current price and AI/Mathematical Target.

---

## 5. Technical Requirements for Engineering

### 5.1 Multi-Provider Capability-Based Architecture

The market data layer is `apps/market-data-server`. It routes each request by capability and asset class to an ordered chain of providers, falling back to the next provider on failure. The engine only calls it (`adapter/market/marketdata`). Default chains, as implemented in market-data-server:

| Capability                                   | Chain (in order)                            |
| -------------------------------------------- | ------------------------------------------- |
| **Equity Ticker** (VN stocks)                | Vnstock, Fmarket                            |
| **IFC Ticker** (VN fund certificates)        | Fmarket, Vnstock                            |
| **Gold Ticker**                              | Fmarket                                     |
| **Crypto Ticker**                            | CoinGecko                                   |
| **USD/VND Exchange Rate**                    | Fmarket, Yahoo, CurrencyAPI                 |
| **USDT/VND Exchange Rate**                   | CoinGecko, CurrencyAPI                      |
| **Price Series** (equity / gold_usd / forex) | Vnstock, Fmarket / Fmarket / Yahoo, Fmarket |
| **Bank Interest Rates**                      | Fmarket                                     |
| **Candles**                                  | Vnstock, Yahoo, Fmarket                     |

Results are cached per capability by market-data-server (see [MARKET_PLATFORM.md §3.3](../../../market-data-server/MARKET_PLATFORM.md#33-default-chains-and-ttls) for TTLs). See [Market provider capabilities §0](../../_technical/Market_Provider_Capabilities.md#0-current-implementation-authoritative) for the engine side and [REQUIREMENTS.md §9.2](../../REQUIREMENTS.md#92-market-data-delegated-to-market-data-server) for status.

### 5.2 API Concurrency & Performance

- **API Concurrency**: Parallelize capability requests (e.g., fetch IFC + Equity + ExchangeRate simultaneously).
- **Unified Caching**: Capability-level cache prevents thundering herd (all subsequent requests within TTL use cached response).
- **Update Frequency**: Ticker prices refresh every 5-60s (configurable per capability; live prices bypass cache in Active Terminal).
- **Caching Strategy** (target; market-data-server currently uses per-capability TTLs):
  - Market summaries (fund lists, rates): cached for 5-10 minutes (per-capability server TTLs)
  - Live ticker prices in Active Terminal: uncached or very short TTL (< 60s; the server's ticker TTL is 60 s)
  - Historical data (NAV, rates): cached for 1 hour (stable data)

---

## 6. Engineering Success Criteria

- **Data Fidelity**: 100% price parity with primary source per asset class (Vnstock for equities, Fmarket for funds, etc.).
- **Fallback Resilience**: If primary provider fails, automatic fallback to secondary (if available) with < 500ms added latency.
- **Ticker Switch Speed**: Switching between tickers reveals deep-dive metrics in < 1s (using cached provider responses where possible).
- **Discovery**: Users find high-liquidity stock opportunities 40% faster than with traditional bank terminals.
- **Multi-Provider Routing**: All capability requests route to correct provider per routing config (no manual routing logic in handlers).
