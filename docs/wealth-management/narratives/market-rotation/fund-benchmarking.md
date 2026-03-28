# Details: Institutional Fund Benchmarking

## 1. Feature Meaning: The Flow of Capital

**Fund Benchmarking** provides a window into the actual movement of "Institutional Money."

## 2. API Source & Logic

- **Primary Source**: `POST /api/fmarket` (action: `getProductsFilterNav`, `getBankInterestRates`).
- **Benchmarking Insight**: This is where we compare **VND Stock Funds (Yield)** vs. **MMF/Bond Funds (Safe Harbor)**.

## 3. Business Use Cases (Actionable)

- **Defensive Positioning**: If `Fmarket` shows Bond Fund yields ($>~10\%$) outperforming most stock-based Mutual Funds, the system will highlight the "Shift-to-Safe" rotation.
- **Deposit Rate Arbitrage**: Comparing current "Bank Savings" from Fmarket against private bank rates in the `Accounts` sheet.

## 4. Why This Hub?

**Fmarket Insights** is the middle-man between the **Market Pulse (Macro)** and **Accounts (User)**. It shows the "Solution" for capital flight.
