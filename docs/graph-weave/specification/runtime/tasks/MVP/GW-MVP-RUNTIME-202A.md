# GW-MVP-RUNTIME-202A: LangGraph Graph Builder

**Objective**: Build real LangGraph StateGraph from workflow JSON schema with node registration and edge condition evaluation.

**Phase**: [MVP]

**Duration**: 1 hour

**Blocked By**: NONE (can build without external dependencies)

**Blocks**: RUNTIME-202B, RUNTIME-202C

## Requirements

### Functional

- StateGraph builder from workflow JSON
- Node types: entry, agent_node, branch, exit (all 4 types supported)
- Edge conditions: JSONPath expressions (e.g., `"$.confidence > 0.7"`)
- Deterministic edge evaluation against workflow state
- State schema: required fields, types, nested objects
- Node parameter validation: ensure all required params present
- Prompt template interpolation: variables in prompts replaced from context
- **Per-node configuration**: each agent_node can specify custom:
  - `system_prompt` (string, can override workflow default)
  - `user_prompt_template` (string with {variables})
  - `provider` (e.g., "github", "openai", defaults to global)
  - `model` (e.g., "claude-3.5-sonnet", defaults to global)
  - `temperature` (optional, 0-1, defaults to global)
  - `max_tokens` (optional, defaults to global)
  - `tools` (optional, list of MCP tools node can access: load_skill, search, verify)
- Node-level configuration validation (ensure provider exists, model valid for provider)
- Fallback to global config if node-level not specified

### Non-Functional

- Graph building <100ms for typical workflow (10-node) even with per-node config
- State schema validation before graph execution
- Clear error messages for invalid workflow JSON and node config
- No external API calls (all local computation)

## Implementation Approach

1. Create `src/adapters/langgraph_graph_builder.py`:
   - `WorkflowParser` class:
     - `parse_workflow_json(workflow_dict)` → `WorkflowSpec`
     - `build_graph(spec)` → `StateGraph`
   - Node builders:
     - `build_entry_node(spec)` → function
     - `build_agent_node(spec, node_config)` → function (with per-node config extraction)
     - `build_branch_node(spec)` → function
     - `build_exit_node(spec)` → function
   - Edge evaluator:
     - `evaluate_condition(jsonpath_expr, state)` → bool
   - Schema validation:
     - Required fields per node type
     - Output key naming conventions
     - Prompt template variable interpolation
   - **Per-node config validation**:
     - Extract `provider`, `model`, `temperature`, `max_tokens`, `tools` from node
     - Validate provider exists (github, openai, etc.)
     - Validate model name for provider
     - Validate temperature in [0, 1] if specified
     - Validate max_tokens >= 1 if specified
     - Validate tools list contains only valid MCP tools

2. Extended node schema example:

   ```json
   {
     "type": "agent_node",
     "id": "research_node",
     "system_prompt": "You are an AI researcher...",
     "user_prompt_template": "Research {topic}",
     "provider": "github",
     "model": "claude-3.5-sonnet",
     "temperature": 0.7,
     "max_tokens": 2000,
     "tools": ["load_skill", "search"],
     "output_key": "research_output"
   }
   ```

3. Use `jsonpath-ng` library for JSONPath evaluation

4. Implementation notes:
   - Each node is a LangGraph node function
   - Edges connect nodes based on condition evaluation
   - State flows through nodes, accumulating outputs
   - Graph is compiled once, reused for multiple executions
   - Per-node config attached to node metadata for downstream use

## Acceptance Criteria

- [ ] Parser loads workflow JSON successfully
- [ ] Graph builds successfully for all 4 node types
- [ ] Edge conditions evaluate correctly (JSONPath expressions)
- [ ] State schema enforced (required fields present at each step)
- [ ] Node parameters validated before graph execution
- [ ] Prompt template interpolation works correctly
- [ ] All tests passing (12+ tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-RUNTIME-040 [MVP,FULL]: A single compiled graph must handle multiple workflows
- FR-WORKFLOW-001 [MVP]: Workflow JSON schema must support entry, agent, branch, exit nodes

## Deliverables

1. `src/adapters/langgraph_graph_builder.py` (150 LOC)
2. `tests/test_langgraph_graph_builder.py` (150+ LOC, 12+ tests)

## Test Coverage (12+ tests)

### Unit Tests

- [ ] Entry node builds and passes through input state
- [ ] Agent node builds with system_prompt, user_prompt_template
- [ ] Agent node extracts per-node config (provider, model, temperature, max_tokens)
- [ ] Agent node falls back to global config if not specified
- [ ] Branch node builds with edge conditions
- [ ] Exit node builds with output mapping
- [ ] JSONPath condition `"$.field > value"` evaluates correctly
- [ ] JSONPath condition `"$.field == 'string'"` evaluates correctly
- [ ] State schema validation: required field missing raises error
- [ ] State schema validation: type mismatch raises error
- [ ] Prompt template interpolation: {topic} replaced correctly
- [ ] Prompt template interpolation: {missing_var} raises error
- [ ] Per-node provider validation: invalid provider raises error
- [ ] Per-node model validation: model not supported for provider raises error
- [ ] Per-node temperature validation: value outside [0,1] raises error
- [ ] Per-node tools validation: unknown tool name raises error
- [ ] Graph with 5 nodes builds without error
- [ ] Graph topology validated (all edges reference existing nodes)
- [ ] Multiple nodes with different providers/models all configured correctly

## Error Scenarios

- [ ] Invalid JSONPath expression → clear error message
- [ ] Unknown node type → error message
- [ ] Missing required node parameters → error
- [ ] Circular edges without exit condition → warning (but allowed)
- [ ] Dangling nodes (no outgoing edges, not exit) → error
- [ ] **Invalid provider in node config → error with suggestion**
- [ ] **Invalid model for provider → error with supported models list**
- [ ] **Temperature outside [0,1] → error**
- [ ] **Tools list contains unknown tool → error with available tools list**

## Environment Variables

None for this task (no external dependencies)

## Implementation Notes

- Use `jsonpath-ng` library for JSONPath evaluation
- No external API calls (all local computation)
- Use TypedDict for WorkflowSpec validation
- Graph building is fast (100ms for typical 10-node workflow)
- This task has NO dependencies - can start immediately
- Prerequisite for RUNTIME-202B and RUNTIME-202C

## Testing Strategy

- Unit: Node builders, edge conditions, template interpolation
- Integration: Full workflow graph building and topology validation
- Error scenarios: Invalid JSON, missing parameters, malformed conditions
- Performance: Large graphs (50+ nodes) build within time budget
