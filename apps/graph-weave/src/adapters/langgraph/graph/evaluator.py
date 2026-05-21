from typing import Any, Dict
from .models import WorkflowParseError
from ..runtime.base.workflow_utils import evaluate_condition as runtime_evaluate_condition
from ..runtime.state import StateResolver

class EdgeEvaluator:
    """Evaluate JSONPath-based edge conditions."""

    @staticmethod
    def evaluate(condition_str: str, state: Dict[str, Any]) -> bool:
        """Evaluate condition against state."""
        if not condition_str:
            return True

        try:
            resolver = StateResolver(state)
            return runtime_evaluate_condition(
                condition_str,
                state,
                get_state_value_cb=lambda path, _: resolver.resolve(path, required=False),
            )
        except Exception as e:
            raise WorkflowParseError(
                f"Invalid edge condition '{condition_str}': {str(e)}"
            )
