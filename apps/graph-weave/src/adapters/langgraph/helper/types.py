from typing import Any, Dict, Optional, List, TypedDict

class ExecutorState(TypedDict):
    input: Dict[str, Any]
    step: int
    current_node: Optional[str]
    node_results: Dict[str, Dict[str, Any]]
    workflow_state: Dict[str, Any]
    status: Optional[str]
    hop_count: int
    last_result: Optional[Dict[str, Any]]
    errors: List[Dict[str, Any]]
