# Wealth Management - FSD Refactoring Plan

## Executive Summary

This document outlines a comprehensive refactoring to migrate from a functional/layer-based architecture to **Feature-Sliced Design (FSD)** with **Vertical Slices** and **Clean Architecture** principles.

**Current State**: 173 TS/TSX files, mixed concerns, tight coupling between layers  
**Target State**: 9 isolated feature modules, clear dependency graph, testable business logic  
**Estimated Timeline**: 3-5 weeks (aggressive full refactor)  
**Risk Level**: Medium-High (extensive file reorganization)

---

## Part 1: Current Architecture Analysis

### Structure Overview

```
src/
├── app/                    # Route handlers + API routes (mixed concerns)
├── components/             # UI only (but imports from lib/sheets directly)
├── hooks/                  # Custom React hooks (3 files)
└── lib/                    # Everything else (types, services, utils, data access)
    ├── ai/                 # AI prompt engineering
    ├── sheets/            # Google Sheets data layer (tightly coupled)
    ├── services/          # Domain services
    ├── types/             # TypeScript types
    └── utils/             # Utilities
```

### Business Domains Identified

| Domain           | Pages | Components | API Routes | Data Layer      | Status        |
| ---------------- | ----- | ---------- | ---------- | --------------- | ------------- |
| **Accounts**     | 1     | 5          | 1          | accounts.ts     | Core          |
| **Transactions** | 1     | 5          | 1          | transactions.ts | Core          |
| **Budget**       | 1     | 4          | 1          | budget.ts       | Core          |
| **Goals**        | 3     | 6          | 1          | goals.ts        | Core          |
| **Loans**        | 1     | 3          | 1          | loans.ts        | Core          |
| **Investments**  | 1     | 0          | 2          | (services)      | Core          |
| **Credit Cards** | 2     | 2          | 0          | (accounts)      | Secondary     |
| **Chat/AI**      | 1     | 12         | 14         | (distributed)   | Cross-cutting |
| **Dashboard**    | 1     | 15         | 0          | (mixed)         | Aggregate     |

### Key Pain Points

#### 1. **Layer Leakage**

```ts
// ❌ Component directly accessing persistence layer
import { getAccounts } from '@/lib/sheets/accounts';

export function AccountsSummary() {
  const accounts = await getAccounts();  // Direct data access
  return <div>...</div>;
}
```

**Impact**: Can't change persistence without touching UI. Hard to test.

#### 2. **AI Scattered Across Codebase**

- 12 AI API routes in `/api/ai/`
- AI components in each domain folder
- Prompts in `/lib/ai/prompts/`
- Orchestration in `/lib/ai/core/`

**Impact**: Duplicated logic, hard to maintain consistent AI behavior.

#### 3. **No Business Logic Layer**

- Validation, calculations, transformations happen in components or API routes
- No shared business logic between API and UI
- Difficult to test business rules

**Impact**: Logic duplication, inconsistency between client/server.

#### 4. **Type Explosion**

- Types scattered in `/lib/types/`
- Each domain has models but no domain-specific types file
- No schema validation at API boundaries

**Impact**: Types not colocated with business logic, hard to evolve.

---

## Part 2: Target Architecture (FSD + Vertical Slices)

### Structure

```
src/
├── app/                          # Next.js app router
│   ├── layout.tsx
│   ├── page.tsx
│   └── (routes)                 # Feature route groups
│
├── features/                     # ⭐ VERTICAL SLICES (FSD)
│   ├── accounts/
│   │   ├── api/                 # API endpoints
│   │   ├── ui/                  # UI components
│   │   ├── model/               # Business logic & types
│   │   ├── hooks/               # Feature-specific hooks
│   │   └── lib/                 # Feature utilities
│   │
│   ├── transactions/
│   │   ├── api/
│   │   ├── ui/
│   │   ├── model/
│   │   └── hooks/
│   │
│   ├── budget/
│   ├── goals/
│   ├── loans/
│   ├── investments/
│   ├── chat/                    # AI Chat
│   └── settings/
│
├── shared/                       # ⭐ SHARED INFRASTRUCTURE
│   ├── api/                     # API middleware, utils
│   ├── ui/                      # Design system components
│   ├── lib/
│   │   ├── ai/                  # Unified AI service layer
│   │   ├── persistence/         # Data access abstraction (Repository pattern)
│   │   ├── services/            # Cross-cutting services
│   │   ├── utils/               # Utilities
│   │   ├── types/               # Shared types
│   │   └── validation/          # Zod schemas
│   ├── hooks/                   # Shared React hooks
│   └── constants/
│
├── core/                         # ⭐ CORE/KERNEL
│   ├── auth/                    # Authentication
│   ├── cache/                   # Caching layer
│   ├── config/                  # App config
│   └── middleware/              # Server middleware

└── widgets/                      # Page-level compositions (optional)
    └── dashboard-widget.tsx
```

### FSD Layer Definition

#### **Shared Layer** (Deepest - Most Reusable)

```
Features can import from:
- shared/ui
- shared/lib
- shared/constants
```

#### **Feature Layer** (Middle)

```
Features can import from:
- self
- ../shared
- ../core

❌ Cannot import from other features
```

#### **App/Page Layer** (Surface)

```
Pages can import from:
- features/*/ui (components only!)
- shared/
- core/
```

### Clean Architecture Principles

#### Dependency Rule

```
      ┌─────────────────────────┐
      │   UI Components         │ (Enterprise rules)
      └────────┬────────────────┘
               │ (control flow)
      ┌────────▼────────────────┐
      │   Use Cases / Models    │ (Application rules)
      └────────┬────────────────┘
               │ (control flow)
      ┌────────▼────────────────┐
      │   Repositories / Gateways │ (Interface adapters)
      └────────┬────────────────┘
               │ (control flow)
      ┌────────▼────────────────┐
      │   External Systems      │ (Google Sheets, AI SDK, etc)
      └─────────────────────────┘
```

**Rule**: Inner circles never depend on outer circles.

---

## Part 3: File Organization Template (Each Feature)

### Accounts Feature Example

```
src/features/accounts/
├── api/
│   ├── route.ts                      # API handlers
│   ├── actions.ts                    # Server actions
│   └── schemas.ts                    # Request/response validation
│
├── ui/
│   ├── accounts-page.tsx             # Page component
│   ├── accounts-summary.tsx
│   ├── account-card.tsx
│   ├── account-list.tsx
│   └── index.ts                      # Public exports
│
├── model/                            # Core business logic
│   ├── types.ts                      # Accounts domain types
│   ├── schema.ts                     # Zod validation schemas
│   ├── queries.ts                    # Data queries
│   ├── mutations.ts                  # Data mutations
│   ├── hooks.ts                      # Feature hooks (useAccounts, etc)
│   └── index.ts                      # Public API
│
├── lib/
│   ├── utils.ts                      # Feature utilities
│   └── constants.ts
│
├── __tests__/
│   ├── accounts.test.ts
│   ├── queries.test.ts
│   └── mutations.test.ts
│
└── index.ts                          # Feature public interface
```

### Inside `accounts/model/types.ts`

```ts
// ✅ Domain types (no persistence details)
export interface Account {
  id: string;
  name: string;
  type: AccountType;
  currency: Currency;
  balance: number;
}

export type AccountType = 'active' | 'savings' | 'investment' | ...;

// Extend with domain-specific business logic
export class AccountAggregate {
  constructor(private account: Account) {}

  isLowBalance(threshold: number): boolean {
    return this.account.balance < threshold;
  }

  canTransfer(amount: number): boolean {
    return this.account.balance >= amount;
  }
}
```

### Inside `accounts/model/queries.ts`

```ts
// ✅ Use case handlers (business logic)
import { Repository } from '@/shared/lib/persistence';

export async function getAccountSummary(accountId: string) {
  const account = await Repository.accounts.findById(accountId);
  if (!account) throw new NotFoundError();

  // Business logic
  return {
    ...account,
    isLowBalance: account.balance < 1000,
    health: calculateAccountHealth(account),
  };
}
```

### Inside `accounts/api/route.ts`

```ts
// ✅ Thin controllers (no business logic)
import { getAccountSummary } from '../model/queries';

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const id = searchParams.get('id');

  if (!id) return NextResponse.json({ error: 'Missing id' }, { status: 400 });

  try {
    const data = await getAccountSummary(id);
    return NextResponse.json(data);
  } catch (error) {
    if (error instanceof NotFoundError) {
      return NextResponse.json({ error: 'Not found' }, { status: 404 });
    }
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}
```

---

## Part 4: Cross-Cutting Concerns

### 1. **AI Service Layer** (Unified)

**Current**: Scattered across `/api/ai/`, components, `/lib/ai/`  
**Target**: Centralized service in `shared/lib/ai/`

```
shared/lib/ai/
├── service.ts                 # Main AI interface
├── prompts/                   # Prompt templates (organized by domain)
│   ├── account-review.ts
│   ├── budget-analysis.ts
│   ├── investment-insight.ts
│   └── index.ts
├── models.ts                  # AI response types
├── tools.ts                   # AI tool definitions
└── providers.ts               # LLM provider setup
```

**Usage in features**:

```ts
// accounts/model/queries.ts
import { AIService } from '@/shared/lib/ai';

export async function getAccountReview(account: Account) {
  return AIService.analyze('account-review', { account });
}
```

### 2. **Data Persistence (Repository Pattern)**

**Current**: Direct imports of `lib/sheets/accounts`, `lib/sheets/transactions`, etc  
**Target**: Repository abstraction

```
shared/lib/persistence/
├── repository.ts              # Abstract base
├── adapters/
│   ├── sheets-adapter.ts      # Google Sheets implementation
│   └── memory-adapter.ts      # Testing
├── types.ts
└── index.ts                   # Exports
```

**Implementation**:

```ts
// shared/lib/persistence/repository.ts
export abstract class Repository<T> {
  abstract findById(id: string): Promise<T | null>;
  abstract findAll(): Promise<T[]>;
  abstract create(data: Partial<T>): Promise<T>;
  abstract update(id: string, data: Partial<T>): Promise<T>;
}

// shared/lib/persistence/index.ts
export const Repository = {
  accounts: createRepository<Account>(AccountSheetsAdapter),
  transactions: createRepository<Transaction>(TransactionSheetsAdapter),
  // ...
};
```

**Usage**:

```ts
// ✅ Only abstraction needed
import { Repository } from '@/shared/lib/persistence';

const account = await Repository.accounts.findById('123');
```

### 3. **Validation & Schemas**

```
shared/lib/validation/
├── schemas.ts                 # Zod schemas
├── types.ts                   # Extracted types from schemas
└── index.ts
```

**All API endpoints must validate**:

```ts
// accounts/api/route.ts
import { AccountSchema } from '@/shared/lib/validation';

export async function POST(request: Request) {
  const body = await request.json();
  const validated = AccountSchema.parse(body); // Throws if invalid
  return NextResponse.json(await createAccount(validated));
}
```

---

## Part 5: Migration Strategy (Feature-by-Feature)

### Phase 1: Infrastructure (Week 1)

- [ ] Create `shared/lib/` structure
- [ ] Create Repository abstraction
- [ ] Create unified AI service layer
- [ ] Set up validation schemas
- [ ] Create `features/` directory structure

### Phase 2: Core Features (Weeks 2-3)

Migrate in order of dependency (least dependent first):

1. **Settings** (no dependencies)
   - Move `/app/settings` → `/features/settings/`
   - Extract types/logic → `/features/settings/model/`

2. **Loans** (low dependency)
   - Move loan routes/components
   - Create loan service

3. **Accounts** (medium dependency)
   - Move account routes/components
   - Create account queries/mutations

4. **Transactions** (depends on accounts, categories)
   - Move transaction routes/components
   - Implement category service

5. **Budget** (depends on transactions)
   - Move budget routes/components

6. **Goals** (depends on accounts, categories)
   - Move goals routes/components

7. **Investments** (isolated)
   - Reorganize investment routes/components

8. **Chat** (depends on multiple domains)
   - Consolidate AI features
   - Create chat feature

### Phase 3: Integration (Week 4-5)

- [ ] Update root `/app/` pages to import from features
- [ ] Remove old `lib/sheets`, `lib/types`, etc
- [ ] Update imports throughout
- [ ] Test all routes
- [ ] Performance testing

---

## Part 6: Specific Refactoring Examples

### Example 1: Accounts Feature Migration

#### Before (Current)

```
src/
├── app/accounts/page.tsx
├── app/api/accounts/route.ts
├── components/accounts/account-card.tsx
├── components/accounts/accounts-summary.tsx
├── lib/types/account.ts
├── lib/sheets/accounts.ts
└── lib/utils/currency.ts
```

#### After (FSD)

```
src/features/accounts/
├── api/
│   └── route.ts                      # Moved from app/api/accounts
├── ui/
│   ├── page.tsx                      # Moved from app/accounts
│   ├── account-card.tsx
│   ├── accounts-summary.tsx
│   └── index.ts
├── model/
│   ├── types.ts                      # Moved from lib/types/account.ts
│   ├── schema.ts                     # Zod schemas (NEW)
│   ├── queries.ts                    # NEW - business logic
│   ├── mutations.ts                  # NEW - mutations
│   └── index.ts
├── lib/
│   └── utils.ts                      # Feature-specific utils
└── index.ts                          # PUBLIC: export AccountPage, useAccounts
```

**Code Migration Example**:

**BEFORE** (app/accounts/page.tsx):

```tsx
import { getAccounts } from '@/lib/sheets/accounts';
import { formatVND } from '@/lib/utils/currency';

export default async function AccountsPage() {
  const accounts = await getAccounts();
  return (
    <div>
      {accounts.map((acc) => (
        <div key={acc.name}>
          {acc.name}: {formatVND(acc.balance)}
        </div>
      ))}
    </div>
  );
}
```

**AFTER** (features/accounts/ui/page.tsx):

```tsx
import { getAccountsList } from '../model/queries';
import { AccountCard } from './account-card';

export default async function AccountsPage() {
  const accounts = await getAccountsList();
  return (
    <div className="grid gap-4">
      {accounts.map((account) => (
        <AccountCard key={account.id} account={account} />
      ))}
    </div>
  );
}
```

**Business Logic** (features/accounts/model/queries.ts):

```ts
import { Repository } from '@/shared/lib/persistence';
import { AccountAggregate } from './types';

export async function getAccountsList() {
  const raw = await Repository.accounts.findAll();
  return raw.map((acc) => new AccountAggregate(acc));
}
```

**Types** (features/accounts/model/types.ts):

```ts
export interface Account {
  id: string;
  name: string;
  balance: number;
  currency: Currency;
}

export class AccountAggregate {
  constructor(private account: Account) {}

  getDisplayBalance(): string {
    return formatCurrency(this.account.balance, this.account.currency);
  }
}
```

---

## Part 7: Key Design Patterns

### 1. **Server Actions Pattern** (for mutations)

```ts
// features/accounts/model/mutations.ts
'use server';

import { Repository } from '@/shared/lib/persistence';

export async function createAccount(data: CreateAccountInput) {
  // Validate
  const validated = CreateAccountSchema.parse(data);

  // Execute
  const account = await Repository.accounts.create(validated);

  // Revalidate cache
  revalidatePath('/accounts');

  return account;
}
```

### 2. **Client-Server Hooks Pattern**

```ts
// features/accounts/model/hooks.ts
'use client';

import { useMutation, useQuery } from '@/shared/hooks/queries';
import { createAccount, updateAccount } from './mutations';

export function useAccounts() {
  return useQuery('accounts', async () => {
    const res = await fetch('/api/accounts');
    return res.json();
  });
}

export function useCreateAccount() {
  return useMutation(createAccount);
}
```

### 3. **Dependency Injection in API Routes**

```ts
// features/accounts/api/route.ts
import { createAccountHandler } from '../model/handlers';
import { Repository } from '@/shared/lib/persistence';
import { ValidationService } from '@/shared/lib/validation';

// Inject dependencies
const handler = createAccountHandler(Repository.accounts, ValidationService, Logger);

export const POST = handler;
```

---

## Part 8: Testing Strategy

### Unit Tests (in `__tests__/`)

```ts
// features/accounts/__tests__/queries.test.ts
import { getAccountsList } from '../model/queries';
import { Repository } from '@/shared/lib/persistence';

vi.mock('@/shared/lib/persistence');

describe('getAccountsList', () => {
  it('returns aggregated accounts', async () => {
    const accounts = await getAccountsList();
    expect(accounts).toHaveLength(2);
  });
});
```

### Integration Tests

```ts
// features/accounts/__tests__/api.test.ts
import { POST } from '../api/route';

describe('POST /api/accounts', () => {
  it('creates account and returns it', async () => {
    const response = await POST(
      new Request(new URL('http://localhost:3000'), {
        method: 'POST',
        body: JSON.stringify({ name: 'Savings' }),
      }),
    );
    expect(response.status).toBe(201);
  });
});
```

---

## Part 9: Import Rules & ESLint Configuration

### `eslint.config.mjs` additions:

```js
{
  rules: {
    // FSD-specific rules
    'no-restricted-imports': [
      'error',
      {
        patterns: [
          {
            group: ['**/features/*/model', '**/features/*/api'],
            message: 'Cannot import model/api from other features. Use public exports.',
          },
          {
            group: ['@/lib/**', '@/app/**'],
            message: 'Use new structure: @/features, @/shared, @/core',
          },
        ],
      },
    ],
  },
}
```

---

## Part 10: Key Metrics & Success Criteria

### Before Refactor

- ❌ Circular dependencies possible
- ❌ 13 AI features scattered
- ❌ Components import data layer directly
- ❌ 60+ files in `/lib` with unclear ownership
- ❌ Difficult to test business logic in isolation

### After Refactor

- ✅ One-directional dependency graph
- ✅ Single `shared/lib/ai` service layer
- ✅ Components only import `features/*/ui`
- ✅ Clear feature ownership
- ✅ 90%+ testable business logic
- ✅ Average feature folder size: 25-35 files
- ✅ Clear feature boundaries

---

## Part 11: Risk Mitigation

### High Risks

- **Risk**: Breaking imports during migration
  - **Mitigation**: Run tests after each feature migration
  - **Tool**: ESLint rule to catch old imports

- **Risk**: Circular dependencies introduced
  - **Mitigation**: Use `dependency-cruiser` to visualize dependency graph
  - **Tool**: Add pre-commit hook

- **Risk**: Performance regression
  - **Mitigation**: Benchmark before/after
  - **Tool**: Lighthouse, Next.js built-in analytics

### Rollback Plan

```bash
# If migration goes wrong:
git checkout HEAD~N -- src/
npm run build  # Verify builds
npm run test   # Verify tests pass
```

---

## Part 12: Quick Start Checklist

- [ ] **Week 1**
  - [ ] Create `/src/shared/` structure
  - [ ] Create `/src/features/` structure
  - [ ] Create `/src/core/` structure
  - [ ] Set up Repository abstraction
  - [ ] Set up unified AI service
  - [ ] Add ESLint rules

- [ ] **Week 2**
  - [ ] Migrate Settings → features/
  - [ ] Migrate Loans → features/
  - [ ] Migrate Accounts → features/
  - [ ] Update imports

- [ ] **Week 3**
  - [ ] Migrate Transactions → features/
  - [ ] Migrate Budget → features/
  - [ ] Migrate Goals → features/

- [ ] **Week 4**
  - [ ] Migrate Investments → features/
  - [ ] Consolidate Chat/AI → features/
  - [ ] Delete old `/lib` folders

- [ ] **Week 5**
  - [ ] Integration testing
  - [ ] Performance optimization
  - [ ] Documentation update

---

## Conclusion

This refactoring will transform the codebase from a functional/layered architecture to a **scalable, maintainable feature-based design**. Each feature becomes a self-contained module with clear boundaries, making the app easier to:

- **Extend** (add new features independently)
- **Test** (isolated unit tests)
- **Maintain** (clear ownership)
- **Onboard** (new developers understand feature structure)
- **Scale** (prepare for team growth)

The migration should take **3-5 weeks** with a full aggressive refactor, resulting in a **production-ready, enterprise-grade codebase**.
