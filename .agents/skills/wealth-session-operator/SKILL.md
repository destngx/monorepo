---
name: wealth-session-operator
description: Guides wealth-management dashboard workflows by choosing the right engine tools for accounts, balances, transactions, holdings, market checks, and system status reviews. Use when the user asks for a session briefing, account review, portfolio snapshot, market context, budget review
compatibility: Requires the wealth-management-engine graph runtime with Google Sheets, Fmarket, and web search tools enabled.
metadata:
  owner: wealth-management
  version: 1.0.0
  primary-ui: ai-test
---

# Wealth Session Operator

## Instructions

Treat each request as a guided workflow:

1. Read the system prompt as the response contract.
2. Read the business prompt as the operating plan.
3. Read the optional user prompt as task-specific detail.
4. Prefer exact tool results before freeform synthesis.

## Tool routing

### Accounts and cash

- Use `GetAccounts` for full account listings.
- Use `GetAccounts` even for a specific balance request, then narrow the response to the requested account such as cash or ZaloPay.
- Use `GetTransactions` when the workflow needs recent activity or cashflow detail.

### Planning and obligations

- Use `GetBudget` for budget state.
- Use `GetGoals` for savings or target planning.
- Use `GetLoans` when debt or repayment context matters.
- Use `GetCategories` and `GetTags` when transaction grouping or reporting labels are needed.

### Investing and market context

- Use `GetInvestmentAssets` for holdings.
- Use `GetFmarketTicker`, `GetFmarketExchangeRate`, `GetFmarketBankRates`, and `GetFmarketPriceSeries` when the workflow needs explicit Fmarket data.
- For explicit exchange-rate requests, use `GetFmarketExchangeRate` with the currency pair the user asked for.
- For exchange-rate outlook or probability requests, gather the exact rate tool first, then add `WebSearch` only if the user also asks for forecast, trend, p90/p95, or external context.
- For Vietnam or domestic gold price requests, use `GetFmarketTicker` with `symbol=SJC` and `type=gold`.
- For global gold reference requests, use `GetFmarketPriceSeries` with `symbol=GOLD` and `type=gold_usd`.
- If the user asks for both Vietnam and global gold context, call both gold tools before responding.
- Use `WebSearch` when the user asks for current news, fresh market context, or external references.
- Prefer the exact Fmarket tools over `WebSearch` when the request is asking for current prices, rates, or explicit market data that the engine already exposes.

### Operations and freshness

- Use `EngineHealth` when the user asks if the system is healthy.
- Use `SyncSheetsCache` when stale Google Sheets data is suspected.
- Use `WarmAIContent` when the workflow should refresh AI-facing content before analysis.

## Response discipline

- In chat mode, respond like an operator: concise, actionable, and grounded in the tool outputs you used.
- In JSON mode, keep the response deterministic and UI-ready.
- If a tool can answer the question exactly, call it before summarizing.
- If the user asks for multiple concrete datapoints, gather each required tool result before summarizing.
- If a workflow spans both account state and market context, gather internal data first, then add market or web context only when it improves the result.
- Prefer MCP-exposed tools for operational workflows so the guidance matches the engine surface available to the AI runtime.

## Reference

Consult `references/tool-playbook.md` when the workflow spans multiple tool families.
