# VERIFY-GW-WF-105-SCHEMA: List workflows endpoint schema

> **Linked Task** : GW-WF-105
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK

---

## API Contract

**Endpoint**: GET /workflows
**Query Parameters**:

- tenant_id (required): string
- status (optional): active|archived|draft
- tags (optional): comma-separated string
- owner (optional): string

**Response (200 OK)**: WorkflowListResponse

```json
{
  "tenant_id": "hedge_fund_research_desk",
  "workflows": [
    {
      "workflow_id": "quant-research:v3.0.0",
      "name": "Quantitative Research Pipeline",
      "version": "3.0.0",
      "description": "...",
      "tags": ["research"],
      "owner": "research_team",
      "status": "active",
      "created_at": "2026-04-09T14:30:00Z",
      "updated_at": "2026-04-09T14:30:00Z"
    }
  ],
  "count": 1
}
```

**Note**: Workflows in list response do NOT include definition field

---

## Filter Semantics

- status: exact match (AND with other filters)
- owner: exact match (AND with other filters)
- tags: any tag match (OR within tags, AND with other filters)

---

## Status

✅ Schema defined and ready for implementation
