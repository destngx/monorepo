# Wealth Management Platform: Product & Engineering Documentation

Welcome to the centralized documentation for the **Wealth Management Platform**. This workspace serves as the single source of truth for Business Analysts (BA), Product Owners (PO), and the Engineering team.

---

## 1. Documentation Map (The Pillars)

Our documentation is organized into four strategic pillars to ensure clarity from vision to implementation.

### 🏛️ [Product Vision & Requirements](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/user-stories/)

_High-level "What" and "Why" from the user's perspective._

- **[Core Ledger](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/user-stories/transactions.md)**: Manual-first entry with high-fidelity audit trails.
- **[Wealth Intelligence](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/user-stories/intelligence.md)**: AI-driven "CFO" insights and briefings.
- **[Portfolio Management](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/user-stories/accounts.md)**: Unified tracking of Bank, Crypto, and Securities.

### 📜 [Functional Specifications](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_specs/README.md)

_Detailed "How" for Feature Teams and UX/UI._

- **[Command Center](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_specs/1-Core-Command-Center/Portfolio_Home_and_Global_Patterns.md)**: Stealth Mode, Net Worth Pulse, and Executive Summaries.
- **[Investment Terminal](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_specs/4-Market-Alpha/Investment_and_Market_Terminal.md)**: Market Pulse (VNStock/Binance) and Ticker Alpha.

### 🛠️ [Technical Architecture](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_technical/README.md)

_Engineering blueprints for System Integrity._

- **[Data Engine](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_technical/1-Data-Engine/Architecture_and_Schema.md)**: Yearly Tab Sharding, Caching Hierarchies, and Schema Design.
- **[AI Orchestration](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_technical/2-AI-Systems/Orchestration_and_Tools.md)**: LLM Tool Selection, System Prompts, and Synthesis Logic.

### 🚀 [Operations & Quality](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_setup/README.md)

_Setup, Connectivity, and Resilience._

- **[Setup & Deployment](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_setup/1-Environment-Requirements/Infrastructure_and_Prerequisites.md)**: Infrastructure specs and interactive bootstrapping.
- **[Testing Strategy](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/_testing/TEST_STRATEGY.md)**: TDD/BDD standards and mandatory Edge Case Matrix.

---

## 2. Core Architectural Decisions (ADR)

### Why Google Sheets as a Database?

- **Universal Accessibility**: Financial data remains human-readable and editable without touching code.
- **Zero-Cost Persistence**: Leverages Google Cloud's highly available infrastructure with zero egress/storage costs for retail-scale ledgers.
- **Built-in Power**: Users can use standard Excel/Sheets formulas for custom reporting alongside the AI dashboard.

### Why Multi-Layer Caching (Redis + SWR)?

- **Performance**: We achieve **p95 < 300ms** latency for Net Worth aggregates by caching sharded sheet data in **Upstash Redis**.
- **Resilience**: The app remains functional in "Stale Mode" even if the Google Sheets API is temporarily rate-limited or offline.

---

## 3. Key Technical Standards

- **Stealth by Default**: All sensitive balances are masked (`••••••`) on every initial load and page navigation to ensure privacy in public spaces.
- **Yearly Sharding**: Transactions are automatically sharded into yearly tabs (e.g., `Transactions_2026`) to ensure O(1) read performance as the ledger grows.
- **AI-Human Hybrid**: The "Audit First" entry model (Email Parser → Review → Ledger) ensures data integrity remains 100% accurate.

---

**Last Refactor**: March 30, 2026  
**Document Custodians**: Product Owner & System Architect  
**Workspace**: Nx Monorepo (`apps/wealth-management`, `libs/wealth-management`)
