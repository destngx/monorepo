# Wealth Management: Roadmap

> **Status**: Approved direction, 2026-10-17. Order of work for turning the app from a sheet viewer into a personal assets assistant.
> **Rules**: an item gets requirement IDs and a status in [REQUIREMENTS.md](./REQUIREMENTS.md) when its PR is built; this file only orders the work. Decisions it relies on are D30-D32 ([§16](./REQUIREMENTS.md#16-open-decisions)). Parked ideas stay in [TODO.md](./TODO.md).

## Principles

- **Answer a question per page.** Each page leads with one headline answer ("Am I okay?", "Am I on track?", "What needs me?"), then the evidence, then the detail tables. The yearly cash flow page (WM-DASH-8) is the reference.
- **Advice is a loop.** A recommendation carries an action, an estimated VND impact and its evidence; the user marks it done, dismissed or snoozed; the next review checks the outcome.
- **Numbers before words.** The engine computes every figure; AI only explains and prioritises them.
- **The app manages its own tabs.** New tabs (`Goal Plan`, `Recurring`, `Preferences`, `Recommendations`, `Snapshots`) are created by the engine on first use and edited in the app; the sheet stays the store and can still be edited by hand. The engine writes only the rows the user changes in the app, and never restructures or deletes existing tabs (`Goals` and `'Loan / Dept'` stay as they are, unread).
- **No personal data in the repository.** No names, balances, goal amounts, account notes or real transactions in code, tests, fixtures, docs, prompts committed to the repo, commit messages or PR text. Examples use invented values (`Person A`, `10.000.000`). Real data stays in the sheet and the local, gitignored test workbook (§14.3).

## Tier 1: Fix numbers that are wrong today

| #   | Item                      | Scope                                                                                                                                                                                                                                                                                                                                                                                                                          | Sheet change                                                                                                                   |
| --- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------ |
| R1  | **Money lent** (D30)      | Per-person ledger derived from `Lending / Repayment` transactions (payee = person): lent, repaid, outstanding, last activity, days outstanding; a negative balance means the user owes that person. Outstanding amounts owed to the user are **assets** in net worth. Lending flows leave income, expense, savings rate, spikes and needs-vs-wants. Replaces `/accounts/loans` and the AI loan review. ✅ Done (WM-LEND-1..5). | App: category nature editor on the Budget page (writes Categories B). The engine stops reading `'Loan / Dept'`; the tab stays. |
| R2  | **Fair month comparison** | "vs last month" compares month-to-date with the same days of last month, not with the whole month. Add a month-end pace for expense ("on pace for X by the last day"). ✅ Done (WM-DASH-2, WM-DASH-9).                                                                                                                                                                                                                         | None                                                                                                                           |

## Tier 2: Cleanup

Removes fakes and duplicates before new features build on them. Ships with the PR that replaces each part where possible.

| #   | Item                       | Scope                                                                                                                                                                                                                                         |
| --- | -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| C1  | Fake goal fields           | Drop the placeholder `Deadline: "TBD"`, `Milestones`, `History`, `StreakCount`, `ContributionType`, `LinkedAccountID`, the keyword type/emoji guess and the 50%-funded status (WM-GOAL-2). Ships with R3. ✅ Done (WM-GOAL-1..10).            |
| C2  | Retired account columns    | Stop reading Accounts C/D (per-account goal amount and progress), replaced by R3. Keep E (cleared balance) only if reconciliation is wanted; otherwise drop it from the UI. ✅ Done (C kept as the card credit limit, D no longer read).      |
| C3  | Loans page and loan review | Removed by R1, with the `loan-review` prompt keys and WM-LOAN-1..5. Debt-to-income (D16) keeps credit cards only. ✅ Done (WM-LEND-4, WM-LOAN-1..5 removed).                                                                                  |
| C4  | One allocation view        | Accounts (by group), Health (donut) and Investments (drift) each compute allocation differently. Keep one engine-computed allocation with targets and drift, shown under Wealth; Health links to it. ✅ Done (WM-INV-20, WM-ACC-8, WM-HEA-1). |
| C5  | One bank-rates list        | Bank savings rates appear on Investments and on Fmarket. Keep one, used by R6. ✅ Done (WM-INV-20: best rate on Investments, full list on Fmarket).                                                                                           |
| C6  | Preferences in the sheet   | Allocation targets live in browser storage only, so they differ per device. Move them to the app-managed `Preferences` tab (layout below), edited on Investments (targets) and Settings (goal pool settings). ✅ Done (WM-SET-6..9).          |
| C7  | Developer text in the UI   | Remove requirement IDs and implementation notes from user-facing copy; one disclaimer per page for rule-based market readings. **Done.**                                                                                                      |
| C8  | Dead requirements          | Close WM-UI-7 (D24), WM-INV-19 (D7) and the legacy WM-ING-2 row. **Done.**                                                                                                                                                                    |
| C9  | Consistent status colours  | One status token set (good / warning / serious / critical, always with icon and label) shared by every page; fix the Health projection colour that fails the dark-mode check. **Done.**                                                       |

## Tier 3: Advisor core

| #   | Item                        | Scope                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Sheet change                                                                                                                                                                                               |
| --- | --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| R3  | **Goal Plan** (D31, D32)    | Goals are claims on a pooled savings balance. Pool = asset accounts − a one-month spending buffer − card debt, excluding crypto (D32) and money lent. The pool fills goals in priority order, emergency fund first. Emergency fund target defaults to 6 × monthly needs (fixed + essential categories). Per goal: funded amount, required per month, pace from pool growth, projected finish date, on track / behind / ahead. Plan check: total required per month vs average monthly savings, with trade-offs (push a date, lower a target). Long-horizon goals show years to reach instead of a ring near 0%. ✅ Done (WM-GOAL-1..10). | App-managed `Goal Plan` tab (below): created on first use with an import of the old `Goals` rows; goals are created, edited, reordered, paused and finished in the app. The old `Goals` tab stays, unread. |
| R4  | **Bills and safe-to-spend** | Recurring items (salary, rent, utilities, subscriptions, card statements) suggested from history and confirmed by the user; a 30-day calendar; safe-to-spend = ready money (no crypto, no accounts marked slow to withdraw) − bills and card payments due before the next wage (D33), in tiers: committed (with fixed costs), essentials, free to spend (D36). ✅ Done (WM-BILL-1..8, WM-SET-10).                                                                                                                                                                                                                                        | App-managed `Recurring` tab (below): created on the first confirmed suggestion; confirm, edit, pause and delete in the app                                                                                 |
| R5  | **Home: Needs you**         | Dashboard leads with an action inbox (drafts, card due, budget over pace, money lent overdue, goal behind, rebalance drift, uncategorized), then safe-to-spend, net worth trend and savings on pace, then a short briefing. ✅ Done (WM-HOME-1..4): one ranked `GET /api/home/needs-you`; rule-backed items come from the live rules, the `Recommendations` tab keeps the user's snooze or dismissal.                                                                                                                                                                                                                                    | None                                                                                                                                                                                                       |
| R6  | **Idle cash check**         | Ready money beyond a day-to-day float, the bills due and the emergency fund (covered first by emergency accounts), compared only with instant-access deposits (no-term or rolling 1-month, D34); estimated extra interest per year and a ladder idea. Advice about rate, never about splitting goal money into accounts (D31). ✅ Done (WM-CASH-1..6).                                                                                                                                                                                                                                                                                   | None                                                                                                                                                                                                       |
| R7  | **Recommendations list**    | One list replaces the separate AI review cards: action, VND impact, evidence link, status (open / done / dismissed / snoozed). Weekly, monthly and yearly reviews add to it; the next review checks outcomes. ✅ Done (WM-REC-1..7).                                                                                                                                                                                                                                                                                                                                                                                                     | App-managed `Recommendations` tab                                                                                                                                                                          |
| R8  | **Monthly close**           | A short checklist: approve drafts, categorize, budget vs actual, surplus to goals, chase money lent, review recommendations. Each item is derived from data and links to its page; the snapshot is written from the checklist. ✅ Done (WM-CLOSE-1..4, WM-NW-1, WM-NW-2).                                                                                                                                                                                                                                                                                                                                                                | App-managed `Snapshots` tab (below): created on the first written snapshot; one row per month                                                                                                              |

### `Goal Plan` tab layout

| Column | Field           | Notes                                                                     |
| ------ | --------------- | ------------------------------------------------------------------------- |
| A      | Name            | Required                                                                  |
| B      | Target          | VND. Blank on the emergency fund row means "computed" (6 × monthly needs) |
| C      | Target date     | `yyyy-mm`; blank = no deadline (pace only)                                |
| D      | Priority        | 1 is filled first                                                         |
| E      | Starting amount | Optional; money already set aside outside the pool                        |
| F      | Status          | `active` / `paused` / `done`                                              |
| G      | Note            | Free text                                                                 |

### `Recurring` tab layout

| Column | Field     | Notes                                                     |
| ------ | --------- | --------------------------------------------------------- |
| A      | Name      | Required                                                  |
| B      | Kind      | `income` / `bill` / `subscription` / `insurance` / `card` |
| C      | Amount    | VND; blank on `card` rows means "statement balance"       |
| D      | Frequency | `monthly` / `quarterly` / `yearly`                        |
| E      | Day       | Day of month                                              |
| F      | Month     | Month of year for `quarterly` (first) and `yearly` items  |
| G      | Account   | Account name                                              |
| H      | Category  | Category name                                             |
| I      | Match     | Payee text that marks a transaction as this item          |
| J      | Status    | `active` / `paused`                                       |
| K      | Note      | Free text                                                 |

### `Preferences` tab layout

| Column | Field | Notes                                                                                                                                                                                              |
| ------ | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A      | Key   | `allocation.target.cash`, `allocation.target.funds`, `allocation.target.crypto`, `goal.emergencyMonths` (default 6), `goal.spendingBufferMonths` (default 1), `safeToSpend.notReadyAccounts` (D33) |
| B      | Value | Number; for `safeToSpend.notReadyAccounts` a JSON list of account names, e.g. `["Bank A","Broker B"]` (blank = none)                                                                               |
| C      | Note  | Free text                                                                                                                                                                                          |

### `Snapshots` tab layout

Written by the engine (one row per month, rewritten in place); months before the first snapshot may be typed by hand with only Month and Net worth.

| Column | Field                  | Notes                                                    |
| ------ | ---------------------- | -------------------------------------------------------- |
| A      | Month                  | `yyyy-mm`; required, one row per month                   |
| B      | As of                  | `yyyy-mm-dd`; the month end, or the write date mid-month |
| C      | Net worth              | VND; required                                            |
| D      | Assets                 | VND                                                      |
| E      | Liabilities            | VND                                                      |
| F      | Money lent             | VND owed to the user                                     |
| G      | Cash                   | VND; cash and deposits allocation line                   |
| H      | Funds                  | VND; fund certificates at the write date's prices        |
| I      | Crypto                 | VND; crypto at the write date's prices                   |
| J      | Income                 | VND; the month to the as-of date                         |
| K      | Expense                | VND; the month to the as-of date                         |
| L      | Savings                | VND; income minus expense                                |
| M      | Savings rate           | Percent                                                  |
| N      | Health score           | 0-100 (WM-HEA-1)                                         |
| O      | Savings pillar         | Score out of 30                                          |
| P      | Emergency pillar       | Score out of 30                                          |
| Q      | Debt pillar            | Score out of 20                                          |
| R      | Diversification pillar | Score out of 20                                          |
| S      | Emergency months       | Months of expenses the liquid assets cover               |
| T      | Written                | `yyyy-mm-dd` the row was written                         |
| U      | Note                   | The user's; never overwritten                            |

Dates typed by hand may arrive as date serials (Sheets parses `yyyy-mm`); readers accept both.

## Tier 4: Markets for the user's money

Market pages today read like a VN-equity trading terminal. Re-aim them at what the user holds.

| #   | Item                    | Scope                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| --- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M1  | **My exposure first**   | Investments opens with the user's allocation (cash, funds, crypto, money lent), the volatile share, and today's change in VND; markets follow. ✅ Done (WM-INV-24, WM-INV-25, WM-INV-28).                                                                                                                                                                                                                                                                                        |
| M2  | **Crypto view**         | Prices for held coins, crypto share of net worth, drawdown scenario ("if crypto falls 30%, net worth falls X%"). Crypto is long-term holding and trading (D32). ✅ Done (WM-INV-26: held coins with their 24h move, share of net worth, fixed 30% and 50% falls).                                                                                                                                                                                                                |
| M3  | **Fund vs deposit**     | Cost basis and gain per fund holding since purchase, compared with a bank deposit over the same period. Cost comes from the fund tab's History column (dates and units) priced at the Fmarket NAV history; no sheet change. **Done** (WM-INV-21..23): NAV history from the market data server, the deposit at today's best published rate as a labelled proxy; holdings whose History does not reconcile with Total Unit show their value only.                                  |
| M4  | **Gold and USD hedges** | Gold and USD trend from the existing sources, framed as protection options, not trading charts. ✅ Done (WM-INV-27: SJC gold and USD/VND daily closes from the market data server, 3M and 1Y against the cash share).                                                                                                                                                                                                                                                            |
| M5  | **Research area**       | Market Pulse, Ticker Analysis, News Scanner and Think Tank move under Research. Think Tank defaults to the user's holdings and feeds R7. ✅ Done (WM-UI-14, WM-INV-29, WM-REC-8): `/research` lists the tools (Fmarket joins them); URLs are unchanged. Think Tank analyses the held symbols unless symbols are given, and its actions phase records 0-3 recommendations as source `think-tank`; the focus and the contract are engine code, so the sheet prompts are unchanged. |

## Tier 5: Polish

| #   | Item                     | Scope                                                                                                                                                                                                                                                                                                         |
| --- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P1  | **Five-area navigation** | Home / Money (month, cash flow, transactions, bills) / Wealth (net worth, accounts, investments, money lent) / Plan (goals, health, projections) / Research, plus Assistant. ✅ Done (WM-UI-10..13): grouped desktop sidebar, a phone area bar with the current area's pages as tabs; the route map is below. |
| P2  | **Page hierarchy**       | Headline answer, evidence, detail on every page (see Principles). ✅ Done (WM-HOME-5) with one `PageHeadline` on Dashboard, Budget, Bills, Cash flow, Goals and Recommendations; Investments, Money lent, Health, Accounts, Transactions and Close keep their own headline for now.                           |
| P3  | **Net worth history**    | 1Y and all-time views with milestones; needs the monthly snapshot from TODO §2. ✅ Done (WM-NW-3..7) on `/health`: months without a snapshot are estimated and marked; savings vs market split where both neighbouring months are known.                                                                      |

### Navigation map (P1)

Every page route belongs to exactly one area; URLs did not change, so a page may sit under another area's path. The current page is the most specific entry, so `/accounts/credit-cards` lights up Money, not Accounts.

| Area      | Pages (in nav order)                                                                                                                                                               | Why there                                                                                           |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Home      | `/`                                                                                                                                                                                | "What needs me?"                                                                                    |
| Money     | Transactions `/transactions` (+ Drafts `/transactions/drafts`, Smart Capture `/transactions/capture`), Budget, Cash Flow, Bills, Credit Cards (`/accounts/credit-cards`, `/:name`) | Day-to-day money: the month's spending, the capture inbox, bills, and card statements and cashback. |
| Wealth    | Accounts `/accounts` (net worth), Investments `/investments`, Money lent `/accounts/lent`                                                                                          | What the user owns and is owed.                                                                     |
| Plan      | Goals `/accounts/goals`, Health `/health` (net worth history and projections), Recommendations `/recommendations`, Monthly close `/close`                                          | "Am I on track?" and the advice loop.                                                               |
| Research  | Overview `/research`, Market Pulse, Ticker Analysis, News Scanner, Think Tank (`/investments/...`), Fmarket `/investments/funds` (+ `/:code`)                                      | Markets and ideas; Fmarket is fund and deposit-rate research, so it joins the M5 tools.             |
| Assistant | `/chat`                                                                                                                                                                            | The chat, also on every page as a floating button.                                                  |
| Settings  | `/settings`                                                                                                                                                                        | Utility entry: the sidebar foot and the phone menu, not the area bar.                               |

On phones a bottom bar holds Home, Money, Wealth, Plan, Research and Assistant (icon and label); the current area's pages are a tab strip under the header, and the menu sheet still lists every page and Settings.

## Implementation route

Fourteen PRs in seven waves (wave 0 included). **All waves are complete** as of wave 7 (`navigation`); new work goes to [TODO.md](./TODO.md). PRs inside a wave touch different files and can run in parallel, each in its own worktree (`../monorepo-worktrees/wealth-management-<topic>`); a wave starts when the previous one is on `main`. Every PR lands with `rebase-merge`.

### Definition of done (every PR)

- TDD + BDD: domain tests first, then service, handler and UI tests; `go vet ./...`, `go test ./...`, `svelte-check`, `vitest` green. ESLint runs on the changed files through lint-staged at commit (it blocks on errors); a full `eslint .` run is not needed.
- Live check against the local test workbook only (§14.3), never the production sheet.
- Requirement IDs and statuses added to REQUIREMENTS.md; this roadmap marks the item done.
- API routes get OpenAPI metadata and an MCP tool (parity test).
- Privacy scan of the diff: no names, balances, account notes or real transactions; fixtures use invented values.
- Tab layouts are documented in REQUIREMENTS.md §2 in the same PR. App-managed tabs are created on first use; writes are covered by local-workbook tests. The engine never restructures or deletes existing tabs.

### Waves

| Wave | PR (branch topic) | Items         | Depends on | Main code areas                                                                                                                            | App writes to the sheet                               |
| ---- | ----------------- | ------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------- |
| 0    | `roadmap`         | This document | -          | `docs/wealth-management/`                                                                                                                  | -                                                     |
| 1    | `money-lent`      | R1, C3        | 0          | `domain/` lending ledger, balance sheet (WM-RULE-1/2), cash flow (WM-RULE-3), health DTI (D16); `/accounts/lent` page; loan review removed | Categories B (nature editor)                          |
| 1    | `month-pace`      | R2            | 0          | `domain/dashboard.go` same-days comparison and month-end pace; dashboard stats                                                             | -                                                     |
| 1    | `ui-tidy`         | C7, C8, C9    | 0          | shared status tokens, page copy, REQUIREMENTS rows                                                                                         | -                                                     |
| 2    | `goal-plan`       | R3, C1, C2    | 1          | `domain/` goal pool and waterfall, needs from category natures; Goal Plan reader; `/accounts/goals` rebuilt; goal review prompt facts      | `Goal Plan` (created on first use, imports old goals) |
| 2    | `allocation`      | C4, C5, C6    | 1          | one engine allocation with targets and drift; one bank-rates source; preferences tab reader                                                | `Preferences` (created on first save)                 |
| 3    | `bills`           | R4            | 2          | recurring detection from history, `Recurring` reader, 30-day calendar, safe-to-spend                                                       | `Recurring` (created on first confirm)                |
| 3    | `fund-vs-deposit` | M3            | 2          | cost basis per fund holding; comparison with a deposit at published rates                                                                  | -                                                     |
| 4    | `idle-cash`       | R6            | 3          | idle cash = ready money beyond the float, bills and the emergency fund vs best instant-access rate (D34)                                   | -                                                     |
| 4    | `my-exposure`     | M1, M2, M4    | 2          | Investments overview: exposure, crypto view and drawdown scenario, gold/USD hedges                                                         | -                                                     |
| 5    | `recommendations` | R7            | 4          | recommendation store (engine-written tab via `CreateSheet` / row writes), status API, reviews feed it, AI cards replaced                   | `Recommendations`                                     |
| 6    | `home-needs-you`  | R5, P2        | 5          | dashboard rebuilt around the action inbox; headline-first layout across pages                                                              | -                                                     |
| 6    | `monthly-close`   | R8, P3        | 5          | checklist page; month-close snapshot (engine-written `Snapshots` tab, TODO §2); net worth 1Y / all-time history                            | `Snapshots`                                           |
| 7    | `navigation`      | P1, M5        | 6          | five-area navigation; Research area; Think Tank defaults to holdings and feeds R7 (done)                                                   | -                                                     |

### Risks

- **Sheet edits by hand**: every new tab has a documented layout and tolerant parsing (blank and header rows skipped, unknown values rejected with a clear message).
- **Writes to the live sheet**: only through app actions on app-managed tabs and the Categories nature column; development and tests use the local workbook only.
- **Pool accuracy**: the goal pool depends on account balances being current; goals show the "as of" date of the last sync.
