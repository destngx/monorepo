"""
Workflow Specification Schema

Deterministic representation of a workflow DAG.
Generated from IntentExtraction via validators.
"""

from typing import Optional, List, Dict, Any, Literal, Union
from pydantic import BaseModel, Field
from enum import Enum


class NodeType(str, Enum):
    """Types of nodes in a workflow."""
    
    LLM_CALL = "llm_call"
    TOOL_CALL = "tool_call"
    CONDITIONAL = "conditional"
    PARALLEL = "parallel"
    LOOP = "loop"
    START = "start"
    END = "end"


class WorkflowNode(BaseModel):
    """A single node in the workflow DAG."""
    
    id: str = Field(..., description="Unique node identifier")
    type: NodeType = Field(..., description="Type of node")
    name: str = Field(..., description="Human-readable name")
    description: str = Field(..., description="What this node does")
    
    # Execution configuration
    operator: Optional[str] = Field(
        default=None,
        description="Tool/operator to invoke (e.g., 'web_search', 'llm_call')"
    )
    
    config: Dict[str, Any] = Field(
        default_factory=dict,
        description="Node-specific configuration"
    )
    
    # Resilience
    retry_policy: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Retry configuration"
    )
    
    timeout_seconds: Optional[int] = Field(
        default=None,
        description="Timeout for this node"
    )
    
    # Metadata
    tags: List[str] = Field(
        default_factory=list,
        description="Tags for categorization (e.g., 'external_api', 'deterministic')"
    )


class WorkflowEdge(BaseModel):
    """An edge connecting two nodes in the workflow DAG."""
    
    source: str = Field(..., description="Source node ID")
    target: str = Field(..., description="Target node ID")
    
    condition: Optional[str] = Field(
        default=None,
        description="Condition for traversing this edge (e.g., 'success', 'error')"
    )
    
    label: Optional[str] = Field(
        default=None,
        description="Human-readable label for this edge"
    )


class WorkflowSpec(BaseModel):
    """
    Complete workflow specification.
    
    This is a deterministic, validated DAG that can be:
    1. Converted to LangGraph StateGraph
    2. Serialized to YAML/JSON
    3. Persisted to database
    4. Executed with full auditability
    """
    
    id: str = Field(..., description="Unique workflow ID")
    name: str = Field(..., description="Workflow name")
    description: str = Field(..., description="Workflow description")
    version: str = Field(default="1.0.0", description="Workflow version")
    
    # DAG structure
    nodes: List[WorkflowNode] = Field(..., description="All nodes in the workflow")
    edges: List[WorkflowEdge] = Field(..., description="All edges in the workflow")
    
    # Entry/exit points
    start_node_id: str = Field(..., description="ID of the start node")
    end_node_ids: List[str] = Field(..., description="IDs of end nodes")
    
    # Metadata
    tags: List[str] = Field(
        default_factory=list,
        description="Workflow tags (e.g., 'data_processing', 'research')"
    )
    
    constraints: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Workflow-level constraints"
    )
    
    success_criteria: List[str] = Field(
        default_factory=list,
        description="Criteria for workflow success"
    )
    
    metadata: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Additional metadata"
    )
    
    def get_node(self, node_id: str) -> Optional[WorkflowNode]:
        """Get a node by ID."""
        for node in self.nodes:
            if node.id == node_id:
                return node
        return None
    
    def get_outgoing_edges(self, node_id: str) -> List[WorkflowEdge]:
        """Get all edges leaving a node."""
        return [e for e in self.edges if e.source == node_id]
    
    def get_incoming_edges(self, node_id: str) -> List[WorkflowEdge]:
        """Get all edges entering a node."""
        return [e for e in self.edges if e.target == node_id]
    
    def topological_sort(self) -> List[str]:
        """Return node IDs in topological order."""
        from collections import defaultdict, deque
        
        # Build adjacency list and in-degree count
        graph = defaultdict(list)
        in_degree = defaultdict(int)
        all_nodes = {n.id for n in self.nodes}
        
        for node_id in all_nodes:
            in_degree[node_id] = 0
        
        for edge in self.edges:
            graph[edge.source].append(edge.target)
            in_degree[edge.target] += 1
        
        # Kahn's algorithm
        queue = deque([n for n in all_nodes if in_degree[n] == 0])
        result = []
        
        while queue:
            node = queue.popleft()
            result.append(node)
            
            for neighbor in graph[node]:
                in_degree[neighbor] -= 1
                if in_degree[neighbor] == 0:
                    queue.append(neighbor)
        
        if len(result) != len(all_nodes):
            raise ValueError("Workflow contains a cycle")
        
        return result
