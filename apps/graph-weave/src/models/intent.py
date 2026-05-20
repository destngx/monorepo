"""
Intent Extraction Schema

Captures user intent in a structured format for deterministic workflow generation.
Based on research: BayesFlow, FlowMind, Opus multimodal extraction.

Key principle: LLM extracts intent ONCE, then deterministic validators construct DAG.
"""

from typing import Optional, List, Dict, Any, Literal
from pydantic import BaseModel, Field, field_validator


class IntentParameter(BaseModel):
    """A parameter or input required for an action."""
    
    name: str = Field(..., description="Parameter name (e.g., 'query', 'file_path')")
    type: Literal["string", "number", "boolean", "array", "object"] = Field(
        default="string",
        description="Parameter type"
    )
    description: str = Field(..., description="What this parameter represents")
    required: bool = Field(default=True, description="Whether this parameter is required")
    default: Optional[Any] = Field(default=None, description="Default value if not provided")
    constraints: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Validation constraints (e.g., {'min': 0, 'max': 100})"
    )


class IntentAction(BaseModel):
    """A single action or step in the workflow."""
    
    id: str = Field(..., description="Unique action identifier (e.g., 'search_web', 'fetch_url')")
    name: str = Field(..., description="Human-readable action name")
    description: str = Field(..., description="What this action does")
    operator: str = Field(
        ...,
        description="Operator/tool to invoke (e.g., 'web_search', 'file_read', 'llm_call')"
    )
    parameters: List[IntentParameter] = Field(
        default_factory=list,
        description="Input parameters for this action"
    )
    dependencies: List[str] = Field(
        default_factory=list,
        description="IDs of actions that must complete before this one"
    )
    retry_policy: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Retry configuration (max_attempts, backoff_strategy)"
    )
    timeout_seconds: Optional[int] = Field(
        default=None,
        description="Timeout for this action"
    )


class IntentExtraction(BaseModel):
    """
    Structured extraction of user intent.
    
    This is the OUTPUT of the LLM's single invocation.
    Everything after this is deterministic validation and DAG construction.
    """
    
    goal: str = Field(..., description="High-level goal or objective")
    description: str = Field(..., description="Detailed description of what the user wants")
    
    actions: List[IntentAction] = Field(
        ...,
        description="Ordered list of actions to achieve the goal"
    )
    
    success_criteria: List[str] = Field(
        default_factory=list,
        description="Criteria to determine if the workflow succeeded"
    )
    
    constraints: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Constraints (e.g., max_cost, max_duration, required_tools)"
    )
    
    metadata: Optional[Dict[str, Any]] = Field(
        default=None,
        description="Additional metadata (user_id, session_id, etc.)"
    )
    
    @field_validator("actions")
    @classmethod
    def validate_action_ids_unique(cls, actions):
        """Ensure all action IDs are unique."""
        action_ids = [a.id for a in actions]
        if len(action_ids) != len(set(action_ids)):
            raise ValueError("Action IDs must be unique")
        return actions
    
    @field_validator("actions")
    @classmethod
    def validate_action_dependencies(cls, actions):
        """Ensure all dependencies reference existing actions."""
        action_ids = {a.id for a in actions}
        for action in actions:
            for dep_id in action.dependencies:
                if dep_id not in action_ids:
                    raise ValueError(f"Action {action.id} depends on non-existent {dep_id}")
        return actions
    
    @field_validator("actions")
    @classmethod
    def validate_no_circular_dependencies(cls, actions):
        """Ensure no circular dependencies exist."""
        def has_cycle(action_id, visited, rec_stack, graph):
            visited.add(action_id)
            rec_stack.add(action_id)
            
            for dep_id in graph.get(action_id, []):
                if dep_id not in visited:
                    if has_cycle(dep_id, visited, rec_stack, graph):
                        return True
                elif dep_id in rec_stack:
                    return True
            
            rec_stack.remove(action_id)
            return False
        
        # Build dependency graph
        graph = {a.id: a.dependencies for a in actions}
        visited = set()
        
        for action_id in graph:
            if action_id not in visited:
                if has_cycle(action_id, visited, set(), graph):
                    raise ValueError(f"Circular dependency detected involving {action_id}")
        
        return actions
