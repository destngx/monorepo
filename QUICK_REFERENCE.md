# Wealth Management Feature Structure - Quick Reference

## 📊 Feature Overview (Sortable by Key Metric)

### By File Count
| Feature | Files | Type | API | Hooks |
|---------|-------|------|-----|-------|
| Chat | 17 | Standalone | ❌ | ✅ |
| Accounts | 16 | Depends: Transactions | ✅ | ✅ |
| Goals | 14 | Standalone | ❌ | ✅ |
| Transactions | 14 | Core (exported) | ✅ | ✅ |
| Budget | 13 | Depends: Transactions | ✅ | ✅ |
| Loans | 11 | Standalone | ❌ | ✅ |
| Investments | 10 | Standalone | ✅ | ✅ |
| Settings | 7 | Standalone | ❌ | ✅ |

### By Dependencies
| Feature | Imports From | Imported By | Risk |
|---------|-------------|-----------|------|
| Transactions | None | Accounts, Budget | Core |
| Accounts | Transactions (type) | None | Low |
| Budget | Transactions (type) | None | Low |
| Chat | @/lib/ai | None | Low |
| Goals | None | None | Low |
| Loans | None | None | Low |
| Investments | None | None | Low |
| Settings | None | None | Low |

### By Complexity
| Feature | UI Components | Model Functions | Complexity | Priority |
|---------|---------------|-----------------|-----------|----------|
| Chat | 10 | 8 | High | Medium |
| Accounts | 8 | 9 | Medium | High |
| Budget | 5 | 6 | Medium | High |
| Transactions | 6 | 7 | Medium | **Critical** |
| Goals | 8 | 9 | Medium | Medium |
| Loans | 4 | 8 | Low | Low |
| Investments | 2 | 7 | Medium | Medium |
| Settings | 1 | 4 | Low | Low |

---

## 🔗 Dependency Graph (Text)

```
Transactions (CORE)
    ├─→ Accounts (imports Transaction type)
    └─→ Budget (imports Transaction type)

Chat (STANDALONE)
    └─→ @/lib/ai/providers

Goals, Loans, Investments, Settings (STANDALONE)
    └─→ @/lib/sheets/* (data access only)
```

---

## 📁 File Distribution

```
accounts/:
  ├── model/ (7 files)      - Types, Queries, Mutations, Hooks, Index
  ├── ui/ (8 files)         - Page, 4 AI components, Credit components, Index
  └── api/ (1 file)         - GET endpoint

budget/:
  ├── model/ (7 files)      - Types, Queries, Mutations, Hooks, Index
  ├── ui/ (5 files)         - Page, Overview, Detail, AI Review, AI Advisor, Index
  └── api/ (1 file)         - GET endpoint

transactions/:
  ├── model/ (7 files)      - Types, Queries, Mutations, Hooks, Index
  ├── ui/ (6 files)         - Page, Form, Table, Filters, Processor, AI Review, Index
  └── api/ (1 file)         - GET/POST endpoints

chat/:
  ├── model/ (6 files)      - Types, Queries, Mutations, Hooks, Index
  └── ui/ (10 files)        - FAB, Drawer, Widget, Interface, Container, Messages, Input, Switcher, Card, Provider, Index

goals/:
  ├── model/ (7 files)      - Types, Queries, Mutations, Hooks, Index
  └── ui/ (8 files)         - Page, Card, Detail Page, Chart, New Page, Flow, AI Panel, AI Card, Index

loans/:
  ├── model/ (8 files)      - Types, Queries, Mutations, Hooks, Server Queries, Index
  └── ui/ (4 files)         - Page, List, Summary, AI Review, Index

investments/:
  ├── model/ (7 files)      - Types, Queries, Mutations, Hooks, Index
  ├── ui/ (1 file)          - Page
  └── api/ (2 files)        - Assets, Prices endpoints

settings/:
  ├── model/ (6 files)      - Types, Queries, Mutations, Hooks, Index
  └── ui/ (1 file)          - Page
```

---

## 🎯 Migration Checklist

### Phase 1: Core Types & Sheets (1-2 days)
- [ ] Create `libs/common/types/`
- [ ] Move 8 type files from `src/lib/types/`
- [ ] Create `libs/sheets/`
- [ ] Move all sheet readers from `src/lib/sheets/`
- [ ] Update imports in all features
- [ ] Verify TypeScript compilation

### Phase 2: Utilities & Constants (1 day)
- [ ] Create `libs/common/utils/`
- [ ] Move validation, date, currency, api-error-handler
- [ ] Create `libs/common/constants/`
- [ ] Move categories, tags, navigation
- [ ] Update imports

### Phase 3: AI & Services (Half day)
- [ ] Create `libs/ai/`
- [ ] Create `libs/services/`
- [ ] Move providers and price-service
- [ ] Update imports

### Phase 4: Update tsconfig & Verify (1 day)
- [ ] Update `tsconfig.base.json` with new paths
- [ ] Run full build
- [ ] Run all tests
- [ ] Check Nx dependency graph

### Phase 5: Optional - Centralize API Routes (Future)
- [ ] Consider moving API routes to `src/app/api/`
- [ ] Evaluate middleware needs

---

## 🚀 Current State Assessment

### ✅ Strengths
1. **Clean Architecture**: All features follow FSD pattern
2. **No Circular Dependencies**: Dependency graph is acyclic
3. **Type Safety**: Type imports only between features
4. **Testability**: Model layer is isolated and testable
5. **Consistency**: Uniform model/ui/api structure

### ⚠️ Improvement Areas
1. **Duplication**: Types exist in both features and /lib
2. **Coupling**: Some features duplicate type definitions
3. **Documentation**: Need clearer import guidelines
4. **Testing**: Missing unit tests in several features

### 🔴 Risks to Avoid
1. ❌ Never import hooks/queries across features
2. ❌ Never create bidirectional imports
3. ❌ Never import UI components across features
4. ❌ Never add new cross-feature dependencies without review

---

## 📋 Complete File List

### By Feature
```
ACCOUNTS (16 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/page.tsx, account-review-ai.tsx, account-trend-sparkline.tsx (3)
  ui/credit-card-summary-ai.tsx, efficiency-chart.tsx (2)
  ui/credit/due-date-countdown.tsx, statement-cycle-bar.tsx (2)
  ui/credit/utilization-ring.tsx, index.ts (2)
  api/route.ts (1)

BUDGET (13 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/page.tsx, budget-overview-view.tsx, category-detail-view.tsx (3)
  ui/budget-review-ai.tsx, ai-budget-advisor-view.tsx, index.ts (3)
  api/route.ts (1)

TRANSACTIONS (14 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/page.tsx, transaction-form.tsx, transaction-table.tsx (3)
  ui/transaction-filters.tsx, notification-processor.tsx (2)
  ui/transaction-review-ai.tsx, index.ts (2)
  api/route.ts (1)

CHAT (17 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/ai-fab.tsx, ai-drawer.tsx, ai-chat-widget.tsx (3)
  ui/chat-interface.tsx, chat-container.tsx, chat-messages.tsx (3)
  ui/chat-input.tsx, model-switcher.tsx, ai-insight-card.tsx (3)
  ui/ai-context-provider.tsx, index.ts (2)

GOALS (14 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/page.tsx, goal-card.tsx, goal-detail-page.tsx (3)
  ui/goal-detail-chart.tsx, new-goal-page.tsx, create-goal-flow.tsx (3)
  ui/ai-insights-panel.tsx, ai-summary-card.tsx, index.ts (3)

LOANS (11 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts (4)
  model/server-queries.ts, index.ts (2)
  ui/page.tsx, loan-list.tsx, loan-summary.tsx (3)
  ui/loan-review-ai.tsx, index.ts (2)

INVESTMENTS (10 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/page.tsx, index.ts (2)
  api/assets/route.ts, prices/route.ts (2)

SETTINGS (7 files)
  model/types.ts, queries.ts, mutations.ts, hooks.ts, index.ts (5)
  ui/page.tsx, index.ts (2)

TOTAL: 112 files
```

---

## 🔑 Key Exports (What Other Features Use)

```
From Transactions:
  ✅ Transaction interface (19 fields)
  ✅ TransactionType ('income' | 'expense' | 'non-budget')
  Used by: Accounts (3 components), Budget (2 components)

From Accounts:
  ✅ Account interface
  ✅ AccountType (9 variants)
  ✅ Currency type
  Used by: None (only imported by API layer)

From Budget:
  ✅ BudgetItem interface
  Used by: None

From Chat:
  ✅ ChatMessage, ChatResponse, AIInsight
  Used by: None (self-contained)

From Goals:
  ✅ Goal, GoalType, GoalProjection
  Used by: None

From Loans:
  ✅ Loan interface
  Used by: None

From Investments:
  ✅ Investment, PortfolioAnalysis
  Used by: None

From Settings:
  ✅ Settings interface
  Used by: None
```

---

## 🛠️ Common Tasks

### Add a New Feature
1. Create `src/features/[name]/`
2. Follow structure: `model/` (types, queries, mutations, hooks, index.ts)
3. Create `ui/` with page.tsx and components
4. Add `index.ts` barrel export
5. Add API routes if needed
6. Export from feature index.ts
7. Add to navigation

### Add a Cross-Feature Import (Type Only)
1. Only import types from `feature/model/types.ts`
2. Never import queries/mutations/hooks
3. Document in code comments
4. Consider moving shared type to `libs/`

### Refactor to Monorepo
1. Move types: `src/lib/types/*` → `libs/common/types/`
2. Move sheets: `src/lib/sheets/*` → `libs/sheets/`
3. Move utils: `src/lib/utils/*` → `libs/common/utils/`
4. Update all imports in features
5. Update `tsconfig.base.json` paths
6. Verify build and tests

---

## 📚 Documentation Files

Located in monorepo root:
- `FEATURE_STRUCTURE_MAP.md` - Detailed breakdown of all features
- `DEPENDENCY_GRAPH.md` - Visual diagrams and dependency flows
- `QUICK_REFERENCE.md` - This file

---

## 🔍 Quick Navigation

### Find a Feature
```bash
# List all features
ls apps/wealth-management/src/features/

# Show feature structure
tree apps/wealth-management/src/features/[name]/

# Find imports from other features
grep -r "from.*features/" apps/wealth-management/src/features/[name]/
```

### Check Dependencies
```bash
# Show all imports in a feature
grep -r "^import" apps/wealth-management/src/features/[name]/model/

# Find circular dependencies
grep -r "from.*features/" apps/wealth-management/src/ | grep -E "\.(ts|tsx):"
```

### Update Imports (for Migration)
```bash
# Find all @/lib/types imports
grep -r "@/lib/types" apps/wealth-management/src/

# Find all @/lib/sheets imports
grep -r "@/lib/sheets" apps/wealth-management/src/

# Find all feature cross-imports
grep -r "from.*features/" apps/wealth-management/src/features/
```

---

## 💡 Remember

**The Golden Rules:**
1. ✅ Features are independent modules
2. ✅ Only types are shared between features
3. ✅ All shared code goes to `libs/`
4. ✅ Model layer (hooks/queries) is feature-specific
5. ✅ UI components never cross feature boundaries
6. ❌ Never import implementation across features
7. ❌ Never create circular dependencies

**For Monorepo Success:**
- Extract shared types early
- Keep feature-specific logic in features
- Use barrel exports (index.ts) consistently
- Document all cross-feature imports
- Use Nx to track dependency graph

