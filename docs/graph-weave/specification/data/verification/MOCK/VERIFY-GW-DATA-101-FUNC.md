# VERIFY-GW-DATA-101-FUNC: Versioned key used by mock app

> **Linked Task** : GW-DATA-101 — `docs/graph-weave/specification/data/tasks/MOCK/GW-DATA-101.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 2. Scope Compliance

- The mock app must use the versioned key shape.

## 3. Type-Specific Criteria

| #       | Criterion      | Expected                                | Actual | Status   |
| ------- | -------------- | --------------------------------------- | ------ | -------- |
| FUNC-01 | Key shape used | Mock app reads/writes the versioned key | Pass   | complete |

## 4. Documentation Check

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 5. Final Decision

| Decision        | Condition                             |
| --------------- | ------------------------------------- |
| Pass            | Mock app uses the versioned key shape |
| Needs Revision  | Lookup still uses the wrong key shape |
| Fail + Rollback | Behavior conflicts with cache spec    |

**Decision:** Pass

## 6. Evidence

- **Test File**: `tests/test_versioned_cache_keys.py`
- **Key Test**: `test_versioned_key_format_generated_correctly` verifies internal key format is `skills:level1:tenant_001:calculate:1.0.0`
- **Implementation**: `apps/graph-weave/src/adapters/cache.py` lines 26-39 - `set_versioned()` and `get_versioned()` methods construct and use versioned keys
- **Result**: 4/4 tests passing, including key format and isolation
- **All Tests**: 53/53 passing
