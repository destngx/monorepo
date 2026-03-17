# FSD Architecture - Visual Reference

## Dependency Flow

```
┌────────────────────────────────────────────────────┐
│  APP ROUTES (src/app/)                            │
│  - pages                                           │
│  - layouts                                        │
│  Import from: features/*/ui, shared, core        │
└────────────────────────────────────────────────────┘
                        ▲
                        │
      ┌─────────────────┼─────────────────┐
      │                 │                 │
┌─────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐
│ FEATURES   │  │ FEATURES    │  │ FEATURES   │
│ accounts   │  │ budget      │  │ goals      │
└─────┬──────┘  └──────┬──────┘  └──────┬──────┘
      │                 │                 │
      │   ┌─────────────┴─────────────┐   │
      │   │                           │   │
      ▼   ▼                           ▼   ▼
┌────────────────────────────────────────────────┐
│ SHARED INFRASTRUCTURE (src/shared/)            │
│ - ui/          (Design system)                 │
│ - lib/ai/      (Unified AI service)            │
│ - lib/validation/ (Zod schemas)                │
│ - lib/persistence/ (Repository pattern)        │
│ - hooks/       (Shared React hooks)            │
│ - constants/   (App constants)                 │
└────────────────────────────────────────────────┘
                        ▲
                        │
┌───────────────────────┴──────────────────────┐
│ CORE/KERNEL (src/core/)                      │
│ - auth/        (Authentication)              │
│ - cache/       (Caching logic)               │
│ - config/      (App configuration)           │
│ - middleware/  (Server middleware)           │
└────────────────────────────────────────────────┘
                        ▲
                        │
            ┌───────────┴───────────┐
            │                       │
    ┌───────┴────────┐   ┌─────────┴──────┐
    │ External APIs  │   │  Databases     │
    │ (Google Sheets,│   │  (Cache layer) │
    │  LLM, etc)     │   │                │
    └────────────────┘   └────────────────┘
```

## Feature Slice Anatomy

```
src/features/accounts/
├── api/                           ← Server endpoints
│   ├── route.ts                   (GET /api/accounts, POST, etc)
│   ├── actions.ts                 (Server actions)
│   └── schemas.ts                 (Request/response validation)
│
├── ui/                            ← Client components
│   ├── page.tsx                   (Page wrapper)
│   ├── account-card.tsx           (Components)
│   ├── account-list.tsx
│   └── index.ts                   (Public exports)
│
├── model/                         ← Business logic & types
│   ├── types.ts                   (Domain types)
│   ├── schema.ts                  (Zod schemas)
│   ├── queries.ts                 (Read operations)
│   ├── mutations.ts               (Write operations)
│   ├── hooks.ts                   (React hooks)
│   └── index.ts                   (Public API)
│
├── lib/                           ← Feature utilities
│   ├── utils.ts
│   └── constants.ts
│
├── __tests__/                     ← Feature tests
│   ├── queries.test.ts
│   ├── mutations.test.ts
│   └── api.test.ts
│
└── index.ts                       ← Feature entry point
```

## What Goes Where?

### ✅ FEATURES/ Directory

**Purpose**: Isolated, self-contained feature modules

**Guidelines**:
- One feature per folder
- Can only import from: self, shared/, core/
- ❌ Cannot import from other features
- Public exports only via index.ts

**Features**:
```
features/
├── accounts/          (Bank/crypto accounts)
├── transactions/      (Transaction history)
├── budget/            (Budget tracking)
├── goals/             (Financial goals)
├── loans/             (Debt tracking)
├── investments/       (Investment portfolio)
├── chat/              (AI assistant)
└── settings/          (User settings)
```

### ✅ SHARED/ Directory

**Purpose**: Cross-cutting concerns, shared by all features

```
shared/
├── ui/
│   ├── button.tsx
│   ├── card.tsx
│   ├── dialog.tsx
│   └── ...
│
├── lib/
│   ├── ai/                        # Unified AI service
│   │   ├── service.ts
│   │   ├── prompts/
│   │   ├── models.ts
│   │   └── index.ts
│   │
│   ├── persistence/               # Repository pattern
│   │   ├── repository.ts
│   │   ├── adapters/
│   │   └── index.ts
│   │
│   ├── validation/                # Schemas
│   │   ├── schemas.ts
│   │   └── index.ts
│   │
│   ├── services/
│   │   ├── exchange-rate.ts
│   │   ├── market-data.ts
│   │   └── index.ts
│   │
│   ├── utils/
│   │   ├── currency.ts
│   │   ├── date.ts
│   │   └── index.ts
│   │
│   └── types/                     # Shared types
│       └── index.ts
│
├── hooks/
│   ├── use-query.ts
│   ├── use-mutation.ts
│   └── index.ts
│
├── constants/
│   ├── navigation.ts
│   ├── categories.ts
│   └── index.ts
│
└── api/
    ├── middleware.ts
    └── error-handler.ts
```

### ✅ CORE/ Directory

**Purpose**: Application core/kernel (rarely changes)

```
core/
├── auth/                          # Auth logic
│   ├── session.ts
│   └── guards.ts
│
├── cache/                         # Caching
│   ├── redis.ts
│   └── index.ts
│
├── config/                        # Configuration
│   └── index.ts
│
└── middleware/                    # Server middleware
    └── index.ts
```

### ✅ APP/ Directory

**Purpose**: Next.js routes & page layouts

```
app/
├── layout.tsx                     # Root layout
├── page.tsx                       # Home page
├── (auth)/                        # Auth group
├── (dashboard)/                   # Dashboard group
│   ├── layout.tsx
│   ├── page.tsx                   (imports features/dashboard/ui/page)
│   ├── accounts/
│   │   └── page.tsx               (imports features/accounts/ui/page)
│   ├── budget/
│   │   └── page.tsx               (imports features/budget/ui/page)
│   └── ...
│
├── api/
│   ├── accounts/
│   │   └── route.ts               (delegates to features/accounts/api)
│   ├── budget/
│   │   └── route.ts
│   └── ...
│
└── error.tsx
```

---

## Import Pattern Examples

### ✅ CORRECT

```ts
// In features/accounts/ui/account-card.tsx
import { Button } from '@/shared/ui/button';           // ✅ shared
import { formatCurrency } from '@/shared/lib/utils';   // ✅ shared
import { Account } from '../model/types';              // ✅ self
import { useAccount } from '../model/hooks';           // ✅ self
```

```ts
// In features/accounts/model/queries.ts
import { Repository } from '@/shared/lib/persistence';  // ✅ shared
import { Account } from './types';                      // ✅ self
```

```ts
// In features/accounts/api/route.ts
import { getAccounts } from '../model/queries';         // ✅ self
import { AccountSchema } from '@/shared/lib/validation'; // ✅ shared
```

### ❌ INCORRECT

```ts
// In features/accounts/ui/account-card.tsx
import { Budget } from '@/features/budget/model/types'; // ❌ other feature
import { formatCurrency } from '@/lib/utils';           // ❌ old path
import { getAccounts } from '@/features/accounts/api';  // ❌ importing internal
```

---

## Layer Boundaries (Clean Architecture)

```
┌─────────────────────────────────────────────────┐
│                   ENTITIES                      │
│  (types.ts, schema.ts - Core business objects) │
│                                                 │
│  Account, Transaction, Goal, etc.              │
│                                                 │
└────────────────────┬────────────────────────────┘
                     ▲
                     │ depend only on
                     │
┌────────────────────┴────────────────────────────┐
│              USE CASES / MODELS                 │
│  (queries.ts, mutations.ts, hooks.ts)          │
│                                                 │
│  getAccounts(), createBudget(), etc.           │
│  Business logic, validation, calculations      │
│                                                 │
└────────────────────┬────────────────────────────┘
                     ▲
                     │ depend only on
                     │
┌────────────────────┴────────────────────────────┐
│          INTERFACE ADAPTERS                     │
│  (api/, Repository, validation/)               │
│                                                 │
│  HTTP handlers, data mapping, schema adapters  │
│                                                 │
└────────────────────┬────────────────────────────┘
                     ▲
                     │ depend only on
                     │
┌────────────────────┴────────────────────────────┐
│        FRAMEWORKS & DRIVERS                     │
│  (React, Next.js, Google Sheets, LLMs)        │
│                                                 │
│  External libraries, API clients                │
│                                                 │
└─────────────────────────────────────────────────┘

Rule: Inner circles ❌ depend on outer circles
      Outer circles ✅ depend on inner circles
```

---

## Decision Tree: Where Does This Code Go?

```
Is it a React component?
├─ Yes → Is it a design system component (Button, Card, Badge)?
│  ├─ Yes → shared/ui/
│  └─ No → Is it feature-specific?
│     ├─ Yes → features/X/ui/
│     └─ No → Can it be shared? → shared/ui/ or features/X/ui/
│
└─ No → Is it business logic or use case?
   ├─ Yes → Is it feature-specific?
   │  ├─ Yes → features/X/model/queries.ts or mutations.ts
   │  └─ No → shared/lib/
   │
   └─ No → Is it a type or interface?
      ├─ Yes → features/X/model/types.ts (if feature-specific)
      │        or shared/lib/types/ (if shared)
      │
      └─ No → Is it a utility function?
         ├─ Yes → features/X/lib/utils.ts (if feature-specific)
         │        or shared/lib/utils/ (if shared)
         │
         └─ No → Is it an API handler?
            └─ Yes → features/X/api/route.ts
```

---

## Checklist: Feature Implementation

When creating a new feature or refactoring existing one:

- [ ] **Create folder structure**
  - [ ] `api/` with `route.ts`
  - [ ] `ui/` with page and components
  - [ ] `model/` with types, queries, mutations
  - [ ] `lib/` for feature utilities
  - [ ] `__tests__/` for tests
  - [ ] `index.ts` for public exports

- [ ] **Define types** in `model/types.ts`
  - [ ] Core domain types
  - [ ] Aggregates with business logic
  - [ ] DTO types for API

- [ ] **Create schemas** in `model/schema.ts`
  - [ ] Input validation (Zod)
  - [ ] Output schemas
  - [ ] Error types

- [ ] **Implement queries** in `model/queries.ts`
  - [ ] Use Repository for data access
  - [ ] Apply business logic
  - [ ] Return domain types

- [ ] **Implement mutations** in `model/mutations.ts`
  - [ ] Validate input
  - [ ] Execute business logic
  - [ ] Persist via Repository
  - [ ] Trigger revalidation

- [ ] **Create API endpoints** in `api/route.ts`
  - [ ] Thin controllers
  - [ ] Delegate to model/
  - [ ] Handle errors
  - [ ] Return appropriate status codes

- [ ] **Create UI components** in `ui/`
  - [ ] Import only from model/, shared/, core/
  - [ ] Use hooks for data fetching
  - [ ] Compose from shared/ui components

- [ ] **Write tests** in `__tests__/`
  - [ ] Unit tests for model/
  - [ ] Integration tests for api/
  - [ ] Component tests for ui/

- [ ] **Export public API** in `index.ts`
  - [ ] Only export what's needed
  - [ ] Hide implementation details

---

## SOLID Principles Checklist

### Single Responsibility Principle ✓
```ts
// ✓ Each file has one reason to change
features/accounts/model/types.ts        # Only: define Account type
features/accounts/model/queries.ts      # Only: read operations
features/accounts/model/mutations.ts    # Only: write operations
features/accounts/api/route.ts          # Only: HTTP handling
```

### Open/Closed Principle ✓
```ts
// ✓ Open for extension via composition, closed for modification
export class AccountAggregate {
  constructor(
    private account: Account,
    private validator: AccountValidator  // Injected
  ) {}
}
```

### Liskov Substitution Principle ✓
```ts
// ✓ Repositories are interchangeable
const repo: Repository<Account> = environment.isDev
  ? new MemoryAccountRepository()
  : new SheetsAccountRepository();
```

### Interface Segregation Principle ✓
```ts
// ✓ Components depend on specific interfaces, not monolithic types
export function useAccounts() {
  // Only returns what's needed
  return useQuery('accounts', {
    data: Account[];
    isLoading: boolean;
    error: Error | null;
  });
}
```

### Dependency Inversion Principle ✓
```ts
// ✓ Depend on abstractions, not concrete implementations
// Not: import { SheetsAccountRepository } from '@/lib/sheets'
// Yes: import { Repository } from '@/shared/lib/persistence'
const accounts = await Repository.accounts.findAll();
```

---

## Performance Optimization Notes

### Code Splitting
Each feature can be code-split via dynamic import:
```ts
// app/accounts/page.tsx
const AccountsPage = dynamic(
  () => import('@/features/accounts/ui/page'),
  { loading: () => <Skeleton /> }
);
```

### Data Fetching Strategy
```ts
// features/accounts/model/queries.ts
export async function getAccounts() {
  // Uses unstable_cache for automatic revalidation
  return cache(
    () => Repository.accounts.findAll(),
    ['accounts'],
    { revalidate: 60 }
  );
}
```

### Bundle Analysis
```bash
# Check bundle size per feature
npx next-bundle-analyzer
```

---

## Troubleshooting

### Circular Dependency
```
Error: Circular dependency detected in features/accounts
```

**Fix**: Move shared logic to `shared/lib/`
```ts
// ❌ features/accounts/lib → features/budget/lib
// ✅ features/accounts/lib → shared/lib/utils
```

### Missing Import
```
Module not found: Can't resolve '@/features/budget/model'
```

**Fix**: Only import from public exports
```ts
// ❌ import { getBalance } from '@/features/budget/model/queries'
// ✅ import { useBalance } from '@/features/budget'
```

### Performance Regression
```
LCP increased after refactor
```

**Causes**:
- Unnecessary re-renders
- Missing memoization
- Over-fetching data

**Solution**: Profile with DevTools, optimize critical paths

---

## Migration Progress Tracker

Use this to track your refactoring progress:

```md
## Migration Status

### Phase 1: Infrastructure (Week 1)
- [ ] shared/lib/persistence/ - Repository abstraction
- [ ] shared/lib/ai/ - Unified AI service
- [ ] shared/lib/validation/ - Schemas
- [ ] ESLint rules updated

### Phase 2: Feature Migration (Weeks 2-3)
- [ ] features/settings/
- [ ] features/loans/
- [ ] features/accounts/
- [ ] features/transactions/
- [ ] features/budget/
- [ ] features/goals/
- [ ] features/investments/
- [ ] features/chat/

### Phase 3: Integration (Weeks 4-5)
- [ ] app/ routes updated
- [ ] Old lib/ removed
- [ ] All imports migrated
- [ ] Tests passing
- [ ] Build succeeds
- [ ] Performance validated
```

---

## Conclusion

This FSD architecture provides:
- **Clear module boundaries** - no accidental dependencies
- **Testability** - each feature is independently testable
- **Scalability** - new features don't affect existing ones
- **Team workflow** - features can be developed in parallel
- **Maintainability** - code is organized by feature, not layer

Follow the patterns and guidelines in this reference to keep your codebase clean and scalable!
