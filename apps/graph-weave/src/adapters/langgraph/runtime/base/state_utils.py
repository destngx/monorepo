import logging
from typing import Any, Dict, Mapping, List

from src.adapters.langgraph.runtime.state.resolver import StateResolver

logger = logging.getLogger(__name__)

def get_state_value(path: Any, state: Mapping[str, Any], handle_function_mapping_cb=None) -> Any:
    """
    Extracts a value from the state using a dot-notated path or JSONPath.
    Now standardized on StateResolver under the hood to ensure 100% consistent resolution.
    """
    if not path:
        return None

    # Handle complex function mappings
    if isinstance(path, dict):
        if path.get("type") == "function" and handle_function_mapping_cb:
            return handle_function_mapping_cb(path, state)
        return None

    # Initialize standard StateResolver and resolve using the unified logic
    resolver = StateResolver(state)
    return resolver.resolve(path, required=False)


def resolve_path(current: Any, keys: List[str]) -> Any:
    """Helper to resolve a nested path in an object. Delegates to StateResolver."""
    resolver = StateResolver({})
    return resolver._resolve_path(current, list(keys))


def handle_function_mapping(mapping: Dict[str, Any], state: Mapping[str, Any], get_state_value_cb) -> Any:
    """Executes a transformation function for complex mappings."""
    func_name = mapping.get("function")
    args = mapping.get("args", {})
    
    if func_name == "str_replace":
        val = get_state_value_cb(args.get("value"), state)
        if val is None:
            return None
        search_str = args.get("search", "")
        replace_str = args.get("replace", "")
        return str(val).replace(search_str, replace_str)
        
    elif func_name == "replace_null_with_empty":
        val = get_state_value_cb(args.get("value"), state)
        return val if val is not None else ""
        
    elif func_name == "array_first":
        val = get_state_value_cb(args.get("array"), state)
        if isinstance(val, list) and len(val) > 0:
            return val[0]
        return None
        
    elif func_name == "array_join":
        val = get_state_value_cb(args.get("array"), state)
        sep = args.get("separator", ", ")
        if isinstance(val, list):
            return sep.join(str(i) for i in val)
        return str(val) if val is not None else ""

    logger.warning(f"Unknown function mapping: {func_name}")
    return None
