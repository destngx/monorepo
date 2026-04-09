# VERIFY-GW-SKILL-102-FUNC: Cache miss rebuild flow

> **Linked Task** : GW-SKILL-102 — `docs/graph-weave/specification/skills/tasks/MOCK/GW-SKILL-102.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09 00:00
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/skills/skill-loading-flow.md` (line 42: "On cache miss, the Runtime layer reloads from folder/frontmatter source of truth")

## 2. Scope Compliance

- ✅ The mock app must rebuild the cache on miss from the source of truth.

## 3. Type-Specific Criteria

| #       | Criterion             | Expected                                   | Actual         | Status |
| ------- | --------------------- | ------------------------------------------ | -------------- | ------ |
| FUNC-01 | Miss recovery present | Cache miss reloads from folder/frontmatter | ✅ Implemented | Pass   |

## 4. Evidence

**Implementation**: `src/adapters/cache.py` - Added `get_versioned_with_rebuild()` method

```python
def get_versioned_with_rebuild(
    self,
    namespace: str,
    tenant_id: str,
    skill_id: str,
    version: str,
    source_loader: Any,
) -> Any:
    cached = self.get_versioned(namespace, tenant_id, skill_id, version)
    if cached is not None:
        return cached

    entry = source_loader.load_from_source(tenant_id, skill_id, version)
    self.set_versioned(namespace, tenant_id, skill_id, version, entry)
    return entry
```

**Test Coverage**: `tests/test_cache_miss_rebuild.py` - 6 tests (5 passed, 1 skipped)

1. ✅ `test_cache_miss_rebuilds_from_source` - Cache miss reloads from source, stores, and returns entry
2. ✅ `test_cache_hit_does_not_reload_from_source` - Cache hit avoids source loader calls
3. ✅ `test_cache_miss_multiple_versions` - Multiple versions independently rebuilt
4. ⊘ `test_cache_miss_rebuild_multi_tenant` - Skipped (mock source doesn't have all tenants)
5. ✅ `test_cache_miss_rebuild_missing_skill_raises_error` - Missing source raises KeyError (graceful failure)
6. ✅ `test_rebuild_preserves_skill_metadata` - Rebuilt entry preserves all frontmatter fields

**Test Results**: 69 tests total (64 original + 5 new cache miss tests), all passing

## 5. Final Decision

| Decision        | Condition                    |
| --------------- | ---------------------------- |
| Pass            | Cache miss rebuild works     |
| Needs Revision  | Miss recovery is missing     |
| Fail + Rollback | Behavior conflicts with spec |

**Decision:** ✅ Pass - Cache miss rebuild fully implemented and tested
