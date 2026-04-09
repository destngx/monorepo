# VERIFY-GW-RUNTIME-104-FUNC: Mock error response shape

> **Linked Task** : GW-RUNTIME-104 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-104.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/runtime/stagnation-detection-logic.md`
- `docs/graph-weave/specification/runtime/circuit-breaker.md`

## 2. Scope Compliance

- The mock app must return one standardized error response.

## 3. Type-Specific Criteria

| #       | Criterion           | Expected                        | Actual | Status |
| ------- | ------------------- | ------------------------------- | ------ | ------ |
| FUNC-01 | Error payload shape | The response is consistent JSON | ✓      | passed |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/stagnation-detection-logic.md` — Error handling follows safe exit patterns

## 5. Final Decision

| Decision        | Condition                    |
| --------------- | ---------------------------- |
| Pass            | Error shape is consistent    |
| Needs Revision  | Error response is unclear    |
| Fail + Rollback | Behavior conflicts with spec |

**Decision:** Pass

**Evidence:**

- Error handling integrated into FastAPI app
- Tests pass: 5/5 (invalid_payload, missing_field, is_json, status_endpoint, events_field)
- Standard FastAPI validation errors (422) for bad payloads
- Status endpoint returns consistent JSON with run_id and events
- Error responses are deterministic and consistent
