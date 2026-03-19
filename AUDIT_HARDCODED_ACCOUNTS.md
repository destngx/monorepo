# Wealth-Management Codebase Audit: Hardcoded Account-Related Strings

**Date**: March 18, 2026  
**Scope**: Complete audit of account-related hardcoding outside config layer  
**Status**: COMPREHENSIVE WITH REFACTORING RECOMMENDATIONS

---

## Executive Summary

The wealth-management codebase has **largely centralized** account type configuration through Phase A-B refactoring. However, significant **UI layout and display configuration duplications** remain scattered across feature and app layers. Additionally, **account-specific business logic** and **AI prompt references** contain hardcoded account names and descriptions.

### Key Findings:

- ✅ **Account types centralized**: `libs/wealth-management/src/config/accounts/types.ts`
- ✅ **Currency rules centralized**: `libs/wealth-management/src/config/accounts/rules.ts`
- ⚠️ **UI display config duplicated**: Hardcoded `TYPE_CONFIG` and `TYPE_ORDER` in 2 component files
- ⚠️ **Account name hardcoding**: Specific account names (`binance`, `Binance`) referenced in AI prompts
- ⚠️ **Duplicate constants**: `CRYPTO_ACCOUNTS` and `USD_ACCOUNTS` in both `types/accounts.ts` and `config/accounts/rules.ts`
- ⚠️ **Default fallback**: Hardcoded `'bank'` as default account type in app layer

---

## 1. HARDCODED ACCOUNT TYPE STRINGS

### Category 1.1: Account Type Duplicate Definitions

**FINDING**: Account type union is defined in BOTH the old types layer AND config layer.

| File                                                  | Lines | Issue                                  |
| ----------------------------------------------------- | ----- | -------------------------------------- |
| `libs/wealth-management/src/types/accounts.ts`        | 1-10  | **DUPLICATE** - Old definition remains |
| `libs/wealth-management/src/config/accounts/types.ts` | 13-22 | **CANONICAL** - Config layer (correct) |
| `libs/wealth-management/src/schemas/accounts.ts`      | 7-17  | Uses z.enum with hardcoded strings     |

**Context (types/accounts.ts)**:

```typescript
export type AccountType =
  | 'active use'
  | 'rarely use'
  | 'long holding'
  | 'deprecated'
  | 'negative active use'
  | 'bank'
  | 'crypto'
  | 'cash'
  | 'investment';
```

**Status**: Types layer re-exports from config, but maintains independent definition. This is **redundant**.

---

### Category 1.2: Hardcoded Account Type References in Schemas

**FINDING**: Zod schema duplicates account type strings instead of deriving from config.

| File                                             | Lines | Strings                               |
| ------------------------------------------------ | ----- | ------------------------------------- |
| `libs/wealth-management/src/schemas/accounts.ts` | 7-17  | All 9 account types hardcoded in enum |

**Current Code**:

```typescript
export const AccountTypeSchema = z.enum([
  'active use',
  'rarely use',
  'long holding',
  'deprecated',
  'negative active use',
  'bank',
  'crypto',
  'cash',
  'investment',
]);
```

**Risk**: If new account types added to config, schema isn't auto-updated.

---

### Category 1.3: Account Type Constant Duplication

**FINDING**: `CRYPTO_ACCOUNTS` and `USD_ACCOUNTS` defined in TWO locations:

| File                                                  | Lines | Scope               | Status              |
| ----------------------------------------------------- | ----- | ------------------- | ------------------- |
| `libs/wealth-management/src/types/accounts.ts`        | 26-27 | **private** (const) | Old location        |
| `libs/wealth-management/src/config/accounts/rules.ts` | 5-6   | **exported**        | Canonical (correct) |

**Current Issue**:

- `types/accounts.ts`: Private constants not exposed
- `config/accounts/rules.ts`: Exported and re-exported via features layer
- Consumer code imports from features (which re-exports config) ✓

**File References**:

- `libs/wealth-management/src/features/accounts/model/types.ts` line 4 correctly re-exports from config/rules

---

## 2. HARDCODED ACCOUNT-RELATED ENUMS & CONSTANTS

### Category 2.1: UI Display Configuration Duplication

**FINDING**: `TYPE_CONFIG` (account type → display labels/colors) and `TYPE_ORDER` hardcoded in TWO component files.

| File                                                       | Type         | Status                                |
| ---------------------------------------------------------- | ------------ | ------------------------------------- |
| `libs/wealth-management/src/features/accounts/ui/page.tsx` | UI Component | ⚠️ Hardcoded TYPE_CONFIG & TYPE_ORDER |
| `apps/wealth-management/src/features/accounts/ui/page.tsx` | UI Component | ⚠️ Hardcoded TYPE_CONFIG & TYPE_ORDER |

**Details - libs version (lines 16-51)**:

```typescript
const TYPE_CONFIG: Record<string, { label: string; icon: React.ReactNode; colorTitle: string; bgSoft: string }> = {
  'active use': {
    label: 'Everyday Banking',
    icon: <Wallet className="h-4 w-4" />,
    colorTitle: 'text-emerald-600 dark:text-emerald-500',
    bgSoft: 'bg-emerald-500/10'
  },
  'rarely use': {
    label: 'Savings & Reserves',
    icon: <Landmark className="h-4 w-4" />,
    colorTitle: 'text-blue-600 dark:text-blue-500',
    bgSoft: 'bg-blue-500/10'
  },
  'long holding': {
    label: 'Investments',
    icon: <BarChart3 className="h-4 w-4" />,
    colorTitle: 'text-indigo-600 dark:text-indigo-500',
    bgSoft: 'bg-indigo-500/10'
  },
  'negative active use': {
    label: 'Credit Cards & Loans',
    icon: <CreditCard className="h-4 w-4" />,
    colorTitle: 'text-slate-800 dark:text-slate-200',
    bgSoft: 'bg-slate-500/10'
  },
  'deprecated': {
    label: 'Archived Accounts',
    icon: <Archive className="h-4 w-4" />,
    colorTitle: 'text-gray-500',
    bgSoft: 'bg-gray-500/10'
  },
};

const TYPE_ORDER = ['active use', 'long holding', 'rarely use', 'negative active use', 'deprecated'];
```

**Details - apps version (lines 50-98)**:

```typescript
const TYPE_CONFIG: Record<AccountType, { label: string; colorTitle: string; bgSoft: string }> = {
  'active use': {
    label: 'Everyday Banking',
    colorTitle: 'text-emerald-600 dark:text-emerald-500',
    bgSoft: 'bg-emerald-500/10',
  },
  'rarely use': {
    label: 'Savings & Reserves',
    colorTitle: 'text-blue-600 dark:text-blue-500',
    bgSoft: 'bg-blue-500/10',
  },
  // ... 7 more account types
  investment: {
    label: 'Investment',
    colorTitle: 'text-purple-600 dark:text-purple-500',
    bgSoft: 'bg-purple-500/10',
  },
};

const TYPE_ORDER: AccountType[] = ['active use', 'long holding', 'rarely use', 'negative active use', 'deprecated'];
```

**Lines Affected**:

- libs: lines 17-51 (TYPE_CONFIG), line 51 (TYPE_ORDER)
- apps: lines 50-96 (TYPE_CONFIG), line 98 (TYPE_ORDER)

**Issue**:

1. Visual/layout configuration mixed with business logic in UI components
2. Inconsistency: libs version missing 'bank', 'crypto', 'cash', 'investment' keys; apps has all 9
3. Adding new account type requires edits in TWO places (or more)

---

### Category 2.2: Default Account Type Hardcoding

**FINDING**: Default fallback to `'bank'` account type in grouping logic.

| File                                                       | Line | Code                                |
| ---------------------------------------------------------- | ---- | ----------------------------------- |
| `apps/wealth-management/src/features/accounts/ui/page.tsx` | 133  | `const key = acc.type \|\| 'bank';` |
| `libs/wealth-management/src/features/accounts/ui/page.tsx` | 84   | `const key = acc.type \|\| 'bank';` |

**Risk**: Account classification hardcoding without validation against ACCOUNT_TYPES.

---

## 3. HARDCODED UI TEXT & LABELS

### Category 3.1: Account Type Display Labels in UI

**FINDING**: Duplicate UI labels for account grouping/display.

| File                                                       | Lines | Labels                                                        |
| ---------------------------------------------------------- | ----- | ------------------------------------------------------------- |
| `libs/wealth-management/src/features/accounts/ui/page.tsx` | 18-47 | 'Everyday Banking', 'Savings & Reserves', 'Investments', etc. |
| `apps/wealth-management/src/features/accounts/ui/page.tsx` | 51-96 | Same labels (with all 9 types)                                |

**Labels Hardcoded**:

- 'Everyday Banking' → active use
- 'Savings & Reserves' → rarely use
- 'Investments' → long holding
- 'Credit Cards & Loans' → negative active use
- 'Archived Accounts' → deprecated
- 'Bank Accounts' → bank (apps only)
- 'Crypto' → crypto (apps only)
- 'Cash' → cash (apps only)
- 'Investment' → investment (apps only)

**Context**: These labels derive from `ACCOUNT_TYPES[type].label` in config but are ALSO hardcoded in TYPE_CONFIG locally.

---

### Category 3.2: Account Type Conditional Logic in Dashboard

**FINDING**: Hardcoded account type check for special display handling.

| File                                                                   | Line | Code                                          | Context                               |
| ---------------------------------------------------------------------- | ---- | --------------------------------------------- | ------------------------------------- |
| `apps/wealth-management/src/components/dashboard/accounts-summary.tsx` | 47   | `if (account.type === 'negative active use')` | Credit card remaining balance display |

**Current Code**:

```typescript
{account.type === 'negative active use' ? (
  <>
    <div className="font-semibold text-sm text-emerald-500">
      <MaskedBalance amount={...} />
    </div>
    <p className="text-[10px] text-muted-foreground">remaining</p>
  </>
) : (
  <div className={...}>
    <MaskedBalance amount={account.balance} />
  </div>
)}
```

**Issue**: Business logic (liability display format) hardcoded. Should use `classifyAccountType()` from config.

---

## 4. HARDCODED ACCOUNT NAME REFERENCES

### Category 4.1: Specific Account Names in AI Prompts

**FINDING**: Account names (`binance`, `Binance`) hardcoded in AI instruction prompts.

| File                                                                 | Lines       | References                          |
| -------------------------------------------------------------------- | ----------- | ----------------------------------- |
| `apps/wealth-management/src/app/api/ai/account-review/route.ts`      | 39          | "Binance Earn, TCB Term Deposit"    |
| `apps/wealth-management/src/app/api/ai/budget-review/route.ts`       | (in prompt) | "Binance transfers between 1st-6th" |
| `apps/wealth-management/src/app/api/ai/transaction-review/route.ts`  | (in prompt) | "Binance transfers as wages"        |
| `apps/wealth-management/src/app/api/ai/parse-notifications/route.ts` | (in prompt) | "Binance account as income"         |

**Specific Quote (account-review/route.ts, line 39)**:

```typescript
2. Idle Cash Leakage: Identify specifically which checking/current accounts hold excessive balances
that aren't generating yield, and strongly advise moving exact amounts to high-yield or investment
accounts (e.g., Binance Earn, TCB Term Deposit).
```

**Pattern Detected**:

- Multiple AI routes reference Binance as example/context
- Used to classify income classification logic
- Hardcoded example limits AI flexibility for users with different exchanges

---

## 5. STRUCTURAL INEFFICIENCIES

### Issue 5.1: Config Not Fully Leveraged in Features

**Status**: ⚠️ MODERATE

| Location          | Expected                        | Actual                       | Gap              |
| ----------------- | ------------------------------- | ---------------------------- | ---------------- |
| UI display labels | Use `ACCOUNT_TYPES[type].label` | TYPE_CONFIG duplicate        | Not DRY          |
| Account icons     | Use `ACCOUNT_TYPES[type].icon`  | Icon map in component        | Multiple sources |
| Account ordering  | Centralized                     | TYPE_ORDER in each component | Not DRY          |
| Business rules    | Use `classifyAccountType()`     | Hardcoded checks             | Not reused       |

### Issue 5.2: Zod Schema Not Derived from Config

**Status**: ⚠️ MODERATE

The schema uses hardcoded enum values instead of deriving from `AccountType`:

```typescript
// Current (bad)
export const AccountTypeSchema = z.enum(['active use', 'rarely use', ...]);

// Should be derived from config (good)
export const AccountTypeSchema = z.enum(Object.keys(ACCOUNT_TYPES) as [string, ...string[]]);
```

### Issue 5.3: Default Account Type Not Validated

**Status**: ⚠️ LOW-MODERATE

Fallback to `'bank'` doesn't verify it exists in ACCOUNT_TYPES:

```typescript
// Current (risky)
const key = acc.type || 'bank';

// Should validate
const key = acc.type && Object.keys(ACCOUNT_TYPES).includes(acc.type) ? acc.type : getDefaultAccountType();
```

---

## 6. FILES ALREADY PROPERLY CENTRALIZED ✅

These files correctly use the config layer:

| File                                                                   | Status         | Notes                         |
| ---------------------------------------------------------------------- | -------------- | ----------------------------- |
| `libs/wealth-management/src/config/accounts/types.ts`                  | ✅ CANONICAL   | Central source of truth       |
| `libs/wealth-management/src/config/accounts/rules.ts`                  | ✅ CANONICAL   | Business logic rules          |
| `libs/wealth-management/src/config/accounts/descriptions.ts`           | ✅ CANONICAL   | Account type guidance         |
| `libs/wealth-management/src/features/accounts/model/types.ts`          | ✅ RE-EXPORTS  | Correctly imports from config |
| `libs/wealth-management/src/features/accounts/model/queries.ts`        | ✅ USES CONFIG | Leverages centralized types   |
| `apps/wealth-management/src/components/dashboard/accounts-summary.tsx` | ✅ USES CONFIG | Imports ACCOUNT_TYPES         |

---

## 7. PROPOSED REFACTORING RECOMMENDATIONS

### Priority 1: IMMEDIATE - Remove Duplications

#### 1.1 Create Centralized UI Configuration Module

**File to Create**: `libs/wealth-management/src/config/accounts/display.ts`

```typescript
/**
 * Account Type Display Configuration
 *
 * UI-specific metadata for account type display and grouping.
 * Derived from ACCOUNT_TYPES for consistency.
 */

import type { AccountType } from './types';

export interface AccountDisplayConfig {
  label: string;
  colorTitle: string;
  bgSoft: string;
}

/**
 * UI labels and styling for account type grouping on dashboard
 * Override standard labels from ACCOUNT_TYPES for UI-specific names
 */
export const ACCOUNT_TYPE_DISPLAY: Record<AccountType, AccountDisplayConfig> = {
  'active use': {
    label: 'Everyday Banking',
    colorTitle: 'text-emerald-600 dark:text-emerald-500',
    bgSoft: 'bg-emerald-500/10',
  },
  'rarely use': {
    label: 'Savings & Reserves',
    colorTitle: 'text-blue-600 dark:text-blue-500',
    bgSoft: 'bg-blue-500/10',
  },
  'long holding': {
    label: 'Investments',
    colorTitle: 'text-indigo-600 dark:text-indigo-500',
    bgSoft: 'bg-indigo-500/10',
  },
  'negative active use': {
    label: 'Credit Cards & Loans',
    colorTitle: 'text-slate-800 dark:text-slate-200',
    bgSoft: 'bg-slate-500/10',
  },
  deprecated: {
    label: 'Archived Accounts',
    colorTitle: 'text-gray-500',
    bgSoft: 'bg-gray-500/10',
  },
  bank: {
    label: 'Bank Accounts',
    colorTitle: 'text-indigo-600 dark:text-indigo-500',
    bgSoft: 'bg-indigo-500/10',
  },
  crypto: {
    label: 'Cryptocurrency',
    colorTitle: 'text-yellow-600 dark:text-yellow-500',
    bgSoft: 'bg-yellow-500/10',
  },
  cash: {
    label: 'Cash',
    colorTitle: 'text-green-600 dark:text-green-500',
    bgSoft: 'bg-green-500/10',
  },
  investment: {
    label: 'Investment Accounts',
    colorTitle: 'text-purple-600 dark:text-purple-500',
    bgSoft: 'bg-purple-500/10',
  },
};

/**
 * Default ordering for account type grouping in UI
 * Defines the visual hierarchy on accounts page
 */
export const ACCOUNT_TYPE_ORDER: AccountType[] = [
  'active use',
  'long holding',
  'rarely use',
  'negative active use',
  'deprecated',
];

/**
 * Get display config for account type with fallback
 */
export function getAccountDisplayConfig(type: string): AccountDisplayConfig {
  if (type in ACCOUNT_TYPE_DISPLAY) {
    return ACCOUNT_TYPE_DISPLAY[type as AccountType];
  }
  // Fallback to first item
  return ACCOUNT_TYPE_DISPLAY['active use'];
}
```

**Update Index**: Add to `libs/wealth-management/src/config/accounts/index.ts`:

```typescript
export {
  ACCOUNT_TYPE_DISPLAY,
  ACCOUNT_TYPE_ORDER,
  getAccountDisplayConfig,
  type AccountDisplayConfig,
} from './display';
```

---

#### 1.2 Refactor `apps/wealth-management/src/features/accounts/ui/page.tsx`

**Changes**:

- Remove local `TYPE_CONFIG` definition (lines 50-96)
- Remove local `TYPE_ORDER` definition (line 98)
- Import from config:

```typescript
import { ACCOUNT_TYPE_DISPLAY, ACCOUNT_TYPE_ORDER } from '@wealth-management/config/accounts';

// Remove these:
// const TYPE_CONFIG = { ... }
// const TYPE_ORDER = [...]

// Replace all references:
const config = ACCOUNT_TYPE_DISPLAY[type]; // instead of TYPE_CONFIG[type]
const typeOrderIndex = ACCOUNT_TYPE_ORDER.indexOf(type); // instead of TYPE_ORDER.indexOf(type)
```

**Lines to Modify**:

- Line 50-96: Delete TYPE_CONFIG
- Line 98: Delete TYPE_ORDER
- Line 141: Change `TYPE_ORDER.filter(...)` → `ACCOUNT_TYPE_ORDER.filter(...)`
- Line 142: Same import ref
- Line 318: Same import ref

---

#### 1.3 Refactor `libs/wealth-management/src/features/accounts/ui/page.tsx`

**Changes**: Same as above (remove local config, import from centralized)

**Lines to Modify**:

- Lines 17-51: Delete TYPE_CONFIG
- Line 51: Delete TYPE_ORDER
- Line 90-93: Update imports
- Line 254: Update reference
- Line 260: Update reference

---

#### 1.4 Remove Duplicate Definition from `libs/wealth-management/src/types/accounts.ts`

**Current State** (lines 1-10):

```typescript
export type AccountType = 'active use' | 'rarely use';
// ... 7 more
```

**Action**:

- Delete the `AccountType` definition from types/accounts.ts
- Import instead: `export type { AccountType } from '../config/accounts/types';`
- This ensures single source of truth

**Before** (lines 1-10):

```typescript
export type AccountType =
  | 'active use'
  | 'rarely use'
  | 'long holding'
  | 'deprecated'
  | 'negative active use'
  | 'bank'
  | 'crypto'
  | 'cash'
  | 'investment';
```

**After**:

```typescript
// Re-export from canonical config source for backward compatibility
export type { AccountType } from '../config/accounts/types';
```

**Also Delete** (lines 26-33):

```typescript
// Remove these private constants (now in config/accounts/rules.ts)
const CRYPTO_ACCOUNTS = ['binance'];
const USD_ACCOUNTS: string[] = [];

export function inferCurrency(name: string, type: string): Currency {
  const lower = name.toLowerCase();
  if (CRYPTO_ACCOUNTS.some((c) => lower.includes(c))) return 'USDT';
  if (USD_ACCOUNTS.some((c) => lower.includes(c))) return 'USD';
  return 'VND';
}
```

**Replace with**:

```typescript
// Re-export from config for backward compatibility
export { inferCurrency, CRYPTO_ACCOUNTS, USD_ACCOUNTS } from '../config/accounts/rules';
```

---

#### 1.5 Update Zod Schema to Derive from Config

**File**: `libs/wealth-management/src/schemas/accounts.ts`

**Current** (lines 7-17):

```typescript
export const AccountTypeSchema = z.enum([
  'active use',
  'rarely use',
  'long holding',
  'deprecated',
  'negative active use',
  'bank',
  'crypto',
  'cash',
  'investment',
]);
```

**Proposed** (lines 1-30):

```typescript
import { z } from 'zod';
import { ACCOUNT_TYPES, type AccountType } from '../config/accounts/types';

// ============================================================================
// Account Schemas
// ============================================================================

/**
 * Generate schema enum from config source of truth
 * Ensures schema always matches ACCOUNT_TYPES
 */
const ACCOUNT_TYPE_VALUES = Object.keys(ACCOUNT_TYPES) as [AccountType, ...AccountType[]];

export const AccountTypeSchema = z.enum(ACCOUNT_TYPE_VALUES);

export const CurrencySchema = z.enum(['VND', 'USD', 'USDT']);

// ... rest of schemas
```

**Benefit**: Schema automatically reflects new account types added to config.

---

#### 1.6 Replace Hardcoded Account Type Check in Dashboard

**File**: `apps/wealth-management/src/components/dashboard/accounts-summary.tsx`

**Current** (line 47):

```typescript
{account.type === 'negative active use' ? (
  // Special handling for liabilities
```

**Proposed** (lines 1-72):

```typescript
import { classifyAccountType } from '@wealth-management/config/accounts/rules';

// ... existing code ...

// Replace hardcoded check:
{classifyAccountType(account.type as AccountType) === 'liability' ? (
  <>
    <div className="font-semibold text-sm text-emerald-500">
      <MaskedBalance amount={...} />
    </div>
    <p className="text-[10px] text-muted-foreground">remaining</p>
  </>
) : (
  <div className={...}>
    <MaskedBalance amount={account.balance} />
  </div>
)}
```

**Benefit**: Reuses centralized business logic instead of duplicating account type knowledge.

---

### Priority 2: MEDIUM - Account Name Generalization in AI

#### 2.1 Extract Account Examples to Config

**File to Create**: `libs/wealth-management/src/config/accounts/examples.ts`

```typescript
/**
 * Account Examples for AI Context
 *
 * Provides realistic examples for AI prompts while allowing
 * customization per user's actual accounts.
 */

export interface AccountExample {
  type: 'crypto' | 'bank' | 'investment' | 'cash';
  name: string;
  description: string;
}

export const ACCOUNT_EXAMPLES: AccountExample[] = [
  {
    type: 'crypto',
    name: 'Binance',
    description: 'Major cryptocurrency exchange with earning/staking features',
  },
  {
    type: 'bank',
    name: 'TCB',
    description: 'Term deposit or fixed income account',
  },
  {
    type: 'bank',
    name: 'Checking Account',
    description: 'Primary transaction account',
  },
  {
    type: 'investment',
    name: 'Brokerage',
    description: 'Equity/ETF investment account',
  },
];

/**
 * Get yield-bearing examples for account optimization advice
 */
export function getYieldBearingExamples(): AccountExample[] {
  return ACCOUNT_EXAMPLES.filter((ex) => ex.type === 'crypto' || ex.type === 'investment');
}

/**
 * Build AI-friendly account context from user's actual accounts
 */
export function buildAccountContext(userAccounts: { name: string; type: string }[]): string {
  const examples = ACCOUNT_EXAMPLES.map((ex) => `${ex.name} (${ex.description})`).join(', ');
  return `Examples of yield-bearing accounts: ${examples}`;
}
```

#### 2.2 Refactor AI Routes to Use Config Examples

**File**: `apps/wealth-management/src/app/api/ai/account-review/route.ts`

**Current** (line 39):

```typescript
2. Idle Cash Leakage: Identify specifically which checking/current accounts hold excessive balances
that aren't generating yield, and strongly advise moving exact amounts to high-yield or investment
accounts (e.g., Binance Earn, TCB Term Deposit).
```

**Proposed**:

```typescript
import { buildAccountContext } from '@wealth-management/config/accounts/examples';

// In taskInstruction:
const accountContext = buildAccountContext(accounts);

const taskInstruction = `
  ...
  2. Idle Cash Leakage: Identify specifically which checking/current accounts hold excessive 
  balances that aren't generating yield, and strongly advise moving exact amounts to 
  high-yield or investment accounts. Examples: ${accountContext}
  ...
`;
```

**Apply Same Pattern To**:

- `apps/wealth-management/src/app/api/ai/budget-review/route.ts` (Binance wage detection)
- `apps/wealth-management/src/app/api/ai/transaction-review/route.ts` (Binance classification)
- `apps/wealth-management/src/app/api/ai/parse-notifications/route.ts` (Binance income)

---

### Priority 3: LATER - Validation Improvements

#### 3.1 Create Account Type Guard Functions

**File to Add to**: `libs/wealth-management/src/config/accounts/guards.ts` (NEW)

```typescript
/**
 * Account Type Guard Functions
 *
 * Provides type-safe validation and classification functions.
 */

import type { AccountType } from './types';
import { ACCOUNT_TYPES } from './types';

/**
 * Check if value is valid account type
 */
export function isValidAccountType(value: unknown): value is AccountType {
  return typeof value === 'string' && value in ACCOUNT_TYPES;
}

/**
 * Get default account type with validation
 */
export function getDefaultAccountType(): AccountType {
  return 'active use'; // Explicitly named instead of magic 'bank' string
}

/**
 * Get first valid account type from array
 */
export function getFirstValidAccountType(types: string[]): AccountType {
  for (const type of types) {
    if (isValidAccountType(type)) {
      return type;
    }
  }
  return getDefaultAccountType();
}

/**
 * Validate and normalize account type
 */
export function normalizeAccountType(type: string | null | undefined): AccountType {
  if (isValidAccountType(type)) {
    return type;
  }
  return getDefaultAccountType();
}
```

#### 3.2 Update Components to Use Guards

**File**: `apps/wealth-management/src/features/accounts/ui/page.tsx`

**Before** (line 133):

```typescript
const key = acc.type || 'bank';
```

**After** (line 133):

```typescript
import { normalizeAccountType } from '@wealth-management/config/accounts/guards';

// ...
const key = normalizeAccountType(acc.type);
```

---

## 8. IMPLEMENTATION PRIORITY MATRIX

| Priority | Task                                       | Files | Effort | Impact | Risk   |
| -------- | ------------------------------------------ | ----- | ------ | ------ | ------ |
| **P0**   | Create `config/accounts/display.ts`        | 1 new | 30min  | High   | Low    |
| **P0**   | Refactor 2 accounts/ui/page.tsx files      | 2     | 45min  | High   | Low    |
| **P0**   | Remove duplicate AccountType from types.ts | 1     | 15min  | Medium | Low    |
| **P0**   | Update Zod schema to derive from config    | 1     | 20min  | Medium | Low    |
| **P1**   | Replace hardcoded classifier in dashboard  | 1     | 15min  | Medium | Low    |
| **P1**   | Create `config/accounts/examples.ts`       | 1 new | 30min  | Medium | Low    |
| **P1**   | Refactor AI routes (4 files)               | 4     | 60min  | Medium | Medium |
| **P2**   | Create account type guards                 | 1 new | 30min  | Low    | Low    |
| **P2**   | Update grouping logic to use guards        | 2     | 20min  | Low    | Low    |

**Total Estimated Effort**: ~3.5-4 hours spread across 2-3 days

---

## 9. VERIFICATION CHECKLIST

After implementation, verify:

- [ ] `npm run build` passes with no errors
- [ ] `npm run lint` shows no config-related warnings
- [ ] All imports of account types use config layer
- [ ] No hardcoded account type strings outside config (except in prompts for now)
- [ ] Zod schema matches ACCOUNT_TYPES
- [ ] Two accounts/ui/page.tsx files import from same config
- [ ] Dashboard uses `classifyAccountType()` not hardcoded checks
- [ ] AI routes use `ACCOUNT_EXAMPLES` config
- [ ] Unit tests verify type guards work correctly

---

## 10. FILES SUMMARY TABLE

### Current Status by File

| File                                        | Type      | Status               | Action                         |
| ------------------------------------------- | --------- | -------------------- | ------------------------------ |
| `config/accounts/types.ts`                  | Config    | ✅ Canonical         | Keep as-is                     |
| `config/accounts/rules.ts`                  | Config    | ✅ Canonical         | Keep as-is                     |
| `config/accounts/descriptions.ts`           | Config    | ✅ Canonical         | Keep as-is                     |
| `types/accounts.ts`                         | Types     | ⚠️ Duplicate         | Simplify to re-exports         |
| `schemas/accounts.ts`                       | Schema    | ⚠️ Hardcoded         | Derive from config             |
| `features/accounts/ui/page.tsx` (libs)      | Component | ⚠️ Duplicate config  | Remove TYPE_CONFIG, TYPE_ORDER |
| `features/accounts/ui/page.tsx` (apps)      | Component | ⚠️ Duplicate config  | Remove TYPE_CONFIG, TYPE_ORDER |
| `components/dashboard/accounts-summary.tsx` | Component | ⚠️ Hardcoded logic   | Use classifyAccountType()      |
| `app/api/ai/account-review/route.ts`        | API       | ⚠️ Account names     | Use ACCOUNT_EXAMPLES           |
| `app/api/ai/budget-review/route.ts`         | API       | ⚠️ Account names     | Use ACCOUNT_EXAMPLES           |
| `app/api/ai/transaction-review/route.ts`    | API       | ⚠️ Account names     | Use ACCOUNT_EXAMPLES           |
| `app/api/ai/parse-notifications/route.ts`   | API       | ⚠️ Account names     | Use ACCOUNT_EXAMPLES           |
| `features/accounts/model/types.ts`          | Export    | ✅ Re-exports config | Keep as bridge                 |
| `features/accounts/model/queries.ts`        | Queries   | ✅ Uses types        | Keep as-is                     |

---

## 11. TESTING RECOMMENDATIONS

### Unit Tests to Add

**File**: `libs/wealth-management/src/config/accounts/__tests__/display.ts`

```typescript
describe('ACCOUNT_TYPE_DISPLAY', () => {
  it('should have entry for each account type', () => {
    Object.keys(ACCOUNT_TYPES).forEach((type) => {
      expect(ACCOUNT_TYPE_DISPLAY[type as AccountType]).toBeDefined();
    });
  });

  it('should not have orphaned display entries', () => {
    Object.keys(ACCOUNT_TYPE_DISPLAY).forEach((type) => {
      expect(ACCOUNT_TYPES[type as AccountType]).toBeDefined();
    });
  });

  it('ORDER should only include valid types', () => {
    ACCOUNT_TYPE_ORDER.forEach((type) => {
      expect(ACCOUNT_TYPES[type]).toBeDefined();
    });
  });
});
```

**File**: `libs/wealth-management/src/config/accounts/__tests__/guards.ts`

```typescript
describe('Account Type Guards', () => {
  it('isValidAccountType should accept all ACCOUNT_TYPES', () => {
    Object.keys(ACCOUNT_TYPES).forEach((type) => {
      expect(isValidAccountType(type)).toBe(true);
    });
  });

  it('normalizeAccountType should return default for invalid', () => {
    expect(normalizeAccountType('invalid')).toBe('active use');
    expect(normalizeAccountType(null)).toBe('active use');
  });
});
```

---

## 12. ROLLBACK PLAN

If issues arise during refactoring:

1. **Git branch strategy**: Create feature branch `refactor/centralize-account-config`
2. **Commit per file**: One commit per file modified (easier bisect if issues found)
3. **Verify at each step**: Run tests after each logical change
4. **Rollback command**: `git reset --hard origin/main` reverts all changes

---

## Conclusion

The codebase has **good foundational centralization** but **lacks consistency in UI layers**. The refactoring proposals address:

1. **DRY Violation**: Duplicate TYPE_CONFIG/TYPE_ORDER eliminated
2. **Single Source of Truth**: Schema, types, and UI all derive from config
3. **Type Safety**: Guards prevent invalid account types at runtime
4. **Maintainability**: Adding new account types requires changes in ONE place
5. **AI Flexibility**: Account examples extracted for better customization

**Recommended**: Implement Priority 0 items first (3 hours) for immediate impact.

---

**Report Generated**: 2026-03-18 | **Auditor**: Sisyphus-Junior (Autonomous)
