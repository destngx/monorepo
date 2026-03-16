# Architecture Diagrams & Visual Maps

## 1. High-Level App Structure (Wealth-Management)

```
┌─────────────────────────────────────────────────────────────┐
│                    src/app/ (Routes Only)                   │
│  ┌────────┐ ┌────────┐ ┌─────────┐ ┌────────┐ ┌──────────┐ │
│  │  /     │ │accounts│ │budget   │ │ chat   │ │ invest   │ │
│  │page.tsx│ │page.tsx│ │page.tsx │ │page.tsx│ │page.tsx  │ │
│  └────────┘ └────────┘ └─────────┘ └────────┘ └──────────┘ │
│  └─────────────────────────────────────────────────────────┘
│                            ↓↓↓
│    All delegate to: features/[feature]/ui/page.tsx
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│         src/app/api/ (API Routes - Delegating)              │
│  ┌────────┐ ┌────────┐ ┌─────────┐ ┌────────┐  ┌────────┐  │
│  │accounts│ │ budget │ │transact │ │ goals  │  │ loans  │  │
│  │route.ts│ │route.ts│ │route.ts │ │route.ts│  │route.ts│  │
│  │export{ │ │export{ │ │export{  │ │export{ │  │export{ │  │
│  │  GET }  │ │  GET } │ │  GET }  │ │ GET }  │  │ GET }  │  │
│  └────────┘ └────────┘ └─────────┘ └────────┘  └────────┘  │
│  └─────────────────────────────────────────────────────────┘
│                            ↓↓↓
│    Delegates to: features/[feature]/api/route
│    (Except AI routes - implemented inline)
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│           src/features/ (Implementation Layer)              │
│  ┌────────────────┐  ┌────────────────┐  ┌───────────────┐ │
│  │  accounts/     │  │  budget/       │  │  chat/        │ │
│  │  ├── api/      │  │  ├── api/      │  │  ├── ui/      │ │
│  │  ├── model/    │  │  ├── model/    │  │  ├── model/   │ │
│  │  └── ui/       │  │  └── ui/       │  │  └── hooks/   │ │
│  │     └─page.tsx │  │     └─page.tsx │  │     └─page.tsx│ │
│  └────────────────┘  └────────────────┘  └───────────────┘ │
│                                                              │
│  ┌────────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │goals/      │  │investments│ │loans/    │  │settings/ │  │
│  │├── model/  │  │├── api/   │ │├── model │  │├── ui/   │  │
│  │└── ui/     │  │├── model/ │ │└── ui/   │  │└─ page.tx│  │
│  └────────────┘  │└── ui/    │ └──────────┘  └──────────┘  │
│                  └──────────┘                               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              src/lib/ (Shared Layer - NOT features)         │
│  ┌──────────┐ ┌────────┐ ┌────────┐ ┌─────┐ ┌──────────┐  │
│  │sheets/   │ │  ai/   │ │ utils/ │ │types│ │constants│  │
│  │├─accounts│ │├─provs │ │├─error │ │├─acc│ │├─nav    │  │
│  │├─budget  │ │├─prompt│ │├─curr  │ │├─bud│ │├─cat    │  │
│  │├─txn     │ │├─tools │ │├─date  │ │├─txn│ │└─tags   │  │
│  │└─loans   │ │└─core/ │ │└─valid │ │└─etc│ │         │  │
│  └──────────┘ └────────┘ └────────┘ └─────┘ └──────────┘  │
│                                                              │
│          ↑↑↑  USED BY: features/, app/api/                  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│         src/components/ (Shared UI Components)              │
│  ┌──────────┐ ┌────────┐ ┌──────────┐ ┌────────────────┐   │
│  │layout/   │ │  ui/   │ │dashboard │ │[feature]/      │   │
│  │├─sidebar │ │├─button│ │├─net-wth │ │├─accounts/     │   │
│  │├─header  │ │├─input │ │├─snapshot│ │├─budget/       │   │
│  │└─wrapper │ │└─table │ │└─charts  │ │├─chat/         │   │
│  │          │ │        │ │          │ │└─transactions/ │   │
│  └──────────┘ └────────┘ └──────────┘ └────────────────┘   │
│                                                              │
│    Used by: features/*/ui/, app/layout.tsx, app/page.tsx   │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Request Flow Diagrams

### Request Flow 1: User Navigates to /budget

```
Browser                    Next.js Router           Feature Code
  │                            │                         │
  ├─ Navigate /budget          │                         │
  │                 ┌──────────┼─────────────────────┐   │
  │                 │ Load Route Handler             │   │
  │                 └──────────┼─────────────────────┘   │
  │                            │                         │
  │                 ┌──────────┼─────────────────────┐   │
  │                 │ app/budget/page.tsx            │   │
  │                 │   imports BudgetPage from      │   │
  │                 │   @/features/budget/ui        │   │
  │                 └──────────┼─────────────────────┘   │
  │                            │                         │
  │                            │     ┌──────────────────┤
  │                            │     │ features/budget  │
  │                            │     │ /ui/page.tsx     │
  │                            │     │ (render)         │
  │                            │     └──────────────────┤
  │                            │                         │
  │ ◄────── Render ────────────┤                         │
  │         HTML                                         │
  │                                                      │
  │  useEffect() runs                                    │
  │  fetch('/api/budget') ──┐                            │
  │                         │                           │
  │                    ┌────┴──────────────────────┐    │
  │                    │ app/api/budget/route.ts  │    │
  │                    │  export { GET } from     │    │
  │                    │  features/budget/api     │    │
  │                    └────┬──────────────────────┘    │
  │                         │                           │
  │                         │  ┌──────────────────────┤
  │                         │  │ features/budget/     │
  │                         │  │ api/route.ts         │
  │                         │  │ ┌──────────────────┐ │
  │                         │  │ │ getBudget()      │ │
  │                         │  │ │ getCategories()  │ │
  │                         │  │ │ return JSON      │ │
  │                         │  │ └──────────────────┘ │
  │                         │  └──────────────────────┤
  │                         │                         │
  │◄─── JSON Response ──────┤                         │
  │                         │                         │
  │  setState(budgetData)                             │
  │  Re-render with data                              │
  │                                                    │
  └────────────────────────────────────────────────────┘
```

### Request Flow 2: AI Route (Budget Advisor)

```
Browser                  App Route             Feature/AI Code
  │                          │                        │
  │  User clicks "Get AI"     │                        │
  │  fetch('/api/ai/budget-   │                        │
  │         advisor',         │                        │
  │         { POST, ...data})│                        │
  │         │                 │                        │
  │         │  ┌──────────────┼──────────────────┐   │
  │         │  │ app/api/ai/  │                  │   │
  │         │  │ budget-      │                  │   │
  │         │  │ advisor/     │                  │   │
  │         │  │ route.ts     │                  │   │
  │         │  │ (full impl)  │                  │   │
  │         │  └──────────────┼──────────────────┘   │
  │         │                 │                        │
  │         │                 │     Parse body        │
  │         │                 │     Get model         │
  │         │                 │     Build prompt      │
  │         │                 │     generateText()    │
  │         │                 │                       │
  │         │                 │  ┌─────────────────┐ │
  │         │                 │  │ lib/ai/         │ │
  │         │                 │  │ providers.ts    │ │
  │         │                 │  │ system-prompt   │ │
  │         │                 │  │ .ts             │ │
  │         │                 │  └─────────────────┘ │
  │         │                 │                       │
  │         │                 │     Call LLM         │
  │         │                 │     (OpenAI)         │
  │         │                 │                       │
  │◄─── Streaming Response ───┤                       │
  │     (text chunks)                                │
  │                                                   │
  │  Display in UI                                    │
  │                                                   │
  └───────────────────────────────────────────────────┘
```

---

## 3. Data Flow Through Layers

```
┌───────────────────────────────────────────────────────────┐
│ Tier 1: Browser (Client - React Components)              │
│                                                           │
│  features/[feature]/ui/page.tsx                         │
│  ├─ useState for local state                            │
│  ├─ useEffect to fetch data                             │
│  ├─ render components with state                        │
│  └─ handle user interactions                            │
└────────────────────────────────────────────────────────────┘
                          ↓ fetch() ↑ JSON
┌────────────────────────────────────────────────────────────┐
│ Tier 2: Next.js Route Handler (API Route)                │
│                                                            │
│  app/api/[feature]/route.ts                              │
│  ├─ parse request                                        │
│  ├─ validate input                                       │
│  ├─ call feature handler or lib function                │
│  └─ return NextResponse.json()                           │
│                                                            │
│  OR: app/api/ai/[action]/route.ts                        │
│  ├─ get request body                                     │
│  ├─ getLanguageModel()                                   │
│  ├─ buildSystemPrompt()                                 │
│  ├─ generateText() or streamText()                       │
│  └─ return response                                      │
└────────────────────────────────────────────────────────────┘
                          ↓ ↑
┌────────────────────────────────────────────────────────────┐
│ Tier 3: Feature Implementation (features/*/api/)          │
│                                                            │
│  features/[feature]/api/route.ts                          │
│  ├─ import { get[Feature] } from '@/lib/sheets/...'      │
│  ├─ await get[Feature]()                                 │
│  ├─ enrich data if needed                                │
│  └─ return data                                          │
└────────────────────────────────────────────────────────────┘
                          ↓ ↑
┌────────────────────────────────────────────────────────────┐
│ Tier 4: Data Access Layer (lib/sheets/)                   │
│                                                            │
│  lib/sheets/[feature].ts                                  │
│  ├─ import { getClient } from './client'                │
│  ├─ const client = await getClient()                     │
│  ├─ const rows = await client.spreadsheet.query()       │
│  ├─ map and enrich rows                                  │
│  └─ return typed data                                    │
└────────────────────────────────────────────────────────────┘
                          ↓ ↑
┌────────────────────────────────────────────────────────────┐
│ Tier 5: External Data Sources                             │
│                                                            │
│  Google Sheets API                                        │
│  Caching layer                                            │
│  External APIs (exchange rates, market data, etc.)        │
└────────────────────────────────────────────────────────────┘
```

---

## 4. Delegation Pattern Diagram

```
┌─ Page Route ─────────────────────────────────────────┐
│                                                       │
│  src/app/[feature]/page.tsx (3 lines)               │
│                                                       │
│  import { [Feature]Page } from                      │
│    '@/features/[feature]/ui'                        │
│  export default [Feature]Page                       │
│                                                       │
│  ✓ Route definition                                  │
│  ✓ Minimal code                                      │
│  ✓ Easy to locate implementation                    │
│                                                       │
└─────────────────┬─────────────────────────────────────┘
                  │
                  ↓ imports
┌─ Feature UI ─────────────────────────────────────────┐
│                                                       │
│  features/[feature]/ui/page.tsx (50+ lines)         │
│                                                       │
│  'use client'                                        │
│                                                       │
│  const [state, setState] = useState()               │
│  useEffect(() => {                                  │
│    fetch('/api/[feature]')                         │
│      .then(r => r.json())                          │
│      .then(setState)                               │
│  }, [])                                             │
│                                                       │
│  return <div>...</div>                             │
│                                                       │
│  ✓ Actual component                                 │
│  ✓ Data fetching                                    │
│  ✓ Event handling                                   │
│  ✓ Re-rendering logic                              │
│                                                       │
└─────────────────┬─────────────────────────────────────┘
                  │
                  ↓ fetch()
┌─ API Handler ────────────────────────────────────────┐
│                                                       │
│  src/app/api/[feature]/route.ts (1 line)           │
│                                                       │
│  export { GET } from                                │
│    '@/features/[feature]/api/route'                │
│                                                       │
│  ✓ Route definition                                  │
│  ✓ Minimal code                                      │
│                                                       │
└─────────────────┬─────────────────────────────────────┘
                  │
                  ↓ imports
┌─ Feature API ────────────────────────────────────────┐
│                                                       │
│  features/[feature]/api/route.ts (15 lines)         │
│                                                       │
│  export async function GET() {                      │
│    try {                                            │
│      const data = await get[Feature]()             │
│      return NextResponse.json(data)                │
│    } catch (error) {                               │
│      return handleApiError(error, '[Feature]')    │
│    }                                               │
│  }                                                  │
│                                                       │
│  ✓ Actual handler                                   │
│  ✓ Error handling                                   │
│  ✓ Data processing                                  │
│                                                       │
└─────────────────┬─────────────────────────────────────┘
                  │
                  ↓ calls
┌─ Data Access ────────────────────────────────────────┐
│                                                       │
│  lib/sheets/[feature].ts                            │
│                                                       │
│  export async function get[Feature]() {            │
│    const client = await getClient()                │
│    const rows = await client.query(...)           │
│    return rows.map(mapTo[Feature])               │
│  }                                                  │
│                                                       │
│  ✓ Database/API calls                              │
│  ✓ Caching                                         │
│  ✓ Data transformation                             │
│                                                       │
└──────────────────────────────────────────────────────┘
```

---

## 5. Feature Dependencies (Wealth-Management)

```
                   ┌─────────────────┐
                   │   src/app/      │
                   │ (routes only)   │
                   └────────┬────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
    ┌────────────┐  ┌─────────────┐  ┌──────────────┐
    │ features/  │  │ lib/        │  │ components/  │
    │ [feature]/ │  │ (shared)    │  │ (shared)     │
    │ {api,ui}   │  │ ├─sheets/   │  │ ├─layout/    │
    │            │  │ ├─ai/       │  │ ├─ui/        │
    │ ├─accounts │  │ ├─utils/    │  │ ├─dashboard/ │
    │ ├─budget   │  │ ├─types/    │  │ ├─[feature]/ │
    │ ├─chat     │  │ └─constants │  │ └─...        │
    │ ├─goals    │  │             │  │              │
    │ ├─invest   │  │             │  │              │
    │ ├─loans    │  │             │  │              │
    │ ├─trans    │  │             │  │              │
    │ └─settings │  │             │  │              │
    └───┬────────┘  └─────────────┘  └──────────────┘
        │                 │                  │
        └─────────────────┼──────────────────┘
                          │
                   Each features/:
                   • imports from lib/
                   • imports from components/
                   • imports from other features/**/model/
                   • exports via barrels (index.ts)
```

---

## 6. Component Hierarchy (Root Layout)

```
<html lang="en">
  <body>
    <ThemeProvider>
      <MaskProvider>
        <AIContextProvider>
          <SidebarProvider>
            <div className="flex min-h-screen">

              ┌─────────────────────┐
              │ <Sidebar />         │  (components/layout/sidebar)
              │ ├─ Navigation links │
              │ └─ State toggle     │
              └─────────────────────┘

              ┌─────────────────────────────────────┐
              │ <LayoutWrapper>                     │
              │ ┌───────────────────────────────┐  │
              │ │ <Header />                    │  │ (components/layout/header)
              │ │ ├─ Top bar                    │  │
              │ │ └─ User menu                  │  │
              │ └───────────────────────────────┘  │
              │                                     │
              │ ┌───────────────────────────────┐  │
              │ │ <main>                        │  │
              │ │ {children}  ← Page content    │  │
              │ │ (from features/*/ui)          │  │
              │ └───────────────────────────────┘  │
              └─────────────────────────────────────┘

              ┌─────────────────────┐
              │ <AIChatWidget />    │  (components/chat/ai-chat-widget)
              │ └─ Floating chat    │
              └─────────────────────┘

            </div>
          </SidebarProvider>
        </AIContextProvider>
      </MaskProvider>
    </ThemeProvider>
  </body>
</html>
```

---

## 7. Import Path Quick Reference

```
From Route (src/app/*/page.tsx):
  import { [Feature]Page } from '@/features/[feature]/ui'
  import { Component } from '@/components/[category]/[component]'
  import { Provider } from '@/components/[provider]'

From Feature UI (features/*/ui/page.tsx):
  import { Type } from '../model/types'
  import { Component } from './[component]'
  import { Component } from '../ui/[component]'
  import { Type } from '@/features/[other]/model/types'
  import { Component } from '@/components/ui/[component]'
  import { get[Feature] } from '@/lib/sheets/[feature]'
  import { util } from '@/lib/utils/[util]'
  import { CONST } from '@/lib/constants/[const]'

From Feature API (features/*/api/route.ts):
  import { NextResponse } from 'next/server'
  import { get[Feature] } from '@/lib/sheets/[feature]'
  import { handleApiError } from '@/lib/utils/api-error-handler'
  import { Type } from '../model/types'

From AI Route (src/app/api/ai/*/route.ts):
  import { NextResponse } from 'next/server'
  import { getLanguageModel } from '@/lib/ai/providers'
  import { buildSystemPrompt } from '@/lib/ai/system-prompt'
  import { generateText, streamText } from 'ai'
  import { financialTools } from '@/lib/ai/tools'
```

---

## Summary

- **Three-tier separation**: app/ (routes) → features/ (impl) → lib/ (shared)
- **Delegation pattern**: Minimize code in routes, maximize in features
- **Clear dependencies**: All cross-feature imports are from `model/types/`
- **Consistent structure**: Every feature follows same layout
- **Shared layer**: lib/ and components/ are reusable across apps
