# GW-WF-103: Create workflow endpoint

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Workflow management API
- **Feature** : POST /workflows endpoint for creating new workflows with validation
- **Rationale** : MOCK phase requires complete workflow CRUD; creation is the entry point to the workflow registry

---

### Input

- **Data / Files** : `[[../../WORKFLOW_MANAGEMENT_API.md]]`, `[[../../WORKFLOW_JSON_SPEC.md]]`, `[[../../../architecture/system-architecture.md]]`
- **Dependencies** : GW-WF-102 (workflow schema multi-label support)
- **External Systems**: None (mock phase)

---

### Scope

- **In Scope** :
  - Implement POST /workflows endpoint in FastAPI
  - Accept CreateWorkflowRequest with full metadata (tenant_id, workflow_id, name, version, description, tags, owner, definition)
  - Validate all input fields per WORKFLOW_MANAGEMENT_API.md specification
  - Validate workflow_id format (must be "{name}:{version}")
  - Validate workflow definition against WORKFLOW_JSON_SPEC
  - Store workflow in MockWorkflowStore
  - Return 201 Created with workflow metadata
  - Prevent duplicate workflow_id per tenant (return 409 Conflict)

- **Out of Scope**:
  - Redis persistence (mock storage only)
  - Workflow execution or runtime validation
  - Authentication/authorization
  - Rate limiting

- **Max Increment**: Single working POST endpoint with schema validation and in-memory storage

---

### Approach

1. Create Workflow, WorkflowCreate, and WorkflowResponse Pydantic models in src/models.py
2. Implement validation for workflow_id format and semantic versioning
3. Create or extend MockWorkflowStore class in src/adapters/workflow.py
4. Add POST /workflows endpoint to src/main.py
5. Return standardized error responses (400, 409, 422)
6. Write functional tests covering: success case, duplicate workflow, validation failures, invalid JSON schema

**Files to Modify/Create**:

- `apps/graph-weave/src/models.py` — Add Workflow, WorkflowCreate, WorkflowResponse models with validators
- `apps/graph-weave/src/adapters/workflow.py` — Create MockWorkflowStore class with create method
- `apps/graph-weave/src/main.py` — Add POST /workflows endpoint with `tags=["Workflows"]`
- `apps/graph-weave/tests/test_workflow_management.py` — Test POST /workflows (minimum 6 tests)

---

### Implementation Notes

- **API Tag**: Endpoint should be decorated with `tags=["Workflows"]` for Swagger UI organization per `[[../plan/api-organization.md]]`
- **OpenAPI Schema**: Swagger UI will group this endpoint under the "Workflows" tag with other workflow CRUD operations

---

### Expected Output

- **Deliverable** : Working POST /workflows endpoint
- **Format** : FastAPI endpoint returning JSON
- **Example Success** :
  - Request: POST /workflows with CreateWorkflowRequest
  - Response: 201 with WorkflowResponse metadata (excludes full definition)

- **Example Error** :
  - Duplicate workflow: 409 Conflict
  - Invalid schema: 422 Validation error
  - Missing required field: 400 Bad request

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-103-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-WF-103-SCHEMA.md]]
[[../../verification/MOCK/VERIFY-GW-WF-103-DOC.md]]

---

### References

[[../../WORKFLOW_MANAGEMENT_API.md]] - Full API specification
[[../../WORKFLOW_JSON_SPEC.md]] - Workflow schema validation rules
[[../../../architecture/system-architecture.md]] - Workflow registry architecture
