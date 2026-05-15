"""
Operator Registry

A²Flow-style operator discovery and dynamic composition.

Algorithm:
1. Case-based generation: Retrieve similar operators from history
2. Clustering: Group operators by capability
3. Deep extraction: Extract operator signatures and constraints

References:
- A²Flow (AAAI 2026): Automated operator discovery framework
"""

from typing import Dict, List, Optional, Any, Callable, Set
from pydantic import BaseModel, Field, ConfigDict
from enum import Enum
import json


class OperatorCapability(str, Enum):
    """Operator capability categories."""
    
    DATA_FETCH = "data_fetch"
    DATA_PROCESS = "data_process"
    DATA_STORE = "data_store"
    LLM_CALL = "llm_call"
    SEARCH = "search"
    COMPUTE = "compute"
    EXTERNAL_API = "external_api"
    FILE_IO = "file_io"
    CUSTOM = "custom"


class OperatorDefinition(BaseModel):
    """Definition of an operator/tool."""
    
    model_config = ConfigDict(arbitrary_types_allowed=True)
    
    id: str = Field(..., description="Unique operator ID")
    name: str = Field(..., description="Human-readable name")
    description: str = Field(..., description="What this operator does")
    
    # Capability and classification
    capability: OperatorCapability = Field(..., description="Primary capability")
    tags: List[str] = Field(default_factory=list, description="Additional tags")
    
    # Input/output schema
    input_schema: Dict[str, Any] = Field(
        default_factory=dict,
        description="JSON schema for inputs"
    )
    output_schema: Dict[str, Any] = Field(
        default_factory=dict,
        description="JSON schema for outputs"
    )
    
    # Constraints
    timeout_seconds: Optional[int] = Field(
        default=None,
        description="Default timeout"
    )
    cost_per_call: Optional[float] = Field(
        default=None,
        description="Estimated cost per call"
    )
    max_retries: int = Field(default=3, description="Maximum retry attempts")
    
    # Metadata
    version: str = Field(default="1.0.0", description="Operator version")
    requires_auth: bool = Field(default=False, description="Requires authentication")
    is_deterministic: bool = Field(default=True, description="Deterministic output")
    
    # Implementation
    handler: Optional[Callable] = Field(
        default=None,
        description="Python callable (not serialized)"
    )


class OperatorRegistry:
    """
    Registry for operators/tools.
    
    Supports:
    - Operator discovery by capability
    - Operator composition
    - Constraint satisfaction
    - Cost/latency estimation
    """
    
    def __init__(self):
        """Initialize empty registry."""
        self._operators: Dict[str, OperatorDefinition] = {}
        self._capability_index: Dict[OperatorCapability, List[str]] = {}
        self._tag_index: Dict[str, List[str]] = {}
    
    def register(self, operator: OperatorDefinition) -> None:
        """Register an operator."""
        self._operators[operator.id] = operator
        
        # Index by capability
        if operator.capability not in self._capability_index:
            self._capability_index[operator.capability] = []
        self._capability_index[operator.capability].append(operator.id)
        
        # Index by tags
        for tag in operator.tags:
            if tag not in self._tag_index:
                self._tag_index[tag] = []
            self._tag_index[tag].append(operator.id)
    
    def get(self, operator_id: str) -> Optional[OperatorDefinition]:
        """Get operator by ID."""
        return self._operators.get(operator_id)
    
    def find_by_capability(
        self,
        capability: OperatorCapability
    ) -> List[OperatorDefinition]:
        """Find all operators with a given capability."""
        operator_ids = self._capability_index.get(capability, [])
        return [self._operators[oid] for oid in operator_ids]
    
    def find_by_tag(self, tag: str) -> List[OperatorDefinition]:
        """Find all operators with a given tag."""
        operator_ids = self._tag_index.get(tag, [])
        return [self._operators[oid] for oid in operator_ids]
    
    def find_by_tags(self, tags: List[str]) -> List[OperatorDefinition]:
        """Find operators matching any of the given tags."""
        matching_ids = set()
        for tag in tags:
            matching_ids.update(self._tag_index.get(tag, []))
        return [self._operators[oid] for oid in matching_ids]
    
    def search(
        self,
        query: str,
        capability: Optional[OperatorCapability] = None,
        tags: Optional[List[str]] = None
    ) -> List[OperatorDefinition]:
        """
        Search for operators by name/description and optional filters.
        
        Args:
            query: Search query (matches name or description)
            capability: Filter by capability
            tags: Filter by tags (any match)
        
        Returns:
            List of matching operators
        """
        query_lower = query.lower()
        results = []
        
        for operator in self._operators.values():
            # Check query match
            if query_lower not in operator.name.lower() and \
               query_lower not in operator.description.lower():
                continue
            
            # Check capability filter
            if capability and operator.capability != capability:
                continue
            
            # Check tags filter
            if tags and not any(tag in operator.tags for tag in tags):
                continue
            
            results.append(operator)
        
        return results
    
    def list_all(self) -> List[OperatorDefinition]:
        """List all registered operators."""
        return list(self._operators.values())
    
    def list_capabilities(self) -> List[OperatorCapability]:
        """List all registered capabilities."""
        return list(self._capability_index.keys())
    
    def get_stats(self) -> Dict[str, Any]:
        """Get registry statistics."""
        return {
            "total_operators": len(self._operators),
            "capabilities": {
                cap.value: len(ops)
                for cap, ops in self._capability_index.items()
            },
            "tags": {
                tag: len(ops)
                for tag, ops in self._tag_index.items()
            }
        }
    
    def to_dict(self) -> Dict[str, Any]:
        """Serialize registry to dict."""
        return {
            "operators": {
                oid: op.model_dump(exclude={"handler"})
                for oid, op in self._operators.items()
            },
            "stats": self.get_stats()
        }
    
    def to_json(self) -> str:
        """Serialize registry to JSON."""
        return json.dumps(self.to_dict(), indent=2)


# Global registry instance
_global_registry = OperatorRegistry()


def get_global_registry() -> OperatorRegistry:
    """Get the global operator registry."""
    return _global_registry


def register_operator(operator: OperatorDefinition) -> None:
    """Register an operator in the global registry."""
    _global_registry.register(operator)
