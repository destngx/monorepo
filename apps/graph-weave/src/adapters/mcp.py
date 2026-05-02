from typing import List, Dict, Any, Optional


class MockMCPServer:
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
                "description": "Executes a bash command.",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "command": {"type": "string", "description": "The bash command to run."},
                        "cwd": {"type": "string", "description": "Optional working directory."}
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
            query = params.get("query")
            return {"results": [f"Result for: {query}"]}

        elif tool_name == "bash":
            command = params.get("command")
            return {"stdout": f"Mock output for {command}", "stderr": "", "exit_code": 0}

        return None
