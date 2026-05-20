"""
Workflow Validator Service

Validates WorkflowSpec or raw workflow definitions for structural integrity, 
executability, cycle safety (DAG), and node reachability.
"""

from typing import Dict, List, Set, Optional, Any, Union
from collections import defaultdict, deque
from ..models import WorkflowSpec, WorkflowNodeType


class WorkflowValidator:
    """Validates workflow specifications for structural integrity and cycle safety."""
    
    def validate(self, workflow: Union[WorkflowSpec, Dict[str, Any]]) -> Dict[str, Any]:
        """
        Validate workflow specification.
        
        Accepts either a WorkflowSpec Pydantic model or a raw definition dictionary.
        
        Returns:
            {
                "valid": bool,
                "errors": List[str],
                "warnings": List[str],
                "metadata": Dict
            }
        """
        errors = []
        warnings = []
        
        # Normalize to dictionary or model
        if isinstance(workflow, WorkflowSpec):
            wf_dict = workflow.model_dump()
            nodes = workflow.nodes
            edges = workflow.edges
            entry_point = workflow.entry_point
            exit_point = workflow.exit_point
        else:
            wf_dict = workflow
            nodes = workflow.get("nodes", [])
            edges = workflow.get("edges", [])
            entry_point = workflow.get("entry_point")
            exit_point = workflow.get("exit_point")

        # Check: Basic structure
        errors.extend(self._validate_structure(nodes, edges, entry_point, exit_point))
        
        if not errors:
            # Check: DAG properties (cycle detection)
            errors.extend(self._validate_dag_properties(nodes, edges))
            
            # Check: Reachability
            errors.extend(self._validate_reachability(nodes, edges, entry_point, exit_point))
        
        # Check: Node configuration
        warnings.extend(self._validate_node_config(nodes))
        
        node_types_count = {}
        for n in nodes:
            t = n.type if hasattr(n, "type") else n.get("type")
            t_str = t.value if hasattr(t, "value") else str(t)
            node_types_count[t_str] = node_types_count.get(t_str, 0) + 1
            
        return {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
            "metadata": {
                "node_count": len(nodes),
                "edge_count": len(edges),
                "node_types": node_types_count,
            }
        }
    
    def _validate_structure(
        self, 
        nodes: List[Any], 
        edges: List[Any], 
        entry_point: Optional[str], 
        exit_point: Optional[str]
    ) -> List[str]:
        """Validate basic workflow structure."""
        errors = []
        
        # Helper to get node id
        def get_node_id(node):
            return node.id if hasattr(node, "id") else node.get("id")
            
        # Helper to get edge endpoints
        def get_edge_endpoints(edge):
            source = edge.source if hasattr(edge, "source") else edge.get("source")
            target = edge.target if hasattr(edge, "target") else edge.get("target")
            return source, target

        # Check: At least one node
        if not nodes:
            errors.append("Workflow must have at least one node")
            return errors
        
        node_ids = [get_node_id(n) for n in nodes if get_node_id(n)]
        
        # Check: Entry point exists
        if not entry_point:
            errors.append("Workflow is missing an entry_point")
        elif entry_point not in node_ids:
            errors.append(f"Entry point '{entry_point}' does not exist in nodes")
        
        # Check: Exit point exists
        if not exit_point:
            errors.append("Workflow is missing an exit_point")
        elif exit_point not in node_ids:
            errors.append(f"Exit point '{exit_point}' does not exist in nodes")
        
        # Check: Node IDs are unique
        if len(node_ids) != len(set(node_ids)):
            errors.append("Node IDs must be unique")
        
        # Check: Edge endpoints exist
        node_id_set = set(node_ids)
        for edge in edges:
            source, target = get_edge_endpoints(edge)
            if not source or source not in node_id_set:
                errors.append(f"Edge source '{source}' does not exist")
            if not target or target not in node_id_set:
                errors.append(f"Edge target '{target}' does not exist")
        
        return errors
    
    def _validate_dag_properties(self, nodes: List[Any], edges: List[Any]) -> List[str]:
        """Validate that workflow is a valid DAG (no cycles)."""
        errors = []
        
        def get_node_id(node):
            return node.id if hasattr(node, "id") else node.get("id")
            
        def get_edge_endpoints(edge):
            source = edge.source if hasattr(edge, "source") else edge.get("source")
            target = edge.target if hasattr(edge, "target") else edge.get("target")
            return source, target
            
        # Build adjacency list
        graph = defaultdict(list)
        for edge in edges:
            source, target = get_edge_endpoints(edge)
            graph[source].append(target)
        
        # Check for cycles using DFS
        visited = set()
        rec_stack = set()
        
        def has_cycle(node_id):
            visited.add(node_id)
            rec_stack.add(node_id)
            
            for neighbor in graph[node_id]:
                if neighbor not in visited:
                    if has_cycle(neighbor):
                        return True
                elif neighbor in rec_stack:
                    return True
            
            rec_stack.remove(node_id)
            return False
        
        for node in nodes:
            node_id = get_node_id(node)
            if node_id not in visited:
                if has_cycle(node_id):
                    errors.append("Workflow contains a cycle")
                    break
        
        return errors
    
    def _validate_reachability(
        self, 
        nodes: List[Any], 
        edges: List[Any], 
        entry_point: str, 
        exit_point: str
    ) -> List[str]:
        """Validate that all nodes are reachable from entry_point and can reach exit_point."""
        errors = []
        
        def get_node_id(node):
            return node.id if hasattr(node, "id") else node.get("id")
            
        def get_edge_endpoints(edge):
            source = edge.source if hasattr(edge, "source") else edge.get("source")
            target = edge.target if hasattr(edge, "target") else edge.get("target")
            return source, target
            
        # Build forward and backward adjacency lists
        forward_graph = defaultdict(list)
        backward_graph = defaultdict(list)
        
        for edge in edges:
            source, target = get_edge_endpoints(edge)
            forward_graph[source].append(target)
            backward_graph[target].append(source)
        
        all_node_ids = {get_node_id(n) for n in nodes}
        
        # Check: All nodes reachable from entry_point
        reachable_from_start = self._bfs(entry_point, forward_graph)
        unreachable = all_node_ids - reachable_from_start
        
        if unreachable:
            errors.append(f"Nodes unreachable from entry_point: {unreachable}")
        
        # Check: All nodes can reach exit_point
        reachable_to_end = self._bfs(exit_point, backward_graph)
        unreachable_to_end = all_node_ids - reachable_to_end
        
        if unreachable_to_end:
            errors.append(
                f"Nodes cannot reach exit_point '{exit_point}': {unreachable_to_end}"
            )
        
        return errors
    
    def _validate_node_config(self, nodes: List[Any]) -> List[str]:
        """Validate node-specific configuration."""
        warnings = []
        
        def get_node_id(node):
            return node.id if hasattr(node, "id") else node.get("id")
            
        def get_node_type(node):
            t = node.type if hasattr(node, "type") else node.get("type")
            return t.value if hasattr(t, "value") else str(t)
            
        def get_node_field(node, field):
            return getattr(node, field) if hasattr(node, field) else node.get(field)
        
        for node in nodes:
            node_id = get_node_id(node)
            node_type = get_node_type(node)
            operator = get_node_field(node, "operator")
            timeout_seconds = get_node_field(node, "timeout_seconds")
            
            # Check: Tool nodes have operator
            if node_type == "tool_call" and not operator:
                warnings.append(f"Tool node '{node_id}' has no operator specified")
            
            # Check: Timeout is positive
            if timeout_seconds is not None and timeout_seconds <= 0:
                warnings.append(f"Node '{node_id}' has non-positive timeout")
        
        return warnings
    
    @staticmethod
    def _bfs(start: str, graph: Dict[str, List[str]]) -> Set[str]:
        """BFS to find all reachable nodes."""
        visited = set()
        queue = deque([start])
        visited.add(start)
        
        while queue:
            node = queue.popleft()
            for neighbor in graph[node]:
                if neighbor not in visited:
                    visited.add(neighbor)
                    queue.append(neighbor)
        
        return visited
