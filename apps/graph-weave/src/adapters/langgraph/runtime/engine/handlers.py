import time
import logging
from datetime import datetime, timezone
from typing import Any, Dict, Optional
from .events import emit_event
from ..state import StateResolver

logger = logging.getLogger(__name__)

def handle_exit_node(
    executor: Any,
    run_id: str,
    workflow_id: str,
    node: Dict[str, Any],
    state: Dict[str, Any]
) -> None:
    config = node.get("config", {})
    mapping = node.get("output_mapping") or config.get("output_mapping", {})
    if mapping:
        final_output = StateResolver(state).resolve_mapping(mapping)
        for key, value in final_output.items():
            logger.debug("Exit mapping %s resolved to: %s", key, type(value).__name__)
        state.setdefault("runtime", {})["output"] = final_output
    else:
        state.setdefault("runtime", {})["output"] = {}

    required_outputs = config.get("required_outputs") or node.get("required_outputs") or []
    missing_outputs = []
    for key in required_outputs:
        value = state.get("runtime", {}).get("output", {}).get(key)
        if value is None or value == "" or value == []:
            missing_outputs.append(key)
    if missing_outputs:
        raise ValueError(
            "Exit node missing required outputs: " + ", ".join(missing_outputs)
        )
    
    emit_event(executor, run_id, "workflow.completed", {
        "run_id": run_id, 
        "workflow_id": workflow_id, 
        "status": "completed", 
        "final_state_keys": list(state.get("runtime", {}).get("output", {}).keys())
    })
    state["status"] = "completed"
    state.setdefault("runtime", {})["status"] = "completed"

def handle_node_failure(executor: Any, run_id: str, node_id: str, state: Dict[str, Any], error: Exception) -> None:
    logger.error(f"Node {node_id} failed: {error}", exc_info=True)
    emit_event(executor, run_id, "node.failed", {"node_id": node_id, "error": str(error)})
    state["errors"].append({
        "node_id": node_id,
        "error": str(error),
        "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
    })

def finalize_execution(
    executor: Any,
    run_id: str,
    thread_id: str,
    tenant_id: str,
    workflow_id: str,
    state: Dict[str, Any],
    start_time: float,
    hop_count: int
) -> Dict[str, Any]:
    state["status"] = state.get("status") or "completed"
    state["hop_count"] = hop_count
    state.setdefault("runtime", {})["status"] = state["status"]
    state.setdefault("runtime", {})["hop_count"] = hop_count
    elapsed = time.monotonic() - start_time
    emit_event(executor, run_id, "request.completed", {
        "status": state["status"], 
        "hop_count": hop_count, 
        "elapsed_seconds": elapsed
    })
    output = state.get("runtime", {}).get("output", {})
    return {
        "run_id": run_id, 
        "thread_id": thread_id, 
        "tenant_id": tenant_id, 
        "workflow_id": workflow_id,
        "status": state["status"], 
        "hop_count": hop_count, 
        "events": executor.execution_events.get(run_id, []),
        "output": output,
        "state": state,
        "elapsed_seconds": elapsed,
    }

def handle_execution_error(
    executor: Any,
    run_id: str,
    thread_id: str,
    tenant_id: str,
    workflow_id: str,
    state: Dict[str, Any],
    error: Exception
) -> Dict[str, Any]:
    logger.exception(f"Execution error: {error}")
    state["status"] = "failed"
    state.setdefault("runtime", {})["status"] = "failed"
    emit_event(executor, run_id, "request.failed", {"error": str(error)})
    return {
        "run_id": run_id, 
        "thread_id": thread_id, 
        "tenant_id": tenant_id, 
        "workflow_id": workflow_id,
        "status": "failed", 
        "error": str(error), 
        "events": executor.execution_events.get(run_id, []),
        "output": state.get("runtime", {}).get("output", {}),
        "state": state,
    }
