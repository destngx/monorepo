Orchestrator Node — Implementation Plan
Build GraphWeave's Orchestrator Node: a dynamic, ReAct-loop node type that lets a workflow delegate to a reasoning LLM which tools to call (and how many times) before handing control back to the graph — without changing how the surrounding DAG routes.

Traceability
ID Requirement
GW-FEAT-ORCH-001 type: "orchestrator" node in workflow schema
GW-FEAT-ORCH-002 Dynamic MCP skill loading per node config
GW-FEAT-ORCH-003 max_iterations + existing circuit-breaker guard
Design Summary
The Orchestrator is a self-contained ReAct loop attached to a single node in the DAG.
From the graph's perspective, it is just another node: it receives workflow_state in, emits a structured result out.
Internally, it iterates Reason → Act → Observe until it reaches a finish decision or hits max_iterations.

workflow_state ──► OrchestratorNode ──► {orchestrator_trace, final_result}
│
┌──────┴──────┐
│ ReAct Loop │
│ ────────── │
│ LLM Reason │
│ MCP Act │
│ Observe │
└─────────────┘
Streaming events (orchestrator.thought, orchestrator.tool_called) are emitted after each inner iteration so the UI gets real-time transparency — satisfying the "Streaming Emulsion" requirement.

Proposed Changes
1 · Schema / Models
[MODIFY]
models.py
Add Pydantic models for the new node config:

python
class OrchestratorConfig(BaseModel):
system_prompt: str
allowed_skills: List[str] # skill names from MCP catalog
max_iterations: int = Field(10, ge=1, le=50)
output_schema: Optional[Dict[str, Any]] = None # JSON-schema for final_result
Also extend the WorkflowCreate.validate_definition validator to recognise "orchestrator" as a valid node type (currently only entry/exit/agent/branch/guardrail/skill_loader are allowed implicitly).

2 · OrchestratorReAct Module
[NEW]
orchestrator_react.py
The core module. Encapsulates the entire ReAct loop so RealLangGraphExecutor can call a single function.

Key responsibilities:

Responsibility Detail
Context window management Rolling sliding window — keep last N messages to avoid token overflow
ReAct iterations Reason calls LLM; if response contains tool_calls → Act; if finish action → exit
Memory update Each tool result is appended to the message history
Structured output Returns {"orchestrator_trace": [...], "final_result": {...}}
Event emission Yields orchestrator.thought and orchestrator.tool_called events via callback
Circuit breaker Respects max_iterations; on breach exits with {"error": "max_iterations_exceeded"}
Pseudo-signature:

python
class OrchestratorReAct:
def **init**(
self,
client: LLMClient,
mcp_router: MCPRouter,
emit: Callable[[str, Dict], None], # event emitter callback
max_context_messages: int = 20,
): ...
def run(
self,
run_id: str,
node_id: str,
config: OrchestratorConfig,
workflow_state: Dict[str, Any],
) -> Dict[str, Any]:
"""
Returns:
{
"orchestrator_trace": [TraceEntry, ...],
"final_result": {...} # validated against output_schema if provided
}
"""
Internal flow per iteration:

1. Build messages = [system_prompt, user_context_summary, *short_term_memory]
2. Call LLM → response with optional tool_calls
3. If tool_calls:
   emit orchestrator.tool_called
   execute each via mcp_router.execute_tool()
   append tool results to short_term_memory
   append TraceEntry(type=action, ...)
4. If finish action / no tool_calls:
   break loop
5. emit orchestrator.thought (the LLM reasoning text)
6. append TraceEntry(type=thought, ...)
   Context overflow guard — \_trim_context(messages, max_context_messages) drops the oldest non-system messages when the window exceeds the limit.

3 · Executor Integration
[MODIFY]
langgraph_executor.py
Register the new node type inside RealLangGraphExecutor.execute() — the main dispatch switch at ~line 603:

python
elif node_type == "orchestrator":
node_result = self.\_execute_orchestrator_node(run_id, node, state, workflow)
Add the new private method \_execute_orchestrator_node:

python
def \_execute_orchestrator_node(
self, run_id: str, node: Dict, state: Dict, workflow: Dict
) -> Dict[str, Any]:
config = OrchestratorConfig(\*\*node.get("config", {}))
react = OrchestratorReAct(
client=self.mcp_router.get_provider_client(
config.provider, config.model
),
mcp_router=self.mcp_router,
emit=lambda etype, data: self.\_emit_event(run_id, etype, data),
)
return react.run(
run_id=run_id,
node_id=node["id"],
config=config,
workflow_state=state["workflow_state"],
)
No change to the routing, stagnation detection, or circuit breaker — the Orchestrator is bounded internally and the graph-level safeguards still apply.

4 · WORKFLOW_JSON_SPEC Update
[MODIFY]
WORKFLOW_JSON_SPEC.md
Add an orchestrator node section documenting:

Field Type Required Description
system_prompt string ✅ LLM persona and task context
allowed_skills string[] ✅ MCP skill names available to the orchestrator
max_iterations int ❌ (default 10) Circuit breaker — max ReAct loops
output_schema object ❌ JSON Schema for validating final_result
provider string ❌ LLM provider (defaults to node config convention)
model string ❌ Model name
Include the "Sandwich Pattern" example (Static → Orchestrator → Static).

5 · Integration Tests
[NEW]
test_e2e_orchestrator_node.py
BDD-style integration tests (following existing patterns in test_e2e_devops_log_analyzer.py):

Test scenario Covers
test_orchestrator_dynamic_tool_selection LLM picks tools based on context; result has orchestrator_trace
test_orchestrator_max_iterations_circuit_breaker Stub LLM always returns tool_call; verify exit at max_iterations
test_orchestrator_streaming_events orchestrator.thought + orchestrator.tool_called emitted each iteration
test_orchestrator_sandwiched_in_workflow Entry → Orchestrator → Static → Exit: routing works correctly
test_orchestrator_output_schema_validation output_schema provided; final_result conforms
[NEW]
test_unit_orchestrator_react.py
Unit tests for the module in isolation with mocked LLM and MCP router.

New Event Types
Event Emitted when Payload
orchestrator.thought After each LLM reasoning step {node_id, iteration, thought}
orchestrator.tool_called After each tool execution {node_id, iteration, tool_name, args, result}
orchestrator.finished ReAct loop exits {node_id, iterations_used, status}
These slot directly into the existing \_emit_event() infrastructure in the executor.

Open Questions
IMPORTANT

Provider/model fields on orchestrator config: The existing agent node puts provider and model inside config. Should OrchestratorConfig follow the same convention (add provider/model as optional fields), or should it inherit from a shared NodeConfig base? Having a base model would avoid duplication across node types.

NOTE

Streaming transport: The spec says events should stream to the UI after each inner iteration. The current architecture persists events to Redis via \_emit_event. If the frontend polls the event stream, no further changes are needed. If true SSE push is required, that's a separate, larger track.

NOTE

Real MCP skills vs. MockMCPServer: MCPRouter.execute_tool() currently only knows load_skill, search, verify. The orchestrator's allowed_skills field references skill names that may not be registered. For now, the plan assumes the existing mock catalog + a pass-through to mcp_server.call_tool() for unknown-but-registered skills. Real MCP server connectivity is tracked separately.

Verification Plan
Automated Tests
bash

# Unit

bunx nx run graph-weave:test -- -k "orchestrator_react"

# Integration (live gateway required)

bunx nx run graph-weave:test -- -k "orchestrator_node" --run-live

# Full suite

bunx nx run graph-weave:test
Manual Verification
Submit a workflow JSON with type: "orchestrator" via the /execute endpoint.
Tail the Redis event stream: verify orchestrator.thought and orchestrator.tool_called events appear after each iteration, interleaved with the normal node.started / node.completed events.
Submit a workflow that exceeds max_iterations and verify the run record shows status: "failed" with orchestrator_trace showing all attempted iterations.
