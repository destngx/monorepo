# VERIFY-GW-WF-107: Delete workflow endpoint

> **Linked Task** : GW-WF-107
> **Verification Type** : FUNC
> **Phase ID** : MOCK

---

## Functional Test Cases

1. ✅ DELETE /workflows/{workflow_id} with valid ID returns 204 No Content
   - Expected: 204, workflow removed from store

2. ✅ DELETE non-existent workflow returns 404 Not Found
   - Input: DELETE /workflows/nonexistent:v1.0.0?tenant_id=X
   - Expected: 404

3. ✅ GET workflow after DELETE returns 404
   - Setup: Create, then delete, then try to get
   - Expected: 404

4. ✅ Missing tenant_id returns 400 Bad Request

5. ✅ Tenant scoping: cannot delete another tenant's workflow
   - Setup: Create workflow in tenant A
   - Action: DELETE /workflows/{workflow_id}?tenant_id=tenant_B
   - Expected: 404 Not Found

---

## Acceptance Criteria

✅ All 5 functional tests pass
✅ Returns 204 No Content (not 200)
✅ Workflow truly removed from store
✅ Error responses follow standard format
