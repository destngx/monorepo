# GW-WF-105: List workflows endpoint

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Workflow management API
- **Feature** : GET /workflows endpoint for listing workflows with optional filtering
- **Rationale** : Clients must be able to discover available workflows, optionally filtered by status, tags, or owner

---

### Input

- **Data / Files** : `[[../../WORKFLOW_MANAGEMENT_API.md]]`, `[[../../../architecture/system-architecture.md]]`
- **Dependencies** : GW-WF-103 (create endpoint + MockWorkflowStore)
- **External Systems**: None (mock phase)

---

### Scope

- **In Scope** :
  - Implement GET /workflows endpoint in FastAPI
  - Accept required tenant_id query parameter
  - Accept optional query parameters: status (active|archived|draft), tags (comma-separated), owner
  - Retrieve all workflows for tenant from MockWorkflowStore
  - Filter by status (exact match) if provided
  - Filter by tags (any tag match) if provided
  - Filter by owner (exact match) if provided
  - Return list of workflows WITHOUT full definition (summary only)
  - Include count of returned workflows
  - Return 200 OK with empty list if no workflows match

- **Out of Scope**:
  - Pagination or cursor-based listing
  - Full-text search
  - Workflow execution or runtime data

- **Max Increment**: Single working GET /workflows endpoint with multi-filter support

---

### Approach

1. Add GET /workflows endpoint to src/main.py
2. Accept query parameters: tenant_id (required), status, tags, owner (all optional)
3. Parse comma-separated tags if provided
4. Query MockWorkflowStore.list(tenant_id, filters)
5. Implement filtering logic: AND for status/owner, OR for tags
6. Return WorkflowListResponse with count
7. Write functional tests covering: all workflows, filtered by status, filtered by tags, filtered by owner, combined filters, no results

**Files to Modify/Create**:

- `apps/graph-weave/src/main.py` — Add GET /workflows endpoint with filter support and `tags=["Workflows"]`
- `apps/graph-weave/src/models.py` — Add WorkflowSummary and WorkflowListResponse models
- `apps/graph-weave/src/adapters/workflow.py` — Add list method to MockWorkflowStore
- `apps/graph-weave/tests/test_workflow_management.py` — Test GET /workflows (minimum 7 tests)

---

### Implementation Notes

- **API Tag**: Endpoint should be decorated with `tags=["Workflows"]` for Swagger UI organization per `[[../plan/api-organization.md]]`
- **OpenAPI Schema**: Swagger UI will group this endpoint under the "Workflows" tag with other workflow CRUD operations

---

### Expected Output

- **Deliverable** : Working GET /workflows endpoint with filtering
- **Format** : FastAPI endpoint returning JSON list
- **Example Success** :
  - Request: GET /workflows?tenant_id=hedge_fund_research_desk&status=active&tags=research
  - Response: 200 with list of matching workflows and count

- **Example Filter Behavior** :
  - No filters: Return all workflows for tenant
  - status=archived: Return only archived workflows
  - tags=research,earnings: Return workflows with either tag
  - owner=research_team: Return only workflows owned by research_team

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-105-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-WF-105-SCHEMA.md]]

---

### References

[[../../WORKFLOW_MANAGEMENT_API.md]] - Full API specification with filtering rules
[[../../../architecture/system-architecture.md]] - Tenant scoping requirements
