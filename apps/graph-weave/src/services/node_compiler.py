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
            return workflow

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

            config = stored_node.config.model_dump()
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

    def detect_cycles(self, workflow: dict) -> list[str]:
        """Detect cycles in workflow edges."""
        errors = []
        edges = workflow.get("edges", [])
        
        # Build adjacency list
        graph = {}
        for edge in edges:
            from_node = edge.get("from")
            to_node = edge.get("to")
            if from_node and to_node:
                graph.setdefault(from_node, []).append(to_node)
        
        # DFS cycle detection
        visited = set()
        rec_stack = set()
        path = []
        
        def dfs(node):
            visited.add(node)
            rec_stack.add(node)
            path.append(node)
            
            for neighbor in graph.get(node, []):
                if neighbor not in visited:
                    if dfs(neighbor):
                        return True
                elif neighbor in rec_stack:
                    # Cycle found
                    cycle_start = path.index(neighbor)
                    cycle = path[cycle_start:] + [neighbor]
                    errors.append(f"Cycle detected: {' → '.join(cycle)}")
                    return True
            
            path.pop()
            rec_stack.remove(node)
            return False
        
        for node in graph:
            if node not in visited:
                dfs(node)
        
        return errors

    def validate_dag_structure(self, workflow: dict) -> list[str]:
        """Validate workflow is a valid DAG (no cycles, all nodes reachable)."""
        errors = []
        
        # Check for cycles
        cycle_errors = self.detect_cycles(workflow)
        errors.extend(cycle_errors)
        
        # Check all nodes are reachable from entry
        nodes = workflow.get("nodes", [])
        edges = workflow.get("edges", [])
        
        # Find entry node
        entry_nodes = [n for n in nodes if n.get("type") == "entry"]
        if not entry_nodes:
            errors.append("No entry node found")
            return errors
        
        entry = entry_nodes[0].get("id") or entry_nodes[0].get("alias")
        
        # Build adjacency list
        graph = {}
        for edge in edges:
            from_node = edge.get("from")
            to_node = edge.get("to")
            if from_node and to_node:
                graph.setdefault(from_node, []).append(to_node)
        
        # BFS from entry
        visited = set()
        queue = [entry]
        while queue:
            node = queue.pop(0)
            if node in visited:
                continue
            visited.add(node)
            for neighbor in graph.get(node, []):
                queue.append(neighbor)
        
        # Check all non-entry/exit nodes are reachable
        for node in nodes:
            node_id = node.get("id") or node.get("alias")
            if node.get("type") not in ("entry", "exit") and node_id not in visited:
                errors.append(f"Node '{node_id}' is not reachable from entry")
        
        return errors
