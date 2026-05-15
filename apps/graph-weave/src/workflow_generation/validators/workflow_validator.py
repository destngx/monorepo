"""
Workflow Validator

Validates WorkflowSpec for structural integrity and executability.
"""

from typing import Dict, List, Set, Optional, Any
from collections import defaultdict, deque
from ..schemas.workflow import WorkflowSpec, WorkflowNode, NodeType


class WorkflowValidator:
    """Validates WorkflowSpec for structural integrity."""
    
    def validate(self, workflow: WorkflowSpec) -> Dict[str, Any]:
        """
        Validate workflow specification.
        
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
        
        # Check: Basic structure
        errors.extend(self._validate_structure(workflow))
        
        # Check: DAG properties
        errors.extend(self._validate_dag_properties(workflow))
        
        # Check: Reachability
        errors.extend(self._validate_reachability(workflow))
        
        # Check: Node configuration
        warnings.extend(self._validate_node_config(workflow))
        
        return {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
            "metadata": {
                "node_count": len(workflow.nodes),
                "edge_count": len(workflow.edges),
                "node_types": {nt.value: sum(1 for n in workflow.nodes if n.type == nt) 
                              for nt in NodeType},
            }
        }
    
    def _validate_structure(self, workflow: WorkflowSpec) -> List[str]:
        """Validate basic workflow structure."""
        errors = []
        
        # Check: At least one node
        if not workflow.nodes:
            errors.append("Workflow must have at least one node")
        
        # Check: Start node exists
        if not any(n.id == workflow.start_node_id for n in workflow.nodes):
            errors.append(f"Start node '{workflow.start_node_id}' does not exist")
        
        # Check: All end nodes exist
        for end_id in workflow.end_node_ids:
            if not any(n.id == end_id for n in workflow.nodes):
                errors.append(f"End node '{end_id}' does not exist")
        
        # Check: Node IDs are unique
        node_ids = [n.id for n in workflow.nodes]
        if len(node_ids) != len(set(node_ids)):
            errors.append("Node IDs must be unique")
        
        # Check: Edge endpoints exist
        node_id_set = {n.id for n in workflow.nodes}
        for edge in workflow.edges:
            if edge.source not in node_id_set:
                errors.append(f"Edge source '{edge.source}' does not exist")
            if edge.target not in node_id_set:
                errors.append(f"Edge target '{edge.target}' does not exist")
        
        return errors
    
    def _validate_dag_properties(self, workflow: WorkflowSpec) -> List[str]:
        """Validate that workflow is a valid DAG (no cycles)."""
        errors = []
        
        # Build adjacency list
        graph = defaultdict(list)
        for edge in workflow.edges:
            graph[edge.source].append(edge.target)
        
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
        
        for node in workflow.nodes:
            if node.id not in visited:
                if has_cycle(node.id):
                    errors.append("Workflow contains a cycle")
                    break
        
        return errors
    
    def _validate_reachability(self, workflow: WorkflowSpec) -> List[str]:
        """Validate that all nodes are reachable from start and can reach an end."""
        errors = []
        
        # Build forward and backward adjacency lists
        forward_graph = defaultdict(list)
        backward_graph = defaultdict(list)
        
        for edge in workflow.edges:
            forward_graph[edge.source].append(edge.target)
            backward_graph[edge.target].append(edge.source)
        
        # Check: All nodes reachable from start
        reachable_from_start = self._bfs(workflow.start_node_id, forward_graph)
        unreachable = {n.id for n in workflow.nodes} - reachable_from_start
        
        if unreachable:
            errors.append(f"Nodes unreachable from start: {unreachable}")
        
        # Check: All nodes can reach an end
        for end_id in workflow.end_node_ids:
            reachable_to_end = self._bfs(end_id, backward_graph)
            unreachable_to_end = {n.id for n in workflow.nodes} - reachable_to_end
            
            if unreachable_to_end:
                errors.append(
                    f"Nodes cannot reach end '{end_id}': {unreachable_to_end}"
                )
        
        return errors
    
    def _validate_node_config(self, workflow: WorkflowSpec) -> List[str]:
        """Validate node-specific configuration."""
        warnings = []
        
        for node in workflow.nodes:
            # Check: Tool nodes have operator
            if node.type == NodeType.TOOL_CALL and not node.operator:
                warnings.append(f"Tool node '{node.id}' has no operator specified")
            
            # Check: Timeout is positive
            if node.timeout_seconds is not None and node.timeout_seconds <= 0:
                warnings.append(f"Node '{node.id}' has non-positive timeout")
        
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
