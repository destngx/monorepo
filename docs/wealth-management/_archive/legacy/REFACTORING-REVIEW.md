# FSD Refactoring Plan - BUSINESS-FOCUSED REVIEW

## Executive Summary

Your wealth management system is **AI-driven assistant for personal wealth & assets management** with three core pillars:

1. **Budget Management** - Spending tracking, budget planning, category analysis
2. **Accounts Management** - Bank accounts, credit cards, loans, goals, savings tracking
3. **Investment Management** - Portfolio tracking, asset management, net worth analysis

The refactoring plan aligns perfectly with these pillars. This document reviews the architecture plan against your actual product requirements and suggests minor refinements.

---

## ✅ What's Correct About the Plan

### 1. Core Feature Domains Match

The plan identifies 8 features, and your actual product focuses on:

```
Core (must keep isolated):
├── budget/          ✅ Budget tracking, category management, analysis
├── accounts/        ✅ Bank accounts, net worth, balance tracking
├── loans/           ✅ Debt tracking, repayment planning
├── goals/           ✅ Savings goals, progress tracking
├── investments/     ✅ Portfolio, net worth components
├── credit-cards/    ✅ Credit card tracking (sub-feature of accounts)
├── chat/            ✅ AI Financial Advisor (critical for your USP)
└── transactions/    ✅ Daily spending data (feeds budget, accounts, AI)
```

**Verdict**: Perfect alignment. These are the right boundaries.

---

### 2. AI as Cross-Cutting Concern is Critical

Your plan correctly identifies AI as a **unified service layer**:

```
Current (scattered):
├── 13 AI API routes in /api/ai/*
├── AI prompts scattered in /lib/ai/prompts/
├── AI calls from components, API routes, services
└── No consistent interface

Target (unified):
└── shared/lib/ai/service.ts
    ├── Unified interface for all AI calls
    ├── Organized prompts by domain
    ├── Provider abstraction (OpenAI, Google, Anthropic)
    └── Streaming support
```

**Why this matters for you**: Your USP is the **AI Financial Advisor**. Centralizing AI service ensures:

- ✅ Consistent behavior across budget/transaction/investment analysis
- ✅ Easy provider switching (you support 3 providers)
- ✅ Streaming responses for chat interface
- ✅ Reusable prompts across domains

**Verdict**: Essential. Don't skip this.

---

### 3. Repository Pattern for Data Abstraction

Your app uses **Google Sheets as headless database**. The plan recommends Repository pattern:

```
Current (tightly coupled):
├── components/ → directly imports lib/sheets/accounts.ts
├── api routes/ → directly imports lib/sheets/transactions.ts
└── Can't switch persistence without touching 50+ files

Target (abstracted):
└── shared/lib/persistence/Repository
    ├── Repository.accounts.findAll()
    ├── Repository.transactions.create()
    └── Adapter pattern: SheetsAdapter implements Repository
```

**Why this matters for you**:

- ✅ If you add database (PostgreSQL) later, only adapter changes
- ✅ Testing: Can mock Repository without touching Sheets
- ✅ Performance: Can add caching layer transparently
- ✅ Multi-sheet sync: Can handle split data sources

**Verdict**: Important for your architecture. Sheets is your current persistence, but abstracting it makes the app future-proof.

---

## 🎯 Refinements to the Plan (Business-Focused)

### 1. Prioritize Chat Feature (Week 1-2)

**Recommendation**: Move Chat/AI consolidation to **Week 2** (not Week 4)

```
Current Plan:
├── Week 1: Infrastructure (Repository, AI, Validation)
├── Week 2: Settings, Loans, Accounts
├── Week 3: Transactions, Budget, Goals
├── Week 4: AI consolidation, Investments, Chat
└── Week 5: Cleanup

Better Plan (for your product):
├── Week 1: Infrastructure (Repository, AI, Validation) - UNCHANGED
├── Week 1.5: Chat feature consolidation - MOVE UP
├── Week 2: Loans, Accounts (foundation)
├── Week 3: Transactions, Budget
├── Week 4: Goals, Investments
└── Week 5: Cleanup & testing
```

**Why**: Your AI Financial Advisor is your **primary differentiator**. Consolidating AI early means:

- ✅ Chat works perfectly before other features are migrated
- ✅ All AI features have unified interface
- ✅ Streaming works consistently everywhere
- ✅ Can showcase AI capabilities while refactoring continues

---

### 2. Group Accounts Features Together

Your "Accounts" domain actually includes multiple sub-features:

```
Current Plan treats as separate:
├── accounts/           (bank/crypto accounts)
├── credit-cards/       (separate)
└── loans/             (separate)

Better Plan (business perspective):
features/accounts/                  ← "Accounts & Liabilities"
├── api/
├── ui/
│   ├── bank-accounts/
│   ├── credit-cards/
│   ├── loans/
│   ├── account-summary.tsx
│   ├── net-worth.tsx
│   └── accounts-page.tsx
├── model/
│   ├── types.ts                   (Account, CreditCard, Loan)
│   ├── queries.ts                 (getAccounts, getLoans, getCreditCards)
│   ├── mutations.ts
│   └── hooks.ts
└── __tests__/
```

**Why grouping matters**:

- ✅ Net worth calculation needs all three together
- ✅ Account balance sync affects all three
- ✅ Single source of truth for "money" in system
- ✅ Reduces cross-feature dependencies

**Implementation note**: Keep separate API routes (`/api/accounts`, `/api/loans`, `/api/credit-cards`) but they all delegate to `features/accounts/model/`

---

### 3. Investments ≠ Accounts

Keep these separate because:

```
Accounts/Liabilities:
├── Track: Money flow (in/out), current balances, debts
├── Goals: Budget compliance, debt reduction
├── AI analysis: Spending patterns, budget advice

Investments:
├── Track: Asset growth, portfolio composition
├── Goals: Wealth growth, asset allocation
├── AI analysis: Portfolio rebalancing, performance

Separate concerns = easier testing & maintenance
```

---

### 4. Chat Feature Structure

Since Chat is critical, here's how to structure it:

```
features/chat/
├── api/
│   ├── route.ts                    (main chat endpoint)
│   └── schemas.ts                  (chat message validation)
├── ui/
│   ├── chat-interface.tsx          (main chat UI)
│   ├── chat-messages.tsx
│   ├── chat-input.tsx
│   ├── ai-fab.tsx                  (floating action button)
│   └── index.ts
├── model/
│   ├── types.ts                    (ChatMessage, ChatContext)
│   ├── queries.ts                  (getConversation, getContext)
│   ├── handlers.ts                 (message handling logic)
│   └── context-builders.ts         (build financial context for AI)
├── lib/
│   ├── utils.ts
│   └── constants.ts
└── __tests__/
```

**Key point**: Context builders prepare financial data for AI:

- Get user's accounts, transactions, budgets
- Format as context for AI prompt
- Include account balances, spending trends, etc.

---

## 🔄 Dependency Graph (Your Product)

Here's how features should depend on each other:

```
┌─────────────────────────────────────────────────────────┐
│  Chat (AI Financial Advisor) - depends on all data     │
├─────────────────────────────────────────────────────────┤
│               ↓        ↓        ↓                       │
├──────────────────────────────────────────────────────────┤
│  Budget   │  Accounts  │  Investments  │  Transactions  │
├──────────────────────────────────────────────────────────┤
│                    ↓                                     │
├──────────────────────────────────────────────────────────┤
│          Shared (Repository, Validation, Utils)        │
├──────────────────────────────────────────────────────────┤
│             ↓           ↓            ↓                 │
├──────────────────────────────────────────────────────────┤
│  AI Service  │  Google Sheets  │  Auth  │  Cache       │
└─────────────────────────────────────────────────────────┘

Rules:
✓ Chat can see all domains
✓ Domains cannot see Chat
✓ Domains can see Shared
✓ No cross-domain imports (Budget ≠ Accounts)
✓ All persistence through Repository
```

---

## 📊 Revised Timeline (Business-Optimized)

```
Week 1: Infrastructure + AI (CRITICAL PATH)
├─ Day 1-2: Create shared/, core/, features/ folders
├─ Day 2-3: Repository abstraction (with tests)
├─ Day 3-4: Unified AI service layer
├─ Day 4-5: Validation schemas, ESLint rules
└─ Day 5: Chat feature consolidation START

Week 2: Chat Completion + Accounts Foundation
├─ Day 1-3: Complete Chat feature migration
├─ Day 3-5: Migrate Loans (no dependencies)
├─ Day 5: START Accounts

Week 3: Accounts + Transactions
├─ Day 1-3: Complete Accounts migration
├─ Day 3-5: Migrate Transactions (depends on Accounts)

Week 4: Budget + Investments
├─ Day 1-3: Migrate Budget (depends on Transactions)
├─ Day 3-5: Migrate Investments (independent)

Week 5: Goals + Cleanup
├─ Day 1-2: Migrate Goals (depends on Accounts)
├─ Day 2-5: Remove old structure, full testing, documentation

Timeline benefits:
✓ Chat (your USP) is perfect from Week 2 onward
✓ Can ship AI improvements while refactoring continues
✓ Accounts/Budget/Transactions work together by Week 3
✓ Full system working by Week 5
```

---

## 🧠 Feature Specifics for Your AI Layer

### What Your AI Service Should Handle

```ts
shared/lib/ai/service.ts should provide:

AIService.analyze({
  domain: 'budget' | 'accounts' | 'transactions' | 'investments',
  action: 'review' | 'analysis' | 'suggestion' | 'advice',
  data: any,
  context?: UserContext,
}): Promise<string>
```

### Domain-Specific Prompts

```
shared/lib/ai/prompts/
├── budget/
│   ├── advisor.ts              "How can I reduce spending?"
│   ├── review.ts               "Review my spending this month"
│   └── insight.ts              "What are my trends?"
├── accounts/
│   ├── review.ts               "Review my account health"
│   ├── forecast.ts             "Project my cash flow"
│   └── recommendation.ts       "Should I adjust my accounts?"
├── transactions/
│   ├── categorization.ts       "Categorize this transaction"
│   ├── analysis.ts             "Analyze my spending patterns"
│   └── suggestions.ts          "Suggest improvements"
├── investments/
│   ├── analysis.ts             "Analyze my portfolio"
│   ├── rebalancing.ts          "Suggest rebalancing"
│   └── performance.ts          "How's my portfolio doing?"
└── general/
    ├── system-prompt.ts        "You are a financial advisor..."
    └── tools.ts                "Define AI tools (fetch accounts, etc)"
```

### Multi-Model Support

Your app supports 3 providers. The unified service should:

```ts
// User can switch from Settings
// Should work seamlessly across all features

AIService.analyze({
  domain: 'budget',
  action: 'advice',
  data: userBudgets,
  provider?: 'openai' | 'google' | 'anthropic',  // Optional, uses default
})
```

---

## 🔒 Data Privacy & Security Considerations

Since you're handling personal financial data:

### In Repository Layer

```ts
// All data access goes through Repository
// Can add encryption/decryption here transparently
Repository.accounts.findAll()
  → Decrypt from sheets/database
  → Return to caller
```

### In AI Service

```ts
// Should NOT send raw data to AI unnecessarily
// Aggregate/summarize instead
// Example: Don't send all 500 transactions
// Instead: "200 transactions, avg $50, top category is food"

AIService.analyze({
  domain: 'transactions',
  data: {
    // Summarized, not raw
    totalCount: 200,
    avgAmount: 50,
    topCategories: [...]
  }
})
```

---

## ✅ Validation: Does the Plan Support Your Product?

### Budget Management Feature

```
✓ Can isolate budget business logic in features/budget/model/
✓ AI can analyze spending via AIService
✓ Transactions feed budget data
✓ Google Sheets sync through Repository
✓ Category management organized in Budget domain
```

### Accounts Management Feature

```
✓ Bank accounts, crypto, loans in single features/accounts/
✓ Net worth calculation in model/queries.ts
✓ Multiple currencies handled consistently
✓ AI can review account health
✓ Google Sheets sync through Repository
```

### Investments Feature

```
✓ Separate from Accounts (correct separation)
✓ Portfolio analysis in features/investments/model/
✓ AI can suggest rebalancing
✓ Can extend with real-time market data
✓ Google Sheets sync through Repository
```

### AI Financial Advisor

```
✓ Unified entry point: shared/lib/ai/service.ts
✓ All features can use AIService
✓ Multi-provider support built in
✓ Streaming for chat interface
✓ Context builders pull data from Repository
```

### Verdict: ✅ EXCELLENT ALIGNMENT

The refactoring plan supports all your core product features without compromise.

---

## 🚀 Quick Start (Revised)

For your team:

1. **Read**: `docs/README-REFACTORING.md` (5 min)
2. **Study**: `docs/REFACTORING-PLAN.md` Part 1-2 (30 min)
3. **Review**: This document (the business alignment) (15 min)
4. **Plan**: Adjusted timeline above (30 min)
5. **Execute**: `docs/IMPLEMENTATION-ROADMAP.md` with Week 1-2 reordering

---

## 📋 Checklist: Before Starting Refactoring

### Architecture Approved?

- [ ] Reviewer: Does FSD match our product vision?
  - Answer: **YES** ✅
- [ ] Reviewer: Are domain boundaries correct?
  - Answer: **YES** ✅ (Budget, Accounts/Loans, Investments, Chat, Transactions)
- [ ] Reviewer: Is AI properly abstracted?
  - Answer: **YES** ✅ (Unified service, organized prompts)
- [ ] Reviewer: Is Google Sheets properly abstracted?
  - Answer: **YES** ✅ (Repository pattern)

### Team Alignment?

- [ ] Engineers understand FSD principles
- [ ] Engineers understand your feature domains
- [ ] Engineers can see path from current → target architecture
- [ ] Engineers agree on ESLint rules

### Execution Ready?

- [ ] Week 1-5 timeline approved (with Chat moved to Week 2)
- [ ] Team capacity allocated
- [ ] Git workflow decided
- [ ] Testing requirements clear
- [ ] Rollback plan understood

---

## Final Verdict

**The refactoring plan is well-suited to your product.**

The only adjustments are **optimization** not **correction**:

1. Move Chat consolidation to Week 2 (it's your USP)
2. Group Accounts/Loans/CreditCards together (they're related)
3. Emphasize unified AI service (it's critical to your value proposition)
4. Plan data privacy considerations in Repository layer (important for finance)

**Ready to execute!** 🚀
