# Use Case Specification: Investment Terminal (Market Alpha)

## UC-INV-001: Integrated Market Pulse Audit (Start-of-Day Analysis)

### 1. Summary

The user opens the Market Pulse terminal to understand current liquidity flows and identifies potential high-momentum tickers.

### 2. Actors

- **Primary Actor**: Quantitative Investor (User)
- **Secondary Actor**: VNStock API, Market Pulse API.

### 3. Basic Flow

1.  User clicks into the "Investments" tab.
2.  System defaults to the "Market Pulse" sub-terminal.
3.  System parallel-fetches "Top Gainer," "Top Loser," and "Most Liquid" lists.
4.  User sees "FPT" is the #1 liquid ticker with +2.5% gain.
5.  User hovers over the "Trend" icon for FPT.
6.  System displays "RSI: 65 (Bullish), Price > MA20 (Safe)."
7.  User identifies "HPG" as a lagging ticker (-1.0%).
8.  System shows a "Daily Summary" of the whole market (e.g., "Market: Neutral-Bullish").

### 4. Alternative/Exception Flows

- **API Stale (E1)**: If VNStock is delayed, system shows "Last Update: [TS]" with a clear status.

---

## UC-INV-002: Ticker Confluence Verification (Multi-Timeframe)

### 1. Summary

User wants to buy FPT but checks if the Weekly (1W) trend is still bullish to avoid a "Fake-out."

### 2. Actors

- **Primary Actor**: Quantitative Investor (User)
- **Secondary Actor**: Technical Analysis Engine.

### 3. Basic Flow

1.  User enters `FPT` into the "Ticker Analysis" field.
2.  User navigates to the "M-T-F Confluence" tab.
3.  System parallel-fetches OHLCV data for 1W, 1D, and 4H timeframes.
4.  System calculates "MA20/50 Cross" and "RSI" for each timeframe.
5.  System displays results in a (3x2) matrix:
    - 1W: Bullish (Price > MA)
    - 1D: Overbought (RSI > 70)
    - 4H: Pullback (Price < MA20)
6.  System renders a "Final Recommendation": "Hold - Wait for 1D RSI reset."
7.  User decides to cancel their "Buy" order until 1D RSI drops below 60.

### 4. Postconditions

- User has avoided a high-risk trade based on multi-timeframe divergence.

---

## UC-INV-003: Seasonal Setup Audit (April Strategy)

### 1. Summary

It's late March, and the user wants to identify which stocks typically go up in April ("The Dividend Season").

### 2. Actors

- **Primary Actor**: Quantitative Investor (User)

### 3. Basic Flow

1.  User clicks the "Seasonality" tab in the Investment Terminal.
2.  User enters `VIC` in the search bar.
3.  System historical-fetches VIC price data for the last 10 years.
4.  System calculates the average % return for each month (Jan..Dec).
5.  System identifies that "April" has a 90% Win-rate for VIC over 10 years.
6.  User sees a "Heatmap" representation of these monthly returns.
7.  User repeats the check for `VNM` and `FPT`.
8.  User identifies `FPT` also has a 100% Win-rate in April.

### 4. Postconditions

- User has identified a statistically high-probability seasonal trade strategy for the next month.
