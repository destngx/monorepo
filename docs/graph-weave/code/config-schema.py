class GraphWeaveConfigurable(TypedDict):
    """Payload that goes INSIDE config['configurable']"""
    tenant_id: str
    thread_id: str
    workflow_id: str
    workflow_version: str
    max_tokens: int
    user_id: str

# Execution construction:
graph.invoke(
    state_input,
    config={
        "configurable": {  # Fits GraphWeaveConfigurable
            "tenant_id": "tenant_abc",
            "thread_id": "session_123",
            # ...
        },
        "recursion_limit": 25  # Top-level, native LangGraph key
    }
)
