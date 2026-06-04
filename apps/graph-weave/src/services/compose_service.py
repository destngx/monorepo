"""
Workflow Composition Service

Handles deterministic stitching and composition of modular catalog nodes into
fully realized compositional WorkflowSpec definitions.
"""

import re
import logging
from typing import Dict, Any, List, Optional, Set
from ..modules.shared.deps import get_node_store

logger = logging.getLogger(__name__)


class WorkflowComposeService:
    """Service to dynamically compose and stitch modular workflows."""

    @staticmethod
    async def compose(
        tenant_id: str,
        skeleton: Dict[str, Any],
        node_map: Dict[str, str]
    ) -> Dict[str, Any]:
        """
        Dynamically stitches and composes modular nodes into a valid Workflow definition.
        
        Args:
            tenant_id: The namespace / tenant ID of the catalog.
            skeleton: The workflow DAG skeleton (structure + edges).
            node_map: Dict mapping step aliases to registered catalog node IDs.
            
        Returns:
            The fully composed, stitched workflow definition dictionary.
        """
        node_store = get_node_store(tenant_id)
        
        # 1. Fetch resolved node definitions from catalog
        node_defs = {}
        for step in skeleton.get("steps", []):
            alias = step.get("id")
            if alias in ("entry", "exit") or step.get("type") in ("entry", "exit"):
                continue
            
            node_id = node_map.get(alias)
            if not node_id:
                raise ValueError(f"Step '{alias}' is missing a registered node ID mapping")
                
            node_def = await node_store.get(node_id)
            if not node_def:
                raise ValueError(f"Mapped node '{node_id}' for step '{alias}' not found in catalog")
            
            node_defs[alias] = node_def.model_dump()

        # 2. Build list of produced fields per step
        produced_fields = {
            "entry": ["file_path", "file_content"]
        }
        for step in skeleton.get("steps", []):
            alias = step.get("id")
            if alias in ("entry", "exit") or step.get("type") in ("entry", "exit"):
                continue
            if alias in node_defs:
                output_contract = node_defs[alias].get("output_contract") or {}
                produced_fields[alias] = [p["name"] for p in output_contract.get("produced", [])]

        # 3. Form adjacency list and trace topological dependencies (ancestors)
        edges = skeleton.get("edges", [])
        predecessors = {}
        for edge in edges:
            frm = edge.get("from")
            to = edge.get("to")
            if to not in predecessors:
                predecessors[to] = []
            predecessors[to].append(frm)

        def get_all_ancestors(node_id: str) -> List[str]:
            """BFS/DFS to find all ancestor steps of a given step, in reverse topological order."""
            visited = set()
            ancestors = []
            queue = [node_id]
            while queue:
                current = queue.pop(0)
                parents = predecessors.get(current, [])
                for parent in parents:
                    if parent not in visited:
                        visited.add(parent)
                        ancestors.append(parent)
                        queue.append(parent)
            return ancestors

        # 4. Compose compositional nodes
        compositional_nodes = [
            {"id": "entry", "type": "entry", "display_name": "Entry"}
        ]
        step_order = ["entry"]
        step_indices = {step.get("id"): idx for idx, step in enumerate(skeleton.get("steps", []))}

        for step in skeleton.get("steps", []):
            alias = step.get("id")
            if alias in ("entry", "exit") or step.get("type") in ("entry", "exit"):
                continue
                
            node_id = node_map.get(alias)
            node_def = node_defs.get(alias, {})
            node_type = node_def.get("type", "agent_node")
            
            # Extract required parameters from template placeholders or fallback to input_contract
            config = node_def.get("config") or {}
            template = config.get("command", "") if node_type == "cli_node" else config.get("user_prompt_template", "")
            required = sorted(list(set(re.findall(r"\{([a-zA-Z0-9_]+?)(?:_shell)?\}", template))))
            
            if not required:
                input_contract = node_def.get("input_contract") or {}
                required = [r["name"] for r in input_contract.get("required", [])]

            # Construct dynamic input mapping for this step
            input_mapping = {}
            ancestors = get_all_ancestors(alias)
            
            # Sort ancestors so that nodes later in the execution flow (higher index) come first
            ancestors.sort(key=lambda a: step_indices.get(a, -1), reverse=True)
            
            if "entry" not in ancestors:
                ancestors.append("entry")

            for req_field in required:
                paths = []
                for prev in ancestors:
                    if req_field in produced_fields.get(prev, []):
                        if prev == "entry":
                            paths.append(f"$.input.{req_field}")
                        elif node_defs.get(prev, {}).get("type") == "cli_node":
                            if req_field in ("stdout", "stderr", "exit_code"):
                                paths.append(f"$.nodes.{prev}.outputs.{req_field}")
                            else:
                                paths.append(f"$.nodes.{prev}.result.{req_field}")
                        else:
                            paths.append(f"$.nodes.{prev}.result.{req_field}")
                
                if len(paths) == 1:
                    input_mapping[req_field] = paths[0]
                elif len(paths) > 1:
                    input_mapping[req_field] = paths
                else:
                    # Ultimate fallback to workflow global input
                    input_mapping[req_field] = f"$.input.{req_field}"

            # Assemble step node exactly as expected by the GraphWeave NodeCompiler
            compositional_nodes.append({
                "id": alias,
                "alias": alias,
                "node_id": node_id,
                "overrides": {
                    "input_mapping": input_mapping,
                    "output_key": alias
                }
            })
            step_order.append(alias)

        # 5. Construct exit node output mapping dynamically and generically
        exit_output_mapping = {}
        
        # Check if skeleton exit step has defined an output_mapping
        exit_step = next((s for s in skeleton.get("steps", []) if s.get("id") == "exit" or s.get("type") == "exit"), None)
        if exit_step:
            skeleton_mapping = exit_step.get("output_mapping") or exit_step.get("config", {}).get("output_mapping")
            if skeleton_mapping:
                exit_output_mapping = {}
                for key, val in skeleton_mapping.items():
                    if isinstance(val, str):
                        # Convert specific node result/outputs paths to generic wildcards/fallbacks
                        match = re.match(r"^\$\.nodes\.[a-zA-Z0-9_-]+\.(result|outputs)\.([a-zA-Z0-9_-]+)$", val)
                        if match:
                            ns, field = match.groups()
                            if field in ("stdout", "stderr", "exit_code"):
                                exit_output_mapping[key] = f"$.nodes.*.outputs.{field}"
                            else:
                                exit_output_mapping[key] = [
                                    f"$.nodes.*.result.{field}",
                                    f"$.input.{field}"
                                ]
                            continue
                    exit_output_mapping[key] = val
                
        # Generic Fallback: If no mapping is defined in the skeleton, build one dynamically.
        if not exit_output_mapping:
            for prev in reversed(step_order):
                produced = produced_fields.get(prev, [])
                for field in produced:
                    if field not in exit_output_mapping:
                        if field in ("stdout", "stderr", "exit_code"):
                            exit_output_mapping[field] = f"$.nodes.*.outputs.{field}"
                        else:
                            exit_output_mapping[field] = [
                                f"$.nodes.*.result.{field}",
                                f"$.input.{field}"
                            ]
                            

        exit_node = {
            "id": "exit",
            "type": "exit",
            "display_name": "Exit",
            "config": {
                "output_mapping": exit_output_mapping
            }
        }
        compositional_nodes.append(exit_node)

        # 6. Build the final workflow dictionary
        workflow_def = {
            "name": skeleton.get("name"),
            "version": skeleton.get("version"),
            "nodes": compositional_nodes,
            "edges": edges,
            "entry_point": "entry",
            "exit_point": "exit"
        }

        return workflow_def
