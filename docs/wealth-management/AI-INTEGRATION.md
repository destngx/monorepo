# AI Integration

## Wealth Management System — Vercel AI SDK Design

**Version:** 1.0
**Last Updated:** 2026-02-28

---

## 1. Overview

The AI layer provides:

1. **Chat interface** - Natural language Q&A about your finances
2. **Auto-categorization** - Classify transactions into spending categories
3. **Budget advisor** - Suggest budget adjustments based on spending patterns
4. **Wealth advisory** - Portfolio analysis, rebalancing suggestions, financial health assessment
5. **Net worth projections** - AI-powered forecasting with what-if scenarios
6. **Goal planning** - AI calculates saving requirements to reach financial goals
7. **Financial health** - AI scores and generates improvement action items
8. **Provider flexibility** - Switch between OpenAI, Google Gemini, and Anthropic

Built on **Vercel AI SDK 4** for native Next.js streaming, tool calling, and multi-provider support.

---

## 2. Provider Architecture

### 2.1 Provider Configuration

```typescript
// src/lib/ai/providers.ts
import { openai, createOpenAI } from '@ai-sdk/openai';
import { google, createGoogleGenerativeAI } from '@ai-sdk/google';
import { anthropic, createAnthropic } from '@ai-sdk/anthropic';

// Auth mode: 'api_key' | 'oauth'
export type AuthMode = 'api_key' | 'oauth';

export interface ProviderAuth {
  mode: AuthMode;
  apiKey?: string;
  oauthToken?: string;
}

export type AIProviderKey = 'openai' | 'google' | 'anthropic';

export interface AIModelConfig {
  provider: AIProviderKey;
  model: string;
  label: string;
  description: string;
}

export const AI_MODELS: Record<string, AIModelConfig> = {
  'gpt-4o': {
    provider: 'openai',
    model: 'gpt-4o',
    label: 'GPT-4o',
    description: 'Best quality, higher cost',
  },
  'gpt-4o-mini': {
    provider: 'openai',
    model: 'gpt-4o-mini',
    label: 'GPT-4o Mini',
    description: 'Good quality, lower cost',
  },
  'gemini-2.0-flash': {
    provider: 'google',
    model: 'gemini-2.0-flash',
    label: 'Gemini 2.0 Flash',
    description: 'Fast, good for categorization',
  },
  'gemini-2.5-pro': {
    provider: 'google',
    model: 'gemini-2.5-pro-preview-06-05',
    label: 'Gemini 2.5 Pro',
    description: 'Best Google model, deep analysis',
  },
  'claude-sonnet': {
    provider: 'anthropic',
    model: 'claude-sonnet-4-20250514',
    label: 'Claude Sonnet',
    description: 'Balanced quality and speed',
  },
};

export function getLanguageModel(modelId: string, auth?: ProviderAuth) {
  const config = AI_MODELS[modelId];
  if (!config) throw new Error(`Unknown model: ${modelId}`);

  switch (config.provider) {
    case 'openai':
      if (auth?.mode === 'oauth' && auth.oauthToken) {
        // OAuth: use access token as API key (becomes Bearer header)
        return createOpenAI({ apiKey: auth.oauthToken })('gpt-4o');
      }
      return openai(config.model); // Falls back to OPENAI_API_KEY env var

    case 'google':
      if (auth?.mode === 'oauth' && auth.oauthToken) {
        // OAuth: Google access token works directly
        return createGoogleGenerativeAI({ apiKey: auth.oauthToken })(config.model);
      }
      return google(config.model); // Falls back to GOOGLE_GENERATIVE_AI_API_KEY env var

    case 'anthropic':
      // Anthropic: API key only (OAuth banned Feb 2026)
      if (auth?.mode === 'oauth') {
        throw new Error('Anthropic does not support OAuth. Use API key.');
      }
      return anthropic(config.model);
  }
}
```

### 2.2 Provider Switching

Users switch providers from the **Settings** page or the **Chat interface** header:

```
Settings → AI Provider:
  ○ OpenAI (GPT-4o)           [API Key: ✅ Configured]
  ○ OpenAI (GPT-4o Mini)      [API Key: ✅ Configured]
  ○ Google (Gemini 2.0 Flash) [API Key: ✅ Configured]
  ○ Google (Gemini 2.5 Pro)   [API Key: ✅ Configured]
  ○ Anthropic (Claude Sonnet) [API Key: ❌ Not configured]
```

Selected model is stored in SQLite `settings` table and loaded on each request.

### 2.4 OAuth Authentication

The app supports dual authentication for AI providers:

**OpenAI OAuth:**

- Flow: Authorization Code with PKCE
- Auth URL: https://auth.openai.com/oauth/authorize
- Token URL: https://auth.openai.com/oauth/token
- Client ID: Registered via OpenAI developer dashboard
- Scopes: openid profile email offline_access
- Token refresh: Automatic using refresh_token before expiration

**Google OAuth (shared with Sheets):**

- Flow: Standard OAuth 2.0 Authorization Code
- Scopes: https://www.googleapis.com/auth/spreadsheets + https://www.googleapis.com/auth/generativelanguage
- Benefit: Single Google login grants both Sheets and Gemini access
- If user has already authorized Sheets via Google OAuth, Gemini access is automatic (if scope was included)

**Anthropic:**

- API key only (sk-ant-api03-xxx)
- OAuth was banned by Anthropic on Feb 19, 2026
- No OAuth support planned

**Auth Priority:**

1. If OAuth token exists and is valid → use OAuth
2. If OAuth token expired → attempt refresh
3. If refresh fails or no OAuth → fall back to API key
4. If no API key → provider unavailable

**Settings UI:**

```
AI Provider: OpenAI
  Authentication:
    ○ API Key [sk-...xxxx] ✅ Configured
    ○ OAuth   [Connect with OpenAI] 🔗 Click to authorize

  Active: OAuth ✅ (connected as user@example.com)
```

### 2.3 Fallback Strategy

If the selected provider fails:

1. Retry once with same provider
2. If still fails, try next configured provider (in order: OpenAI → Google → Anthropic)
3. If all fail, return error message with "AI temporarily unavailable"

---

## 3. System Prompt

### 3.1 Dynamic System Prompt Builder

The system prompt is rebuilt on each request to include fresh financial context:

```typescript
// src/lib/ai/system-prompt.ts

export async function buildSystemPrompt(): Promise<string> {
  // Fetch minimal context (cached, fast)
  const accountSummary = await getAccountSummaryForAI();
  const budgetStatus = await getBudgetStatusForAI();
  const portfolioSummary = await getPortfolioSummaryForAI();
  const goalsSummary = await getGoalsSummaryForAI();
  const healthScore = await getHealthScoreForAI();
  const currentDate = new Date().toLocaleDateString('vi-VN');

  return `You are a personal wealth management advisor for a Vietnamese user. You have access to their real financial data via tools.

## Your Role
- Analyze spending patterns and provide actionable insights
- Track and advise on investment portfolio (crypto, gold, stocks, savings deposits, real estate)
- Monitor progress toward financial goals and suggest adjustments
- Provide asset allocation and rebalancing recommendations
- Run net worth projections and what-if scenarios
- Calculate financial health scores and suggest improvements
- Help categorize transactions and suggest budget improvements
- Be concise, data-driven, and specific to their financial situation
- When showing amounts in VND, format as: 1,000,000 ₫
- When showing USDT, format as: $1,500.00
- Understand Vietnamese financial context (SJC gold, kỳ hạn savings, VND)

## Current Context
- Today: ${currentDate}
- Accounts: ${accountSummary}
- This month's budget status: ${budgetStatus}
- Portfolio: ${portfolioSummary}
- Goals: ${goalsSummary}
- Financial health: ${healthScore}/100

## Available Tools
You have access to tools to query the user's financial data. Always use tools to get accurate data rather than guessing.

### Spending & Budget Tools
- getAccountBalances: Get all account balances
- getTransactions: Query transactions with filters
- getBudgetStatus: Budget vs. actual spending
- getSpendingByCategory: Spending breakdown by category
- getMonthlyTrend: Monthly income/expense totals
- searchTransactions: Search transactions by text
- categorizeTransaction: Suggest category for a transaction

### Wealth Management Tools
- getPortfolioSummary: Full investment portfolio with allocation, P&L, current values
- getAssetPrice: Get current price for any tracked asset
- getNetWorthHistory: Historical net worth snapshots for trend analysis
- getGoalProgress: Progress toward financial goals with projections
- getFinancialHealthScore: Current health score with metric breakdown
- projectNetWorth: Run projection model (input: years, additional savings, expected returns)
- suggestRebalancing: Compare current allocation to targets, suggest trades
- getPassiveIncome: Passive income streams and totals
- runWhatIfScenario: Model impact of financial decisions

## Rules
1. Always use tools to fetch data before answering - never guess
2. For investment advice, always caveat that you're not a licensed financial advisor
3. Format currency appropriately (VND for local, USD for crypto/international)
4. When comparing periods, always specify which periods
5. For portfolio advice, consider the user's risk profile and Vietnamese market context
6. Be direct and concise - insights, not lectures
7. If you can't determine something from the data, say so clearly
8. Numbers use Vietnamese formatting: 1.000.000 ₫ for VND in chat`;
```

---

## 4. AI Tools (Function Calling)

### 4.1 Tool Definitions

```typescript
// src/lib/ai/tools.ts
import { z } from 'zod';
import { tool } from 'ai';

export const financialTools = {
  getAccountBalances: tool({
    description: 'Get all account balances including bank accounts (VND) and crypto (USDT)',
    parameters: z.object({}),
    execute: async () => {
      const accounts = await getAccounts();
      const rate = await getExchangeRate();
      return {
        accounts: accounts.map((a) => ({
          name: a.name,
          balance: a.balance,
          currency: a.currency,
          type: a.type,
          balanceInVND: a.currency === 'USDT' ? a.balance * rate : a.balance,
        })),
        exchangeRate: rate,
        totalNetWorthVND: calculateNetWorth(accounts, rate),
      };
    },
  }),

  getTransactions: tool({
    description:
      'Get transactions with optional filters. Use this to look up specific transactions or analyze spending.',
    parameters: z.object({
      dateFrom: z.string().optional().describe('Start date in YYYY-MM-DD format'),
      dateTo: z.string().optional().describe('End date in YYYY-MM-DD format'),
      account: z.string().optional().describe('Filter by account name'),
      category: z.string().optional().describe('Filter by category name'),
      limit: z.number().optional().default(50).describe('Maximum number of transactions to return'),
    }),
    execute: async ({ dateFrom, dateTo, account, category, limit }) => {
      const transactions = await getTransactions({
        dateFrom: dateFrom ? new Date(dateFrom) : undefined,
        dateTo: dateTo ? new Date(dateTo) : undefined,
        account,
        category,
        limit,
      });
      return {
        count: transactions.length,
        totalPayments: sumPayments(transactions),
        totalDeposits: sumDeposits(transactions),
        transactions: transactions.map((t) => ({
          date: formatDate(t.date),
          payee: t.payee,
          category: t.category,
          amount: t.payment || t.deposit,
          type: t.payment ? 'payment' : 'deposit',
          account: t.accountName,
          memo: t.memo,
        })),
      };
    },
  }),

  getBudgetStatus: tool({
    description: 'Get budget status showing limit vs actual spending for each category',
    parameters: z.object({
      month: z.string().optional().describe('Month in YYYY-MM format. Defaults to current month.'),
    }),
    execute: async ({ month }) => {
      const targetMonth = month || getCurrentMonth();
      const budget = await getBudgetWithActuals(targetMonth);
      return {
        month: targetMonth,
        categories: budget.map((b) => ({
          category: b.category,
          monthlyLimit: b.monthlyLimit,
          spent: b.monthlySpent,
          remaining: b.monthlyRemaining,
          percentUsed: b.monthlyPercentUsed,
          status: b.monthlyPercentUsed > 100 ? 'over' : b.monthlyPercentUsed > 80 ? 'warning' : 'ok',
        })),
        totalBudget: sumField(budget, 'monthlyLimit'),
        totalSpent: sumField(budget, 'monthlySpent'),
      };
    },
  }),

  getSpendingByCategory: tool({
    description: 'Get spending breakdown by category for a date range',
    parameters: z.object({
      dateFrom: z.string().describe('Start date in YYYY-MM-DD format'),
      dateTo: z.string().describe('End date in YYYY-MM-DD format'),
    }),
    execute: async ({ dateFrom, dateTo }) => {
      const transactions = await getTransactions({
        dateFrom: new Date(dateFrom),
        dateTo: new Date(dateTo),
        type: 'payment',
      });
      const byCategory = groupByCategory(transactions);
      return {
        period: { from: dateFrom, to: dateTo },
        totalSpent: sumPayments(transactions),
        categories: Object.entries(byCategory)
          .map(([cat, txns]) => ({
            category: cat,
            amount: sumPayments(txns),
            count: txns.length,
            percentage: ((sumPayments(txns) / sumPayments(transactions)) * 100).toFixed(1),
          }))
          .sort((a, b) => b.amount - a.amount),
      };
    },
  }),

  getExchangeRate: tool({
    description: 'Get current USDT to VND exchange rate',
    parameters: z.object({}),
    execute: async () => {
      const rate = await getExchangeRate();
      return {
        pair: 'USDT/VND',
        rate,
        timestamp: new Date().toISOString(),
      };
    },
  }),

  categorizeTransaction: tool({
    description: 'Suggest a spending category for a transaction based on payee and memo',
    parameters: z.object({
      payee: z.string().describe('Transaction payee/merchant name'),
      memo: z.string().optional().describe('Transaction memo or description'),
      amount: z.number().optional().describe('Transaction amount'),
    }),
    execute: async ({ payee, memo, amount }) => {
      return await suggestCategory(payee, memo, amount);
    },
  }),

  getMonthlyTrend: tool({
    description: 'Get monthly income, expense, and net totals for trend analysis',
    parameters: z.object({
      months: z.number().optional().default(6).describe('Number of months to look back'),
    }),
    execute: async ({ months }) => {
      return await getMonthlyTrend(months);
    },
  }),

  searchTransactions: tool({
    description: 'Search transactions by text in payee or memo fields',
    parameters: z.object({
      query: z.string().describe('Search text to find in payee or memo'),
      limit: z.number().optional().default(20),
    }),
    execute: async ({ query, limit }) => {
      return await searchTransactions(query, limit);
    },
  }),

  getPortfolioSummary: tool({
    description: 'Get full investment portfolio summary: all assets, allocation percentages, total value, P&L',
    parameters: z.object({}),
    execute: async () => {
      const portfolio = await getPortfolioWithPrices();
      return {
        totalValueVND: portfolio.totalValueVND,
        totalCostBasisVND: portfolio.totalCostBasisVND,
        totalPnLVND: portfolio.totalPnLVND,
        totalPnLPercent: portfolio.totalPnLPercent,
        allocation: portfolio.allocation, // { crypto: 45%, cash: 30%, gold: 15%, ... }
        assets: portfolio.assets.map((a) => ({
          name: a.name,
          type: a.type,
          quantity: a.quantity,
          costBasis: a.costBasis,
          currentPrice: a.currentPrice,
          currentValue: a.currentValueVND,
          pnl: a.pnlVND,
          pnlPercent: a.pnlPercent,
          allocation: a.allocationPercent,
        })),
      };
    },
  }),

  getAssetPrice: tool({
    description: 'Get current market price for a specific asset (crypto, gold, stock)',
    parameters: z.object({
      symbol: z.string().describe('Asset symbol: BTC, ETH, GOLD_SJC, VN30, etc.'),
    }),
    execute: async ({ symbol }) => {
      return await getMarketPrice(symbol);
    },
  }),

  getNetWorthHistory: tool({
    description: 'Get historical net worth snapshots for trend analysis',
    parameters: z.object({
      months: z.number().optional().default(12).describe('Number of months of history'),
    }),
    execute: async ({ months }) => {
      return await getNetWorthSnapshots(months);
    },
  }),

  getGoalProgress: tool({
    description: 'Get progress toward financial goals with projected completion dates',
    parameters: z.object({
      goalId: z.string().optional().describe('Specific goal ID, or omit for all goals'),
    }),
    execute: async ({ goalId }) => {
      if (goalId) return await getGoalById(goalId);
      return await getAllGoalsWithProgress();
    },
  }),

  getFinancialHealthScore: tool({
    description: 'Get financial health score (0-100) with breakdown by metric',
    parameters: z.object({}),
    execute: async () => {
      return await calculateHealthScore();
    },
  }),

  projectNetWorth: tool({
    description: 'Project future net worth based on current savings rate, investment returns, and spending',
    parameters: z.object({
      years: z.number().default(5).describe('Years to project'),
      additionalMonthlySavings: z.number().optional().describe('Extra monthly savings in VND'),
      expectedAnnualReturn: z
        .number()
        .optional()
        .describe('Expected annual return % (default: based on current portfolio)'),
    }),
    execute: async ({ years, additionalMonthlySavings, expectedAnnualReturn }) => {
      return await runProjection({ years, additionalMonthlySavings, expectedAnnualReturn });
    },
  }),

  suggestRebalancing: tool({
    description: 'Compare current asset allocation to targets and suggest rebalancing moves',
    parameters: z.object({}),
    execute: async () => {
      return await generateRebalancingSuggestions();
    },
  }),

  getPassiveIncome: tool({
    description: 'Get passive income summary: staking rewards, interest, dividends, rental income',
    parameters: z.object({
      months: z.number().optional().default(6).describe('Months of passive income history'),
    }),
    execute: async ({ months }) => {
      return await getPassiveIncomeSummary(months);
    },
  }),

  runWhatIfScenario: tool({
    description: 'Model the impact of a financial decision on future net worth',
    parameters: z.object({
      scenario: z.string().describe('Description of the scenario to model'),
      monthlyChange: z.number().optional().describe('Monthly change in savings/spending (VND)'),
      oneTimeChange: z.number().optional().describe('One-time change in asset value (VND)'),
      assetAllocationChange: z.record(z.number()).optional().describe('New target allocation percentages'),
      years: z.number().optional().default(5).describe('Years to project'),
    }),
    execute: async (params) => {
      return await modelScenario(params);
    },
  }),
};
```

### 4.2 Tool Flow

```
User: "How much did I spend on food this month?"
  │
  ▼
AI decides to call: getSpendingByCategory({ dateFrom: "2026-02-01", dateTo: "2026-02-28" })
  │
  ▼
Tool executes: Fetches transactions → Groups by category → Returns breakdown
  │
  ▼
AI receives tool result → Filters for "Food & Dining" → Formats response
  │
  ▼
Response: "You spent 3,450,000 ₫ on Food & Dining this month across 23 transactions.
           That's 72% of your 4,800,000 ₫ monthly budget."
```

---

### 4.3 Wealth Advisory Tool Flow

**Scenario 1: Portfolio Analysis**

```
User: "How should I rebalance my portfolio?"
  |
  ▼
AI calls: getPortfolioSummary() + suggestRebalancing()
  |
  ▼
Tools return: Current allocation + target allocation + suggested moves
  |
  ▼
AI formats response with specific recommendations
```

**Scenario 2: Net Worth Projection**

```
User: "Project my net worth in 5 years"
  |
  ▼
AI calls: getPortfolioSummary() + getMonthlyTrend({months: 12}) + projectNetWorth({years: 5})
  |
  ▼
Tools return: Current net worth + savings trend + projected growth curve
  |
  ▼
AI formats: Current net worth, projected values at years 1-5, assumptions used
```

**Scenario 3: Financial Health Check**

```
User: "What's my financial health score?"
  |
  ▼
AI calls: getFinancialHealthScore() + getGoalProgress()
  |
  ▼
Tools return: Health score + metrics breakdown + goal status
  |
  ▼
AI formats: Score, improvement areas, action items
```

## 5. Auto-Categorization

### 5.1 Categorization Pipeline

```
New Transaction (payee, memo, amount)
    │
    ▼
┌─────────────────────────┐
│ 1. Check corrections DB │ ← Past user corrections for same payee
│    (exact payee match)  │
│    Found? → Use corrected category
└────────┬────────────────┘
         │ Not found
         ▼
┌─────────────────────────┐
│ 2. Rule-based matching  │ ← Keyword/pattern rules
│    "Grab" + amount<200k │
│    → Transportation     │
│    "Grab" + amount>200k │
│    → Food & Dining      │
│    Match? → Use rule    │
└────────┬────────────────┘
         │ No match
         ▼
┌─────────────────────────┐
│ 3. AI categorization    │ ← Send to AI with context
│    Prompt includes:     │
│    - Category taxonomy  │
│    - Recent corrections │
│    - Transaction details│
│    Returns: category,   │
│    confidence, reasoning│
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ 4. Confidence check     │
│    ≥80% → Auto-assign   │
│    <80% → Suggest top 3 │
│    for user confirmation │
└─────────────────────────┘
```

### 5.2 Categorization Prompt

```typescript
// src/lib/ai/categorizer.ts

const CATEGORIZE_PROMPT = `You are a transaction categorization engine for a Vietnamese user's personal finances.

Given a transaction, assign it to the most appropriate category from the taxonomy below.

## Categories
${CATEGORY_LIST_WITH_KEYWORDS}

## Transaction Details
- Payee: {payee}
- Memo: {memo}
- Amount: {amount} {currency}
- Account: {account}

## Recent corrections for similar payees (learn from these):
{corrections}

## Instructions
1. Consider the payee name, memo, amount, and account context
2. Vietnamese merchants: match based on common Vietnamese business names
3. Amount can help distinguish (e.g., Grab small amount = transport, large = food delivery)
4. If uncertain, provide your top 3 guesses with confidence scores

Respond in JSON:
{
  "category": "Category Name",
  "subcategory": "Subcategory Name",
  "confidence": 0.95,
  "reasoning": "Brief explanation",
  "alternatives": [
    {"category": "Alt Category", "confidence": 0.7}
  ]
}`;
```

### 5.3 Learning from Corrections

When a user corrects a category:

1. Store correction in `CategoryCorrection` table
2. Future transactions with same payee → check corrections first
3. Corrections are included in AI prompt as examples
4. Over time, the system learns the user's specific categorization preferences

---

## 6. Chat API Implementation

### 6.1 Chat Route

```typescript
// src/app/api/chat/route.ts
import { streamText } from 'ai';
import { getLanguageModel } from '@/lib/ai/providers';
import { financialTools } from '@/lib/ai/tools';
import { buildSystemPrompt } from '@/lib/ai/system-prompt';

export async function POST(req: Request) {
  const { messages, modelId } = await req.json();

  // Get selected model (from request or settings)
  const selectedModel = modelId || (await getDefaultModelId());
  const model = getLanguageModel(selectedModel);

  // Build system prompt with fresh financial context
  const systemPrompt = await buildSystemPrompt();

  const result = streamText({
    model,
    system: systemPrompt,
    messages,
    tools: financialTools,
    maxSteps: 5, // Allow up to 5 tool call rounds
    onFinish: async ({ text, toolCalls, toolResults }) => {
      // Persist chat history to SQLite
      await saveChatMessage({
        role: 'assistant',
        content: text,
        toolCalls,
        toolResults,
        model: selectedModel,
      });
    },
  });

  return result.toDataStreamResponse();
}
```

### 6.2 Client-Side Chat

```typescript
// src/components/chat/chat-interface.tsx
'use client';

import { useChat } from 'ai/react';

export function ChatInterface() {
  const { messages, input, handleInputChange, handleSubmit, isLoading } = useChat({
    api: '/api/chat',
    body: {
      modelId: selectedModel, // From settings context
    },
  });

  return (
    <div className="flex flex-col h-full">
      <div className="flex-1 overflow-y-auto">
        {messages.map(m => (
          <MessageBubble key={m.id} message={m} />
        ))}
      </div>
      <form onSubmit={handleSubmit}>
        <input
          value={input}
          onChange={handleInputChange}
          placeholder="Ask about your finances..."
        />
        <button type="submit" disabled={isLoading}>Send</button>
      </form>
    </div>
  );
}
```

---

## 7. Structured Output (Non-Chat AI)

For non-chat AI tasks (categorization, budget suggestions), use `generateObject` for reliable structured output:

```typescript
import { generateObject } from 'ai';
import { z } from 'zod';

const categorizationSchema = z.object({
  category: z.string(),
  subcategory: z.string().optional(),
  confidence: z.number().min(0).max(1),
  reasoning: z.string(),
  alternatives: z
    .array(
      z.object({
        category: z.string(),
        confidence: z.number(),
      }),
    )
    .optional(),
});

export async function categorizeWithAI(payee: string, memo: string | null, amount: number) {
  const model = getLanguageModel(await getDefaultModelId());

  const { object } = await generateObject({
    model,
    schema: categorizationSchema,
    prompt: buildCategorizationPrompt(payee, memo, amount),
  });

  return object;
}
```

---

## 8. Cost Management

### 8.1 Token Estimation

| Operation                  | Est. Tokens  | Cost (GPT-4o)   | Frequency |
| -------------------------- | ------------ | --------------- | --------- |
| Chat message (with tools)  | ~2,000-5,000 | $0.01-0.025     | ~10/day   |
| Transaction categorization | ~500         | $0.0025         | ~5/day    |
| Budget analysis            | ~3,000       | $0.015          | ~2/day    |
| Portfolio analysis         | ~3,000-5,000 | $0.015-0.025    | ~3/day    |
| Goal planning/projection   | ~2,000-4,000 | $0.01-0.02      | ~2/day    |
| Health score + actions     | ~2,000       | $0.01           | ~1/day    |
| What-if scenario           | ~3,000-5,000 | $0.015-0.025    | ~1/day    |
| **Daily estimate**         |              | **~$0.25-0.50** |           |
| **Monthly estimate**       |              | **~$8-15**      |           |

### 8.2 Cost Optimization

1. **Use cheaper models for categorization** — GPT-4o Mini or Gemini Flash (5–10x cheaper)
2. **Cache AI responses** — Same question within 1 hour → return cached
3. **Rate limiting** — Max 30 chat messages per hour
4. **Rule-based first** — Categorization checks rules before calling AI
5. **Minimal context** — System prompt only includes summary, not full transaction history

### 8.3 Model Selection Guide

| Task                    | Recommended Model                       | Reason                                |
| ----------------------- | --------------------------------------- | ------------------------------------- |
| Chat (complex analysis) | GPT-4o / Gemini 2.5 Pro                 | Best reasoning                        |
| Chat (simple Q&A)       | GPT-4o Mini / Gemini Flash              | Fast, cheap                           |
| Categorization          | GPT-4o Mini / Gemini Flash              | Simple classification                 |
| Budget suggestions      | GPT-4o / Claude Sonnet                  | Needs nuanced reasoning               |
| Wealth advisory         | GPT-4o / Claude Sonnet / Gemini 2.5 Pro | Complex reasoning + financial context |
| What-if scenarios       | GPT-4o / Gemini 2.5 Pro                 | Mathematical projection + explanation |
| Rebalancing             | GPT-4o Mini / Gemini Flash              | Pattern matching against targets      |

---

## 9. Example Conversations

### Example 1: Spending Query

```
User: How much have I spent on food this month?

[AI calls getSpendingByCategory({dateFrom: "2026-02-01", dateTo: "2026-02-28"})]
[Tool returns: Food & Dining: 3,450,000 VND, 23 transactions]

AI: You've spent **3,450,000 ₫** on Food & Dining this month across 23 transactions.

That's **72%** of your 4,800,000 ₫ monthly budget, with 0 days remaining in February.

Top payees:
- GrabFood: 1,200,000 ₫ (8 orders)
- Highland Coffee: 450,000 ₫ (15 visits)
- Bách Hóa Xanh: 800,000 ₫ (4 trips)
```

### Example 2: Budget Advice

```
User: Am I on track with my budget?

[AI calls getBudgetStatus({month: "2026-02"})]
[AI calls getMonthlyTrend({months: 3})]

AI: Here's your February budget status:

✅ **On track:** Transportation (45%), Bills (60%), Personal (35%)
⚠️ **Watch out:** Food & Dining (72%), Entertainment (78%)
🔴 **Over budget:** Shopping (115% — 920,000 ₫ over)

Compared to the last 3 months, your Food & Dining spending is trending up (+15%).
Shopping spike was due to the 1,500,000 ₫ electronics purchase on Feb 15.

**Suggestion:** Your Entertainment budget (2,000,000 ₫) has been consistently under-used
(avg 65% usage). Consider shifting 500,000 ₫ from Entertainment to Food & Dining.
```

### Example 3: Category Help

```
User: I have a transaction from "Thế Giới Di Động" for 15,000,000 VND. What category?

[AI calls categorizeTransaction({payee: "Thế Giới Di Động", amount: 15000000})]

AI: **Thế Giới Di Động** (Mobile World) is Vietnam's largest electronics retailer.

For 15,000,000 ₫, this is likely a phone or laptop purchase.

Suggested category: **Shopping → Electronics** (95% confidence)

Should I categorize it as Shopping, or would you prefer a different category?
```

### Example 4: Portfolio Analysis

```
User: What's my portfolio performance this year?

[AI calls getPortfolioSummary()]
[Tool returns: 12 assets, total value 850M VND, total P&L +120M VND (+16.4%)]

AI: Your portfolio is up **16.4%** year-to-date with a total value of **850,000,000 ₫**.

**Top performers:**
- Bitcoin (BTC): +42% (+45,000,000 ₫)
- Ethereum (ETH): +28% (+18,000,000 ₫)
- SJC Gold: +12% (+8,500,000 ₫)

**Underperformers:**
- VN30 Index Fund: -3% (-2,100,000 ₫)

**Allocation:** 45% Crypto, 25% Cash, 18% Gold, 12% Other

Your crypto allocation is above your 30% target. Consider taking some profits.
```

### Example 5: Net Worth Projection

```
User: Project my net worth in 5 years if I save an extra 5 million per month

[AI calls getPortfolioSummary() + getMonthlyTrend({months: 12}) + projectNetWorth({years: 5, additionalMonthlySavings: 5000000})]

AI: Based on your current trajectory plus 5,000,000 ₫/month extra savings:

| Year | Projected Net Worth | Growth |
|------|---------------------|--------|
| Now  | 850,000,000 ₫     | -      |
| +1   | 1,120,000,000 ₫   | +32%   |
| +3   | 1,850,000,000 ₫   | +118%  |
| +5   | 2,750,000,000 ₫   | +224%  |

**Assumptions:** 8% annual return (based on your historical portfolio performance), current savings rate of 12M ₫/month plus 5M ₫ additional.

At this rate, you'd hit 1 tỷ VND in about 10 months. Your emergency fund goal would be fully funded by June 2026.
```

### Example 6: Financial Health

```
User: What's my financial health score?

[AI calls getFinancialHealthScore()]

AI: Your financial health score is **72/100** - Good, with room for improvement.

**Score Breakdown:**
- Savings Rate: 85/100 (24% - excellent, above 20% target)
- Emergency Fund: 60/100 (2.8 months covered, target is 6 months)
- Diversification: 55/100 (crypto-heavy at 45%, target max 30%)
- Budget Adherence: 78/100 (on track in 12 of 16 categories)
- Net Worth Growth: 82/100 (+16.4% YoY)
- Debt Ratio: 90/100 (low debt, healthy)

**Top 3 Actions:**
1. Build emergency fund to 6 months (need extra approximately 150M ₫)
2. Reduce crypto allocation from 45% to 30% - move profits to gold or savings
3. Fix overspending in Shopping and Entertainment categories
```
