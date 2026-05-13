import time
import logging
from typing import Any, Dict, Optional
from .events import emit_event

logger = logging.getLogger(__name__)

def should_stop(
    executor: Any,
    run_id: str,
    tenant_id: str,
    start_time: float,
    timeout: int,
    detector: Any,
    state: Dict[str, Any]
) -> bool:
    elapsed = time.monotonic() - start_time
    if elapsed > timeout:
        emit_event(executor, run_id, "execution.timeout", {"elapsed_seconds": elapsed})
        state["status"] = "timeout"
        return True

    if check_kill_flag(executor, run_id, tenant_id):
        emit_event(executor, run_id, "execution.killed", {"elapsed_seconds": elapsed})
        state["status"] = "killed"
        return True

    if detector.is_stagnated():
        emit_event(executor, run_id, "execution.stagnated", {
            "visits": detector.get_summary(), 
            "max_hops": detector.max_hops
        })
        state["status"] = "stagnated"
        return True
    return False

def resolve_node_input(executor: Any, node: Dict[str, Any], state: Dict[str, Any]) -> Dict[str, Any]:
    node_type = node.get("type")
    if node_type in {"agent_node", "agent", "orchestrator", "cli_node", "bash"}:
        config = node.get("config", {})
        input_mapping = node.get("input_mapping") or config.get("input_mapping", {})
        if input_mapping:
            resolved = {}
            for key, path in input_mapping.items():
                resolved[key] = executor._get_state_value(path, state)
            return resolved
    elif node_type == "entry":
        return state.get("input", {})
    return {}

def check_kill_flag(executor: Any, run_id: str, tenant_id: str) -> bool:
    if not executor.redis_client:
        return False
    try:
        return executor.redis_client.exists(f"kill:{tenant_id}:{run_id}")
    except Exception:
        return False
