# Wealth Management App — AGENTS.md

**Generated**: 2026-03-27  
**Commit**: 3d4e52e

---

## OVERVIEW

Next.js full-stack dashboard for personal wealth management. Integrates AI chat, stock data via vnstock, Google Sheets for accounts/transactions, and Upstash Redis caching. Real-time market data, portfolio analysis, investment recommendations, and goal tracking.

**Stack**: Next.js 14, React 18, Anthropic AI, vnstock APIs, Google Sheets API, Upstash Redis

---

## STRUCTURE

```
apps/wealth-management/
├── src/
│   ├── app/              # Next.js 14 app router
│   ├── components/       # Local UI components
│   │   ├── chat/         # AI chat interface
│   │   └── ui/           # Reusable UI elements
│   └── hooks/            # Local custom hooks
├── public/               # Static assets
└── scripts/              # Build/setup scripts
```

**Most logic lives in**: `libs/wealth-management/src/` (see wealth-management library AGENTS.md)

---

## WHERE TO LOOK

| Task                                    | Location                                      |
| --------------------------------------- | --------------------------------------------- |
| Add new feature (accounts, goals, etc.) | `libs/wealth-management/src/features/`        |
| AI chat system prompt & tools           | `libs/wealth-management/src/ai/`              |
| Google Sheets integration               | `libs/wealth-management/src/services/sheets/` |
| Market data caching                     | `libs/wealth-management/src/services/cache/`  |
| API routes                              | `src/app/api/`                                |
| Page layouts                            | `src/app/`                                    |

---

## KEY PATTERNS

### API Routes (Next.js)

```typescript
// src/app/api/[feature]/route.ts
export async function POST(req: Request) {
  try {
    // Use server-side services from libs/wealth-management
    const result = await featureService.process(data);
    return Response.json(result);
  } catch (error) {
    throw new AppError('Failed to process', { code: 'PROCESS_ERROR' });
  }
}
```

### Shared Imports

Always import from `@wealth-management/*` scoped packages:

```typescript
// ✅ Correct
import { useAccount } from '@wealth-management/features/accounts';
import { ChatProvider } from '@wealth-management/features/chat';

// ❌ Wrong
import { useAccount } from '../../../libs/wealth-management/src/features';
```

---

## AI INTEGRATION

**System Prompt**: `libs/wealth-management/src/ai/system-prompt.ts`

- Defines assistant personality, constraints, and available tools
- Tools auto-invoke vnstock, Google Sheets, and portfolio calculations

**Streaming Response Pattern**:

```typescript
const response = await anthropic.messages.create({
  stream: true,
  model: 'claude-3-5-sonnet-20241022',
  system: SYSTEM_PROMPT,
  tools: AVAILABLE_TOOLS,
  messages: userMessages,
});
```

---

## GOOGLE SHEETS + REDIS CACHING

**Architecture**:

1. User data stored in Google Sheets (accounts, transactions, goals)
2. Upstash Redis caches sheets data + vnstock quotes
3. Cache invalidated on user action or TTL expiry

**Key Service**: `libs/wealth-management/src/services/sheets/google-sheets.ts`

---

## GOTCHAS & ANTI-PATTERNS

**DO NOT:**

- Swallow errors silently (all errors must log + notify)
- Hardcode stock symbols—always validate via vnstock APIs
- Mix app-local state with shared lib state
- Assume Google Sheets cells always contain expected data (validate first)

**ALWAYS:**

- Use `AppError` for custom errors
- Cache vnstock quotes (API limits exist)
- Log all AI tool invocations
- Validate user input before Google Sheets queries

---

## ENVIRONMENT SETUP

See `docs/wealth-management/SETUP-GUIDE.md` for:

- Google Sheets API key + service account
- Anthropic API key
- Upstash Redis URL
- vnstock API setup (free vs. sponsored)

---

## COMMON TASKS

| Task                 | Location                                              |
| -------------------- | ----------------------------------------------------- |
| Add new account type | `libs/wealth-management/src/features/accounts/model/` |
| Extend AI tools      | `libs/wealth-management/src/ai/tools/`                |
| Add caching layer    | `libs/wealth-management/src/services/cache/`          |
| Create new page      | `src/app/[page]/page.tsx`                             |
