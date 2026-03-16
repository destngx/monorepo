# Architecture Documentation Index

## Overview

This directory contains comprehensive documentation of the Next.js App Router architecture used across the monorepo. These documents will help with refactoring while preserving routing structure.

---

## 📄 Documents

### 1. **ARCHITECTURE_MAP.md** (Main Reference)

**What**: Complete structure breakdown of all 3 Next.js apps

**Contains**:

- Overall routing patterns (three-layer delegation)
- All page routes and their dependencies
- All API routes with categorization
- Feature structure breakdown
- Shared code architecture (lib/ organization)
- Layout composition
- App-specific vs. shared code analysis
- Abstraction opportunities with confidence levels

**Use When**: You need the big picture or to understand how a specific route works

**Key Sections**:

- Wealth-Management routes table (14 pages)
- Wealth-Management API routes (27 endpoints categorized)
- Feature structure documentation
- Abstraction opportunity matrix

---

### 2. **ROUTING_QUICK_REFERENCE.md** (Developer Cheatsheet)

**What**: Fast lookup guide for route delegation and dependencies

**Contains**:

- Route delegation flow diagrams
- Module-level dependencies
- Critical import paths
- Data flow examples
- File change impact matrix
- Refactoring safety guidelines

**Use When**: You're about to make a change and need to understand impact

**Key Sections**:

- Budget page load flow example
- AI advisor trigger flow example
- Chat with tools flow example
- Import path reference

---

### 3. **FEATURE_STRUCTURE_GUIDE.md** (Implementation Template)

**What**: How to structure a feature and create new ones

**Contains**:

- Standard feature layout template
- 4 real examples (Budget, Accounts, Transactions, Chat)
- Delegation pattern details (4 patterns explained)
- Import chain examples
- Step-by-step guide to creating a new feature

**Use When**: Creating a new feature or refactoring existing one

**Key Sections**:

- Budget feature full example
- Accounts feature example
- Creating a new feature (8-step guide)

---

### 4. **ARCHITECTURE_DIAGRAMS.md** (Visual Reference)

**What**: ASCII diagrams of architecture and data flows

**Contains**:

- High-level app structure diagram
- Request flow diagrams (2 examples)
- Data flow through layers
- Delegation pattern diagram
- Feature dependencies graph
- Component hierarchy diagram
- Import path quick reference

**Use When**: You need visual understanding of how pieces fit together

**Key Diagrams**:

- Request flow for page navigation
- Request flow for AI route
- Full delegation pattern
- Component hierarchy from root

---

## 🎯 How to Use This Documentation

### Scenario 1: "I need to understand the current structure"

1. Start with **ARCHITECTURE_MAP.md** - read sections 1-4
2. Look at **ARCHITECTURE_DIAGRAMS.md** for visual overview
3. Reference **ROUTING_QUICK_REFERENCE.md** for specific routes

### Scenario 2: "I want to refactor the budget feature"

1. Check **FEATURE_STRUCTURE_GUIDE.md** - Budget example section
2. Find all dependencies in **ARCHITECTURE_MAP.md** - Budget API table
3. Check import impact in **ROUTING_QUICK_REFERENCE.md** - Import paths

### Scenario 3: "I'm creating a new feature"

1. Follow **FEATURE_STRUCTURE_GUIDE.md** - "Creating a new feature" section
2. Use **ARCHITECTURE_MAP.md** to see similar features
3. Reference **ROUTING_QUICK_REFERENCE.md** for import patterns

### Scenario 4: "I'm moving code to a shared lib"

1. Check **ARCHITECTURE_MAP.md** - Section 4 "Abstraction Opportunities"
2. Find all usages in **ARCHITECTURE_DIAGRAMS.md** - Dependency graph
3. Verify impact in **ROUTING_QUICK_REFERENCE.md** - Impact matrix

### Scenario 5: "I broke a route - where's the problem?"

1. Find the route in **ARCHITECTURE_MAP.md** - Routes table
2. Follow dependency chain in **ROUTING_QUICK_REFERENCE.md**
3. Check delegation in **ARCHITECTURE_DIAGRAMS.md** - Delegation pattern

---

## 📊 Quick Facts About Wealth-Management

- **Routes**: 14 user-facing pages
- **API endpoints**: 27 documented
- **Features**: 8 (accounts, budget, chat, goals, investments, loans, settings, transactions)
- **Shared code**: lib/ (sheets, ai, utils, types, constants) + components/
- **Architecture pattern**: Three-layer delegation (app/ → features/ → lib/)
- **No nested features**: Each feature is sibling (goals is separate, not nested under accounts)

---

## 🔄 Refactoring Checklist

Before refactoring, use these docs to:

- [ ] Identify all routes affected (check ARCHITECTURE_MAP.md tables)
- [ ] Map all dependencies (check ROUTING_QUICK_REFERENCE.md imports)
- [ ] Understand delegation chain (check ARCHITECTURE_DIAGRAMS.md)
- [ ] Check abstraction safety (check ARCHITECTURE_MAP.md section 4)
- [ ] Plan import path changes (check ROUTING_QUICK_REFERENCE.md)
- [ ] Test all affected routes (use file impact matrix)

---

## 🛡️ Preservation Rules

When refactoring, preserve:

✅ **MUST NOT CHANGE**:

- Route paths in `src/app/` (e.g., `/budget`, `/accounts`)
- API endpoint paths in `src/app/api/` (e.g., `/api/budget`)
- Page component export names from features

✅ **CAN CHANGE**:

- Internal implementation in `features/*/`
- Library imports (move to shared libs)
- Component imports (as long as exports stay consistent)
- Internal file organization within features

---

## 📝 Key Insights

### What Can Be Abstracted (High Confidence)

- `lib/ai/core/` - AI orchestration patterns
- `lib/utils/` - Error handling and utilities
- `components/layout/` - Layout component patterns
- Type definition system - `lib/types/` structure

### What Cannot Be Abstracted

- Google Sheets integration (wealth-specific)
- Financial domain logic
- Feature-specific UI pages
- Cross-feature aggregation APIs

### What Might Be Abstracted (Medium Confidence)

- Feature-specific AI routes (if pattern generalizes)
- Chat integration (if tools become generic)
- Dashboard components (if made generic enough)

---

## 🔗 Related Concepts

### Feature-Based Architecture

- Each feature owns its routes, handlers, UI, and logic
- Features are self-contained but can import types from other features
- `model/` folder contains domain types and business logic
- `api/` folder contains request handlers
- `ui/` folder contains React components

### Three-Layer Pattern

```
app/                  → Route definitions only
features/[feature]/   → Feature implementation
lib/ + components/    → Shared utilities and components
```

### Delegation Pattern

Routes in `app/` are thin wrappers that import from `features/`:

- Keeps routing logic centralized
- Allows feature movement without path changes
- Makes implementation location obvious

---

## 💡 Tips for Refactoring

1. **Update imports first**: Map out all import changes before editing
2. **Test routing**: Verify all routes still work after changes
3. **Check barrel exports**: Ensure `index.ts` files still export correctly
4. **Preserve API signatures**: Don't change request/response shapes
5. **Document changes**: Update import paths in code comments
6. **Use LSP**: Let your editor help find all usages

---

## 📞 Reference Guide

| Question                   | Document                | Section                    |
| -------------------------- | ----------------------- | -------------------------- |
| What routes exist?         | ARCHITECTURE_MAP        | 1.2                        |
| How do API routes work?    | ARCHITECTURE_MAP        | 1.3                        |
| What can I abstract?       | ARCHITECTURE_MAP        | 4                          |
| How do I create a feature? | FEATURE_STRUCTURE       | Creating a New Feature     |
| What are import paths?     | ROUTING_QUICK_REFERENCE | Critical Import Paths      |
| Show me a flow diagram     | ARCHITECTURE_DIAGRAMS   | Request Flow Diagrams      |
| How do routes delegate?    | ARCHITECTURE_DIAGRAMS   | Delegation Pattern Diagram |
| Which files import what?   | ROUTING_QUICK_REFERENCE | Import Chain Examples      |
| Is this safe to move?      | ARCHITECTURE_MAP        | 5 (Refactoring Safety)     |

---

## Version & Last Updated

- **Last Updated**: 2024
- **Covers**: Wealth-Management (primary), Portfolio-Landpage, Cloudinary-Photos-App
- **Architecture Pattern**: Next.js 13+ App Router with feature-based organization
- **Status**: Comprehensive - ready for refactoring work

---

## Questions or Additions?

If these docs don't answer your question about the architecture, check:

1. The actual files mentioned (they're all linked from docs)
2. Code comments in the implementation files
3. Test files to see how components are used
4. The monorepo README for project-level context

---
