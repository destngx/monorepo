# Specification: Operations, Bootstrapping & Running Flow

**Author**: Product Owner / Operations Analyst  
**Intended Audience**: Dev / Setup Engineering  
**Status**: Mandatory Procedure  
**Keywords**: Installation Lifecycle, Bootstrapping, Verification, Modes

---

## 1. Objective

To ensure a consistent "First-Run" experience across local, staging, and production environments, with zero manual data entry required besides the initial bootstrapping phase.

---

## 2. The Integrated Installation Lifecycle

**BA Requirement**: The system must be brought online in three distinct phases.

### 2.1 Component Installation (Phase 1)

- **Objective**: Synchronize workspace-aware dependencies (Bun/Nx).
- **Outcome**: All local libraries and external node/python modules are available.

### 2.2 Interactive Bootstrapping (Phase 2)

The application requires two interactive "Onboarding" flows:

1.  **AI Provisioning**: Interactive CLI for selecting and authenticating the AI provider (OpenAI, Gemini, etc.).
2.  **OAuth Grant**: Interactive CLI for the Google Cloud OAuth2 "Handshake," resulting in the long-lived **Refresh Token**.

### 2.3 Health Verification (Phase 3)

- **Objective**: Automated integrity check of the connection string.
- **Outcome**: Confirmed read/write access to the `Transactions_YYYY` sheet and the `Accounts` reference tab.

---

## 3. Operational Runtime Modes

The behavior of the Wealth Manager changes based on the execution mode:

| Mode            | Objective         | Behavior                                                                   |
| :-------------- | :---------------- | :------------------------------------------------------------------------- |
| **Development** | Feature Iteration | Detailed AI traces; Localhost-only; Low Redis TTL (30s).                   |
| **Staging**     | QA & Integration  | Production-mirror accounts; Simulated transactions.                        |
| **Production**  | Actual Wealth     | High Redis TTL (5min); Minimal Trace (Security); Production-grade secrets. |

---

## 4. Connectivity Protocols (Procedures)

- **Manual Sheet Sync**: Use the "Force Refresh" button in the Dashboard for immediate reconcile (SWR invalidation).
- **Token Expiration Recovery**: If the system displays `invalid_grant`, the DevOps/User must re-run the Phase 2 OAuth Grant.

---

## 5. Deployment Overview (BA/PO Scope)

The Wealth Manager is an orchestrated monorepo:

- **Frontend/API Service Layer**: Next.js 16+ (Standalone Docker or Serverless).
- **Python Crawler Service**: Active `vnstock-server` for Asian market depth.
- **Cache Persistence Layer**: Redis-over-REST (Upstash).
- **Cold Storage**: Google Sheets persistence.

---

## 6. Success Metrics for Engineering

- **Onboarding Speed**: A first-time developer can bring the app alive in < 15 minutes.
- **Reliability**: 0% "Silent Connection Failures" (All must have UI feedback).
- **Portability**: The configuration must be environment-agnostic; no hardcoded URLs in the codebase.
