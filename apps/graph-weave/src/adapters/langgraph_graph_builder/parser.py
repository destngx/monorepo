from typing import Any, Dict
from .models import NodeConfig, WorkflowParseError, VALID_PROVIDERS, PROVIDER_MODELS, VALID_TOOLS

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
