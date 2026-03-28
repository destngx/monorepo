# Specification: System Resilience & Edge Cases

**Author**: Product Owner / SRE BA  
**Intended Audience**: Engineering Team (Architecture & QA)  
**Status**: Enforced for All Features  
**Keywords**: Offline Blocking, Rate Limiting, Data Integrity, Graceful Degradation

---

## 1. Business Objective

To ensure the user's financial confidence is never broken by localized platform failure. The system must always appear "Solid" and "Intentional," even during external API downtime.

---

## 2. Core Resilience Strategies

### 2.1 The "Blocking" Offline Safety (PO Requirement)

**Why**: Avoid confusing the user with "Offline Retry" queues or corrupted ledger writes.

- **Requirement**: Monitor `navigator.onLine`.
- **Implementation**: If offline, the UI MUST display a full-page **Blocking Modal** ("Connecting to Financial Grid...").
- **Constraint**: No data entry is permitted in the "Offline" state.

### 2.2 Google Sheets API Rate Limit (SLA Compliance)

**Why**: Google Sheets is a shared API resource with hard quotas.

- **Metric (Read/Write)**: 60 requests/min (User) / 300 requests/min (Project).
- **BA Strategy**: **Aggregation & Batching**.
- **Implementation**: Batch multiple ledger entries into a single `batchUpdate` call whenever possible.

---

## 3. Graceful Degradation (BA Hierarchy)

When a primary service (e.g., VNStock, Binance, Google Sheets) fails, the dev team must follow this hierarchy:

| Service Level    | Primary Source      | Secondary (Fallback)  | Tertiary (Last Resort)         |
| :--------------- | :------------------ | :-------------------- | :----------------------------- |
| **Ledger Data**  | Google Sheets Live  | Redis (Upstash) Cache | LocalStorage Cache             |
| **Market Depth** | VNStock/Binance API | Last Cached Price     | Manual Price Entry             |
| **AI Insights**  | GPT-4o (Primary)    | Claude (Secondary)    | Standard "Rules-based" Summary |

---

## 4. UI/UX Consistency Standards (NFR Sync)

**PO Expectation**: The "Stealth" (Privacy) level is a non-negotiable requirement.

- **Rule 1**: Masked by Default.
- **Rule 2**: Privacy Reset on client-side routing.
- **Rule 3**: Sensitive number fields must use the `MaskedBalance` utility component.

---

## 5. Engineering Success Metrics

- **Uptime Target**: 99.9% availability of the Next.js service layer.
- **Error Recovery**: 100% of catastrophic API failures must result in a user-friendly "Retry" or "Fallback" message.
- **Data Integrity**: Zero "Duplicate Transaction" reports in any 30-day window (enforced via `Ref#` and `ID` validation).
