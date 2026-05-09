from fastapi import FastAPI
from fastapi.responses import JSONResponse
from fastapi.exceptions import RequestValidationError
import logging
import os
from contextlib import asynccontextmanager

from .config import GraphWeaveConfig
from .app_logging import setup_logging, get_logger
from .modules.shared.deps import (
    init_services,
    get_cache,
    get_workflow_store,
    get_scheduler_service,
)

# Import routers
from .routers import execution, skills, workflows, schedules

setup_logging(debug=GraphWeaveConfig.DEBUG)
logger = get_logger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    try:
        GraphWeaveConfig.validate()
        logger.info("Configuration validation passed")
    except ValueError as e:
        logger.warning(f"Configuration note: {e}")

    init_services()
    logger.info("Services initialized")

    # Auto-create system workflows on startup
    try:
        workflow_store = get_workflow_store()
        workflow_store.sync_predefined_workflows("system")
        logger.info("System workflows synchronized")
    except Exception as e:
        logger.warning(f"System workflow synchronization failed: {e}")

    if GraphWeaveConfig.SCHEDULER_ENABLED:
        scheduler_service = get_scheduler_service()
        try:
            # Connect the scheduler execution handler from the execution router
            scheduler_service.execution_handler = execution.scheduler_execution_handler
            scheduler_service.start()

            # Load all enabled schedules for existing tenants.
            # Best-effort during startup so local serve can still boot when Redis is unavailable.
            scheduler_service.sync_schedules("default")
            logger.info("Scheduler started and synced for 'default' tenant")
        except Exception as e:
            logger.warning(f"Scheduler startup skipped: {e}")

    yield

    if GraphWeaveConfig.SCHEDULER_ENABLED:
        get_scheduler_service().shutdown()
        logger.info("Scheduler shut down")


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
        {
            "name": "Schedules",
            "description": "Cron-based workflow scheduling CRUD operations",
        },
    ],
)

app.include_router(execution.router)
app.include_router(skills.router)
app.include_router(workflows.router)
app.include_router(schedules.router)


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
