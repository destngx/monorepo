class GraphCache:
    async def get_or_compile(
        self, tenant_id: str, workflow_id: str, version: str
    ) -> CompiledGraph:
        cache_key = f"graphweave:graph_cache:{tenant_id}:{workflow_id}:v{version}"
        cached = await redis.get(cache_key)
        if cached:
            return pickle.loads(cached)

        workflow_json = await self._load_workflow(tenant_id, workflow_id, version)
        graph = await self._compile_graph(workflow_json)
        ttl = 3600 if workflow_json["limits"].get("experimental", False) else 86400
        await redis.setex(cache_key, ttl, pickle.dumps(graph))
        return graph

    async def invalidate(self, tenant_id: str, workflow_id: str, version: str):
        cache_key = f"graphweave:graph_cache:{tenant_id}:{workflow_id}:v{version}"
        await redis.delete(cache_key)


class SSEStreamer:
    async def stream_workflow(self, graph: CompiledGraph, config: dict, state: dict):
        async for event in graph.astream(
            state, config=config, stream_mode=["values", "updates", "debug"]
        ):
            if event["type"] == "node_start":
                yield {"event": "node_start", "data": {"node_id": event["name"]}}
            elif event["type"] == "token":
                yield {"event": "token", "data": {"content": event["content"]}}
            elif event["type"] == "tool_result":
                yield {
                    "event": "tool_result",
                    "data": {"tool_id": event["tool"], "summary": event["summary"]},
                }
            elif event["type"] == "complete":
                yield {
                    "event": "complete",
                    "data": {"final_response": state.get("final_response", "")},
                }


from fastapi.responses import StreamingResponse


async def execute_streaming(request: ExecutionRequest, config: RunnableConfig):
    graph = await load_or_compile_graph(request.workflow_id, request.workflow_version)

    async def sse_generator():
        async for event in graph.astream_events(
            {"messages": [HumanMessage(content=request.input.user_message)]},
            config,
            version="v2",
        ):
            if event["event"] == "on_chat_model_stream":
                yield f"data: {json.dumps({'event': 'token', 'data': event['data']['chunk']})}\n\n"
            elif event["event"] == "on_node_start":
                yield f"data: {json.dumps({'event': 'node_start', 'data': event['name']})}\n\n"
            elif event["event"] == "on_tool_start":
                yield f"data: {json.dumps({'event': 'tool_start', 'data': event['name']})}\n\n"

    return StreamingResponse(sse_generator(), media_type="text/event-stream")
