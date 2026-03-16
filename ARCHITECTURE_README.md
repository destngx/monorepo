# Architecture Documentation - Complete Reference

## 📚 Complete Documentation Package

A comprehensive 3,084-line architecture mapping of the Next.js App Router structure across the monorepo, designed to support safe refactoring while preserving routing integrity.

---

## 📖 How to Use This Documentation

### Quick Start (5 minutes)
1. **Read this file** - You're doing it! 
2. **Skim ARCHITECTURE_SUMMARY.md** - Get the big picture
3. **Bookmark ARCHITECTURE_INDEX.md** - Your navigation hub

### Deep Dive (30 minutes)
1. Read **ARCHITECTURE_MAP.md** (sections 1-4) - Understand the structure
2. Scan **ARCHITECTURE_DIAGRAMS.md** - Visualize the architecture
3. Reference **ROUTING_QUICK_REFERENCE.md** - Learn specific patterns

### For Hands-On Work
- **Creating a feature?** → **FEATURE_STRUCTURE_GUIDE.md**
- **Refactoring code?** → **ROUTING_QUICK_REFERENCE.md** + **ARCHITECTURE_MAP.md**
- **Understanding flow?** → **ARCHITECTURE_DIAGRAMS.md**
- **Finding something?** → **ARCHITECTURE_INDEX.md** (use as index)

---

## 📄 Documentation Files

### 1. ARCHITECTURE_README.md (This File)
**Purpose**: Orientation and how to use the docs

**What's Inside**:
- File directory and descriptions
- Quick start guide
- Key findings summary
- When to use each document

---

### 2. ARCHITECTURE_SUMMARY.md ⭐ START HERE
**Purpose**: Executive summary and refactoring checklist

**What's Inside**:
- What was delivered (overview)
- Key findings (routes, APIs, features)
- Abstraction opportunities
- Refactoring impact analysis
- Before/after checklist
- Next steps

**Use When**: You want the executive summary or need a refactoring checklist

**Key Sections**:
- 14 page routes documented
- 27 API endpoints by type
- 8 features identified
- 10+ abstraction opportunities assessed

**Time to Read**: 10 minutes

---

### 3. ARCHITECTURE_INDEX.md
**Purpose**: Navigation guide for all documentation

**What's Inside**:
- Document quick reference table
- How to use docs for different scenarios
- Quick facts about wealth-management
- Refactoring checklist
- File change impact matrix
- FAQ reference guide

**Use When**: You're looking for something specific and need direction

**Key Features**:
- Scenario-based navigation ("I want to refactor X")
- Quick facts table
- Comprehensive FAQ
- Reference guide by question

**Time to Read**: 5 minutes (or bookmark and reference)

---

### 4. ARCHITECTURE_MAP.md ⭐ MAIN REFERENCE
**Purpose**: Complete structure breakdown

**What's Inside**:
- Section 1: Overview of all 3 apps
- Section 2: Wealth-Management routes (14 pages)
- Section 3: Wealth-Management API routes (27 endpoints)
- Section 4: Feature structure patterns
- Section 5: Shared code architecture
- Section 6: Layout composition
- Section 7: App-specific vs shared code
- Section 8: Refactoring impact analysis

**Use When**: You need the complete picture or detailed reference

**Key Tables**:
- Page Routes (14 entries with dependencies)
- API Routes (27 entries categorized)
- Feature Structure (template pattern)
- Abstraction Opportunities (3 confidence levels)

**Time to Read**: 20-30 minutes (reference doc)

---

### 5. ROUTING_QUICK_REFERENCE.md
**Purpose**: Fast lookup for routes and dependencies

**What's Inside**:
- Route delegation flow (visual)
- Module-level dependencies (by layer)
- Critical import paths (by file type)
- Data flow examples (3 detailed walkthroughs)
- File change impact matrix
- Refactoring safety guidelines

**Use When**: You're about to change something and need to understand impact

**Key Sections**:
- Budget page load flow example
- AI budget advisor trigger flow
- Chat with tools flow
- Import path reference
- Impact matrix

**Time to Read**: 10-15 minutes (reference doc)

---

### 6. FEATURE_STRUCTURE_GUIDE.md
**Purpose**: How to build features correctly

**What's Inside**:
- Standard feature layout template
- 4 real examples from codebase
  - Budget feature (complete)
  - Accounts feature (with nesting)
  - Transactions feature (with model logic)
  - Chat feature (complex, stateful)
- 4 delegation patterns explained
- Import chain examples
- Step-by-step new feature creation (8 steps)

**Use When**: Creating new features or refactoring existing ones

**Key Content**:
- Feature structure template (boilerplate)
- 4 real-world examples with full code
- 8-step new feature creation guide
- Common import patterns

**Time to Read**: 15-20 minutes (hands-on reference)

---

### 7. ARCHITECTURE_DIAGRAMS.md
**Purpose**: Visual reference for understanding structure

**What's Inside**:
1. High-level app structure diagram
2. Request flow diagram (page navigation)
3. Request flow diagram (AI route)
4. Data flow through layers
5. Delegation pattern diagram
6. Feature dependencies graph
7. Component hierarchy diagram (root layout)
8. Import path quick reference

**Use When**: You learn better visually or need to explain to others

**Key Diagrams**:
- Full app structure (3 layers)
- Request flow for /budget page load
- Request flow for AI route
- Delegation pattern (page and API)
- Component hierarchy from root layout

**Time to Read**: 10 minutes (visual scanning)

---

## 🎯 Quick Reference Table

| Need | Document | Section |
|------|----------|---------|
| **Overview** | ARCHITECTURE_SUMMARY | Key Findings |
| **How to use docs** | ARCHITECTURE_INDEX | How to Use |
| **Route inventory** | ARCHITECTURE_MAP | Section 1-3 |
| **Feature structure** | FEATURE_STRUCTURE_GUIDE | Standard Layout |
| **Import paths** | ROUTING_QUICK_REFERENCE | Critical Paths |
| **Data flow** | ARCHITECTURE_DIAGRAMS | Request Flows |
| **Safe refactoring** | ARCHITECTURE_MAP | Section 5 |
| **New feature guide** | FEATURE_STRUCTURE_GUIDE | Creating New Feature |
| **Impact analysis** | ROUTING_QUICK_REFERENCE | Impact Matrix |
| **Visual overview** | ARCHITECTURE_DIAGRAMS | High-level Diagram |

---

## 📊 By the Numbers

- **Total documentation**: 3,084 lines
- **Total size**: ~100 KB
- **Number of files**: 7 documents
- **Page routes documented**: 14
- **API endpoints documented**: 27
- **Features documented**: 8
- **Diagrams included**: 8
- **Code examples**: 20+
- **Real feature examples**: 4

---

## 🔑 Key Architecture Insights

### Three-Layer Pattern

```
Layer 1: app/              (Routes only - thin wrappers)
    ↓
Layer 2: features/         (Implementation - where work happens)
    ↓
Layer 3: lib/ + components/ (Shared code - utilities, data access)
```

### Delegation Pattern

Routes delegate to features to keep routing layer minimal:
- Flexibility: Move feature without changing routes
- Clarity: Route file shows where implementation lives
- Maintainability: Route stays focused on routing
- Testability: Feature tested independently

### Features (8 Total)

1. **Accounts** - Banking, credit cards
2. **Budget** - Allocations, AI coaching
3. **Chat** - AI interface with tools
4. **Goals** - Goal tracking
5. **Investments** - Portfolio management
6. **Loans** - Loan tracking
7. **Settings** - User preferences
8. **Transactions** - History, categorization

### Shared Code

- **lib/sheets/** - Google Sheets data access
- **lib/ai/** - AI orchestration (providers, prompts, tools)
- **lib/utils/** - Utilities (error handling, currency, validation)
- **lib/types/** - Domain types
- **lib/constants/** - Configuration
- **components/** - Shared UI components

---

## ✅ What's Safe to Refactor

### Safe Changes
- ✅ Move `lib/ai/core/` to shared libs
- ✅ Move `lib/utils/` to shared libs
- ✅ Move `components/layout/` to shared libs
- ✅ Refactor feature implementations
- ✅ Extract new utilities

### Dangerous Changes
- ❌ Change route paths (`/app/*`)
- ❌ Change API endpoints (`/api/*`)
- ❌ Rename page exports
- ❌ Change API signatures

---

## 🚀 Getting Started with Refactoring

### Step 1: Plan
- [ ] Read ARCHITECTURE_SUMMARY.md
- [ ] Identify routes to change (ARCHITECTURE_MAP.md)
- [ ] List dependencies (ROUTING_QUICK_REFERENCE.md)
- [ ] Check abstraction safety (ARCHITECTURE_MAP.md)

### Step 2: Prepare
- [ ] Plan import path changes
- [ ] Identify all files to modify
- [ ] Check for side effects
- [ ] Plan testing strategy

### Step 3: Execute
- [ ] Update implementations (safe)
- [ ] Update import statements (safe)
- [ ] Verify barrel exports (safe)
- [ ] Test all routes (safe)

### Step 4: Verify
- [ ] Run `npm run build`
- [ ] Test all affected routes (browser)
- [ ] Check API endpoints (network)
- [ ] Run tests if available

---

## 📞 Quick Answers

**Q: Can I move a feature to a separate package?**
A: Yes, if it doesn't depend on app-specific code. See ARCHITECTURE_MAP.md section 4.

**Q: How do I create a new feature?**
A: Follow FEATURE_STRUCTURE_GUIDE.md - "Creating a New Feature" section (8 steps).

**Q: What can I abstract to shared libs?**
A: AI core, error handling, layout components. See ARCHITECTURE_MAP.md section 4.

**Q: Will refactoring break routes?**
A: Only if you change `src/app/*` paths. Otherwise, safe. See ROUTING_QUICK_REFERENCE.md.

**Q: How do I understand a data flow?**
A: See ARCHITECTURE_DIAGRAMS.md - Request Flow sections (2 detailed examples).

**Q: Where's the route definition for /budget?**
A: `src/app/budget/page.tsx` (delegates to features/budget/ui/page.tsx). See ARCHITECTURE_MAP.md section 1.2.

---

## 🔗 Document Dependencies

```
ARCHITECTURE_SUMMARY (entry point)
    ↓ refers to
ARCHITECTURE_INDEX (navigation)
    ↓ refers to
├─ ARCHITECTURE_MAP (main reference)
├─ ROUTING_QUICK_REFERENCE (fast lookup)
├─ FEATURE_STRUCTURE_GUIDE (implementation)
└─ ARCHITECTURE_DIAGRAMS (visual reference)
```

**Recommended reading order**:
1. This file (ARCHITECTURE_README)
2. ARCHITECTURE_SUMMARY (big picture)
3. Use other docs as references based on task

---

## 📝 Keeping Docs Up to Date

As code changes:
- Update ARCHITECTURE_MAP.md when routes/APIs change
- Update FEATURE_STRUCTURE_GUIDE.md for new patterns
- Update ARCHITECTURE_DIAGRAMS.md for structural changes
- Update ARCHITECTURE_INDEX.md for new tools

---

## 🎓 Using This for Onboarding

New team members should:
1. Read ARCHITECTURE_SUMMARY.md (10 min)
2. Skim ARCHITECTURE_DIAGRAMS.md (10 min)
3. Study FEATURE_STRUCTURE_GUIDE.md (20 min)
4. Reference other docs as needed (30+ min)

Total: 60-90 minutes to understand architecture

---

## Questions?

Refer to:
- **General**: ARCHITECTURE_SUMMARY.md
- **Routes**: ARCHITECTURE_MAP.md
- **Specific code**: FEATURE_STRUCTURE_GUIDE.md
- **Visual**: ARCHITECTURE_DIAGRAMS.md
- **Navigation**: ARCHITECTURE_INDEX.md

---

**Ready to refactor safely!** 🚀

Start with ARCHITECTURE_SUMMARY.md, then use other docs as references.

