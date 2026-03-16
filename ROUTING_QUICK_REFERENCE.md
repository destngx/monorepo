# Wealth-Management Routing & Dependency Quick Reference

## Route Delegation Flow

### Page Routes (User-Facing)

```
src/app/[feature]/page.tsx (2-3 lines)
    └─── exports { [Feature]Page } from '@/features/[feature]/ui'
         └─── features/[feature]/ui/page.tsx (implementation)
              ├─── uses: features/[feature]/ui/[components]
              ├─── uses: lib/sheets/*  (data fetching)
              ├─── uses: lib/ai/*      (if AI involved)
              └─── uses: components/*  (shared UI)
```

### API Routes (Backend)

```
Two patterns:

1. DELEGATION PATTERN (Simple data fetch):
   src/app/api/[feature]/route.ts (1 export)
       └─── export { GET } from '@/features/[feature]/api/route'
            └─── features/[feature]/api/route.ts
                 ├─── import { get[Feature] } from '@/lib/sheets/[feature]'
                 ├─── import { handleApiError } from '@/lib/utils/...'
                 └─── return NextResponse.json(data)

2. INLINE PATTERN (Complex logic):
   src/app/api/ai/[action]/route.ts (full implementation)
       ├─── import { getLanguageModel } from '@/lib/ai/providers'
       ├─── import { buildSystemPrompt } from '@/lib/ai/system-prompt'
       ├─── import { generateText } from 'ai'
       └─── export async function POST(req) { ... }
```

---

## Module-Level Dependencies

### Data Access Layer (`lib/sheets/`)

All data fetching functions follow this pattern:

```typescript
export async function get[Feature](forceFresh?: boolean): Promise<[Feature][]>
```

**Imports**:

- `@/lib/sheets/client` (Google Sheets client)
- `@/lib/utils/api-error-handler` (error handling)
- Feature-specific types from `@/lib/types/*`

**Used by**:

- Feature API routes (`features/[feature]/api/route.ts`)
- Server pages (`features/[feature]/ui/page.tsx`)
- Dashboard root page (`src/app/page.tsx`)

---

### AI Layer (`lib/ai/`)

**Providers** (`providers.ts`):

```typescript
export function getLanguageModel(modelId: string): LanguageModel;
```

Selects between: GPT-4o, GitHub GPT-4o, GPT-4o-mini, Claude, Gemini

**Used by**: All AI routes (`/api/ai/*`, `/api/chat/*`)

**System Prompt** (`system-prompt.ts`):

```typescript
export async function buildSystemPrompt(taskInstruction: string): Promise<string>;
```

Builds base prompt with:

- Model-specific instructions
- Tool descriptions
- Format requirements

**Used by**: All AI routes

**Tools** (`tools/index.ts`):

- Financial calculations tools
- Web search tools
- Data aggregation tools

**Used by**: `/api/chat` (streaming chat interface)

---

### Shared UI Components (`components/`)

| Component        | Location             | Used By        | Dependencies               |
| ---------------- | -------------------- | -------------- | -------------------------- |
| Sidebar          | `layout/sidebar.tsx` | Root layout    | Navigation config          |
| Header           | `layout/header.tsx`  | Root layout    | Navigation state           |
| Dashboard cards  | `dashboard/*.tsx`    | Dashboard page | Chart libraries            |
| Loading skeleton | `ui/loading.tsx`     | Feature pages  | shadcn UI                  |
| Form components  | `ui/*.tsx`           | All forms      | shadcn UI, React Hook Form |

---

## Critical Import Paths

### From `src/app/` (route definitions):

```typescript
// Delegate to feature
import { [Feature]Page } from '@/features/[feature]/ui'
import { [Feature]Page } from '@/features/[feature]/ui'

// Use shared layout components
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'

// Use providers
import { ThemeProvider } from '@/components/theme-provider'
import { SidebarProvider } from '@/components/layout/sidebar-provider'
import { AIContextProvider } from '@/components/chat/ai-context-provider'
```

### From `features/[feature]/api/route.ts`:

```typescript
// Data fetching
import { get[Feature] } from '@/lib/sheets/[feature]'

// Error handling
import { handleApiError } from '@/lib/utils/api-error-handler'

// Types
import { [Feature] } from '@/lib/types/[feature]'

// Next.js
import { NextResponse } from 'next/server'
```

### From `features/[feature]/ui/page.tsx`:

```typescript
// Other features
import { [OtherFeature]Type } from '@/features/[otherFeature]/model/types'

// Data access
import { get[Feature] } from '@/lib/sheets/[feature]'

// Shared UI
import { [Component] } from '@/components/ui/[component]'
import { [Feature]Component } from '@/components/[feature]/[component]'

// Feature components
import { [Component] } from '../ui/[component]'
import { [Type] } from '../model/types'

// Utilities
import { cn } from '@/lib/utils'
import { format } from 'date-fns'
```

### From AI routes (`/api/ai/*/route.ts`):

```typescript
// AI orchestration
import { getLanguageModel } from '@/lib/ai/providers';
import { buildSystemPrompt } from '@/lib/ai/system-prompt';

// AI SDK
import { generateText, streamText } from 'ai';

// Data (fetch in route handler)
const data = await Promise.all([getAccounts(), getTransactions(), getBudget(), getLoans()]);

// Types
import { NextResponse } from 'next/server';
```

---

## Data Flow Examples

### Example 1: Budget Page Load

```
1. User navigates to /budget
   ↓
2. Browser loads: src/app/budget/page.tsx
   → exports BudgetPage from @/features/budget/ui
   ↓
3. React renders: features/budget/ui/page.tsx (client component)
   → useState hooks for budget, transactions, view state
   ↓
4. On mount, useEffect runs:
   fetch('/api/budget').then(r => r.json())  // calls handler below
   fetch('/api/transactions').then(r => r.json())
   ↓
5. Browser requests: /api/budget
   → src/app/api/budget/route.ts
   → export { GET } from '@/features/budget/api/route'
   ↓
6. Handler: features/budget/api/route.ts
   → const budget = await getBudget()          // lib/sheets/budget
   → const categories = await getCategories()  // lib/sheets/categories
   → return NextResponse.json(enrichedBudget)
   ↓
7. Sheet fetching: lib/sheets/budget.ts
   → Uses Google Sheets client (lib/sheets/client)
   → Returns cached or fresh budget items
   ↓
8. Response back to browser
   → Budget page renders with fetched data
   → User can switch views (overview, detail, advisor)
```

### Example 2: AI Budget Advisor Trigger

```
1. User clicks "Get AI Advisor" on budget page
   ↓
2. Page component handles click:
   fetch('/api/ai/budget-advisor', {
     method: 'POST',
     body: { budget, transactions, date, modelId }
   })
   ↓
3. Handler: src/app/api/ai/budget-advisor/route.ts
   → Parse request body
   → const model = getLanguageModel(modelId)  // lib/ai/providers
   ↓
4. Build system prompt:
   → const systemPrompt = await buildSystemPrompt(taskInstruction)
   ↓
5. Call LLM:
   → const { text } = await generateText({
       model,
       system: systemPrompt,
       prompt: `Review the budget for ${view} (${date}).`
     })
   ↓
6. Return response:
   → NextResponse.json({ review: text })
   ↓
7. Page renders AI response
```

### Example 3: Chat with Tools

```
1. User sends message in chat widget
   ↓
2. Chat component (features/chat/ui) sends:
   fetch('/api/chat', {
     method: 'POST',
     body: { messages: [...], modelId, context }
   })
   ↓
3. Handler: src/app/api/chat/route.ts (INLINE, not delegated)
   ↓
4. Get model:
   → const model = getLanguageModel(selectedModel)  // lib/ai/providers
   ↓
5. Build system prompt:
   → const systemPrompt = await buildSystemPrompt(taskInstruction)
   ↓
6. Import tools:
   → import { financialTools } from '@/lib/ai/tools'
   ↓
7. Stream response with tools:
   → const result = await streamText({
       model,
       system: systemPrompt,
       messages,
       tools: financialTools,
       maxSteps: 10
     })
   → return result.toUIMessageStreamResponse()
   ↓
8. Browser receives streaming response
   → Chat widget renders messages + tool calls
   → Tools execute (web search, calculations, etc.)
```

---

## File Change Impact Matrix

| Change Location                   | Impact Scope      | Requires Route Update      | Requires Feature Update       |
| --------------------------------- | ----------------- | -------------------------- | ----------------------------- |
| `lib/sheets/accounts.ts`          | accounts feature  | ❌ NO                      | ❌ NO (uses same import path) |
| `lib/ai/providers.ts`             | All AI routes     | ❌ NO                      | ❌ NO                         |
| `features/budget/api/route.ts`    | Budget API        | ❌ NO                      | ❌ NO                         |
| `features/budget/ui/page.tsx`     | Budget page       | ❌ NO                      | ❌ NO                         |
| `src/app/budget/page.tsx` (route) | Budget route path | ✅ YES - path change       | ✅ YES - delegation change    |
| `src/app/layout.tsx`              | Root layout       | ✅ YES - affects all pages | ❌ NO                         |
| `components/layout/sidebar.tsx`   | Navigation UI     | ❌ NO                      | ❌ NO                         |

---

## Refactoring Safety Guidelines

### ✅ SAFE to move to shared lib

- `lib/ai/core/*` (orchestration patterns)
- `lib/utils/*` (utilities)
- `components/layout/*` (layout components)
- `components/ui/*` (base UI components)

**Why safe**: Wealth-management imports these via absolute paths `@/lib/...`. If moved to another workspace lib, just update import paths.

### ⚠️ CAREFUL refactoring

- `features/*/api/route.ts` (some may duplicate across apps)
- `lib/sheets/*` (app-specific data integration)
- Feature-specific AI routes

**Why careful**: Requires analyzing if other apps need same logic.

### ❌ DO NOT refactor

- `src/app/*/page.tsx` (route definitions - path must stay)
- `src/app/api/*/route.ts` (API paths - cannot change)
- Feature UI pages (domain-specific to wealth-management)

**Why not**: Changes break routing.

---

## Testing Route Changes

Before committing refactoring:

```bash
# Check that all routes still resolve
npm run build

# Check that API handlers are still reachable
npm run test -- integration/routes.spec.ts

# Manual verification (if not automated)
# 1. Navigate to each page: /, /accounts, /budget, /chat, etc.
# 2. Click features that call APIs
# 3. Check network tab: all /api/* requests succeed
```

---
