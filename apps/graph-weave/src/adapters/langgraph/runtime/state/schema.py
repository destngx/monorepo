from typing import Any, Dict, List, Literal, Optional, TypedDict


class RuntimeState(TypedDict, total=False):
    status: Optional[str]
    step: int
    hop_count: int
    current_node: Optional[str]
    last_node: Optional[str]


class NodeResultPayload(TypedDict, total=False):
    status: Literal["completed", "failed", "skipped"]
    result: Any
    outputs: Dict[str, Any]
    error: Optional[str]
    metadata: Dict[str, Any]


class ExecutorState(TypedDict):
    input: Dict[str, Any]
    workflow: Dict[str, Any]
    nodes: Dict[str, NodeResultPayload]
    runtime: RuntimeState
    errors: List[Dict[str, Any]]
