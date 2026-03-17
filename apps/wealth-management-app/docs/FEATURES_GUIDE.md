# Feature Guide

This document describes the core features available in the Wealth Management System.

## 1. Unified Dashboard
The command center for your finances.
- **Financial Health Score**: AI-powered assessment (0-100) based on your income, expenses, and debt levels.
- **Net Worth Card**: Real-time calculation of Assets minus Liabilities.
- **Spending Chart**: Monthly breakdown of payments vs. deposits.
- **Budget Overview**: Visual progress bars showing spending against category limits.
- **Recent Transactions**: Quick view of latest financial activity.

## 2. Transaction Management
Full control over your daily spending.
- **Categorized Search**: Fast lookup of transactions by category, payee, or amount.
- **Add Transaction**: Simple form to log new expenses or income.
- **Searchable Categories**: Chip-based category selection for easy entry.
- **Sync to Sheets**: Every transaction is persisted directly to your Google Sheet.

## 3. Loan & Debt Tracker
Dedicated tracking for external borrowing and lending.
- **Lender Details**: Track who you owe or who owes you.
- **Monthly Repayment Progress**: Visual progress bars for current month's installments.
- **Yearly Balance tracking**: Full view of total remaining debt for the year.
- **Summary Cards**: Aggregated view of total monthly debt and yearly remaining balance.

## 4. AI Financial Advisor (Chat)
Interactive natural language interface for your financial data.
- **Multi-Model Support**: Switch between GPT-4o, Claude, or Gemini.
- **Context-Aware**: AI can "see" your accounts, transactions, and loans to answer specific questions.
- **Actionable Advice**: Get personalized recommendations on how to save more or pay down debt.

## 5. Account Summary
Consolidated view of all linked financial accounts.
- **Live Balances**: Fetched directly from Google Sheets.
- **Account Types**: Grouping by Bank, Credit Card, or Cash.
- **Net Position**: Instant view of your total liquidity across all banks.

## 6. Google Sheets Integration
The system uses Google Sheets as a "Headless Database".
- **Source of Truth**: All data lives in your sheet.
- **Bidirectional**: Updates in the sheet reflect in the app, and vice versa.
- **Local Cache**: High-performance in-memory caching to ensure sub-second page loads despite network latency.
