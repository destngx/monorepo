from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from fastapi.exceptions import RequestValidationError
import logging
import uuid
from contextlib import asynccontextmanager

from .config import GraphWeaveConfig
from .logging import setup_logging, get_logger
from .modules.shared.deps import (
    init_services,
    get_cache,
    get_workflow_store,
    get_checkpoint_store,
)
from .adapters.ai_provider import MockAIProvider
from .adapters.langgraph_executor import MockLangGraphExecutor
from .models import (
    ExecuteRequest,
    ExecuteResponse,
    InvalidateRequest,
    InvalidateResponse,
    WorkflowCreate,
    WorkflowDetailResponse,
    WorkflowListResponse,
    UpdateWorkflowRequest,
)

setup_logging(debug=GraphWeaveConfig.DEBUG)
logger = get_logger(__name__)

ai_provider = MockAIProvider()
langgraph_executor = MockLangGraphExecutor(ai_provider=ai_provider)
execution_runs: dict = {}


@asynccontextmanager
async def lifespan(app: FastAPI):
    try:
        GraphWeaveConfig.validate()
        logger.info("Configuration validation passed")
    except ValueError as e:
        logger.warning(f"Configuration note: {e}")

    init_services()
    logger.info("Services initialized")

    yield


app = FastAPI(
    title="GraphWeave",
    description="Workflow execution engine with LangGraph, Redis, and MCP",
    version="0.1.0",
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
    lifespan=lifespan,
    openapi_tags=[
        {
            "name": "Execution",
            "description": "Workflow execution and status tracking - POST to execute, GET status by run_id",
        },
        {
            "name": "Skills",
            "description": "Skill management and cache operations - invalidate cached skill definitions",
        },
        {
            "name": "Workflows",
            "description": "Workflow CRUD operations - create, read, list, update, delete workflow definitions",
        },
    ],
)


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request, exc):
    details = []
    for error in exc.errors():
        details.append(
            {
                "field": ".".join(str(x) for x in error["loc"][1:]),
                "message": error["msg"],
                "type": error["type"],
            }
        )

    return JSONResponse(
        status_code=422,
        content={
            "error": "ValidationError",
            "message": "Request validation failed",
            "details": details,
            "status_code": 422,
        },
    )
    try:
        GraphWeaveConfig.validate()
        logger.info("Configuration validation passed")
    except ValueError as e:
        logger.warning(f"Configuration note: {e}")

    init_services()
    logger.info("Services initialized")


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.get("/api/docs")
async def docs():
    return app.openapi()


@app.post("/execute", response_model=ExecuteResponse, tags=["Execution"])
async def execute(request: ExecuteRequest):
    """
    Execute a workflow by ID.

    TODO [MVP]: Authenticate user and verify permission to execute workflow
    TODO [MVP]: Check workflow is not archived/deprecated before execution
    TODO [MVP]: Validate input matches workflow input schema
    TODO [MVP]: Queue execution if async execution mode is enabled
    TODO [MVP]: Add execution timeout handling (default 5 min, configurable per workflow)
    """
    run_id = str(uuid.uuid4())
    thread_id = str(uuid.uuid4())

    logger.info(
        f"Mock execution started: run_id={run_id}, thread_id={thread_id}, workflow_id={request.workflow_id}"
    )

    try:
        workflow_store = get_workflow_store()
        workflow = workflow_store.get(request.tenant_id, request.workflow_id)

        if not workflow:
            # MOCK PHASE: Auto-create minimal workflow for testing
            # TODO [MVP]: Remove this - workflows must be pre-created
            logger.warning(
                f"Workflow not found, creating mock workflow: {request.workflow_id}"
            )
            mock_workflow = {
                "tenant_id": request.tenant_id,
                "workflow_id": request.workflow_id,
                "name": request.workflow_id,
                "version": "1.0.0",
                "description": "Auto-created mock workflow for testing",
                "owner": "system",
                "tags": ["mock"],
                "definition": {
                    "nodes": [
                        {"id": "entry", "type": "entry_node", "config": {}},
                        {"id": "exit", "type": "exit_node", "config": {}},
                    ],
                    "edges": [{"from": "entry", "to": "exit"}],
                    "entry_point": "entry",
                    "exit_point": "exit",
                },
            }
            workflow_store.create(request.tenant_id, mock_workflow)
            workflow = workflow_store.get(request.tenant_id, request.workflow_id)

        checkpoint_store = get_checkpoint_store()
        cache = get_cache()

        langgraph_executor.set_current_run_id(run_id)

        execution_result = langgraph_executor.execute(
            workflow=workflow,
            input_data=request.input,
            checkpoint_store=checkpoint_store,
            cache=cache,
        )

        execution_runs[run_id] = execution_result

        logger.info(
            f"Execution completed: run_id={run_id}, status={execution_result.get('status')}"
        )

        # MOCK PHASE: Always return pending status for consistent mock behavior
        # TODO [MVP]: Return actual execution status from executor
        return ExecuteResponse(
            run_id=run_id,
            thread_id=thread_id,
            status="pending",
            workflow_id=request.workflow_id,
            tenant_id=request.tenant_id,
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Execution failed: {str(e)}", exc_info=True)
        return ExecuteResponse(
            run_id=run_id,
            thread_id=thread_id,
            status="error",
            workflow_id=request.workflow_id,
            tenant_id=request.tenant_id,
        )


@app.get("/execute/{run_id}/status", tags=["Execution"])
async def get_execution_status(run_id: str):
    logger.info(f"Fetching status for run_id={run_id}")

    if run_id not in execution_runs:
        # MOCK PHASE: Return mock status for non-existent run_ids
        # TODO [MVP]: Remove this - return 404 for non-existent runs
        logger.warning(f"Run not found, returning mock status: {run_id}")
        return {
            "run_id": run_id,
            "status": "pending",
            "workflow_id": None,
            "tenant_id": None,
            "events": [],
            "final_state": None,
            "hop_count": 0,
        }

    result = execution_runs[run_id]

    return {
        "run_id": run_id,
        "status": result.get("status", "unknown"),
        "workflow_id": result.get("workflow_id"),
        "tenant_id": result.get("tenant_id"),
        "events": result.get("events", []),
        "final_state": result.get("final_state"),
        "hop_count": result.get("hop_count"),
    }


@app.post("/invalidate", response_model=InvalidateResponse, tags=["Skills"])
async def invalidate(request: InvalidateRequest):
    cache = get_cache()

    logger.info(
        f"Cache invalidation triggered: tenant_id={request.tenant_id}, skill_id={request.skill_id}, reason={request.reason}"
    )

    for version in ["latest", "1.0.0", "2.0.0"]:
        key = f"skills:level1:{request.tenant_id}:{request.skill_id}:{version}"
        if cache.exists(key):
            cache.delete(key)
            logger.info(f"Invalidated cache key: {key}")

    return InvalidateResponse(
        status="invalidated",
        tenant_id=request.tenant_id,
        skill_id=request.skill_id,
        reason=request.reason,
    )


# ============================================================================
# Workflow CRUD Endpoints
# ============================================================================


@app.post(
    "/workflows",
    response_model=WorkflowDetailResponse,
    status_code=201,
    tags=["Workflows"],
)
async def create_workflow(request: WorkflowCreate):
    """
    Create a new workflow definition.

    TODO [MVP]: Add authentication/authorization to verify tenant ownership
    TODO [MVP]: Validate workflow definition schema (DAG connectivity, node types)
    TODO [MVP]: Check for name/ID uniqueness across organization
    TODO [MVP]: Persist to PostgreSQL with audit logging
    TODO [MVP]: Register workflow in skill discovery system
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


@app.get("/workflows", response_model=WorkflowListResponse, tags=["Workflows"])
async def list_workflows(
    tenant_id: str = None,
    status: str = None,
    owner: str = None,
    tags: str = None,  # Comma-separated list
):
    """
    List workflows for a tenant with optional filtering.

    TODO [MVP]: Support pagination (offset, limit)
    TODO [MVP]: Add sorting options (by name, created_at, updated_at)
    TODO [MVP]: Cache results with 5-minute TTL
    TODO [MVP]: Add permission checks (user can only see workflows they own or are shared)
    TODO [MVP]: Full-text search on workflow name/description
    """
    # Validate tenant_id is provided
    if not tenant_id:
        raise HTTPException(status_code=400, detail="tenant_id is required")

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
        workflows=workflows,
    )


@app.get(
    "/workflows/{workflow_id}",
    response_model=WorkflowDetailResponse,
    tags=["Workflows"],
)
async def get_workflow(workflow_id: str, tenant_id: str = None):
    """
    Get a specific workflow definition by ID.

    TODO [MVP]: Add permission check (user can only read if they own or have access)
    TODO [MVP]: Return version history if requested
    TODO [MVP]: Include execution statistics (runs, success rate, avg duration)
    TODO [MVP]: Add cache with 30-minute TTL
    """
    # Validate tenant_id is provided
    if not tenant_id:
        raise HTTPException(status_code=400, detail="tenant_id is required")

    store = get_workflow_store()
    workflow = store.get(tenant_id, workflow_id)

    if not workflow:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Retrieved workflow: {workflow_id} for tenant {tenant_id}")

    return WorkflowDetailResponse(**workflow)


@app.put(
    "/workflows/{workflow_id}",
    response_model=WorkflowDetailResponse,
    tags=["Workflows"],
)
async def update_workflow(
    workflow_id: str,
    request: UpdateWorkflowRequest,
    tenant_id: str = None,
):
    """
    Update workflow metadata and configuration.

    TODO [MVP]: Validate that immutable fields (workflow_id, version, created_at) are not modified
    TODO [MVP]: Add permission check (only owner or admin can update)
    TODO [MVP]: Create version history entry
    TODO [MVP]: Trigger workflow recompilation if definition changed
    TODO [MVP]: Notify dependent workflows of changes
    TODO [MVP]: Audit log the changes with user/timestamp
    """
    # Validate tenant_id is provided
    if not tenant_id:
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
    updates = {}
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
        updates["definition"] = request.definition

    result = store.update(tenant_id, workflow_id, updates)

    if result is None:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Updated workflow: {workflow_id} for tenant {tenant_id}")

    return WorkflowDetailResponse(**result)


@app.delete("/workflows/{workflow_id}", status_code=204, tags=["Workflows"])
async def delete_workflow(workflow_id: str, tenant_id: str = None):
    """
    Delete a workflow definition.

    TODO [MVP]: Add permission check (only owner or admin can delete)
    TODO [MVP]: Check if workflow is referenced by other workflows (prevent orphans)
    TODO [MVP]: Check if there are active executions using this workflow
    TODO [MVP]: Create audit log entry for deletion
    TODO [MVP]: Soft delete (mark as deleted, keep history) instead of hard delete
    TODO [MVP]: Notify users who have saved this workflow as template
    """
    # Validate tenant_id is provided
    if not tenant_id:
        raise HTTPException(status_code=400, detail="tenant_id is required")

    store = get_workflow_store()
    success = store.delete(tenant_id, workflow_id)

    if not success:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Deleted workflow: {workflow_id} for tenant {tenant_id}")

    # Return 204 No Content (FastAPI handles this automatically with status_code=204)
