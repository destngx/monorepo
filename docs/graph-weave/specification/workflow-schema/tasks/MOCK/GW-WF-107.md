# GW-WF-107: Delete workflow endpoint

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Low
- **Status** : Pending
- **Estimated Effort**: S
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Workflow management API
- **Feature** : DELETE /workflows/{workflow_id} endpoint for removing workflows
- **Rationale** : Workflows should be deletable when no longer needed

---

### Input

- **Data / Files** : `[[../../WORKFLOW_MANAGEMENT_API.md]]`, `[[../../../architecture/system-architecture.md]]`
- **Dependencies** : GW-WF-103 (create endpoint + MockWorkflowStore)
- **External Systems**: None (mock phase)

---

### Scope

- **In Scope** :
  - Implement DELETE /workflows/{workflow_id} endpoint in FastAPI
  - Accept workflow_id path parameter and tenant_id query parameter
  - Delete workflow from MockWorkflowStore
  - Return 204 No Content on successful deletion
  - Return 404 Not Found if workflow does not exist for tenant
  - Return 400 Bad Request if tenant_id is missing or invalid

- **Out of Scope**:
  - Soft delete or archiving (use PUT with status=archived instead)
  - Cascade deletion of related execution state
  - Audit logging of deletions

- **Max Increment**: Single working DELETE endpoint with tenant scoping

---

### Approach

1. Add DELETE /workflows/{workflow_id} endpoint to src/main.py
2. Accept tenant_id as required query parameter
3. Query MockWorkflowStore.delete(tenant_id, workflow_id)
4. Return 404 if not found
5. Return 204 No Content on success
6. Write functional tests covering: successful deletion, workflow not found, missing tenant_id

**Files to Modify/Create**:

- `apps/graph-weave/src/main.py` — Add DELETE /workflows/{workflow_id} endpoint with `tags=["Workflows"]`
- `apps/graph-weave/src/adapters/workflow.py` — Add delete method to MockWorkflowStore
- `apps/graph-weave/tests/test_workflow_management.py` — Test DELETE /workflows/{workflow_id} (minimum 3 tests)

---

### Implementation Notes

- **API Tag**: Endpoint should be decorated with `tags=["Workflows"]` for Swagger UI organization per `[[../plan/api-organization.md]]`
- **OpenAPI Schema**: Swagger UI will group this endpoint under the "Workflows" tag with other workflow CRUD operations

---

### Expected Output

- **Deliverable** : Working DELETE /workflows/{workflow_id} endpoint
- **Format** : FastAPI endpoint returning 204 No Content
- **Example Success** :
  - Request: DELETE /workflows/quant-research:v3.0.0?tenant_id=hedge_fund_research_desk
  - Response: 204 No Content

- **Example Error** :
  - Workflow not found: 404 Not Found
  - Missing tenant_id: 400 Bad request

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-107-FUNC.md]]

---

### References

[[../../WORKFLOW_MANAGEMENT_API.md]] - Full API specification
[[../../../architecture/system-architecture.md]] - Tenant scoping requirements
