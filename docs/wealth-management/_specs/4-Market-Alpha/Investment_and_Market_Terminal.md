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

The market data layer uses a **capability-based routing system** that intelligently routes requests to the most authoritative provider per asset class:

| Capability                            | Primary Source | Fallback | Rationale                                        |
| ------------------------------------- | -------------- | -------- | ------------------------------------------------ |
| **Equity Ticker** (VN stocks)         | Vnstock        | Fmarket  | Vnstock has native HOSE equities                 |
| **IFC Ticker** (VN fund certificates) | Fmarket        | Vnstock  | Fmarket specializes in Vietnamese funds          |
| **USD/VND Exchange Rate**             | Fmarket        | Vnstock  | Fmarket faster/more accurate for VN rates        |
| **Fund NAV History**                  | Fmarket        | —        | Fmarket only source for historical NAV data      |
| **Gold Price (SJC)**                  | Fmarket        | —        | Fmarket only source for Vietnamese physical gold |
| **Bank Interest Rates**               | Fmarket        | —        | Fmarket only source for deposit rate comparison  |
| **Crypto Ticker**                     | Binance        | —        | Native crypto exchange (future provider)         |

**Key Benefits**:

- ✅ Automatic fallback if primary provider fails (resilience)
- ✅ Config-driven routing (no code changes to add Provider 3 or swap priorities)
- ✅ Unified caching at capability level (prevents redundant upstream calls)
- ✅ Extensible for future providers (Binance, etc.)

See [\_technical/Market_Provider_Capabilities.md](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_technical/Market_Provider_Capabilities.md) for architecture details.

### 5.2 API Concurrency & Performance

- **API Concurrency**: Parallelize capability requests (e.g., fetch IFC + Equity + ExchangeRate simultaneously).
- **Unified Caching**: Capability-level cache prevents thundering herd (all subsequent requests within TTL use cached response).
- **Update Frequency**: Ticker prices refresh every 5-60s (configurable per capability; live prices bypass cache in Active Terminal).
- **Caching Strategy**:
  - Market summaries (fund lists, rates): cached for 5-10 minutes (MARKET_DATA_TTL)
  - Live ticker prices in Active Terminal: uncached or very short TTL (< 60s)
  - Historical data (NAV, rates): cached for 1 hour (stable data)

---

## 6. Engineering Success Criteria

- **Data Fidelity**: 100% price parity with primary source per asset class (Vnstock for equities, Fmarket for funds, etc.).
- **Fallback Resilience**: If primary provider fails, automatic fallback to secondary (if available) with < 500ms added latency.
- **Ticker Switch Speed**: Switching between tickers reveals deep-dive metrics in < 1s (using cached provider responses where possible).
- **Discovery**: Users find high-liquidity stock opportunities 40% faster than with traditional bank terminals.
- **Multi-Provider Routing**: All capability requests route to correct provider per routing config (no manual routing logic in handlers).
