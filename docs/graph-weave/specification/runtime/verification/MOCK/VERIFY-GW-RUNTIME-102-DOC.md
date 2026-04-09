# VERIFY-GW-RUNTIME-102-DOC: Attempt identity rule documented

> **Linked Task** : GW-RUNTIME-102 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-102.md`
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

- The lifecycle spec must describe the internal attempt identity rule.

## 3. Type-Specific Criteria

| #      | Criterion             | Expected                              | Actual | Status   |
| ------ | --------------------- | ------------------------------------- | ------ | -------- |
| DOC-01 | Attempt wording trace | The spec explains `thread_id` clearly | Pass   | complete |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md` - Section 4.1 "Why Two IDs Exist" clearly documents:
  - `thread_id` is the live execution handle used by runtime state
  - A rerun creates a new `thread_id` while keeping the same `run_id`
  - When a rerun creates a new `thread_id`, the old thread is considered closed

## 5. Final Decision

| Decision        | Condition                        |
| --------------- | -------------------------------- |
| Pass            | Attempt wording remains clear    |
| Needs Revision  | Attempt wording is ambiguous     |
| Fail + Rollback | Wording conflicts with lifecycle |

**Decision:** Pass

## 6. Evidence

- **Spec Document**: `docs/graph-weave/specification/runtime/request-lifecycle.md` lines 46-52
- **Key Wording**:
  - "thread_id is the live execution handle used by runtime state"
  - "If a run is retried or replayed later, the same run_id can keep the user-facing history while a new thread_id can represent the new live attempt"
  - "When a rerun creates a new thread_id, the old thread is considered closed for runtime state purposes"
- **Implementation Alignment**: Endpoint always generates fresh `thread_id` on every execute call
