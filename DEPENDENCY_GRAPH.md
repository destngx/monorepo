# Dependency Graph & Architectural Visualization

## Feature Dependency Tree

```
apps/wealth-management/src/
├── features/
│   ├── accounts/ (16 files)
│   │   ├── model/
│   │   │   ├── types.ts (Account, AccountType, Currency)
│   │   │   ├── queries.ts (getAccounts, getAccountById, getAccountsByType)
│   │   │   ├── mutations.ts (create/update/deleteAccount)
│   │   │   ├── hooks.ts (useAccounts, useAccountById)
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── page.tsx ⬅️ IMPORTS: Transaction (from transactions)
│   │   │   ├── account-review-ai.tsx
│   │   │   ├── account-trend-sparkline.tsx ⬅️ IMPORTS: Transaction
│   │   │   ├── credit-card-summary-ai.tsx ⬅️ IMPORTS: Transaction
│   │   │   ├── efficiency-chart.tsx
│   │   │   ├── credit/
│   │   │   │   ├── due-date-countdown.tsx
│   │   │   │   ├── statement-cycle-bar.tsx
│   │   │   │   └── utilization-ring.tsx
│   │   │   └── index.ts
│   │   └── api/
│   │       └── route.ts (GET /api/accounts)
│   │
│   ├── budget/ (13 files)
│   │   ├── model/
│   │   │   ├── types.ts (BudgetItem)
│   │   │   ├── queries.ts
│   │   │   ├── mutations.ts
│   │   │   ├── hooks.ts (useBudgetItems)
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── page.tsx ⬅️ IMPORTS: Transaction (from transactions)
│   │   │   ├── budget-overview-view.tsx
│   │   │   ├── category-detail-view.tsx ⬅️ IMPORTS: Transaction
│   │   │   ├── budget-review-ai.tsx
│   │   │   ├── ai-budget-advisor-view.tsx
│   │   │   └── index.ts
│   │   └── api/
│   │       └── route.ts (GET /api/budget)
│   │
│   ├── transactions/ (14 files) [CORE DATA]
│   │   ├── model/
│   │   │   ├── types.ts (Transaction, TransactionType) ⬅️ EXPORTED
│   │   │   ├── queries.ts
│   │   │   ├── mutations.ts
│   │   │   ├── hooks.ts
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── page.tsx
│   │   │   ├── transaction-form.tsx
│   │   │   ├── transaction-table.tsx
│   │   │   ├── transaction-filters.tsx
│   │   │   ├── notification-processor.tsx
│   │   │   ├── transaction-review-ai.tsx
│   │   │   └── index.ts
│   │   └── api/
│   │       └── route.ts (GET/POST /api/transactions)
│   │
│   ├── chat/ (17 files) [STANDALONE]
│   │   ├── model/
│   │   │   ├── types.ts
│   │   │   ├── queries.ts
│   │   │   ├── mutations.ts
│   │   │   ├── hooks.ts
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── ai-fab.tsx
│   │   │   ├── ai-drawer.tsx
│   │   │   ├── chat-widget.tsx
│   │   │   ├── chat-interface.tsx
│   │   │   ├── chat-container.tsx
│   │   │   ├── chat-messages.tsx
│   │   │   ├── chat-input.tsx
│   │   │   ├── model-switcher.tsx
│   │   │   ├── ai-insight-card.tsx
│   │   │   ├── ai-context-provider.tsx
│   │   │   └── index.ts
│   │   └── NO API ROUTE (uses server actions)
│   │
│   ├── goals/ (14 files) [STANDALONE]
│   │   ├── model/
│   │   │   ├── types.ts (Goal, GoalType, GoalProjection, etc.)
│   │   │   ├── queries.ts
│   │   │   ├── mutations.ts
│   │   │   ├── hooks.ts
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── page.tsx
│   │   │   ├── goal-card.tsx
│   │   │   ├── goal-detail-page.tsx
│   │   │   ├── goal-detail-chart.tsx
│   │   │   ├── new-goal-page.tsx
│   │   │   ├── create-goal-flow.tsx
│   │   │   ├── ai-insights-panel.tsx
│   │   │   ├── ai-summary-card.tsx
│   │   │   └── index.ts
│   │   └── NO API ROUTE
│   │
│   ├── loans/ (11 files) [STANDALONE]
│   │   ├── model/
│   │   │   ├── types.ts (Loan)
│   │   │   ├── queries.ts
│   │   │   ├── mutations.ts
│   │   │   ├── hooks.ts
│   │   │   ├── server-queries.ts
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── page.tsx
│   │   │   ├── loan-list.tsx
│   │   │   ├── loan-summary.tsx
│   │   │   ├── loan-review-ai.tsx
│   │   │   └── index.ts
│   │   └── NO API ROUTE
│   │
│   ├── investments/ (10 files) [STANDALONE]
│   │   ├── model/
│   │   │   ├── types.ts (Investment, PortfolioAnalysis)
│   │   │   ├── queries.ts
│   │   │   ├── mutations.ts
│   │   │   ├── hooks.ts
│   │   │   └── index.ts
│   │   ├── ui/
│   │   │   ├── page.tsx
│   │   │   └── index.ts
│   │   └── api/
│   │       ├── assets/route.ts (GET /api/investments/assets)
│   │       └── prices/route.ts (GET /api/investments/prices)
│   │
│   └── settings/ (7 files) [STANDALONE]
│       ├── model/
│       │   ├── types.ts (Settings)
│       │   ├── queries.ts
│       │   ├── mutations.ts
│       │   ├── hooks.ts
│       │   └── index.ts
│       ├── ui/
│       │   ├── page.tsx
│       │   └── index.ts
│       └── NO API ROUTE
│
└── lib/
    ├── types/
    │   ├── account.ts
    │   ├── budget.ts
    │   ├── category.ts
    │   ├── goals.ts
    │   ├── loan.ts
    │   ├── notification.ts
    │   ├── settings.ts
    │   └── transaction.ts
    ├── sheets/ [DATA ACCESS LAYER]
    │   ├── client.ts (Google Sheets auth)
    │   ├── auth.ts
    │   ├── mappers.ts
    │   ├── accounts.ts (reads from Google Sheets)
    │   ├── budget.ts
    │   ├── transactions.ts
    │   ├── loans.ts
    │   ├── goals.ts
    │   ├── categories.ts
    │   ├── notifications.ts
    │   └── *.test.ts
    ├── ai/
    │   └── providers.ts (AI_MODELS definitions)
    ├── services/
    │   └── price-service.ts
    ├── utils/
    │   ├── validation.ts
    │   ├── date.ts
    │   ├── currency.ts
    │   ├── api-error-handler.ts
    │   └── utils.ts
    ├── constants/
    │   ├── categories.ts
    │   ├── tags.ts
    │   └── navigation.ts
    └── utils.ts
```

---

## Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Feature Components (UI Layer)                     │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │ Accounts   │  │  Budget    │  │  Loans     │  │  Investments│  │
│  │ Components │  │ Components │  │ Components │  │ Components │   │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘   │
│        │                │              │               │            │
└────────┼────────────────┼──────────────┼───────────────┼────────────┘
         │                │              │               │
         │   ┌────────────┴──────────────┴───────────────┴────┐
         │   │                                                 │
┌────────▼───▼─────────────────────────────────────────────────────────┐
│                    Model Layer (Hooks, Queries)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ useAccounts  │  │ useBudgetItems│  │ useGoals   │              │
│  │ useTransact  │  │ getBudgetItems│  │ useLoans   │              │
│  │ useChat      │  │ getTotalSpent │  │ useAssets  │              │
│  └──────┬───────┘  └───────┬──────┘  └──────┬──────┘              │
│         │                  │                │                      │
└─────────┼──────────────────┼────────────────┼──────────────────────┘
          │                  │                │
          │                  ▼                │
          │          ┌────────────────┐      │
          │          │  Transactions  │      │
          │          │  Model/Hooks   │      │
          │          │ (CORE DATA)    │      │
          │          └────────┬───────┘      │
          │                   │              │
          └───────────────────┼──────────────┘
                              │
┌─────────────────────────────▼──────────────────────────────────────┐
│                      Data Access Layer                              │
│                    (@/lib/sheets/*)                                │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────────┐  │
│  │ accounts   │ │ budget     │ │ transactions│ │ loans        │  │
│  └─────┬──────┘ └──────┬─────┘ └──────┬─────┘ └──────┬───────┘  │
│        │               │              │              │            │
│        └───────────────┼──────────────┼──────────────┘            │
│                        │              │                           │
│        ┌───────────────▼──────────────▼─────┐                    │
│        │  sheets/client.ts                  │                    │
│        │  (Google Sheets API Client)        │                    │
│        └───────────────┬──────────────────┬─┘                    │
└────────────────────────┼──────────────────┼──────────────────────┘
                         │                  │
                    ┌────▼──────────────────▼───┐
                    │  Google Sheets API        │
                    │  External Data Source     │
                    └──────────────────────────┘
```

---

## Import Dependency Flow

### Safe Imports ✅

```
Feature UI Components
          ↓
Feature Model (hooks/queries/types)
          ↓
Shared Libraries (@/lib/*)
          ↓
External APIs & Services
```

### Current Cross-Feature Imports (Type-Only) ⚠️

```
accounts/ui/page.tsx ──────→ Transaction type from transactions/model/types
       ↓
transactions/model/queries ──→ Google Sheets API

budget/ui/page.tsx ────────→ Transaction type from transactions/model/types
       ↓
transactions/model/queries ──→ Google Sheets API
```

---

## Monorepo Integration Structure (Target)

```
monorepo/
├── apps/
│   ├── wealth-management/ ← Main app (features stay here)
│   │   └── src/
│   │       ├── features/
│   │       │   ├── accounts/
│   │       │   ├── budget/
│   │       │   ├── transactions/
│   │       │   ├── chat/
│   │       │   ├── goals/
│   │       │   ├── loans/
│   │       │   ├── investments/
│   │       │   └── settings/
│   │       └── app/ (pages, layouts)
│   └── [other apps...]
│
├── libs/ ← MOVE HERE (Shared across apps)
│   ├── common/
│   │   ├── types/ ← Move from app/lib/types
│   │   │   ├── account.ts
│   │   │   ├── budget.ts
│   │   │   ├── transaction.ts
│   │   │   ├── goals.ts
│   │   │   ├── loan.ts
│   │   │   ├── notification.ts
│   │   │   ├── settings.ts
│   │   │   └── index.ts
│   │   ├── utils/ ← Move from app/lib/utils
│   │   │   ├── currency.ts
│   │   │   ├── date.ts
│   │   │   ├── validation.ts
│   │   │   ├── api-error-handler.ts
│   │   │   └── index.ts
│   │   └── constants/ ← Move from app/lib/constants
│   │       ├── categories.ts
│   │       ├── tags.ts
│   │       ├── navigation.ts
│   │       └── index.ts
│   │
│   ├── sheets/ ← Move from app/lib/sheets
│   │   ├── client.ts
│   │   ├── auth.ts
│   │   ├── mappers.ts
│   │   ├── accounts.ts
│   │   ├── budget.ts
│   │   ├── transactions.ts
│   │   ├── loans.ts
│   │   ├── goals.ts
│   │   ├── categories.ts
│   │   ├── notifications.ts
│   │   └── index.ts
│   │
│   ├── ai/ ← Move from app/lib/ai
│   │   ├── providers.ts
│   │   └── index.ts
│   │
│   └── services/ ← Move from app/lib/services
│       ├── price-service.ts
│       └── index.ts
│
└── tsconfig.base.json (updated with @/lib/* paths)
```

---

## Migration Path: Path Aliases

### Current (Pre-Migration)
```json
// tsconfig.json (wealth-management app)
{
  "compilerOptions": {
    "paths": {
      "@/*": ["src/*"]
    }
  }
}
```

### After Phase 1 (Monorepo Integration)
```json
// tsconfig.base.json (workspace root)
{
  "compilerOptions": {
    "paths": {
      "@/lib/types/*": ["libs/common/types/*"],
      "@/lib/sheets/*": ["libs/sheets/*"],
      "@/lib/utils/*": ["libs/common/utils/*"],
      "@/lib/constants/*": ["libs/common/constants/*"],
      "@/lib/ai/*": ["libs/ai/*"],
      "@/lib/services/*": ["libs/services/*"],
      "@/features/*": ["apps/wealth-management/src/features/*"]
    }
  }
}
```

---

## Circular Dependency Risk Map

```
✅ SAFE: Type imports only
┌─────────────────────┐
│ accounts/ui/page    │
│   ↓ (type import)   │
│ transactions/types  │
└─────────────────────┘

✅ SAFE: No bidirectional imports
┌──────────────────┐      ┌──────────────┐
│ Accounts Feature │─────→│ Transactions │
│                  │      │   (NO BACK)  │
└──────────────────┘      └──────────────┘

✅ SAFE: Model layer is isolated
┌────────────────┐
│ accounts/model │
│  (types only)  │─────→ @/lib/sheets
└────────────────┘

❌ WOULD BE UNSAFE: Hooks across features
┌────────────────────────────────────┐
│ DON'T: import useAccounts from     │ ← PREVENTS circular deps
│ features/accounts in other feature │
└────────────────────────────────────┘
```

---

## API Route Organization

### Current Structure
```
src/features/
├── accounts/api/route.ts       (GET /api/accounts)
├── budget/api/route.ts         (GET /api/budget)
├── transactions/api/route.ts   (GET/POST /api/transactions)
├── investments/api/
│   ├── assets/route.ts         (GET /api/investments/assets)
│   └── prices/route.ts         (GET /api/investments/prices)
└── [goals, loans, settings]    (no API routes)
```

### Alternative (Future Consideration)
```
src/app/api/
├── accounts/route.ts
├── budget/route.ts
├── transactions/route.ts
├── investments/
│   ├── assets/route.ts
│   └── prices/route.ts
└── [future endpoints]

Benefits:
- Centralized API organization
- Easier to add middleware
- Better routing consistency
```

---

## Feature Maturity Matrix

```
Feature          │ API Routes │ Hooks │ Tests │ AI Integration │ Complexity
─────────────────┼────────────┼───────┼───────┼────────────────┼──────────
accounts         │    ✅      │  ✅   │  ❌   │      ✅        │  Medium
budget           │    ✅      │  ✅   │  ❌   │      ✅        │  Medium
transactions     │    ✅      │  ✅   │  ✅   │      ✅        │  Medium
chat             │    ❌      │  ✅   │  ❌   │      ✅        │  High
goals            │    ❌      │  ✅   │  ❌   │      ✅        │  Medium
loans            │    ❌      │  ✅   │  ❌   │      ✅        │  Low
investments      │    ✅      │  ✅   │  ❌   │      ❌        │  Medium
settings         │    ❌      │  ✅   │  ❌   │      ❌        │  Low
```

