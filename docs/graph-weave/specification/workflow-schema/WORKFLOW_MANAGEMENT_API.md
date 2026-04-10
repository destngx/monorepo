# Workflow Management API Specification

## Overview

The Workflow Management API enables full lifecycle management of workflows: create, retrieve, list, update, and delete operations. All endpoints are tenant-scoped and return standardized responses.

**Phase**: MOCK, MVP

**API Organization**: Endpoints are organized under the **Workflows** tag in Swagger UI for semantic grouping with related workflow operations.

## Critical Requirement: Workflow Pre-Creation [MVP]

**Workflows must be created via POST /workflows BEFORE they can be executed.**

When a client calls `POST /execute` with a workflow_id that does not exist:

- POST /execute returns **404 Not Found**
- Message: `{"error": "NotFound", "message": "Workflow not found"}`

This enforces a clear separation of concerns:

1. **Deployment Phase**: Create workflows via POST /workflows
2. **Execution Phase**: Execute workflows via POST /execute

**Rationale:**

- Enables workflow versioning and rollback
- Clear audit trail of what workflows are deployed
- Prevents accidental execution of non-existent workflows
- Aligns with containerized deployment model: "deploy first, run second"

## Objective

Provide a complete CRUD interface for workflow registry that allows clients to:

- Define and register workflows with full metadata
- Query workflows by ID or list all for a tenant
- Update workflow metadata and definition
- Delete workflows when no longer needed
- Validate workflow schemas on create/update

## API Endpoints

### 1. Create Workflow

**Endpoint**: `POST /workflows`

**Tag**: `Workflows`

**Request Body**:

```json
{
  "tenant_id": "hedge_fund_research_desk",
  "workflow_id": "quant-research:v3.0.0",
  "name": "Quantitative Research Pipeline",
  "version": "3.0.0",
  "description": "Comprehensive quantitative research workflow for equities analysis",
  "tags": ["research", "equities", "earnings"],
  "owner": "research_team",
  "definition": {
    /* full workflow JSON from WORKFLOW_JSON_SPEC */
  }
}
```

**Response** (201 Created):

```json
{
  "workflow_id": "quant-research:v3.0.0",
  "tenant_id": "hedge_fund_research_desk",
  "name": "Quantitative Research Pipeline",
  "version": "3.0.0",
  "description": "Comprehensive quantitative research workflow for equities analysis",
  "tags": ["research", "equities", "earnings"],
  "owner": "research_team",
  "status": "active",
  "created_at": "2026-04-09T14:30:00Z",
  "updated_at": "2026-04-09T14:30:00Z"
}
```

**Validation**:

- `tenant_id`: Required, 1-128 chars, alphanumeric + underscore/hyphen
- `workflow_id`: Required, 1-128 chars, format: `name:vX.Y.Z` (must match version field)
- `name`: Required, 1-256 chars
- `version`: Required, semantic versioning (X.Y.Z)
- `description`: Optional, max 1000 chars
- `tags`: Optional, list of strings
- `owner`: Required, 1-128 chars
- `definition`: Required, valid workflow JSON per WORKFLOW_JSON_SPEC

**Error Cases**:

- 400: Invalid request body (missing required fields, validation errors)
- 409: Workflow already exists (duplicate workflow_id for tenant)
- 422: Validation error (schema invalid, version mismatch, etc.)

---

### 2. Get Workflow

**Endpoint**: `GET /workflows/{workflow_id}`

**Tag**: `Workflows`

**Description**: Retrieve a single workflow by its ID. Requires tenant scoping.

**Query Parameters**:

- `tenant_id` (required): Tenant identifier

**Response** (200 OK):

```json
{
  "workflow_id": "quant-research:v3.0.0",
  "tenant_id": "hedge_fund_research_desk",
  "name": "Quantitative Research Pipeline",
  "version": "3.0.0",
  "description": "Comprehensive quantitative research workflow for equities analysis",
  "tags": ["research", "equities", "earnings"],
  "owner": "research_team",
  "status": "active",
  "definition": {
    /* full workflow JSON */
  },
  "created_at": "2026-04-09T14:30:00Z",
  "updated_at": "2026-04-09T14:30:00Z"
}
```

**Error Cases**:

- 404: Workflow not found
- 400: Missing or invalid tenant_id

---

### 3. List Workflows

**Endpoint**: `GET /workflows`

**Tag**: `Workflows`

**Description**: List all workflows for a tenant with optional filtering by status, tags, and owner.

**Query Parameters**:

- `tenant_id` (required): Tenant identifier
- `status` (optional): Filter by status (active, archived, draft). Default: all statuses
- `tags` (optional): Filter by tags (comma-separated). Returns workflows matching ANY tag
- `owner` (optional): Filter by owner

**Response** (200 OK):

```json
{
  "tenant_id": "hedge_fund_research_desk",
  "workflows": [
    {
      "workflow_id": "quant-research:v3.0.0",
      "name": "Quantitative Research Pipeline",
      "version": "3.0.0",
      "description": "Comprehensive quantitative research workflow for equities analysis",
      "tags": ["research", "equities", "earnings"],
      "owner": "research_team",
      "status": "active",
      "created_at": "2026-04-09T14:30:00Z",
      "updated_at": "2026-04-09T14:30:00Z"
    }
  ],
  "count": 1
}
```

**Error Cases**:

- 400: Missing or invalid tenant_id

---

### 4. Update Workflow

**Endpoint**: `PUT /workflows/{workflow_id}`

**Tag**: `Workflows`

**Description**: Update workflow metadata, status, or definition. Immutable fields (workflow_id, version, created_at) cannot be changed.

**Query Parameters**:

- `tenant_id` (required): Tenant identifier

**Request Body** (all fields optional, must have at least one):

```json
{
  "name": "Updated Workflow Name",
  "description": "Updated description",
  "tags": ["new", "tags"],
  "owner": "new_owner",
  "status": "archived",
  "definition": {
    /* updated workflow JSON */
  }
}
```

**Response** (200 OK):

```json
{
  "workflow_id": "quant-research:v3.0.0",
  "tenant_id": "hedge_fund_research_desk",
  "name": "Updated Workflow Name",
  "version": "3.0.0",
  "description": "Updated description",
  "tags": ["new", "tags"],
  "owner": "new_owner",
  "status": "archived",
  "created_at": "2026-04-09T14:30:00Z",
  "updated_at": "2026-04-09T15:45:00Z"
}
```

**Validation**:

- Version cannot be changed (immutable)
- workflow_id cannot be changed (immutable)
- If `definition` is provided, must be valid workflow JSON
- `status` must be one of: active, archived, draft

**Error Cases**:

- 404: Workflow not found
- 400: Missing tenant_id or invalid request body
- 422: Validation error (invalid status, schema error, etc.)

---

### 5. Delete Workflow

**Endpoint**: `DELETE /workflows/{workflow_id}`

**Tag**: `Workflows`

**Description**: Permanently delete a workflow. Returns 204 No Content on success.

**Query Parameters**:

- `tenant_id` (required): Tenant identifier

**Response** (204 No Content)

**Error Cases**:

- 404: Workflow not found
- 400: Missing or invalid tenant_id

---

## Data Model

### Workflow Object

```python
{
    workflow_id: str              # Primary key: "{name}:{version}"
    tenant_id: str                # Tenant scoping
    name: str                     # Human-readable name
    version: str                  # Semantic versioning (X.Y.Z)
    description: str              # Optional markdown description
    tags: List[str]               # Searchable tags
    owner: str                    # Creator/owner identifier
    definition: Dict[str, Any]    # Complete workflow JSON per WORKFLOW_JSON_SPEC
    status: str                   # "active" | "archived" | "draft"
    created_at: datetime          # ISO 8601 timestamp
    updated_at: datetime          # ISO 8601 timestamp
}
```

### Validation Rules

| Field         | Rule                                                             |
| ------------- | ---------------------------------------------------------------- |
| `workflow_id` | Must be `{name}:{version}`, immutable, unique per tenant         |
| `tenant_id`   | 1-128 chars, alphanumeric + underscore/hyphen, required          |
| `name`        | 1-256 chars, required                                            |
| `version`     | Semantic versioning (X.Y.Z), immutable, must match version field |
| `description` | Max 1000 chars, optional                                         |
| `tags`        | List of 1-100 chars strings, optional                            |
| `owner`       | 1-128 chars, required                                            |
| `status`      | One of: "active", "archived", "draft", default: "active"         |
| `definition`  | Valid JSON per WORKFLOW_JSON_SPEC, required                      |

---

## Error Response Format

All error responses follow this format:

```json
{
  "error": "ErrorType",
  "message": "Human-readable error message",
  "status_code": 400,
  "details": {
    "field": "field_name",
    "reason": "Specific validation failure reason"
  }
}
```

---

## API Organization (Swagger Tags)

All workflow endpoints are organized under the **Workflows** semantic tag in the Swagger UI. This groups related workflow CRUD operations together with other functional groups:

| Tag           | Endpoints                                                                                                                    | Purpose                                |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- |
| **Execution** | POST /execute, GET /execute/{run_id}/status                                                                                  | Workflow execution and status tracking |
| **Skills**    | POST /invalidate                                                                                                             | Skill management and cache operations  |
| **Workflows** | POST /workflows, GET /workflows, GET /workflows/{workflow_id}, PUT /workflows/{workflow_id}, DELETE /workflows/{workflow_id} | Workflow CRUD operations               |

The tag grouping improves API documentation clarity by organizing endpoints by functional domain rather than a flat "default" group.

---

## Tenant Scoping

- All endpoints require `tenant_id` parameter
- Workflows are strictly isolated by tenant
- A workflow_id in tenant A is completely separate from the same workflow_id in tenant B
- List operations only return workflows for the specified tenant

---

## Storage [MOCK Phase]

**Implementation**: In-memory dictionary (Python dict) with structure:

```python
{
    "hedge_fund_research_desk": {
        "quant-research:v3.0.0": { /* workflow object */ },
        "sentiment-analysis:v1.2.0": { /* workflow object */ }
    },
    "other_tenant": { ... }
}
```

**Persistence**: None (MOCK phase). Data lost on restart.

**Thread Safety [MVP]**: Will use Redis with atomic operations.

---

## References

- [[WORKFLOW_JSON_SPEC.md]] - Complete workflow JSON schema
- [[../architecture/system-architecture.md]] - System boundaries
- [[../architecture/multi-tenant.md]] - Tenant isolation rules
