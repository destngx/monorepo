from typing import Any, Dict
from .models import WorkflowParseError

try:
    from jsonpath_ng import parse as jsonpath_parse
except ImportError:
    jsonpath_parse = None

class EdgeEvaluator:
    """Evaluate JSONPath-based edge conditions."""

    @staticmethod
    def evaluate(condition_str: str, state: Dict[str, Any]) -> bool:
        """Evaluate condition against state."""
        if not condition_str:
            return True

        try:
            if ">=" in condition_str:
                parts = condition_str.split(">=")
                path, value = parts[0].strip(), parts[1].strip()
                result = eval_jsonpath(path, state)
                return result >= parse_value(value)
            elif ">" in condition_str:
                parts = condition_str.split(">")
                path, value = parts[0].strip(), parts[1].strip()
                result = eval_jsonpath(path, state)
                return result > parse_value(value)
            elif "<=" in condition_str:
                parts = condition_str.split("<=")
                path, value = parts[0].strip(), parts[1].strip()
                result = eval_jsonpath(path, state)
                return result <= parse_value(value)
            elif "<" in condition_str:
                parts = condition_str.split("<")
                path, value = parts[0].strip(), parts[1].strip()
                result = eval_jsonpath(path, state)
                return result < parse_value(value)
            elif "==" in condition_str:
                parts = condition_str.split("==")
                path, value = parts[0].strip(), parts[1].strip()
                result = eval_jsonpath(path, state)
                return result == parse_value(value)
            elif "!=" in condition_str:
                parts = condition_str.split("!=")
                path, value = parts[0].strip(), parts[1].strip()
                result = eval_jsonpath(path, state)
                return result != parse_value(value)
            else:
                return True
        except Exception as e:
            raise WorkflowParseError(
                f"Invalid edge condition '{condition_str}': {str(e)}"
            )

def eval_jsonpath(path: str, state: Dict[str, Any]) -> Any:
    """Evaluate JSONPath expression."""
    if not path.startswith("$"):
        raise WorkflowParseError(f"JSONPath must start with $, got {path}")

    if not jsonpath_parse:
        raise WorkflowParseError("jsonpath-ng not installed")

    try:
        expr = jsonpath_parse(path)
        matches = expr.find(state)
        if not matches:
            return None
        return matches[0].value
    except Exception as e:
        raise WorkflowParseError(f"Invalid JSONPath '{path}': {str(e)}")

def parse_value(value_str: str) -> Any:
    """Parse value string to appropriate type."""
    value_str = value_str.strip()
    if value_str.startswith("'") and value_str.endswith("'"):
        return value_str[1:-1]
    if value_str.startswith('"') and value_str.endswith('"'):
        return value_str[1:-1]
    if value_str.lower() in ("true", "false"):
        return value_str.lower() == "true"
    try:
        if "." in value_str:
            return float(value_str)
        return int(value_str)
    except ValueError:
        return value_str
