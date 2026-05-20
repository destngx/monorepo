"""
Operator Registry

A²Flow-style operator discovery and dynamic composition.

References:
- A²Flow (AAAI 2026): Automated operator discovery framework
"""

from typing import Dict, List, Optional, Any, Callable, Set
from pydantic import BaseModel, Field, ConfigDict
import json

from ..models import OperatorCapability


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


# Built-in standard operators definition
WEB_SEARCH = OperatorDefinition(
    id="web_search",
    name="Web Search",
    description="Search the web for information",
    capability=OperatorCapability.SEARCH,
    tags=["external", "search", "information-retrieval"],
    input_schema={
        "type": "object",
        "properties": {
            "query": {"type": "string", "description": "Search query"},
            "max_results": {"type": "integer", "default": 10}
        },
        "required": ["query"]
    },
    output_schema={
        "type": "array",
        "items": {
            "type": "object",
            "properties": {
                "title": {"type": "string"},
                "url": {"type": "string"},
                "snippet": {"type": "string"}
            }
        }
    },
    timeout_seconds=30,
    cost_per_call=0.01,
    is_deterministic=False
)

FETCH_URL = OperatorDefinition(
    id="fetch_url",
    name="Fetch URL",
    description="Fetch and parse content from a URL",
    capability=OperatorCapability.DATA_FETCH,
    tags=["external", "http", "content-extraction"],
    input_schema={
        "type": "object",
        "properties": {
            "url": {"type": "string", "description": "URL to fetch"},
            "timeout": {"type": "integer", "default": 30}
        },
        "required": ["url"]
    },
    output_schema={
        "type": "object",
        "properties": {
            "content": {"type": "string"},
            "status_code": {"type": "integer"},
            "headers": {"type": "object"}
        }
    },
    timeout_seconds=60,
    cost_per_call=0.001,
    max_retries=3
)

LLM_CALL = OperatorDefinition(
    id="llm_call",
    name="LLM Call",
    description="Call a language model for text generation or analysis",
    capability=OperatorCapability.LLM_CALL,
    tags=["ai", "nlp", "generation"],
    input_schema={
        "type": "object",
        "properties": {
            "prompt": {"type": "string", "description": "Input prompt"},
            "model": {"type": "string", "default": "gpt-4"},
            "temperature": {"type": "number", "default": 0.7},
            "max_tokens": {"type": "integer", "default": 1000}
        },
        "required": ["prompt"]
    },
    output_schema={
        "type": "object",
        "properties": {
            "text": {"type": "string"},
            "tokens_used": {"type": "integer"}
        }
    },
    timeout_seconds=120,
    cost_per_call=0.05,
    is_deterministic=False
)

PYTHON_SCRIPT = OperatorDefinition(
    id="python_script",
    name="Python Script",
    description="Execute a Python script",
    capability=OperatorCapability.COMPUTE,
    tags=["compute", "local", "deterministic"],
    input_schema={
        "type": "object",
        "properties": {
            "script": {"type": "string", "description": "Python code to execute"},
            "timeout": {"type": "integer", "default": 60}
        },
        "required": ["script"]
    },
    output_schema={
        "type": "object",
        "properties": {
            "result": {},
            "stdout": {"type": "string"},
            "stderr": {"type": "string"}
        }
    },
    timeout_seconds=300,
    cost_per_call=0.0,
    is_deterministic=True
)

FILE_READ = OperatorDefinition(
    id="file_read",
    name="File Read",
    description="Read content from a file",
    capability=OperatorCapability.FILE_IO,
    tags=["file", "local", "io"],
    input_schema={
        "type": "object",
        "properties": {
            "path": {"type": "string", "description": "File path"},
            "encoding": {"type": "string", "default": "utf-8"}
        },
        "required": ["path"]
    },
    output_schema={
        "type": "object",
        "properties": {
            "content": {"type": "string"},
            "size": {"type": "integer"}
        }
    },
    timeout_seconds=10,
    cost_per_call=0.0,
    is_deterministic=True
)

FILE_WRITE = OperatorDefinition(
    id="file_write",
    name="File Write",
    description="Write content to a file",
    capability=OperatorCapability.FILE_IO,
    tags=["file", "local", "io"],
    input_schema={
        "type": "object",
        "properties": {
            "path": {"type": "string", "description": "File path"},
            "content": {"type": "string", "description": "Content to write"},
            "mode": {"type": "string", "default": "w"}
        },
        "required": ["path", "content"]
    },
    output_schema={
        "type": "object",
        "properties": {
            "success": {"type": "boolean"},
            "bytes_written": {"type": "integer"}
        }
    },
    timeout_seconds=10,
    cost_per_call=0.0,
    is_deterministic=True
)


def register_builtin_operators():
    """Register all built-in operators."""
    for operator in [
        WEB_SEARCH,
        FETCH_URL,
        LLM_CALL,
        PYTHON_SCRIPT,
        FILE_READ,
        FILE_WRITE,
    ]:
        register_operator(operator)


# Auto-register on import
register_builtin_operators()
