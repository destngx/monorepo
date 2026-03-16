# Wealth-Management FSD → Monorepo Integration Refactoring Plan

**Status**: Strategy Approved & Ready for Execution  
**Target**: Refactor wealth-management app to integrate into monorepo structure while maintaining FSD, clean architecture, DRY, and SOLID principles  
**Timeline**: 4-5 working days (Wave 1-4)  
**Risk Level**: LOW (Metis-approved approach)

---

## EXECUTIVE SUMMARY

**Current State**: Fully working Next.js 16 app with internal FSD structure

- 8 features (loans, accounts, budget, chat, goals, investments, settings, transactions)
- ~100+ feature files
- 20+ shared UI components
- Multiple cross-feature dependencies

**Target State**: Monorepo-integrated FSD with:

- Features remain in-app (no feature→package migration yet)
- Shared code extracted to monorepo libs (`@wealth-management/ui`, `@wealth-management/types`, `@wealth-management/utils`)
- Clear dependency boundaries enforced with ESLint
- API route business logic extracted to reusable libs
- Zero circular dependencies

**Why This Approach**:

1. ✅ Features stay in-app → minimal breaking changes
2. ✅ Extract only genuinely shared code → faster iteration
3. ✅ DRY violations fixed immediately (ui, types, utils)
4. ✅ Preserves current app structure → no routing changes
5. ✅ Foundation for future feature→package migration (Wave 2+)

---

## STRATEGIC DECISIONS (LOCKED)

| Decision                | Choice                                                                             | Rationale                                                            |
| ----------------------- | ---------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| **Feature Ownership**   | Keep features in-app for now; defer feature→package migration                      | FSD already isolates; packages add overhead without ROI              |
| **API Abstraction**     | Extract business logic to libs; keep routing in app                                | Next.js Request/Response tight coupling; logic extraction sufficient |
| **Path Aliases**        | `@wealth-management/{ui,types,utils,schemas}`                                      | Domain-scoped, scales if multiple apps share code                    |
| **ESLint Enforcement**  | Enable `@nx/enforce-module-boundaries`                                             | Prevents accidental cross-feature imports; catches circular deps     |
| **Type Centralization** | `@wealth-management/types` for models; `@wealth-management/schemas` for validators | Single source of truth; prevents duplication                         |
| **Lib Layout**          | Per-type grouping (`ui/`, `types/`, `utils/`) under `libs/wealth-management-*/`    | Clear ownership; mirrors FSD intent                                  |

---

## CURRENT CODE INVENTORY (from exploration)

### Features (8 total)

```
src/features/
├── accounts/     (5 files: ui/pages, model/hooks, model/types, model/mutations)
├── loans/        (6 files: ui/components, model/hooks, model/types, model/mutations, model/queries)
├── budget/       (5 files: ui/components, model/hooks, model/mutations, model/types)
├── chat/         (8 files: ui/components, model/hooks, model/types, api delegation)
├── goals/        (6 files: ui/pages, model/hooks, model/types, model/mutations)
├── investments/  (4 files: ui/components, model/types)
├── settings/     (5 files: ui/pages, model/hooks, model/mutations, model/types)
└── transactions/ (7 files: ui/components, model/hooks, model/types, model/queries)
```

### Shared Code (to extract)

```
src/shared/
├── lib/
│   ├── ai/              → @wealth-management/ai (AI providers, streaming logic)
│   ├── persistence/     → @wealth-management/persistence (storage adapters)
│   └── validation/      → @wealth-management/schemas (Zod validators)
├── ui/                  → @wealth-management/ui (6 components - currently unused)
└── lib/index.ts         → @wealth-management/utils (helpers)

src/components/          → @wealth-management/ui (20+ components)
├── dashboard/           (feature-specific, stays in app if used only locally)
├── chat/                (feature-specific, can move to @wealth-management/ui)
├── layout/              (genuinely shared, → @wealth-management/ui/layout)
└── ui/                  (21 base components, → @wealth-management/ui)

src/lib/                 → Split across multiple libs
├── ai/                  → @wealth-management/ai
├── services/            → @wealth-management/services
├── types/               → @wealth-management/types
├── utils/               → @wealth-management/utils
├── sheets/              → @wealth-management/sheets (Google Sheets integration)
└── constants/           → @wealth-management/constants
```

### API Routes (15+ routes)

```
src/app/api/
├── /accounts       → features/accounts/model
├── /loans          → features/loans/model
├── /goals          → features/goals/model
├── /transactions   → features/transactions/model
├── /tags           → shared lib (create @wealth-management/tags)
├── /chat           → features/chat/model (with streaming)
├── /ai/*           → @wealth-management/ai (5 AI providers)
└── /exchange-rate  → @wealth-management/services (external API)
```

### Cross-Feature Dependencies Found

```
transactions → accounts         (for account list, balance updates)
goals → transactions            (for transaction queries)
budget → transactions           (for spending queries)
chat → all features             (for data aggregation)
```

---

## EXECUTION PLAN (4 WAVES)

### WAVE 1: Monorepo Infrastructure Setup (2-3 hours)

**Goal**: Prepare monorepo for lib integration. Zero code moves; configuration only.

#### 1.1 Configure Path Aliases (30 mins)

**File**: `/tsconfig.base.json`
Add to `compilerOptions.paths`:

```json
{
  "@wealth-management/types": ["libs/wealth-management-types/src/index.ts"],
  "@wealth-management/types/*": ["libs/wealth-management-types/src/*"],
  "@wealth-management/schemas": ["libs/wealth-management-schemas/src/index.ts"],
  "@wealth-management/schemas/*": ["libs/wealth-management-schemas/src/*"],
  "@wealth-management/ui": ["libs/wealth-management-ui/src/index.ts"],
  "@wealth-management/ui/*": ["libs/wealth-management-ui/src/*"],
  "@wealth-management/utils": ["libs/wealth-management-utils/src/index.ts"],
  "@wealth-management/utils/*": ["libs/wealth-management-utils/src/*"],
  "@wealth-management/services": ["libs/wealth-management-services/src/index.ts"],
  "@wealth-management/services/*": ["libs/wealth-management-services/src/*"],
  "@wealth-management/ai": ["libs/wealth-management-ai/src/index.ts"],
  "@wealth-management/ai/*": ["libs/wealth-management-ai/src/*"]
}
```

#### 1.2 Update ESLint Configuration (30 mins)

**File**: `/.eslintrc.json`
Add enforcement rules (after explore confirms structure):

```json
{
  "overrides": [
    {
      "files": ["apps/wealth-management/**/*.{ts,tsx}"],
      "rules": {
        "@nx/enforce-module-boundaries": [
          "error",
          {
            "allow": [
              "@wealth-management/types",
              "@wealth-management/schemas",
              "@wealth-management/ui",
              "@wealth-management/utils",
              "@wealth-management/services",
              "@wealth-management/ai"
            ],
            "depConstraints": [
              {
                "sourceTag": "feature:*",
                "onlyDependOnLibsWithTags": ["lib:shared"]
              }
            ]
          }
        ]
      }
    }
  ]
}
```

#### 1.3 Create Lib Directory Structure (1 hour)

Create skeleton libs (no files moved yet):

```bash
mkdir -p libs/wealth-management-types/src
mkdir -p libs/wealth-management-schemas/src
mkdir -p libs/wealth-management-ui/src
mkdir -p libs/wealth-management-utils/src
mkdir -p libs/wealth-management-services/src
mkdir -p libs/wealth-management-ai/src

# Create package.json for each lib (based on task-management pattern)
```

**Template `package.json` for each lib**:

```json
{
  "name": "@wealth-management/types",
  "version": "0.0.1",
  "private": true,
  "type": "module",
  "exports": {
    ".": "./src/index.ts"
  },
  "types": "./src/index.ts",
  "dependencies": {}
}
```

#### 1.4 Create Nx Project Configuration (30 mins)

**File**: `libs/wealth-management-types/project.json`

```json
{
  "name": "wealth-management-types",
  "sourceRoot": "libs/wealth-management-types/src",
  "projectType": "library",
  "targets": {
    "lint": {
      "executor": "@nx/linter:eslint",
      "options": {
        "lintFilePatterns": ["libs/wealth-management-types/**/*.{ts,tsx}"]
      }
    },
    "test": {
      "executor": "@nx/jest:jest",
      "options": {
        "jestConfig": "libs/wealth-management-types/jest.config.ts"
      }
    }
  },
  "tags": ["lib:shared", "scope:wealth-management"]
}
```

---

### WAVE 2: Extract Shared Code to Libs (6-8 hours)

**Goal**: Move code from `src/shared/`, `src/components/`, `src/lib/` to monorepo libs.

#### 2.1 Extract Type Definitions → `@wealth-management/types` (2 hours)

**Source**: Scan `src/lib/types/` and `src/features/*/model/types.ts`

**Action**: Copy all type definitions to `libs/wealth-management-types/src/`

```
libs/wealth-management-types/src/
├── index.ts                    (re-exports all)
├── accounts.ts                 (Account, Loan, CreditCard types)
├── transactions.ts             (Transaction, Category types)
├── goals.ts                    (Goal, Milestone types)
├── budget.ts                   (Budget types)
├── chat.ts                     (Message, Conversation types)
├── investments.ts              (Investment, Portfolio types)
├── common.ts                   (shared types: ID, Timestamp, etc)
└── __tests__/
    └── index.test.ts           (verify exports)
```

**Validation**: No circular imports; each file imports only from:

- Other files in this lib
- `zod` (for schema inference if needed)
- External libs

#### 2.2 Extract Validators → `@wealth-management/schemas` (2 hours)

**Source**: Scan `src/shared/lib/validation/` and `src/features/*/model/` for Zod schemas

**Action**: Copy all validators to `libs/wealth-management-schemas/src/`

```
libs/wealth-management-schemas/src/
├── index.ts                    (re-exports all)
├── accounts.ts                 (Account, Loan validation)
├── transactions.ts             (Transaction validation)
├── goals.ts                    (Goal validation)
├── budget.ts                   (Budget validation)
└── common.ts                   (shared schemas)
```

**Dependencies**:

- Imports from `@wealth-management/types` (for type inference)
- Imports `zod`

#### 2.3 Extract UI Components → `@wealth-management/ui` (2 hours)

**Source**: `src/components/ui/` and `src/shared/ui/`

**Action**: Copy all UI components to `libs/wealth-management-ui/src/`

```
libs/wealth-management-ui/src/
├── index.ts                    (re-exports all)
├── button.tsx
├── card.tsx
├── input.tsx
├── select.tsx
├── dialog.tsx
├── badge.tsx
├── chart.tsx
├── layout/
│   ├── sidebar.tsx
│   ├── header.tsx
│   └── footer.tsx
├── dashboard/                  (if genuinely shared)
└── __tests__/
    └── button.test.tsx
```

**Dependencies**:

- Imports from `@wealth-management/types` (for prop types)
- React, Radix UI, Tailwind (external)
- NOT from features

#### 2.4 Extract Utilities → `@wealth-management/utils` (1.5 hours)

**Source**: `src/lib/utils/`, `src/shared/lib/`

**Action**: Copy all utility functions

```
libs/wealth-management-utils/src/
├── index.ts
├── math.ts                    (currency, percentage helpers)
├── format.ts                  (date, currency formatting)
├── validation.ts              (common validators)
├── string.ts                  (slugify, truncate, etc)
└── array.ts                   (groupBy, unique, etc)
```

**Dependencies**:

- date-fns, clsx (external)
- Minimal internal deps

#### 2.5 Extract Services → `@wealth-management/services` (1.5 hours)

**Source**: `src/lib/services/`

**Action**: Copy service layers that handle external API calls

```
libs/wealth-management-services/src/
├── index.ts
├── exchange-rate.ts           (currency API)
├── google-sheets.ts           (Sheets integration)
└── external-api.ts            (any other 3rd party)
```

**Keep in app** (server actions):

- `/src/app/actions/*` - stays in app (Next.js specific)

#### 2.6 Extract AI Providers → `@wealth-management/ai` (1 hour)

**Source**: `src/lib/ai/` and `src/app/api/ai/*`

**Action**: Extract provider logic to lib

```
libs/wealth-management-ai/src/
├── index.ts
├── providers/
│   ├── anthropic.ts           (Claude)
│   ├── openai.ts              (GPT)
│   ├── google.ts              (Gemini)
│   └── azure.ts               (Azure OpenAI)
├── streaming.ts               (streaming helpers)
└── prompts/                   (system prompts)
    ├── account-review.ts
    ├── budget-advisor.ts
    └── ... (others)
```

**Keep in app**:

- `/src/app/api/ai/*` route handlers (these delegate to `@wealth-management/ai`)

---

### WAVE 3: Update Imports & Break Cross-Feature Dependencies (4-6 hours)

**Goal**: Update all code to use new lib aliases; eliminate feature-to-feature imports.

#### 3.1 Update Feature Imports (3 hours)

**For each feature in `src/features/`:**

**Before**:

```typescript
import { AccountType } from '../../lib/types';
import { formatCurrency } from '../../lib/utils';
import { Button } from '../../components/ui';
import { TransactionTable } from '../transactions/ui/transaction-table'; // ❌ CROSS-FEATURE
```

**After**:

```typescript
import { AccountType } from '@wealth-management/types';
import { formatCurrency } from '@wealth-management/utils';
import { Button } from '@wealth-management/ui';
import { useTransactions } from '../transactions/model/hooks'; // ✅ Only if must share
```

**Action**: Run automated import updates via AST tool

```bash
# This will be done using ast_grep_replace in implementation phase
```

#### 3.2 Break Cross-Feature Dependencies (2 hours)

**Found dependencies**:

```
❌ transactions → accounts (imports Account type)
❌ goals → transactions (imports Transaction type)
❌ budget → transactions (imports Transaction type)
❌ chat → all features (queries all data)
```

**Solution**: Create shared "bridge" types in `@wealth-management/types`

**Example**:

```typescript
// @wealth-management/types/index.ts
export type { Account, Loan, CreditCard } from './accounts';
export type { Transaction, Category, Tag } from './transactions';
export type { Goal, Milestone } from './goals';

// features/transactions/model/types.ts
export type { Transaction } from '@wealth-management/types';

// features/budget/model/mutations.ts
import { Transaction } from '@wealth-management/types'; // ✅ From types lib, not feature
```

#### 3.3 Update Shared Component Imports (1 hour)

**From**:

```typescript
import { DashboardCard } from '../components/dashboard-card';
import { Dialog } from '../shared/ui/dialog';
```

**To**:

```typescript
import { DashboardCard, Dialog } from '@wealth-management/ui';
```

---

### WAVE 4: Verification & Testing (2-3 hours)

**Goal**: Ensure refactoring is complete, no breaking changes, no circular dependencies.

#### 4.1 Run Linting (30 mins)

```bash
pnpm lint --fix
```

Verify:

- ✅ No ESLint errors in wealth-management features
- ✅ No type errors in features
- ✅ No undefined imports

#### 4.2 Check Circular Dependencies (30 mins)

```bash
nx dep-graph --file=wealth-management-deps.json
```

Verify graph has NO cycles:

- Features → types lib only
- Features → ui/utils/services libs only
- No feature → feature imports
- AI lib → types/utils only

#### 4.3 Run Type Checking (30 mins)

```bash
pnpm tsc --noEmit
```

Verify: Zero type errors in apps/wealth-management

#### 4.4 Build Test (30 mins)

```bash
nx build wealth-management
```

Verify: Build succeeds, bundles include only necessary libs

#### 4.5 Unit Tests (1 hour)

```bash
nx test wealth-management
```

Verify: All existing tests pass (no changes needed in tests)

#### 4.6 Visual Verification (30 mins)

- [ ] Dev server starts: `pnpm dev`
- [ ] All routes load without errors
- [ ] Features work as before (UI, API calls)
- [ ] No console errors/warnings

---

## FILE MOVEMENT SUMMARY

### Moved to Libs (No Code Changes)

```
src/lib/types/*                    → libs/wealth-management-types/src/*
src/lib/utils/*                    → libs/wealth-management-utils/src/*
src/shared/lib/validation/*        → libs/wealth-management-schemas/src/*
src/shared/lib/persistence/*       → libs/wealth-management-services/src/*  (partially)
src/shared/lib/ai/*                → libs/wealth-management-ai/src/*
src/components/ui/*                → libs/wealth-management-ui/src/ui/*
src/shared/ui/*                    → libs/wealth-management-ui/src/shared/*
src/lib/ai/*                       → libs/wealth-management-ai/src/*
src/lib/sheets/*                   → libs/wealth-management-services/src/sheets/*
src/lib/constants/*                → libs/wealth-management-utils/src/constants/*
```

### Stays in App (No Move)

```
src/app/*                          (routing, layouts, Next.js specific)
src/features/*/model/*             (hooks, queries, mutations)
src/features/*/ui/*                (page components)
src/hooks/*                        (app-level hooks if any)
```

### Updated in App (Import Changes Only)

```
src/features/**/*.{ts,tsx}         (all files: update imports)
src/lib/services/*                 (some: already abstract enough)
src/app/actions/*                  (some: update imports)
src/app/api/*                      (some: delegation patterns)
```

---

## RISK MITIGATION

| Risk                                 | Likelihood | Impact | Mitigation                                                |
| ------------------------------------ | ---------- | ------ | --------------------------------------------------------- |
| **Breaking import changes**          | Medium     | High   | Automated AST rewrites; careful validation                |
| **Type inference breaks**            | Low        | Medium | Comprehensive type checking; re-exports in index.ts       |
| **Circular dependencies introduced** | Medium     | High   | Strict ESLint boundaries; dep-graph verification          |
| **Build/bundle size issues**         | Low        | Low    | Tree-shaking works same; Nx optimizes automatically       |
| **Feature tests fail**               | Low        | Medium | Tests unchanged; routes unchanged; should pass            |
| **Performance regression**           | Low        | Medium | Libs tree-shake same; no bundler changes; monitor metrics |

---

## SUCCESS CRITERIA

**Phase complete when ALL of the following are TRUE:**

- [ ] All path aliases resolve correctly (tsc pass)
- [ ] Zero circular dependencies in nx dep-graph
- [ ] All ESLint rules pass (no boundary violations)
- [ ] Wealth-management app builds successfully
- [ ] All existing tests pass
- [ ] Dev server starts and routes load
- [ ] No console errors in browser
- [ ] No type errors reported by IDE
- [ ] All features work as before (manual smoke test)

---

## NEXT STEPS

1. **Review & Confirm** this plan with team
2. **Start WAVE 1** - infrastructure setup
3. **Start WAVE 2** - mechanical code extraction
4. **Run WAVE 3** - import updates (automated + manual verification)
5. **Complete WAVE 4** - comprehensive validation

**Estimated total time**: 12-18 hours of actual work (can be done in 2-3 days)

---

## DEFERRED DECISIONS (NOT BLOCKING)

These can be revisited after core refactoring:

- [ ] Feature→Package migration (defer until 2+ apps consume a feature)
- [ ] Advanced API abstraction patterns (defer until 3+ apps have APIs)
- [ ] Performance optimization (defer until metrics indicate need)
- [ ] Monorepo package versioning/releases (defer until publishing)
- [ ] Advanced E2E testing infrastructure (defer until needed)

---

## QUESTIONS FOR TEAM CONFIRMATION

Before starting implementation:

1. ✅ Do you agree features should stay in-app (defer feature→package migration to Wave 2)?
2. ✅ Are path aliases `@wealth-management/*` acceptable, or prefer different naming?
3. ✅ Is ESLint boundary enforcement acceptable (prevents cross-feature imports)?
4. ✅ Are there any features that MUST be reusable across multiple apps now?
5. ✅ Should E2E tests be updated during this refactor, or after?

---

**Plan prepared by**: Metis + Sisyphus exploration tasks  
**Date**: 2026-03-16  
**Status**: Ready for execution
