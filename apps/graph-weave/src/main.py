from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse
from fastapi.exceptions import RequestValidationError
import logging
import os
import uuid
import threading
import time
from contextlib import asynccontextmanager
from typing import Any, Optional

from .config import GraphWeaveConfig
from .app_logging import setup_logging, get_logger
from .modules.shared.deps import (
    init_services,
    get_cache,
    get_workflow_store,
    get_checkpoint_store,
    get_checkpoint_service,
    get_thread_lifecycle_service,
    get_redis_client,
)
from .adapters.langgraph_executor import RealLangGraphExecutor
from .adapters.mcp_router import MCPRouter
from .adapters.ai_gateway_adapter import AIGatewayClient
from .services.status_service import StatusService
from .models import (
    ExecuteRequest,
    ExecuteResponse,
    InvalidateRequest,
    InvalidateResponse,
    RecoveryRequest,
    CancelResponse,
    WorkflowCreate,
    WorkflowDetailResponse,
    WorkflowListResponse,
    WorkflowSummary,
    UpdateWorkflowRequest,
)

setup_logging(debug=GraphWeaveConfig.DEBUG)
logger = get_logger(__name__)

mcp_router = MCPRouter()
langgraph_executor = RealLangGraphExecutor(
    mcp_router=mcp_router,
    redis_client=get_redis_client(),
)
execution_runs: dict[str, dict[str, Any]] = {}
status_service = StatusService(execution_runs)


def _background_execute_run(
    run_id: str,
    thread_id: str,
    tenant_id: str,
    workflow_id: str,
    workflow: dict[str, Any],
    input_data: dict[str, Any],
) -> None:
    """Execute the workflow in a background thread using the real executor."""
    try:
        logger.info(
            f"[BG EXEC] Starting background execution: run_id={run_id}, workflow_id={workflow_id}"
        )
        execution_result = langgraph_executor.execute(
            run_id=run_id,
            thread_id=thread_id,
            tenant_id=tenant_id,
            workflow=workflow,
            input_data=input_data,
        )
        logger.info(
            f"[BG EXEC] Execution completed: run_id={run_id}, status={execution_result.get('status')}"
        )
    except Exception as e:
        logger.error(
            f"[BG EXEC] Execution failed: run_id={run_id}, error={str(e)}",
            exc_info=True,
        )
        execution_result = {
            "run_id": run_id,
            "thread_id": thread_id,
            "workflow_id": workflow_id,
            "tenant_id": tenant_id,
            "status": "failed",
            "error": str(e),
            "events": [],
            "final_state": None,
            "hop_count": 0,
        }

    execution_runs[run_id] = execution_result
    status_service.transition_status(
        tenant_id,
        run_id,
        execution_result.get("status", "failed"),
        execution_result,
    )


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
    """Health check for all external services."""
    health_status = {"status": "ok", "services": {}}

    cache = get_cache()
    try:
        cache.set("health_check", "ok")
        cache.get("health_check")
        health_status["services"]["redis"] = {"status": "healthy"}
    except Exception as e:
        logger.warning(f"Redis health check failed: {e}")
        health_status["services"]["redis"] = {"status": "unhealthy", "error": str(e)}
        health_status["status"] = "unhealthy"
        return health_status

    try:
        gateway_url = os.getenv("AI_GATEWAY_URL", "http://localhost:8080/v1")
        import httpx

        # Check if gateway is reachable
        try:
            # We check the root or v1 endpoint for a 404/200/405 (anything that proves connectivity)
            # Standard health check for the gateway
            response = httpx.get(gateway_url, timeout=5.0)
            health_status["services"]["ai_gateway"] = {
                "status": "healthy",
                "url": gateway_url,
                "response_code": response.status_code,
            }
        except httpx.HTTPError as e:
            logger.warning(f"AI Gateway health check failed: {e}")
            health_status["services"]["ai_gateway"] = {
                "status": "unhealthy",
                "error": str(e),
            }
            health_status["status"] = "degraded"

    except Exception as e:
        logger.error(f"Unexpected error in AI Gateway health check: {e}")
        health_status["services"]["ai_gateway"] = {"status": "error", "error": str(e)}
        health_status["status"] = "degraded"

    except Exception as e:
        logger.error(f"Unexpected error in GitHub Copilot health check: {e}")
        health_status["services"]["github_copilot"] = {
            "status": "error",
            "error": str(e),
        }
        health_status["status"] = "degraded"

    return health_status


@app.get("/api/docs")
async def docs():
    return app.openapi()


@app.post("/execute", response_model=ExecuteResponse, tags=["Execution"])
async def execute(request: ExecuteRequest):
    """
    Execute a workflow by ID.

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
            raise HTTPException(status_code=404, detail="Workflow not found")

        checkpoint_store = get_checkpoint_store()
        cache = get_cache()

        status_service.set_status(
            request.tenant_id,
            run_id,
            "queued",
            {"workflow_id": request.workflow_id, "thread_id": thread_id},
        )

        execution_runs[run_id] = {
            "run_id": run_id,
            "thread_id": thread_id,
            "workflow_id": request.workflow_id,
            "tenant_id": request.tenant_id,
            "status": "queued",
            "events": [],
            "final_state": None,
            "hop_count": 0,
        }

        worker = threading.Thread(
            target=_background_execute_run,
            args=(
                run_id,
                thread_id,
                request.tenant_id,
                request.workflow_id,
                workflow,
                request.input,
            ),
            daemon=True,
        )
        worker.start()

        logger.info(
            f"Execution queued: run_id={run_id}, thread_id={thread_id}, workflow_id={request.workflow_id}"
        )

        return ExecuteResponse(
            run_id=run_id,
            thread_id=thread_id,
            status="queued",
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
            status="failed",
            workflow_id=request.workflow_id,
            tenant_id=request.tenant_id,
        )


@app.get("/execute/{run_id}/status", tags=["Execution"])
async def get_execution_status(run_id: str):
    logger.info(f"Fetching status for run_id={run_id}")

    if run_id not in execution_runs:
        raise HTTPException(status_code=404, detail="Run not found")

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


@app.post(
    "/execute/{run_id}/recover", response_model=ExecuteResponse, tags=["Execution"]
)
async def recover_execution(run_id: str, request: RecoveryRequest):
    if run_id not in execution_runs:
        raise HTTPException(status_code=404, detail="Run not found")

    checkpoint_service = get_checkpoint_service()
    checkpoint = checkpoint_service.load_checkpoint(
        execution_runs[run_id].get("tenant_id", "unknown"), request.thread_id
    )
    if checkpoint is None:
        raise HTTPException(status_code=404, detail="Checkpoint not found")

    execution_runs[run_id]["status"] = "running"
    execution_runs[run_id]["final_state"] = checkpoint.get("workflow_state", {})
    return ExecuteResponse(
        run_id=run_id,
        thread_id=request.thread_id,
        status="running",
        workflow_id=execution_runs[run_id].get("workflow_id", "unknown"),
        tenant_id=execution_runs[run_id].get("tenant_id", "unknown"),
    )


@app.post("/execute/{run_id}/cancel", response_model=CancelResponse, tags=["Execution"])
async def cancel_execution(run_id: str):
    if run_id not in execution_runs:
        raise HTTPException(status_code=404, detail="Run not found")

    result = execution_runs[run_id]
    tenant_id = result.get("tenant_id", "unknown")
    thread_id = result.get("thread_id", run_id)

    thread_service = get_thread_lifecycle_service()
    thread_service.set_kill_flag(tenant_id, thread_id)
    thread_service.remove_active_thread(tenant_id, thread_id)
    checkpoint_service = get_checkpoint_service()
    checkpoint_service.clear_checkpoint(tenant_id, thread_id)

    result["status"] = "cancelled"
    status_service.transition_status(tenant_id, run_id, "cancelled", result)

    return CancelResponse(run_id=run_id, status="cancelled", thread_id=thread_id)


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
    tenant_id: Optional[str] = None,
    status: Optional[str] = None,
    owner: Optional[str] = None,
    tags: Optional[str] = None,
):
    """
    List workflows for a tenant with optional filtering.

    """
    # Validate tenant_id is provided
    if tenant_id is None:
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
        workflows=[WorkflowSummary(**workflow) for workflow in workflows],
    )


@app.get(
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


@app.put(
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
        updates["definition"] = request.definition

    result = store.update(tenant_id, workflow_id, updates)

    if result is None:
        raise HTTPException(status_code=404, detail="Workflow not found")

    logger.info(f"Updated workflow: {workflow_id} for tenant {tenant_id}")

    return WorkflowDetailResponse(**result)


@app.delete("/workflows/{workflow_id}", status_code=204, tags=["Workflows"])
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

    # Return 204 No Content (FastAPI handles this automatically with status_code=204)
