# VERIFY-GW-WF-API-TAGS: Swagger UI semantic tag organization

> **Related Tasks** : GW-WF-103, GW-WF-104, GW-WF-105, GW-WF-106, GW-WF-107, GW-WF-108
> **Verification Type** : SCHEMA, DOC
> **Phase ID** : MOCK
> **Focus**: OpenAPI schema and Swagger UI endpoint organization

---

## Objective

Verify that all 8 GraphWeave API endpoints are correctly organized into semantic tag groups in Swagger UI, improving API documentation clarity and navigation.

---

## Tag Organization Requirements

### Tag Definitions

| Tag           | Description                            | Endpoints                                                                                                                    |
| ------------- | -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| **Execution** | Workflow execution and status tracking | POST /execute, GET /execute/{run_id}/status                                                                                  |
| **Skills**    | Skill management and cache operations  | POST /invalidate                                                                                                             |
| **Workflows** | Workflow CRUD operations               | POST /workflows, GET /workflows, GET /workflows/{workflow_id}, PUT /workflows/{workflow_id}, DELETE /workflows/{workflow_id} |

### Implementation Requirements

1. **FastAPI Decorators**: Each endpoint must include `tags=["TagName"]` parameter in decorator
2. **OpenAPI Metadata**: FastAPI app initialization must include `openapi_tags` list with group descriptions
3. **Swagger UI**: Groups must display as collapsible sections organized by tag name

---

## Verification Checklist

### Code Verification (SCHEMA)

- [ ] **SCHEMA-001**: Endpoint POST /workflows includes `tags=["Workflows"]` decorator
- [ ] **SCHEMA-002**: Endpoint GET /workflows includes `tags=["Workflows"]` decorator
- [ ] **SCHEMA-003**: Endpoint GET /workflows/{workflow_id} includes `tags=["Workflows"]` decorator
- [ ] **SCHEMA-004**: Endpoint PUT /workflows/{workflow_id} includes `tags=["Workflows"]` decorator
- [ ] **SCHEMA-005**: Endpoint DELETE /workflows/{workflow_id} includes `tags=["Workflows"]` decorator
- [ ] **SCHEMA-006**: Endpoint POST /execute includes `tags=["Execution"]` decorator
- [ ] **SCHEMA-007**: Endpoint GET /execute/{run_id}/status includes `tags=["Execution"]` decorator
- [ ] **SCHEMA-008**: Endpoint POST /invalidate includes `tags=["Skills"]` decorator
- [ ] **SCHEMA-009**: FastAPI app has `openapi_tags` parameter with 3 tag groups (Execution, Skills, Workflows)
- [ ] **SCHEMA-010**: Each tag group includes "name", "description", and "operations" (optional) fields

### OpenAPI Schema Verification (SCHEMA)

- [ ] **OPENAPI-001**: Generated `/openapi.json` includes `tags` array at root level
- [ ] **OPENAPI-002**: Each endpoint operation includes `tags` array with one tag name
- [ ] **OPENAPI-003**: All 3 tag groups appear in `tags` array in OpenAPI schema
- [ ] **OPENAPI-004**: Tag descriptions from FastAPI match descriptions in OpenAPI schema

### Swagger UI Verification (DOC)

- [ ] **UI-001**: Swagger UI at `/docs` displays 3 collapsible sections: "Execution", "Skills", "Workflows"
- [ ] **UI-002**: "Execution" section contains 2 endpoints (POST /execute, GET /execute/{run_id}/status)
- [ ] **UI-003**: "Skills" section contains 1 endpoint (POST /invalidate)
- [ ] **UI-004**: "Workflows" section contains 5 endpoints (all workflow CRUD operations)
- [ ] **UI-005**: Endpoints are grouped correctly with no cross-contamination
- [ ] **UI-006**: Tag descriptions are visible in Swagger UI for each group

### Documentation Verification (DOC)

- [ ] **DOC-001**: WORKFLOW_MANAGEMENT_API.md includes "API Organization" section with tag table
- [ ] **DOC-002**: WORKFLOW_MANAGEMENT_API.md lists all 5 workflow endpoints with "Tag: Workflows" descriptor
- [ ] **DOC-003**: plan/api-organization.md exists with design rationale and implementation details
- [ ] **DOC-004**: Task files GW-WF-103 through GW-WF-108 include "Implementation Notes" section referencing API tags
- [ ] **DOC-005**: delta-changes.md includes entry documenting Swagger API Tags implementation

### Functional Verification (FUNC)

- [ ] **FUNC-001**: Test coverage exists for OpenAPI schema generation
- [ ] **FUNC-002**: Tests verify all 8 endpoints appear in `/openapi.json`
- [ ] **FUNC-003**: Tests verify endpoint operations include correct `tags` values
- [ ] **FUNC-004**: No test regressions (120/121 tests passing)

---

## Test Case: OpenAPI Schema Validation

**Objective**: Verify OpenAPI schema includes correct tag metadata

**Test Code** (Python pseudo-code):

```python
def test_openapi_tags_organization():
    """Verify OpenAPI schema includes organized endpoint tags"""
    client = TestClient(app)
    response = client.get("/openapi.json")
    schema = response.json()

    # Verify tag definitions
    assert "tags" in schema
    tag_names = {tag["name"] for tag in schema["tags"]}
    assert tag_names == {"Execution", "Skills", "Workflows"}

    # Verify each tag has description
    for tag in schema["tags"]:
        assert "description" in tag
        assert len(tag["description"]) > 0

    # Verify Workflows group has 5 endpoints
    workflows_endpoints = [
        path for path, methods in schema["paths"].items()
        for method, details in methods.items()
        if "Workflows" in details.get("tags", [])
    ]
    assert len(workflows_endpoints) == 5

    # Verify Execution group has 2 endpoints
    execution_endpoints = [
        path for path, methods in schema["paths"].items()
        for method, details in methods.items()
        if "Execution" in details.get("tags", [])
    ]
    assert len(execution_endpoints) == 2

    # Verify Skills group has 1 endpoint
    skills_endpoints = [
        path for path, methods in schema["paths"].items()
        for method, details in methods.items()
        if "Skills" in details.get("tags", [])
    ]
    assert len(skills_endpoints) == 1
```

---

## Acceptance Criteria

✅ **All** of the following must be true for this verification to pass:

1. ✅ All 8 endpoints are tagged with appropriate semantic groups (no "default" tag)
2. ✅ Swagger UI `/docs` displays 3 organized collapsible endpoint groups
3. ✅ Documentation (WORKFLOW_MANAGEMENT_API.md) reflects tag organization
4. ✅ No test regressions from baseline (138/139 passing vs 120/121 baseline)
5. ✅ OpenAPI schema (`/openapi.json`) includes tag metadata with descriptions

**Status**: ✅ **VERIFICATION PASSED** - All 18 tag validation tests passing

---

## Test Implementation Summary

**File**: `tests/test_openapi.py`

**Test Classes Added**: 5

- `TestOpenAPITags` - Tag array and definition validation (5 tests)
- `TestWorkflowEndpointTags` - Workflow endpoints tag verification (5 tests)
- `TestExecutionEndpointTags` - Execution endpoints tag verification (2 tests)
- `TestSkillEndpointTags` - Skills endpoint tag verification (1 test)
- `TestEndpointTagCoverage` - Cross-contamination and coverage validation (5 tests)

**Total New Tests**: 18
**Test Results**: 18/18 passing ✅
**Total Test Suite**: 139 passing (baseline 120 + 18 new + 1 skipped)

---

## References

- `[[../WORKFLOW_MANAGEMENT_API.md]]` — API specification with tag organization
- `[[plan/api-organization.md]]` — Design decision and rationale
- `[[../tasks/MOCK/GW-WF-103.md]]` through `[[../tasks/MOCK/GW-WF-108.md]]` — Individual task files
- `[[../../delta-changes.md]]` — Implementation history
