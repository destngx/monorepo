# Agile Migration Management (Go-Svelte)

**Project**: Wealth Management Platform (High-Efficiency Transition)  
**Role**: PM & Dev Lead Orchestration  
**Objective**: To migrate the legacy Next.js architecture to a high-performance **Golang Backend** and **Svelte Frontend** while preserving the current **Python Data Server**.

---

## 1. Sprint Roadmap

The migration is prioritized into four functional increments:

1.  **[Sprint 1: System Foundation](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_1/README.md)**
    - _Focus_: Scaffold **Engine** and **Dashboard**, OAuth Google Sheets client, Redis Connectivity.
    - _Legacy_: Mark Next.js as [LEGACY].

2.  **[Sprint 2: Core Data Engine](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_2/README.md)**
    - _Focus_: Sharded Transactions, Budget logic, Account aggregates in Go.
    - _Frontend_: Initial Svelte Ledger & Commands.

3.  **[Sprint 3: Wealth Intelligence](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_3/README.md)**
    - _Focus_: Port GPT-4.1 Orchestrator to Go, Chat streaming (NDJSON), Daily Briefings.

4.  **[Sprint 4: Market Alpha & Final Migration](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/SPRINT_4/README.md)**
    - _Focus_: VNStock/Binance client, Investment Terminal (**Dashboard**), Final Redirect.

---

## 2. Agile Operational Flow

- **Task Format**: Every task follows the **Jira Standard** (ID, Title, Status, Priority, Description, AC) separated by `---`.
- **Review Protocol**: Each Sprint has a mandatory **Definition of Done (DoD)** that must be fulfilled before starting the next sprint.
- **Reporting**: Weekly updates will be appended to localized `progress.md` files.

### Sprint Doc Structure

- Each sprint is now a folder: `SPRINT_1/`, `SPRINT_2/`, `SPRINT_3/`, `SPRINT_4/`.
- Each sprint `README.md` contains only sprint-level details (timeline, dependencies, DoD, references).
- Each task is stored as an individual markdown file in its sprint folder (`WM-xxx.md`).

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
- **Domain Naming Convention (Go)**: In `apps/wealth-management-engine/domain`, use singular snake_case file names mapped to business concepts (e.g., `accounts.go`, `health.go`) and concise PascalCase type names without redundant suffixes (e.g., `Accounts`, not `AccountsSheet`).
- **Dashboard Architecture (Svelte)**: Use **Domain-Driven Design (DDD)** with **Feature-Sliced Design (FSD)** influence. Organize code into `features/`, `entities/`, and `shared/` layers.
- **NX Workspace Strategy**: Leverage **Nx Monorepo** structures. Use tags (`type:app`, `type:lib`, `scope:wm`) and ensure all shared Go/Svelte logic is extracted into unified core libraries (**`libs/wm-core`** for Go, **`libs/wm-core-ui`** for Svelte) before app consumption.

---

## 5. Standing Conventions (Permanent)

This section is the single source of truth for implementation conventions across all wealth-management sprints. Once defined here, these rules are assumed for every task and do not need to be repeated.

### 5.1 Execution and Quality Workflow

- Follow **TDD + BDD** by default for backend implementation tasks:
  - write/adjust tests first (or in the same change set),
  - implement to satisfy behavior scenarios,
  - keep acceptance criteria testable.
- Prefer real provider behavior tests for AI-provider adapters when feasible; avoid mock-only validation for provider-critical logic.
- `mcp` server is **always enabled** when starting `wealth-management-engine`.

### 5.2 Environment and Runtime Rules

- Use project-local temporary storage under repo `tmp/` (not root `/tmp`) for Go build/test caches and related task scripts.
- Use `.env.local` for runtime config loading only; **never** read or expose secret values in docs, code comments, logs, or task writeups.

### 5.3 Naming and Structure (Ports/Adapters/Domain)

- Maintain strict **Hexagonal Architecture**:
  - `port/` contains interfaces/contracts,
  - `adapter/` contains implementations/integrations.
- Use generalized integration names over vendor-specific names in domain-facing paths:
  - cache adapters under `adapter/cache/` (not `adapter/redis/`),
  - config adapter under `adapter/config/` (not `adapter/env/`),
  - database adapters under `adapter/db/...` with Google Sheets as one backend,
  - market data provider adapters under `adapter/market/...` with `vnstock` as one provider implementation.
- Prefer business domain naming:
  - use `accounts` (not `accounts_sheet`) in domain concepts and service naming.
- Keep account domain models in dedicated files and avoid coupling names to storage implementation details.

### 5.4 Project Tooling

- Keep `go.work` and `go.work.sum` at monorepo root for cross-project workspace consistency.
- `project.json` for `wealth-management-engine` must include:
  - `test` target runnable via `bunx nx run wealth-management-engine:test`,
  - formatting command support.
- Husky/lint-staged should include Go formatting for staged `.go` files.

### 5.5 Market Provider Pattern (Capability-Based Architecture)

- All market providers (Fmarket, Vnstock, future providers) implement the **same `port.MarketProvider` interface**.
- The interface defines **capabilities** (e.g., `GetTicker`, `GetExchangeRate`, `GetFundNavHistory`), not actions.
- Capabilities accept **generic type parameters** (e.g., `tickerType TickerType`, `from, to Currency`).
- **Routing is config-driven**: The service layer looks up provider priority per capability+type from injected routing config.
- **No provider-specific handlers**: HTTP endpoints are generic and capability-based, not action-based or provider-specific.
- **Fallback chains**: Each capability maps to a provider chain (e.g., [Fmarket, Vnstock]); service tries each in order on failure.
- **Unified caching**: Cache keys are capability-level (e.g., `market:getTicker:equity:ACB`), not per-provider.
- **Provider unsupported types**: If a provider doesn't support a capability+type, it returns `ErrUnsupported`; service tries fallback.
- **Dependency injection**: All providers and routing config injected at startup in `main.go`, enabling easy testing and multi-environment config.
- See [\_technical/Market_Provider_Capabilities.md](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_technical/Market_Provider_Capabilities.md) for detailed architecture.

### 5.6 Scope Decisions Already Agreed

- Skip legacy-marking tasks unless explicitly reintroduced.
- Skip benchmark and stress-test requirements unless explicitly requested.
- Treat `vnstock` as a single provider implementation under a provider abstraction, not as the only market integration model.

### 5.7 API Contract and Documentation Policy

- Do not expose temporary `/api/test/*` endpoints in the engine.
- Use real, domain-facing API routes (e.g., `/api/ai/stream`, `/api/ai/json`, `/api/ai/tools`).
- OpenAPI spec must be generated from code/runtime route registration, not maintained as hardcoded YAML.
- Swagger grouping/tagging must follow API domain boundary: use the second path segment under `/api/{domain}/...` (for example, `/api/market/*` -> `market` tag).
- Documentation infrastructure endpoints must not appear as business API operations in Swagger paths:
  - exclude `/api/docs`
  - exclude `/api/openapi.json`

### 5.8 Test Placement Convention

- Integration/e2e tests for HTTP APIs must live in the same package area as the API adapter being tested (for engine APIs: `adapter/fiber`), not in a separate top-level `integration/` package.
- Keep live API tests env-gated (`RUN_LIVE_API_TESTS=true`) and runnable through Nx target `test-e2e`, which executes all `TestE2E*` tests.
- E2E tests should validate response meaning (schema/content) rather than status-code-only checks.
- During live e2e runs, runtime/provider failures must fail the suite (no skip-based pass masking).
- `test-e2e` must return non-zero exit status when any e2e test fails.

---

**Project Lead**: Antigravity  
**Last Updated**: March 31, 2026
