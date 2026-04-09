# VERIFY-GW-WF-103: Create workflow endpoint

> **Linked Task** : GW-WF-103 — `docs/graph-weave/specification/workflow-schema/tasks/MOCK/GW-WF-103.md`
> **Verification Types** : FUNC, SCHEMA, DOC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Status** : Pending (awaiting implementation)

---

## Functional Verification (FUNC)

### Test Cases

1. ✅ POST /workflows with valid CreateWorkflowRequest returns 201 Created
   - Input: tenant_id, workflow_id, name, version, description, tags, owner, definition
   - Expected: 201 with WorkflowResponse metadata (no definition in response)
   - Assertion: run_id present, created_at/updated_at set, status="active"

2. ✅ Duplicate workflow_id in same tenant returns 409 Conflict
   - Input: POST /workflows twice with same tenant_id and workflow_id
   - Expected: First succeeds (201), second fails (409)

3. ✅ Invalid workflow_id format returns 422
   - Input: workflow_id without version separator (e.g., "quant-research" instead of "quant-research:v3.0.0")
   - Expected: 422 Validation error

4. ✅ Invalid workflow JSON definition returns 422
   - Input: definition missing required nodes/edges fields
   - Expected: 422 with descriptive error message

5. ✅ Missing required field returns 400
   - Input: CreateWorkflowRequest without tenant_id
   - Expected: 400 Bad request

6. ✅ Workflow stored in MockWorkflowStore and retrievable
   - Input: POST /workflows then GET /workflows/{workflow_id}
   - Expected: Created workflow is returned with same metadata

---

## Schema Verification (SCHEMA)

| #         | Criterion                    | Expected                                       | Status     |
| --------- | ---------------------------- | ---------------------------------------------- | ---------- |
| SCHEMA-01 | WorkflowCreate model exists  | Pydantic model with all required fields        | ✅ Pending |
| SCHEMA-02 | Validators on WorkflowCreate | Fields validate per WORKFLOW_MANAGEMENT_API.md | ✅ Pending |
| SCHEMA-03 | WorkflowResponse model       | Returns metadata without definition            | ✅ Pending |
| SCHEMA-04 | MockWorkflowStore.create()   | Returns stored workflow object                 | ✅ Pending |

---

## Documentation Verification (DOC)

- OpenAPI docs include POST /workflows endpoint with request/response schemas
- Endpoint description clearly states purpose: "Create new workflow with validation"
- Request body schema shows all fields with examples
- Response 201 shows WorkflowResponse structure
- Error responses documented (400, 409, 422)

---

## Acceptance Criteria

- ✅ All 6 functional tests pass
- ✅ All schema validations pass
- ✅ OpenAPI docs generated correctly
- ✅ Error responses follow standard format
- ✅ created_at and updated_at timestamps are ISO 8601 format
