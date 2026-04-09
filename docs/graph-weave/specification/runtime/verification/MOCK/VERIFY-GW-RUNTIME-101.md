# VERIFY-GW-RUNTIME-101-FUNC-DOC: Retry and rerun identity rule

> **Linked Task** : GW-RUNTIME-101 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-101.md`
> **Verification Types** : FUNC, DOC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/runtime/request-lifecycle.md`
- `docs/graph-weave/specification/runtime/plan/request-lifecycle-and-streaming.md`

## 2. Scope Compliance

- The docs must state that reruns can keep the same `run_id`.
- The docs must state that reruns may use a new `thread_id`.

## 3. Type-Specific Criteria

### FUNC

| #       | Criterion                     | Expected                                         | Actual | Status      |
| ------- | ----------------------------- | ------------------------------------------------ | ------ | ----------- |
| FUNC-01 | Stable public identity        | `run_id` remains the public record across reruns |        | in progress |
| FUNC-02 | New internal attempt identity | `thread_id` can change for a rerun or replay     |        | in progress |

### DOC

| #      | Criterion                 | Expected                                                  | Actual | Status      |
| ------ | ------------------------- | --------------------------------------------------------- | ------ | ----------- |
| DOC-01 | Identity split described  | The docs explain public vs internal identity clearly      |        | in progress |
| DOC-02 | Replay behavior traceable | Retry/replay wording matches the lifecycle reference docs |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 5. Final Decision

| Decision        | Condition                                      |
| --------------- | ---------------------------------------------- |
| Pass            | Rerun rules are explicit and aligned           |
| Needs Revision  | The split is still ambiguous                   |
| Fail + Rollback | The rule conflicts with runtime identity model |

**Decision:** Pending
