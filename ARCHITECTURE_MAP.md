# Next.js App Router Architecture Map

## Overview

Monorepo contains 3 Next.js apps using App Router. Focus: **wealth-management** (most complex). All use **feature-based architecture** with clear separation between routing and implementation.

---

## 1. WEALTH-MANAGEMENT APP

### 1.1 Routing Pattern: Three-Layer Delegation

```
src/app/[route]/
    ↓ (empty delegate)
src/features/[feature]/ui/
    ↓ (imports shared code)
src/lib/ + src/components/
```

**Key principle**: App Router (`src/app/`) contains ONLY route definitions and delegating files. All actual implementation lives in `features/`.

### 1.2 Page Routes (User-Facing)

| Route Path                      | Delegates To                                    | Dependencies                                           | Abstraction Level        |
| ------------------------------- | ----------------------------------------------- | ------------------------------------------------------ | ------------------------ |
| `/` (dashboard)                 | `page.tsx` (direct)                             | `lib/sheets/*`, `components/dashboard/*`               | ⚠️ HIGH - mixed concerns |
| `/accounts`                     | `features/accounts/ui/page`                     | `features/accounts/api`, `components/accounts/*`       | ✅ GOOD                  |
| `/accounts/goals`               | `features/goals/ui/page` (via accounts feature) | `features/goals/*`, `lib/sheets/goals`                 | ✅ GOOD                  |
| `/accounts/goals/[id]`          | Dynamic goal details                            | `features/goals/ui`                                    | ✅ GOOD                  |
| `/accounts/goals/new`           | Goal creation form                              | `features/goals/ui`                                    | ✅ GOOD                  |
| `/accounts/credit-cards`        | Credit card feature                             | `features/accounts/ui`                                 | ✅ GOOD                  |
| `/accounts/credit-cards/[name]` | Dynamic card details                            | `features/accounts/ui`, `lib/sheets/accounts`          | ✅ GOOD                  |
| `/accounts/loans`               | Loan feature                                    | `features/loans/ui`, `lib/sheets/loans`                | ✅ GOOD                  |
| `/budget`                       | `features/budget/ui/page`                       | `features/budget/api`, `features/transactions/*`       | ✅ GOOD                  |
| `/transactions`                 | `features/transactions/ui/page`                 | `features/transactions/api`, `lib/sheets/transactions` | ✅ GOOD                  |
| `/investments`                  | Investment feature                              | `features/investments/ui`, `lib/sheets/*`              | ✅ GOOD                  |
| `/chat`                         | `features/chat/ui/ChatContainer`                | `lib/ai/*`, `features/chat/*`                          | ✅ GOOD                  |
| `/health`                       | Financial health dashboard                      | `components/dashboard/*`, `lib/*`                      | ⚠️ MEDIUM                |
| `/settings`                     | `features/settings/ui/page`                     | `features/settings/*`, `lib/types/settings`            | ✅ GOOD                  |

### 1.3 API Routes (Backend Handlers)

#### Pattern: Delegation → Implementation

**All API routes follow**: `app/api/[path]/route.ts` → `features/[feature]/api/route.ts`

##### Category: Core Data APIs (Delegate)

| Endpoint                | Implements                  | Imports From                                         | Can Abstract?            |
| ----------------------- | --------------------------- | ---------------------------------------------------- | ------------------------ |
| `GET /api/accounts`     | `features/accounts/api`     | `lib/sheets/accounts`, `lib/utils/api-error-handler` | ✅ YES - pure data fetch |
| `GET /api/budget`       | `features/budget/api`       | `lib/sheets/budget`, `lib/sheets/categories`         | ✅ YES - pure data fetch |
| `GET /api/transactions` | `features/transactions/api` | `lib/sheets/transactions`                            | ✅ YES - pure data fetch |
| `GET /api/goals`        | Direct handler              | `lib/sheets/goals`                                   | ✅ YES                   |
| `GET /api/loans`        | Direct handler              | `lib/sheets/loans`                                   | ✅ YES                   |
| `GET /api/tags`         | Direct handler              | `lib/sheets/tags`                                    | ✅ YES                   |
| `GET /api/categories`   | Direct handler              | `lib/sheets/categories`                              | ✅ YES                   |

##### Category: Investment APIs (Delegate)

| Endpoint                      | Implements                        | Imports From         | Can Abstract? |
| ----------------------------- | --------------------------------- | -------------------- | ------------- |
| `GET /api/investments/assets` | `features/investments/api/assets` | `lib/sheets/*`       | ✅ YES        |
| `GET /api/investments/prices` | `features/investments/api/prices` | Market data provider | ✅ YES        |

##### Category: AI-Powered Routes (Direct in app/api - NOT delegated)

| Endpoint                             | Purpose                     | Key Imports                          | Feature-Specific?                | Can Abstract?              |
| ------------------------------------ | --------------------------- | ------------------------------------ | -------------------------------- | -------------------------- |
| `POST /api/ai/budget-advisor`        | Generate AI budget coaching | `lib/ai/providers`, `generateText`   | ⚠️ MEDIUM - uses budget data     | ✅ MAYBE - tool is generic |
| `POST /api/ai/budget-review`         | AI budget analysis          | `lib/ai/providers`, task instruction | ⚠️ MEDIUM - budget-specific      | ✅ MAYBE                   |
| `POST /api/ai/financial-health`      | Overall financial score     | `lib/ai/providers`                   | ❌ NO - cross-feature aggregate  | ❌ NO                      |
| `POST /api/ai/intelligence-briefing` | Daily briefing              | `lib/ai/providers`                   | ❌ NO - cross-feature aggregate  | ❌ NO                      |
| `POST /api/ai/investment-analysis`   | Investment recommendations  | `lib/ai/providers`                   | ⚠️ MEDIUM - investments feature  | ✅ MAYBE                   |
| `POST /api/ai/account-review`        | Account analysis            | `lib/ai/providers`                   | ⚠️ MEDIUM - accounts feature     | ✅ MAYBE                   |
| `POST /api/ai/loan-review`           | Loan analysis               | `lib/ai/providers`                   | ⚠️ MEDIUM - loans feature        | ✅ MAYBE                   |
| `POST /api/ai/credit-summary`        | Credit card summary         | `lib/ai/providers`                   | ⚠️ MEDIUM - credit cards         | ✅ MAYBE                   |
| `POST /api/ai/transaction-review`    | Transaction categorization  | `lib/ai/providers`                   | ⚠️ MEDIUM - transactions feature | ✅ MAYBE                   |
| `POST /api/ai/chart-insight`         | Chart analysis              | `lib/ai/providers`                   | ❌ NO - cross-feature            | ❌ NO                      |
| `POST /api/ai/parse-notifications`   | Parse bank notifications    | `lib/ai/providers`                   | ⚠️ MEDIUM - notification parsing | ✅ MAYBE                   |
| `POST /api/ai/suggest-category`      | Auto-categorize transaction | `lib/ai/providers`                   | ⚠️ MEDIUM - transactions feature | ✅ MAYBE                   |
| `POST /api/chat`                     | Main chat interface         | `lib/ai/providers`, `financialTools` | ❌ NO - aggregate tool calls     | ❌ NO                      |
| `POST /api/chat/suggestions`         | Quick suggestion prompts    | Direct handler                       | ⚠️ MEDIUM - UI-specific          | ✅ MAYBE                   |

##### Category: Data Management APIs

| Endpoint                 | Purpose                 | Can Delegate?                 |
| ------------------------ | ----------------------- | ----------------------------- |
| `GET /api/exchange-rate` | Currency conversion     | ✅ YES - pure utility         |
| `POST /api/sync`         | Data synchronization    | ⚠️ NO - app-wide coordination |
| `GET /api/notifications` | Notification management | ✅ YES - can delegate         |
| `GET /api/market-pulse`  | Market data feed        | ✅ YES - pure data fetch      |

### 1.4 Feature Structure (Core Pattern)

Each feature follows this structure:

```
features/[feature]/
├── api/
│   ├── route.ts          # Handler (often delegates to lib)
│   └── [nested]/route.ts # Nested routes
├── model/
│   ├── types.ts          # Feature-specific types
│   └── utils.ts          # Feature logic
├── ui/
│   ├── page.tsx          # Page component (server or client)
│   ├── index.ts          # UI exports barrel file
│   └── [component].tsx   # Feature-specific components
└── hooks/                # Feature-specific hooks (if used)
```

**Current features**:

- `accounts/` - Accounts, credit cards (accounts feature)
- `budget/` - Budget tracking & AI advisor
- `chat/` - AI chat interface
- `goals/` - Financial goals (nested under accounts)
- `investments/` - Investment tracking
- `loans/` - Loan management
- `settings/` - User settings
- `transactions/` - Transaction history

### 1.5 Shared Code Architecture

#### `lib/sheets/` - Data Access Layer

Core data fetching from Google Sheets (shared dependency):

```
accounts.ts      → Fetch account balances
budget.ts        → Fetch budget allocations
categories.ts    → Fetch category mappings
goals.ts         → Fetch financial goals
loans.ts         → Fetch loan data
transactions.ts  → Fetch all transactions
notifications.ts → Parse notifications
tags.ts          → Fetch custom tags
```

**Pattern**: Each module exports a single async function: `getXXX(forceFresh?: boolean)`

- Implements caching via `@/lib/sheets/client` (Google Sheets client)
- Error handling via `@/lib/utils/api-error-handler`

#### `lib/ai/` - AI Orchestration (Used by multiple routes)

```
providers.ts        → Model selection (GPT-4o, GitHub GPT, etc.)
system-prompt.ts    → Base system prompt builder
tools/             → Financial tools for AI agent
core/
  ├── orchestrator.ts → Multi-step AI workflows
  └── parser.ts      → Response parsing
```

**Dependencies**: Every AI route imports `getLanguageModel()` and `buildSystemPrompt()`

#### `lib/utils/` - Utilities

```
api-error-handler.ts → Standardized API error handling
currency.ts         → Currency conversion & formatting
date.ts            → Date utilities
validation.ts      → Data validation
cashback.ts        → Cashback calculation
```

#### `lib/types/` - Domain Types

```
account.ts
budget.ts
category.ts
goals.ts
loan.ts
transaction.ts
notification.ts
settings.ts
```

#### `lib/constants/` - App Configuration

```
navigation.ts  → Route definitions & sidebar structure
categories.ts  → Budget category mappings
tags.ts       → Transaction tag options
```

#### `components/` - Shared UI Components

```
layout/
  ├── sidebar.tsx
  ├── header.tsx
  ├── layout-wrapper.tsx
  └── sidebar-provider.tsx
dashboard/
  ├── net-worth-trend-card.tsx
  ├── ai-daily-briefing.tsx
  ├── snapshot-cards-row.tsx
  ├── accounts-summary.tsx
  ├── budget-overview.tsx
  └── spending-chart.tsx
ui/              → shadcn UI components
chat/
  ├── ai-chat-widget.tsx
  ├── ai-context-provider.tsx
  └── message.tsx
[feature]/      → Feature-specific components
```

### 1.6 Layout Composition

**Root Layout** (`src/app/layout.tsx`):

```
html/body
└─ ThemeProvider
   └─ MaskProvider
      └─ AIContextProvider
         └─ SidebarProvider
            └─ flex container
               ├─ Sidebar (navigation)
               ├─ LayoutWrapper
               │  ├─ Header
               │  └─ main (content)
               └─ AIChatWidget (floating)
```

**Components involved**:

- `ThemeProvider` - Dark/light theme (from `@/components/theme-provider`)
- `SidebarProvider` - Sidebar state management
- `AIContextProvider` - Global AI state (chat context)
- `MaskProvider` - Probably masking sensitive data

**No nested layouts per feature** - all features use the root layout.

### 1.7 App-Specific vs. Shared Dependencies

#### App-Specific (Wealth-Management Only)

- Google Sheets integration (`lib/sheets/`)
- Budget/financial domain logic
- AI system prompts for finance
- All feature-specific components

#### Could Be Abstracted to Shared Lib

- AI orchestration core (`lib/ai/core/`)
- Error handling utilities
- Type definitions system
- Layout components (Sidebar, Header pattern)
- UI component library pattern (using shadcn)

---

## 2. PORTFOLIO-LANDPAGE APP

### Structure

```
src/app/
├── layout.tsx   → Root layout
├── page.tsx     → Single landing page
├── page.css     → Page styles
├── error.tsx    → Error boundary
└── globals.css  → Global styles
```

**Characteristics**:

- **NO feature-based architecture** (single page static)
- Minimal routing (only home page)
- NO API routes
- NO dynamic routes
- Direct component imports in `page.tsx`

**Can Abstract**: Components & design system only

---

## 3. CLOUDINARY-PHOTOS-APP

### Structure

```
src/app/
├── layout.tsx            → Root layout
├── page.tsx              → Gallery home
├── albums/
│  ├── page.tsx          → Albums list
│  ├── [albumName]/
│  │  └── page.tsx       → Album detail (dynamic)
│  └── album-card.tsx    → Album component
├── favorites/
│  ├── page.tsx
│  └── favorites-list.tsx
├── detail/
│  ├── page.tsx
│  └── photo-detail.tsx
├── gallery/
│  ├── page.tsx
│  ├── gallery-grid.tsx
│  ├── upload-button.tsx
│  └── search-form.tsx
└── [loading.tsx, error.tsx]
```

**Characteristics**:

- Component-based pages (lightweight)
- Cloudinary API integration
- Photo/album management
- Simple routing (no complex feature delegation)
- NO separate features/ folder

**Can Abstract**: Cloudinary integration utilities

---

## 4. ABSTRACTION OPPORTUNITIES

### High Confidence ✅ (Can move to shared lib)

1. **AI Orchestration Core** (`lib/ai/core/`)
   - Orchestrator pattern for multi-step workflows
   - Parser utilities
   - **Benefit**: Reusable for any AI-powered app
   - **Risk**: None - no wealth-specific code

2. **Error Handling Utilities** (`lib/utils/api-error-handler`)
   - Standardized API error responses
   - **Benefit**: Consistent error handling across apps
   - **Risk**: None - generic utility

3. **Layout Component System**
   - Sidebar, Header, LayoutWrapper patterns
   - SidebarProvider state management
   - **Benefit**: Reusable dashboard layout template
   - **Risk**: None - UI-only

4. **Type Definition System**
   - Domain types pattern (account, transaction, etc.)
   - **Benefit**: Template for other domain-driven apps
   - **Risk**: None - just patterns

5. **Feature-Based Architecture Pattern**
   - Structure: `features/[feature]/{api, ui, model}`
   - Delegation pattern from `app/` to `features/`
   - **Benefit**: Consistent across all apps
   - **Risk**: Requires documentation

### Medium Confidence ⚠️ (Depends on use case)

1. **Feature-Specific AI Routes** (`api/ai/[feature]/*`)
   - Routes like `/api/ai/budget-advisor`, `/api/ai/investment-analysis`
   - **Can abstract if**: Other apps need domain-specific AI analysis
   - **Cannot abstract if**: Only wealth-management needs this

2. **Chat Integration** (`api/chat/route.ts`)
   - Main chat interface with tool calling
   - **Can abstract if**: Pattern generalizes to non-financial apps
   - **Cannot abstract**: Domain-specific financial tools embedded

3. **Dashboard Components** (`components/dashboard/*`)
   - KPI cards, trend charts, data visualization
   - **Can abstract if**: Pattern is generic enough
   - **Cannot abstract**: Heavy financial domain logic

### Low Confidence ❌ (App-specific)

1. **Wealth-Management Domain Logic**
   - Budget calculations, financial metrics
   - Google Sheets integration
   - Category/tag management

2. **Cross-Feature Aggregation APIs**
   - `/api/ai/financial-health` (aggregates all data)
   - `/api/ai/intelligence-briefing` (cross-domain briefing)

---

## 5. REFACTORING IMPACT ANALYSIS

### Safe to Refactor (Low Impact)

**Move to shared lib**:

- ✅ `lib/ai/core/*` → `libs/ai-orchestration/`
- ✅ `lib/utils/api-error-handler` → `libs/utils/`
- ✅ `components/layout/*` → `libs/ui-layout/`
- ✅ `components/ui/*` → Already in `libs/ui` (if migrated)

**Benefit**: Reduces code duplication, centralizes patterns

**Impact on wealth-management**: None - uses via imports

---

## 6. ROUTING PRESERVATION CHECKLIST

When refactoring:

- ✅ **DO NOT change**: Route paths in `src/app/`
- ✅ **DO NOT change**: API endpoint paths (`/api/[path]`)
- ✅ **DO NOT change**: Page component names exported from features
- ✅ **DO change**: Move implementations to shared libs via `@alias` imports
- ✅ **DO document**: New import paths if lib locations change

### Dynamic Route Preservation

| Route                           | Parameter   | Constraint          | Preserve? |
| ------------------------------- | ----------- | ------------------- | --------- |
| `/accounts/goals/[id]`          | `id`        | Financial goal UUID | ✅ YES    |
| `/accounts/credit-cards/[name]` | `name`      | Card name string    | ✅ YES    |
| `/albums/[albumName]`           | `albumName` | Album folder name   | ✅ YES    |

---

## 7. DEPENDENCY GRAPH (Wealth-Management)

### Page → Feature Dependencies

```
/                  → dashboard components + lib/sheets
/accounts          → features/accounts + lib/sheets/accounts
/accounts/goals    → features/goals + lib/sheets/goals
/budget            → features/budget + features/transactions + lib/sheets
/chat              → features/chat + lib/ai/*
/investments       → features/investments + lib/sheets
/health            → components/dashboard + lib/*
/transactions      → features/transactions + lib/sheets/transactions
/settings          → features/settings
```

### API → Library Dependencies

```
/api/accounts               → lib/sheets/accounts
/api/budget                 → lib/sheets/{budget, categories}
/api/transactions           → lib/sheets/transactions
/api/*/ai/*                 → lib/ai/{providers, system-prompt, tools}
/api/investments/assets     → lib/sheets/investments
/api/investments/prices     → external market data
```

### Cross-Feature Dependencies

```
budget/ui/page.tsx          → features/transactions/model/types
features/chat/ui            → lib/ai/*
features/*/ui               → components/* (shared)
```

---

## Summary

**Wealth-Management** follows a well-structured feature-based architecture suitable for refactoring. Key patterns:

1. **Three-layer separation**: `app/` (routes only) → `features/` (implementation) → `lib/` (shared)
2. **Clear delegation**: Pages and APIs delegate to features with minimal logic
3. **Shared layer**: `lib/` contains data access, AI orchestration, utilities, types
4. **Reusable components**: Centralized UI components and layout system

**Refactoring strategy**:

- Move `lib/ai/core/*`, layout components, and utilities to shared libs
- Preserve all route paths and page/API structure
- Update imports to point to new lib locations
- Document delegation patterns for other apps

---
