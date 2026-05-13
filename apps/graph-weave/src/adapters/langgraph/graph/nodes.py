from typing import Any, Dict, List, Optional, Callable, Protocol
from .models import NodeConfig
from .prompts import interpolate_prompt
from .evaluator import EdgeEvaluator

class AgentNodeFunc(Protocol):
    """Protocol for agent node functions with config attribute."""
    def __call__(self, state: Dict[str, Any]) -> Dict[str, Any]: ...
    config: NodeConfig

def build_entry_node(
    state: Dict[str, Any],
) -> Callable[[Dict[str, Any]], Dict[str, Any]]:
    """Build entry node (pass-through)."""
    def entry_func(input_data: Dict[str, Any]) -> Dict[str, Any]:
        return {**state, **input_data}
    return entry_func

def build_agent_node(
    node_config: NodeConfig, system_prompt: str, user_prompt_template: str
) -> Callable[[Dict[str, Any]], Dict[str, Any]]:
    """Build agent node with per-node configuration."""
    def agent_func(state: Dict[str, Any]) -> Dict[str, Any]:
        output_key = f"{state.get('current_node', 'agent')}_output"
        result = interpolate_prompt(user_prompt_template, state)
        return {**state, output_key: result}
    setattr(agent_func, "config", node_config)
    return agent_func

def build_branch_node(condition: str) -> Callable[[Dict[str, Any]], str]:
    """Build branch node with condition evaluation."""
    def branch_func(state: Dict[str, Any]) -> str:
        result = EdgeEvaluator.evaluate(condition, state)
        return "true_branch" if result else "false_branch"
    return branch_func

def build_exit_node(
    output_mapping: Optional[Dict[str, Any]] = None,
) -> Callable[[Dict[str, Any]], Dict[str, Any]]:
    """Build exit node."""
    def exit_func(state: Dict[str, Any]) -> Dict[str, Any]:
        return state
    return exit_func
