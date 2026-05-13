import logging
from typing import Any, Dict, Mapping, Optional

logger = logging.getLogger(__name__)

def evaluate_condition(condition: str, state: Mapping[str, Any], get_state_value_cb) -> bool:
    """Evaluates a transition condition against the state."""
    if not condition:
        return True
    try:
        for op in ["==", "!=", ">=", "<=", ">", "<"]:
            if op in condition:
                parts = condition.split(op)
                if len(parts) == 2:
                    left_path = parts[0].strip()
                    right_val = parts[1].strip().strip("'").strip('"')
                    
                    left_val = get_state_value_cb(left_path, state)
                    if left_val is None:
                        return False
                        
                    def parse_val(v):
                        if isinstance(v, str):
                            vl = v.lower()
                            if vl == 'true': return True
                            if vl == 'false': return False
                            try:
                                if '.' in v: return float(v)
                                return int(v)
                            except ValueError:
                                return v
                        return v

                    l_val = parse_val(left_val)
                    r_val = parse_val(right_val)

                    try:
                        if op == "==": return l_val == r_val
                        if op == "!=": return l_val != r_val
                        if op == ">": return l_val > r_val
                        if op == "<": return l_val < r_val
                        if op == ">=": return l_val >= r_val
                        if op == "<=": return l_val <= r_val
                    except TypeError:
                        ls, rs = str(l_val), str(r_val)
                        if op == "==": return ls == rs
                        if op == "!=": return ls != rs
                        if op == ">": return ls > rs
                        if op == "<": return ls < rs
                        if op == ">=": return ls >= rs
                        if op == "<=": return ls <= rs
        return True
    except Exception as e:
        logger.error(f"Error evaluating condition '{condition}': {e}")
        return False

def normalize_workflow(workflow: Dict[str, Any]) -> Dict[str, Any]:
    """Ensures the workflow dictionary has the expected structure."""
    if "definition" in workflow:
        workflow = workflow["definition"]
    elif "workflow" in workflow:
        workflow = workflow["workflow"]
    return workflow

def find_entry_node(workflow: Dict[str, Any]) -> Optional[str]:
    """Finds the ID of the entry node."""
    if "entry_point" in workflow:
        return workflow["entry_point"]
    
    for node in workflow.get("nodes", []):
        if node.get("type") == "entry":
            return node.get("id")
    return None

def find_node(workflow: Dict[str, Any], node_id: str) -> Optional[Dict[str, Any]]:
    """Finds a node by its ID."""
    for node in workflow.get("nodes", []):
        if node.get("id") == node_id:
            return node
    return None

def find_exit_node(workflow: Dict[str, Any]) -> Optional[str]:
    """Finds the ID of the exit node."""
    if "exit_point" in workflow:
        return workflow["exit_point"]
    
    for node in workflow.get("nodes", []):
        if node.get("type") == "exit":
            return node.get("id")
    return None

def get_field(node: Dict[str, Any], name: str, default: Any = None) -> Any:
    """Gets a field from a node, checking both root level and config level."""
    config = node.get("config", {})
    return node.get(name) or config.get(name) or default
