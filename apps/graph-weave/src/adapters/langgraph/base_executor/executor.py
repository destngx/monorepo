import logging
from typing import Any, Dict, List, Mapping, Optional

from .state_utils import get_state_value, resolve_path, handle_function_mapping
from .content_utils import clean_filler, interpolate_prompt
from .workflow_utils import evaluate_condition, normalize_workflow, find_entry_node, find_node, find_exit_node, get_field

logger = logging.getLogger(__name__)

class BaseLangGraphExecutor:
    """Base class for LangGraph executors with common utilities."""
    
    def __init__(self, ai_provider=None, tools=None, config=None):
        self.ai_provider = ai_provider
        self.tools = tools or {}
        self.config = config

    # --- Backward compatibility wrappers and internal helpers ---

    def _get_state_value(self, path: Any, state: Mapping[str, Any]) -> Any:
        return get_state_value(
            path, 
            state, 
            handle_function_mapping_cb=self._handle_function_mapping
        )

    def _resolve_path(self, current: Any, keys: List[str]) -> Any:
        return resolve_path(current, keys)

    def _handle_function_mapping(self, mapping: Dict[str, Any], state: Mapping[str, Any]) -> Any:
        return handle_function_mapping(
            mapping, 
            state, 
            get_state_value_cb=self._get_state_value
        )

    def _clean_filler(self, content: str) -> str:
        return clean_filler(content)

    def _interpolate_prompt(
        self, 
        template: str, 
        state: Mapping[str, Any], 
        local_context: Optional[Dict[str, Any]] = None
    ) -> str:
        return interpolate_prompt(
            template, 
            state, 
            get_state_value_cb=self._get_state_value,
            local_context=local_context
        )

    def _evaluate_condition(self, condition: str, state: Mapping[str, Any]) -> bool:
        return evaluate_condition(
            condition, 
            state, 
            get_state_value_cb=self._get_state_value
        )

    def _normalize_workflow(self, workflow: Dict[str, Any]) -> Dict[str, Any]:
        return normalize_workflow(workflow)

    def _find_entry_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        return find_entry_node(workflow)

    def _find_node(self, workflow: Dict[str, Any], node_id: str) -> Optional[Dict[str, Any]]:
        return find_node(workflow, node_id)

    def _find_exit_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        return find_exit_node(workflow)

    def _get_field(self, node: Dict[str, Any], name: str, default: Any = None) -> Any:
        return get_field(node, name, default)
