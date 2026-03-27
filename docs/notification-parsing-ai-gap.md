# Notification Parsing AI Provider Gap Analysis

**Date**: 2026-03-27  
**Status**: GAP IDENTIFIED ✓  
**Scope**: Refactor notification parsing to use shared AI providers

---

## EXECUTIVE SUMMARY

The **notification parsing API** (`/api/ai/parse-notifications`) is **NOT using shared AI provider infrastructure** unlike other AI features in the app. It directly instantiates OpenAI (`openai('gpt-4o')`) instead of delegating to:

1. **`getLanguageModel()`** - centralized provider factory
2. **`AIOrchestrator`** - unified orchestration layer
3. **Dynamic model selection** - respecting app configuration

### Gap Impact

- **No provider flexibility** - hardcoded to OpenAI, can't switch to GitHub Copilot or other providers
- **No modelId parameter** - users can't request different models
- **Inconsistent pattern** - other AI endpoints use `AIOrchestrator` or `getLanguageModel()`
- **Maintenance burden** - changes to AI infrastructure won't reach this endpoint

---

## FILE LOCATIONS

### 1. CURRENT (BROKEN) IMPLEMENTATION

**File**: `/apps/wealth-management/src/app/api/ai/parse-notifications/route.ts`

```typescript
// ❌ WRONG: Direct OpenAI instantiation (line 40)
import { openai } from '@ai-sdk/openai';

const { text } = await generateText({
  model: openai('gpt-4o'), // ← Hardcoded, no flexibility
  system: systemPrompt,
  prompt: actionPrompt,
});
```

**Issues**:

- Imports `openai` directly (line 3)
- No `modelId` parameter in request body
- No provider selection logic
- Inconsistent with app's multi-provider architecture

---

### 2. SHARED AI PROVIDER CONFIGURATION

**File**: `/libs/wealth-management/src/ai/providers.ts` (104 lines)

```typescript
export function getLanguageModel(modelId: string) {
  const config = AI_MODELS[modelId];

  if (!config) {
    if (modelId.startsWith('github-')) return github(modelId.replace('github-', ''));
    return github('gpt-4o'); // Intelligent default fallback
  }

  switch (config.provider) {
    case 'openai':
      return openai(config.model);
    case 'google':
      return google(config.model);
    case 'anthropic':
      return anthropic(config.model);
    case 'github':
      return github(config.model);
    default:
      return openai('gpt-4o-mini');
  }
}
```

**Features**:

- ✅ Supports OpenAI, Google, Anthropic, GitHub Copilot
- ✅ Dynamic model selection from `AI_MODELS` config
- ✅ Intelligent fallbacks
- ✅ GitHub Copilot token management (lines 10-31)

---

### 3. UNIFIED ORCHESTRATOR (RECOMMENDED PATTERN)

**File**: `/libs/wealth-management/src/ai/core/orchestrator.ts` (45 lines)

```typescript
export class AIOrchestrator {
  /**
   * Standard text generation task
   */
  static async run(options: OrchestratorOptions): Promise<string> {
    const model = getLanguageModel(options.modelId || 'github-gpt-4o');
    const system = await buildSystemPrompt(options.systemPromptInstruction);

    const { text } = await generateText({
      model,
      system,
      prompt: options.prompt,
      temperature: options.temperature,
    });

    return text;
  }

  /**
   * JSON-safe generation with automatic parsing
   */
  static async runJson<T>(options: OrchestratorOptions): Promise<T> {
    const text = await this.run(options);
    return extractAndParseJSON<T>(text);
  }
}
```

**Benefits**:

- ✅ Single source of truth for AI execution
- ✅ Built-in system prompt building
- ✅ Automatic JSON extraction
- ✅ Configurable modelId support

---

## REFERENCE IMPLEMENTATIONS (GOOD PATTERNS)

### Pattern A: Using AIOrchestrator (RECOMMENDED)

**File**: `/apps/wealth-management/src/app/api/ai/budget-advisor/route.ts`

```typescript
import { AIOrchestrator } from '@wealth-management/ai/core';

export async function POST(req: Request) {
  const { budget, transactions, goals, date, modelId } = body;

  // ✅ CORRECT: Accepts modelId from client
  const result = await AIOrchestrator.runJson<Record<string, unknown>>({
    modelId: modelId, // ← Respects client model selection
    systemPromptInstruction: taskInstruction,
    prompt: actionPrompt,
  });

  return NextResponse.json(result);
}
```

---

### Pattern B: Using getLanguageModel (ACCEPTABLE)

**File**: `/apps/wealth-management/src/app/api/ai/transaction-review/route.ts`

```typescript
import { getLanguageModel } from '@wealth-management/ai/providers';

export async function POST(req: Request) {
  // ✅ CORRECT: Uses shared provider factory
  const model = getLanguageModel('github-gpt-4.1');

  const { text } = await generateText({
    model, // ← Via shared factory
    system: systemPrompt,
    prompt: actionPrompt,
  });

  // Extract structured response
  const structured = extractAndParseJSON<StructuredInsight>(text);
  return NextResponse.json({ review: structured });
}
```

---

### Pattern C: Chat API (Multi-Provider, Full Control)

**File**: `/apps/wealth-management/src/app/api/chat/route.ts`

```typescript
import { getLanguageModel } from '@wealth-management/ai/providers';

export async function POST(req: Request) {
  const { messages, modelId, context } = body;

  // ✅ CORRECT: Intelligent default with user override
  let selectedModel = modelId;
  if (!selectedModel || selectedModel === 'gpt-4o-mini') {
    selectedModel = process.env.GITHUB_TOKEN ? 'github-gpt-4o' : 'gpt-4o-mini';
  }

  const model = getLanguageModel(selectedModel);

  const result = streamText({
    model, // ← Via shared factory
    system: systemPrompt,
    messages: sdkMessages,
    tools: financialTools,
    maxSteps: 10,
  });

  return result.toUIMessageStreamResponse();
}
```

---

## REFACTORING PLAN

### Step 1: Update Request Interface

Add optional `modelId` parameter to align with other endpoints:

```typescript
interface ParseNotificationInput {
  id: string;
  content: string;
}

interface ParseNotificationsRequest {
  notifications: ParseNotificationInput[];
  modelId?: string; // ← NEW: Allow client model selection
}
```

### Step 2: Replace Direct Instantiation

**Current** (lines 1-3, 39-43):

```typescript
import { openai } from '@ai-sdk/openai';

const { text } = await generateText({
  model: openai('gpt-4o'),
  system: systemPrompt,
  prompt: actionPrompt,
});
```

**Refactored**:

```typescript
import { getLanguageModel } from '@wealth-management/ai/providers';

const model = getLanguageModel(modelId || 'github-gpt-4o');

const { text } = await generateText({
  model, // ← Via shared factory
  system: systemPrompt,
  prompt: actionPrompt,
});
```

### Step 3: Remove Direct OpenAI Dependency

- Delete: `import { openai } from '@ai-sdk/openai';` (line 3)
- Keep: All other imports (ai, buildSystemPrompt, etc.)

### Step 4: Extract JSON Parsing

Use shared utility from `AIOrchestrator`:

```typescript
import { extractAndParseJSON } from '@wealth-management/ai/server';

// Instead of manual JSON extraction (lines 45-62):
const cleanText = text.trim();
const startBracket = cleanText.indexOf('[');
const endBracket = cleanText.lastIndexOf(']');

// Use:
try {
  const parsed = extractAndParseJSON<unknown>(text);
  return NextResponse.json(parsed);
} catch {
  throw new AppError('AI returned malformed JSON. Please try again.');
}
```

---

## TESTING CHECKLIST

After refactoring:

- [ ] Endpoint still accepts `notifications` array
- [ ] Default model selection works (falls back to 'github-gpt-4o')
- [ ] Optional `modelId` parameter is respected
- [ ] JSON parsing handles valid responses
- [ ] Error handling covers:
  - Missing notifications array
  - Invalid JSON from AI
  - Missing API keys
- [ ] Works with:
  - OpenAI (if `OPENAI_API_KEY` set)
  - GitHub Copilot (if `GITHUB_TOKEN` set)
  - Google (if `GOOGLE_GENERATIVE_AI_API_KEY` set)
  - Anthropic (if `ANTHROPIC_API_KEY` set)

---

## DOWNSTREAM IMPACT

### Dependencies on parse-notifications endpoint:

- Frontend: `src/features/transactions/ui/notification-processor.tsx`
- Service: `libs/wealth-management/src/services/sheets/notifications.ts`

### Compatibility:

- ✅ Changes are **backward compatible** (modelId is optional)
- ✅ Existing clients continue to work without modification
- ✅ New clients can pass `modelId` for flexibility

---

## ARCHITECTURE BENEFIT

After refactoring, notification parsing will align with:

1. **Centralized provider management** - single point of AI provider configuration
2. **Dynamic model selection** - respects `AI_MODELS` config changes
3. **Multi-provider support** - can switch from OpenAI → Copilot seamlessly
4. **Consistent error handling** - uses shared error infrastructure
5. **Maintenance efficiency** - future AI changes benefit all endpoints

---

## SUMMARY TABLE

| Aspect                     | Current            | After Refactor                      |
| -------------------------- | ------------------ | ----------------------------------- |
| **Provider Instantiation** | Direct `openai()`  | Via `getLanguageModel()`            |
| **Model Selection**        | Hardcoded 'gpt-4o' | Dynamic via modelId param           |
| **Multi-provider Support** | ❌ No              | ✅ Yes (all 4 providers)            |
| **JSON Parsing**           | Manual regex       | Via `extractAndParseJSON()`         |
| **Shared Infrastructure**  | ❌ No              | ✅ Yes (AIOrchestrator-compatible)  |
| **Consistency**            | ❌ No              | ✅ Yes (matches other AI endpoints) |
| **Backward Compatibility** | N/A                | ✅ Yes (optional params)            |

---

## FILES TO MODIFY

1. **`/apps/wealth-management/src/app/api/ai/parse-notifications/route.ts`** (72 lines)
   - Remove: Direct OpenAI import
   - Add: `getLanguageModel` import
   - Update: Model instantiation logic
   - Update: JSON parsing to use `extractAndParseJSON`
   - Update: Request interface to accept optional `modelId`

---

## NEXT STEPS

1. ✅ Gap identified and documented (this file)
2. 📝 Ready for implementation
3. 🧪 Unit tests should cover:
   - Default model selection
   - Client-provided modelId handling
   - JSON extraction from AI response
   - Error cases (missing keys, malformed JSON)
4. 📤 Create PR with changes + tests
