class GraphWeaveConfigurable(TypedDict):
    tenant_id: str
    thread_id: str
    workflow_id: str
    workflow_version: str
    max_tokens: int
    user_id: str


graph.invoke(
    state_input,
    config={
        "configurable": {
            "tenant_id": "tenant_abc",
            "thread_id": "session_123",
        },
        "recursion_limit": 25,
    },
)
