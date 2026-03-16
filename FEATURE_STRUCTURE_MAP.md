# Wealth Management App - Feature Structure Analysis

## Executive Summary
- **Features**: 8 independent modules (accounts, budget, chat, goals, investments, loans, settings, transactions)
- **Total Files**: 112 TypeScript/TSX files across all features
- **Architecture Pattern**: FSD-aligned with model/ui/api separation
- **Key Dependency**: Transactions feature used by Accounts & Budget
- **Shared Infrastructure**: `/lib` directory contains 40+ shared utilities, services, and types

---

## FEATURE BREAKDOWN

### 1. ACCOUNTS
**Purpose**: Manage financial accounts (checking, savings, credit cards, crypto wallets)

**File Count**: 16 files
**Structure**:
- `model/` (7 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (8 files): page.tsx, 4 AI components, 2 credit-specific, efficiency-chart
- `api/` (1 file): route.ts

**Model Layer**:
- **Types**: AccountType (9 variants), Currency (VND/USD/USDT), Account interface
- **Queries**: getAccounts(), getAccountById(), getAccountsByType()
- **Mutations**: createAccount(), updateAccount(), deleteAccount()
- **Hooks**: useAccounts(), useAccountById()

**UI Components**:
- AccountsPage (main page)
- AccountReviewAI (AI analysis component)
- AccountTrendSparkline (trend visualization)
- CreditCardSummaryAI (credit-specific analysis)
- EfficiencyChart (efficiency visualization)
- Credit-specific: due-date-countdown, statement-cycle-bar, utilization-ring

**Cross-Feature Dependencies**:
- ✅ Imports: @/features/transactions/model/types (Transaction type)
- Used by: account-trend-sparkline.tsx, credit-card-summary-ai.tsx, page.tsx

**External Dependencies**:
- @/lib/sheets/accounts (getAccounts)
- @/lib/utils/api-error-handler
- @/components/ui/* (shared UI components)
- @/components/dashboard/* (AI insights)

**API**:
- GET /api/accounts - fetches all accounts with optional force refresh

---

### 2. TRANSACTIONS
**Purpose**: Track income/expense transactions, categorization, and filtering

**File Count**: 14 files
**Structure**:
- `model/` (7 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (6 files): page.tsx, transaction-form, transaction-table, transaction-filters, notification-processor, transaction-review-ai
- `api/` (1 file): route.ts

**Model Layer**:
- **Types**: TransactionType (income/expense/non-budget), Transaction interface (19 fields)
- **Queries**: getTransactions(), filterTransactions(), getTransactionsByDateRange()
- **Mutations**: addTransaction(), updateTransaction(), deleteTransaction()
- **Hooks**: useTransactions(), useTransactionFilters()

**UI Components**:
- TransactionsPage (main page)
- TransactionForm (create/edit)
- TransactionTable (list view with pagination)
- TransactionFilters (date, category, account filters)
- NotificationProcessor (real-time updates)
- TransactionReviewAI (AI analysis)

**Cross-Feature Dependencies**:
- ✅ USED BY: Accounts, Budget features
- ✅ Exports: Transaction type to other features
- Used in: accounts/ui/account-trend-sparkline.tsx, accounts/ui/page.tsx, budget/ui/page.tsx, budget/ui/category-detail-view.tsx

**External Dependencies**:
- @/lib/sheets/transactions (getTransactions, addTransaction)
- @/lib/sheets/categories (getCategories)
- @/lib/utils/validation (TransactionSchema)
- @/lib/utils/date (parseDate)

**API**:
- GET /api/transactions - fetch with filters
- POST /api/transactions - create new transaction

---

### 3. BUDGET
**Purpose**: Set and track budget limits by category, compare against actual spending

**File Count**: 13 files
**Structure**:
- `model/` (7 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (5 files): page.tsx, budget-overview-view, category-detail-view, budget-review-ai, ai-budget-advisor-view
- `api/` (1 file): route.ts

**Model Layer**:
- **Types**: BudgetItem interface (category, limit, spent, period)
- **Queries**: getBudgetItems(), getBudgetByCategory(), getTotalBudgetSpent()
- **Mutations**: createBudgetItem(), updateBudgetItem(), deleteBudgetItem()
- **Hooks**: useBudgetItems()

**UI Components**:
- BudgetPage (main page with month navigation)
- BudgetOverviewView (all categories summary)
- CategoryDetailView (detailed category breakdown)
- BudgetReviewAI (AI analysis)
- AIBudgetAdvisorView (AI recommendations)

**Cross-Feature Dependencies**:
- ✅ Imports: @/features/transactions/model/types (Transaction)
- Depends on: Transactions data to calculate spent amounts
- Uses: Transaction data to populate budget vs actual comparisons

**External Dependencies**:
- @/lib/sheets/budget (getBudget)
- @/lib/sheets/categories (getCategories)
- @/lib/utils/api-error-handler
- @/hooks/use-ai-settings (AI settings)

**API**:
- GET /api/budget - fetch budget limits and spending data

---

### 4. CHAT
**Purpose**: AI-powered chat assistant for financial queries and insights

**File Count**: 17 files
**Structure**:
- `model/` (6 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (10 files): ai-chat-widget, ai-drawer, ai-fab, chat-interface, chat-container, chat-messages, chat-input, model-switcher, ai-insight-card, ai-context-provider
- NO API route (uses server actions)

**Model Layer**:
- **Types**: ChatMessage, ChatResponse, SuggestionItem, SuggestionsResponse, AIInsight
- **Queries**: loadChatHistory(), generateMessageId(), getActiveModelLabel(), fetchSuggestions()
- **Mutations**: saveChatHistory(), clearChatHistory(), sendChatMessage()
- **Hooks**: useChatMessages(), useChatStream(), useChatSuggestions()

**UI Components**:
- AIFab (floating action button)
- AIDrawer (drawer container)
- ChatWidget (standalone widget)
- ChatInterface (main chat UI)
- ChatContainer (layout)
- ChatMessages (message list)
- ChatInput (input field)
- ModelSwitcher (select AI model)
- AIInsightCard (display insights)
- AIContextProvider (context for chat)

**Cross-Feature Dependencies**:
- ❌ NONE - fully self-contained
- No imports from other features

**External Dependencies**:
- @/lib/ai/providers (AI_MODELS)
- localStorage API (for chat history)
- Server actions for chat streaming

**Storage**:
- localStorage key: "wealthos-chat-history"

---

### 5. GOALS
**Purpose**: Create, track, and project financial goals with AI insights

**File Count**: 14 files
**Structure**:
- `model/` (7 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (8 files): page.tsx, goal-card, goal-detail-page, goal-detail-chart, new-goal-page, create-goal-flow, ai-insights-panel, ai-summary-card
- NO API route

**Model Layer**:
- **Types**: Goal, GoalType, GoalProjection, Milestone, Contribution, AIStatus
- **Queries**: getGoals(), getGoalById(), getGoalProjection()
- **Mutations**: createGoal(), updateGoal(), deleteGoal(), addContribution()
- **Hooks**: useGoals(), useGoal(), useGoalProjection()

**UI Components**:
- GoalsPage (main list)
- GoalCard (individual goal card)
- GoalDetailPage (detailed view)
- GoalDetailChart (progress chart)
- NewGoalPage (create goal)
- CreateGoalFlow (wizard)
- AISummaryCard (AI summary)
- AIInsightsPanel (detailed insights)

**Cross-Feature Dependencies**:
- ❌ NONE - fully self-contained

**External Dependencies**:
- @/lib/sheets/goals (getGoals)
- @/lib/ai/providers (for AI insights)

---

### 6. LOANS
**Purpose**: Manage and track loans, repayment schedules, and interest

**File Count**: 11 files
**Structure**:
- `model/` (8 files): types.ts, queries.ts, mutations.ts, hooks.ts, server-queries.ts, index.ts
- `ui/` (4 files): page.tsx, loan-list, loan-summary, loan-review-ai
- NO API route

**Model Layer**:
- **Types**: Loan interface with repayment schedule
- **Queries**: getLoans(), getLoanById()
- **Server Queries**: getLoans() - server-side variant
- **Mutations**: createLoan(), updateLoan(), deleteLoan(), addRepayment()
- **Hooks**: useLoans(), useLoanById()

**UI Components**:
- LoansPage (main list)
- LoanList (formatted list)
- LoanSummary (summary card)
- LoanReviewAI (AI analysis)

**Cross-Feature Dependencies**:
- ❌ NONE - fully self-contained

**External Dependencies**:
- @/lib/sheets/loans (getLoans)

---

### 7. INVESTMENTS
**Purpose**: Track investment portfolio, asset prices, and performance

**File Count**: 10 files
**Structure**:
- `model/` (7 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (1 file): page.tsx
- `api/` (2 files): assets/route.ts, prices/route.ts

**Model Layer**:
- **Types**: Investment (symbol, quantity, prices), PortfolioAnalysis
- **Queries**: getAssets(), getAssetPrices()
- **Mutations**: addAsset(), updateAsset(), removeAsset()
- **Hooks**: useAssets(), useAssetPrices()

**UI Components**:
- InvestmentsPage (main dashboard)

**Cross-Feature Dependencies**:
- ❌ NONE - fully self-contained

**External Dependencies**:
- @/lib/services/price-service (getPrice)
- External price APIs (stocks, crypto)

**API**:
- GET /api/investments/assets - fetch portfolio
- GET /api/investments/prices - fetch current prices

---

### 8. SETTINGS
**Purpose**: User preferences, notification settings, app configuration

**File Count**: 7 files
**Structure**:
- `model/` (6 files): types.ts, queries.ts, mutations.ts, hooks.ts, index.ts
- `ui/` (1 file): page.tsx
- NO API route

**Model Layer**:
- **Types**: Settings interface (AI preferences, notifications, currency)
- **Queries**: getSettings()
- **Mutations**: updateSettings(), updateSetting()
- **Hooks**: useSettings()

**UI Components**:
- SettingsPage (settings form)

**Cross-Feature Dependencies**:
- ❌ NONE - fully self-contained

**External Dependencies**:
- localStorage API
- @/lib/types/settings

---

## CROSS-FEATURE DEPENDENCY MAP

```
                    ┌─────────────┐
                    │ Transactions│
                    └──────┬──────┘
                           │ (Transaction type)
              ┌────────────┼────────────┐
              │            │            │
        ┌─────▼──┐   ┌─────▼──┐   ┌────▼─────┐
        │Accounts│   │ Budget  │   │(exported)│
        └────────┘   └─────────┘   └──────────┘

Standalone (no dependencies):
├── Chat (uses @/lib/ai only)
├── Goals
├── Loans
├── Investments
└── Settings
```

### Dependency Matrix

| Feature | Depends On | Exports | Files | Has API | Has Hooks |
|---------|-----------|---------|-------|---------|-----------|
| Accounts | Transactions | Account, AccountType, Currency | 16 | ✅ | ✅ |
| Budget | Transactions | BudgetItem | 13 | ✅ | ✅ |
| Chat | @/lib/ai only | ChatMessage, ChatResponse | 17 | ❌ | ✅ |
| Goals | None | Goal, GoalType, GoalProjection | 14 | ❌ | ✅ |
| Loans | None | Loan | 11 | ❌ | ✅ |
| Investments | None | Investment, PortfolioAnalysis | 10 | ✅ | ✅ |
| Settings | None | Settings | 7 | ❌ | ✅ |
| Transactions | None | Transaction, TransactionType | 14 | ✅ | ✅ |

---

## SHARED INFRASTRUCTURE (/lib)

### Type Definitions (`/lib/types`)
- account.ts - Account type definitions
- budget.ts - Budget type definitions
- category.ts - Category/spending category types
- goals.ts - Goals type definitions
- loan.ts - Loan type definitions
- notification.ts - Notification types
- settings.ts - Settings type definitions
- transaction.ts - Transaction type definitions

### Data Access Layer (`/lib/sheets`)
- **Core**: client.ts (Google Sheets auth & API), auth.ts (authentication)
- **Feature-specific readers**:
  - accounts.ts - read accounts from Google Sheets
  - budget.ts - read budget limits
  - transactions.ts - read & write transactions
  - loans.ts - read loan data
  - goals.ts - read goal data
  - categories.ts - read spending categories
  - notifications.ts - read notifications
- **Utilities**: mappers.ts (data normalization)
- **Tests**: *.test.ts files for critical readers

### AI Services (`/lib/ai`)
- providers.ts - AI model definitions (Claude, GPT, etc.)
- Integration hooks for chat and feature AI components

### Price Services (`/lib/services`)
- price-service.ts - fetches current prices for investments (stocks, crypto)

### Utilities (`/lib/utils`)
- validation.ts - Zod schemas (TransactionSchema, TransactionInput)
- date.ts - date parsing and formatting (formatDate, parseDate)
- currency.ts - currency formatting (formatVND)
- api-error-handler.ts - standardized error handling
- utils.ts - general utilities (cn class merger, etc.)

### Constants (`/lib/constants`)
- categories.ts - category definitions and config
- tags.ts - transaction tag definitions
- navigation.ts - navigation structure

---

## CURRENT ARCHITECTURE PATTERNS

### Model Layer (FSD Aligned)
All features follow consistent pattern:
```
feature/model/
├── types.ts        # TypeScript interfaces & types
├── queries.ts      # Async read operations
├── mutations.ts    # Async write operations
├── hooks.ts        # React hooks (client-side)
├── server-queries.ts  # Server-side queries (optional, only in some features)
└── index.ts        # Barrel export of everything
```

### UI Layer
- **Page component**: page.tsx (entry point for feature route)
- **Sub-components**: Feature-specific, named descriptively
- **AI components**: Follow naming: *-ai.tsx
- **Nested structures**: Some components have subdirectories (e.g., accounts/ui/credit/)

### API Layer
- **Route.ts files**: Handle GET/POST requests
- Only present in: Accounts, Budget, Transactions, Investments (2 routes)
- Others: Use client-side data fetching from /lib/sheets
- Pattern: `src/features/[feature]/api/route.ts`

---

## IMPORT PATTERNS & BEST PRACTICES

### ✅ Safe Patterns (Currently Used)

Within features:
```typescript
import { Account } from './types'
import { getAccounts } from './queries'
import { useAccounts } from './hooks'
import { AccountsPage } from './ui'
```

From shared libs:
```typescript
import { formatVND } from '@/lib/utils/currency'
import { getAccounts } from '@/lib/sheets/accounts'
import { BudgetItem } from '@/lib/types/budget'
import { AI_MODELS } from '@/lib/ai/providers'
```

### ⚠️ Type-Only Cross-Feature (Currently Used)

```typescript
// ✅ SAFE: Only types/interfaces
import { Transaction } from '@/features/transactions/model/types'

// ❌ NOT ALLOWED: Would create circular dependencies
// import { useTransactions } from '@/features/transactions/model/hooks'
```

### ❌ Anti-Patterns (AVOID)

```typescript
// ❌ Cross-feature query imports
import { getAccounts } from '@/features/accounts/model/queries'

// ❌ Cross-feature hook imports
import { useAccounts } from '@/features/accounts/model/hooks'

// ❌ Cross-feature component imports
import { AccountsPage } from '@/features/accounts/ui'

// ❌ Deep path imports
import { something } from '@/features/accounts/ui/some/nested/path'
```

---

## FILES TO MOVE TO MONOREPO LIBS

### Phase 1: High Priority (Core Types) 
```
Current:  apps/wealth-management/src/lib/types/*
Target:   libs/common/types/

Files to move:
├── account.ts
├── budget.ts
├── transaction.ts
├── goals.ts
├── loan.ts
├── notification.ts
└── settings.ts
```

### Phase 2: High Priority (Data Access Layer)
```
Current:  apps/wealth-management/src/lib/sheets/*
Target:   libs/sheets/

Files to move:
├── client.ts (core)
├── auth.ts
├── mappers.ts
├── accounts.ts
├── budget.ts
├── transactions.ts
├── loans.ts
├── goals.ts
├── categories.ts
├── notifications.ts
└── *.test.ts
```

### Phase 3: Medium Priority (Utilities)
```
Current:  apps/wealth-management/src/lib/utils/*
Target:   libs/common/utils/

Files to move:
├── currency.ts
├── date.ts
├── validation.ts
└── api-error-handler.ts
```

### Phase 4: Medium Priority (Constants)
```
Current:  apps/wealth-management/src/lib/constants/*
Target:   libs/common/constants/

Files to move:
├── categories.ts
├── tags.ts
└── navigation.ts
```

### Phase 5: Medium Priority (AI & Services)
```
Current:  apps/wealth-management/src/lib/ai/*
Target:   libs/ai/

Current:  apps/wealth-management/src/lib/services/*
Target:   libs/services/
```

### Phase 6: Low Priority (Keep in App for Now)
- Feature-specific hooks
- Feature-specific UI components
- Feature-specific business logic (queries, mutations)

---

## REFACTORING ROADMAP FOR MONOREPO INTEGRATION

### Step 1: Create New Lib Structure
```bash
mkdir -p libs/common/{types,utils,constants}
mkdir -p libs/sheets
mkdir -p libs/ai
mkdir -p libs/services
```

### Step 2: Move & Update Imports (Phase by Phase)
1. Move types: `libs/common/types/`
2. Move sheets: `libs/sheets/`
3. Move utils: `libs/common/utils/`
4. Move constants: `libs/common/constants/`
5. Move services: `libs/services/`, `libs/ai/`

Update all imports in:
- `/apps/wealth-management/src/features/*/`
- `/apps/wealth-management/src/app/api/`

### Step 3: Update Path Aliases
Add to `tsconfig.base.json`:
```json
{
  "compilerOptions": {
    "paths": {
      "@/lib/types/*": ["libs/common/types/*"],
      "@/lib/sheets/*": ["libs/sheets/*"],
      "@/lib/utils/*": ["libs/common/utils/*"],
      "@/lib/constants/*": ["libs/common/constants/*"],
      "@/lib/ai/*": ["libs/ai/*"],
      "@/lib/services/*": ["libs/services/*"]
    }
  }
}
```

### Step 4: Create Barrel Exports
```typescript
// libs/common/types/index.ts
export * from './account';
export * from './budget';
export * from './transaction';
// ... etc

// libs/sheets/index.ts
export * from './accounts';
export * from './budget';
export * from './transactions';
// ... etc
```

### Step 5: Verify Builds & Tests
- Nx dependency graph clean
- All imports resolve correctly
- TypeScript compilation passes
- All tests pass

---

## CIRCULAR DEPENDENCY ANALYSIS

### Current Status: SAFE ✅
Only 2 cross-feature imports exist, both are type-only:

1. **Accounts → Transactions**
   ```typescript
   import { Transaction } from '@/features/transactions/model/types'
   ```
   Used in: account-trend-sparkline.tsx, credit-card-summary-ai.tsx, page.tsx
   Why safe: Only type imports, no function/hook imports

2. **Budget → Transactions**
   ```typescript
   import { Transaction } from '@/features/transactions/model/types'
   ```
   Used in: budget/ui/page.tsx, category-detail-view.tsx
   Why safe: Only type imports, no function/hook imports

**Transactions has NO cross-feature imports** ✅

### Risk Mitigation Rules
1. ✅ **Type imports ONLY** between features
2. ✅ Move shared types to `libs/common/types/`
3. ❌ **NEVER** import hooks/queries/components across features
4. ❌ **NEVER** create bidirectional imports
5. ❌ **NEVER** import from ui/ layer across features

---

## SUMMARY STATISTICS

| Metric | Value |
|--------|-------|
| Total Features | 8 |
| Total Files | 112 |
| Average Files per Feature | 14 |
| Features with API routes | 4 (Accounts, Budget, Transactions, Investments) |
| Features with hooks | 7 (all except Investments-light) |
| API routes total | 4 main + 2 nested = 6 |
| Cross-feature imports | 2 (both type-only) |
| Shared lib directories | 8 |
| Circular dependencies | 0 (SAFE) |
| Model-layer files | ~56 |
| UI-layer files | ~48 |
| API-layer files | 6 |
| Lines of model code | ~2,500+ |
| Lines of UI code | ~3,500+ |

---

## KEY INSIGHTS

### Strengths ✅
1. **Clean separation**: Features are well-isolated
2. **Consistent patterns**: All follow FSD model/ui/api structure
3. **Type safety**: Only safe cross-feature imports
4. **No circular dependencies**: Dependency graph is acyclic
5. **Testable**: Model layer is easily unit testable
6. **Scalable**: Easy to add new features following same pattern

### Areas for Improvement 🔄
1. **Shared types duplication**: Types in /lib/types could be extracted to libs/
2. **Data layer coupling**: /lib/sheets could become reusable library
3. **Feature expansion**: As features grow, might benefit from sub-slicing
4. **API organization**: Currently scattered, could centralize API layer

### Monorepo Benefits 📦
1. **Code reuse**: Shared libs across multiple apps
2. **Dependency management**: Nx provides better tracking
3. **Build optimization**: Nx task caching and distributed builds
4. **Type safety**: Shared types at workspace level
5. **Testing**: Unified test infrastructure

