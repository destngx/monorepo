# Wealth Session Operator Tool Playbook

## Session briefing

1. Start with `GetAccounts` or `GetInvestmentAssets` depending on whether the user needs cash or portfolio context first.
2. Add `GetTransactions` if recent movement is relevant.
3. Add `WebSearch` or an Fmarket tool only when the user explicitly needs current market context.
4. End with a short synthesis tied to the requested response mode.

## Account support

- Full account picture: `GetAccounts`
- Single balance: `GetAccounts`, then focus the answer on the requested account
- Recent movement: `GetTransactions`
- Planning overlay: `GetBudget`, `GetGoals`, `GetLoans`

## Market support

- News and macro context: `WebSearch`
- Explicit Fmarket data: `GetFmarketTicker`, `GetFmarketExchangeRate`, `GetFmarketBankRates`, `GetFmarketPriceSeries`
- Exchange-rate request: call `GetFmarketExchangeRate` for the requested pair before adding any commentary
- Exchange-rate outlook / p90 / p95: call `GetFmarketExchangeRate` first, then `WebSearch` only if the user also needs forecast or external context
- Vietnam gold price: `GetFmarketTicker` with `symbol=SJC`, `type=gold`
- Global gold reference: `GetFmarketPriceSeries` with `symbol=GOLD`, `type=gold_usd`
- Combined Vietnam + global gold request: call both gold tools first, then summarize the relationship between domestic VND pricing and global USD pricing

## Reliability checks

- Health check: `EngineHealth`
- Refresh Sheets-backed data: `SyncSheetsCache`
- Refresh AI-facing content: `WarmAIContent`
