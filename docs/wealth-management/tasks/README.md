# Agile Migration Management (Go-Svelte)

**Project**: Wealth Management Platform (High-Efficiency Transition)  
**Role**: PM & Dev Lead Orchestration  
**Objective**: To migrate the legacy Next.js architecture to a high-performance **Golang Backend** and **Svelte Frontend** while preserving the current **Python Data Server**.

---

## 1. Sprint Roadmap

The migration is prioritized into four functional increments:

1.  **[Sprint 1: System Foundation](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_1.md)**
    - _Focus_: Scaffold **Engine** and **Dashboard**, OAuth Google Sheets client, Redis Connectivity.
    - _Legacy_: Mark Next.js as [LEGACY].

2.  **[Sprint 2: Core Data Engine](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_2.md)**
    - _Focus_: Sharded Transactions, Budget logic, Account aggregates in Go.
    - _Frontend_: Initial Svelte Ledger & Commands.

3.  **[Sprint 3: Wealth Intelligence](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_3.md)**
    - _Focus_: Port GPT-4.1 Orchestrator to Go, Chat streaming (NDJSON), Daily Briefings.

4.  **[Sprint 4: Market Alpha & Final Migration](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_4.md)**
    - _Focus_: VNStock/Binance client, Investment Terminal (**Dashboard**), Final Redirect.

---

## 2. Agile Operational Flow

- **Task Format**: Every task follows the **Jira Standard** (ID, Title, Status, Priority, Description, AC) separated by `---`.
- **Review Protocol**: Each Sprint has a mandatory **Definition of Done (DoD)** that must be fulfilled before starting the next sprint.
- **Reporting**: Weekly updates will be appended to localized `progress.md` files.

---

## 3. High-Level Requirements & Constraints

- **Legacy Persistence**: `apps/wealth-management` (Next.js) is maintained only for reference until Sprint 4 is complete.
- **Service Preservation**: `apps/vnstock-server` (Python) remains as-is; it is an external data provider for the new **Engine** core.
- **NFR Policy**: All new **Engine** and **Dashboard** features must meet the documented **Non-Functional Requirements** (p95 < 300ms, Stealth-by-Default).

---

## 4. Coding & Engineering Standards

**Requirement**: Every developer and automated agent MUST follow these standards for the new architecture:

- **Package Management**: Use `bun` and `bunx` exclusively for all commands (`bun install`, `bun nx ...`). NEVER use `npm`, `yarn`, or `pnpm`.
- **API Standard**: All new endpoints must have mandatory **MCP (Model Context Protocol)** support. The **Engine** must act as an MCP Server, exposing its core logic (Ledger, Market, Intelligence) as **MCP Tools** for LLM-based orchestration.
- **Software Principles**: Strictly follow **SOLID** and **DRY**.
- **Engine Architecture (Go)**: Use **Hexagonal Architecture** (Ports & Adapters). Business logic (Domain) must be isolated from external drivers (Sheets, Redis, AI) via clean ports.
- **Dashboard Architecture (Svelte)**: Use **Domain-Driven Design (DDD)** with **Feature-Sliced Design (FSD)** influence. Organize code into `features/`, `entities/`, and `shared/` layers.
- **NX Workspace Strategy**: Leverage **Nx Monorepo** structures. Use tags (`type:app`, `type:lib`, `scope:wm`) and ensure all shared Go/Svelte logic is extracted into unified core libraries (**`libs/wm-core`** for Go, **`libs/wm-core-ui`** for Svelte) before app consumption.

---

**Project Lead**: Antigravity  
**Last Updated**: March 30, 2026
