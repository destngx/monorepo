from pydantic import BaseModel, Field, field_validator, model_validator
from typing import Optional, List, Dict, Any
from ...validation import validate_resource_id
from .validators import validate_workflow_id_format, validate_semantic_version

class WorkflowCreate(BaseModel):
    """Request model for creating a new workflow"""

    tenant_id: str = Field(
        default="default",
        description="Tenant identifier (e.g., default)",
        json_schema_extra={"example": "default"},
        min_length=1,
        max_length=128,
    )
    workflow_id: str = Field(
        ...,
        description="Workflow identifier in format 'name:vX.Y.Z' (e.g., workflow-generator:v1.0.0)",
        json_schema_extra={"example": "workflow-generator:v1.0.0"},
        min_length=1,
        max_length=128,
    )
    name: str = Field(
        ...,
        description="Human-readable workflow name",
        json_schema_extra={"example": "Workflow Generator"},
        min_length=1,
        max_length=256,
    )
    version: str = Field(
        ...,
        description="Semantic versioning X.Y.Z (must match version in workflow_id)",
        json_schema_extra={"example": "1.0.0"},
    )
    description: Optional[str] = Field(
        None,
        description="Workflow description (markdown supported)",
        json_schema_extra={
            "example": "Comprehensive quantitative research workflow for equities analysis"
        },
        max_length=1000,
    )
    tags: Optional[List[str]] = Field(
        None,
        description="Searchable tags",
        json_schema_extra={"example": ["research", "equities", "earnings"]},
    )
    owner: str = Field(
        ...,
        description="Workflow owner/creator",
        json_schema_extra={"example": "research_team"},
        min_length=1,
        max_length=128,
    )
    definition: Dict[str, Any] = Field(
        ...,
        description="Complete workflow JSON definition per WORKFLOW_JSON_SPEC",
    )

    @field_validator("tenant_id")
    @classmethod
    def validate_tenant_id(cls, v: str) -> str:
        return validate_resource_id(v, "tenant_id")

    @field_validator("workflow_id")
    @classmethod
    def validate_workflow_id(cls, v: str) -> str:
        return validate_workflow_id_format(v)

    @field_validator("version")
    @classmethod
    def validate_version(cls, v: str) -> str:
        return validate_semantic_version(v)

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("name cannot be empty or whitespace-only")
        return v.strip()

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        return validate_resource_id(v, "owner")

    @field_validator("definition")
    @classmethod
    def validate_definition(cls, v: Dict[str, Any]) -> Dict[str, Any]:
        if not isinstance(v, dict):
            raise ValueError("definition must be a valid JSON object")
        if not v:
            raise ValueError("definition cannot be empty")
        required_fields = {"nodes", "edges", "entry_point", "exit_point"}
        missing_fields = required_fields - set(v.keys())
        if missing_fields:
            raise ValueError(
                f"workflow definition missing required fields: {', '.join(sorted(missing_fields))}"
            )
        nodes = v.get("nodes")
        if not isinstance(nodes, list) or not nodes:
            raise ValueError("workflow definition nodes must be a non-empty array")

        for node in nodes:
            if not isinstance(node, dict):
                raise ValueError("workflow definition nodes must be JSON objects")
            node_type = node.get("type")
            if node_type not in {"entry", "exit"} and not node.get("node_id"):
                node_label = node.get("alias") or node.get("id") or "<unknown>"
                raise ValueError(
                    f"workflow node '{node_label}' missing node_id for compositional workflow"
                )
        return v

    @model_validator(mode="after")
    def validate_workflow_id_matches_version(self):
        workflow_id = self.workflow_id
        version = self.version
        id_version = workflow_id.split(":")[-1][1:]
        if id_version != version:
            raise ValueError(
                f"workflow_id version (v{id_version}) must match version field ({version})"
            )
        return self


from enum import Enum


class WorkflowNodeType(str, Enum):
    """Types of nodes in a workflow."""
    
    LLM_CALL = "llm_call"
    TOOL_CALL = "tool_call"
    CONDITIONAL = "conditional"
    PARALLEL = "parallel"
    LOOP = "loop"
    ENTRY = "entry"
    EXIT = "exit"


class WorkflowNode(BaseModel):
    """A single node in the workflow DAG."""
    
    id: str = Field(..., description="Unique node identifier")
    type: WorkflowNodeType = Field(..., description="Type of node")
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
    
    This is a deterministic, validated DAG that can be executed.
    """
    
    id: str = Field(..., description="Unique workflow ID")
    name: str = Field(..., description="Workflow name")
    description: str = Field(..., description="Workflow description")
    version: str = Field(default="1.0.0", description="Workflow version")
    
    # DAG structure
    nodes: List[WorkflowNode] = Field(..., description="All nodes in the workflow")
    edges: List[WorkflowEdge] = Field(..., description="All edges in the workflow")
    
    # Entry/exit points
    entry_point: str = Field(..., description="ID of the start/entry node")
    exit_point: str = Field(..., description="ID of the exit/end node")
    
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

    @property
    def start_node_id(self) -> str:
        """Backward compatible getter for entry_point."""
        return self.entry_point

    @property
    def end_node_ids(self) -> List[str]:
        """Backward compatible getter for exit_point."""
        return [self.exit_point]
    
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
