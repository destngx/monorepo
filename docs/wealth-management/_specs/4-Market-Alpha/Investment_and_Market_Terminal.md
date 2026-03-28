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

- **API Concurrency**: Parallelize HOSE (VN) and Binance (Global) calls for unified asset lists.
- **Update Frequency**: Ticker prices MUST refresh every 5 to 60 seconds (Configurable).
- **Caching**: Market summaries cached for 1 hour; live ticker prices uncached in "Active Terminal."

---

## 6. Engineering Success Criteria

- **Data Fidelity**: 100% price parity withHOSE/Binance sources.
- **Ticker Switch Speed**: Switching between tickers reveals deep-dive metrics in < 1s.
- **Discovery**: Users find high-liquidity stock opportunities 40% faster than with traditional bank terminals.
