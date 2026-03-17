# Implementation Plan

## Wealth Management System — Step-by-Step Build Guide

**Version:** 1.0
**Last Updated:** 2026-02-28

---

## Overview

This document breaks Phase 1 into **13 implementation sprints**, each producing a working increment. Estimated total: **70–100 hours** for a developer familiar with the stack.

---

## Sprint 0: Project Scaffolding (2–3 hours)

### 0.1 Initialize Project

```bash
pnpm create next-app@latest wealth-management --typescript --tailwind --eslint --app --src-dir
cd wealth-management
```

### 0.2 Install Dependencies

```bash
# UI
pnpm add @radix-ui/react-slot class-variance-authority clsx tailwind-merge lucide-react

# shadcn/ui (init + components)
pnpm dlx shadcn@latest init
pnpm dlx shadcn@latest add button card input table dialog select badge progress sheet tabs separator avatar scroll-area dropdown-menu tooltip

# AI
pnpm add ai @ai-sdk/openai @ai-sdk/google @ai-sdk/anthropic

# Google Sheets
pnpm add googleapis google-auth-library

# Database
pnpm add @prisma/client
pnpm add -D prisma

# Charts
pnpm add recharts

# Utilities
pnpm add zod date-fns swr react-hook-form @hookform/resolvers
```

### 0.3 Configure Prisma

- Create `prisma/schema.prisma` (see ARCHITECTURE.md Section 4)
- Run `pnpm prisma db push` to create SQLite database
- Create `src/lib/db/prisma.ts` singleton

### 0.4 Environment Setup

- Create `.env.example` with all required variables
- Create `.env.local` with actual values
- Add `.env.local` and `prisma/dev.db` to `.gitignore`

### 0.5 Project Configuration

- Configure `tsconfig.json` path aliases (`@/`)
- Configure `tailwind.config.ts` with shadcn theme
- Configure `next.config.ts` (environment variables, image domains)
- Set up ESLint config

### 0.6 Git Init

```bash
git init
git add .
git commit -m "chore: initial project scaffolding"
```

**Deliverable:** Empty Next.js app runs with `pnpm dev`, Prisma connected, all deps installed.

---

## Sprint 1: Layout & Navigation (3–4 hours)

### 1.1 Root Layout

- Create `src/app/layout.tsx` with:
  - Global providers (SWR, Theme)
  - Sidebar navigation
  - Top header with sync status indicator
  - Main content area with padding

### 1.2 Sidebar Component

- `src/components/layout/sidebar.tsx`
- Navigation links:
  - 🏠 Dashboard (`/`)
  - 💳 Transactions (`/transactions`)
  - 📊 Budget (`/budget`)
  - 🏦 Accounts (`/accounts`)
  - 💬 AI Chat (`/chat`)
  - ⚙️ Settings (`/settings`)
- Active route highlighting
- Collapsible on mobile

### 1.3 Header Component

- `src/components/layout/header.tsx`
- Page title (dynamic based on route)
- Sync status indicator (last synced timestamp)
- Manual sync button
- Dark mode toggle (optional, nice to have)

### 1.4 Placeholder Pages

- Create all page.tsx files with placeholder content
- Verify navigation between all pages works

**Deliverable:** App with working navigation, sidebar, and all placeholder pages.

---

## Sprint 2: Google Sheets Integration (6–8 hours)

### 2.1 Authentication

- `src/lib/sheets/auth.ts`
- Google OAuth 2.0 setup (Service Account for simplicity, or OAuth for user-facing)
  - **Recommendation:** Start with Service Account (share sheet with service account email)
  - Phase 2: Migrate to OAuth for multi-user or user-specific sheets
- Token management

### 2.2 Sheets Client

- `src/lib/sheets/client.ts`
- Generic Google Sheets read/write functions:
  - `readSheet(sheetName, range)` → `string[][]`
  - `appendRow(sheetName, values)` → success/fail
  - `updateRow(sheetName, rowIndex, values)` → success/fail
  - `deleteRow(sheetName, rowIndex)` → success/fail
- Error handling (rate limits, auth failures, sheet not found)

### 2.3 Data Mappers

- `src/lib/sheets/mappers.ts`
- **Accounts mapper:** Sheet row → `Account` type
  - Map: ACCOUNTS → name, Date to pay → dueDate, Goal → goalAmount, % → goalProgress, Cleared → clearedBalance, Balance → balance, Type → type, Note → note
- **Transactions mapper:** Sheet row → `Transaction` type
  - Map: Account → accountName, Date → date, Num → referenceNumber, Payee → payee, Tag → tags, Memo → memo, Category → category, Clr → cleared, PAYMENT → payment, DEPOSIT → deposit, Account Balance → accountBalance, Cleared Balance → clearedBalance, BALANCE → balance
- **Budget mapper:** Sheet row → `BudgetItem` type
- Reverse mappers (Type → Sheet row) for writes

### 2.4 Domain Services

- `src/lib/sheets/accounts.ts` — `getAccounts()`, `getAccountByName()`
- `src/lib/sheets/transactions.ts` — `getTransactions(filters)`, `addTransaction()`, `updateTransaction()`, `deleteTransaction()`
- `src/lib/sheets/budget.ts` — `getBudget()`, `updateBudgetItem()`

### 2.5 Caching Layer

- `src/lib/db/cache.ts`
- Cache Google Sheets responses in SQLite
- TTL: 5 minutes default
- Manual invalidation on writes
- `getCached(key)`, `setCache(key, data, ttl)`, `invalidateCache(key)`

### 2.6 API Routes

- `src/app/api/accounts/route.ts` — GET (list accounts)
- `src/app/api/transactions/route.ts` — GET (list with filters), POST (create)
- `src/app/api/budget/route.ts` — GET (list budget items), PUT (update)
- `src/app/api/sync/route.ts` — POST (force cache invalidation + re-fetch)

### 2.7 Testing

- Verify read from each sheet tab
- Verify write (add transaction) and confirm it appears in Google Sheet
- Verify cache hit/miss behavior
- Verify error handling (wrong sheet ID, no permissions)

**Deliverable:** All Google Sheets data accessible via API routes. CRUD working end-to-end.

---

## Sprint 3: TypeScript Types & Constants (2–3 hours)

### 3.1 Type Definitions

- `src/lib/types/account.ts`
  ```typescript
  interface Account {
    name: string;
    dueDate: string | null;
    goalAmount: number | null;
    goalProgress: number | null;
    clearedBalance: number;
    balance: number;
    type: AccountType;
    currency: Currency;
    note: string | null;
  }
  ```
- `src/lib/types/transaction.ts`
  ```typescript
  interface Transaction {
    id: string;             // Row index as ID
    accountName: string;
    date: Date;
    referenceNumber: string | null;
    payee: string;
    tags: string[];
    memo: string | null;
    category: string;
    cleared: boolean;
    payment: number | null;  // Debit
    deposit: number | null;  // Credit
    accountBalance: number;
    clearedBalance: number;
    runningBalance: number;
  }
  ```
- `src/lib/types/budget.ts`
- `src/lib/types/category.ts`
- `src/lib/types/settings.ts`

### 3.2 Category Constants

- `src/lib/constants/categories.ts`
- Full taxonomy from PRD Section 5
- Each category: id, name, icon, subcategories, keywords (for AI matching)

### 3.3 Validation Schemas

- `src/lib/utils/validation.ts`
- Zod schemas for all API inputs
- Transaction creation schema
- Budget update schema
- Settings update schema

### 3.4 Utility Functions

- `src/lib/utils/currency.ts` — VND/USDT formatting, conversion
- `src/lib/utils/date.ts` — Date formatting, parsing, ranges

**Deliverable:** Full type system, validation, and utility functions ready for use.

---

## Sprint 4: Dashboard (6–8 hours)

### 4.1 Exchange Rate Service

- `src/lib/services/exchange-rate-service.ts`
- Fetch USDT/VND from CoinGecko free API
- Cache for 15 minutes
- Fallback to last known rate if API fails

### 4.2 Dashboard Page (Server Component)

- `src/app/page.tsx`
- Fetch: accounts, budget summary, recent transactions, exchange rate
- Loading skeleton (`src/app/loading.tsx`)

### 4.3 Net Worth Card

- `src/components/dashboard/net-worth-card.tsx`
- Total net worth in VND (sum all accounts, convert USDT → VND)
- Breakdown: VND accounts total, USDT accounts total (in both USDT and VND)
- Change indicator (vs. last month, if data available)

### 4.4 Accounts Summary

- `src/components/dashboard/accounts-summary.tsx`
- Grid of account cards
- Each card: name, type icon, balance, currency
- Goal progress bar (if goal set)

### 4.5 Spending Chart

- `src/components/dashboard/spending-chart.tsx`
- Monthly spending bar chart (last 6 months)
- Income vs. Expenses overlay
- Clickable bars → navigate to transaction list for that month

### 4.6 Budget Overview

- `src/components/dashboard/budget-overview.tsx`
- Top 5 budget categories with progress bars
- Color coding: green (<80%), yellow (80–100%), red (>100%)
- "View All" link to budget page

### 4.7 Recent Transactions

- `src/components/dashboard/recent-transactions.tsx`
- Last 10 transactions in a compact table
- Category badge, amount (green for deposit, red for payment)
- "View All" link to transactions page

### 4.8 Quick Stats

- `src/components/dashboard/quick-stats.tsx`
- This month: Total income, Total expenses, Net (income - expenses), Savings rate %

**Deliverable:** Fully functional dashboard with real data from Google Sheets.

---

## Sprint 5: Transactions Page (6–8 hours)

### 5.1 Transaction Table

- `src/components/transactions/transaction-table.tsx`
- Columns: Date, Payee, Category, Account, Amount (Payment/Deposit), Balance
- Sortable columns (date, amount)
- Pagination (50 per page)
- Row click → edit dialog

### 5.2 Transaction Filters

- `src/components/transactions/transaction-filters.tsx`
- Date range picker (from/to)
- Account selector (dropdown)
- Category selector (dropdown)
- Type filter (Payment/Deposit/All)
- Search box (searches payee + memo)
- Active filter badges with clear button

### 5.3 Transaction Form

- `src/components/transactions/transaction-form.tsx`
- Create/Edit mode
- Fields: Date, Account, Payee, Memo, Category, Amount, Type (Payment/Deposit)
- Category dropdown with AI suggestion button ("✨ Suggest Category")
- Form validation with Zod
- Submit → API → Google Sheets

### 5.4 Category Badge

- `src/components/transactions/category-badge.tsx`
- Colored badge per category
- Icon + label
- Click to edit category

### 5.5 Transactions Page

- `src/app/transactions/page.tsx`
- Filters + Table + "Add Transaction" button
- Data fetching with SWR
- Loading skeleton

### 5.6 Bulk Actions

- Select multiple transactions (checkboxes)
- Bulk re-categorize (AI suggests categories for selected)
- Bulk delete (with confirmation)

**Deliverable:** Full transaction management with filtering, CRUD, and category suggestions.

---

## Sprint 6: Budget Page (4–5 hours)

### 6.1 Budget Table

- `src/components/budget/budget-table.tsx`
- Columns: Category, Monthly Limit, Yearly Limit, Spent (Month), Spent (Year), Remaining
- Editable cells (click to edit limit)
- Color-coded remaining (green/yellow/red)

### 6.2 Budget Progress

- `src/components/budget/budget-progress.tsx`
- Visual progress bars per category
- Monthly view (default) with toggle to yearly
- Sorted by spending % (highest first)

### 6.3 Budget vs. Actual Chart

- `src/components/budget/budget-vs-actual.tsx`
- Bar chart: Budget limit vs. Actual spending per category
- Side-by-side comparison

### 6.4 Category Editor

- `src/components/budget/category-editor.tsx`
- Add new category (with AI suggestion)
- Edit category name
- Merge categories
- Set monthly/yearly limits
- Reorder categories

### 6.5 Budget Page

- `src/app/budget/page.tsx`
- Summary cards (total budget, total spent, remaining)
- Budget progress component
- Budget table component
- "AI Budget Suggestions" button → AI analyzes spending and suggests adjustments

**Deliverable:** Budget management with visual progress tracking and AI suggestions.

---

## Sprint 7: AI Integration (8–10 hours)

### 7.1 Provider Configuration

- `src/lib/ai/providers.ts`
- Factory pattern for provider switching:
  ```typescript
  function getAIProvider(provider: 'openai' | 'google' | 'anthropic') {
    switch (provider) {
      case 'openai': return openai('gpt-4o');
      case 'google': return google('gemini-2.0-flash');
      case 'anthropic': return anthropic('claude-sonnet-4-20250514');
    }
  }
  ```
- Settings stored in SQLite (default provider, model preferences)

### 7.2 System Prompt

- `src/lib/ai/system-prompt.ts`
- Dynamic system prompt builder that includes:
  - User's financial context (account summary, current month budget status)
  - Available tools and their descriptions
  - Vietnamese Dong formatting rules
  - Current date/time
  - Category taxonomy
  - Conversation style (concise, data-driven, Vietnamese context-aware)

### 7.3 AI Tools

- `src/lib/ai/tools.ts`
- Tool definitions for Vercel AI SDK:

| Tool | Description | Parameters |
|---|---|---|
| `getAccountBalances` | Fetch all account balances | none |
| `getTransactions` | Fetch transactions with filters | dateFrom, dateTo, account, category, limit |
| `getBudgetStatus` | Get budget vs. actual for current month | month? |
| `getSpendingByCategory` | Spending breakdown by category | dateFrom, dateTo |
| `getExchangeRate` | Current USDT/VND rate | none |
| `categorizeTransaction` | AI-categorize a transaction | payee, memo, amount |
| `getMonthlyTrend` | Monthly income/expense totals | months (number of months) |
| `searchTransactions` | Search transactions by text | query |

### 7.4 Chat API Route

- `src/app/api/chat/route.ts`
- Streaming response using Vercel AI SDK `streamText()`
- Tool execution with automatic tool result handling
- Chat history persistence to SQLite
- Rate limiting (max 30 requests/minute)

### 7.5 Transaction Categorizer

- `src/lib/ai/categorizer.ts`
- Dedicated categorization function:
  - Input: payee, memo, amount, account
  - Output: category, confidence score, reasoning
- Batch categorization support
- Learning from corrections (query CategoryCorrection table)
- Prompt includes category taxonomy + past corrections for same payee

### 7.6 Chat Interface

- `src/components/chat/chat-interface.tsx`
- Uses Vercel AI SDK `useChat` hook
- Message list with auto-scroll
- Input with send button
- Tool call result display (formatted tables/charts inline)
- Loading indicator during AI response
- Provider selector in header

### 7.7 Chat Page

- `src/app/chat/page.tsx`
- Chat sessions sidebar (history)
- Main chat area
- New chat button
- Provider selector

### 7.8 Settings Page

- `src/app/settings/page.tsx`
- AI Provider selection (OpenAI / Gemini / Anthropic)
- Model selection per provider
- API key status indicators (configured/missing)
- Google Sheets connection status
- Sheet ID configuration
- Default currency display preference

**Deliverable:** Full AI chat with tool calling, auto-categorization, and provider switching.

---

## Sprint 8: Accounts Page & Polish (4–5 hours)

### 8.1 Accounts Page

- `src/app/accounts/page.tsx`
- Account cards with detailed info
- Balance history chart (if data available)
- Goal tracking with progress visualization
- Due date indicators for bills

### 8.2 Polish & UX

- Loading skeletons for all pages
- Error boundaries with retry buttons
- Empty states ("No transactions this month")
- Toast notifications for actions (sync complete, transaction added)
- Keyboard shortcuts (N for new transaction, / for search)
- Responsive design verification (mobile/tablet)

### 8.3 Dark Mode

- Theme toggle in header
- Tailwind dark mode classes
- Persist preference in settings

### 8.4 Final Testing

- End-to-end flow: Add transaction → auto-categorize → appears in budget → AI chat about it
- Sync reliability (read/write round-trip)
- AI provider switching works for all three providers
- Error handling: disconnect Sheets, invalid API key, rate limit

**Deliverable:** Complete, polished Phase 1 application.

---

## Sprint 9: Market Data & Investment Infrastructure (4–5 hours)

### 9.1 Market Data Service

- `src/lib/services/market-data-service.ts`
- CoinGecko integration for crypto prices (BTC, ETH, USDT, etc.)
- Gold price fetching (SJC gold price from Vietnamese sources or free gold API)
- Stock price fetching (VN30 index, individual stocks from free API like Yahoo Finance)
- Cache prices in SQLite MarketPrice table (15-minute TTL for crypto, 1-hour for gold/stocks)
- Fallback to last known price if API fails
- Rate limiting to stay within free API tiers

### 9.2 Investment Sheet Integration

- `src/lib/sheets/investments.ts`
- Read from new "Investments" Google Sheets tab
- Headers: `Asset Name | Type | Symbol | Quantity | Purchase Price | Purchase Date | Currency | Account | Notes`
- Type enum: crypto, stock, gold, savings_deposit, real_estate, business_equity, other
- Mapper: Sheet row to Investment type
- Write: Add/update/delete investments in sheet

### 9.3 Investment Types

- `src/lib/types/investment.ts`
```typescript
interface Investment {
  id: string;              // Row index
  name: string;            // "Bitcoin", "SJC Gold 1 tael"
  type: InvestmentType;    // crypto, stock, gold, savings_deposit, real_estate, business_equity, other
  symbol: string;          // "BTC", "GOLD_SJC", "VN30", "" for real estate
  quantity: number;        // 0.5 BTC, 2 taels gold, 1 apartment
  purchasePrice: number;   // Cost basis per unit
  purchaseDate: Date;
  currency: Currency;      // VND, USD, USDT
  account: string;         // "Binance", "VCBS", "Physical"
  notes: string | null;
  // Computed fields
  currentPrice?: number;
  currentValueVND?: number;
  pnlVND?: number;
  pnlPercent?: number;
}
```

### 9.4 Portfolio Service

- `src/lib/services/portfolio-service.ts`
- `getPortfolioWithPrices()`: Fetch all investments + current prices → calculate values
- `calculateAllocation()`: Group by asset type → percentage breakdown
- `calculatePnL()`: Per-asset and total P&L (unrealized)
- `getAssetClassBreakdown()`: Cash (from accounts) + investments by type

### 9.5 API Routes

- `src/app/api/investments/route.ts` — GET (list), POST (create)
- `src/app/api/market-data/route.ts` — GET (all prices or specific symbol)

### 9.6 Database Schema

- Add `MarketPrice` model to Prisma schema
- Run `pnpm prisma db push`

**Deliverable:** Investment data accessible from Google Sheets, live prices fetched and cached, portfolio calculations working.

---

## Sprint 10: Investment Portfolio & Allocation Pages (6–8 hours)

### 10.1 Portfolio Overview Component

- `src/components/investments/portfolio-overview.tsx`
- Total portfolio value in VND with P&L indicator
- Grid: total value, total cost basis, total P&L, P&L %

### 10.2 Asset Table

- `src/components/investments/asset-table.tsx`
- Columns: Name, Type, Quantity, Cost Basis, Current Price, Current Value, P&L, P&L%, Allocation%
- Sortable columns
- Color-coded P&L (green positive, red negative)
- Row click to edit

### 10.3 Asset Form

- `src/components/investments/asset-form.tsx`
- Create/Edit mode
- Fields: Name, Type, Symbol (with auto-complete for crypto/stocks), Quantity, Purchase Price, Date, Currency, Account, Notes
- Symbol lookup: when user types "BTC", auto-suggest "Bitcoin (BTC)"
- Form validation with Zod

### 10.4 Allocation Chart

- `src/components/investments/allocation-chart.tsx`
- Donut/pie chart (Recharts) showing asset class %
- Labels: Crypto X%, Cash X%, Gold X%, Savings X%, Stocks X%, Real Estate X%, Other X%
- Click segment to filter asset table

### 10.5 Allocation Targets

- `src/components/investments/allocation-targets.tsx`
- Set target allocation per asset class
- Visual comparison: current vs target (side-by-side bars)
- Deviation indicator: "Crypto is 15% over target"

### 10.6 Price Ticker

- `src/components/investments/price-ticker.tsx`
- Compact display of tracked asset prices
- Auto-refresh every 15 minutes
- Show price change % (24h)

### 10.7 Investments Page

- `src/app/investments/page.tsx`
- Portfolio overview + allocation chart + asset table + "Add Asset" button
- Data fetching with SWR

### 10.8 Dashboard Additions

- `src/components/dashboard/asset-allocation.tsx` — Mini allocation pie on dashboard
- `src/components/dashboard/net-worth-history.tsx` — Net worth trend line chart on dashboard
- Update dashboard page to include these new widgets

**Deliverable:** Full investment portfolio page with asset management, allocation charts, and live prices.

---

## Sprint 11: Net Worth History & Projections (4–5 hours)

### 11.1 Net Worth Snapshot Service

- `src/lib/services/snapshot-service.ts`
- `takeSnapshot()`: Calculate current net worth (all accounts + all investments at current prices) → store in SQLite
- `getSnapshots(months)`: Retrieve historical snapshots
- Schedule: Take snapshot daily (can be triggered on first dashboard load of the day)
- Breakdown stored as JSON: { cash: X, crypto: Y, gold: Z, savings: W, stocks: V, realEstate: U, other: T }

### 11.2 Projection Service

- `src/lib/services/projection-service.ts`
- `projectNetWorth({ years, additionalMonthlySavings, expectedAnnualReturn })`:
  - Calculate current savings rate from last 6 months of transactions
  - Apply compound growth with monthly contributions
  - Return yearly milestones with total and per-asset-class breakdown
- `runWhatIfScenario({ monthlyChange, oneTimeChange, assetAllocationChange, years })`:
  - Clone current portfolio state
  - Apply modifications
  - Project forward
  - Return comparison: baseline vs. scenario

### 11.3 Net Worth History Chart

- `src/components/dashboard/net-worth-history.tsx` (update if already created in Sprint 10)
- Line chart showing net worth over time
- Stacked area variant showing composition (cash, crypto, gold, etc.)
- Toggle: 3M / 6M / 1Y / All time
- Hover tooltip with exact values

### 11.4 Projection Calculator

- `src/components/investments/projection-calculator.tsx`
- Interactive form:
  - Years to project (slider: 1-30)
  - Additional monthly savings (input)
  - Expected annual return % (input with smart default from historical data)
- Real-time chart update showing projected growth curve
- Milestones marked: "1B VND by 2028", "2B VND by 2031"

### 11.5 What-If Scenario Tool

- `src/components/investments/what-if-scenarios.tsx`
- Preset scenarios:
  - "Crypto drops 30%"
  - "I increase savings by 5M/month"
  - "I buy 2 taels of SJC gold"
  - Custom scenario
- Side-by-side comparison chart: baseline vs. scenario
- Impact summary: "This would increase your net worth by X in Y years"

### 11.6 Database Schema

- Add `NetWorthSnapshot` model to Prisma schema
- Run `pnpm prisma db push`

### 11.7 API Routes

- `src/app/api/portfolio/history/route.ts` — GET net worth history
- `src/app/api/projections/route.ts` — POST run projection / what-if

**Deliverable:** Historical net worth tracking, AI projections, and interactive what-if scenarios.

---

## Sprint 12: Financial Goals & Health Score (5–6 hours)

### 12.1 Goal Service

- `src/lib/services/goal-service.ts`
- `createGoal()`, `updateGoal()`, `deleteGoal()` — CRUD to SQLite
- `calculateProgress()`: Current amount / target amount, estimated completion date
- `suggestSavingsPlan()`: "Save X/month to reach goal by target date"
- `prioritizeGoals()`: Rank by urgency (closest deadline first) and impact

### 12.2 Health Score Service

- `src/lib/services/health-service.ts`
- Calculate composite score (0-100) from weighted metrics:
  - Savings Rate (25%): score based on % of income saved (20%+ = 100)
  - Emergency Fund (20%): months of expenses covered (6+ months = 100)
  - Diversification (15%): how balanced across asset classes vs. targets
  - Budget Adherence (15%): % of budget categories on track
  - Net Worth Growth (15%): YoY growth rate
  - Debt Ratio (10%): debt / income (lower = better)
- Generate AI action items based on lowest-scoring metrics

### 12.3 Goal Card Component

- `src/components/goals/goal-card.tsx`
- Goal name, type icon, progress bar, current/target amounts
- Estimated completion date
- Status badge: on track, behind, ahead, completed

### 12.4 Goal Form

- `src/components/goals/goal-form.tsx`
- Fields: Name, Type (dropdown), Target Amount, Target Date, Linked Account, Priority, Notes
- Validation with Zod

### 12.5 Goals Page

- `src/app/goals/page.tsx`
- Grid of goal cards, sorted by priority
- "Add Goal" button
- Summary: total goals, completed, in progress
- AI suggestion: "Based on your finances, focus on [goal] first"

### 12.6 Health Score Page

- `src/app/health/page.tsx`
- Large circular score display (0-100) with color (red/yellow/green)
- Metric breakdown cards (6 metrics with individual scores + explanations)
- AI-generated action items list (top 3-5 improvements)
- Historical health score chart (if enough snapshots)

### 12.7 Dashboard Additions

- `src/components/dashboard/goals-overview.tsx` — Top 3 goals with progress bars
- `src/components/dashboard/passive-income.tsx` — Monthly passive income summary
- Update dashboard to include these

### 12.8 Database Schema

- Add `FinancialGoal` model to Prisma schema
- Run `pnpm prisma db push`

### 12.9 API Routes

- `src/app/api/goals/route.ts` — GET, POST
- `src/app/api/goals/[id]/route.ts` — PUT, DELETE
- `src/app/api/health/route.ts` — GET

**Deliverable:** Financial goals with AI-powered planning, health score with actionable insights.

---

## Sprint 13: OAuth & Wealth AI Tools (5–6 hours)

### 13.1 OAuth Infrastructure

- `src/lib/auth/oauth-config.ts` — Provider OAuth configurations
- `src/lib/auth/token-store.ts` — Encrypted token storage (SQLite OAuthToken table)
- `src/lib/auth/providers/openai.ts` — OpenAI OAuth flow (PKCE)
- `src/lib/auth/providers/google.ts` — Google OAuth flow (shared scopes: Sheets + Gemini)

### 13.2 OAuth API Routes

- `src/app/api/auth/[provider]/route.ts` — Initiate OAuth (redirect to provider)
- `src/app/api/auth/callback/route.ts` — Handle OAuth callback
- `src/app/api/auth/status/route.ts` — Check connection status per provider

### 13.3 Updated AI Provider Config

- Update `src/lib/ai/providers.ts` to check OAuth tokens before API keys
- Token refresh logic: check expiration, refresh if needed, fallback to API key

### 13.4 Wealth AI Tools

- Add all new tools to `src/lib/ai/tools.ts`:
  - getPortfolioSummary, getAssetPrice, getNetWorthHistory
  - getGoalProgress, getFinancialHealthScore
  - projectNetWorth, suggestRebalancing
  - getPassiveIncome, runWhatIfScenario
- Update system prompt to include wealth advisor context

### 13.5 Settings Page Update

- Update `src/app/settings/page.tsx`:
  - OAuth connect/disconnect buttons per provider
  - OAuth status indicators (connected as user@email)
  - Auth mode selector: API Key vs OAuth per provider

### 13.6 Database Schema

- Add `OAuthToken` model to Prisma schema
- Run `pnpm prisma db push`

### 13.7 Testing

- Test OAuth flow for OpenAI
- Test OAuth flow for Google (verify both Sheets and Gemini work)
- Test AI wealth tools with real data
- Test fallback: OAuth → API key
- Verify all wealth AI example conversations work

**Deliverable:** OAuth authentication for AI providers, all wealth AI tools integrated into chat.
---

## Summary Timeline

| Sprint | Feature | Estimated Hours | Cumulative |
|---|---|---|---|
| 0 | Project Scaffolding | 2–3h | 2–3h |
| 1 | Layout & Navigation | 3–4h | 5–7h |
| 2 | Google Sheets Integration | 6–8h | 11–15h |
| 3 | Types & Constants | 2–3h | 13–18h |
| 4 | Dashboard | 6–8h | 19–26h |
| 5 | Transactions Page | 6–8h | 25–34h |
| 6 | Budget Page | 4–5h | 29–39h |
| 7 | AI Integration | 8–10h | 37–49h |
| 8 | Accounts & Polish | 4–5h | 41–54h |
| 9 | Market Data & Investment Infrastructure | 4–5h | 45–59h |
| 10 | Investment Portfolio & Allocation Pages | 6–8h | 51–67h |
| 11 | Net Worth History & Projections | 4–5h | 55–72h |
| 12 | Financial Goals & Health Score | 5–6h | 60–78h |
| 13 | OAuth & Wealth AI Tools | 5–6h | **65–84h** |

---

## Build Order Rationale

1. **Scaffolding first** — Everything depends on project setup
2. **Layout early** — Navigation is needed for all subsequent pages
3. **Google Sheets before UI** — Data layer must work before building views
4. **Types alongside Sheets** — Strong typing prevents bugs in data transformation
5. **Dashboard first page** — Most value, proves the data pipeline works
6. **Transactions before Budget** — Budget depends on transaction data (spent amounts)
7. **AI late** — Needs all data access layers to be working for tool calls
8. **Polish last** — UX refinement after core features are solid
9. **Investments infrastructure before UI** — Market data and portfolio service needed before building investment pages
10. **Portfolio pages after infrastructure** — Visual layer built on working data layer
11. **Projections after portfolio** — Needs historical data and current portfolio for accurate projections
12. **Goals & Health parallel with projections** — Independent features, can be built simultaneously
13. **OAuth & Wealth AI last** — Needs all services built to wire tools, OAuth is additive enhancement

---

## Dependencies Between Sprints

```
Sprint 0 (Scaffolding)
    |
Sprint 1 (Layout)
    |
Sprint 2 (Google Sheets) <-> Sprint 3 (Types) [parallel]
    |
Sprint 4 (Dashboard)
    |
Sprint 5 (Transactions)
    |
Sprint 6 (Budget)
    |
Sprint 7 (AI Integration)
    |
Sprint 8 (Polish)
    |
Sprint 9 (Market Data & Investments Infrastructure)
    |
Sprint 10 (Investment Portfolio & Allocation Pages)
    |
Sprint 11 (Net Worth History & Projections) <-> Sprint 12 (Goals & Health) [parallel]
    |
Sprint 13 (OAuth & Wealth AI Tools)
```

Sprint 2 and 3 can be developed in parallel. Sprint 11 and 12 can be developed in parallel. All other sprints are sequential.
