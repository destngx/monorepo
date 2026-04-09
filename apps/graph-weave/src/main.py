from fastapi import FastAPI
import logging
import uuid

from .config import GraphWeaveConfig
from .logging import setup_logging, get_logger
from .modules.shared.deps import init_services, get_cache
from .models import (
    ExecuteRequest,
    ExecuteResponse,
    InvalidateRequest,
    InvalidateResponse,
)

setup_logging(debug=GraphWeaveConfig.DEBUG)
logger = get_logger(__name__)

app = FastAPI(
    title="GraphWeave",
    description="Workflow execution engine with LangGraph, Redis, and MCP",
    version="0.1.0",
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
)


@app.on_event("startup")
async def startup_event():
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


@app.post("/execute", response_model=ExecuteResponse)
async def execute(request: ExecuteRequest):
    run_id = request.run_id if request.run_id else str(uuid.uuid4())
    thread_id = str(uuid.uuid4())

    logger.info(
        f"Mock execution started: run_id={run_id}, thread_id={thread_id}, workflow_id={request.workflow_id}"
    )

    return ExecuteResponse(
        run_id=run_id,
        thread_id=thread_id,
        status="pending",
        workflow_id=request.workflow_id,
        tenant_id=request.tenant_id,
    )


@app.get("/execute/{run_id}/status")
async def get_execution_status(run_id: str):
    logger.info(f"Fetching status for run_id={run_id}")

    return {"run_id": run_id, "status": "pending", "events": []}


@app.post("/invalidate", response_model=InvalidateResponse)
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
