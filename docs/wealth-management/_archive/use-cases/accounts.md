# Use Case Specification: Portfolio Accounts (Asset Distribution)

## UC-ACC-001: Aggregated Balance Synchronization

### 1. Summary

The user refreshes the Accounts view to get the latest balances across all financial entities.

### 2. Actors

- **Primary Actor**: Asset Allocator (User)
- **Secondary Actor**: Accounts API, Fmarket API (for USD rates), Google Sheets.

### 3. Basic Flow

1.  User navigates to the "Portfolio Accounts" page.
2.  System parallel-fetches "Account Metadata" and "Transactions" for the current period.
3.  System calculates current balances for each account (Starting Balance + Σ Transactions).
4.  System identifies any **USD** accounts.
5.  System queries the `Fmarket Proxy API` for the latest bank exchange rate.
6.  System converts all accounts to a common **Base Currency** (e.g., VND).
7.  System renders the "Consolidated Balance Grid" with all accounts categorized by Type.

### 4. Alternative/Exception Flows

- **Stale Rate (E1)**: If Fmarket is down, system uses the last cached USD/VND rate or the institutional fallback (25,450 VND).
- **Missing Reconciliation**: If a balance doesn't match the user's manual expectations, user can trigger "Manual Sync" to adjust starting totals.

---

## UC-ACC-002: Risk Allocation Analysis (Pie Chart Drill-down)

### 1. Summary

User audits their risk exposure by class (e.g., over-concentration in Stocks vs. Bonds).

### 2. Actors

- **Primary Actor**: Asset Allocator (User)

### 3. Basic Flow

1.  User clicks the "Allocation" tab on the Accounts page.
2.  System renders a Pie/Radar Chart showing current asset class distribution.
3.  User hovers over a slice (e.g., "Equity").
4.  System displays: "Total Value: 2.5B VND (75% of Portfolio)."
5.  User clicks the slice to "Drill Down."
6.  System filters the accounts list below to show only "Equity-related" accounts.

### 4. Postconditions

- User has a precise understanding of their sector-level exposure across all bank and securities accounts.

---

## UC-ACC-003: Liquidity Readiness Audit (T-plus check)

### 1. Summary

User checks how much cash they could deploy within the next 24 hours.

### 2. Actors

- **Primary Actor**: Asset Allocator (User)

### 3. Preconditions

- Accounts have "Liquidity Tags" assigned (e.g., Immediate, T+2).

### 4. Basic Flow

1.  User selects the "Liquidity View" from the Accounts toolbar.
2.  System groups all accounts into "Immediate," "T+1," "T+2," and "Restricted."
3.  User scans the "Immediate" total (Expected: 250M VND).
4.  User identifies that "T+2" (Stocks) currently holds 2B VND.
5.  User decides if they need to sell some Stocks to increase "Immediate" liquidity.

### 5. Postconditions

- User has established their "Firepower" for new market opportunities.
