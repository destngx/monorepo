# FSD Migration Guide

This document describes the Feature-Sliced Design (FSD) refactoring of the wealth-management application and provides guidance for developing new features following this architecture.

## Overview

The wealth-management codebase has been migrated from a monolithic component structure to a **Feature-Sliced Design (FSD)** architecture. This provides:

- **Clear separation of concerns**: Each feature owns its UI, business logic, API integration, and type definitions
- **Reduced dependencies**: Features depend on `@/shared/` and each other's public exports, not internal implementation details
- **Scalability**: Adding new features is straightforward with the established pattern
- **Maintainability**: Code is organized by feature, not by technical layer

## Architecture

### Feature Structure

Each feature follows this directory structure:

```
src/features/[feature-name]/
├── model/                    # Business logic and data management
│   ├── types.ts             # TypeScript interfaces and types
│   ├── queries.ts           # Data fetching (client-side or server-side)
│   ├── mutations.ts         # Data mutations (create, update, delete)
│   ├── hooks.ts             # React hooks (useQuery, useMutation, state hooks)
│   ├── server-queries.ts    # Server-only queries (optional, if needed)
│   └── index.ts             # Public API exports
├── ui/                       # React components
│   ├── [component-name].tsx # Individual components
│   ├── page.tsx             # Main feature page
│   └── [other-pages].tsx    # Additional pages for nested routes
├── api/                      # API route handlers (optional)
│   └── route.ts             # API endpoints that delegate to sheets or backend
├── lib/                      # Feature-specific utilities (optional)
└── __tests__/               # Test files (optional)
```

### Shared Resources

```
src/shared/
├── ui/                      # Shared UI components (Button, Card, Input, etc.)
├── hooks/                   # Shared hooks (useMediaQuery, etc.)
├── types/                   # Global types used across features
├── lib/                     # Shared utilities (formatters, validators, etc.)
└── constants/               # App-wide constants
```

### Infrastructure

```
src/
├── lib/                     # Core libraries (sheets, AI, utils, types)
├── hooks/                   # Global hooks (useAISettings, etc.)
└── app/                     # Next.js app routes (mostly delegates to features)
```

## Migrated Features

### 1. Accounts (`src/features/accounts/`)

Manages bank account views, balance tracking, and account details.

**Key exports:**

- `getAccounts()` - Fetch all accounts
- `useAccounts()` - React hook for accounts
- Components: `AccountCard`, `AccountTable`, `AccountDetail`, etc.

**App routes delegated:**

- `/accounts` → `GoalsPage` from feature
- `/accounts/credit-cards` → feature pages

### 2. Transactions (`src/features/transactions/`)

Handles transaction listing, filtering, and review.

**Key exports:**

- `getTransactions(filters)` - Fetch transactions with optional filters
- `useTransactions()` - React hook
- `createTransaction()` - Add new transaction
- Components: `TransactionTable`, `TransactionForm`, `TransactionFilters`

**App routes delegated:**

- `/transactions` → `TransactionsPage` from feature

### 3. Budget (`src/features/budget/`)

Budget tracking and category spending analysis.

**Key exports:**

- `getBudgetItems()` - Fetch all budget categories
- `useBudget()` - React hook
- Components: `BudgetOverviewView`, `CategoryDetailView`, etc.

**App routes delegated:**

- `/budget` → `BudgetPage` from feature

### 4. Investments (`src/features/investments/`)

Investment portfolio and asset tracking.

**Key exports:**

- `getAssets()` - Fetch investment assets
- `getAssetPrices()` - Fetch current prices
- Components: `InvestmentsPage`

**App routes delegated:**

- `/investments` → `InvestmentsPage` from feature

### 5. Goals (`src/features/goals/`)

Financial goal management and tracking.

**Key exports:**

- `getGoals()` - Fetch all goals
- `useGoals()` - React hook
- `createGoal()` - Create new goal
- Components: `GoalCard`, `GoalDetailChart`, `CreateGoalFlow`

**App routes delegated:**

- `/accounts/goals` → `GoalsPage` from feature
- `/accounts/goals/[id]` → `GoalDetailPage`
- `/accounts/goals/new` → `NewGoalPage`

### 6. Loans (`src/features/loans/`)

Debt and loan tracking.

**Key exports:**

- `getLoans()` - Fetch all loans (server-side via sheets)
- `useLoans()` - React hook
- Components: `LoanCard`, `LoanList`, `LoanSummary`

**App routes delegated:**

- `/accounts/loans` → `LoansPage` from feature

### 7. Settings (`src/features/settings/`)

Application settings and preferences.

**Key exports:**

- `useAISettings()` - AI provider settings hook
- Components: Settings configuration UI

**App routes delegated:**

- `/settings` → `SettingsPage` from feature

### 8. Chat (`src/features/chat/`)

Conversational AI interface (migrated in earlier phases).

## Import Patterns

### Correct Imports

```typescript
// Importing from feature model layer
import { getAccounts, useAccounts } from '@/features/accounts/model/queries';
import { useAccounts } from '@/features/accounts/model/hooks';
import { Account } from '@/features/accounts/model/types';

// Or via feature index (preferred)
import { queries, mutations, hooks, type Account } from '@/features/accounts';

// Importing shared UI
import { Button } from '@/shared/ui/button';
import { Card } from '@/shared/ui/card';

// Importing shared hooks
import { useMediaQuery } from '@/shared/hooks/use-media-query';

// Importing shared utilities
import { cn } from '@/shared/lib/utils';
```

### Incorrect Imports (Avoid)

```typescript
// ❌ DO NOT import from old paths
import { Account } from '@/lib/types/account'; // Old path
import { getAccounts } from '@/lib/sheets/accounts'; // Old internal path
import { AccountCard } from '@/components/accounts'; // Old monolithic path

// ❌ DO NOT import between features directly
import { Account } from '@/features/transactions/model';
// Instead, if types are shared, move to @/shared/types or accept re-export

// ❌ DO NOT import internal implementation
import { someHelper } from '@/features/accounts/lib/internal-helpers';
// Use public API from index.ts instead
```

## Creating a New Feature

### Step 1: Create directory structure

```bash
mkdir -p src/features/my-feature/{model,ui,api,lib,__tests__}
```

### Step 2: Create model layer

**`src/features/my-feature/model/types.ts`**

```typescript
export interface MyEntity {
  id: string;
  name: string;
  // ... other fields
}
```

**`src/features/my-feature/model/queries.ts`**

```typescript
import { MyEntity } from './types';

export async function getMyEntities(): Promise<MyEntity[]> {
  const response = await fetch('/api/my-feature');
  if (!response.ok) throw new Error('Failed to fetch');
  return response.json();
}
```

**`src/features/my-feature/model/mutations.ts`**

```typescript
import { MyEntity } from './types';

export async function createMyEntity(data: Omit<MyEntity, 'id'>): Promise<MyEntity> {
  const response = await fetch('/api/my-feature', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) throw new Error('Failed to create');
  return response.json();
}
```

**`src/features/my-feature/model/hooks.ts`**

```typescript
'use client';

import { useEffect, useState } from 'react';
import { MyEntity } from './types';
import * as queries from './queries';

export function useMyEntities() {
  const [data, setData] = useState<MyEntity[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    queries
      .getMyEntities()
      .then(setData)
      .catch(setError)
      .finally(() => setLoading(false));
  }, []);

  return { data, loading, error };
}
```

**`src/features/my-feature/model/index.ts`**

```typescript
export * from './types';
export * as queries from './queries';
export * as mutations from './mutations';
export * from './hooks';
```

### Step 3: Create UI components

**`src/features/my-feature/ui/my-entity-card.tsx`**

```typescript
'use client';

import { MyEntity } from '../model/types';

export function MyEntityCard({ entity }: { entity: MyEntity }) {
  return (
    <div className="p-4 border rounded">
      <h3>{entity.name}</h3>
    </div>
  );
}
```

**`src/features/my-feature/ui/page.tsx`**

```typescript
'use client';

import { useMyEntities } from '../model/hooks';
import { MyEntityCard } from './my-entity-card';

export default function MyFeaturePage() {
  const { data, loading, error } = useMyEntities();

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      {data.map(entity => (
        <MyEntityCard key={entity.id} entity={entity} />
      ))}
    </div>
  );
}
```

### Step 4: Create API routes (if needed)

**`src/features/my-feature/api/route.ts`**

```typescript
import { getMyFeatureData } from '@/lib/sheets/my-feature';

export async function GET() {
  const data = await getMyFeatureData();
  return Response.json(data);
}

export async function POST(req: Request) {
  const body = await req.json();
  // ... process request
  return Response.json({ success: true });
}
```

### Step 5: Create app route delegate

**`src/app/my-feature/page.tsx`**

```typescript
import MyFeaturePage from '@/features/my-feature/ui/page';

export default MyFeaturePage;
```

## Common Patterns

### Using Feature Data in Other Components

```typescript
// In a dashboard or layout component
import { useAccounts } from '@/features/accounts/model/hooks';

export function DashboardAccountsWidget() {
  const { accounts, loading } = useAccounts();

  if (loading) return <div>Loading accounts...</div>;

  return (
    <div>
      {accounts.map(account => (
        <div key={account.id}>{account.name}</div>
      ))}
    </div>
  );
}
```

### Sharing Types Between Features

**Bad approach:** Importing from another feature's model directly

```typescript
// ❌ Avoid
import { Transaction } from '@/features/transactions/model/types';
```

**Good approach:** Keep shared types in `@/shared/types`

```typescript
// ✅ Prefer
// In src/shared/types/transaction.ts
export interface Transaction {
  id: string;
  amount: number;
  // ...
}

// Both features can import from shared
import { Transaction } from '@/shared/types/transaction';
```

### Handling Cross-Feature Dependencies

If features need to share data:

1. **Extract to shared/types** - Move common types there
2. **Use public API exports** - Each feature exports from `index.ts`
3. **Keep features decoupled** - Features should not directly import other features' implementation

```typescript
// In feature A's ui component
import { useTransactions } from '@/features/transactions/model/hooks';
import { AccountCard } from '@/features/accounts/ui/account-card'; // ✅ OK - UI component
```

## Old Paths Reference

The following old paths have been replaced. For reference:

| Old Path                        | New Path                                 | Notes                    |
| ------------------------------- | ---------------------------------------- | ------------------------ |
| `@/lib/types/account.ts`        | `@/features/accounts/model/types.ts`     | Account type definitions |
| `@/lib/sheets/accounts`         | `@/features/accounts/model/queries.ts`   | Account queries          |
| `@/components/accounts/...`     | `@/features/accounts/ui/...`             | Account components       |
| `@/lib/types/transaction.ts`    | `@/features/transactions/model/types.ts` | Transaction types        |
| `@/components/transactions/...` | `@/features/transactions/ui/...`         | Transaction components   |
| `@/lib/types/budget.ts`         | `@/features/budget/model/types.ts`       | Budget types             |
| `@/components/budget/...`       | `@/features/budget/ui/...`               | Budget components        |
| `@/lib/types/goals.ts`          | `@/features/goals/model/types.ts`        | Goals types              |
| `@/components/goals/...`        | `@/features/goals/ui/...`                | Goals components         |
| `@/lib/types/loan.ts`           | `@/features/loans/model/types.ts`        | Loans types              |
| `@/components/loans/...`        | `@/features/loans/ui/...`                | Loans components         |
| `@/lib/types/settings.ts`       | `@/features/settings/model/types.ts`     | Settings types           |
| `@/app/settings/page.tsx`       | `@/features/settings/ui/page.tsx`        | Settings page            |

## Verification Checklist

Before considering the migration complete:

- [ ] All TypeScript files compile without errors: `npx tsc --noEmit`
- [ ] Build succeeds: `pnpm build`
- [ ] Development server runs: `pnpm dev`
- [ ] All feature imports use new paths (`@/features/*`)
- [ ] App routes delegate to feature pages
- [ ] No circular dependencies between features
- [ ] Feature indices export all public API
- [ ] Old component files kept for safety (not deleted)

## Questions?

Refer to the `IMPLEMENTATION-ROADMAP.md` for the original migration plan and structure.
