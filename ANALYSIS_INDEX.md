# Wealth Management App - Feature Analysis Index

> **Complete feature structure analysis for monorepo integration**

## 📚 Documentation Files

### 1. **FEATURE_STRUCTURE_MAP.md** (Primary Reference)
The most comprehensive breakdown. Start here to understand everything.

**Contains:**
- Executive summary with statistics
- Detailed breakdown of all 8 features
- File listings for each feature
- Model layer specifications (types, queries, mutations, hooks)
- UI component inventory
- Cross-feature dependencies
- Shared infrastructure overview
- Refactoring roadmap (5 phases)
- Circular dependency analysis
- Import patterns (safe vs unsafe)

**Best For:** Understanding complete feature organization and refactoring plan

---

### 2. **DEPENDENCY_GRAPH.md** (Visual Reference)
Diagrams and visual representations of the architecture.

**Contains:**
- Feature dependency tree (full file structure)
- Data flow diagram
- Import dependency flows
- Monorepo target structure
- Migration path for path aliases
- Circular dependency risk map
- API route organization
- Feature maturity matrix

**Best For:** Visual learners, understanding data flow, planning monorepo structure

---

### 3. **QUICK_REFERENCE.md** (Lookup Guide)
Tables and checklists for quick navigation.

**Contains:**
- Feature overview tables (sorted by file count, dependencies, complexity)
- Dependency graph (text version)
- File distribution breakdown
- Migration checklist (5 phases)
- Current state assessment
- Complete file list by feature
- Key exports reference
- Common tasks and procedures
- Quick navigation commands

**Best For:** Quick lookups, migration planning, remembering commands

---

## 🎯 How to Use These Documents

### If you're NEW to the codebase:
1. Start with **QUICK_REFERENCE.md** - Feature overview tables
2. Read **FEATURE_STRUCTURE_MAP.md** - Executive summary section
3. Check **DEPENDENCY_GRAPH.md** - Data flow diagram

### If you're planning MIGRATION:
1. Review **FEATURE_STRUCTURE_MAP.md** - Refactoring roadmap section
2. Use **QUICK_REFERENCE.md** - Migration checklist
3. Reference **DEPENDENCY_GRAPH.md** - Target structure

### If you're ADDING A FEATURE:
1. Check **FEATURE_STRUCTURE_MAP.md** - Architecture patterns section
2. Use **QUICK_REFERENCE.md** - Common tasks section
3. Reference any similar feature in FEATURE_STRUCTURE_MAP.md

### If you're DEBUGGING DEPENDENCIES:
1. Use **DEPENDENCY_GRAPH.md** - Circular dependency risk map
2. Check **FEATURE_STRUCTURE_MAP.md** - Cross-feature dependencies section
3. Run commands in **QUICK_REFERENCE.md** - Check dependencies section

### If you're REFACTORING:
1. Review **FEATURE_STRUCTURE_MAP.md** - Files to move section
2. Follow **QUICK_REFERENCE.md** - Migration checklist
3. Reference **DEPENDENCY_GRAPH.md** - Path aliases migration

---

## 📊 Key Statistics at a Glance

| Metric | Value |
|--------|-------|
| Total Features | 8 |
| Total Files | 112 |
| Features with API routes | 4 |
| Cross-feature imports | 2 (type-only) |
| Circular dependencies | 0 (SAFE) |
| Estimated code lines | 5,500+ |

---

## 🔗 Feature Dependency Summary

```
CORE FEATURE:
  Transactions (imported by Accounts & Budget)

STANDALONE FEATURES (No external feature dependencies):
  Chat, Goals, Loans, Investments, Settings

DEPENDENT FEATURES (Import from Transactions):
  Accounts (3 UI components)
  Budget (2 UI components)
```

---

## 🛠️ Quick Start Commands

### View all features
```bash
ls apps/wealth-management/src/features/
```

### Check feature structure
```bash
tree apps/wealth-management/src/features/[feature-name]/
```

### Find cross-feature imports
```bash
grep -r "from.*features/" apps/wealth-management/src/features/
```

### Check specific feature's imports
```bash
grep -r "^import" apps/wealth-management/src/features/[feature]/model/
```

See **QUICK_REFERENCE.md** section "Quick Navigation" for more commands.

---

## 🚀 Refactoring Phases

### Phase 1: Core Types & Sheets (1-2 days) ⭐ START HERE
Extract types and data access layer to monorepo libs

**Files to move:**
- `src/lib/types/*` → `libs/common/types/`
- `src/lib/sheets/*` → `libs/sheets/`

### Phase 2: Utilities & Constants (1 day)
Extract utilities and constants

**Files to move:**
- `src/lib/utils/*` → `libs/common/utils/`
- `src/lib/constants/*` → `libs/common/constants/`

### Phase 3: AI & Services (0.5 day)
Extract specialized services

**Files to move:**
- `src/lib/ai/*` → `libs/ai/`
- `src/lib/services/*` → `libs/services/`

### Phase 4: Verify & Optimize (1 day)
Test and validate everything works

### Phase 5: Optional - Centralize API (Future)
Consider moving API routes to `src/app/api/`

---

## 📋 Files NOT to Move

Keep these in the app (feature-specific):
- Feature model hooks (queries, mutations)
- Feature UI components
- Feature-specific business logic
- Feature API routes (for now)

---

## ⚠️ Critical Rules

### ✅ DO THESE:
- ✅ Import types between features (if needed)
- ✅ Import from @/lib/*
- ✅ Use barrel exports (index.ts)
- ✅ Keep model layer isolated
- ✅ Extract shared code to libs

### ❌ NEVER DO THESE:
- ❌ Import hooks/queries across features
- ❌ Create circular dependencies
- ❌ Import UI components across features
- ❌ Use deep path imports
- ❌ Add new cross-feature dependencies without review

---

## 📁 Current Project Structure

```
apps/wealth-management/src/
├── features/
│   ├── accounts/      (16 files)
│   ├── budget/        (13 files)
│   ├── chat/          (17 files)
│   ├── goals/         (14 files)
│   ├── investments/   (10 files)
│   ├── loans/         (11 files)
│   ├── settings/      (7 files)
│   └── transactions/  (14 files)
└── lib/
    ├── ai/
    ├── constants/
    ├── services/
    ├── sheets/
    ├── types/
    └── utils/
```

---

## 🎓 Understanding the Architecture

### Model Layer (Per Feature)
```
feature/model/
├── types.ts        # Interfaces & types
├── queries.ts      # Read operations
├── mutations.ts    # Write operations
├── hooks.ts        # React hooks
└── index.ts        # Barrel export
```

### UI Layer (Per Feature)
```
feature/ui/
├── page.tsx        # Main page
├── component1.tsx  # Feature components
├── component2.tsx
└── index.ts        # Barrel export
```

### API Layer (Optional)
```
feature/api/
└── route.ts        # Next.js API route
```

---

## 🔍 Dependency Decision Tree

**Q: Can I import something from another feature?**

→ Is it a TYPE only? → ✅ YES (type imports are safe)
→ Is it a HOOK/QUERY? → ❌ NO (implementation not allowed)
→ Is it a UI COMPONENT? → ❌ NO (components stay in feature)

**Q: Should I extract shared code?**

→ Is it used by 2+ features? → ✅ MOVE to libs/
→ Is it used by 1 feature? → ⏭️ KEEP in feature
→ Is it a TYPE used by 2+ features? → ✅ MOVE to libs/common/types

---

## 📞 Support References

For each feature, check FEATURE_STRUCTURE_MAP.md for:
- Purpose statement
- File inventory
- Model layer functions
- UI components
- Dependencies
- External APIs used

---

## 🎯 Next Actions

1. **Read** FEATURE_STRUCTURE_MAP.md (main document)
2. **Review** DEPENDENCY_GRAPH.md (understand flow)
3. **Plan** using QUICK_REFERENCE.md (migration checklist)
4. **Execute** Phase 1: Extract types and sheets
5. **Validate** with build and tests

---

**Last Updated:** Mar 16, 2026
**Analysis Scope:** apps/wealth-management (Complete)
**Status:** Ready for monorepo integration

