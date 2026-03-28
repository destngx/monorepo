# User Stories: Investment Terminal (Market Alpha)

## 1. Persona Definition: Quantitative Investor

- **Role**: Specialized Wealth Manager focused on stock market research, technical analysis, valuation modeling, and trade execution.
- **Needs**: High-fidelity data, sentiment signals, historical seasonality, volatility analysis, and multi-timeframe confluence.

---

## 2. Market Pulse & Sentiment

### US-INV-001: Integrated Market Pulse (Telemetry)

- **Story**: As a Quantitative Investor, I want to see a single "Market Pulse" summary showing Top Gainers, Losers, and Liquid tickers, so that I can immediately identify where the "Money Flow" is concentrating.
- **Acceptance Criteria**:
  - Ticker list must include: Price, 1D % Change, and Total Value (Liquidity).
  - Tickers should be categorized into "Bullish" (e.g., Green RSI/MA) and "Bearish" (e.g., Red RSI/MA).

### US-INV-002: Real-time News Sentiment Score

- **Story**: As a Quantitative Investor, I want an AI-generated sentiment score for a specific ticker (e.g., "FPT") based on current news headlines, so that I can understand if the market is reacting to "Positive" or "Negative" catalysts.
- **Acceptance Criteria**:
  - Score must be from -1 (Extremely Bearish) to +1 (Extremely Bullish).
  - Must display 3 key summary bullet points for EACH ticker analyzed.

---

## 3. Technical & Quantitative Analysis

### US-INV-003: Multi-Timeframe Confluence (M-T-F)

- **Story**: As a Quantitative Investor, I want to see a ticker's trend across 1W, 1D, and 4H timeframes simultaneously, so that I can ensure I'm not "Buying a Dip in a Bear Market" on a higher timeframe.
- **Acceptance Criteria**:
  - Matrix must show MA crossover status and RSI for each timeframe.
  - Final "Action Recommendation" (e.g., "Full Long" vs. "Wait for 1D Reset") based on confluence.

### US-INV-004: Historical Seasonality Search

- **Story**: As a Quantitative Investor, I want to see the "Win Rate" of a stock like "VIC" over the last 10 years for each specific month (e.g., "Does VIC always go up in April?"), so that I can play high-probability seasonal setups.
- **Acceptance Criteria**:
  - Returns a Heatmap of Monthly Returns (%) for the selected ticker.
  - Calculation must filter out years with insufficient data.

---

## 4. Valuation & Simulation

### US-INV-005: Fair Value Valuation Modeling

- **Story**: As a Quantitative Investor, I want to input my own "Expected Dividend" or "Growth Rate" for a stock and see its "Fair Value" target, so that I can know if the current market price is "Cheaper" or "Expensive" relative to its fundamentals.
- **Acceptance Criteria**:
  - Supports Dividend Discount Model (DDM) or Earnings Multiple (P/E) models.
  - Result must show a "% Gap" between current price and target fair value.

---

## 5. Ticker Deep-Dive

### US-INV-006: Comprehensive Ticker Analysis Report

- **Story**: As a Quantitative Investor, I want to click a ticker and see "News," "Technicals," "Market Depth," and "Volatility," all on one page, so that I don't have to navigate to 5 different sub-menus.
- **Acceptance Criteria**:
  - Interface MUST allow one-click switching between these sub-terminals.
  - Data must refresh instantly upon ticker selection.

---

## 6. Portfolio Management & Risk

### US-INV-007: AI-Powered Portfolio Rebalancing Suggestions

- **Story**: As a Quantitative Investor, I want the AI to analyze my current asset allocation against my targets and suggest "Rebalancing Trades," so that I can maintain my desired risk profile during market surges or crashes.
- **Acceptance Criteria**:
  - AI must detect when an asset class (e.g., Crypto) deviates by more than 5% from its target.
  - Suggestion must include: "Sell X amount of Asset A" and "Buy Y amount of Asset B."
  - Must include a summary of "Reasoning" (e.g., _"Take profit on high-volatility surge"_).
  - Should integrate with the "Daily Briefing" for high-priority alerts.
