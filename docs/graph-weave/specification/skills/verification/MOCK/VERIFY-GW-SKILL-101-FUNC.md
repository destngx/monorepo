# VERIFY-GW-SKILL-101-FUNC: Invalidation API shape

> **Linked Task** : GW-SKILL-101 — `docs/graph-weave/specification/skills/tasks/MOCK/GW-SKILL-101.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`

## 2. Scope Compliance

- The mock app must expose an invalidation API with tenant, skill, and reason.

## 3. Type-Specific Criteria

| #       | Criterion          | Expected                                   | Actual | Status   |
| ------- | ------------------ | ------------------------------------------ | ------ | -------- |
| FUNC-01 | API inputs present | Tenant, skill identifier, and reason exist | Pass   | complete |

## 4. Documentation Check

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`

## 5. Final Decision

| Decision        | Condition                        |
| --------------- | -------------------------------- |
| Pass            | Invalidation inputs are explicit |
| Needs Revision  | API inputs are incomplete        |
| Fail + Rollback | API conflicts with skill spec    |

**Decision:** Pass

## 6. Evidence

- **Test File**: `tests/test_skill_invalidation.py`
- **Key Test**: `test_invalidate_removes_cache_entry` verifies cache entry removal via invalidation
- **Implementation**: `apps/graph-weave/src/main.py` lines 68-88 - POST /invalidate endpoint accepts tenant_id, skill_id, reason
- **Response Model**: InvalidateResponse contains status, tenant_id, skill_id, reason
- **Result**: 6/6 tests passing, all API inputs validated and functional
- **All Tests**: 64/64 passing
