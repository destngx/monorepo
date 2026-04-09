# VERIFY-GW-RUNTIME-101-DOC: Stable identity rule documented

> **Linked Task** : GW-RUNTIME-101 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-101.md`
> **Verification Type** : DOC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 2. Scope Compliance

- The lifecycle spec must describe the stable public identity rule.

## 3. Type-Specific Criteria

| #      | Criterion              | Expected                           | Actual | Status   |
| ------ | ---------------------- | ---------------------------------- | ------ | -------- |
| DOC-01 | Identity wording trace | The spec explains `run_id` clearly | Pass   | complete |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md` - Section 4.1 "Why Two IDs Exist" clearly documents:
  - `run_id` is the stable public record of the submission
  - `thread_id` is the live execution handle
  - Reruns can create new `thread_id` while keeping same `run_id`
  - This separation enables reruns and audit trails

## 5. Final Decision

| Decision        | Condition                        |
| --------------- | -------------------------------- |
| Pass            | Identity wording remains clear   |
| Needs Revision  | Identity wording is ambiguous    |
| Fail + Rollback | Wording conflicts with lifecycle |

**Decision:** Pass

## 6. Evidence

- **Spec Document**: `docs/graph-weave/specification/runtime/request-lifecycle.md` lines 46-52
- **Key Wording**:
  - "run_id is the stable public record of the submission"
  - "If a run is retried or replayed later, the same run_id can keep the user-facing history while a new thread_id can represent the new live attempt"
  - "This separation makes reruns, recovery, and audit trails easier to understand without changing the client-facing job identity"
- **Implementation Alignment**: Model accepts optional `run_id` in ExecuteRequest, endpoint logic preserves it if provided
