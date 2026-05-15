"""
Workflow Template Schema

YAML/JSON-based workflow templates with variable substitution.
"""

from typing import Dict, List, Optional, Any, Literal
from pydantic import BaseModel, Field, field_validator
import re
import json


class TemplateVariable(BaseModel):
    """A variable in a workflow template."""
    
    name: str = Field(..., description="Variable name")
    type: Literal["string", "number", "boolean", "array", "object"] = Field(
        default="string",
        description="Variable type"
    )
    description: str = Field(..., description="Variable description")
    required: bool = Field(default=True, description="Whether required")
    default: Optional[Any] = Field(default=None, description="Default value")
    pattern: Optional[str] = Field(default=None, description="Regex pattern for validation")
    
    def validate_value(self, value: Any) -> bool:
        """Validate a value against this variable definition."""
        if value is None:
            return not self.required
        
        # Type check
        type_map = {
            "string": str,
            "number": (int, float),
            "boolean": bool,
            "array": list,
            "object": dict,
        }
        
        if not isinstance(value, type_map[self.type]):
            return False
        
        # Pattern check
        if self.pattern and isinstance(value, str):
            if not re.match(self.pattern, value):
                return False
        
        return True


class TemplateNode(BaseModel):
    """A node in a workflow template."""
    
    id: str = Field(..., description="Node ID")
    name: str = Field(..., description="Node name")
    description: str = Field(..., description="Node description")
    operator: str = Field(..., description="Operator to invoke")
    
    # Configuration with variable substitution
    config: Dict[str, Any] = Field(
        default_factory=dict,
        description="Node configuration (may contain {{variable}} references)"
    )
    
    # Dependencies
    dependencies: List[str] = Field(
        default_factory=list,
        description="IDs of nodes that must complete first"
    )
    
    # Constraints
    timeout_seconds: Optional[int] = Field(default=None)
    retry_policy: Optional[Dict[str, Any]] = Field(default=None)
    
    def substitute_variables(self, variables: Dict[str, Any]) -> "TemplateNode":
        """Substitute variables in this node's configuration."""
        import copy
        
        node = copy.deepcopy(self)
        
        def substitute_value(value):
            """Recursively substitute variables in a value."""
            if isinstance(value, str):
                # Replace {{variable}} with actual values
                for var_name, var_value in variables.items():
                    pattern = r"\{\{" + re.escape(var_name) + r"\}\}"
                    if re.search(pattern, value):
                        # If the entire string is just the variable, replace with the value
                        if value == "{{" + var_name + "}}":
                            return var_value
                        # Otherwise, convert to string and substitute
                        value = re.sub(pattern, str(var_value), value)
                return value
            elif isinstance(value, dict):
                return {k: substitute_value(v) for k, v in value.items()}
            elif isinstance(value, list):
                return [substitute_value(v) for v in value]
            else:
                return value
        
        node.config = substitute_value(node.config)
        return node


class WorkflowTemplate(BaseModel):
    """A workflow template with variables and nodes."""
    
    id: str = Field(..., description="Template ID")
    name: str = Field(..., description="Template name")
    description: str = Field(..., description="Template description")
    version: str = Field(default="1.0.0", description="Template version")
    
    # Variables
    variables: List[TemplateVariable] = Field(
        default_factory=list,
        description="Template variables"
    )
    
    # Nodes
    nodes: List[TemplateNode] = Field(
        ...,
        description="Template nodes"
    )
    
    # Entry/exit
    start_node_id: str = Field(..., description="Start node ID")
    end_node_ids: List[str] = Field(..., description="End node IDs")
    
    # Metadata
    tags: List[str] = Field(default_factory=list, description="Template tags")
    
    @field_validator("nodes")
    @classmethod
    def validate_node_ids_unique(cls, nodes):
        """Ensure node IDs are unique."""
        node_ids = [n.id for n in nodes]
        if len(node_ids) != len(set(node_ids)):
            raise ValueError("Node IDs must be unique")
        return nodes
    
    @field_validator("nodes")
    @classmethod
    def validate_dependencies(cls, nodes):
        """Ensure all dependencies reference existing nodes."""
        node_ids = {n.id for n in nodes}
        for node in nodes:
            for dep_id in node.dependencies:
                if dep_id not in node_ids:
                    raise ValueError(f"Node {node.id} depends on non-existent {dep_id}")
        return nodes
    
    def get_required_variables(self) -> List[TemplateVariable]:
        """Get all required variables."""
        return [v for v in self.variables if v.required]
    
    def validate_variables(self, values: Dict[str, Any]) -> Dict[str, Any]:
        """
        Validate provided variable values.
        
        Returns:
            {"valid": bool, "errors": List[str]}
        """
        errors = []
        
        # Check required variables
        for var in self.variables:
            if var.required and var.name not in values:
                errors.append(f"Required variable '{var.name}' not provided")
        
        # Validate types and patterns
        for var_name, var_value in values.items():
            var_def = next((v for v in self.variables if v.name == var_name), None)
            if var_def and not var_def.validate_value(var_value):
                errors.append(
                    f"Variable '{var_name}' has invalid value: {var_value} "
                    f"(expected {var_def.type})"
                )
        
        return {"valid": len(errors) == 0, "errors": errors}
    
    def instantiate(self, variables: Dict[str, Any]) -> "WorkflowTemplate":
        """
        Create an instance of this template with variables substituted.
        
        Returns:
            New WorkflowTemplate with variables substituted
        """
        import copy
        
        # Validate variables
        validation = self.validate_variables(variables)
        if not validation["valid"]:
            raise ValueError(f"Variable validation failed: {validation['errors']}")
        
        # Substitute variables in nodes
        template = copy.deepcopy(self)
        template.nodes = [
            node.substitute_variables(variables)
            for node in template.nodes
        ]
        
        return template
