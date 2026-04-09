# VERIFY-GW-WF-104-SCHEMA: Get workflow endpoint schema

> **Linked Task** : GW-WF-104
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK

---

## API Contract

**Endpoint**: GET /workflows/{workflow_id}
**Query Parameters**: tenant_id (required)

**Response (200 OK)**: Full WorkflowDetailResponse

```json
{
  "workflow_id": "quant-research:v3.0.0",
  "tenant_id": "hedge_fund_research_desk",
  "name": "Quantitative Research Pipeline",
  "version": "3.0.0",
  "description": "...",
  "tags": ["research", "equities"],
  "owner": "research_team",
  "status": "active",
  "definition": {
    /* full workflow JSON */
  },
  "created_at": "2026-04-09T14:30:00Z",
  "updated_at": "2026-04-09T14:30:00Z"
}
```

**Response (404 Not Found)**: Error response
**Response (400 Bad Request)**: Missing tenant_id

---

## Status

✅ Schema defined and ready for implementation
