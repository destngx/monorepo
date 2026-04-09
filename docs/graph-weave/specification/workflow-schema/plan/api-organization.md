# API Organization & Swagger UI Semantic Grouping [MOCK]

## Overview

The GraphWeave API endpoints are organized into semantic tag groups in Swagger UI to improve API documentation clarity and navigation. This decision improves developer experience by grouping related operations by functional domain rather than displaying all endpoints in a flat "default" group.

## Design Decision

**Problem**: All 8 GraphWeave API endpoints were grouped under a single "default" tag in Swagger UI, making the API appear as an undifferentiated flat list despite having distinct functional domains (workflow execution, skill management, workflow CRUD).

**Solution**: Organize endpoints into three semantic tag groups:

- **Execution**: Workflow execution and status tracking
- **Skills**: Skill management and cache operations
- **Workflows**: Workflow CRUD operations (5 endpoints)

**Implementation**:

1. Added `tags=["GroupName"]` to each FastAPI endpoint decorator
2. Added `openapi_tags` metadata to FastAPI app initialization with group descriptions
3. Updated API specification docs to reference tag assignments

## Benefits

- **Improved Navigation**: Developers can quickly locate related endpoints by scrolling to the relevant tag group instead of scanning a flat list
- **Clear Functional Domains**: Tag names make the API's organizational structure self-documenting
- **Maintainability**: New endpoints can be assigned to appropriate groups, keeping organization consistent as API grows
- **Standards Compliance**: Follows OpenAPI 3.0 tag semantics as a best practice

## Tag Assignments

| Tag           | Endpoints                                                                                                                    | Purpose                                |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| **Execution** | POST /execute, GET /execute/{run_id}/status                                                                                  | Workflow execution and status tracking |
| **Skills**    | POST /invalidate                                                                                                             | Skill management and cache operations  |
| **Workflows** | POST /workflows, GET /workflows, GET /workflows/{workflow_id}, PUT /workflows/{workflow_id}, DELETE /workflows/{workflow_id} | Workflow CRUD operations               |

## Implementation Details

### Code Changes

- **File**: `apps/graph-weave/src/main.py`
- **Changes**:
  - Added 8 endpoint decorators with `tags=[...]` parameter
  - Added `openapi_tags` list to FastAPI app initialization with group metadata

### Documentation Changes

- **File**: `[[../WORKFLOW_MANAGEMENT_API.md]]`
- **Changes**:
  - Added "Tag: Workflows" descriptor to all 5 workflow endpoints (POST, GET, GET list, PUT, DELETE)
  - Added new "API Organization" section with semantic tag grouping reference table

### Testing

- Verified OpenAPI schema generation includes tag metadata
- Confirmed Swagger UI renders organized endpoint groups (no test regressions: 120/121 passing)

## Schema Impact

**Backward Compatibility**: Tag grouping is a presentation-layer change only. It does not affect API request/response schemas, HTTP semantics, or client behavior. Existing clients continue to function unchanged.

**OpenAPI Schema**: The generated OpenAPI schema now includes:

```json
{
  "tags": [
    {
      "name": "Execution",
      "description": "Workflow execution and status tracking - POST to execute, GET status by run_id"
    },
    {
      "name": "Skills",
      "description": "Skill management and cache operations - invalidate cached skill definitions"
    },
    {
      "name": "Workflows",
      "description": "Workflow CRUD operations - create, read, list, update, delete workflow definitions"
    }
  ]
}
```

## Future Considerations

- **Phase: MVP**: Monitor whether additional endpoints (e.g., execution history, audit logs) fit existing tag groups or require new groups
- **Phase: FULL**: Consider adding sub-tags or hierarchical grouping if endpoint count grows significantly (e.g., separate "Execution History" from core "Execution" group)
- **Documentation**: Consider providing endpoint reference cards grouped by tag for developer onboarding

## Related Documents

- `[[../WORKFLOW_MANAGEMENT_API.md]]` — API specification with tag assignments
- `[[../../delta-changes.md]]` — Implementation history and design rationale
- `[[../README.md]]` — Workflow schema overview
