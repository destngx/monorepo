class GraphCache:
    """Cache compiled LangGraph graphs to avoid recompilation"""

    async def get_or_compile(self, tenant_id: str, workflow_id: str, version: str) -> CompiledGraph:
        cache_key = f"graphweave:graph_cache:{tenant_id}:{workflow_id}:v{version}"

        # Try to get from Redis
        cached = await redis.get(cache_key)
        if cached:
            return pickle.loads(cached)

        # Compile graph from workflow JSON
        workflow_json = await self._load_workflow(tenant_id, workflow_id, version)
        graph = await self._compile_graph(workflow_json)

        # Cache with TTL (1 hour for active workflows, 24 hours for stable)
        ttl = 3600 if workflow_json["limits"].get("experimental", False) else 86400
        await redis.setex(cache_key, ttl, pickle.dumps(graph))

        return graph

    async def invalidate(self, tenant_id: str, workflow_id: str, version: str):
        """Invalidate cache after workflow update"""
        cache_key = f"graphweave:graph_cache:{tenant_id}:{workflow_id}:v{version}"
        await redis.delete(cache_key)
class SSEStreamer:
    """Maps LangGraph events to SSE format"""

    async def stream_workflow(self, graph: CompiledGraph, config: dict, state: dict):
        """Stream workflow execution with granular events"""
        async for event in graph.astream(
            state,
            config=config,
            stream_mode=["values", "updates", "debug"]
        ):
            # Node start
            if event["type"] == "node_start":
                yield {
                    "event": "node_start",
                    "data": {
                        "node_id": event["name"],
                        "timestamp": datetime.utcnow().isoformat(),
                        "step": event["step"]
                    }
                }

            # Token streaming (from LLM nodes)
            if event["type"] == "token":
                yield {
                    "event": "token",
                    "data": {
                        "content": event["content"],
                        "node": event["node"],
                        "token_count": len(event["content"].split())
                    }
                }

            # Tool execution
            if event["type"] == "tool_call":
                yield {
                    "event": "tool_call",
                    "data": {
                        "tool_id": event["tool"],
                        "input": event["input"],
                        "status": "running"
                    }
                }

            # Tool result
            if event["type"] == "tool_result":
                yield {
                    "event": "tool_result",
                    "data": {
                        "tool_id": event["tool"],
                        "status": "success" if event["success"] else "error",
                        "summary": event["summary"]  # Never raw output
                    }
                }

            # Node end
            if event["type"] == "node_end":
                yield {
                    "event": "node_end",
                    "data": {
                        "node_id": event["name"],
                        "duration_ms": event["duration"]
                    }
                }

            # Completion
            if event["type"] == "complete":
                yield {
                    "event": "complete",
                    "data": {
                        "final_response": state.get("final_response", ""),
                        "total_tokens": sum(state.get("token_usage", {}).values()),
                        "steps": event["total_steps"]
                    }
                }
#####
from fastapi.responses import StreamingResponse

async def execute_streaming(request: ExecutionRequest, config: RunnableConfig):
    cache_key = f"compiled_graph:{request.tenant_id}:{request.workflow_id}:{request.workflow_version}"
    # In practice, serialize graph topology logic, cache it, or compile dynamically and cache in app memory
    graph = await load_or_compile_graph(request.workflow_id, request.workflow_version)

    async def sse_generator():
        async for event in graph.astream_events({"messages": [HumanMessage(content=request.input.user_message)]}, config, version="v2"):
            if event["event"] == "on_chat_model_stream" and event["name"] == "orchestrator":
                yield f"data: {json.dumps({'event': 'token', 'data': event['data']['chunk']})}\n\n"
            elif event["event"] == "on_node_start":
                yield f"data: {json.dumps({'event': 'node_start', 'data': event['name']})}\n\n"
            elif event["event"] == "on_tool_start":
                yield f"data: {json.dumps({'event': 'tool_start', 'data': event['name']})}\n\n"

    return StreamingResponse(sse_generator(), media_type="text/event-stream")
