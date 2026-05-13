import re
import shlex
import json
import logging
from typing import Any, Dict, List, Mapping, Optional

logger = logging.getLogger(__name__)

def get_state_value(path: Any, state: Mapping[str, Any], handle_function_mapping_cb=None) -> Any:
    """
    Extracts a value from the state using a dot-notated path or JSONPath.
    """
    if not path:
        return None

    # Handle complex function mappings
    if isinstance(path, dict):
        if path.get("type") == "function" and handle_function_mapping_cb:
            return handle_function_mapping_cb(path, state)
        return None

    if not isinstance(path, str):
        return None

    # Clean path
    clean_path = path
    if clean_path.startswith("$."):
        clean_path = clean_path[2:]
    original_clean_path = clean_path
    
    # Handle common JS-isms like tags.join(",") or summary.first()
    virtual_transform = None
    if ".join(" in clean_path:
        clean_path = clean_path.split(".join(")[0]
        virtual_transform = "joined"
    elif clean_path.endswith("_joined"):
        clean_path = clean_path[:-7]
        virtual_transform = "joined"
    elif ".first(" in clean_path or clean_path.endswith(".first"):
        clean_path = clean_path.split(".first")[0]
        virtual_transform = "first"
    elif clean_path.endswith("_first"):
        clean_path = clean_path[:-6]
        virtual_transform = "first"
    elif ".sh_quote(" in clean_path or clean_path.endswith(".sh_quote"):
        clean_path = clean_path.split(".sh_quote")[0]
        virtual_transform = "sh_quote"
    elif clean_path.endswith("_shell"):
        clean_path = clean_path[:-6]
        virtual_transform = "sh_quote"
    elif ".json_quote(" in clean_path or clean_path.endswith(".json_quote"):
        clean_path = clean_path.split(".json_quote")[0]
        virtual_transform = "json_quote"
    elif ".json_escape(" in clean_path or clean_path.endswith(".json_escape"):
        clean_path = clean_path.split(".json_escape")[0]
        virtual_transform = "json_quote"
    elif clean_path.endswith("_json"):
        clean_path = clean_path[:-5]
        virtual_transform = "json_quote"
    elif ".shell(" in clean_path or clean_path.endswith(".shell"):
        clean_path = clean_path.split(".shell")[0]
        virtual_transform = "sh_quote"

    keys = clean_path.split('.')
    first_part = keys[0]
    
    # Handle virtual suffixes for lists/strings
    if first_part.endswith("_joined"):
        virtual_transform = "joined"
        first_part = first_part[:-7]
    elif first_part.endswith("_first"):
        virtual_transform = "first"
        first_part = first_part[:-6]
    elif first_part.endswith("_shell"):
        virtual_transform = "sh_quote"
        first_part = first_part[:-6]
    elif first_part.endswith("_json"):
        virtual_transform = "json_quote"
        first_part = first_part[:-5]

    # Handle array index in the first part, e.g., "summary[0]"
    array_match = re.match(r"([^\[]+)\[(\d+)\]", first_part)
    if array_match:
        first_key, first_index = array_match.groups()
        remaining_keys = [f"[{first_index}]"] + keys[1:]
    else:
        first_key = first_part
        remaining_keys = keys[1:]

    node_results = state.get("node_results", {})
    workflow_state = state.get("workflow_state", {})
    
    res = None
    # 1. Try finding in node_results first
    if first_key in node_results:
        res = resolve_path(node_results[first_key], remaining_keys)
    # 2. Try workflow_state
    elif first_key in workflow_state:
        res = resolve_path(workflow_state[first_key], remaining_keys)
    # 3. Try root level
    elif first_key in state:
        res = resolve_path(state[first_key], remaining_keys)

    # 4. Deep search
    if res is None and "." not in clean_path:
        for container in [workflow_state, node_results]:
            for val in container.values():
                if isinstance(val, dict) and first_key in val:
                    res = resolve_path(val[first_key], remaining_keys)
                    if res is not None:
                        break
            if res is not None: break

    # 5. Fallback for terminal scalars
    if res is None and remaining_keys:
        for trim in range(1, len(remaining_keys) + 1):
            parent_keys = remaining_keys[:-trim]
            parent_val = None
            if first_key in node_results:
                parent_val = resolve_path(node_results[first_key], parent_keys)
            elif first_key in workflow_state:
                parent_val = resolve_path(workflow_state[first_key], parent_keys)
            elif first_key in state:
                parent_val = resolve_path(state[first_key], parent_keys)
            
            if parent_val is not None and isinstance(parent_val, str):
                logger.debug(f"Fallback: path '{path}' resolved parent to string")
                res = parent_val
                break

    if res is not None:
        if virtual_transform == "joined" and isinstance(res, list):
            sep_key = original_clean_path.lower()
            sep = ", " if any(k in sep_key for k in ["tag", "author", "name"]) else "\n"
            res = sep.join(str(i) for i in res)
        elif virtual_transform == "first" and isinstance(res, list) and len(res) > 0:
            res = res[0]
        elif virtual_transform == "sh_quote":
            res = shlex.quote(str(res))
        elif virtual_transform == "json_quote":
            res = json.dumps(res)
        
        return res

    return None

def resolve_path(current: Any, keys: List[str]) -> Any:
    """Helper to resolve a nested path in an object."""
    for key in keys:
        if current is None:
            return None
            
        bracket_index_match = re.match(r"\[(\d+)\]", key)
        if bracket_index_match:
            index = int(bracket_index_match.group(1))
            if isinstance(current, list) and index < len(current):
                current = current[index]
            else:
                return None
            continue

        array_match = re.match(r"([^\[]+)\[(\d+)\]", key)
        if array_match:
            name, index = array_match.groups()
            index = int(index)
            if isinstance(current, dict) and name in current:
                current = current[name]
                if isinstance(current, list) and index < len(current):
                    current = current[index]
                else:
                    return None
            else:
                return None
        elif isinstance(current, dict) and key in current:
            current = current[key]
        elif isinstance(current, list) and key.isdigit():
            idx = int(key)
            if idx < len(current):
                current = current[idx]
            else:
                return None
        else:
            return None
    return current

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
