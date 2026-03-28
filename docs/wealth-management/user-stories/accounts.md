# User Stories: Portfolio Accounts (Asset Distribution)

## 1. Persona Definition: Asset Allocator

- **Role**: Specialized Wealth Manager focused on capital distribution, risk management, and liquidity.
- **Needs**: Comprehensive accounts list, asset class categorization, sub-ledger details, and historical balance trends.

---

## 2. Asset visibility

### US-ACC-001: Consolidated Accounts List

- **Story**: As an Asset Allocator, I want a unified list of all my financial accounts (Banks, Brokerages, Wallets), so that I don't have to log into 10 different apps to see my total liquidity.
- **Acceptance Criteria**:
  - Must display Account Name, Institution, Type (Cash, Stock, Crypto), and Balance.
  - Accounts must be grouped by Type (e.g., "Cash & Equivalents" vs. "Equity Assets").

### US-ACC-002: Integrated Cross-Currency Services (Display & Metrics)

- **Story**: As an Asset Allocator, I want every account balance (USD, Crypto, Gold) to be processed through a centralized "Cross-Currency Service," so that I can see my Net Worth in a single normalized currency (VND) with transparent exchange-rate sourcing.
- **Acceptance Criteria**:
  - **Normalization**: Every multi-currency value must be normalized to "Base Currency" (VND) for sum/aggregate calculations.
  - **Dynamic Toggle**: UI should allow toggling the "View Currency" (e.g., viewing entire portfolio as USD vs VND).
  - **Service-Driven**: Must use a centralized calculator that fetches spot rates (Fmarket/Binance/Commercial Bank) and applies them to both Dashboard metrics and Sheet-based calculations.

---

## 3. Allocation Strategy

### US-ACC-003: High-Visibility Asset Allocation Chart

- **Story**: As an Asset Allocator, I want a pie chart showing my portfolio distribution by asset class, so that I can see if I am over-concentrated in one area (e.g., 90% in Stocks).
- **Acceptance Criteria**:
  - Chart must represent current balances from all "Connected" accounts.
  - Hovering over a slice must show the percentage and absolute VND value.

### US-ACC-004: Liquidity Tiering Review

- **Story**: As an Asset Allocator, I want to see my assets categorized by "Liquidity" (e.g., Immediate vs. T+2 vs. Illiquid), so that I know how much cash I can deploy for a sudden investment opportunity.
- **Acceptance Criteria**:
  - System must categorize Bank/Cash as "Immediate" and Stocks as "T+2" by default.
  - User should be able to override liquidity tags in Account Settings.

---

## 4. History and Trends

### US-ACC-005: Individual Account Balance History

- **Story**: As an Asset Allocator, I want to click any account and see its balance trend over the last 12 months, so that I can see if a specific holding is growing or stagnating.
- **Acceptance Criteria**:
  - Trend line must reflect monthly snapshots from the ledger history.
  - Chart must handle missing months gracefully (e.g., straight line integration).
