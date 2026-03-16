# Feature Structure Template & Examples

## Standard Feature Layout

Every feature in wealth-management follows this consistent structure:

```
features/[feature-name]/
├── api/                      # API handlers (optional if no backend needed)
│   ├── route.ts             # GET/POST/etc handlers
│   └── [nested]/route.ts    # Nested endpoints (e.g., /api/feature/sub)
├── model/                    # Domain logic & types (optional)
│   ├── types.ts             # Feature-specific TypeScript types
│   ├── utils.ts             # Business logic & calculations
│   └── index.ts             # Barrel export
├── ui/                       # React components (required)
│   ├── page.tsx             # Main page component
│   ├── [component].tsx      # Feature components
│   ├── [view].tsx           # Different views/sections
│   └── index.ts             # Barrel export (for delegation)
└── hooks/                    # Feature hooks (optional)
    └── use-[hook].ts
```

---

## Real Examples from Wealth-Management

### Example 1: Budget Feature

```
features/budget/
├── api/
│   └── route.ts
│       export async function GET() {
│         const budget = await getBudget()
│         const categories = await getCategories()
│         return NextResponse.json(enrichedBudget)
│       }
├── model/
│   └── types.ts
│       export interface BudgetItem {
│         category: string
│         monthlyLimit: number
│         monthlySpent: number
│         categoryType?: CategoryType
│       }
├── ui/
│   ├── page.tsx           # Main budget page (client)
│   ├── budget-overview-view.tsx
│   ├── category-detail-view.tsx
│   ├── ai-budget-advisor-view.tsx
│   └── index.ts           # export { default as BudgetPage }
└── hooks/
    ├── use-ai-settings.ts
    └── use-budget-filters.ts
```

**How it's used**:

```typescript
// src/app/budget/page.tsx
import { BudgetPage } from '@/features/budget/ui';
export default BudgetPage;

// Browser loads: features/budget/ui/page.tsx
// Which renders: <BudgetOverviewView> + <CategoryDetailView> + <AIBudgetAdvisorView>
```

---

### Example 2: Accounts Feature (With Nesting)

```
features/accounts/
├── api/
│   └── route.ts
│       export async function GET() {
│         const accounts = await getAccounts(forceFresh)
│         return NextResponse.json(accounts)
│       }
├── model/
│   ├── types.ts
│   │   export interface Account {
│   │     name: string
│   │     type: 'checking' | 'savings' | 'investment' | 'credit'
│   │     balance: number
│   │     currency: string
│   │   }
│   └── index.ts
├── ui/
│   ├── page.tsx                    # /accounts main page
│   ├── account-review-ai.tsx       # Account AI analysis
│   ├── account-trend-sparkline.tsx # Trend chart
│   ├── credit-card-summary-ai.tsx  # Credit card view
│   ├── efficiency-chart.tsx
│   └── index.ts
│       export { default as AccountsPage }
│       export { CreditCardSummaryAI }
│       export { ...other components }
└── (no nested features - goals/loans are separate features)
```

**How goals are nested**:

```
features/goals/              # SEPARATE feature (not features/accounts/goals)
├── ui/
│   ├── page.tsx            # /accounts/goals main page
│   ├── goal-form.tsx
│   ├── goal-card.tsx
│   ├── [id]/
│   │   └── page.tsx        # /accounts/goals/[id] detail
│   └── index.ts
│       export { default as GoalsPage }
```

---

### Example 3: Transactions Feature (With Model Logic)

```
features/transactions/
├── api/
│   └── route.ts
│       export async function GET() {
│         const transactions = await getTransactions()
│         return NextResponse.json(transactions)
│       }
├── model/
│   ├── types.ts
│   │   export interface Transaction {
│   │     id: string
│   │     date: string
│   │     payee: string
│   │     category: string
│   │     payment?: number   // outgoing
│   │     deposit?: number   // incoming
│   │     balance: number
│   │     tags: string[]
│   │   }
│   └── utils.ts
│       export function categorizeTransaction(tx: Transaction): string
│       export function calculateMonthlySpending(txs: Transaction[]): Record<string, number>
│       export function filterByCategory(txs: Transaction[], cat: string): Transaction[]
├── ui/
│   ├── page.tsx                 # /transactions main page
│   ├── transaction-table.tsx    # Data table
│   ├── filter-panel.tsx         # Filters & search
│   ├── category-breakdown.tsx   # Category pie chart
│   └── index.ts
│       export { default as TransactionsPage }
└── hooks/
    ├── use-transaction-filters.ts
    └── use-category-breakdown.ts
```

---

### Example 4: Chat Feature (Complex, Stateful)

```
features/chat/
├── (NO api/ subdirectory - calls /api/chat directly)
├── model/
│   ├── types.ts
│   │   export interface Message {
│   │     id: string
│   │     role: 'user' | 'assistant'
│   │     content: string
│   │     toolInvocations?: ToolCall[]
│   │   }
│   │   export interface ChatContext {
│   │     messages: Message[]
│   │     isLoading: boolean
│   │     error?: string
│   │   }
│   └── index.ts
├── ui/
│   ├── page.tsx                  # /chat main page
│   ├── ChatContainer.tsx         # Main chat wrapper
│   ├── message.tsx               # Message rendering
│   ├── message-input.tsx         # User input
│   ├── tool-call-display.tsx     # Tool results
│   └── index.ts
│       export { ChatContainer }
└── hooks/
    ├── use-chat.ts               # Chat state management
    ├── use-message-stream.ts     # Handle streaming
    └── use-tool-calls.ts         # Process tool results
```

---

## Delegation Pattern Details

### Pattern 1: Simple Page Delegation

```typescript
// src/app/[feature]/page.tsx (2-3 lines)
/**
 * Delegating page route
 * Maps app/[feature] to features/[feature]/ui/page
 */
import { [Feature]Page } from '@/features/[feature]/ui'

export default [Feature]Page
```

**Pros**:

- Clear separation of routing (app/) and implementation (features/)
- Feature can be moved/refactored without changing route
- Keeps app/ lightweight and focused

**Used for**: Most pages (accounts, budget, transactions, etc.)

---

### Pattern 2: Simple API Delegation

```typescript
// src/app/api/[feature]/route.ts (1 line)
export { GET } from '@/features/[feature]/api/route';

// src/features/[feature]/api/route.ts (10-20 lines)
export async function GET(request: Request) {
  try {
    const data = await get[Feature]();
    return NextResponse.json(data);
  } catch (error) {
    return handleApiError(error, '[Feature]');
  }
}
```

**Pros**:

- Route handler delegates to feature implementation
- Error handling is standardized
- Feature can change implementation without route changes

**Used for**: Simple data fetch APIs (accounts, budget, transactions)

---

### Pattern 3: Inline Page Component

```typescript
// src/app/[feature]/page.tsx (direct implementation)
'use client'

import { Component } from '@/features/[feature]/ui'

export default function Page() {
  return (
    <div>
      <Component />
    </div>
  )
}
```

**When used**: Chat page (small wrapper around feature)

**Pros**: Direct control of page structure

**Cons**: Less clean separation

---

### Pattern 4: Inline API Handler

```typescript
// src/app/api/[path]/route.ts (full implementation)
export async function POST(req: Request) {
  const { data } = await req.json()

  const model = getLanguageModel(modelId)
  const systemPrompt = await buildSystemPrompt(instruction)

  const { text } = await generateText({ model, system: systemPrompt, ... })

  return NextResponse.json({ result: text })
}
```

**When used**: Complex logic (AI routes, chat, multi-step processes)

**Pros**: All logic in one place for coherent workflows

**Cons**: Harder to test; can get complex

---

## Import Chain Examples

### Building Accounts Feature

**File: features/accounts/ui/page.tsx**

```typescript
import { useState, useEffect } from 'react'

// From same feature
import { Account } from '../model/types'
import { AccountCard } from './account-card'

// From other features
import { Transaction } from '@/features/transactions/model/types'

// From shared lib
import { getAccounts } from '@/lib/sheets/accounts'
import { getTransactions } from '@/lib/sheets/transactions'
import { formatCurrency } from '@/lib/utils/currency'

// From components
import { Card } from '@/components/ui/card'
import { Loading } from '@/components/ui/loading'

// External
import { format } from 'date-fns'
import { TrendingUp } from 'lucide-react'

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      fetch('/api/accounts').then(r => r.json()),
      fetch('/api/transactions').then(r => r.json())
    ]).then(([accts, txs]) => {
      setAccounts(accts)
      setTransactions(txs)
      setLoading(false)
    })
  }, [])

  if (loading) return <Loading />

  return (
    <div className="space-y-6">
      <div className="grid gap-4">
        {accounts.map(account => (
          <AccountCard key={account.name} account={account} />
        ))}
      </div>
    </div>
  )
}
```

---

## Creating a New Feature

### Step 1: Create Feature Directory

```bash
mkdir -p src/features/new-feature/{api,model,ui}
```

### Step 2: Define Types

**File: features/new-feature/model/types.ts**

```typescript
export interface NewFeatureItem {
  id: string;
  name: string;
  // ... other properties
}
```

### Step 3: Create API Handler

**File: features/new-feature/api/route.ts**

```typescript
import { NextResponse } from 'next/server';
import { getNewFeatureData } from '@/lib/sheets/new-feature';
import { handleApiError } from '@/lib/utils/api-error-handler';

export async function GET(request: Request) {
  try {
    const data = await getNewFeatureData();
    return NextResponse.json(data);
  } catch (error) {
    return handleApiError(error, 'NewFeature');
  }
}
```

### Step 4: Create UI Page

**File: features/new-feature/ui/page.tsx**

```typescript
'use client'

import { useState, useEffect } from 'react'
import { NewFeatureItem } from '../model/types'
import { Loading } from '@/components/ui/loading'

export default function NewFeaturePage() {
  const [items, setItems] = useState<NewFeatureItem[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/new-feature')
      .then(r => r.json())
      .then(data => {
        setItems(data)
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [])

  if (loading) return <Loading />

  return (
    <div>
      {/* Render items */}
    </div>
  )
}
```

### Step 5: Create Barrel Export

**File: features/new-feature/ui/index.ts**

```typescript
export { default as NewFeaturePage } from './page';
// Export other components as needed
```

### Step 6: Create Route Delegation

**File: src/app/new-feature/page.tsx**

```typescript
/**
 * Delegating page route
 * Maps app/new-feature to features/new-feature/ui/page
 */
import { NewFeaturePage } from '@/features/new-feature/ui';

export default NewFeaturePage;
```

### Step 7: Create API Delegation

**File: src/app/api/new-feature/route.ts**

```typescript
/**
 * Delegating API route
 * Maps app/api/new-feature to features/new-feature/api/route
 */
export { GET } from '@/features/new-feature/api/route';
```

### Step 8: Add Data Access Function

**File: lib/sheets/new-feature.ts**

```typescript
import { getClient } from './client';

export async function getNewFeatureData(): Promise<NewFeatureItem[]> {
  const client = await getClient();
  const rows = await client.spreadsheet.find('NewFeature');
  return rows.map(mapToNewFeatureItem);
}

function mapToNewFeatureItem(row: any): NewFeatureItem {
  // Transform raw sheet data to domain type
}
```

---

## Key Takeaways

1. **Every feature follows the same structure** (`api/`, `model/`, `ui/`)
2. **Routes delegate to features** - app/ is routing only
3. **Features are self-contained** - all logic in one place
4. **Cross-feature imports are allowed** for types and shared components
5. **lib/ is the shared layer** - data access, utilities, types
6. **Each feature owns its API** - one endpoint per feature
7. **Barrels exports simplify imports** - `import { X } from '@/features/y/ui'`

---
