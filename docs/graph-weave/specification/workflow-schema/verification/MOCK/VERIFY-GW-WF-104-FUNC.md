# VERIFY-GW-WF-104: Get workflow endpoint

> **Linked Task** : GW-WF-104
> **Verification Type** : FUNC
> **Phase ID** : MOCK

---

## Functional Test Cases

1. ✅ GET /workflows/{workflow_id} with valid tenant_id returns 200 OK
   - Expected: Full workflow object including definition

2. ✅ Workflow not found returns 404 Not Found
   - Input: GET /workflows/nonexistent:v1.0.0?tenant_id=hedge_fund_research_desk
   - Expected: 404 with error message

3. ✅ Missing tenant_id query parameter returns 400 Bad Request
   - Input: GET /workflows/{workflow_id} without tenant_id
   - Expected: 400

4. ✅ Tenant scoping: different tenants cannot access each other's workflows
   - Setup: Create workflow in tenant A
   - Action: GET /workflows/{workflow_id}?tenant_id=tenant_B
   - Expected: 404 Not Found

---

## Acceptance Criteria

✅ All 4 functional tests pass
✅ Response includes definition field (differs from list endpoint)
✅ created_at and updated_at timestamps present
✅ Error responses follow standard format
