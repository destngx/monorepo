# System Architecture

## Wealth Management System — Technical Design

**Version:** 1.0
**Last Updated:** 2026-02-28

---

## 1. Tech Stack

| Layer           | Technology                   | Rationale                                                            |
| --------------- | ---------------------------- | -------------------------------------------------------------------- |
| **Framework**   | Next.js 15 (App Router)      | Full-stack React, API routes, server components, streaming           |
| **Language**    | TypeScript (strict mode)     | Type safety across the entire codebase                               |
| **Styling**     | Tailwind CSS 4 + shadcn/ui   | Rapid UI development, consistent design system                       |
| **Charts**      | Recharts                     | Lightweight, React-native charting for dashboards                    |
| **AI**          | Vercel AI SDK 4              | Provider-agnostic AI with streaming, tool-calling, structured output |
| **Data Source** | Google Sheets API v4         | Bidirectional sync with existing financial spreadsheet               |
| **Caching**     | In-memory Cache              | High-performance TTL-based cache for sheet data                      |
| **Storage**     | Google Sheets                | Central source of truth for all financial data                       |
| **Market Data** | CoinGecko API + free sources | Crypto, gold, stock prices for portfolio valuation                   |
| **Auth (AI)**   | OAuth 2.0 + API Keys         | Dual auth: OAuth for OpenAI/Google, API key for all                  |

---

## 2. System Architecture

### 2.1 High-Level Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLIENT (Browser)                             │
│                                                                      │
│  ┌────────────┐ ┌────────────┐ ┌──────────────┐ ┌─────────────────┐ ┌──────────────┐ ┌───────────────┐  │
│  │  Dashboard  │ │ Transactions│ │    Budget     │ │ Investments     │ │    Goals       │ │   AI Chat     │  │
│  │  Page       │ │ Page        │ │    Page       │ │ Page            │ │    Page        │ │   Interface   │  │
│  │             │ │             │ │              │ │                 │ │                │ │               │  │
│  │ - Net Worth │ │ - List/CRUD │ │ - Categories │ │ - Portfolio     │ │ - Goals list   │ │ - Streaming   │  │
│  │ - Balances  │ │ - Filters   │ │ - Monthly    │ │ - P&L           │ │ - Progress     │ │ - Tool calls  │  │
│  │ - Charts    │ │ - AI Categ. │ │ - vs Actual  │ │ - Assets        │ │ - Timeline     │ │ - History     │  │
│  │ - Summary   │ │ - Search    │ │ - Alerts     │ │ - Allocation    │ │ - Priorities   │ │ - Multi-model │  │
│  └──────┬─────┘ └──────┬─────┘ └──────┬───────┘ └────────┬────────┘ └────────┬────────┘ └───────┬───────┘  │
│         │               │              │                 │               │                  │          │
│  ═══════╧═══════════════╧══════════════╧══════════════════╧════════  │
│                    React Server Components + Client Components       │
│                    SWR / React Query for data fetching               │
└──────────────────────────────┬───────────────────────────────────────┘
                               │ HTTP / Streaming
┌──────────────────────────────▼───────────────────────────────────────┐
│                       SERVER (Next.js API Routes)                     │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                        API Layer (app/api/)                      │ │
│  │                                                                  │ │
│  │  /api/chat          → AI streaming endpoint                      │ │
│  │  /api/accounts      → Account CRUD (→ Google Sheets)            │ │
│  │  /api/transactions  → Transaction CRUD (→ Google Sheets)        │ │
│  │  /api/budget        → Budget CRUD (→ Google Sheets)             │ │
│  │  /api/investments   → Investment CRUD (→ Google Sheets)         │ │
│  │  /api/portfolio     → Portfolio analytics & allocation           │ │
│  │  /api/goals        → Financial goals CRUD                       │ │
│  │  /api/health       → Financial health score                     │ │
│  │  /api/market-data  → Live prices (→ CoinGecko, Yahoo, etc.)      │ │
│  │  /api/projections  → Net worth projections & what-if            │ │
│  │  /api/auth/[provider] → OAuth initiation                           │ │
│  │  /api/auth/callback → OAuth callback handler                     │ │
│  │  /api/sync          → Manual sync trigger                        │ │
│  │  /api/settings      → AI provider config, preferences           │ │
│  └───────┬─────────────────┬───────────────────┬───────────────────┘ │
│          │                 │                   │                      │
│  ┌───────▼──────┐  ┌──────▼───────┐  ┌───────▼──────────┐          │
│  │   Service     │  │   AI Engine   │  │   Data Access     │          │
│  │   Layer       │  │               │  │   Layer            │          │
│  │              │  │  Vercel AI    │  │                    │          │
│  │ - Business   │  │  SDK          │  │ - Google Sheets   │          │
│  │   logic      │  │ - OpenAI      │  │   Client          │          │
│  │ - Validation │  │ - Gemini      │  │ - Prisma ORM      │          │
│  │ - Transform  │  │ - Anthropic   │  │ - CoinGecko       │          │
│  │              │  │ - Tools       │  │                    │          │
│  └───────┬──────┘  └──────┬───────┘  └───────┬──────────┘          │
│          │                │                   │                      │
│  ════════╧════════════════╧═══════════════════╧════════════════════  │
│                                                                      │
│  ┌──────────────┐  ┌──────────────────┐  ┌────────────────────┐     │
│  │ LocalStorage │  │  Google Sheets    │  │  External APIs     │     │
│  │ (Browser)    │  │  (googleapis)     │  │                    │     │
│  │              │  │                   │  │  - CoinGecko       │     │
│  │ - Chat hist. │  │  - Accounts tab   │  │  - AI Providers    │     │
│  │              │  │  - Budget tab     │  │                    │     │
│  │              │  │  - Transactions   │  │                    │     │
│  │              │  │    tab            │  │                    │     │
│  └──────────────┘  └──────────────────┘  └────────────────────┘     │
└──────────────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow

#### Reading Data (Dashboard Load)

```
Browser → Server Component → Google Sheets API → Transform → Render
                                   ↓ (cache)
                             In-Memory Variable (TTL 5m)
```

1. Server component calls Google Sheets API
2. Response is cached in SQLite with timestamp
3. Data is transformed to internal types
4. Components render with fresh data
5. Subsequent loads use cache if <5 min old, revalidate in background

#### Writing Data (New Transaction)

```
Browser → API Route → Validate → Google Sheets API (append row)
                                         ↓
                                  Cache Invalidation
                                        ↓
                                 AI categorize (if uncategorized)
                                        ↓
                                 Google Sheets API (update category)
```

#### AI Chat Flow

```
Browser → /api/chat (POST, streaming)
    ↓
Vercel AI SDK → Model Selection (from settings)
    ↓
System Prompt (with financial context)
    ↓
Tool Calls (if needed):
  - getAccountBalances() → Google Sheets
  - getTransactions(filters) → Google Sheets
  - getBudgetStatus() → Google Sheets
  - getExchangeRate() → CoinGecko
  - categorizeTransaction() → AI
    ↓
Streaming Response → Browser (real-time)
    ↓
Chat History → LocalStorage (Client-side)
```

#### Market Data Flow

```
CoinGecko/Yahoo/Gold APIs → Server Cache (SQLite, 15min TTL)
                                         ↓
                              Portfolio Valuation
                                         ↓
                              Current prices for assets
```

#### Net Worth Snapshot Flow

```
Daily cron / manual trigger → Fetch all accounts + investments
                                   ↓
                              Calculate total net worth
                                   ↓
                              Store snapshot in SQLite
                                   ↓
                              Historical tracking for charts
```

#### OAuth Token Flow

```
User clicks "Connect OpenAI" → Redirect to provider auth URL (PKCE)
                                      ↓
                            User grants access
                                      ↓
                            Callback with auth code
                                      ↓
                            Exchange code for tokens
                                      ↓
                            Store encrypted tokens in SQLite
                                      ↓
                            Use tokens for AI API calls
```

---

## 3. Project Structure

```
wealth-management/
├── docs/                           # Project documentation
│   ├── PRD.md
│   ├── ARCHITECTURE.md             # ← This file
│   ├── IMPLEMENTATION-PLAN.md
│   ├── GOOGLE-SHEETS-INTEGRATION.md
│   ├── AI-INTEGRATION.md
│   └── SETUP-GUIDE.md
│
├── prisma/
│   └── schema.prisma               # Database schema
│
├── public/
│   └── ...                          # Static assets
│
├── src/
│   ├── app/                         # Next.js App Router
│   │   ├── layout.tsx               # Root layout (providers, nav)
│   │   ├── page.tsx                 # Dashboard (home page)
│   │   ├── loading.tsx              # Dashboard skeleton
│   │   │
│   │   ├── accounts/
│   │   │   └── page.tsx             # Accounts overview
│   │   │
│   │   ├── transactions/
│   │   │   ├── page.tsx             # Transaction list
│   │   │   └── [id]/
│   │   │       └── page.tsx         # Transaction detail/edit
│   │   │
│   │   ├── budget/
│   │   │   └── page.tsx             # Budget management
|
│   │   ├── investments/
│   │   │   └── page.tsx             # Investment portfolio
│   │   │
│   │   ├── goals/
│   │   │   └── page.tsx             # Financial goals
│   │   │
│   │   ├── health/
│   │   │   └── page.tsx             # Financial health score
│   │   │
│   │   │
│   │   ├── chat/
│   │   │   └── page.tsx             # AI chat interface
│   │   │
│   │   ├── settings/
│   │   │   └── page.tsx             # AI provider config, preferences
│   │   │
│   │   └── api/
│   │       ├── chat/
│   │       │   └── route.ts         # AI chat streaming endpoint
│   │       ├── accounts/
│   │       │   └── route.ts         # Account CRUD
│   │       ├── transactions/
│   │       │   ├── route.ts         # Transaction CRUD
│   │       │   └── categorize/
│   │       │       └── route.ts     # AI categorization
│   │       ├── budget/
│   │       │   └── route.ts         # Budget CRUD

│   │       ├── investments/
│   │       │   └── route.ts         # Investment CRUD
│   │       ├── portfolio/
│   │       │   └── route.ts         # Portfolio analytics
│   │       ├── goals/
│   │       │   └── route.ts         # Goals CRUD
│   │       ├── health/
│   │       │   └── route.ts         # Health score calculation
│   │       ├── market-data/
│   │       │   └── route.ts         # Live asset prices
│   │       ├── projections/
│   │       │   └── route.ts         # Net worth projections
│   │       ├── auth/
│   │       │   ├── [provider]/
│   │       │   │   └── route.ts         # OAuth initiation
│   │       │   └── callback/
│   │       │       └── route.ts         # OAuth callback
│   │       ├── sync/
│   │       │   └── route.ts         # Manual sync trigger
│   │       ├── exchange-rate/
│   │       │   └── route.ts         # USDT/VND rate
│   │       └── settings/
│   │           └── route.ts         # Settings CRUD
│   │
│   ├── components/
│   │   ├── ui/                      # shadcn/ui components
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── input.tsx
│   │   │   ├── table.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── select.tsx
│   │   │   ├── badge.tsx
│   │   │   ├── progress.tsx
│   │   │   ├── sheet.tsx
│   │   │   └── ...
│   │   │
│   │   ├── layout/
│   │   │   ├── sidebar.tsx          # Navigation sidebar
│   │   │   ├── header.tsx           # Top header bar
│   │   │   └── mobile-nav.tsx       # Mobile navigation
│   │   │
│   │   ├── dashboard/
│   │   │   ├── net-worth-card.tsx   # Total net worth display
│   │   │   ├── accounts-summary.tsx # Account balances grid
│   │   │   ├── spending-chart.tsx   # Monthly spending chart
│   │   │   ├── budget-overview.tsx  # Budget progress bars
│   │   │   ├── recent-transactions.tsx
│   │   │   └── quick-stats.tsx      # Income/expense/savings

│   │   │   ├── asset-allocation.tsx     # Mini allocation pie chart
│   │   │   ├── net-worth-history.tsx    # Net worth trend line chart
│   │   │   ├── passive-income.tsx       # Passive income summary
│   │   │   └── goals-overview.tsx       # Top goals progress
│   │   │
│   │   ├── transactions/
│   │   │   ├── transaction-table.tsx
│   │   │   ├── transaction-form.tsx
│   │   │   ├── transaction-filters.tsx
│   │   │   └── category-badge.tsx
│   │   │
│   │   ├── budget/
│   │   │   ├── budget-table.tsx
│   │   │   ├── budget-progress.tsx
│   │   │   ├── category-editor.tsx
│   │   │   └── budget-vs-actual.tsx
│   │   │

│   │   ├── investments/
│   │   │   ├── portfolio-overview.tsx   # Total portfolio value, P&L
│   │   │   ├── asset-table.tsx          # Asset list with prices
│   │   │   ├── asset-form.tsx           # Add/edit asset
│   │   │   ├── allocation-chart.tsx     # Pie chart: asset class %
│   │   │   └── price-ticker.tsx         # Live price updates
│   │   │
│   │   ├── goals/
│   │   │   ├── goal-card.tsx            # Goal progress card
│   │   │   ├── goal-form.tsx            # Create/edit goal
│   │   │   └── goal-progress.tsx        # Progress visualization
│   │   │
│   │   ├── health/
│   │   │   ├── health-score.tsx         # Overall score display (0-100)
│   │   │   ├── health-metrics.tsx       # Individual metric cards
│   │   │   └── action-items.tsx         # AI-generated improvements
│   │   │
│   │   └── chat/
│   │       ├── chat-interface.tsx   # Main chat component
│   │       ├── message-bubble.tsx   # Message display
│   │       ├── tool-result.tsx      # Tool call result display
│   │       └── provider-selector.tsx # AI model switcher
│   │
│   ├── lib/
│   │   ├── ai/
│   │   │   ├── providers.ts         # AI provider configuration
│   │   │   ├── tools.ts             # AI tool definitions
│   │   │   ├── system-prompt.ts     # System prompt builder
│   │   │   └── categorizer.ts       # Transaction categorization logic
│   │   │
│   │   ├── sheets/
│   │   │   ├── client.ts            # Google Sheets API client
│   │   │   ├── accounts.ts          # Accounts sheet operations
│   │   │   ├── transactions.ts      # Transactions sheet operations
│   │   │   ├── budget.ts            # Budget sheet operations
│   │   │   └── mappers.ts           # Sheet row ↔ Type mappers
│   │   │
│   │   ├── db/
│   │   │   ├── prisma.ts            # Prisma client singleton
│   │   │   ├── cache.ts             # Cache read/write helpers
│   │   │   └── chat-history.ts      # Chat history persistence
│   │   │
│   │   ├── services/
│   │   │   ├── account-service.ts   # Account business logic
│   │   │   ├── transaction-service.ts
│   │   │   ├── budget-service.ts
│   │   │   └── exchange-rate-service.ts

│   │   │   ├── investment-service.ts    # Investment business logic
│   │   │   ├── portfolio-service.ts     # Portfolio calculations (P&L, allocation)
│   │   │   ├── goal-service.ts          # Goal progress, planning
│   │   │   ├── health-service.ts        # Financial health score
│   │   │   ├── projection-service.ts    # Net worth projections, what-if
│   │   │   ├── market-data-service.ts   # Price fetching (CoinGecko, gold, stocks)
│   │   │   └── auth-service.ts          # OAuth token management
│   │   │
│   │   ├── sheets/
│   │   │   └── investments.ts           # Investments sheet operations
│   │   │
│   │   ├── auth/
│   │   │   ├── oauth-config.ts          # Provider OAuth configs
│   │   │   ├── token-store.ts           # Encrypted token storage
│   │   │   └── providers/
│   │   │       ├── openai.ts            # OpenAI OAuth flow
│   │   │       └── google.ts            # Google OAuth flow (Sheets + Gemini)
│   │   │
│   │   ├── types/
│   │   │   ├── account.ts           # Account types
│   │   │   ├── transaction.ts       # Transaction types
│   │   │   ├── budget.ts            # Budget types
│   │   │   ├── category.ts          # Category types & constants
│   │   │   └── settings.ts          # Settings types

│   │   │   ├── investment.ts            # Investment/asset types
│   │   │   ├── portfolio.ts             # Portfolio/allocation types
│   │   │   ├── goal.ts                  # Goal types
│   │   │   └── health.ts                # Health score types
│   │   │
│   │   ├── utils/
│   │   │   ├── currency.ts          # VND/USDT formatting & conversion
│   │   │   ├── date.ts              # Date formatting (DD/MM/YYYY)
│   │   │   └── validation.ts        # Zod schemas for API validation
│   │   │
│   │   └── constants/
│   │       └── categories.ts        # Category taxonomy definition
│   │
│   └── hooks/
│       ├── use-accounts.ts          # Account data fetching hook
│       ├── use-transactions.ts      # Transaction data fetching hook
│       ├── use-budget.ts            # Budget data fetching hook
│       └── use-exchange-rate.ts     # Exchange rate hook

│   │   ├── use-investments.ts           # Investment data hook
│   │   ├── use-portfolio.ts             # Portfolio analytics hook
│   │   ├── use-goals.ts                 # Goals data hook
│   │   ├── use-health-score.ts          # Health score hook
│   │   └── use-market-data.ts           # Live prices hook
│
├── .env.local                       # Environment variables (secrets)
├── .env.example                     # Template for env vars
├── .gitignore
├── eslint.config.mjs
├── next.config.ts
├── package.json
├── pnpm-lock.yaml
├── postcss.config.mjs
├── tailwind.config.ts
├── tsconfig.json
└── README.md
```

---

## 4. Database Schema (SQLite via Prisma)

SQLite is used for **caching and app state only**. Google Sheets remains the source of truth.

```prisma
// prisma/schema.prisma

generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "sqlite"
  url      = "file:./dev.db"
}

// Cache for Google Sheets data (avoid re-fetching on every request)
model CacheEntry {
  id        String   @id @default(cuid())
  key       String   @unique  // e.g., "accounts", "transactions:2026-02", "budget"
  data      String   // JSON stringified
  updatedAt DateTime @updatedAt
  expiresAt DateTime
}

// AI Chat history
model ChatMessage {
  id          String   @id @default(cuid())
  sessionId   String   // Group messages by chat session
  role        String   // "user" | "assistant" | "tool"
  content     String   // Message content
  toolCalls   String?  // JSON stringified tool calls (if any)
  toolResults String?  // JSON stringified tool results (if any)
  model       String?  // Which AI model was used
  createdAt   DateTime @default(now())

  @@index([sessionId])
}

// Chat sessions
model ChatSession {
  id        String   @id @default(cuid())
  title     String   @default("New Chat")
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}

// AI category learning (corrections improve future predictions)
model CategoryCorrection {
  id              String   @id @default(cuid())
  payee           String   // Transaction payee
  memo            String?  // Transaction memo
  amount          Float?   // Transaction amount
  suggestedCat    String   // AI's original suggestion
  correctedCat    String   // User's correction
  createdAt       DateTime @default(now())

  @@index([payee])
}

// App settings (persisted)
model Setting {
  id    String @id @default(cuid())
  key   String @unique
  value String // JSON stringified

  @@index([key])
}


// Net worth snapshots (for historical tracking)
model NetWorthSnapshot {
  id        String   @id @default(cuid())
  date      DateTime @unique  // One snapshot per day
  totalVND  Float             // Total net worth in VND
  breakdown String            // JSON: { cash: X, crypto: Y, gold: Z, ... }
  createdAt DateTime @default(now())

  @@index([date])
}

// Financial goals
model FinancialGoal {
  id            String   @id @default(cuid())
  name          String            // "Emergency Fund"
  type          String            // emergency_fund, retirement, home, vacation, education, custom
  targetAmount  Float             // Target in VND
  currentAmount Float   @default(0) // Current progress
  targetDate    DateTime?         // Optional deadline
  linkedAccount String?           // Optional: linked account/asset name
  priority      Int     @default(0)  // Priority ranking
  status        String  @default("active") // active, completed, paused
  notes         String?
  createdAt     DateTime @default(now())
  updatedAt     DateTime @updatedAt
}

// OAuth tokens (encrypted)
model OAuthToken {
  id            String   @id @default(cuid())
  provider      String   @unique  // "openai" | "google"
  accessToken   String            // Encrypted
  refreshToken  String?           // Encrypted
  expiresAt     DateTime?
  scopes        String?           // JSON array of granted scopes
  createdAt     DateTime @default(now())
  updatedAt     DateTime @updatedAt
}

// Market data cache (separate from general cache for different TTL)
model MarketPrice {
  id        String   @id @default(cuid())
  symbol    String   @unique  // "BTC", "ETH", "GOLD_SJC", "VN30"
  priceUSD  Float?            // Price in USD
  priceVND  Float             // Price in VND
  source    String            // "coingecko", "yahoo", "custom"
  updatedAt DateTime @updatedAt

  @@index([symbol])
}
```

---

## 5. Component Architecture

### 5.1 Server vs. Client Components

| Component         | Type   | Reason                              |
| ----------------- | ------ | ----------------------------------- |
| Dashboard page    | Server | Initial data fetch, SEO             |
| Net worth card    | Server | Static display of fetched data      |
| Spending chart    | Client | Interactive chart (Recharts)        |
| Transaction table | Client | Sorting, filtering, pagination      |
| Transaction form  | Client | Form state, validation              |
| Budget progress   | Client | Interactive progress bars           |
| AI Chat           | Client | Streaming, real-time updates        |
| Sidebar           | Client | Active route state, collapse toggle |
| Settings page     | Client | Form state, API calls               |

| Allocation chart | Client | Interactive pie chart (Recharts) |
| Net worth history | Client | Interactive line chart (Recharts) |
| Health score | Server | Static display of computed score |
| Goal cards | Server | Static display of goal progress |
| Asset table | Client | Sorting, filtering, live prices |
| What-if calculator | Client | Interactive form with real-time projection |

### 5.2 Data Fetching Strategy

```typescript
// Server Components: Direct fetch in component
async function DashboardPage() {
  const accounts = await getAccounts();       // Google Sheets → cache
  const budget = await getBudgetSummary();     // Google Sheets → cache
  const recent = await getRecentTransactions(); // Google Sheets → cache
  const rate = await getExchangeRate();         // CoinGecko → cache

  return (
    <>
      <NetWorthCard accounts={accounts} exchangeRate={rate} />
      <BudgetOverview budget={budget} />
      <RecentTransactions transactions={recent} />
    </>
  );
}

// Client Components: SWR for real-time updates
function TransactionTable() {
  const { data, mutate } = useSWR('/api/transactions', fetcher);
  // ... interactive table with optimistic updates
}
```

### 5.3 State Management

- **Server state:** SWR (stale-while-revalidate) for Google Sheets data
- **UI state:** React useState/useReducer (local component state)
- **Form state:** React Hook Form + Zod validation
- **AI chat state:** Vercel AI SDK `useChat` hook
- **Global settings:** React Context (AI provider, theme, currency preference)

---

## 6. API Design

### 6.1 REST API Endpoints

```
GET    /api/accounts                    → List all accounts
GET    /api/accounts/:id                → Get account details

GET    /api/transactions                → List transactions (with filters)
POST   /api/transactions                → Create transaction
PUT    /api/transactions/:id            → Update transaction
DELETE /api/transactions/:id            → Delete transaction
POST   /api/transactions/categorize     → AI categorize transaction(s)

GET    /api/budget                      → Get budget (monthly/yearly)
PUT    /api/budget                      → Update budget allocations

POST   /api/chat                        → AI chat (streaming)

POST   /api/sync                        → Force sync with Google Sheets

GET    /api/exchange-rate               → Get USDT/VND rate

GET    /api/settings                    → Get app settings
PUT    /api/settings                    → Update settings


GET    /api/investments              → List all investments/assets
POST   /api/investments              → Add investment
PUT    /api/investments/:id          → Update investment
DELETE /api/investments/:id          → Delete investment

GET    /api/portfolio                → Portfolio summary (allocation, P&L, total value)
GET    /api/portfolio/history        → Net worth history (snapshots)

GET    /api/goals                    → List financial goals
POST   /api/goals                    → Create goal
PUT    /api/goals/:id               → Update goal
DELETE /api/goals/:id               → Delete goal

GET    /api/health                   → Financial health score + metrics

GET    /api/market-data              → Current prices for all tracked assets
GET    /api/market-data/:symbol      → Price for specific asset

POST   /api/projections/net-worth    → Run net worth projection
POST   /api/projections/what-if      → Run what-if scenario

GET    /api/auth/:provider           → Initiate OAuth flow (redirect)
GET    /api/auth/callback            → OAuth callback handler
DELETE /api/auth/:provider           → Disconnect OAuth provider
GET    /api/auth/status              → OAuth connection status per provider
```

### 6.2 Query Parameters (Transactions)

```
GET /api/transactions?
  from=2026-01-01          # Start date
  &to=2026-02-28           # End date
  &account=VietcomBank     # Filter by account
  &category=Food           # Filter by category
  &type=payment            # payment | deposit
  &search=grab             # Search payee/memo
  &limit=50                # Pagination
  &offset=0
```

---

## 7. Security Architecture

### 7.1 API Key Management

```
.env.local (NEVER committed):
  GOOGLE_CLIENT_ID=...
  GOOGLE_CLIENT_SECRET=...
  GOOGLE_SHEETS_ID=...
  OPENAI_API_KEY=...
  GOOGLE_AI_API_KEY=...
  ANTHROPIC_API_KEY=...
```

### 7.2 Authentication Flow

```
1. User visits app → Redirect to Google OAuth
2. User grants Google Sheets access
3. OAuth tokens stored in HTTP-only cookies (encrypted)
4. API routes validate session before accessing Sheets
5. Refresh tokens handled automatically
```

### 7.3 OAuth Authentication Flows

**OpenAI OAuth:** Authorization Code + PKCE flow

- Auth URL: https://auth.openai.com/oauth/authorize
- Token URL: https://auth.openai.com/oauth/token
- Scopes: openid profile email offline_access

**Google OAuth:** Standard Authorization Code flow

- Scopes: spreadsheets + generative-language (single login for both Sheets + Gemini)
- If user connects Google OAuth, it replaces the service account for Sheets access AND provides Gemini access

**Anthropic:** API key only (OAuth banned Feb 2026)

**Token Storage & Management:**

- Tokens stored encrypted in SQLite OAuthToken table
- Token refresh handled automatically before expiration
- Fallback: If OAuth token expired and refresh fails, fall back to API key

### 7.4 Security Measures

- All API keys server-side only (never exposed to client)
- Google OAuth scope limited to spreadsheets only
- SQLite DB is local (no network exposure)
- Input validation with Zod on all API endpoints
- Rate limiting on AI chat endpoint (prevent runaway API costs)

---

## 8. Error Handling

### 8.1 Error Hierarchy

```typescript
// Base error
class AppError extends Error {
  constructor(
    message: string,
    public code: string,
    public status: number = 500,
    public details?: unknown
  ) {
    super(message);
  }
}

// Specific errors
class SheetsError extends AppError { ... }      // Google Sheets failures
class AIError extends AppError { ... }           // AI provider failures
class ValidationError extends AppError { ... }   // Input validation
class AuthError extends AppError { ... }         // Authentication failures
```

### 8.2 Error Strategy

| Scenario                 | Handling                                                        |
| ------------------------ | --------------------------------------------------------------- |
| Google Sheets API down   | Serve from SQLite cache, show "offline" indicator               |
| AI provider error        | Fall back to next configured provider, or show "AI unavailable" |
| Invalid transaction data | Return 400 with Zod validation errors                           |
| Rate limit exceeded      | Return 429, show cool-down timer in UI                          |
| Network timeout          | Retry with exponential backoff (max 3 attempts)                 |

---

## 9. Performance Optimizations

1. **Server Components** — Zero client JS for static dashboard sections
2. **Streaming** — AI responses stream token-by-token via Vercel AI SDK
3. **Caching** — SQLite cache layer with 5-minute TTL for Sheets data
4. **Optimistic Updates** — UI updates immediately, syncs in background
5. **Incremental Sync** — Only fetch changed rows when possible (timestamp-based)
6. **Code Splitting** — Dynamic imports for chart library (heavy)
7. **Image Optimization** — Next.js Image component for any assets

---

## 10. Navigation Structure

### Sidebar Navigation Items

- Dashboard (/)
- Transactions (/transactions)
- Accounts (/accounts)
- Budget (/budget)
- Investments (/investments) # New
- Goals (/goals) # New
- Health (/health) # New
- Chat (/chat)
- Settings (/settings)
