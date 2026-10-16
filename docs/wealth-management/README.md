# Wealth Management Documentation

Documentation for the wealth-management app: a Go engine (`apps/wealth-management`) that serves the JSON API, an MCP server and the SvelteKit UI (`apps/wealth-management/ui`, mounted at `/ui`). Google Sheets is the database. It is a single Nx project (`wealth-management`).

Start with [REQUIREMENTS.md](./REQUIREMENTS.md). It is the source of truth for scope, data contract, business rules and implementation status. Where any other document here disagrees with it, REQUIREMENTS.md wins.

---

## 1. Index

| Read this                                      | For                                                                                                                                    |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| [REQUIREMENTS.md](./REQUIREMENTS.md)           | Source of truth: sheet data contract, features, rules, status, open decisions.                                                         |
| [AGENTS.md](./AGENTS.md)                       | Code layout and conventions for the Go engine (hexagonal) and SvelteKit UI (feature-sliced).                                           |
| [ROADMAP.md](./ROADMAP.md)                     | Approved build order: wrong-number fixes, cleanup, advisor core (goals, bills, recommendations), markets for the user's money, polish. |
| [TODO.md](./TODO.md)                           | Parked ideas: capture automation and bank feed research, financial health improvements. Not approved scope.                            |
| [\_setup/](./_setup/README.md)                 | Setup: layout, environment variables and defaults, Nx targets, Google auth CLIs, live E2E tests.                                       |
| [\_technical/](./_technical/README.md)         | Technical specs: data engine, AI orchestration, financial logic, resilience, code structure, market providers.                         |
| [\_specs/](./_specs/README.md)                 | Functional specs per module: command center, ledger and budget, intelligence, market terminal.                                         |
| [\_testing/](./_testing/TEST_STRATEGY.md)      | Test strategy (details in REQUIREMENTS.md §14).                                                                                        |
| [user-stories/](./user-stories/)               | User stories per area (accounts, budget, transactions, investments, intelligence, and others).                                         |
| [narratives/](./narratives/)                   | Design rationale essays (market providers, USD rate, portfolio valuation, risk, AI advisor).                                           |
| [tasks/](./tasks/README.md)                    | Historical sprint plans. Status markers are stale; see REQUIREMENTS.md for current status.                                             |
| [\_api/](./_api/) and [features/](./features/) | Legacy docs for the removed Next.js app. The live API reference is the engine's `/api/openapi.json` (Swagger UI at `/api/docs`).       |
| `_archive/`                                    | Historical material, kept as is.                                                                                                       |

Market data routing as implemented is described in [\_technical/Market_Provider_Capabilities.md §0](./_technical/Market_Provider_Capabilities.md#0-current-implementation-authoritative).

---

## 2. Key decisions

- **Google Sheets as the database**: data stays human-readable and editable without code, at no storage cost. The sheet layout is fixed by the data contract in REQUIREMENTS.md §2 (single `Transactions` tab, per-year `Budget_{year}` tabs).
- **Optional cache**: one Upstash Redis REST cache in front of the sheet and market providers. Without it, reads go straight to the sheet.
- **Stealth by default**: sensitive balances are masked on load and on navigation.
- **Manual-first entry**: transactions are entered by hand or via Smart Capture with review before write. Automatic email ingestion is deferred (REQUIREMENTS.md §6.1).
