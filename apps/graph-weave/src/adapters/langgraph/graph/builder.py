from typing import Any, Dict
from .parser import WorkflowParser
from .nodes import build_entry_node, build_agent_node, build_branch_node, build_exit_node
from .models import WorkflowParseError
from ...node import RedisNodeStore
from ....services.node_compiler import WorkflowCompiler, WorkflowFormatError, WorkflowCompileError, WorkflowValidationError


class GraphBuilder:
    def __init__(self, node_store: RedisNodeStore = None):
        self.node_store = node_store
        self.compiler = WorkflowCompiler(node_store) if node_store else None

    async def build(self, workflow_dict: Dict[str, Any]) -> Dict[str, Any]:
        if self.compiler:
            try:
                workflow_dict = await self.compiler.compile(workflow_dict)
                contract_errors = await self.compiler.validate_contracts(workflow_dict)
                if contract_errors:
                    raise WorkflowValidationError(
                        f"Contract validation failed: {contract_errors}"
                    )
            except WorkflowFormatError:
                pass

        parsed = WorkflowParser.parse_workflow_json(workflow_dict)

        nodes_by_id = {}
        for node_dict in parsed["nodes"]:
            node_id = node_dict.get("id")
            node_type = node_dict.get("type")

            if node_type == "entry":
                nodes_by_id[node_id] = build_entry_node({})
            elif node_type in {"agent_node", "agent"}:
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
            source = edge_dict.get("source") or edge_dict.get("from")
            target = edge_dict.get("target") or edge_dict.get("to")
            condition = edge_dict.get("condition")
            edges.append({"source": source, "target": target, "condition": condition})

        return {"nodes": nodes_by_id, "edges": edges, "definition": workflow_dict}

    @staticmethod
    def build_sync(workflow_dict: Dict[str, Any]) -> Dict[str, Any]:
        parsed = WorkflowParser.parse_workflow_json(workflow_dict)

        nodes_by_id = {}
        for node_dict in parsed["nodes"]:
            node_id = node_dict.get("id")
            node_type = node_dict.get("type")

            if node_type == "entry":
                nodes_by_id[node_id] = build_entry_node({})
            elif node_type in {"agent_node", "agent"}:
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
            source = edge_dict.get("source") or edge_dict.get("from")
            target = edge_dict.get("target") or edge_dict.get("to")
            condition = edge_dict.get("condition")
            edges.append({"source": source, "target": target, "condition": condition})

        return {"nodes": nodes_by_id, "edges": edges, "definition": workflow_dict}
