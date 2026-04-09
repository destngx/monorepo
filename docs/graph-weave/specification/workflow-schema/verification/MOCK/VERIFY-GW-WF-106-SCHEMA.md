# VERIFY-GW-WF-106-SCHEMA: Update workflow endpoint schema

> **Linked Task** : GW-WF-106
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK

---

## API Contract

**Endpoint**: PUT /workflows/{workflow_id}
**Query Parameters**: tenant_id (required)

**Request**: UpdateWorkflowRequest (all fields optional, at least one required)

```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "tags": ["new", "tags"],
  "owner": "new_owner",
  "status": "archived",
  "definition": {
    /* workflow JSON */
  }
}
```

**Response (200 OK)**: WorkflowResponse with updated fields

```json
{
  "workflow_id": "quant-research:v3.0.0",
  "version": "3.0.0",
  "status": "archived",
  "updated_at": "2026-04-09T15:45:00Z",
  ...
}
```

**Immutable Fields**: workflow_id, version, created_at

**Validation**:

- status must be one of: active, archived, draft
- definition (if provided) must be valid workflow JSON
- At least one field must be present

---

## Status

✅ Schema defined and ready for implementation
