import re
import logging
from typing import Any, Dict, List, Optional
from .events import emit_event

logger = logging.getLogger(__name__)

def route_by_edge(
    executor: Any,
    run_id: str,
    workflow: Dict[str, Any],
    current_node_id: str,
    state: Dict[str, Any]
) -> Optional[str]:
    edges = normalize_edges(workflow.get("edges", []))
    # Support both 'source' (standard) and 'from' (legacy) keys
    matching_edges = [e for e in edges if (e.get("source") or e.get("from")) == current_node_id]
    if not matching_edges:
        return executor._find_exit_node(workflow)

    target_node_id = None
    routing_edge = None
    for edge in matching_edges:
        condition = edge.get("condition")
        if not condition or executor._evaluate_condition(condition, state):
            # Support both 'target' (standard) and 'to' (legacy) keys
            target_node_id = edge.get("target") or edge.get("to")
            routing_edge = edge
            break

    if not target_node_id:
        target_node_id = executor._find_exit_node(workflow)
        emit_event(executor, run_id, "edge_route.unmatched", {
            "from": current_node_id,
            "to": target_node_id,
            "reason": "no_condition_matched",
        })
        return target_node_id

    entry_node_id = executor._find_entry_node(workflow)
    if target_node_id == entry_node_id and current_node_id != entry_node_id:
        exit_node_id = executor._find_exit_node(workflow)
        emit_event(executor, run_id, "edge_route.cycle_blocked", {
            "from": current_node_id,
            "blocked_to": target_node_id,
            "to": exit_node_id,
            "reason": "entry_reentry",
        })
        return exit_node_id

    if target_node_id:
        emit_event(executor, run_id, "edge_route", {
            "from": current_node_id, 
            "to": target_node_id, 
            "condition": routing_edge.get("condition") if routing_edge else None
        })
    return target_node_id

def normalize_edges(edges: Any) -> List[Dict[str, Any]]:
    if edges is None:
        return []
    if not isinstance(edges, list):
        raise ValueError("Workflow edges must be a list")

    if edges and all(is_node_id_pair(edge) for edge in edges):
        return [{"from": edge[0], "to": edge[1]} for edge in edges]

    flattened = flatten_edge_values(edges)
    if flattened and all(is_bare_node_id(edge) for edge in flattened):
        return [
            {"from": source, "to": target}
            for source, target in zip(flattened, flattened[1:])
        ]

    normalized: List[Dict[str, Any]] = []
    for edge in flattened:
        if isinstance(edge, str):
            normalized.append(parse_edge_string(edge))
            continue
        if not isinstance(edge, dict):
            raise ValueError(f"Workflow edge must be an object, got {type(edge).__name__}")
        normalized.append(edge)
    return normalized

def flatten_edge_values(edges: List[Any]) -> List[Any]:
    flattened: List[Any] = []
    pending = list(edges)
    while pending:
        edge = pending.pop(0)
        if isinstance(edge, list):
            pending = edge + pending
            continue
        flattened.append(edge)
    return flattened

def is_bare_node_id(value: Any) -> bool:
    return isinstance(value, str) and bool(re.match(r"^\s*[A-Za-z_][A-Za-z0-9_-]*\s*$", value))

def is_node_id_pair(value: Any) -> bool:
    return (
        isinstance(value, list)
        and len(value) == 2
        and all(is_bare_node_id(item) for item in value)
    )

def parse_edge_string(edge: str) -> Dict[str, Any]:
    match = re.match(r"^\s*([A-Za-z_][A-Za-z0-9_-]*)\s*(?:->|=>|to)\s*([A-Za-z_][A-Za-z0-9_-]*)\s*$", edge)
    if not match:
        raise ValueError(f"Workflow edge string must use 'from -> to' syntax, got {edge!r}")
    source, target = match.groups()
    return {"from": source, "to": target}
