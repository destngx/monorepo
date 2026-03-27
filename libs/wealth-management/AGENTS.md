# Wealth Management Library — AGENTS.md

**Generated**: 2026-03-27  
**Commit**: 3d4e52e

---

## OVERVIEW

Reusable TypeScript library implementing wealth management domain logic. Follows **Feature-Sliced Design (FSD)** architecture. Exports features (accounts, investments, goals, chat), utilities (error handling, types), services (Google Sheets, cache, AI), and configuration.

**Used by**: `apps/wealth-management` (main consumer)  
**Stack**: TypeScript, React 18, Anthropic SDK, vnstock APIs, Google Sheets API

---

## STRUCTURE

```
libs/wealth-management/src/
├── features/             # Business features (FSD slices)
│   ├── accounts/         # Account management
│   ├── investments/      # Investment tracking
│   ├── goals/            # Financial goals
│   ├── loans/            # Loan tracking
│   ├── transactions/     # Transaction history
│   └── chat/             # AI chat interface
├── ai/                   # AI system + tools
│   ├── system-prompt.ts  # Anthropic system prompt
│   ├── tools/            # Tool definitions
│   └── index.ts          # AI client + streaming
├── services/             # Cross-feature services
│   ├── sheets/           # Google Sheets integration
│   ├── cache/            # Upstash Redis caching
│   └── services/         # Other utilities
├── types/                # Shared TypeScript types
├── utils/                # Utilities (errors, helpers)
├── config/               # App configuration
└── index.ts              # Public exports
```

---

## FEATURE-SLICED DESIGN (FSD)

Each feature folder contains:

```
features/{feature}/
├── ui/                   # React components
├── model/                # Business logic hooks
├── types.ts              # Feature-specific types
└── index.ts              # Public API
```

**Pattern**:

- **UI layer**: React components (no business logic)
- **Model layer**: `useXyz()` custom hooks (business logic, state)
- **Type layer**: TypeScript interfaces scoped to feature

**Import Rule**: Always use feature's `index.ts` (public API):

```typescript
// ✅ Correct
import { useAccount } from '@wealth-management/features/accounts';

// ❌ Wrong
import { useAccount } from '@wealth-management/features/accounts/model/use-account';
```

---

## AI SYSTEM

**System Prompt**: `ai/system-prompt.ts`

- Defines assistant behavior, constraints, and guardrails
- References available tools (fetch accounts, analyze portfolio, etc.)

**Available Tools**:

- `get_account_summary` — User account balances
- `get_portfolio_analysis` — Stock holdings analysis
- `fetch_stock_quote` — Real-time prices via vnstock
- `set_financial_goal` — Create goal
- `analyze_spending` — Transaction patterns

**Tool Invocation Pattern**:

```typescript
const response = await ai.createMessage({
  system: SYSTEM_PROMPT,
  tools: AVAILABLE_TOOLS,
  messages: userMessages,
});

// Process tool calls
for (const toolUse of response.content) {
  if (toolUse.type === 'tool_use') {
    const result = await executeToolById(toolUse.id);
  }
}
```

---

## GOOGLE SHEETS INTEGRATION

**Service**: `services/sheets/google-sheets.ts`

Stores user data:

- Accounts (checking, savings, credit cards)
- Transactions (income, expenses)
- Investment holdings
- Financial goals

**Usage**:

```typescript
const sheets = new GoogleSheetsService(credentialsFile);
const accounts = await sheets.getAccounts(userId);
const transactions = await sheets.getTransactions(userId, dateRange);
```

---

## REDIS CACHING LAYER

**Service**: `services/cache/`

Caches:

- Stock quotes (vnstock API results)
- Google Sheets data (reduces API calls)
- User account summaries

**TTL Defaults**:

- Stock quotes: 5 minutes
- Sheet data: 30 minutes
- Account summaries: 1 hour

---

## ERROR HANDLING

**Always use `AppError`** (`utils/errors/`):

```typescript
import { AppError } from '@wealth-management/utils/errors';

throw new AppError('Account not found', {
  code: 'ACCOUNT_NOT_FOUND',
  statusCode: 404,
  context: { userId, accountId },
});
```

Never swallow errors — log and propagate.

---

## TYPES

**Central types** in `types/`:

- `Account`, `Transaction`, `Investment`, `Goal`
- `Portfolio`, `PortfolioAnalysis`
- `ApiResponse<T>`

Import from `@wealth-management/types`:

```typescript
import { Account, Portfolio } from '@wealth-management/types';
```

---

## CONFIGURATION

**Config files** in `config/`:

- `transactions/` — Expense categories, rules
- `accounts/` — Account type definitions
- `constants.ts` — Global constants (API keys, URLs)

---

## PUBLIC EXPORTS

**Via `index.ts`**, users import:

```typescript
import {
  // Features
  useAccount,
  useInvestment,
  useGoal,
  // Services
  GoogleSheetsService,
  CacheService,
  // AI
  AIClient,
  SYSTEM_PROMPT,
  // Types
  Account,
  Portfolio,
  // Utils
  AppError,
} from '@wealth-management';
```

---

## ANTI-PATTERNS

**DO NOT:**

- Import implementation details directly (use `index.ts`)
- Create circular dependencies between features
- Hardcode API keys (use `config/constants.ts`)
- Mutate shared state (use React hooks for encapsulation)
- Mix UI logic with business logic

**ALWAYS:**

- Keep features independent (minimal cross-feature imports)
- Export public APIs via `index.ts`
- Use TypeScript strict mode
- Test business logic separately from UI

---

## COMMON TASKS

| Task        | Location                                        |
| ----------- | ----------------------------------------------- |
| Add feature | Create `features/{name}/` with FSD structure    |
| Add AI tool | `ai/tools/` + register in `ai/system-prompt.ts` |
| Add type    | `types/`                                        |
| Add service | `services/`                                     |
| Add utility | `utils/`                                        |
