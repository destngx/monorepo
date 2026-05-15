"""
Pattern 2: Operator Registry → MCP Integration Example

This example demonstrates how to integrate the Operator Registry pattern
with Model Context Protocol (MCP) for dynamic tool discovery and execution.

Use Case: MCP server that exposes workflow operators as tools and resources.

Date: May 15, 2026
"""

from dataclasses import dataclass, field
from typing import Any, Optional
from enum import Enum
import json

# ============================================================================
# MCP Protocol Primitives (Simplified for Example)
# ============================================================================


class ToolInputType(str, Enum):
    """MCP tool input types"""
    STRING = "string"
    NUMBER = "number"
    BOOLEAN = "boolean"
    OBJECT = "object"
    ARRAY = "array"


@dataclass
class MCPToolInput:
    """MCP tool input schema"""
    name: str
    type: ToolInputType
    description: str
    required: bool = True
    default: Optional[Any] = None


@dataclass
class MCPTool:
    """MCP Tool primitive"""
    name: str
    description: str
    inputs: list[MCPToolInput]
    
    def to_dict(self) -> dict:
        """Convert to MCP tool JSON"""
        return {
            "name": self.name,
            "description": self.description,
            "inputSchema": {
                "type": "object",
                "properties": {
                    inp.name: {
                        "type": inp.type.value,
                        "description": inp.description,
                    }
                    for inp in self.inputs
                },
                "required": [inp.name for inp in self.inputs if inp.required],
            }
        }


@dataclass
class MCPResource:
    """MCP Resource primitive"""
    uri: str
    name: str
    description: str
    mime_type: str = "application/json"
    content: Optional[str] = None
    
    def to_dict(self) -> dict:
        """Convert to MCP resource JSON"""
        return {
            "uri": self.uri,
            "name": self.name,
            "description": self.description,
            "mimeType": self.mime_type,
        }


# ============================================================================
# Operator Registry Integration
# ============================================================================


class OperatorCapability(str, Enum):
    """Operator capabilities (from Pattern 2)"""
    SEARCH = "search"
    FETCH = "fetch"
    TRANSFORM = "transform"
    ANALYZE = "analyze"
    EXECUTE = "execute"


@dataclass
class OperatorDefinition:
    """Operator definition (from Pattern 2)"""
    id: str
    name: str
    description: str
    capabilities: list[OperatorCapability]
    inputs: dict[str, str]  # {param_name: type}
    outputs: dict[str, str]  # {output_name: type}
    tags: list[str] = field(default_factory=list)
    
    def to_mcp_tool(self) -> MCPTool:
        """Convert operator to MCP tool"""
        inputs = [
            MCPToolInput(
                name=param_name,
                type=ToolInputType.STRING,  # Simplified
                description=f"Parameter: {param_name}",
                required=True,
            )
            for param_name in self.inputs.keys()
        ]
        
        return MCPTool(
            name=self.name,
            description=self.description,
            inputs=inputs,
        )
    
    def to_mcp_resource(self) -> MCPResource:
        """Convert operator to MCP resource"""
        return MCPResource(
            uri=f"operator://{self.id}",
            name=self.name,
            description=self.description,
            content=json.dumps({
                "id": self.id,
                "name": self.name,
                "description": self.description,
                "capabilities": [c.value for c in self.capabilities],
                "inputs": self.inputs,
                "outputs": self.outputs,
                "tags": self.tags,
            }),
        )


class OperatorRegistry:
    """Global operator registry (from Pattern 2)"""
    
    _instance: Optional["OperatorRegistry"] = None
    
    def __init__(self):
        self.operators: dict[str, OperatorDefinition] = {}
        self._capability_index: dict[OperatorCapability, list[str]] = {}
        self._tag_index: dict[str, list[str]] = {}
    
    @classmethod
    def get_instance(cls) -> "OperatorRegistry":
        """Get global registry singleton"""
        if cls._instance is None:
            cls._instance = cls()
            cls._instance._init_builtin_operators()
        return cls._instance
    
    def _init_builtin_operators(self):
        """Initialize built-in operators"""
        builtin_ops = [
            OperatorDefinition(
                id="web_search",
                name="Web Search",
                description="Search the web for information",
                capabilities=[OperatorCapability.SEARCH],
                inputs={"query": "string", "max_results": "number"},
                outputs={"results": "array"},
                tags=["search", "web", "information-retrieval"],
            ),
            OperatorDefinition(
                id="fetch_url",
                name="Fetch URL",
                description="Fetch content from a URL",
                capabilities=[OperatorCapability.FETCH],
                inputs={"url": "string", "timeout": "number"},
                outputs={"content": "string", "status_code": "number"},
                tags=["fetch", "http", "content-retrieval"],
            ),
            OperatorDefinition(
                id="llm_call",
                name="LLM Call",
                description="Call an LLM for text generation",
                capabilities=[OperatorCapability.ANALYZE, OperatorCapability.TRANSFORM],
                inputs={"prompt": "string", "model": "string"},
                outputs={"response": "string", "tokens": "number"},
                tags=["llm", "generation", "analysis"],
            ),
            OperatorDefinition(
                id="python_script",
                name="Python Script",
                description="Execute Python code",
                capabilities=[OperatorCapability.EXECUTE, OperatorCapability.TRANSFORM],
                inputs={"code": "string", "timeout": "number"},
                outputs={"result": "object", "error": "string"},
                tags=["execution", "python", "scripting"],
            ),
            OperatorDefinition(
                id="file_read",
                name="File Read",
                description="Read file content",
                capabilities=[OperatorCapability.FETCH],
                inputs={"path": "string", "encoding": "string"},
                outputs={"content": "string", "size": "number"},
                tags=["file", "io", "read"],
            ),
            OperatorDefinition(
                id="file_write",
                name="File Write",
                description="Write content to file",
                capabilities=[OperatorCapability.EXECUTE],
                inputs={"path": "string", "content": "string"},
                outputs={"success": "boolean", "path": "string"},
                tags=["file", "io", "write"],
            ),
        ]
        
        for op in builtin_ops:
            self.register(op)
    
    def register(self, operator: OperatorDefinition):
        """Register an operator"""
        self.operators[operator.id] = operator
        
        # Index by capability
        for cap in operator.capabilities:
            if cap not in self._capability_index:
                self._capability_index[cap] = []
            self._capability_index[cap].append(operator.id)
        
        # Index by tag
        for tag in operator.tags:
            if tag not in self._tag_index:
                self._tag_index[tag] = []
            self._tag_index[tag].append(operator.id)
    
    def find_by_capability(self, capability: OperatorCapability) -> list[OperatorDefinition]:
        """Find operators by capability"""
        op_ids = self._capability_index.get(capability, [])
        return [self.operators[op_id] for op_id in op_ids]
    
    def find_by_tag(self, tag: str) -> list[OperatorDefinition]:
        """Find operators by tag"""
        op_ids = self._tag_index.get(tag, [])
        return [self.operators[op_id] for op_id in op_ids]
    
    def search(self, query: str) -> list[OperatorDefinition]:
        """Search operators by name/description"""
        query_lower = query.lower()
        results = []
        for op in self.operators.values():
            if (query_lower in op.name.lower() or 
                query_lower in op.description.lower() or
                query_lower in " ".join(op.tags)):
                results.append(op)
        return results
    
    def get_all(self) -> list[OperatorDefinition]:
        """Get all operators"""
        return list(self.operators.values())
    
    def get_stats(self) -> dict:
        """Get registry statistics"""
        return {
            "total_operators": len(self.operators),
            "capabilities": {cap.value: len(ops) for cap, ops in self._capability_index.items()},
            "tags": {tag: len(ops) for tag, ops in self._tag_index.items()},
        }


# ============================================================================
# MCP Server Implementation
# ============================================================================


class MCPWorkflowServer:
    """MCP server exposing workflow operators as tools and resources"""
    
    def __init__(self):
        self.registry = OperatorRegistry.get_instance()
        self.tools: dict[str, MCPTool] = {}
        self.resources: dict[str, MCPResource] = {}
        self._init_tools_and_resources()
    
    def _init_tools_and_resources(self):
        """Initialize MCP tools and resources from registry"""
        for operator in self.registry.get_all():
            # Convert operator to MCP tool
            tool = operator.to_mcp_tool()
            self.tools[tool.name] = tool
            
            # Convert operator to MCP resource
            resource = operator.to_mcp_resource()
            self.resources[resource.uri] = resource
    
    def list_tools(self) -> list[dict]:
        """List available tools (MCP protocol)"""
        return [tool.to_dict() for tool in self.tools.values()]
    
    def list_resources(self) -> list[dict]:
        """List available resources (MCP protocol)"""
        return [resource.to_dict() for resource in self.resources.values()]
    
    def call_tool(self, tool_name: str, arguments: dict) -> dict:
        """Call a tool (MCP protocol)"""
        # Find operator by tool name
        operators = self.registry.search(tool_name)
        if not operators:
            return {"error": f"Tool '{tool_name}' not found"}
        
        operator = operators[0]
        
        # Simulate tool execution
        return {
            "operator_id": operator.id,
            "tool_name": tool_name,
            "arguments": arguments,
            "status": "executed",
            "result": {
                "message": f"Executed {tool_name} with args: {arguments}",
                "operator_capabilities": [c.value for c in operator.capabilities],
            }
        }
    
    def read_resource(self, uri: str) -> dict:
        """Read a resource (MCP protocol)"""
        resource = self.resources.get(uri)
        if not resource:
            return {"error": f"Resource '{uri}' not found"}
        
        return {
            "uri": resource.uri,
            "name": resource.name,
            "description": resource.description,
            "content": resource.content,
        }
    
    def search_resources(self, query: str) -> list[dict]:
        """Search resources by query"""
        # Search operators by query
        operators = self.registry.search(query)
        
        # Return matching resources
        results = []
        for operator in operators:
            resource = operator.to_mcp_resource()
            results.append(resource.to_dict())
        
        return results
    
    def get_capabilities(self) -> dict:
        """Get server capabilities"""
        return {
            "tools": {
                "listTools": True,
                "callTool": True,
            },
            "resources": {
                "listResources": True,
                "readResource": True,
                "searchResources": True,
            },
            "registry": {
                "totalOperators": len(self.registry.operators),
                "capabilities": list(OperatorCapability),
            }
        }


# ============================================================================
# Example Usage
# ============================================================================


def example_mcp_server():
    """Example: MCP server with operator registry"""
    print("=" * 70)
    print("Pattern 2: Operator Registry → MCP Integration")
    print("=" * 70)
    print()
    
    # Initialize MCP server
    server = MCPWorkflowServer()
    
    # 1. List available tools
    print("1. Available Tools (MCP Protocol)")
    print("-" * 70)
    tools = server.list_tools()
    for tool in tools:
        print(f"  • {tool['name']}: {tool['description']}")
        print(f"    Inputs: {list(tool['inputSchema']['properties'].keys())}")
    print()
    
    # 2. List available resources
    print("2. Available Resources (MCP Protocol)")
    print("-" * 70)
    resources = server.list_resources()
    for resource in resources:
        print(f"  • {resource['uri']}: {resource['name']}")
    print()
    
    # 3. Call a tool
    print("3. Call Tool Example")
    print("-" * 70)
    result = server.call_tool("Web Search", {"query": "Python workflow patterns"})
    print(f"  Tool call result:")
    print(f"    Operator: {result['operator_id']}")
    print(f"    Status: {result['status']}")
    print(f"    Result: {result['result']}")
    print()
    
    # 4. Read a resource
    print("4. Read Resource Example")
    print("-" * 70)
    resource_data = server.read_resource("operator://web_search")
    print(f"  Resource: {resource_data['name']}")
    print(f"  Description: {resource_data['description']}")
    print(f"  Content: {resource_data['content']}")
    print()
    
    # 5. Search resources
    print("5. Search Resources Example")
    print("-" * 70)
    search_results = server.search_resources("fetch")
    print(f"  Query: 'fetch'")
    print(f"  Results: {len(search_results)} resources found")
    for result in search_results:
        print(f"    • {result['name']}: {result['description']}")
    print()
    
    # 6. Server capabilities
    print("6. Server Capabilities")
    print("-" * 70)
    capabilities = server.get_capabilities()
    print(f"  Tools: {capabilities['tools']}")
    print(f"  Resources: {capabilities['resources']}")
    print(f"  Registry: {capabilities['registry']}")
    print()
    
    # 7. Registry statistics
    print("7. Registry Statistics")
    print("-" * 70)
    stats = server.registry.get_stats()
    print(f"  Total operators: {stats['total_operators']}")
    print(f"  Capabilities: {stats['capabilities']}")
    print(f"  Tags: {stats['tags']}")
    print()


def example_dynamic_operator_registration():
    """Example: Dynamic operator registration at runtime"""
    print("=" * 70)
    print("Pattern 2: Dynamic Operator Registration")
    print("=" * 70)
    print()
    
    registry = OperatorRegistry.get_instance()
    
    # Register a custom operator
    custom_op = OperatorDefinition(
        id="custom_analyzer",
        name="Custom Analyzer",
        description="Custom data analysis operator",
        capabilities=[OperatorCapability.ANALYZE, OperatorCapability.TRANSFORM],
        inputs={"data": "object", "analysis_type": "string"},
        outputs={"analysis": "object", "confidence": "number"},
        tags=["custom", "analysis", "data-science"],
    )
    
    print("1. Registering custom operator...")
    registry.register(custom_op)
    print(f"   ✓ Registered: {custom_op.name}")
    print()
    
    # Find by capability
    print("2. Find operators by ANALYZE capability:")
    analyze_ops = registry.find_by_capability(OperatorCapability.ANALYZE)
    for op in analyze_ops:
        print(f"   • {op.name} ({op.id})")
    print()
    
    # Find by tag
    print("3. Find operators by 'custom' tag:")
    custom_ops = registry.find_by_tag("custom")
    for op in custom_ops:
        print(f"   • {op.name} ({op.id})")
    print()
    
    # Search
    print("4. Search for 'analyzer':")
    search_results = registry.search("analyzer")
    for op in search_results:
        print(f"   • {op.name}: {op.description}")
    print()


def example_capability_based_workflow():
    """Example: Build workflow based on required capabilities"""
    print("=" * 70)
    print("Pattern 2: Capability-based Workflow Construction")
    print("=" * 70)
    print()
    
    registry = OperatorRegistry.get_instance()
    
    # Define required capabilities for a workflow
    required_capabilities = [
        OperatorCapability.SEARCH,
        OperatorCapability.FETCH,
        OperatorCapability.ANALYZE,
    ]
    
    print("Building workflow with required capabilities:")
    for cap in required_capabilities:
        print(f"  • {cap.value}")
    print()
    
    # Find operators for each capability
    print("Operators available for each capability:")
    workflow_operators = {}
    for cap in required_capabilities:
        ops = registry.find_by_capability(cap)
        workflow_operators[cap.value] = [op.id for op in ops]
        print(f"  {cap.value}:")
        for op in ops:
            print(f"    - {op.name} ({op.id})")
    print()
    
    # Construct workflow
    print("Constructed workflow:")
    workflow = {
        "id": "research_workflow",
        "steps": [
            {
                "id": "search",
                "operator": workflow_operators["search"][0],
                "inputs": {"query": "Python workflow patterns"},
            },
            {
                "id": "fetch",
                "operator": workflow_operators["fetch"][0],
                "inputs": {"url": "${search.results[0].url}"},
                "depends_on": ["search"],
            },
            {
                "id": "analyze",
                "operator": workflow_operators["analyze"][0],
                "inputs": {"content": "${fetch.content}"},
                "depends_on": ["fetch"],
            },
        ]
    }
    
    print(json.dumps(workflow, indent=2))
    print()


if __name__ == "__main__":
    # Run examples
    example_mcp_server()
    print()
    example_dynamic_operator_registration()
    print()
    example_capability_based_workflow()
    
    print("=" * 70)
    print("✓ All examples completed successfully!")
    print("=" * 70)
