from typing import List, Dict, Any, Optional


class ToolRegistry:
    def __init__(self):
        self._tools = {
            "calculate": {
                "name": "calculate",
                "description": "Performs basic mathematical operations",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "operation": {
                            "type": "string",
                            "enum": ["add", "subtract", "multiply", "divide"],
                        },
                        "a": {"type": "number"},
                        "b": {"type": "number"},
                    },
                    "required": ["operation", "a", "b"],
                },
            },
            "search": {
                "name": "search",
                "description": "Searches for information",
                "inputSchema": {
                    "type": "object",
                    "properties": {"query": {"type": "string"}},
                    "required": ["query"],
                },
            },
            "bash": {
                "name": "bash",
                "description": "Executes a bash command. Use this for running predefined scripts or CLI tools. IMPORTANT: When running a .sh script, always prefix the command with 'bash ' (e.g., 'bash path/to/script.sh --arg value') to ensure reliable execution.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "command": {
                            "type": "string", 
                            "description": "The full bash command to run. If calling a script file, you MUST include 'bash ' prefix."
                        },
                        "cwd": {"type": "string", "description": "Optional working directory. Defaults to workspace root."}
                    },
                    "required": ["command"],
                },
            },
            "fs": {
                "name": "fs",
                "description": "File system operations (list_files, read_file, write_file, etc.)",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "operation": {
                            "type": "string",
                            "enum": ["list_files", "search_files", "read_file", "write_file", "move_file", "copy_file", "delete_file", "make_directory", "get_file_info"]
                        },
                        "path": {"type": "string"},
                        "content": {"type": "string"},
                        "pattern": {"type": "string"},
                        "replacement": {"type": "string"},
                        "source": {"type": "string"},
                        "destination": {"type": "string"},
                        "recursive": {"type": "boolean"}
                    },
                    "required": ["operation"]
                }
            },
            "fetch": {
                "name": "fetch",
                "description": "Fetches content from a URL. Supports GET and POST methods. Returns cleaned text for HTML pages.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "url": {"type": "string", "description": "The URL to fetch"},
                        "method": {"type": "string", "enum": ["GET", "POST"], "default": "GET"},
                        "headers": {"type": "object", "description": "Optional HTTP headers"}
                    },
                    "required": ["url"]
                }
            },
            "load_skill": {
                "name": "load_skill",
                "description": "Loads a specialized skill/prompt for the agent.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "skill_name": {"type": "string", "description": "Name of the skill to load"}
                    },
                    "required": ["skill_name"]
                }
            },
            "verify": {
                "name": "verify",
                "description": "Verifies a claim against known evidence.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "claim": {"type": "string", "description": "The claim to verify"}
                    },
                    "required": ["claim"]
                }
            },
            "node_registry": {
                "name": "node_registry",
                "description": "Interact with the Node Registry to search, retrieve, and resolve node definitions.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "operation": {
                            "type": "string",
                            "enum": ["search_nodes", "get_node", "find_compatible", "create_node", "list_current", "resolve_steps"],
                            "description": "The operation to perform"
                        },
                        "query": {"type": "string", "description": "Search query for nodes"},
                        "tags": {"type": "array", "items": {"type": "string"}, "description": "Filter by tags"},
                        "node_name": {"type": "string", "description": "Filter by node name"},
                        "node_id": {"type": "string", "description": "Exact node ID (for get_node)"},
                        "required_version": {"type": "string", "description": "Required version (for find_compatible)"},
                        "node_data": {"type": "object", "description": "Node data (for create_node)"},
                        "steps": {"type": "array", "items": {"type": "object"}, "description": "Intent-analysis steps (for resolve_steps)"},
                        "page_size": {"type": "integer", "description": "Maximum nodes to inspect"}
                    },
                    "required": ["operation"]
                }
            },
        }

    def list_tools(self) -> List[Dict[str, Any]]:
        return list(self._tools.values())

    def list_resources(self) -> List[Dict[str, Any]]:
        return []

    def call_tool(self, tool_name: str, params: Dict[str, Any]) -> Any:
        if tool_name not in self._tools:
            raise ValueError(f"Tool '{tool_name}' not found")

        if tool_name == "calculate":
            operation = params.get("operation")
            a = params.get("a", 0)
            b = params.get("b", 0)

            if operation == "add":
                return {"result": a + b}
            elif operation == "subtract":
                return {"result": a - b}
            elif operation == "multiply":
                return {"result": a * b}
            elif operation == "divide":
                return {"result": a / b if b != 0 else None}

        elif tool_name == "search":
            # NOTE: For now, search is still handled here as a mock until a real engine is integrated.
            query = params.get("query")
            return {"results": [f"Result for: {query}"]}

        # Tools handled by MCPRouter (bash, fs, fetch, load_skill, verify) 
        # should generally not reach here for execution.
        return None
