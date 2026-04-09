# VERIFY-GW-WF-103-DOC: Create workflow endpoint documentation

> **Linked Task** : GW-WF-103
> **Verification Type** : DOC
> **Phase ID** : MOCK

---

## OpenAPI Documentation

### Endpoint Summary

- **Path**: POST /workflows
- **Summary**: Create new workflow with full metadata
- **Description**: Register a new workflow in the workflow registry for the specified tenant. Validates workflow JSON schema and prevents duplicate workflow IDs per tenant.

### Request Body Documentation

```json
{
  "description": "Full workflow creation request",
  "required": ["tenant_id", "workflow_id", "name", "version", "owner", "definition"],
  "properties": {
    "tenant_id": {
      "type": "string",
      "description": "Tenant identifier (e.g., hedge_fund_research_desk)",
      "minLength": 1,
      "maxLength": 128,
      "example": "hedge_fund_research_desk"
    },
    "workflow_id": {
      "type": "string",
      "description": "Workflow identifier in format 'name:vX.Y.Z'",
      "pattern": "^[a-z][a-z0-9_-]*:v[0-9]+\\.[0-9]+\\.[0-9]+$",
      "example": "quant-research:v3.0.0"
    },
    "name": {
      "type": "string",
      "description": "Human-readable workflow name",
      "minLength": 1,
      "maxLength": 256,
      "example": "Quantitative Research Pipeline"
    },
    "definition": {
      "type": "object",
      "description": "Complete workflow JSON definition per WORKFLOW_JSON_SPEC"
    }
  }
}
```

### Response Documentation

**201 Created**: WorkflowResponse with metadata

```json
{
  "description": "Workflow successfully created",
  "properties": {
    "workflow_id": { "type": "string", "example": "quant-research:v3.0.0" },
    "tenant_id": { "type": "string", "example": "hedge_fund_research_desk" },
    "name": { "type": "string" },
    "status": { "type": "string", "enum": ["active", "archived", "draft"] },
    "created_at": { "type": "string", "format": "date-time" },
    "updated_at": { "type": "string", "format": "date-time" }
  }
}
```

**400 Bad Request**:

- Missing required fields
- Invalid field values

**409 Conflict**:

- Workflow already exists for tenant

**422 Unprocessable Entity**:

- Invalid workflow JSON schema
- workflow_id format mismatch with version

---

## Error Documentation

All errors follow standard format:

```json
{
  "error": "ErrorType",
  "message": "Human-readable message",
  "status_code": 400,
  "details": {}
}
```

---

## Migration Notes

- Workflow registry is separate from execution state
- WorkflowCreate validation includes schema validation
- Timestamps are immutable after creation
- definition field is stored but not returned in response body

---

## Status

✅ Documentation contract defined and ready for implementation
