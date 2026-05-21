# State Management Refactoring Specification

> **Project**: Graph Weave
> **Component**: Runtime / State Management
> **Status**: Draft
> **Author**: Sisyphus
> **Date**: 2026-05-21

## Executive Summary

Graph Weave's current state model mixes node outputs, shared workflow data, execution metadata, and final response data into overlapping dictionaries. This creates ambiguous state paths, duplicated lookup logic, hidden fallback behavior, and inconsistent node result shapes.

This specification proposes a **breaking v2 state architecture** instead of a compatibility-first wrapper. The new architecture removes the dual-store model, standardizes node outputs, requires explicit namespaced paths, and routes all state access through one strict resolver.

**Risk**: High, intentional breaking change
**Impact**: High, removes ambiguous state semantics and establishes a clean runtime contract

---

## Goals

- Replace implicit, fallback-heavy state access with explicit namespaced paths.
- Stop merging node results into shared workflow state.
- Standardize the node result shape for all node handlers.
- Use one resolver for input mapping, prompt interpolation, edge conditions, and exit output mapping.
- Move transforms out of path suffixes and into explicit mapping configuration.
- Make exit nodes responsible for final API output projection.
- Update built-in workflows, generator prompts, tests, and checkpoints to the new state model.

## Non-Goals

- Preserve legacy state paths such as `$.summary`, `$.final_result`, or `$.previous_node.result`.
- Preserve response aliases such as `workflow_state`, `node_results`, `orchestrator_result`, or `{node_id}_output`.
- Maintain compatibility with old persisted checkpoints without migration.
- Keep permissive behavior for missing required state paths.

---

## Current Pain Points

### 1. Dual State Stores

The current executor writes every node result to both containers:

```python
state["node_results"][current_node_id] = node_result
state["workflow_state"].update(node_result)
```

This makes the same value accessible through multiple paths and allows unrelated node outputs to overwrite each other in `workflow_state`.

### 2. Ambiguous State Resolution

The current resolver checks `node_results`, then `workflow_state`, then root state, then performs deep scans and terminal scalar fallback. This behavior hides broken paths and makes reads dependent on dictionary shape and insertion order.

### 3. Inconsistent Node Result Shapes

Handlers return different keys:

- `result`
- `final_result`
- `orchestrator_result`
- `branch_result`
- `{node_id}_output`
- `{node_id}_status`
- handler-specific fields such as `tool_calls`, `turns`, `stdout`, and `stderr`

Downstream consumers must know handler-specific conventions instead of using one contract.

### 4. Duplicated Input Mapping

Multiple handlers and runtime utilities repeat the same pattern:

```python
for key, path in input_mapping.items():
    resolved[key] = executor._get_state_value(path, state)
```

Each copy depends on the same ambiguous resolver.

### 5. Duplicate Prompt and Condition Evaluation

Graph-level and runtime-level implementations use different rules for missing variables, dot paths, transforms, and errors.

### 6. Magic Path Transforms

Transforms are encoded as suffixes or pseudo-methods:

- `_shell`
- `_json`
- `_joined`
- `_first`
- `.join(...)`
- `.first(...)`
- `.json_quote(...)`

This makes path parsing unpredictable and mixes lookup semantics with formatting concerns.

---

## Proposed Architecture

### Canonical Runtime State

Replace the current `ExecutorState` shape with explicit namespaces:

```python
from typing import Any, Dict, List, Literal, Optional, TypedDict


class RuntimeState(TypedDict, total=False):
    status: Optional[str]
    step: int
    hop_count: int
    current_node: Optional[str]
    last_node: Optional[str]


class NodeResultPayload(TypedDict, total=False):
    status: Literal["completed", "failed", "skipped"]
    result: Any
    outputs: Dict[str, Any]
    error: Optional[str]
    metadata: Dict[str, Any]


class ExecutorState(TypedDict):
    input: Dict[str, Any]
    workflow: Dict[str, Any]
    nodes: Dict[str, NodeResultPayload]
    runtime: RuntimeState
    errors: List[Dict[str, Any]]
```

### Namespace Responsibilities

| Namespace  | Mutability  | Purpose                              |
| ---------- | ----------- | ------------------------------------ |
| `input`    | Immutable   | Original execution input             |
| `workflow` | Mutable     | Explicit shared workflow data only   |
| `nodes`    | Mutable     | Canonical per-node result payloads   |
| `runtime`  | Mutable     | Execution metadata and control state |
| `errors`   | Append-only | Node and execution errors            |

### Valid State Paths

All paths must start with one of the supported namespaces:

```text
$.input.alert
$.workflow.topic
$.nodes.researcher.result
$.nodes.researcher.outputs.summary
$.nodes.researcher.metadata.tokens_used
$.runtime.status
$.errors[0].message
```

Invalid examples:

```text
$.summary
$.final_result
$.previous_node.result
$.node_results.researcher.result
$.workflow_state.summary
```

### Node Result Contract

All node handlers return the same payload shape:

```python
class NodeResultPayload(TypedDict, total=False):
    status: Literal["completed", "failed", "skipped"]
    result: Any
    outputs: Dict[str, Any]
    error: Optional[str]
    metadata: Dict[str, Any]
```

Rules:

- Handlers do not attach `node_id`; the executor owns node identity.
- Primary semantic output goes in `result`.
- Named secondary outputs go in `outputs`.
- Diagnostics and non-domain execution details go in `metadata`.
- Failure details go in `error` and `metadata`, with `status="failed"`.
- The executor normalizes missing `status` to `"completed"` for successful handlers.

### Executor Write Semantics

The executor stores a node result only under its node namespace:

```python
state["nodes"][current_node_id] = normalized_payload
state["runtime"]["last_node"] = current_node_id
```

The executor must not merge node results into `workflow`.

Shared workflow values must be written explicitly by a dedicated node behavior or by configured output projection, not as an automatic side effect of node completion.

### Final Response Shape

Execution responses expose a projected `output` plus the internal state:

```json
{
  "run_id": "run-123",
  "thread_id": "thread-123",
  "tenant_id": "tenant-1",
  "workflow_id": "workflow:v2.0.0",
  "status": "completed",
  "hop_count": 4,
  "output": {
    "answer": "..."
  },
  "state": {
    "input": {},
    "workflow": {},
    "nodes": {},
    "runtime": {},
    "errors": []
  },
  "events": []
}
```

Rules:

- `output` is the stable consumer-facing result.
- `state` is the internal execution state for debugging and recovery.
- `workflow_state` and `final_state` are removed from the final API contract.
- Exit nodes own `output` projection through `output_mapping`.

---

## State Resolver

### Canonical Resolver API

Create a strict resolver module:

```text
apps/graph-weave/src/adapters/langgraph/runtime/state/resolver.py
```

```python
class StateResolutionError(ValueError):
    pass


class MissingStatePathError(StateResolutionError):
    pass


class InvalidStatePathError(StateResolutionError):
    pass


class StateResolver:
    def __init__(self, state: ExecutorState):
        self.state = state

    def resolve(self, path: str) -> Any:
        ...

    def resolve_mapping(self, mapping: dict[str, Any]) -> dict[str, Any]:
        ...
```

### Resolution Rules

- Path must be a string starting with `$.`.
- First segment after `$` must be one of `input`, `workflow`, `nodes`, `runtime`, or `errors`.
- Dot notation resolves dictionaries.
- Array notation supports list indices, for example `$.errors[0].message`.
- Missing required paths raise `MissingStatePathError`.
- Invalid syntax raises `InvalidStatePathError`.
- Resolver never deep-searches.
- Resolver never trims trailing segments.
- Resolver never silently falls back to another namespace.

### Mapping Forms

Simple mapping:

```json
{
  "topic": "$.input.event.details.topic"
}
```

Mapping with transform:

```json
{
  "command_arg": {
    "path": "$.nodes.planner.outputs.steps",
    "transform": "shell_quote"
  }
}
```

Mapping with optional default:

```json
{
  "priority": {
    "path": "$.input.priority",
    "required": false,
    "default": "normal"
  }
}
```

### Explicit Transforms

Create transforms in:

```text
apps/graph-weave/src/adapters/langgraph/runtime/state/transforms.py
```

Supported transforms:

| Transform        | Behavior                                           |
| ---------------- | -------------------------------------------------- |
| `shell_quote`    | Safely quote structured values for shell arguments |
| `json_stringify` | Serialize value to normalized JSON                 |
| `join`           | Join list values using configured separator        |
| `first`          | Return first list item                             |

Example with transform options:

```json
{
  "tags": {
    "path": "$.nodes.extractor.outputs.tags",
    "transform": "join",
    "separator": ", "
  }
}
```

---

## Prompt Interpolation

Use the canonical resolver for prompt placeholders.

### Runtime Interpolation

Runtime node prompts resolve placeholders from explicit state paths or local mapped inputs:

```text
Investigate {topic} using evidence from {$.nodes.researcher.outputs.summary}
```

Rules:

- Local input keys can be referenced as `{topic}` after `input_mapping` resolution.
- Direct state placeholders must use explicit paths such as `{ $.input.topic }` without spaces in actual workflow JSON.
- Missing required variables raise by default.
- Optional placeholders must be configured explicitly if unresolved placeholders should be preserved.

### Build-Time Interpolation

Graph builder interpolation uses the same parsing core but always raises on missing values. This keeps workflow validation strict.

---

## Edge Conditions

Runtime and graph-level condition evaluators must use the same resolver.

Supported condition format:

```text
$.nodes.classifier.outputs.confidence >= 0.8
$.nodes.guardrail.result == true
$.runtime.status != "failed"
```

Rules:

- Left side must be an explicit state path.
- Right side may be a string, number, boolean, null, or explicit state path.
- Missing paths fail deterministically.
- Invalid conditions raise validation errors at workflow compilation/build time when possible.

---

## Exit Node Projection

Exit nodes produce final response `output` through explicit `output_mapping`.

Example:

```json
{
  "id": "exit",
  "type": "exit",
  "config": {
    "output_mapping": {
      "answer": "$.nodes.answerer.result.answer",
      "confidence": "$.nodes.answerer.outputs.confidence",
      "trace": "$.nodes.investigate.outputs.orchestrator_trace"
    },
    "required_outputs": ["answer"]
  }
}
```

Rules:

- Production workflows should define `output_mapping`.
- Missing required outputs fail the execution.
- If `output_mapping` is omitted in tests or development workflows, the executor may return an empty object and keep full internal state under `state`.
- Exit node must not replace internal state.

---

## Handler-Specific Output Mapping

### Entry Node

Entry node initializes state only:

```python
state["input"] = input_data
state["workflow"] = {}
state["nodes"] = {}
state["runtime"] = {...}
state["errors"] = []
```

It does not duplicate input into workflow.

### Agent Node

Agent node returns:

```python
{
    "status": "completed",
    "result": result_data,
    "outputs": {
        "tool_calls": all_tool_calls
    },
    "metadata": {
        "tokens_used": total_tokens,
        "turns": turns
    }
}
```

### Orchestrator Node

Orchestrator node returns:

```python
{
    "status": "completed",
    "result": final_result,
    "outputs": {
        "orchestrator_trace": trace
    },
    "metadata": {
        "iterations": iteration_count,
        "tokens_used": tokens_used
    }
}
```

The legacy `orchestrator_result` alias is removed.

### CLI/Bash Node

CLI node returns:

```python
{
    "status": "completed",
    "result": stdout,
    "outputs": {
        "stdout": stdout,
        "stderr": stderr,
        "exit_code": exit_code
    },
    "metadata": {
        "command": redacted_command,
        "cwd": cwd
    }
}
```

### Branch Node

Branch node should not need a synthetic `branch_result` unless the workflow explicitly consumes it. Routing should evaluate conditions directly from state.

If a branch node must record its decision:

```python
{
    "status": "completed",
    "result": selected_branch,
    "outputs": {
        "selected_branch": selected_branch
    }
}
```

---

## Contract Validation

`ContractField.state_path` already exists and must use explicit paths.

Validation rules:

- Required input contract fields must have resolvable paths before node execution.
- Optional input contract fields may define defaults.
- `input_mapping` keys should correspond to contract field names when a contract exists.
- Output contract fields must describe canonical output paths under `$.nodes.<node_id>`.
- Empty `state_path` is invalid for new v2 nodes.

Example:

```json
{
  "input_contract": {
    "required": [
      {
        "name": "topic",
        "type": "string",
        "state_path": "$.input.event.details.topic"
      }
    ],
    "optional": [
      {
        "name": "priority",
        "type": "string",
        "state_path": "$.input.priority"
      }
    ]
  },
  "output_contract": {
    "produced": [
      {
        "name": "summary",
        "type": "string",
        "state_path": "$.nodes.summarizer.outputs.summary"
      }
    ]
  }
}
```

---

## Implementation Plan

### Phase 1: State Architecture Foundation

Create:

```text
apps/graph-weave/src/adapters/langgraph/runtime/state/schema.py
apps/graph-weave/src/adapters/langgraph/runtime/state/resolver.py
apps/graph-weave/src/adapters/langgraph/runtime/state/transforms.py
apps/graph-weave/src/adapters/langgraph/runtime/state/context.py
```

Implement:

- `ExecutorState`, `RuntimeState`, and `NodeResultPayload`.
- Strict path parsing and resolution.
- Explicit transform handling.
- Mapping resolution with `required`, `default`, and `transform`.
- Unit tests for all resolver and transform behavior.

### Phase 2: Executor State Cutover

Update:

```text
apps/graph-weave/src/adapters/langgraph/helper/types.py
apps/graph-weave/src/adapters/langgraph/runtime/engine/executor.py
apps/graph-weave/src/adapters/langgraph/runtime/engine/handlers.py
apps/graph-weave/src/adapters/langgraph/runtime/base/executor.py
```

Implement:

- New state initialization.
- Node result normalization.
- Node result writes to `state["nodes"][node_id]`.
- Runtime metadata writes to `state["runtime"]`.
- Error writes to `state["errors"]`.
- Final response shape with `output` and `state`.
- Remove automatic `workflow.update(node_result)`.

### Phase 3: Canonical Resolver Adoption

Update all state access call sites:

```text
apps/graph-weave/src/adapters/langgraph/runtime/engine/utils.py
apps/graph-weave/src/adapters/langgraph/runtime/engine/routing.py
apps/graph-weave/src/adapters/langgraph/runtime/base/workflow_utils.py
apps/graph-weave/src/adapters/langgraph/runtime/base/content_utils.py
apps/graph-weave/src/adapters/langgraph/graph/evaluator.py
apps/graph-weave/src/adapters/langgraph/graph/prompts.py
```

Implement:

- One resolver-backed input mapping function.
- One resolver-backed condition evaluator.
- One resolver-backed interpolation core with build-time and runtime modes.
- Delete or deprecate old fallback resolver behavior.

### Phase 4: Node Handler Refactor

Update:

```text
apps/graph-weave/src/adapters/langgraph/nodes/agent/handler.py
apps/graph-weave/src/adapters/langgraph/nodes/orchestrator.py
apps/graph-weave/src/adapters/langgraph/nodes/cli.py
apps/graph-weave/src/adapters/langgraph/runtime/engine/executor.py
```

Implement:

- Handler input resolution through canonical resolver.
- Canonical `NodeResultPayload` returns.
- Remove handler-emitted legacy aliases.
- Move handler-specific extras into `outputs` or `metadata`.

### Phase 5: Workflow and Compiler Migration

Update:

```text
apps/graph-weave/src/resources/workflows/
apps/graph-weave/src/resources/nodes/
apps/graph-weave/src/services/node_compiler.py
docs/graph-weave/code/
```

Implement:

- Rewrite built-in workflow paths to `$.input`, `$.workflow`, `$.nodes`, and `$.runtime`.
- Update workflow-generator prompts so generated paths are explicit.
- Update node compiler contract validation for v2 paths.
- Reject newly generated flat or legacy paths.

### Phase 6: Checkpoint and API Migration

Update:

```text
apps/graph-weave/src/services/checkpoint_service.py
apps/graph-weave/src/routers/execution.py
apps/graph-weave/tests/unit/test_checkpoint_service.py
apps/graph-weave/tests/unit/test_checkpoint_recovery.py
```

Implement:

- Store and restore the new `ExecutorState` shape.
- Return `output` and `state` from execution status endpoints.
- Remove old `workflow_state` response dependencies.
- Decide whether old checkpoints are invalidated or migrated.

Default decision: invalidate old checkpoints unless production data retention requires migration.

### Phase 7: Remove Legacy State Semantics

Remove:

- Deep search fallback.
- Terminal scalar fallback.
- `workflow_state` runtime writes.
- `node_results` runtime writes.
- `orchestrator_result`.
- `{node_id}_output`.
- `{node_id}_status`.
- Transform suffix parsing from state paths.

---

## Test Plan

### Resolver Unit Tests

- Resolves `$.input`, `$.workflow`, `$.nodes`, `$.runtime`, and `$.errors`.
- Resolves nested dictionaries and list indices.
- Rejects paths without `$.`.
- Rejects unknown namespaces.
- Raises on missing required paths.
- Returns defaults for optional mappings.
- Applies explicit transforms.
- Does not deep-search.
- Does not trim path segments.

### Node Result Tests

- Executor normalizes handler payloads.
- Executor stores results under `state["nodes"][node_id]`.
- Executor does not merge node results into `workflow`.
- Handler-provided `node_id` is rejected or ignored.
- `outputs` and `metadata` remain distinct.

### Prompt Tests

- Runtime interpolation resolves local mapped inputs.
- Runtime interpolation resolves explicit state placeholders.
- Build-time interpolation raises on missing variables.
- Shell and JSON transforms produce safe output.
- Legacy suffix transforms are rejected in v2 mappings.

### Edge Condition Tests

- Evaluates explicit state paths.
- Compares strings, numbers, booleans, and nulls.
- Supports right-hand state paths.
- Fails deterministically for missing paths.
- Rejects malformed conditions.

### Exit Node Tests

- Projects final `output` from explicit paths.
- Fails when required output is missing.
- Does not replace internal state.
- Returns empty output for development workflows without mapping, if allowed.

### Integration and E2E Tests

- Orchestrator workflow maps input from `$.input`.
- Orchestrator output is read from `$.nodes.<id>.result`.
- Orchestrator trace is read from `$.nodes.<id>.outputs.orchestrator_trace`.
- CLI workflow receives shell-quoted transformed arguments.
- Workflow generator emits v2 paths.
- Checkpoint recovery restores the new state shape.
- Execution API returns `output` and `state`.

### Regression Tests

- No references remain to `workflow_state` in runtime logic.
- No references remain to `node_results` in runtime logic.
- No tests depend on `orchestrator_result`.
- No tests depend on `{node_id}_output` or `{node_id}_status`.
- No generated resource workflow uses flat implicit state paths.

---

## Success Metrics

| Metric                                        | Target                                                 |
| --------------------------------------------- | ------------------------------------------------------ |
| Runtime state containers                      | `input`, `workflow`, `nodes`, `runtime`, `errors` only |
| State resolvers                               | 1 canonical resolver                                   |
| Automatic node result merges into workflow    | 0                                                      |
| Legacy output aliases                         | 0                                                      |
| Deep-search fallback usage                    | 0                                                      |
| Suffix transform parsing in paths             | 0                                                      |
| Built-in workflows using explicit v2 paths    | 100%                                                   |
| Node handlers returning canonical payload     | 100%                                                   |
| Exit nodes projecting final output explicitly | 100% for production workflows                          |

---

## Risk Assessment

| Risk                                  | Likelihood | Impact | Mitigation                                                                          |
| ------------------------------------- | ---------- | ------ | ----------------------------------------------------------------------------------- |
| Existing workflows break              | High       | High   | Intentional v2 breaking change; update bundled workflows and tests in same branch   |
| Checkpoint recovery incompatibility   | High       | Medium | Invalidate old checkpoints by default or add one-time migration if needed           |
| Generated workflows use legacy paths  | Medium     | High   | Update generator prompts and compiler validation before enabling v2                 |
| API consumers expect `workflow_state` | High       | High   | Update routers/tests/docs together; expose `output` as stable final result          |
| Shell argument escaping regression    | Medium     | High   | Preserve existing escaping behavior in explicit `shell_quote` transform tests       |
| Missing path failures increase        | High       | Medium | Treat as desired strictness; improve validation errors with node id and mapping key |

---

## Rollout Strategy

This is a breaking architecture refactor and should be released as a v2 state model.

Recommended rollout:

1. Land the state schema and resolver with exhaustive unit tests.
2. Cut over executor internals and node handlers in one branch.
3. Update resource workflows and generator prompts before running e2e tests.
4. Update API response expectations.
5. Remove legacy resolver behavior before merging.
6. Document the v1-to-v2 path migration rules.

Do not ship an intermediate half-standardized runtime where both old and new state semantics are accepted by default.

---

## Migration Guide

### Path Migration

| Old Path                      | New Path                               |
| ----------------------------- | -------------------------------------- |
| `$.field`                     | `$.input.field` or `$.workflow.field`  |
| `$.previous_node.result`      | `$.nodes.previous_node.result`         |
| `$.node_results.agent.result` | `$.nodes.agent.result`                 |
| `$.workflow_state.topic`      | `$.workflow.topic`                     |
| `$.final_result`              | `$.nodes.<orchestrator_id>.result`     |
| `$.orchestrator_result`       | `$.nodes.<orchestrator_id>.result`     |
| `$.stdout`                    | `$.nodes.<cli_node_id>.outputs.stdout` |

### Transform Migration

| Old Mapping       | New Mapping                                                                            |
| ----------------- | -------------------------------------------------------------------------------------- |
| `$.steps_shell`   | `{ "path": "$.nodes.planner.outputs.steps", "transform": "shell_quote" }`              |
| `$.items_json`    | `{ "path": "$.nodes.extractor.outputs.items", "transform": "json_stringify" }`         |
| `$.tags_joined`   | `{ "path": "$.nodes.extractor.outputs.tags", "transform": "join", "separator": ", " }` |
| `$.results_first` | `{ "path": "$.nodes.search.outputs.results", "transform": "first" }`                   |

### Output Migration

Old implicit final output:

```json
{
  "workflow_state": {
    "final_result": {
      "answer": "..."
    }
  }
}
```

New explicit exit projection:

```json
{
  "output_mapping": {
    "answer": "$.nodes.investigate.result.answer"
  }
}
```

New response:

```json
{
  "output": {
    "answer": "..."
  }
}
```

---

## References

- [Graph Weave AGENTS.md](../../AGENTS.md)
- [Architecture Living Spec](./README.md)
- [Runtime Specification](../runtime/README.md)
- [Workflow Schema Specification](../workflow-schema/README.md)
