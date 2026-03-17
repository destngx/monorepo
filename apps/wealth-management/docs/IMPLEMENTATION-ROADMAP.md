# FSD Implementation Roadmap - Week by Week

## Overview
- **Total Duration**: 5 weeks (aggressive)
- **Team Size**: Recommended 2-3 developers
- **Risk Level**: Medium (extensive file reorganization)
- **Rollback Plan**: Git branches, can revert per feature

---

## WEEK 1: Infrastructure Setup

### Goal
Set up the new directory structure and abstraction layers. No feature migration yet.

### Tasks

#### 1.1 Create Directory Structure
```bash
mkdir -p src/shared/{ui,lib/{ai,persistence,validation,services,utils,types},hooks,constants,api}
mkdir -p src/core/{auth,cache,config,middleware}
mkdir -p src/features/{accounts,transactions,budget,goals,loans,investments,chat,settings}
```

#### 1.2 Persistence Layer (Repository Pattern)
**File**: `src/shared/lib/persistence/repository.ts`

```ts
// Abstract base
export abstract class Repository<T> {
  abstract findById(id: string): Promise<T | null>;
  abstract findAll(): Promise<T[]>;
  abstract create(data: Partial<T>): Promise<T>;
  abstract update(id: string, data: Partial<T>): Promise<T>;
  abstract delete(id: string): Promise<void>;
}

// Factory
export function createRepository<T>(
  adapter: new () => RepositoryAdapter<T>
): Repository<T> {
  return new adapter();
}
```

**File**: `src/shared/lib/persistence/adapters/sheets-adapter.ts`
```ts
import { Repository } from '../repository';
import { getAccounts as sheetsGetAccounts } from '@/lib/sheets/accounts';

export class AccountSheetsAdapter extends Repository<Account> {
  async findById(id: string) {
    const accounts = await sheetsGetAccounts();
    return accounts.find(a => a.name === id) || null;
  }

  async findAll() {
    return sheetsGetAccounts();
  }

  async create(data: Partial<Account>) {
    // Delegate to sheets layer
    return createAccountInSheets(data);
  }
  // ... implement others
}
```

**File**: `src/shared/lib/persistence/index.ts`
```ts
import { AccountSheetsAdapter } from './adapters/sheets-adapter';
import { TransactionSheetsAdapter } from './adapters/transaction-adapter';
// ... etc

export const Repository = {
  accounts: createRepository(AccountSheetsAdapter),
  transactions: createRepository(TransactionSheetsAdapter),
  budget: createRepository(BudgetSheetsAdapter),
  goals: createRepository(GoalsSheetsAdapter),
  loans: createRepository(LoansSheetsAdapter),
};

export * from './repository';
```

**Status**: ✅ TESTABLE - Can test Repository without touching actual sheets

#### 1.3 Unified AI Service Layer
**File**: `src/shared/lib/ai/service.ts`

```ts
import { generateText } from 'ai';
import * as Prompts from './prompts';

interface AIAnalysisRequest {
  domain: 'accounts' | 'budget' | 'transactions' | 'goals' | 'investments';
  action: 'review' | 'analysis' | 'suggestion' | 'summary';
  data: any;
}

export class AIService {
  static async analyze(request: AIAnalysisRequest): Promise<string> {
    const prompt = this.buildPrompt(request);
    const result = await generateText({
      model: getModel(request.domain),
      prompt,
      system: SYSTEM_PROMPT,
    });
    return result.text;
  }

  private static buildPrompt(request: AIAnalysisRequest): string {
    const prompts = {
      'accounts.review': () => Prompts.accountReview(request.data),
      'budget.analysis': () => Prompts.budgetAnalysis(request.data),
      // ... map all combinations
    };
    const key = `${request.domain}.${request.action}`;
    return prompts[key]?.() || '';
  }
}
```

**Status**: ✅ INTERFACE UNIFIED - All AI calls go through this service

#### 1.4 Validation Schemas
**File**: `src/shared/lib/validation/schemas.ts`

```ts
import { z } from 'zod';

// Account schemas
export const AccountSchema = z.object({
  name: z.string().min(1),
  type: z.enum(['active use', 'savings', 'investment']),
  currency: z.enum(['VND', 'USD', 'USDT']),
  balance: z.number(),
});

export const CreateAccountSchema = AccountSchema.omit({ balance: true }).extend({
  balance: z.number().optional(),
});

// Transaction schemas
export const TransactionSchema = z.object({
  date: z.date(),
  payee: z.string(),
  category: z.string(),
  amount: z.number(),
  accountName: z.string(),
});

// ... all domain schemas
```

**Status**: ✅ CENTRALIZED - Single source of truth for validation

#### 1.5 ESLint Configuration
**File**: Update `eslint.config.mjs`

```js
export default [
  {
    files: ['src/**/*.{ts,tsx}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          patterns: [
            // Prevent importing from feature internals
            {
              group: ['**/features/*/model', '**/features/*/api'],
              message: 'Cannot import internal feature logic. Use public exports from feature index.ts',
            },
            // Prevent old import paths
            {
              group: ['@/lib/**'],
              importNames: ['*'],
              message: 'Old lib structure. Use @/shared, @/core, or @/features',
            },
            // Prevent cross-feature imports
            {
              group: ['**/features/accounts/**', '**/features/budget/**'],
              message: 'Cross-feature imports not allowed. Move logic to @/shared',
            },
          ],
        },
      ],
    },
  },
];
```

**Status**: ✅ GUARDED - ESLint prevents architecture violations

#### 1.6 Create Feature Templates
**File**: `src/features/accounts/index.ts` (as template)

```ts
// Public API for accounts feature
export { AccountsPage } from './ui/page';
export { useAccounts } from './model/hooks';
export { getAccounts } from './model/queries';

// Types
export type { Account } from './model/types';

// ❌ Do NOT export internals:
// export { Repository } from './api/repository';
// export { queries } from './model/queries';
```

**Status**: ✅ TEMPLATE - Clear pattern for other features

#### 1.7 Set Up TypeScript Path Aliases (Already configured)
Verify `tsconfig.json` has:
```json
{
  "compilerOptions": {
    "paths": {
      "@/*": ["./src/*"],
      "@/features/*": ["./src/features/*"],
      "@/shared/*": ["./src/shared/*"],
      "@/core/*": ["./src/core/*"]
    }
  }
}
```

**Status**: ✅ CONFIGURED

### Week 1 Deliverables
- [x] `/src/shared/`, `/src/core/`, `/src/features/` structure created
- [x] Repository abstraction implemented
- [x] Unified AI service created
- [x] Validation schemas centralized
- [x] ESLint rules configured
- [x] Feature template created
- [x] All tests passing
- [x] No code removed yet (old structure still intact)

**Verification**:
```bash
npm run build    # Should succeed
npm run test     # Should pass (no changes to features yet)
npm run lint     # Should pass (new rules are warnings)
```

---

## WEEK 2: Core Feature Migration (Settings, Loans, Accounts)

### Goal
Migrate 3 independent features to new structure. Establish migration pattern.

### Feature Priority
1. **Settings** - No dependencies, simplest
2. **Loans** - Minimal dependencies
3. **Accounts** - Foundation for other features

---

### 2.1 Migrate SETTINGS Feature

#### Current Structure
```
src/
├── app/settings/page.tsx
├── components/settings/ (if any)
├── lib/types/settings.ts
```

#### Target Structure
```
src/features/settings/
├── ui/
│   ├── page.tsx           (from app/settings/page.tsx)
│   └── index.ts
├── model/
│   ├── types.ts           (from lib/types/settings.ts)
│   ├── schema.ts
│   ├── queries.ts         (new - data fetching)
│   ├── mutations.ts       (new - data mutations)
│   └── index.ts
├── __tests__/
├── index.ts
```

#### Step-by-step
1. Copy settings page to `features/settings/ui/page.tsx`
2. Create `features/settings/model/types.ts` from old types
3. Create `features/settings/model/queries.ts`:
   ```ts
   export async function getSettings() {
     // TODO: implement
   }
   ```
4. Create `features/settings/index.ts`:
   ```ts
   export { SettingsPage } from './ui/page';
   ```
5. Update `app/settings/page.tsx`:
   ```tsx
   import { SettingsPage } from '@/features/settings';
   export default SettingsPage;
   ```
6. ✅ Keep old files for now (will delete after testing)

**Verification**:
```bash
npm run build
npm run test
# Open http://localhost:3000/settings - should work
```

#### Status: ✅ ROUTED - Settings still works, migrated to new structure

---

### 2.2 Migrate LOANS Feature

#### Current Structure
```
src/
├── app/accounts/loans/page.tsx
├── app/api/loans/route.ts
├── components/loans/
│   ├── loan-list.tsx
│   ├── loan-review-ai.tsx
│   └── loan-summary.tsx
├── lib/types/loan.ts
├── lib/sheets/loans.ts
```

#### Target Structure
```
src/features/loans/
├── api/
│   └── route.ts           (from app/api/loans/route.ts)
├── ui/
│   ├── page.tsx
│   ├── loan-list.tsx
│   ├── loan-review-ai.tsx
│   ├── loan-summary.tsx
│   └── index.ts
├── model/
│   ├── types.ts           (from lib/types/loan.ts)
│   ├── schema.ts
│   ├── queries.ts         (wraps lib/sheets/loans.ts)
│   ├── mutations.ts
│   ├── hooks.ts
│   └── index.ts
├── __tests__/
├── index.ts
```

#### Implementation
**File**: `src/features/loans/model/queries.ts`
```ts
import { Repository } from '@/shared/lib/persistence';
import { Loan } from './types';

export async function getLoans(): Promise<Loan[]> {
  return Repository.loans.findAll();
}

export async function getLoanById(id: string): Promise<Loan | null> {
  return Repository.loans.findById(id);
}
```

**File**: `src/features/loans/api/route.ts`
```ts
import { NextResponse } from 'next/server';
import { getLoans } from '../model/queries';

export async function GET() {
  try {
    const loans = await getLoans();
    return NextResponse.json(loans);
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to fetch loans' },
      { status: 500 }
    );
  }
}
```

**File**: `src/app/accounts/loans/page.tsx` (update)
```tsx
import { LoansPage } from '@/features/loans/ui';
export default LoansPage;
```

**Verification**:
```bash
npm run build
npm run test
# Open http://localhost:3000/accounts/loans - should work
# GET /api/loans - should work
```

#### Status: ✅ MIGRATED - Loans feature moved and working

---

### 2.3 Migrate ACCOUNTS Feature (Most Important)

#### Target Structure
```
src/features/accounts/
├── api/
│   ├── route.ts
│   └── schemas.ts
├── ui/
│   ├── page.tsx
│   ├── account-summary.tsx
│   ├── accounts-card.tsx
│   ├── account-trend-sparkline.tsx
│   ├── account-review-ai.tsx
│   └── index.ts
├── model/
│   ├── types.ts
│   ├── schema.ts
│   ├── queries.ts
│   ├── mutations.ts
│   ├── hooks.ts
│   ├── utils.ts
│   └── index.ts
├── lib/
│   └── utils.ts
├── __tests__/
│   ├── queries.test.ts
│   ├── mutations.test.ts
│   └── api.test.ts
└── index.ts
```

#### Key Implementation
**File**: `src/features/accounts/model/types.ts`
```ts
export interface Account {
  name: string;
  type: AccountType;
  currency: Currency;
  balance: number;
  clearedBalance: number;
  // ...
}

export class AccountAggregate {
  constructor(private account: Account) {}

  getDisplayBalance(): string {
    return formatCurrency(this.account.balance, this.account.currency);
  }

  isLowBalance(threshold = 1000): boolean {
    return this.account.balance < threshold;
  }

  getAccountHealth(): 'good' | 'warning' | 'critical' {
    if (this.account.balance > 10000) return 'good';
    if (this.account.balance > 1000) return 'warning';
    return 'critical';
  }
}
```

**File**: `src/features/accounts/model/queries.ts`
```ts
import { Repository } from '@/shared/lib/persistence';
import { AccountAggregate } from './types';

export async function getAccounts() {
  const raw = await Repository.accounts.findAll();
  return raw.map(account => new AccountAggregate(account));
}

export async function getAccountById(id: string) {
  const account = await Repository.accounts.findById(id);
  if (!account) return null;
  return new AccountAggregate(account);
}

// AI features integrated
export async function getAccountReview(account: Account) {
  const { AIService } = await import('@/shared/lib/ai');
  return AIService.analyze({
    domain: 'accounts',
    action: 'review',
    data: account,
  });
}
```

**File**: `src/features/accounts/model/hooks.ts`
```ts
'use client';

import { useQuery } from '@/shared/hooks/query';
import { getAccounts } from './queries';

export function useAccounts() {
  return useQuery(
    ['accounts'],
    () => getAccounts(),
    { revalidate: 60 }
  );
}
```

**File**: `src/features/accounts/ui/page.tsx`
```tsx
'use client';

import { useAccounts } from '../model/hooks';
import { AccountCard } from './account-card';

export function AccountsPage() {
  const { data: accounts, isLoading } = useAccounts();

  if (isLoading) return <div>Loading...</div>;

  return (
    <div className="grid gap-4">
      {accounts?.map(account => (
        <AccountCard key={account.name} account={account} />
      ))}
    </div>
  );
}
```

**File**: `src/features/accounts/api/route.ts`
```ts
import { NextResponse } from 'next/server';
import { getAccounts } from '../model/queries';
import { AccountSchema } from './schemas';

export async function GET() {
  try {
    const accounts = await getAccounts();
    return NextResponse.json(accounts);
  } catch (error) {
    return NextResponse.json(
      { error: 'Failed to fetch accounts' },
      { status: 500 }
    );
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    const validated = AccountSchema.parse(body);
    // TODO: implement createAccount
    return NextResponse.json(validated, { status: 201 });
  } catch (error) {
    if (error instanceof z.ZodError) {
      return NextResponse.json(
        { error: error.errors },
        { status: 400 }
      );
    }
    return NextResponse.json(
      { error: 'Failed to create account' },
      { status: 500 }
    );
  }
}
```

**Status**: ✅ COMPLETE - Accounts fully migrated

### Week 2 Deliverables
- [x] Settings migrated
- [x] Loans migrated
- [x] Accounts migrated (most complex)
- [x] All routes working
- [x] All tests passing
- [x] Old files still in place (safe rollback)

**Verification**:
```bash
npm run build      # ✅ Success
npm run test       # ✅ All pass
npm run lint       # ✅ No errors
```

---

## WEEK 3: Continue Feature Migration (Transactions, Budget, Goals)

### Goal
Migrate 3 interconnected features.

### 3.1 Migrate TRANSACTIONS (depends on Account)
- Most complex data structure
- Most frequently used
- AI features integrated

**Key Points**:
- Import Account types from `features/accounts/model/types`
- Use Repository for transaction persistence
- Consolidate transaction-related AI features

### 3.2 Migrate BUDGET (depends on Transactions)
- Budget calculation uses transaction history
- Cross-feature dependency pattern

### 3.3 Migrate GOALS (depends on Accounts)
- Goal progress calculated from account balance

### Pattern
Each feature follows same structure as Accounts/Loans.

---

## WEEK 4: Consolidate AI & Investments, Integration

### Goal
Consolidate scattered AI features, migrate investments, integrate all routes.

### 4.1 Consolidate AI Features
- Move 13 scattered AI endpoints to feature-specific locations
- All AI calls route through `shared/lib/ai/`

### 4.2 Migrate INVESTMENTS
- Least complex
- Good for testing new patterns

### 4.3 Update App Routes
```
app/
├── layout.tsx
├── page.tsx                  (home/dashboard)
├── (dashboard)/
│   ├── layout.tsx
│   ├── accounts/
│   │   └── page.tsx         (import from features/accounts/ui)
│   ├── transactions/
│   │   └── page.tsx         (import from features/transactions/ui)
│   ├── budget/
│   │   └── page.tsx         (import from features/budget/ui)
│   ├── goals/
│   │   └── page.tsx
│   ├── loans/
│   │   └── page.tsx
│   └── investments/
│       └── page.tsx
│
└── api/
    ├── accounts/
    │   └── route.ts         (delegate to features/accounts/api)
    ├── transactions/
    │   └── route.ts
    ├── budget/
    │   └── route.ts
    ├── goals/
    │   └── route.ts
    ├── loans/
    │   └── route.ts
    ├── investments/
    │   ├── assets/
    │   │   └── route.ts
    │   └── prices/
    │       └── route.ts
    └── ...
```

---

## WEEK 5: Cleanup, Testing, Documentation

### Goal
Remove old files, verify everything works, document.

### 5.1 Remove Old Structure
```bash
# Backup just in case
git branch backup/pre-cleanup

# Delete old structure
rm -rf src/lib/types
rm -rf src/lib/sheets (after verifying all migrated to Repository)
rm -rf src/lib/ai (after moving to shared/lib/ai)
rm -rf src/components (after moving to features/*/ui)
```

### 5.2 Full System Testing
```bash
npm run build       # Full build
npm run test        # All unit tests
npm run lint        # No errors
npm run dev         # Manual testing
```

### 5.3 Performance Audit
```bash
# Check bundle size
npx next-bundle-analyzer

# Check LCP, FID, CLS
npm run dev
# Open Lighthouse in DevTools
```

### 5.4 Documentation
- Update README with new structure
- Create ARCHITECTURE.md
- Update contribution guidelines

---

## Parallel Streams (If 2-3 Developers)

### Developer 1: Infrastructure + Accounts
- Week 1: All infrastructure
- Week 2: Accounts (largest)

### Developer 2: Loans + Transactions
- Week 1: Setup (copy of infrastructure)
- Week 2: Loans
- Week 3: Transactions

### Developer 3: Budget + Goals + Investments
- Week 1: Setup (copy of infrastructure)
- Week 2: Follow along
- Week 3: Budget + Goals
- Week 4: Investments

**Merge Strategy**:
- Each feature is independent
- Weekly merge to main after testing
- No conflicts if following structure strictly

---

## Testing During Migration

### Unit Tests
```ts
// features/accounts/__tests__/queries.test.ts
import { getAccounts } from '../model/queries';
import { Repository } from '@/shared/lib/persistence';

vi.mock('@/shared/lib/persistence');

describe('getAccounts', () => {
  it('returns account aggregates', async () => {
    vi.mocked(Repository.accounts.findAll).mockResolvedValue([
      { name: 'Savings', balance: 1000, ... }
    ]);

    const result = await getAccounts();
    expect(result[0]).toBeInstanceOf(AccountAggregate);
  });
});
```

### Integration Tests
```ts
// features/accounts/__tests__/api.test.ts
import { GET, POST } from '../api/route';

describe('GET /api/accounts', () => {
  it('returns 200 with accounts', async () => {
    const response = await GET(new Request(...));
    expect(response.status).toBe(200);
  });
});
```

### E2E Tests (Optional)
```ts
// tests/e2e/accounts.test.ts
describe('Accounts Flow', () => {
  it('displays accounts on page', async ({ page }) => {
    await page.goto('/accounts');
    await expect(page.locator('text=Savings')).toBeVisible();
  });
});
```

---

## Rollback Procedure

If anything goes wrong:

```bash
# Per-feature rollback
git checkout main -- src/features/accounts

# Full rollback to start of week
git revert <commit-hash>

# Emergency: back to before refactor started
git checkout <week-0-commit> -- src/
```

---

## Success Metrics

| Metric | Target | Current |
|--------|--------|---------|
| Build time | < 30s | ~20s |
| Type errors | 0 | 0 |
| ESLint errors | 0 | TBD |
| Test coverage | > 70% | TBD |
| Bundle size | < 400KB | TBD |
| LCP | < 2.5s | TBD |

---

## Checklist for Each Feature Migration

- [ ] Create folder structure
- [ ] Move files to new location
- [ ] Update imports (feature internally first)
- [ ] Create `index.ts` with public exports
- [ ] Create `model/` layer with business logic
- [ ] Create/update `api/` routes
- [ ] Create/update `ui/` components
- [ ] Write tests
- [ ] Update `app/` routes to delegate
- [ ] Test in browser
- [ ] Run full build
- [ ] ESLint pass
- [ ] All tests pass
- [ ] Merge to main
- [ ] Keep old files until all features done (rollback safety)
- [ ] Delete old files in Week 5
- [ ] Final system test

---

## Common Issues & Solutions

### Issue: Circular Dependencies
**Cause**: Feature A imports from Feature B, B imports from A  
**Solution**: Move shared logic to `shared/lib/`  
**Prevention**: ESLint rules in place

### Issue: Type Errors After Migration
**Cause**: Missing exports in `index.ts`  
**Solution**: Verify all public APIs exported  
**Prevention**: Use barrel files strictly

### Issue: API Routes Not Working
**Cause**: Path mismatch between new location and expected route  
**Solution**: Ensure `app/api/X/route.ts` delegates to `features/X/api/route.ts`  
**Prevention**: Test routes after each migration

### Issue: Performance Regression
**Cause**: Over-fetching, missing memoization  
**Solution**: Profile with DevTools, optimize hot paths  
**Prevention**: Run Lighthouse after each feature

---

## Final Notes

- **Commit After Each Feature**: Atomic commits make rollback easier
- **Branch Strategy**: One branch per feature OR one feature per developer
- **Testing is Non-Negotiable**: Don't skip tests to save time
- **Documentation**: Update as you go, not at the end
- **Communication**: Daily sync if team of 2+ to avoid merge conflicts

This refactor is a **significant undertaking** but will result in a **much better codebase** for long-term maintenance and scaling.

**You've got this!** 🚀
