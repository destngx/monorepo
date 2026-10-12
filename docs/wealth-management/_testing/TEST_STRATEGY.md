# Specification: Quality Assurance & Testing Strategy

**Author**: Product Owner / QA Lead  
**Intended Audience**: Development (Frontend/Backend) & QA Engineering  
**Status**: Mandatory Operational Standard  
**Keywords**: TDD, BDD, Edge Cases, Financial Integrity, Resilience

---

## 1. Quality Mission Statement

To ensure **100% Financial Accuracy** and **System Resilience**. We treat every calculation as a legal record and every service failure as a predictable event that must be handled gracefully.

---

## 2. The Multi-Model Testing Philosophy

### 2.1 BDD (Behavior-Driven Development)

Functional specs are transformed into `Gherkin`-style scenarios before implementation begins.

- **Objective**: Align engineering output with the **BA User Stories**.
- **Template**:
  - `GIVEN` [a specific financial state, e.g., Net Worth of 1B VND]
  - `WHEN` [a user action occurs, e.g., Records a 5M VND expense]
  - `THEN` [the expected outcome, e.g., Budget ceiling updates and Net Worth reflects the change]

### 2.2 TDD (Test-Driven Development)

Engineers must follow the **Red-Green-Refactor** lifecycle for all core logic (especially the engine's `domain/` and `service/` packages in `apps/wealth-management`).

1.  **RED**: Write a failing test for a new utility (e.g., `calculateFreedomScore`).
2.  **GREEN**: Implement the minimal logic to make the test pass.
3.  **REFACTOR**: Clean the code while ensuring the test suite remains green.

---

## 3. Mandatory "Edge Case" Matrix

**Requirement**: No feature is considered "Done" without a dedicated test suite for boundary conditions and service failures.

| Category             | Requirement            | Example Test Case                                                |
| :------------------- | :--------------------- | :--------------------------------------------------------------- |
| **Financial Bound**  | Handle Zero / Extremes | Calculate % growth when starting balance is `0`.                 |
| **Network Failure**  | Graceful Error UI      | API returns `500` or `429` (Rate Limit).                         |
| **Data Integrity**   | Malformed Input        | Google Sheet contains a string in an "Amount" column.            |
| **AI Hallucination** | Logic Validation       | AI suggests a rebalance that exceeds total portfolio value.      |
| **Large Numbers**    | Precision Loss         | Correctly sum VND values in the billions without `float` errors. |

---

## 4. Test Layers & Tooling (Nx Monorepo)

The current layer definitions and targets are in [REQUIREMENTS.md §14](../REQUIREMENTS.md#14-test-plan). In short:

- **Unit Layer**: table-driven Go tests (`pnpx nx run wealth-management:test`) for mappers, business rules and market math; Vitest (`pnpx nx run wealth-management:test-ui`) for UI logic.
  - _SLA_: 95%+ coverage for financial formulas.
- **Contract Layer**: sheet adapter against recorded, anonymised Sheets API responses; handlers and MCP tools against fakes.
- **Live Layer**: engine against a disposable copy of the real sheet, never the production sheet (REQUIREMENTS.md §14.3).
- **E2E Layer (Playwright)**: Testing critical "Wealth Journeys" in the UI against the engine wired to the test copy.
- **AI Regression**: Using fixed seeds to ensure AI briefings remain consistent across deployments.

---

## 5. Deployment Quality Gates (CI/CD)

1.  **Lint Check**: Zero warnings allowed.
2.  **Test Suite**: All Unit and Integration tests must pass in the Nx CI pipeline.
3.  **Audit Check**: Automatic check for sensitive API keys in the commit history (Leaked Secret Detection).

---

## 6. Success Metrics for Engineering

- **Math Parity**: 100% success rate in matching "Expected Ledger" outcomes.
- **Resilience**: 0% application crashes during simulated "Offline Mode" (Edge Case).
- **Regression**: 0% "Fix-Break-Fix" cycles; every bug report must result in a new failing test case before being fixed.
