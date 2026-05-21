from typing import Any, Dict, Optional, List, TypedDict


class ExecutorState(TypedDict, total=False):
    input: Dict[str, Any]
    workflow: Dict[str, Any]
    nodes: Dict[str, Dict[str, Any]]
    runtime: Dict[str, Any]
    errors: List[Dict[str, Any]]
    step: int
    current_node: Optional[str]
    status: Optional[str]
    hop_count: int
    last_result: Optional[Dict[str, Any]]
