from fastapi import APIRouter, HTTPException, Query
from typing import Optional, Any
from ..app_logging import get_logger
from ..modules.shared.deps import get_workflow_store
from ..models import (
    WorkflowCreate,
    WorkflowDetailResponse,
    WorkflowListResponse,
    TenantListResponse,
    WorkflowSummary,
    UpdateWorkflowRequest,
)

logger = get_logger(__name__)
router = APIRouter()

@router.post(
    "/workflows",
    response_model=WorkflowDetailResponse,
    status_code=201,
    tags=["Workflows"],
)
async def create_workflow(request: WorkflowCreate):
    """
    Create a new workflow definition.
    """
    # Validate tenant_id is provided
    if not request.tenant_id:
        raise HTTPException(status_code=422, detail="tenant_id is required")

    # Validate workflow_id format (should be like "workflow-name:version")
    if ":" not in request.workflow_id or not request.version:
        raise HTTPException(
            status_code=422,
            detail="workflow_id must be in format 'name:version' and version must match",
        )
    # Validate definition with DAG cycle safety and reachability checks
    from ..services.workflow_validator import WorkflowValidator
    validator = WorkflowValidator()
    validation_result = validator.validate(request.definition)
    if not validation_result["valid"]:
        logger.warning(f"Invalid workflow definition: {validation_result['errors']}")
        raise HTTPException(
            status_code=422,
            detail=f"Invalid workflow definition: {'; '.join(validation_result['errors'])}"
        )

    store = get_workflow_store()

    # Create workflow via mock store
    workflow_data = {
        "tenant_id": request.tenant_id,
        "workflow_id": request.workflow_id,
        "name": request.name,
        "version": request.version,
        "description": request.description,
        "owner": request.owner,
        "tags": request.tags or [],
        "definition": request.definition,
    }

    result = store.create(request.tenant_id, workflow_data)

    if result is None:
        # Workflow already exists (409 Conflict)
        logger.warning(f"Workflow already exists: {request.workflow_id}")
        raise HTTPException(status_code=409, detail="Conflict: Workflow already exists")

    logger.info(
        f"Workflow created: {request.workflow_id} for tenant {request.tenant_id}"
    )

    return WorkflowDetailResponse(**result)


@router.get("/workflows/tenants", response_model=TenantListResponse, tags=["Workflows"])
async def list_workflow_tenants():
    """
    List all tenants that have workflows registered.
    """
    store = get_workflow_store()
    tenants = store.list_tenants()
    return TenantListResponse(tenants=tenants, count=len(tenants))


@router.get("/workflows", response_model=WorkflowListResponse, tags=["Workflows"])
async def list_workflows(
    tenant_id: str = Query(..., description="Tenant ID for the workflows"),
    status: Optional[str] = None,
    owner: Optional[str] = None,
    tags: Optional[str] = None,
):
    """
    List workflows for a tenant with optional filtering.
    """
    store = get_workflow_store()

    # Parse tags if provided (comma-separated)
    tag_list = None
    if tags:
        tag_list = [t.strip() for t in tags.split(",")]

    # Filter workflows using store
    workflows = store.list_for_tenant(
        tenant_id=tenant_id,
        status=status,
        owner=owner,
        tags=tag_list,
    )

    logger.info(f"Listed {len(workflows)} workflows for tenant {tenant_id}")

    return WorkflowListResponse(
        tenant_id=tenant_id,
        count=len(workflows),
        workflows=[WorkflowSummary(**workflow) for workflow in workflows],
    )


@router.get(
    "/workflows/{workflow_id}",
    response_model=WorkflowDetailResponse,
    tags=["Workflows"],
)
async def get_workflow(workflow_id: str, tenant_id: Optional[str] = None):
    """
    Get a specific workflow definition by ID.
    """
    # Validate tenant_id is provided
    if tenant_id is None:
        raise HTTPException(status_code=400, detail="tenant_id is required")

    store = get_workflow_store()
    workflow = store.get(tenant_id, workflow_id)

    if not workflow:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Retrieved workflow: {workflow_id} for tenant {tenant_id}")

    return WorkflowDetailResponse(**workflow)


@router.put(
    "/workflows/{workflow_id}",
    response_model=WorkflowDetailResponse,
    tags=["Workflows"],
)
async def update_workflow(
    workflow_id: str,
    request: UpdateWorkflowRequest,
    tenant_id: Optional[str] = None,
):
    """
    Update workflow metadata and configuration.
    """
    # Validate tenant_id is provided
    if tenant_id is None:
        raise HTTPException(status_code=400, detail="tenant_id is required")

    # Validate that at least one field is provided
    if not any(
        [
            request.name,
            request.description,
            request.owner,
            request.status,
            request.tags,
            request.definition,
        ]
    ):
        raise HTTPException(
            status_code=400, detail="At least one field must be provided for update"
        )

    # Validate status if provided
    if request.status and request.status not in [
        "active",
        "archived",
        "draft",
        "deprecated",
    ]:
        raise HTTPException(
            status_code=422,
            detail="status must be one of: active, archived, draft, deprecated",
        )

    store = get_workflow_store()

    # Build updates dictionary
    updates: dict[str, Any] = {}
    if request.name is not None:
        updates["name"] = request.name
    if request.description is not None:
        updates["description"] = request.description
    if request.owner is not None:
        updates["owner"] = request.owner
    if request.status is not None:
        updates["status"] = request.status
    if request.tags is not None:
        updates["tags"] = request.tags
    if request.definition is not None:
        from ..services.workflow_validator import WorkflowValidator
        validator = WorkflowValidator()
        validation_result = validator.validate(request.definition)
        if not validation_result["valid"]:
            logger.warning(f"Invalid workflow definition: {validation_result['errors']}")
            raise HTTPException(
                status_code=422,
                detail=f"Invalid workflow definition: {'; '.join(validation_result['errors'])}"
            )
        updates["definition"] = request.definition

    result = store.update(tenant_id, workflow_id, updates)

    if result is None:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Updated workflow: {workflow_id} for tenant {tenant_id}")

    return WorkflowDetailResponse(**result)


@router.delete("/workflows/{workflow_id}", status_code=204, tags=["Workflows"])
async def delete_workflow(workflow_id: str, tenant_id: Optional[str] = None):
    """
    Delete a workflow definition.
    """
    # Validate tenant_id is provided
    if not tenant_id:
        raise HTTPException(status_code=400, detail="tenant_id is required")

    store = get_workflow_store()
    success = store.delete(tenant_id, workflow_id)

    if not success:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Deleted workflow: {workflow_id} for tenant {tenant_id}")
