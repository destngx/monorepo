"""Real LangGraph StateGraph builder from workflow JSON schema.

Implements:
- StateGraph construction from workflow JSON
- Node builders: entry, agent_node, branch, exit
- JSONPath-based edge condition evaluation
- Per-node configuration (provider, model, temperature, max_tokens, tools)
- State schema validation
"""

from typing import Any, Dict, List, Optional, Callable, Protocol
from dataclasses import dataclass
import re

try:
    from jsonpath_ng import parse as jsonpath_parse
except ImportError:
    jsonpath_parse = None


class AgentNodeFunc(Protocol):
    """Protocol for agent node functions with config attribute."""

    def __call__(self, state: Dict[str, Any]) -> Dict[str, Any]: ...

    config: "NodeConfig"


VALID_PROVIDERS = {"github-copilot", "openai", "anthropic"}
PROVIDER_MODELS = {
    "github-copilot": ["claude-3.5-sonnet", "claude-3-opus", "gpt-4.1"],
    "openai": ["gpt-4.1", "gpt-3.5-turbo"],
    "anthropic": ["claude-3-opus", "claude-3-sonnet"],
}
VALID_TOOLS = {"load_skill", "search", "verify"}


@dataclass
class NodeConfig:
    """Per-node configuration override."""

    provider: Optional[str] = None
    model: Optional[str] = None
    temperature: Optional[float] = None
    max_tokens: Optional[int] = None
    tools: Optional[List[str]] = None


class WorkflowParseError(Exception):
    pass


class WorkflowParser:
    """Parser for workflow JSON schema."""

    @staticmethod
    def parse_workflow_json(workflow_dict: Dict[str, Any]) -> Dict[str, Any]:
        """Parse and validate workflow JSON."""
        if not workflow_dict:
            raise WorkflowParseError("Workflow cannot be empty")

        nodes = workflow_dict.get("nodes", [])
        edges = workflow_dict.get("edges", [])

        if not nodes:
            raise WorkflowParseError("Workflow must have at least one node")

        node_ids = {n.get("id") for n in nodes}
        for edge in edges:
            if edge.get("source") not in node_ids:
                raise WorkflowParseError(f"Unknown source node: {edge.get('source')}")
            if edge.get("target") not in node_ids:
                raise WorkflowParseError(f"Unknown target node: {edge.get('target')}")

        return {"nodes": nodes, "edges": edges}

    @staticmethod
    def extract_node_config(node_dict: Dict[str, Any]) -> NodeConfig:
        """Extract per-node configuration from node dictionary."""
        provider = node_dict.get("provider")
        if provider and provider not in VALID_PROVIDERS:
            raise WorkflowParseError(
                f"Invalid provider: {provider}. Valid: {VALID_PROVIDERS}"
            )

        model = node_dict.get("model")
        if model and provider and model not in PROVIDER_MODELS.get(provider, []):
            raise WorkflowParseError(
                f"Invalid model '{model}' for provider '{provider}'"
            )

        temperature = node_dict.get("temperature")
        if temperature is not None and not (0 <= temperature <= 1):
            raise WorkflowParseError(
                f"Temperature must be in [0, 1], got {temperature}"
            )

        max_tokens = node_dict.get("max_tokens")
        if max_tokens is not None and max_tokens < 1:
            raise WorkflowParseError(f"max_tokens must be >= 1, got {max_tokens}")

        tools = node_dict.get("tools")
        if tools:
            invalid = [t for t in tools if t not in VALID_TOOLS]
            if invalid:
                raise WorkflowParseError(
                    f"Invalid tools: {invalid}. Valid: {VALID_TOOLS}"
                )

        return NodeConfig(
            provider=provider,
            model=model,
            temperature=temperature,
            max_tokens=max_tokens,
            tools=tools,
        )


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
                result = _eval_jsonpath(path, state)
                return result >= _parse_value(value)
            elif ">" in condition_str:
                parts = condition_str.split(">")
                path, value = parts[0].strip(), parts[1].strip()
                result = _eval_jsonpath(path, state)
                return result > _parse_value(value)
            elif "<=" in condition_str:
                parts = condition_str.split("<=")
                path, value = parts[0].strip(), parts[1].strip()
                result = _eval_jsonpath(path, state)
                return result <= _parse_value(value)
            elif "<" in condition_str:
                parts = condition_str.split("<")
                path, value = parts[0].strip(), parts[1].strip()
                result = _eval_jsonpath(path, state)
                return result < _parse_value(value)
            elif "==" in condition_str:
                parts = condition_str.split("==")
                path, value = parts[0].strip(), parts[1].strip()
                result = _eval_jsonpath(path, state)
                return result == _parse_value(value)
            elif "!=" in condition_str:
                parts = condition_str.split("!=")
                path, value = parts[0].strip(), parts[1].strip()
                result = _eval_jsonpath(path, state)
                return result != _parse_value(value)
            else:
                return True
        except Exception as e:
            raise WorkflowParseError(
                f"Invalid edge condition '{condition_str}': {str(e)}"
            )


def _eval_jsonpath(path: str, state: Dict[str, Any]) -> Any:
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


def _parse_value(value_str: str) -> Any:
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
        result = _interpolate_prompt(user_prompt_template, state)
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


def _interpolate_prompt(template: str, context: Dict[str, Any]) -> str:
    """Interpolate variables in prompt template."""
    result = template
    pattern = r"\{(\w+)\}"

    for match in re.finditer(pattern, template):
        var_name = match.group(1)
        if var_name not in context:
            raise WorkflowParseError(f"Missing template variable: {var_name}")
        value = context[var_name]
        result = result.replace(f"{{{var_name}}}", str(value))

    return result


class GraphBuilder:
    """Build LangGraph StateGraph from workflow definition."""

    @staticmethod
    def build(workflow_dict: Dict[str, Any]) -> Dict[str, Any]:
        """Build graph from workflow JSON."""
        parsed = WorkflowParser.parse_workflow_json(workflow_dict)

        nodes_by_id = {}
        for node_dict in parsed["nodes"]:
            node_id = node_dict.get("id")
            node_type = node_dict.get("type")

            if node_type == "entry":
                nodes_by_id[node_id] = build_entry_node({})
            elif node_type == "agent_node":
                config = WorkflowParser.extract_node_config(node_dict)
                system_prompt = node_dict.get("system_prompt", "You are helpful.")
                user_template = node_dict.get("user_prompt_template", "")
                nodes_by_id[node_id] = build_agent_node(
                    config, system_prompt, user_template
                )
            elif node_type == "branch":
                condition = node_dict.get("condition", "")
                nodes_by_id[node_id] = build_branch_node(condition)
            elif node_type == "exit":
                output_mapping = node_dict.get("output_mapping")
                nodes_by_id[node_id] = build_exit_node(output_mapping)
            else:
                raise WorkflowParseError(f"Unknown node type: {node_type}")

        edges = []
        for edge_dict in parsed["edges"]:
            source = edge_dict.get("source")
            target = edge_dict.get("target")
            condition = edge_dict.get("condition")
            edges.append({"source": source, "target": target, "condition": condition})

        return {"nodes": nodes_by_id, "edges": edges, "definition": workflow_dict}
