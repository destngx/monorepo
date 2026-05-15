from typing import Any, Dict
from .models import NodeConfig, WorkflowParseError, VALID_PROVIDERS, PROVIDER_MODELS, VALID_TOOLS


class WorkflowParser:
    @staticmethod
    def parse_workflow_json(workflow_dict: Dict[str, Any]) -> Dict[str, Any]:
        if not workflow_dict:
            raise WorkflowParseError("Workflow cannot be empty")

        nodes = workflow_dict.get("nodes", [])
        edges = workflow_dict.get("edges", [])

        if not nodes:
            raise WorkflowParseError("Workflow must have at least one node")

        has_node_refs = any("node_id" in n for n in nodes)
        has_entry_exit = any(n.get("type") in ("entry", "exit") for n in nodes)

        if not has_node_refs and not has_entry_exit:
            raise WorkflowParseError(
                "Invalid workflow format. Expected compositional format with "
                "alias/node_id references. Legacy embedded format not supported."
            )

        return WorkflowParser._parse_compositional(nodes, edges)

    @staticmethod
    def _parse_compositional(
        nodes: list, edges: list
    ) -> Dict[str, Any]:
        aliases = set()
        for node in nodes:
            if node.get("type") in ("entry", "exit"):
                continue
            if "alias" not in node:
                raise WorkflowParseError(f"Node missing 'alias': {node}")
            if "node_id" not in node:
                raise WorkflowParseError(f"Node missing 'node_id': {node}")
            aliases.add(node["alias"])

        for edge in edges:
            from_id = edge.get("from")
            to_id = edge.get("to")

            if from_id not in ("entry",) and from_id not in aliases:
                raise WorkflowParseError(
                    f"Edge 'from' references unknown alias: {from_id}"
                )

            if to_id not in ("exit",) and to_id not in aliases:
                raise WorkflowParseError(
                    f"Edge 'to' references unknown alias: {to_id}"
                )

        return {"nodes": nodes, "edges": edges}

    @staticmethod
    def extract_node_config(node_dict: Dict[str, Any]) -> NodeConfig:
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
