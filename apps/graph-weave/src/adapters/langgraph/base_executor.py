import json
import logging
import re
import shlex
from typing import Any, Dict, List, Mapping, Optional

logger = logging.getLogger(__name__)

class BaseLangGraphExecutor:
    """Base class for LangGraph executors with common utilities."""
    
    def __init__(self, ai_provider=None, tools=None):
        self.ai_provider = ai_provider
        self.tools = tools or {}

    def _get_state_value(self, path: Any, state: Mapping[str, Any]) -> Any:
        """
        Extracts a value from the state using a dot-notated path or JSONPath.
        Supports:
        1. $.node_id.field
        2. $.field (from workflow_state)
        3. Simple field name
        4. Array indices: summary[0], metadata.tags[1]
        5. Function mappings: {"type": "function", "function": "...", "args": {...}}
        """
        if not path:
            return None

        # Handle complex function mappings
        if isinstance(path, dict):
            if path.get("type") == "function":
                return self._handle_function_mapping(path, state)
            return None

        if not isinstance(path, str):
            return None

        # Clean path
        clean_path = path
        if clean_path.startswith("$."):
            clean_path = clean_path[2:]
        
        # Handle common JS-isms like tags.join(",") or summary.first()
        virtual_transform = None
        if ".join(" in clean_path:
            clean_path = clean_path.split(".join(")[0]
            virtual_transform = "joined"
        elif ".first(" in clean_path or clean_path.endswith(".first"):
            clean_path = clean_path.split(".first")[0]
            virtual_transform = "first"
        elif ".sh_quote(" in clean_path or clean_path.endswith(".sh_quote"):
            clean_path = clean_path.split(".sh_quote")[0]
            virtual_transform = "sh_quote"
        elif ".shell(" in clean_path or clean_path.endswith(".shell"):
            clean_path = clean_path.split(".shell")[0]
            virtual_transform = "sh_quote"

        keys = clean_path.split('.')
        first_part = keys[0]
        
        # Handle virtual suffixes for lists/strings (the preferred syntax)
        if first_part.endswith("_joined"):
            virtual_transform = "joined"
            first_part = first_part[:-7]
        elif first_part.endswith("_first"):
            virtual_transform = "first"
            first_part = first_part[:-6]
        elif first_part.endswith("_shell"):
            virtual_transform = "sh_quote"
            first_part = first_part[:-6]

        # Handle array index in the first part, e.g., "summary[0]" -> "summary", ["0"]
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
        # 1. Try finding in node_results first (e.g. entry.query)
        if first_key in node_results:
            res = self._resolve_path(node_results[first_key], remaining_keys)
        # 2. Try workflow_state (the flat namespace)
        elif first_key in workflow_state:
            res = self._resolve_path(workflow_state[first_key], remaining_keys)
        # 3. Try root level (fallback)
        elif first_key in state:
            res = self._resolve_path(state[first_key], remaining_keys)

        # 4. Deep search in workflow_state and node_results (shorthand fallback for LLM)
        if res is None and "." not in clean_path:
            # Check if this key exists inside any of the top-level objects
            for container in [workflow_state, node_results]:
                for val in container.values():
                    if isinstance(val, dict) and first_key in val:
                        # Found the base key, now resolve the rest (including indices)
                        res = self._resolve_path(val[first_key], remaining_keys)
                        if res is not None:
                            break
                if res is not None: break

        # 5. Fallback: if resolution returned None but we have a terminal scalar
        #    at a parent path, return that scalar. This handles the common case where
        #    LLMs return flat strings like "CREATED: path/to/file" in `result` instead
        #    of nested objects like {"draft_path": "path/to/file"}.
        #    E.g. $.create_draft.result.draft_path -> create_draft.result is "path/..."
        if res is None and remaining_keys:
            # Try resolving without the last key segment(s)
            for trim in range(1, len(remaining_keys) + 1):
                parent_keys = remaining_keys[:-trim]
                parent_val = None
                if first_key in node_results:
                    parent_val = self._resolve_path(node_results[first_key], parent_keys)
                elif first_key in workflow_state:
                    parent_val = self._resolve_path(workflow_state[first_key], parent_keys)
                elif first_key in state:
                    parent_val = self._resolve_path(state[first_key], parent_keys)
                
                if parent_val is not None and isinstance(parent_val, str):
                    # The parent is a terminal string, return it as the value
                    logger.debug(f"Fallback: path '{path}' resolved parent to string: {parent_val[:80]}")
                    res = parent_val
                    break

        if res is not None:
            if virtual_transform == "joined" and isinstance(res, list):
                # Auto-join lists: authors/tags by comma, summary/claims by newline
                sep = ", " if any(k in first_key.lower() for k in ["tag", "author", "name"]) else "\n"
                res = sep.join(str(i) for i in res)
            elif virtual_transform == "first" and isinstance(res, list) and len(res) > 0:
                res = res[0]
            elif virtual_transform == "sh_quote":
                res = shlex.quote(str(res))
            
            return res

        return None

    def _resolve_path(self, current: Any, keys: List[str]) -> Any:
        """Helper to resolve a nested path in an object."""
        for key in keys:
            if current is None:
                return None
                
            # Handle explicit array index like "[0]"
            bracket_index_match = re.match(r"\[(\d+)\]", key)
            if bracket_index_match:
                index = int(bracket_index_match.group(1))
                if isinstance(current, list) and index < len(current):
                    current = current[index]
                else:
                    return None
                continue

            # Handle array index combined with name like "summary[0]"
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

    def _handle_function_mapping(self, mapping: Dict[str, Any], state: Mapping[str, Any]) -> Any:
        """Executes a transformation function for complex mappings."""
        func_name = mapping.get("function")
        args = mapping.get("args", {})
        
        if func_name == "str_replace":
            val = self._get_state_value(args.get("value"), state)
            if val is None:
                return None
            search_str = args.get("search", "")
            replace_str = args.get("replace", "")
            return str(val).replace(search_str, replace_str)
            
        elif func_name == "replace_null_with_empty":
            val = self._get_state_value(args.get("value"), state)
            return val if val is not None else ""
            
        elif func_name == "array_first":
            val = self._get_state_value(args.get("array"), state)
            if isinstance(val, list) and len(val) > 0:
                return val[0]
            return None
            
        elif func_name == "array_join":
            val = self._get_state_value(args.get("array"), state)
            sep = args.get("separator", ", ")
            if isinstance(val, list):
                return sep.join(str(i) for i in val)
            return str(val) if val is not None else ""

        logger.warning(f"Unknown function mapping: {func_name}")
        return None

    def _clean_filler(self, content: str) -> str:
        """
        Removes common LLM filler/preamble text from responses.
        Ensures we only get the core content (usually JSON or markdown).
        """
        if not content or not isinstance(content, str):
            return content or ""
        
        # Strip common filler patterns
        lines = content.strip().split('\n')
        skip_prefixes = (
            "Sure", "Here is", "Here's", "Certainly", "Of course",
            "I'll", "Let me", "Below is", "The following", "Based on",
            "OK", "Okay",
        )
        
        # Skip leading filler lines
        start = 0
        for i, line in enumerate(lines):
            stripped = line.strip()
            if stripped:
                if any(stripped.lower().startswith(p.lower()) for p in skip_prefixes) and not stripped.startswith("```"):
                    continue
                start = i
                break
        
        # Skip trailing filler
        end = len(lines)
        for i in range(len(lines) - 1, start, -1):
            stripped = lines[i].strip()
            if stripped:
                if any(stripped.lower().startswith(p.lower()) for p in ("let me know", "feel free", "i hope", "hope this")):
                    continue
                end = i + 1
                break
        
        return '\n'.join(lines[start:end]).strip()

    def _interpolate_prompt(
        self, 
        template: str, 
        state: Mapping[str, Any], 
        local_context: Optional[Dict[str, Any]] = None
    ) -> str:
        """
        Interpolates variables into a prompt template using the execution state.
        Supports {variable} and {{variable}} syntax.
        Only replaces placeholders that resolve to known values — unknown
        placeholders are left as-is to avoid destroying JSON literals in prompts.
        """
        if not template or not isinstance(template, str):
            return template

        result = template
        placeholders = re.findall(r"\{+([^{}]+)\}+", template)

        # Merge context: local_context > workflow_state > root state
        context = dict(state.get("workflow_state", {}))
        context.update({k: v for k, v in state.items() if k != "workflow_state"})
        if local_context:
            context.update(local_context)

        # Collect the set of known variable names from input_mapping / local_context
        known_vars = set()
        if local_context:
            known_vars.update(local_context.keys())

        for p in placeholders:
            # Try resolving using the robust _get_state_value logic
            # This handles $.prefix, array indices [0], and nested paths
            val = self._get_state_value(p, state)
            
            # Fallback to local_context if not found in state
            if val is None and local_context and p in local_context:
                val = local_context[p]
                
            # Fallback to context dictionary (workflow_state + state)
            if val is None and p in context:
                val = context[p]

            # CRITICAL: Only replace if we actually found a value or if the
            # placeholder is a known variable name (from input_mapping).
            # This prevents destroying JSON literals like {"status": "processed"}
            # in system prompts where "status" is not a workflow variable.
            if val is None and p not in known_vars and p not in context:
                # Unknown placeholder — leave it as-is in the template
                continue

            val_str = ""
            if val is not None:
                # Smart Content Extraction
                if isinstance(val, dict):
                    if "raw_response" in val:
                        val = val["raw_response"]
                    elif "result" in val:
                        val = val["result"]

                # Auto-join lists as human-readable strings for prompt interpolation
                # Tags/authors use comma, summaries/claims use newline
                if isinstance(val, list):
                    if any(hint in p.lower() for hint in ["tag", "author", "name"]):
                        val_str = ", ".join(str(i) for i in val)
                    else:
                        val_str = "\n".join(str(i) for i in val)
                elif isinstance(val, dict):
                    try:
                        val_str = json.dumps(val, indent=2)
                    except (TypeError, ValueError):
                        val_str = str(val)
                else:
                    val_str = str(val)

            # Replace ALL variations: {{p}} and {p}
            result = result.replace(f"{{{{{p}}}}}", val_str)
            result = result.replace(f"{{{p}}}", val_str)

        return result


    def _evaluate_condition(self, condition: str, state: Mapping[str, Any]) -> bool:
        """Evaluates a transition condition against the state."""
        if not condition:
            return True
        try:
            # Simple condition evaluation (e.g., status == 'success')
            # This is a placeholder for a more robust condition engine
            for op in ["==", "!=", ">=", "<=", ">", "<"]:
                if op in condition:
                    parts = condition.split(op)
                    if len(parts) == 2:
                        left_path = parts[0].strip()
                        right_val = parts[1].strip().strip("'").strip('"')
                        
                        left_val = self._get_state_value(left_path, state)
                        if left_val is None:
                            return False
                            
                        # Basic type conversion for comparison
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
                            # Fallback to string comparison if types are incompatible
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

    def _normalize_workflow(self, workflow: Dict[str, Any]) -> Dict[str, Any]:
        """Ensures the workflow dictionary has the expected structure."""
        if "definition" in workflow:
            # Unwrap if nested under definition (standard store format)
            workflow = workflow["definition"]
        elif "workflow" in workflow:
            # Unwrap if nested under workflow
            workflow = workflow["workflow"]
        return workflow

    def _find_entry_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        """Finds the ID of the entry node."""
        if "entry_point" in workflow:
            return workflow["entry_point"]
        
        for node in workflow.get("nodes", []):
            if node.get("type") == "entry":
                return node.get("id")
        return None

    def _find_node(self, workflow: Dict[str, Any], node_id: str) -> Optional[Dict[str, Any]]:
        """Finds a node by its ID."""
        for node in workflow.get("nodes", []):
            if node.get("id") == node_id:
                return node
        return None

    def _find_exit_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        """Finds the ID of the exit node."""
        if "exit_point" in workflow:
            return workflow["exit_point"]
        
        for node in workflow.get("nodes", []):
            if node.get("type") == "exit":
                return node.get("id")
        return None

    def _get_field(self, node: Dict[str, Any], name: str, default: Any = None) -> Any:
        """Gets a field from a node, checking both root level and config level."""
        config = node.get("config", {})
        return node.get(name) or config.get(name) or default
