# VERIFY-GW-WF-103-SCHEMA: Create workflow endpoint schema

> **Linked Task** : GW-WF-103
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK

---

## Data Contracts

### CreateWorkflowRequest Model

```python
{
    tenant_id: str                    # 1-128 chars, required
    workflow_id: str                  # format: "name:vX.Y.Z", required
    name: str                         # 1-256 chars, required
    version: str                      # X.Y.Z semantic versioning, required
    description: Optional[str]        # max 1000 chars
    tags: Optional[List[str]]         # list of 1-100 char strings
    owner: str                        # 1-128 chars, required
    definition: Dict[str, Any]        # valid workflow JSON per WORKFLOW_JSON_SPEC
}
```

### WorkflowResponse Model

```python
{
    workflow_id: str
    tenant_id: str
    name: str
    version: str
    description: Optional[str]
    tags: Optional[List[str]]
    owner: str
    status: str                       # "active" by default
    created_at: str                   # ISO 8601
    updated_at: str                   # ISO 8601
}
```

### MockWorkflowStore Structure

```python
{
    "hedge_fund_research_desk": {
        "quant-research:v3.0.0": {
            "workflow_id": "quant-research:v3.0.0",
            "tenant_id": "hedge_fund_research_desk",
            "name": "Quantitative Research Pipeline",
            ...
            "definition": { /* full workflow JSON */ }
        }
    }
}
```

---

## Validation Rules

| Field       | Rule                                          | Passes |
| ----------- | --------------------------------------------- | ------ |
| workflow_id | Format: "name:vX.Y.Z", matches version field  | ✅     |
| tenant_id   | 1-128 chars, alphanumeric + underscore/hyphen | ✅     |
| version     | Semantic versioning (must be X.Y.Z)           | ✅     |
| definition  | Valid per WORKFLOW_JSON_SPEC                  | ✅     |
| status      | Default to "active"                           | ✅     |

---

## API Contract

**Endpoint**: POST /workflows
**Request**: CreateWorkflowRequest (JSON body)
**Response (201)**:

```json
{
  "workflow_id": "quant-research:v3.0.0",
  "tenant_id": "hedge_fund_research_desk",
  "name": "Quantitative Research Pipeline",
  "version": "3.0.0",
  "status": "active",
  "created_at": "2026-04-09T14:30:00Z",
  "updated_at": "2026-04-09T14:30:00Z"
}
```

**Response (409 Conflict)**:

```json
{
  "error": "Conflict",
  "message": "Workflow already exists",
  "status_code": 409,
  "details": { "workflow_id": "quant-research:v3.0.0" }
}
```

---

## OpenAPI Schema

**Tag Assignment**: Endpoint should be registered with `tags=["Workflows"]` per `[[../plan/api-organization.md]]`

**Generated OpenAPI Entry**:

```json
{
  "post": {
    "tags": ["Workflows"],
    "summary": "Create Workflow",
    "operationId": "create_workflow",
    "requestBody": {
      "required": true,
      "content": {
        "application/json": {
          "schema": { "$ref": "#/components/schemas/WorkflowCreate" }
        }
      }
    },
    "responses": {
      "201": {
        "description": "Workflow created successfully",
        "content": {
          "application/json": {
            "schema": { "$ref": "#/components/schemas/WorkflowResponse" }
          }
        }
      },
      "400": { "description": "Invalid request body" },
      "409": { "description": "Workflow already exists" },
      "422": { "description": "Validation error" }
    }
  }
}
```

**Validation Checklist**:

- ✅ Endpoint includes `tags=["Workflows"]` decorator
- ✅ Swagger UI displays endpoint under "Workflows" group
- ✅ OpenAPI schema includes tag metadata
- ✅ All status codes (201, 400, 409, 422) documented

---

## Status

✅ Schema defined and ready for implementation
