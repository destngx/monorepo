# Requirements

This document outlines the functional and non-functional requirements for the Wealth Management System.

## 1. Functional Requirements

### 1.1 Data Source & Integrity

- **Single Source of Truth**: All financial data (accounts, transactions, budget, loans) must reside in Google Sheets.
- **Bidirectional Sync**: The application must reflect changes made in the sheet and update the sheet when changes are made in the app.
- **Parallel Fetching**: The system should fetch multiple data entities (Accounts, Budget, Loans, etc.) in parallel to minimize load times.

### 1.2 Dashboard & Visualization

- **Net Worth Calculation**: Automatically calculate total assets minus total liabilities (including external loans).
- **Financial Health**: Provide an AI-driven "Health Score" based on current financial position and recent throughput.
- **Reporting**: Display spending trends and budget adherence through interactive charts.

### 1.3 Transaction Management

- **Entry**: Allow users to add bank/cash transactions with payees, amounts, and dates.
- **Categorization**: Provide a searchable category list. Support AI-based categorization suggestions.
- **Search**: Users must be able to search transactions by text or filter by category.

### 1.4 Loan & Debt Tracker

- **Asset/Liability Logic**: Distinguish between money owed (Loan) and money borrowed (Debt).
- **Progress Tracking**: Display monthly repayment progress against yearly totals.
- **Aggregated View**: Show total monthly obligations in summary cards.

### 1.5 AI Chat

- **Financial Context**: The AI must have access to real-time account balances and transaction history.
- **Natural Language Query**: Users can ask questions like "How much did I spend on food?" or "Is my debt-to-income ratio safe?".
- **Model Agnostic**: Support switching between various LLM providers (GPT, Claude, Gemini).

---

## 2. Non-Functional Requirements

### 2.1 Performance

- **Loading Time**: Main dashboard should be interactive in under 2 seconds.
- **Concurrency**: Support at least 5 parallel requests to Google Sheets API without rate-limiting issues for a single user.

### 2.2 Security & Privacy

- **Stateless Core**: No persistent storage of financial PII in a local database (using In-memory cache + Google Sheets).
- **Environment Safety**: Secrets (API Keys) must never be exposed to the client.

### 2.3 Reliability

- **Offline Resilience**: Use TTL-based in-memory caching to allow partial functionality if the Google Sheets API is temporarily unreachable.
- **Error Handling**: Graceful degradation when AI services or external APIs are down.

### 2.4 Scalability

- **Headerless Design**: Architecture should support adding new financial sheets (e.g., Investments, Goals) without major refactors of the core sync engine.
