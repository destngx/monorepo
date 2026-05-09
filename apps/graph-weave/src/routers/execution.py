from fastapi import APIRouter, HTTPException
import uuid
import threading
from typing import Any
from ..app_logging import get_logger
from ..modules.shared.deps import (
    get_executor,
    get_workflow_store,
    get_checkpoint_service,
    get_thread_lifecycle_service
)
from ..services.status_service import StatusService
from ..models import (
    ExecuteRequest,
    ExecuteResponse,
    RecoveryRequest,
    CancelResponse
)

logger = get_logger(__name__)
router = APIRouter()

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
        executor = get_executor()

        # Immediate transition to 'running'
        status_service.transition_status(
            tenant_id,
            run_id,
            "running",
            {"workflow_id": workflow_id, "thread_id": thread_id}
        )
        # Update local memory too
        if run_id in execution_runs:
            execution_runs[run_id]["status"] = "running"
            
        execution_result = executor.execute(
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


def scheduler_execution_handler(tenant_id: str, workflow_id: str, input_data: dict[str, Any]):
    """Handler called by APScheduler to trigger a workflow."""
    run_id = str(uuid.uuid4())
    thread_id = str(uuid.uuid4())

    workflow_store = get_workflow_store()
    workflow = workflow_store.get(tenant_id, workflow_id)

    if not workflow:
        logger.error(f"Scheduled execution failed: Workflow {workflow_id} not found for tenant {tenant_id}")
        return

    logger.info(f"Triggering scheduled execution: workflow_id={workflow_id}, tenant_id={tenant_id}, run_id={run_id}")

    status_service.set_status(
        tenant_id,
        run_id,
        "queued",
        {"workflow_id": workflow_id, "thread_id": thread_id, "is_scheduled": True},
    )

    execution_runs[run_id] = {
        "run_id": run_id,
        "thread_id": thread_id,
        "workflow_id": workflow_id,
        "tenant_id": tenant_id,
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
            tenant_id,
            workflow_id,
            workflow,
            input_data,
        ),
        daemon=True,
    )
    worker.start()


@router.post("/execute", response_model=ExecuteResponse, tags=["Execution"])
async def execute(request: ExecuteRequest):
    """Execute a workflow by ID."""
    run_id = str(uuid.uuid4())
    thread_id = str(uuid.uuid4())

    logger.info(
        f"Execution started: run_id={run_id}, thread_id={thread_id}, workflow_id={request.workflow_id}"
    )

    try:
        workflow_store = get_workflow_store()
        workflow = workflow_store.get(request.tenant_id, request.workflow_id)

        if not workflow:
            raise HTTPException(status_code=404, detail="Workflow not found")

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


@router.get("/execute/{run_id}/status", tags=["Execution"])
async def get_execution_status(run_id: str):
    logger.info(f"Fetching status for run_id={run_id}")

    if run_id not in execution_runs:
        raise HTTPException(status_code=404, detail="Run not found")

    result = execution_runs[run_id]
    
    # Fetch live events from executor if the run is still active
    events = result.get("events", [])
    if (not events or result.get("status") in ["queued", "running"]):
        try:
            executor = get_executor()
            if run_id in executor.execution_events:
                events = executor.execution_events[run_id]
        except Exception as e:
            logger.warning(f"Failed to fetch live events from executor: {e}")

    return {
        "run_id": run_id,
        "status": result.get("status", "unknown"),
        "workflow_id": result.get("workflow_id"),
        "tenant_id": result.get("tenant_id"),
        "events": events,
        "final_state": result.get("final_state"),
        "workflow_state": result.get("workflow_state"),
        "hop_count": result.get("hop_count"),
    }


@router.post("/execute/{run_id}/recover", response_model=ExecuteResponse, tags=["Execution"])
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


@router.post("/execute/{run_id}/cancel", response_model=CancelResponse, tags=["Execution"])
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
