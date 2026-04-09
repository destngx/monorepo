# GW-WF-104: Get workflow endpoint

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Low
- **Status** : Pending
- **Estimated Effort**: S
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Workflow management API
- **Feature** : GET /workflows/{workflow_id} endpoint for retrieving specific workflows
- **Rationale** : Clients must be able to retrieve workflow metadata and definition by ID

---

### Input

- **Data / Files** : `[[../../WORKFLOW_MANAGEMENT_API.md]]`, `[[../../../architecture/system-architecture.md]]`
- **Dependencies** : GW-WF-103 (create endpoint + MockWorkflowStore)
- **External Systems**: None (mock phase)

---

### Scope

- **In Scope** :
  - Implement GET /workflows/{workflow_id} endpoint in FastAPI
  - Accept workflow_id path parameter and tenant_id query parameter
  - Retrieve workflow from MockWorkflowStore
  - Return full workflow object (metadata + definition) on success
  - Return 404 Not Found if workflow does not exist for tenant
  - Return 400 Bad Request if tenant_id is missing or invalid

- **Out of Scope**:
  - Filtering or search operations (covered in list endpoint)
  - Execution or runtime information
  - Authentication/authorization

- **Max Increment**: Single working GET endpoint with tenant scoping and error handling

---

### Approach

1. Add GET /workflows/{workflow_id} endpoint to src/main.py
2. Accept tenant_id as required query parameter
3. Query MockWorkflowStore.get(tenant_id, workflow_id)
4. Return 404 if not found, 400 if tenant_id invalid or missing
5. Return full WorkflowDetailResponse (includes definition)
6. Write functional tests covering: success case, workflow not found, missing tenant_id

**Files to Modify/Create**:

- `apps/graph-weave/src/main.py` — Add GET /workflows/{workflow_id} endpoint with `tags=["Workflows"]`
- `apps/graph-weave/src/models.py` — Add WorkflowDetailResponse model (includes full definition)
- `apps/graph-weave/tests/test_workflow_management.py` — Test GET /workflows/{workflow_id} (minimum 4 tests)

---

### Implementation Notes

- **API Tag**: Endpoint should be decorated with `tags=["Workflows"]` for Swagger UI organization per `[[../plan/api-organization.md]]`
- **OpenAPI Schema**: Swagger UI will group this endpoint under the "Workflows" tag with other workflow CRUD operations

---

### Expected Output

- **Deliverable** : Working GET /workflows/{workflow_id} endpoint
- **Format** : FastAPI endpoint returning JSON
- **Example Success** :
  - Request: GET /workflows/quant-research:v3.0.0?tenant_id=hedge_fund_research_desk
  - Response: 200 with full workflow object including definition

- **Example Error** :
  - Workflow not found: 404 Not Found
  - Missing tenant_id: 400 Bad request

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-104-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-WF-104-SCHEMA.md]]

---

### References

[[../../WORKFLOW_MANAGEMENT_API.md]] - Full API specification
[[../../../architecture/system-architecture.md]] - Tenant scoping requirements
