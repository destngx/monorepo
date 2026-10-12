# Specification: Data Architecture & Schema Logic

**Author**: Product Owner / Business Analyst  
**Intended Audience**: Engineering Team (Backend & Data)  
**Status**: Approved for Implementation  
**Keywords**: Google Sheets Backend, Sharded Ledger, Caching Strategy

---

## 1. Business Objective

To provide a high-performance, cost-effective, and auditable financial database that remains human-readable (in Google Sheets) while supporting complex real-time queries through the Go engine (`apps/wealth-management`) and its SvelteKit UI.

> The authoritative sheet data contract (tabs, ranges, columns, read and write rules) is [REQUIREMENTS.md §2](../../REQUIREMENTS.md#2-data-contract-the-google-sheet). Where this document disagrees, REQUIREMENTS.md wins.

---

## 2. Core Entity Schema

### 2.0 Domain Modeling & Naming Convention (Go Engine)

- In `apps/wealth-management/domain`, each business entity must live in its own domain file.
- Use singular snake_case file names mapped to business concepts (for example: `accounts.go`, `transaction.go`, `budget.go`).
- Use concise PascalCase domain type names without redundant transport/storage suffixes (for example: `Accounts`, `Transaction`, not `AccountsSheet`).

### 2.1 Transaction Ledger (`Transactions_YYYY`)

_This is the core transactional database for the entire domain._

| Field        | Requirement | Business Rule                                                 |
| :----------- | :---------- | :------------------------------------------------------------ |
| **Account**  | Required    | Must match an existing entry in the `Accounts` tab.           |
| **Date**     | ISO Date    | Determines the **Sharding Destination** (which Yearly Tab).   |
| **Amount**   | Number      | Stored in native currency; UI converts via `CurrencyService`. |
| **Category** | Required    | Linked to the `Budget_YYYY` control targets.                  |
| **Status**   | Enum        | "Cleared" (C) vs. "Pending."                                  |

### 2.2 Account Metadata (`Accounts`)

_Defines the global state of user wealth entities._

- **Strategic Goal**: Total aggregation of liquid and illiquid wealth.
- **Key Fields**: `Name`, `Type` (Cash, Crypto, Investment), `Balance`, `Cleared Balance`.

---

## 3. High-Performance Sharding (Yearly)

> **Legacy plan, not implemented.** The sheet has a single `Transactions` tab and a per-year budget tab (`Budget_{year}`, configurable via `BUDGET_SHEET_TEMPLATE`); decision D5 in [REQUIREMENTS.md §16](../../REQUIREMENTS.md#16-open-decisions) keeps it that way. The `Transactions_YYYY` names in §2.1 and §7 follow this unimplemented plan.

**PO Requirement**: The system must NOT slow down over time as the user accumulates years of data (e.g., 5,000+ rows).

- **Implementation Strategy**: **Tab-based Sharding**.
- **Logic**: Each year gets its own tab (e.g., `Transactions_2025`, `Transactions_2026`).
- **Read Pattern**: "Current View" = Union of (Current Year + Last 30 days of Previous Year).
- **Write Pattern**: Dynamic routing by year-part of the input date.

---

## 4. Multi-Layer Caching (SLA: < 300ms)

**BA Goal**: Ensure the "Net Worth" pulse is visible within 300ms of page load.

> **Legacy design.** The two-level cache describes the legacy app. The engine has one optional cache (Upstash Redis REST, keys prefixed by `CACHE_KEY_PREFIX`), reads fall back to the sheet when it is absent, and TTLs follow WM-RULE-10 in [REQUIREMENTS.md §12](../../REQUIREMENTS.md#12-canonical-business-rules).

1.  **Level 1 (In-Memory)**: 30-second TTL for real-time totals.
2.  **Level 2 (Upstash Redis)**: 5-minute TTL for ledger lists.
3.  **Level 3 (Google Sheets)**: Source of truth. Refreshed only on **Cache Miss** or **User Force Refresh**.

---

## 5. Data Flow (BA Overview)

1.  **Ingest (Write)**: Validate Schema → Identify Shard → Append to Sheet → **Invalidate Redis Tag**.
2.  **Fetch (Read)**: Check Redis → (Hit: Return) → (Miss: Read Sheets → Parse/Normalize → Cache → Return).

---

## 6. Success Metrics for Engineering

- **Integrity**: Zero orphan transactions (all must have an account/category).
- **Scalability**: Sub-second dashboard renders with 10+ sharded tabs.
- **Accuracy**: 100% parity between Google Sheets totals and UI aggregates.

---

## 7. Hybrid Data Ingestion (Email Entry Staging)

> **Legacy, deferred.** The consolidated app does not read `EmailNotifications`; automatic ingestion is docs-only and Smart Capture replaces it ([REQUIREMENTS.md §6.1 and §6.2](../../REQUIREMENTS.md#61-deferred-automatic-balance-change-ingestion-docs-only)).

**Business Case**: Reduce manual friction for high-volume transactions (Bank alerts, E-wallets) while maintaining the "Auditor Control" human-in-the-loop requirement.

### 7.1 Staging Lifecycle (External Service → App)

1.  **Ingest (External)**: An external parser appends transaction details to the `EmailNotifications` sheet tab.
2.  **Filter (Application)**: App reads the sheet using `getPendingNotifications()` where `Status != "DONE"`.
3.  **Audit (Review Queue)**: User reviews the "Review & Approve" list in the dashboard.
4.  **Promote (Write)**: Clicking "Approve" moves data to `Transactions_YYYY` and updates the source row in `EmailNotifications` to `Status: "DONE"`.

### 7.2 Data Security & Expiration

- **Privacy**: Parsed content in the staging sheet must respect the same masking rules as the main ledger.
- **Auto-Cleanup**: Data older than 90 days in the staging sheet may be archived to prevent query degradation.
