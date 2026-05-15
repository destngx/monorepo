from ..adapters.node import RedisNodeStore
from ..models.node import NodeResponse
from ..services.node_validator import NodeValidator


class WorkflowFormatError(Exception):
    pass


class WorkflowCompileError(Exception):
    pass


class WorkflowValidationError(Exception):
    pass


class WorkflowCompiler:
    def __init__(self, node_store: RedisNodeStore):
        self.node_store = node_store

    async def compile(self, workflow: dict) -> dict:
        nodes = workflow.get("nodes", [])

        if not nodes:
            return workflow

        has_node_refs = any("node_id" in n for n in nodes)

        if has_node_refs:
            return await self._compile_compositional(workflow)

        has_entry_exit = any(n.get("type") in ("entry", "exit") for n in nodes)
        if has_entry_exit:
            return await self._compile_compositional(workflow)

        raise WorkflowFormatError(
            "Legacy embedded workflow format not supported. "
            "Use compositional format with alias/node_id references."
        )

    async def _compile_compositional(self, workflow: dict) -> dict:
        compiled_nodes = []

        for node_ref in workflow.get("nodes", []):
            if node_ref.get("type") in ("entry", "exit"):
                compiled_nodes.append(node_ref)
                continue

            node_id = node_ref.get("node_id")
            alias = node_ref.get("alias")
            overrides = node_ref.get("overrides", {})

            if not node_id:
                raise WorkflowCompileError(
                    f"Node reference missing node_id: {node_ref}"
                )

            stored_node = await self.node_store.get(node_id)
            if not stored_node:
                raise WorkflowCompileError(f"Node {node_id} not found in registry")

            config = stored_node.config.dict()
            input_mapping = overrides.get("input_mapping", {})
            if input_mapping:
                config["input_mapping"] = input_mapping

            if "output_key" in overrides:
                config["output_key"] = overrides["output_key"]

            compiled_node = {
                "id": alias,
                "type": stored_node.type,
                "display_name": stored_node.name,
                "description": stored_node.description,
                "config": config,
            }
            compiled_nodes.append(compiled_node)

        return {**workflow, "nodes": compiled_nodes}

    async def validate_references(self, workflow: dict) -> list[str]:
        errors = []
        for node_ref in workflow.get("nodes", []):
            if "node_id" in node_ref:
                if not await self.node_store.exists(node_ref["node_id"]):
                    errors.append(f"Node {node_ref['node_id']} not found")
        return errors

    async def validate_contracts(self, workflow: dict) -> list[str]:
        errors = []
        validator = NodeValidator()

        nodes_by_alias = {}
        for node_ref in workflow.get("nodes", []):
            alias = node_ref.get("alias") or node_ref.get("id")
            if "node_id" in node_ref:
                stored_node = await self.node_store.get(node_ref["node_id"])
                if stored_node:
                    nodes_by_alias[alias] = stored_node

        for edge in workflow.get("edges", []):
            from_node = nodes_by_alias.get(edge["from"])
            to_node = nodes_by_alias.get(edge["to"])

            if from_node and to_node:
                edge_errors = validator.validate_contract_compatibility(
                    from_node, to_node
                )
                for err in edge_errors:
                    errors.append(f"Edge {edge['from']}→{edge['to']}: {err}")

        return errors
