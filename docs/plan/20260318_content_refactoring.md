# Content & Configuration Centralization Refactoring

**Date**: 2026-03-18  
**Objective**: Move all prompts, task instructions, and account-related configuration from the app layer to the libs layer, eliminating hardcoded content from application logic.

---

## Architecture

### Current State

- Prompts: `libs/wealth-management/src/ai/prompts/` (domain-based: budget, dashboard, finance, market, system)
- Account types: `libs/wealth-management/src/features/accounts/model/types.ts`
- Hardcoded content: Scattered across app routes and components
- No centralized config for account descriptors, rules, or metadata

### Target State

```
libs/wealth-management/src/
├── ai/
│   ├── prompts/          (existing domain-based prompts)
│   └── server.ts         (server-safe exports)
├── config/               (NEW - centralized config)
│   ├── accounts/
│   │   ├── types.ts      (move AccountType enum + metadata)
│   │   ├── descriptions.ts (descriptions for each account type)
│   │   └── rules.ts      (account rules, inference logic)
│   ├── transactions/
│   │   ├── categories.ts (transaction category definitions)
│   │   └── rules.ts      (auto-tagging, categorization rules)
│   ├── budget/
│   │   └── defaults.ts   (default budget categories, thresholds)
│   └── index.ts          (central export)
└── content/              (NEW - text/instructions/copy for tasks)
    ├── tasks/            (task descriptions, instructions)
    ├── messages/         (success/error messages, alerts)
    └── index.ts          (central export)
```

### Features Folder (unchanged)

- Features import from `@wealth-management/config` instead of defining enums locally
- API routes and hooks remain thin, just orchestration

---

## Tasks

### Phase 1: Discover & Plan (Current - 1h)

- [x] Audit existing hardcoded data in app/
- [x] Audit existing config in libs/features/accounts/
- [x] Map what needs to move where
- [ ] Document data flow changes needed in consuming code

### Phase 2: Create Config Layer (2-3h)

#### Task 2.1: Create accounts config structure

- [ ] Create `libs/wealth-management/src/config/accounts/types.ts`
  - Move `AccountType` enum from `features/accounts/model/types.ts`
  - Add metadata interface: `{ id, label, icon, color, description }`
  - Create `ACCOUNT_TYPES` constant with full metadata

- [ ] Create `libs/wealth-management/src/config/accounts/descriptions.ts`
  - Human-readable descriptions for each account type
  - Usage guidelines and best practices
  - Export `ACCOUNT_TYPE_DESCRIPTIONS` map

- [ ] Create `libs/wealth-management/src/config/accounts/rules.ts`
  - Move `CRYPTO_ACCOUNTS`, `USD_ACCOUNTS` from types.ts
  - Move `inferCurrency()` function
  - Add account classification rules
  - Export `ACCOUNT_RULES` config object

- [ ] Create `libs/wealth-management/src/config/transactions/categories.ts`
  - Define all valid transaction categories
  - Category metadata (icon, color, type)
  - Export `TRANSACTION_CATEGORIES` map

- [ ] Create `libs/wealth-management/src/config/transactions/rules.ts`
  - Auto-tagging rules (keyword patterns → tags)
  - Categorization rules (keywords/patterns → category)
  - Export `TAG_RULES`, `CATEGORY_RULES`

- [ ] Create `libs/wealth-management/src/config/index.ts`
  - Re-export all config items for convenience
  - Single source of truth

#### Task 2.2: Update features to use config

- [ ] Update `libs/wealth-management/src/features/accounts/model/types.ts`
  - Remove `AccountType` enum (now imported from config)
  - Keep interface types (Account, Currency)
  - Keep `inferCurrency()` function (or move to config if pure utility)
  - Export types from config for backward compat

- [ ] Update `libs/wealth-management/src/features/accounts/model/queries.ts`
  - Use `ACCOUNT_RULES` from config instead of local constants
  - No behavioral changes, just imports

- [ ] Update any other features files that reference account types
  - Replace direct enum references with config imports

### Phase 3: Create Content Layer (1-2h)

#### Task 3.1: Create task instructions/content

- [ ] Create `libs/wealth-management/src/content/tasks/index.ts`
  - Export all task-related text (instructions, descriptions)
  - Keep structure flat for now (no subdirs yet)

- [ ] Create `libs/wealth-management/src/content/messages/index.ts`
  - Success messages for account actions
  - Error message templates
  - Alert/notification text

- [ ] Create `libs/wealth-management/src/content/index.ts`
  - Central export point

### Phase 4: Update App to Use Config (1-2h)

- [ ] Audit all API routes in `apps/wealth-management/src/app/api/`
  - Find hardcoded strings, categories, account types
  - List files to update

- [ ] Update API routes to import from `@wealth-management/config`
  - Budget advisor route (if using account metadata)
  - Transaction review route (if using category config)
  - Category suggestion route

- [ ] Update features in `apps/wealth-management/src/features/`
  - Remove hardcoded category/account data
  - Import from `@wealth-management/config`

- [ ] Verify no hardcoded account types in components
  - UI should not know AccountType enum
  - Get metadata from config instead

### Phase 5: Validation & Testing (1h)

- [ ] Build libs: `nx build wealth-management` (should pass)
- [ ] Build app: `bun run build wealth-management` (should pass)
- [ ] Verify no new type errors introduced
- [ ] Manual QA: Check account pages, transaction forms load correctly

---

## Notes

### Design Principles

1. **One source of truth**: All config defined once in libs
2. **No circular dependencies**: App imports from libs, never reverse
3. **Server-safe**: Account config is safe to use in both client & server contexts (no fs, no secrets)
4. **Extensible**: Adding new account types = one change in config, automatic everywhere

### Data Flow Example: Account Type Display

**Before (scattered)**:

- `features/accounts/model/types.ts` has enum
- Component imports enum directly
- Account descriptions hardcoded in UI
- Icon/color logic in component

**After (centralized)**:

- `config/accounts/types.ts` has enum + metadata
- Component imports `ACCOUNT_TYPES` from config
- Gets label, description, icon, color from metadata
- Component stays simple, just displays

### Files to Keep Unchanged

- `libs/wealth-management/src/ai/prompts/` — already domain-organized
- `libs/wealth-management/src/types/` — domain types (Account, Transaction, etc.)
- `libs/wealth-management/src/services/` — API integration logic
- `libs/wealth-management/src/utils/` — utilities (formatting, calculations)

### Files Affected

- `features/accounts/model/types.ts` — imports will change
- App routes that hardcode data — will import from config
- Any component using account/category enums — will import from config

---

## Deliverables

1. **libs/wealth-management/src/config/** directory with:
   - `accounts/types.ts`, `accounts/descriptions.ts`, `accounts/rules.ts`
   - `transactions/categories.ts`, `transactions/rules.ts`
   - `index.ts` (main export)

2. **libs/wealth-management/src/content/** directory with:
   - `tasks/index.ts`
   - `messages/index.ts`
   - `index.ts` (main export)

3. **Updated imports** in:
   - All app API routes
   - Features files
   - Components using account/category types

4. **Zero hardcoded content** in app layer
   - All config/enums imported from libs
   - All text/instructions imported from libs/content

5. **Build passes** — no type errors, no bundle issues

---

## Apply Order

1. Create config layer structure (tasks 2.1 + 2.2)
2. Create content layer structure (task 3.1)
3. Update all imports in app & features (task 4)
4. Build & verify (task 5)

---

## References

- Existing prompts: `libs/wealth-management/src/ai/prompts/`
- Existing account types: `libs/wealth-management/src/features/accounts/model/types.ts`
- App structure: `apps/wealth-management/src/`
- Current issue: Build passes but app has scattered hardcoded data
