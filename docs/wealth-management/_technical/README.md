# Technical Product Specifications

**Role**: Product Owner & Business Analyst  
**Intended Recipient**: Wealth Management Development Team  
**Objective**: To provide clear, goal-oriented technical specifications for the implementation of the wealth-management platform.

---

## 1. Document Pillars

The documentation is organized into five pillars plus the market provider reference of system integrity:

### 1-Data-Engine

- **[Architecture & Schema Logic](./1-Data-Engine/Architecture_and_Schema.md)**
- _Focus_: Google Sheets as a DB, Data Flow, and Caching (the yearly sharding section is a legacy plan, not implemented).
- _Naming Standard_: See section **2.0 Domain Modeling & Naming Convention (Go Engine)** in Architecture & Schema Logic.

### 2-AI-Systems

- **[Orchestration & Tooling](./2-AI-Systems/Orchestration_and_Tools.md)**
- _Focus_: LLM interaction (OpenAI-compatible provider), Tool Registry, Contextual Memory, and Synthesis.

### 3-Financial-Logic

- **[Investment & Goal Engines](./3-Financial-Logic/Investment_and_Goal_Engines.md)**
- _Focus_: Wealth Trajectory (Freedom Score), Asset Allocation Drift (Rebalancing), and Normalization.

### Market-Provider-Capabilities

- **[Market Provider Capabilities](./Market_Provider_Capabilities.md)**
- _Focus_: Capability-based provider routing; section 0 lists the chains the engine actually uses.

### 4-Resilience

- **[Safety & Edge Cases](./4-Resilience/Safety_and_Edge_Cases.md)**
- _Focus_: Offline "Full-Stop" protocols, API Rate Limits, and Service Fallbacks.

### 5-Code-Structure

- **[Modularization & Testability Strategy](./5-Code-Structure/Modularization_and_Testability_Strategy.md)**
- _Focus_: file/function size budgets, vertical slice strategy, and strict e2e ownership/failure policy.

---

## 2. Engineering Standards

- **Tone**: All logic should follow "Defensive Engineering."
- **Integrity**: Prioritize data accuracy in the Transaction Ledger over UI speed.
- **Privacy**: The **"Stealth by Default"** rule is a mandatory global filter for all UI components.
- **API Documentation Contract**:
  - OpenAPI is generated dynamically from code route registration.
  - Swagger tags are grouped by `/api/{domain}` path segment.
  - Documentation endpoints (`/api/docs`, `/api/openapi.json`) are excluded from business API operations in the OpenAPI `paths` section.
- **AI Endpoint Namespace**:
  - Real AI APIs are exposed under `/api/ai/*`.
  - `/api/test/*` endpoints are not part of the production API contract.

---

**Last Refactor**: March 2026  
**Document Owner**: Product Owner
