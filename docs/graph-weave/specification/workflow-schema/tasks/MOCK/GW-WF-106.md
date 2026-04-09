# GW-WF-106: Update workflow endpoint

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Workflow management API
- **Feature** : PUT /workflows/{workflow_id} endpoint for updating workflow metadata and definition
- **Rationale** : Workflows must be updatable to reflect changes in requirements, tags, ownership, or definition

---

### Input

- **Data / Files** : `[[../../WORKFLOW_MANAGEMENT_API.md]]`, `[[../../WORKFLOW_JSON_SPEC.md]]`, `[[../../../architecture/system-architecture.md]]`
- **Dependencies** : GW-WF-103 (create endpoint + MockWorkflowStore)
- **External Systems**: None (mock phase)

---

### Scope

- **In Scope** :
  - Implement PUT /workflows/{workflow_id} endpoint in FastAPI
  - Accept workflow_id path parameter and tenant_id query parameter
  - Accept UpdateWorkflowRequest with optional fields: name, description, tags, owner, status, definition
  - Require at least one field to be present in request body
  - Prevent updates to immutable fields (workflow_id, version, created_at)
  - Validate status must be one of: active, archived, draft
  - If definition is provided, validate it against WORKFLOW_JSON_SPEC
  - Update updated_at timestamp
  - Return 200 OK with updated workflow metadata
  - Return 404 Not Found if workflow does not exist
  - Return 400 Bad Request if required parameters missing or no fields provided
  - Return 422 Unprocessable Entity if validation fails

- **Out of Scope**:
  - Version bumping (version is immutable)
  - Workflow execution or state changes
  - Authentication/authorization

- **Max Increment**: Single working PUT endpoint with validation and immutable field protection

---

### Approach

1. Add PUT /workflows/{workflow_id} endpoint to src/main.py
2. Create UpdateWorkflowRequest model with all fields optional
3. Validate at least one field is provided
4. Query MockWorkflowStore.get(tenant_id, workflow_id)
5. Return 404 if not found
6. Validate all provided fields (status enum, definition schema)
7. Prevent mutation of workflow_id, version, created_at
8. Update updated_at to current timestamp
9. Call MockWorkflowStore.update()
10. Return updated WorkflowResponse
11. Write functional tests covering: successful update, no workflows, invalid tenant_id, validation failures, immutable field protection

**Files to Modify/Create**:

- `apps/graph-weave/src/main.py` — Add PUT /workflows/{workflow_id} endpoint with `tags=["Workflows"]`
- `apps/graph-weave/src/models.py` — Add UpdateWorkflowRequest model with validators
- `apps/graph-weave/src/adapters/workflow.py` — Add update method to MockWorkflowStore
- `apps/graph-weave/tests/test_workflow_management.py` — Test PUT /workflows/{workflow_id} (minimum 7 tests)

---

### Implementation Notes

- **API Tag**: Endpoint should be decorated with `tags=["Workflows"]` for Swagger UI organization per `[[../plan/api-organization.md]]`
- **OpenAPI Schema**: Swagger UI will group this endpoint under the "Workflows" tag with other workflow CRUD operations

---

### Expected Output

- **Deliverable** : Working PUT /workflows/{workflow_id} endpoint
- **Format** : FastAPI endpoint returning JSON
- **Example Success** :
  - Request: PUT /workflows/quant-research:v3.0.0?tenant_id=hedge_fund_research_desk with UpdateWorkflowRequest
  - Response: 200 with updated workflow metadata

- **Example Error** :
  - Workflow not found: 404 Not Found
  - Invalid status value: 422 Validation error
  - Attempted to update version: 400 Bad request (immutable field)

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-106-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-WF-106-SCHEMA.md]]

---

### References

[[../../WORKFLOW_MANAGEMENT_API.md]] - Full API specification including immutable field rules
[[../../WORKFLOW_JSON_SPEC.md]] - Workflow schema validation
[[../../../architecture/system-architecture.md]] - Workflow registry architecture
