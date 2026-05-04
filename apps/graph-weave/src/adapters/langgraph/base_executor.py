import json
import logging
import re
from typing import Any, Dict, List, Mapping, Optional

logger = logging.getLogger(__name__)

class BaseLangGraphExecutor:
    """Base class for LangGraph executors with common utilities."""
    
    def __init__(self, ai_provider=None, tools=None):
        self.ai_provider = ai_provider
        self.tools = tools or {}

    def _get_state_value(self, path: str, state: Mapping[str, Any]) -> Any:
        """
        Extracts a value from the state using a dot-notated path or JSONPath.
        Supports:
        1. $.node_id.field
        2. $.field (from workflow_state)
        3. Simple field name
        4. Array indices: summary[0], metadata.tags[1]
        """
        if not path:
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

        keys = clean_path.split('.')
        first_part = keys[0]
        
        # Handle virtual suffixes for lists/strings (the preferred syntax)
        if first_part.endswith("_joined"):
            virtual_transform = "joined"
            first_part = first_part[:-7]
        elif first_part.endswith("_first"):
            virtual_transform = "first"
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

        if res is not None:
            if virtual_transform == "joined" and isinstance(res, list):
                # Auto-join lists: authors/tags by comma, summary/claims by newline
                sep = ", " if any(k in first_key.lower() for k in ["tag", "author", "name"]) else "\n"
                res = sep.join(str(i) for i in res)
            elif virtual_transform == "first" and isinstance(res, list) and len(res) > 0:
                res = res[0]
            
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

            if val is not None:
                # Smart Content Extraction
                if isinstance(val, dict):
                    if "raw_response" in val:
                        val = val["raw_response"]
                    elif "result" in val:
                        val = val["result"]

                # Convert complex objects to pretty JSON strings
                if isinstance(val, (dict, list)):
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
                        if right_val.lower() == 'true': right_val = True
                        elif right_val.lower() == 'false': right_val = False
                        elif right_val.isdigit(): right_val = int(right_val)
                        
                        if op == "==": return str(left_val) == str(right_val)
                        if op == "!=": return str(left_val) != str(right_val)
                        # ... handle others if needed
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
