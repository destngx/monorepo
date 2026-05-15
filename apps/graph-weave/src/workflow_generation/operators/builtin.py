"""
Built-in Operators

Standard operators for common workflow tasks.
"""

from .registry import OperatorDefinition, OperatorCapability, register_operator


# Data Fetch Operators
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

# Data Process Operators
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

# Data Store Operators
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


# Register all built-in operators
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
