# Specification: Financial Ledger & Budgetary Control

**Author**: Product Owner / Financial BA  
**Intended Audience**: Development (Full-Stack) & Data Ops  
**Status**: Mid-Stage Implementation  
**Keywords**: Transaction Ledger, Budget Control, Reconciliation, Double-Entry Validation

---

## 1. Product Intent

To provide a robust, searchable, and categorizable history of every VND/USD movement while allowing the user to define their financial "Safe Zones" through monthly spending limits.

---

## 2. Functional Specification (The Ledger)

**BA Requirement**: The Ledger is the **Heartbeat** of the system. Accuracy is paramount.

### 2.1 Unified Transaction Input (Quick Entry)

- **Feature**: Fast-entry modal for logging daily movements.
- **Rules**:
  - Requires **Account**, **Category**, **Date**, **Amount**.
  - Auto-shards to correct `Transactions_YYYY` tab.
- **AC**: Successful write updates the "Financial Health" (Status: "Success") in < 1.5s.

### 2.2 Intelligent Categorization (Automation)

- **Feature**: Suggest categories based on historical payee patterns.
- **AC**: Must allow manual override if the AI/Regex suggests incorrectly.

---

## 3. Account Lifecycle Management

- **Objective**: Consolidate Bank, Crypto, and Securities into a single location tracker.
- **Feature**: Unified asset allocation analysis (Pie chart of wealth distribution).
- **AC**: Must clearly show "Liquidity Tiering" (e.g., Immediately Liquid vs. Invested).

---

## 4. Budgetary Ceilings (Tactical Limits)

- **Objective**: Prevent month-end financial shocks through real-time progress bars.
- **Logic**: Compare `Transactions_YYYY` actuals against `Budget_YYYY` category targets.
- **Visual**: Color-coded progress bars (Green < 70%, Yellow 70-90%, Red > 90%).
- **AC**: "The Wall" (Historical Comparison bar chart) must show at least 6 months of historical variance.

---

## 5. Technical Requirements for Engineering

- **Data Integrity**: Implement ACID-like checks for every write to prevent row corruption.
- **Search Latency**: Filtered search results (by Vendor, Date, Account) must resolve in < 500ms using local SWR cache.
- **Pagination**: Infinite scroll implementation for Transaction lists.

---

## 6. Engineering Success Criteria

- **Accuracy**: 100% parity with "Manual Sheet Totals."
- **Efficiency**: Reduces user "Logged Entry" friction to under 15 seconds per transaction.
- **Scalability**: Seamlessly "Union" multiple yearly tabs into a single UI ledger view.
