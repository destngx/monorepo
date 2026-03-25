# User Flows

This document outlines the primary user interactions and data movements within the Wealth Management System.

## 1. Dashboard Loading & Data Sync

The dashboard is the central hub. It fetches all financial data in parallel to provide a holistic view.

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Server as Next.js Server
    participant Cache as In-memory Cache
    participant Sheets as Google Sheets API

    User->>Browser: Opens Dashboard
    Browser->>Server: Request / (Page Load)
    Server->>Server: Promise.all([getAccounts, getBudget, getTransactions, getLoans])

    rect rgb(240, 240, 240)
        Note over Server, Sheets: Data Fetching Pattern
        Server->>Cache: Check Cache (e.g., 'accounts')
        alt Cache Hit
            Cache-->>Server: Return Cached Data
        else Cache Miss
            Server->>Sheets: Read Sheet Range
            Sheets-->>Server: Raw Data
            Server->>Server: Map to Business Types
            Server->>Cache: Set Cache (5 min TTL)
        end
    end

    Server-->>Browser: Rendered Dashboard (RSC)
    Browser->>User: Display Net Worth, Accounts, AI Health
```

## 2. AI Financial Health Assessment

The health card uses AI to analyze current state vs. recent performance.

```mermaid
sequenceDiagram
    participant User
    participant Card as FinancialHealthCard (Client)
    participant API as /api/ai/financial-health
    participant LLM as AI Model (GPT-4o)

    Card->>API: POST { accounts, transactions, loans }
    API->>API: Calculate Net Worth & Liabilities (Assets - Debts)
    API->>LLM: Send Prompt with Financial Context
    LLM-->>API: Analysis & Health Signals (JSON)
    API-->>Card: Return Score & Signals
    Card->>User: Display Health Score & Recommendations
```

## 3. Adding a Transaction

Adding a transaction triggers a direct write to Google Sheets and a cache invalidation.

```mermaid
sequenceDiagram
    participant User
    participant Form as TransactionForm
    participant API as /api/transactions (POST)
    participant Sheets as Google Sheets API

    User->>Form: Enter Payee, Amount, Date
    Form->>API: Submit Data
    API->>Sheets: Append Row to 'Transactions' Sheet
    Sheets-->>API: Success
    API->>API: Invalidate Local Cache
    API-->>Form: Success Toast
    Form->>User: Update UI
```

## 4. Loan & Debt Tracking

Tracking external liabilities and repayment progress.

```mermaid
sequenceDiagram
    participant User
    participant Page as LoansPage
    participant API as /api/loans
    participant Sheets as Google Sheets API

    User->>Page: Navigate to /loans
    Page->>API: getLoans()
    API->>Sheets: Read 'Loan / Dept'!A10:H50
    Sheets-->>API: Return rows
    API->>API: Map to Loan[] interface
    API-->>Page: Return Loans
    Page->>User: Display Summary Cards & Detailed Table
```
