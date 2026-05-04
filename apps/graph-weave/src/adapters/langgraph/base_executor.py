import json
import re
from typing import Any, Dict, Optional, List, Mapping
from src.app_logging import get_logger

logger = get_logger(__name__)

class BaseLangGraphExecutor:
    """
    Base class for LangGraph executors containing shared utility methods.
    """

    def _normalize_workflow(self, workflow: Dict[str, Any]) -> Dict[str, Any]:
        """Normalize workflow format to ensure 'nodes' and 'edges' are available."""
        if "definition" in workflow:
            definition = workflow["definition"]
            # Create a copy to avoid mutating original
            normalized = workflow.copy()
            normalized["nodes"] = definition.get("nodes", [])
            normalized["edges"] = definition.get("edges", [])
            normalized["entry_point"] = definition.get("entry_point")
            normalized["exit_point"] = definition.get("exit_point")
            if "limits" in definition:
                normalized["limits"] = definition.get("limits", normalized.get("limits", {}))
            return normalized
        return workflow

    def _find_entry_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        # 1. Check explicit entry_point key
        entry_point = workflow.get("entry_point")
        if entry_point:
            return entry_point
            
        # 2. Check node types
        for node in workflow.get("nodes", []):
            if node.get("type") == "entry":
                return node.get("id")
                
        # 3. Check for node named "entry"
        for node in workflow.get("nodes", []):
            if node.get("id") == "entry":
                return "entry"
                
        return None

    def _find_exit_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        # 1. Check explicit exit_point key
        exit_point = workflow.get("exit_point")
        if exit_point:
            return exit_point
            
        # 2. Check node types
        for node in workflow.get("nodes", []):
            if node.get("type") == "exit":
                return node.get("id")
                
        # 3. Check for node named "exit"
        for node in workflow.get("nodes", []):
            if node.get("id") == "exit":
                return "exit"
                
        return None

    def _find_node(
        self, workflow: Dict[str, Any], node_id: str
    ) -> Optional[Dict[str, Any]]:
        for node in workflow.get("nodes", []):
            if node.get("id") == node_id:
                return node
        return None

    def _get_state_value(self, path: str, state: Mapping[str, Any]) -> Any:
        """
        Extract a value from the state based on a JSONPath-like string.
        Logic order:
        1. Node results (if path starts with a node_id, e.g., 'entry.intent')
        2. workflow_state (the application data pool)
        3. Root state (metadata like 'step', 'hop_count')
        """
        if not path:
            return None
        if path.startswith("$."):
            path = path[2:]

        keys = path.split(".")
        first_key = keys[0]

        # 1. Try finding in node_results first (e.g. $.entry.query)
        node_results = state.get("node_results", {})
        if first_key in node_results:
            current = node_results[first_key]
            if len(keys) == 1:
                return current
            
            found = True
            for key in keys[1:]:
                if isinstance(current, dict) and key in current:
                    current = current.get(key)
                else:
                    found = False
                    break
            if found:
                return current

        # 2. Try finding in workflow_state next (flat pool)
        workflow_state = state.get("workflow_state", {})
        if first_key in workflow_state:
            current = workflow_state
            found = True
            for key in keys:
                if isinstance(current, dict) and key in current:
                    current = current.get(key)
                else:
                    found = False
                    break
            if found:
                return current

        # 3. Fallback: Root state
        current = state
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current.get(key)
            else:
                return None

        return current

    def _interpolate_prompt(
        self,
        template: str,
        state: Mapping[str, Any],
        local_context: Optional[Dict[str, Any]] = None,
    ) -> str:
        """
        Interpolate variables into the prompt template.
        Supports:
        1. Local context (from input_mapping) - HIGHEST PRIORITY
        2. Root-level state keys like {step}, {hop_count}
        3. workflow_state keys like {query}, {subquestions}
        4. JSONPath-like keys starting with $.
        """
        if not template:
            return ""

        result = template

        # Support both {var} and {{var}}
        placeholders = re.findall(r"\{+([^{}]+)\}+", template)

        # Merge context: local_context > workflow_state > root state
        context = dict(state.get("workflow_state", {}))
        context.update({k: v for k, v in state.items() if k != "workflow_state"})
        if local_context:
            context.update(local_context)

        for p in placeholders:
            val = None
            if p.startswith("$."):
                val = self._get_state_value(p, state)
            elif "." in p:
                # Support dot notation for nested objects (e.g. metadata.title)
                keys = p.split('.')
                current = context
                found = True
                for key in keys:
                    if isinstance(current, dict) and key in current:
                        current = current[key]
                    else:
                        found = False
                        break
                
                if found and current is not None:
                    val = current
                else:
                    val = context.get(p)
            else:
                val = context.get(p)

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
        if not condition:
            return True
        try:
            for op in ["==", "!=", ">=", "<=", ">", "<"]:
                if op in condition:
                    left, right = condition.split(op, 1)
                    left_val = self._get_state_value(left.strip(), state)
                    right_val = right.strip().strip("'\"")
                    
                    if op == "==":
                        return str(left_val) == str(right_val)
                    elif op == "!=":
                        return str(left_val) != str(right_val)
                    
                    # Numeric comparisons
                    try:
                        f_left = float(left_val)
                        f_right = float(right_val)
                        if op == ">=": return f_left >= f_right
                        if op == "<=": return f_left <= f_right
                        if op == ">": return f_left > f_right
                        if op == "<": return f_left < f_right
                    except (ValueError, TypeError):
                        return False
            return True
        except Exception:
            return False

    def _clean_filler(self, text: str) -> str:
        """Strip common LLM conversational filler and greetings from a response."""
        if not text:
            return ""
        
        text = text.strip()
        if text.startswith("```"):
            lines = text.split("\n")
            if lines[0].startswith("```"):
                lines = lines[1:]
            if lines and lines[-1].strip() == "```":
                lines = lines[:-1]
            text = "\n".join(lines).strip()

        fillers = [
            "Hello!", "Certainly!", "Here is the result:", "Sure,", "I am an AI",
            "How can I assist", "I'd be happy to", "The requested output is:",
            "Here's the JSON", "As an AI assistant", "I have generated"
        ]
        
        lines = text.split("\n")
        cleaned_lines = []
        for line in lines:
            trimmed = line.strip()
            is_filler = False
            for f in fillers:
                if trimmed.lower().startswith(f.lower()):
                    is_filler = True
                    break
            if not is_filler:
                cleaned_lines.append(line)
        
        return "\n".join(cleaned_lines).strip()
