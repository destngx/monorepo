from typing import Any, Dict
from .parser import WorkflowParser
from .nodes import build_entry_node, build_agent_node, build_branch_node, build_exit_node
from .models import WorkflowParseError

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
            source = edge_dict.get("source")
            target = edge_dict.get("target")
            condition = edge_dict.get("condition")
            edges.append({"source": source, "target": target, "condition": condition})

        return {"nodes": nodes_by_id, "edges": edges, "definition": workflow_dict}
