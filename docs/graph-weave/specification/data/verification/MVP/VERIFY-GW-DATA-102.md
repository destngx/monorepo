# VERIFY-GW-DATA-102-SCHEMA: Cache versus checkpoint boundary

> **Linked Task** : GW-DATA-102 — `docs/graph-weave/specification/data/tasks/MVP/GW-DATA-102.md`
> **Verification Type** : SCHEMA
> **Phase ID** : MVP
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/data/compiled-graph-and-checkpoint-storage.md`
- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 2. Scope Compliance

- The docs must keep compiled graphs and checkpoints distinct.
- The docs must keep live state separate from cache state.

## 3. Type-Specific Criteria

| #         | Criterion           | Expected                                               | Actual | Status      |
| --------- | ------------------- | ------------------------------------------------------ | ------ | ----------- |
| SCHEMA-01 | Cache boundary      | Compiled graphs are clearly described as cache         |        | in progress |
| SCHEMA-02 | Checkpoint boundary | Checkpoints are clearly described as live resume state |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/data/compiled-graph-and-checkpoint-storage.md`

## 5. Final Decision

| Decision        | Condition                                    |
| --------------- | -------------------------------------------- |
| Pass            | Cache and checkpoint boundaries are explicit |
| Needs Revision  | Boundary language is ambiguous               |
| Fail + Rollback | Live state and cache are conflated           |

**Decision:** Pending
