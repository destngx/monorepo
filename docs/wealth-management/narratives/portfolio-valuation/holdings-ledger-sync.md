# Details: Holdings & Ledger Synchronization

## 1. Feature Meaning: The Source of Truth

This represents the **"Static History"** of the user's wealth.

## 2. API Source & Logic

- **Primary Source**: `GET /api/accounts` and `GET /api/transactions`.
- **Sync Logic**: These APIs read from the Google Sheet rows that record individual "Stock Purchases" (Payee: `FPT`) or "Dividends" (Category: `Investment Income`).

## 3. Business Use Cases (Actionable)

- **Account Aggregation**: Combines "Multiple Cash/Savings accounts" with "Security account ledger" values.
- **Cost Basis Analysis**: Determining the initial "Price Paid" from the transaction history versus the "Current Market Value."

## 4. Why This Hub?

**Accounts/Transactions** are the base for the **Asset Ledger Dashboard**. They are the starting point for all "Wealth Visualization" in the application.
