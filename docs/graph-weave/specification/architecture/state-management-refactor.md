# State Management Refactoring Specification

> **Project**: Graph Weave  
> **Component**: Runtime / State Management  
> **Status**: Draft  
> **Author**: Sisyphus  
> **Date**: 2026-05-21

## Executive Summary

Graph-weave's current node data passing system has **9 identified pain points** stemming from duplicated patterns, stringly-typed access, and inconsistent naming. This specification proposes a **StateContext** pattern with typed accessors to eliminate complexity while maintaining backward compatibility.

**Effort**: 4 days  
**Risk**: Low → Medium (incremental)  
**Impact**: High — eliminates 6x code duplication, adds type safety, improves debuggability

---

## Table of Contents

1. [Current Architecture](#current-architecture)
2. [Pain Points Analysis](#pain-points-analysis)
3. [Best Practices Research](#best-practices-research)
4. [Proposed Architecture](#proposed-architecture)
5. [Implementation Plan](#implementation-plan)
6. [Migration Guide](#migration-guide)
7. [Success Metrics](#success-metrics)
8. [Risk Assessment](#risk-assessment)
9. [References](#references)

---

## Current Architecture

### State Model

Graph-weave uses a **shared mutable state dictionary** pattern where a single state dict flows through every node. There are two state layers:

#### Production Runtime State (`ExecutorState`)

```python
# File: apps/graph-weave/src/adapters/langgraph/helper/types.py

class ExecutorState(TypedDict):
    input: Dict[str, Any]                    # Original request payload
    step: int                                # Current hop count
    current_node: Optional[str]              # Active node ID
    node_results: Dict[str, Dict[str, Any]]  # Per-node output cache (isolated)
    workflow_state: Dict[str, Any]           # Shared accumulated state (merged)
    status: Optional[str]                    # Execution status
    hop_count: int                           # Total hops
    last_result: Optional[Dict[str, Any]]    # Last node result
    errors: List[Dict[str, Any]]             # Error accumulation
```

#### Design/Reference State (`GraphWeaveState`)

```python
# File: docs/graph-weave/code/state_schema.py

class GraphWeaveState(TypedDict):
    messages: Annotated[List[Dict[str, Any]], add_messages]
    available_skills: Dict[str, Any]
    active_mcp_contexts: Dict[str, Any]
    agent_summaries: List[Dict[str, Any]]
    routing_directive: Literal["Agent_order_tools", "Agent_return_tools", "FINISH", "FORCE_EXIT"]
    agent_payload: Dict[str, Any]
    final_response: str
    stagnation_history: Annotated[List[str], lambda x, y: (x + y)[-3:]]
    token_usage: Annotated[Dict[str, int], lambda x, y: {**x, **y}]
    remaining_steps: int
```

### Data Flow Pipeline

```
User Input
  │
  ▼
[State Init] ── input_data seeded into workflow_state
  │
  ▼
[Node A]
  ├── input_mapping: reads from state via dot-paths
  ├── executes (LLM call, tool, etc.)
  ├── returns {"node_A_output": result, "status": "completed", ...}
  ├── state["node_results"]["A"] = result      ← isolated cache
  └── state["workflow_state"].update(result)    ← merged into shared state
  │
  ▼
[Edge] ── evaluates condition against state → routes to next node
  │
  ▼
[Node B]
  ├── input_mapping: reads from node_results + workflow_state
  └── ... (repeat)
  │
  ▼
[Exit Node] ── output_mapping projects final state → returns result
```

### State Access Mechanisms

| Mechanism              | Usage                          | Location            |
| ---------------------- | ------------------------------ | ------------------- |
| `input_mapping`        | Declarative per-node config    | Workflow JSON       |
| `get_state_value()`    | Dot-notation path resolution   | `state_utils.py`    |
| `interpolate_prompt()` | Template variable substitution | `content_utils.py`  |
| `evaluate_condition()` | Edge routing conditions        | `workflow_utils.py` |

---

## Pain Points Analysis

### Pain Point 1: Duplicated `input_mapping` Resolution (6 locations)

The exact same pattern — iterate over `input_mapping`, call `_get_state_value`, fallback to `workflow_state` — is copy-pasted across 6 files with minor variations.

**Pattern A** (agent/orchestrator nodes):

```python
# agent/handler.py:39-45, orchestrator.py:28-34
input_mapping = get_field("input_mapping", {})
if input_mapping:
    agent_input_context = {}
    for key, path in input_mapping.items():
        agent_input_context[key] = self.executor._get_state_value(path, state)
else:
    agent_input_context = dict(state.get("workflow_state", {}))
```

**Pattern B** (CLI node):

```python
# cli.py:29-36
input_mapping = config.get("input_mapping") or node.get("input_mapping", {})
if input_mapping:
    cli_input_context = {}
    for key, path in input_mapping.items():
        cli_input_context[key] = self.executor._get_state_value(path, state)
else:
    cli_input_context = dict(state.get("workflow_state", {}))
```

**Pattern C** (engine utils):

```python
# engine/utils.py:41-46
input_mapping = node.get("input_mapping") or config.get("input_mapping", {})
if input_mapping:
    resolved = {}
    for key, path in input_mapping.items():
        resolved[key] = executor._get_state_value(path, state)
    return resolved
```

**Files affected**:

- `apps/graph-weave/src/adapters/langgraph/nodes/agent/handler.py`
- `apps/graph-weave/src/adapters/langgraph/nodes/orchestrator.py`
- `apps/graph-weave/src/adapters/langgraph/nodes/cli.py`
- `apps/graph-weave/src/adapters/langgraph/runtime/engine/utils.py`

---

### Pain Point 2: Inconsistent State Key Naming Conventions

Node results are stored using multiple conflicting naming patterns:

| Pattern               | Where Used                       | Example                         |
| --------------------- | -------------------------------- | ------------------------------- |
| `{node_id}_output`    | agent/handler.py:336, cli.py:114 | `"my_node_output": result_data` |
| `{node_id}_status`    | agent/handler.py:337, cli.py:115 | `"my_node_status": "completed"` |
| `result`              | agent/handler.py:334, cli.py:113 | `"result": result_data`         |
| `orchestrator_result` | orchestrator.py:70               | `"orchestrator_result": {...}`  |
| `final_result`        | loop.py:138                      | Returned from OrchestratorReAct |
| `branch_result`       | executor.py:173                  | `"branch_result": "true"`       |
| `node_id`             | Multiple files                   | Redundant field in every result |

**Impact**: Consumers must know which key to look for. No guaranteed shape.

---

### Pain Point 3: 5-Level Fallback Chain in `get_state_value()`

The 228-line `get_state_value()` function has an escalating fallback strategy:

```python
# state_utils.py - Simplified fallback chain

# Level 1: node_results lookup
if first_key in node_results:
    res = resolve_path(node_results[first_key], remaining_keys)

# Level 2: workflow_state lookup
elif first_key in workflow_state:
    res = resolve_path(workflow_state[first_key], remaining_keys)

# Level 3: root state lookup
elif first_key in state:
    res = resolve_path(state[first_key], remaining_keys)

# Level 4: Deep search across ALL values (DANGEROUS)
if res is None and "." not in clean_path:
    for container in [workflow_state, node_results]:
        for val in container.values():
            if isinstance(val, dict) and first_key in val:
                res = resolve_path(val[first_key], remaining_keys)

# Level 5: Trimming fallback for terminal scalars
if res is None and remaining_keys:
    for trim in range(1, len(remaining_keys) + 1):
        # Re-lookup in all 3 locations again
```

**Level 4 is dangerous**: O(n) deep scan across all values with no guarantee of which match is returned if multiple exist.

---

### Pain Point 4: Duplicate `interpolate_prompt()` Implementations

Two completely different implementations with incompatible semantics:

| Version       | Location                               | Behavior                                                      |
| ------------- | -------------------------------------- | ------------------------------------------------------------- |
| Graph-level   | `graph/prompts.py:5-36`                | Takes flat dict, RAISES on missing variable                   |
| Runtime-level | `runtime/base/content_utils.py:45-121` | Takes full state, SILENTLY skips missing, supports transforms |

**Impact**: Agent nodes built via `GraphBuilder.build()` use different interpolation than agents executed by `RealLangGraphExecutor`.

---

### Pain Point 5: Duplicate Condition Evaluators

| Version | Location                  | Behavior                                                 |
| ------- | ------------------------- | -------------------------------------------------------- |
| Runtime | `workflow_utils.py:6-55`  | Uses `get_state_value_cb`, string comparison fallback    |
| Graph   | `graph/evaluator.py:9-54` | Uses `jsonpath_ng`, requires `$` prefix, raises on error |

---

### Pain Point 6: Near-Identical `build()`/`build_sync()`

```python
# builder.py:14-58 (build) vs builder.py:61-94 (build_sync)
# The ONLY difference: build() compiles via WorkflowCompiler first.
# The node-building loop and edge-building loop are copy-pasted verbatim.
```

---

### Pain Point 7: Dual State Stores Confusion

In `executor.py:130-131`, every node result is written to **both** containers:

```python
state["node_results"][current_node_id] = node_result  # Per-node storage
state["workflow_state"].update(node_result)            # Flat merge into global state
```

**Consequences**:

1. Same data accessible via two different paths
2. `get_state_value()` searches `node_results` first, then `workflow_state` — confusing resolution order
3. Keys from different nodes **overwrite each other** in `workflow_state` if they share the same key name
4. Exit node handler **replaces** `workflow_state` entirely

---

### Pain Point 8: `orchestrator_result` vs `final_result` Double Wrapping

```python
# orchestrator.py:68-71
return {
    **result,                                              # includes "final_result"
    "orchestrator_result": result.get("final_result", {}), # DUPLICATE
    "node_id": node_id,
}
```

Downstream consumers must know to look for either `final_result` or `orchestrator_result`.

---

### Pain Point 9: Virtual Transform Mess

Lines 36-83 of `state_utils.py` handle 8 different virtual transforms with **two overlapping detection mechanisms**:

```python
# Suffix-based detection
elif clean_path.endswith("_joined"): ...
elif clean_path.endswith("_first"): ...
elif clean_path.endswith("_shell"): ...
elif clean_path.endswith("_json"): ...

# Method-call-based detection
if ".join(" in clean_path: ...
elif ".first(" in clean_path or clean_path.endswith(".first"): ...

# THEN redundant suffix detection AGAIN on first_key
if first_part.endswith("_joined"): ...
elif first_part.endswith("_first"): ...
```

**Impact**: Triple-overlapping detection logic makes behavior unpredictable.

---

## Best Practices Research

### LangGraph Official Patterns

#### 1. Typed State Schemas

```python
# ✅ TypedDict for development (fast iteration)
from typing_extensions import TypedDict, Annotated

class AgentState(TypedDict):
    messages: list[str]
    user_id: str
    iteration: int

# ✅ Pydantic for production (validation)
from pydantic import BaseModel, Field, field_validator

class AgentState(BaseModel):
    messages: list[str] = Field(default_factory=list)
    user_id: str
    iteration: int = 0

    @field_validator('messages')
    @classmethod
    def validate_message_count(cls, v):
        if len(v) > 10:
            return v[-10:]
        return v
```

#### 2. Reducers for Parallel Safety

```python
class WorkflowState(TypedDict):
    # Last-write-wins (default)
    status: str

    # Append to list (commutative)
    messages: Annotated[list, add_messages]

    # Sum numeric values
    total_cost: Annotated[int, operator.add]

    # Merge dicts
    metadata: Annotated[dict, lambda old, new: {**old, **new}]
```

**Key insight**: Reducers enable **parallel node execution** without race conditions.

#### 3. Context vs State Separation (v0.6+)

```python
# ✅ MUTABLE: Changes during workflow
class WorkflowState(TypedDict):
    messages: Annotated[list, add_messages]
    current_step: str

# ✅ IMMUTABLE: Set once, never changes
class WorkflowContext(TypedDict):
    user_id: str
    db_connection: object
    api_key: str

graph = StateGraph(
    state_schema=WorkflowState,
    context_schema=WorkflowContext
)

def research_node(state: WorkflowState, runtime: Runtime[WorkflowContext]) -> dict:
    user_id = runtime.context["user_id"]  # Immutable access
    return {"current_step": "done"}
```

---

### Workflow Engine Comparison

| Aspect              | Temporal             | Prefect          | Airflow            | LangGraph                 |
| ------------------- | -------------------- | ---------------- | ------------------ | ------------------------- |
| **State Model**     | Event-sourced replay | Task results     | Metadata DB + XCom | Typed channels + reducers |
| **Durability**      | Full workflow state  | Task-level only  | Task-level only    | Per-node checkpointing    |
| **Context Passing** | Typed activities     | Typed parameters | XCom (serialized)  | `context_schema` (typed)  |
| **Parallelism**     | Activity workers     | Task workers     | Celery/K8s         | Reducer-safe channels     |
| **Best For**        | Mission-critical     | Python pipelines | Scheduled ETL      | LLM agent orchestration   |

---

### Production Multi-Agent Patterns

#### Typed Accessor Class

```python
class StateAccessor:
    def __init__(self, state: State):
        self._state = state

    @property
    def last_message(self) -> str:
        return self._state["messages"][-1] if self._state["messages"] else ""

    @property
    def is_complete(self) -> bool:
        return self._state["status"] == "complete"

    def add_message(self, msg: str) -> dict:
        return {"messages": [msg]}

def node(state: State) -> dict:
    accessor = StateAccessor(state)
    if accessor.is_complete:
        return {}
    return accessor.add_message(f"Last: {accessor.last_message}")
```

---

## Proposed Architecture

### Core Concept: StateContext

Introduce a `StateContext` class that wraps state access with:

1. **Typed getters** — `get_node_output()`, `get_workflow_value()`
2. **Validated path resolution** — checks against `InputContract`
3. **Access logging** — debug what was accessed and what was missing
4. **Single resolution point** — eliminates 6x duplication

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Node Handler                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  StateContext                          │  │
│  │                                                       │  │
│  │  • get_node_output(node_id, key) → Any                │  │
│  │  • get_workflow_value(key) → Any                      │  │
│  │  • resolve_path(dot_path) → Any (validated)           │  │
│  │  • resolve_input_mapping(mapping) → Dict              │  │
│  │  • get_access_summary() → Dict[str, int]              │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│                           │                                  │
│                           ▼                                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              ExecutorState (existing)                  │  │
│  │                                                       │  │
│  │  node_results: Dict[str, Dict]                        │  │
│  │  workflow_state: Dict[str, Any]                       │  │
│  │  ... other fields                                     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### StateContext API

```python
class StateContext:
    """Typed state accessor with validation and debug logging."""

    def __init__(
        self,
        state: Dict[str, Any],
        node_input_contract: Optional['InputContract'] = None
    ):
        self.state = state
        self.contract = node_input_contract
        self._access_log: List[Dict[str, str]] = []

    def get_node_output(
        self,
        node_id: str,
        key: str,
        default: Any = None
    ) -> Any:
        """
        Typed access to a specific node's output.

        Args:
            node_id: The node identifier
            key: The output key within that node's result
            default: Value to return if not found

        Returns:
            The value from node_results[node_id][key], or default
        """
        ...

    def get_workflow_value(
        self,
        key: str,
        default: Any = None
    ) -> Any:
        """
        Typed access to accumulated workflow state.

        Args:
            key: The key in workflow_state
            default: Value to return if not found

        Returns:
            The value from workflow_state[key], or default
        """
        ...

    def resolve_path(self, path: str) -> Any:
        """
        Resolve dot-notation path with optional contract validation.

        Args:
            path: Dot-notation path (e.g., "researcher_output.summary")

        Returns:
            The resolved value

        Raises:
            ValidationError: If path not in contract (when contract provided)
        """
        ...

    def resolve_input_mapping(
        self,
        input_mapping: Dict[str, str]
    ) -> Dict[str, Any]:
        """
        Resolve a full input_mapping dict.

        Args:
            input_mapping: Dict of {local_key: state_path}

        Returns:
            Dict of {local_key: resolved_value}
        """
        ...

    def get_access_summary(self) -> Dict[str, int]:
        """
        Get summary of state accesses for debugging.

        Returns:
            Dict with "found" and "missing" counts
        """
        ...
```

---

### Standardized Node Result Schema

```python
class NodeResultPayload(TypedDict, total=False):
    """Standardized shape for all node handler return values."""

    node_id: str                                    # Required: Node identifier
    status: str                                     # Required: "completed" | "failed" | "skipped"
    result: Any                                     # Primary output data
    output_data: Dict[str, Any]                     # Additional named outputs
    tokens_used: int                                # Token consumption
    error: Optional[str]                            # Error message if failed
    metadata: Dict[str, Any]                        # Extra context
```

**Before** (inconsistent):

```python
# agent/handler.py
return {
    "node_id": node_id,
    "status": "completed",
    "result": result_data,
    "tool_calls": all_tool_calls,
    f"{node_id}_output": result_data,
    f"{node_id}_status": "completed",
    "tokens_used": total_tokens,
    "turns": turns,
}

# orchestrator.py
return {
    **result,
    "orchestrator_result": result.get("final_result", {}),
    "node_id": node_id,
}
```

**After** (standardized):

```python
# All node handlers
return NodeResultPayload(
    node_id=node_id,
    status="completed",
    result=result_data,
    output_data={"tool_calls": all_tool_calls},
    tokens_used=total_tokens,
    metadata={"turns": turns},
)
```

---

### Unified State Resolution

**Before** (5-level fallback):

```python
def get_state_value(path, state):
    # Level 1: node_results
    # Level 2: workflow_state
    # Level 3: root state
    # Level 4: Deep scan (DANGEROUS)
    # Level 5: Trim fallback
```

**After** (3-level, explicit):

```python
def get_state_value(path, state):
    # Level 1: node_results (explicit node output)
    # Level 2: workflow_state (accumulated shared state)
    # Level 3: root state (execution metadata)
    # NO deep scan, NO trim fallback
```

---

## Implementation Plan

### Phase 1: Foundation (Day 1) — Low risk, High impact

| Task | File                             | Change                                      | Priority |
| ---- | -------------------------------- | ------------------------------------------- | -------- |
| 1.1  | `runtime/base/state_context.py`  | **NEW** — Create `StateContext` class       | P0       |
| 1.2  | `runtime/base/input_resolver.py` | **NEW** — Extract `resolve_input_mapping()` | P0       |
| 1.3  | `runtime/base/state_utils.py`    | **SIMPLIFY** — Remove Level 4 & 5 fallbacks | P0       |
| 1.4  | `runtime/base/result_schema.py`  | **NEW** — Define `NodeResultPayload`        | P1       |

**Deliverable**: All node handlers can use `StateContext` instead of raw dict access.

#### Task 1.1: Create StateContext Class

```python
# File: apps/graph-weave/src/adapters/langgraph/runtime/base/state_context.py

from typing import Any, Dict, List, Optional
from pydantic import ValidationError
import logging

logger = logging.getLogger(__name__)


class StateContext:
    """Typed state accessor with validation and debug logging."""

    def __init__(
        self,
        state: Dict[str, Any],
        node_input_contract: Optional[Any] = None
    ):
        self.state = state
        self.contract = node_input_contract
        self._access_log: List[Dict[str, str]] = []

    def get_node_output(
        self,
        node_id: str,
        key: str,
        default: Any = None
    ) -> Any:
        """Typed access to a specific node's output."""
        node_results = self.state.get("node_results", {})

        if node_id not in node_results:
            self._log_access(f"node:{node_id}.{key}", "missing_node")
            return default

        node_result = node_results[node_id]

        if not isinstance(node_result, dict):
            self._log_access(f"node:{node_id}.{key}", "invalid_type")
            return default

        val = node_result.get(key, default)
        status = "found" if val is not None else "missing_key"
        self._log_access(f"node:{node_id}.{key}", status)

        return val

    def get_workflow_value(
        self,
        key: str,
        default: Any = None
    ) -> Any:
        """Typed access to accumulated workflow state."""
        workflow_state = self.state.get("workflow_state", {})
        val = workflow_state.get(key, default)
        status = "found" if val is not None else "missing"
        self._log_access(f"workflow:{key}", status)
        return val

    def resolve_path(self, path: str) -> Any:
        """Resolve dot-notation path with optional contract validation."""
        # Contract validation (if provided)
        if self.contract:
            declared_paths = set()
            for field in getattr(self.contract, 'required', []):
                if hasattr(field, 'state_path'):
                    declared_paths.add(field.state_path)
            for field in getattr(self.contract, 'optional', []):
                if hasattr(field, 'state_path'):
                    declared_paths.add(field.state_path)

            if path not in declared_paths:
                logger.warning(
                    f"Path '{path}' not declared in InputContract. "
                    f"Declared: {declared_paths}"
                )

        # Use existing get_state_value
        from .state_utils import get_state_value
        try:
            return get_state_value(path, self.state)
        except Exception as e:
            logger.error(f"Failed to resolve path '{path}': {e}")
            raise

    def resolve_input_mapping(
        self,
        input_mapping: Dict[str, str]
    ) -> Dict[str, Any]:
        """Resolve a full input_mapping dict."""
        if not input_mapping:
            # Return full workflow_state if no mapping
            return dict(self.state.get("workflow_state", {}))

        resolved = {}
        for key, path in input_mapping.items():
            try:
                resolved[key] = self.resolve_path(path)
            except Exception as e:
                logger.warning(f"Failed to resolve '{key}' -> '{path}': {e}")
                resolved[key] = None

        return resolved

    def _log_access(self, path: str, status: str):
        """Log state access for debugging."""
        self._access_log.append({
            "path": path,
            "status": status,
        })

    def get_access_summary(self) -> Dict[str, int]:
        """Get summary of state accesses for debugging."""
        return {
            "total": len(self._access_log),
            "found": sum(1 for x in self._access_log if x["status"] == "found"),
            "missing": sum(
                1 for x in self._access_log
                if x["status"] in ["missing_key", "missing_node", "missing"]
            ),
            "invalid": sum(1 for x in self._access_log if x["status"] == "invalid_type"),
        }

    def get_access_log(self) -> List[Dict[str, str]]:
        """Get full access log for detailed debugging."""
        return self._access_log.copy()
```

#### Task 1.2: Extract Input Resolver

```python
# File: apps/graph-weave/src/adapters/langgraph/runtime/base/input_resolver.py

from typing import Any, Dict, Optional
from .state_context import StateContext
import logging

logger = logging.getLogger(__name__)


def resolve_input_mapping(
    state: Dict[str, Any],
    input_mapping: Optional[Dict[str, str]] = None,
    node_input_contract: Optional[Any] = None,
) -> Dict[str, Any]:
    """
    Resolve input_mapping for a node handler.

    This is the SINGLE canonical implementation for all node handlers.
    Replaces 6 duplicated implementations.

    Args:
        state: Full execution state
        input_mapping: Dict of {local_key: state_path}, or None
        node_input_contract: Optional InputContract for validation

    Returns:
        Dict of resolved input values
    """
    ctx = StateContext(state, node_input_contract)
    return ctx.resolve_input_mapping(input_mapping or {})
```

#### Task 1.3: Simplify get_state_value()

Remove Level 4 (deep scan) and Level 5 (trim fallback) from `state_utils.py`.

**Before** (228 lines, 5 levels):

```python
def get_state_value(path, state):
    # Level 1: node_results
    # Level 2: workflow_state
    # Level 3: root state
    # Level 4: Deep scan across ALL values ← REMOVE
    # Level 5: Trim fallback ← REMOVE
```

**After** (~100 lines, 3 levels):

```python
def get_state_value(path, state):
    """Resolve dot-notation path in state. 3-level lookup only."""
    clean_path = path.strip().lstrip("$").lstrip(".")

    if not clean_path:
        return state

    # Handle virtual transforms
    clean_path, transform = _extract_transform(clean_path)

    parts = clean_path.split(".")
    first_key = parts[0]
    remaining_keys = parts[1:] if len(parts) > 1 else []

    # Level 1: node_results
    node_results = state.get("node_results", {})
    if first_key in node_results:
        result = _resolve_path(node_results[first_key], remaining_keys)
        if result is not None:
            return _apply_transform(result, transform)

    # Level 2: workflow_state
    workflow_state = state.get("workflow_state", {})
    if first_key in workflow_state:
        result = _resolve_path(workflow_state[first_key], remaining_keys)
        if result is not None:
            return _apply_transform(result, transform)

    # Level 3: root state
    if first_key in state:
        result = _resolve_path(state[first_key], remaining_keys)
        if result is not None:
            return _apply_transform(result, transform)

    # Not found
    logger.warning(f"Path '{path}' not found in state")
    return None
```

---

### Phase 2: Unify Patterns (Day 2) — Medium risk

| Task | File                             | Change                                      | Priority |
| ---- | -------------------------------- | ------------------------------------------- | -------- |
| 2.1  | `runtime/base/content_utils.py`  | **MERGE** — Single `interpolate_prompt()`   | P0       |
| 2.2  | `runtime/base/workflow_utils.py` | **MERGE** — Single condition evaluator      | P0       |
| 2.3  | `graph/builder.py`               | **EXTRACT** — Shared node/edge construction | P1       |
| 2.4  | `nodes/orchestrator.py`          | **FIX** — Remove double wrapping            | P1       |

#### Task 2.1: Unified interpolate_prompt()

Keep the runtime version (more capable), deprecate graph-level version.

```python
# File: apps/graph-weave/src/adapters/langgraph/runtime/base/content_utils.py

def interpolate_prompt(
    template: str,
    state: Dict[str, Any],
    local_context: Optional[Dict[str, Any]] = None,
) -> str:
    """
    Single canonical prompt interpolation implementation.

    Supports:
    - {variable} substitution
    - {path.to.value} dot-notation
    - {_joined}, {_first}, {_shell}, {_json} transforms
    - local_context override (takes priority over state)
    """
    if not template:
        return ""

    ctx = StateContext(state)
    result = template

    for match in re.finditer(r'\{([^}]+)\}', template):
        var_name = match.group(1)
        full_match = match.group(0)

        # Check local_context first
        if local_context and var_name in local_context:
            value = local_context[var_name]
        else:
            try:
                value = ctx.resolve_path(var_name)
            except Exception:
                logger.warning(f"Could not resolve template variable: {var_name}")
                continue

        if value is not None:
            # Apply transform if present
            value = _apply_template_transform(value, var_name)
            result = result.replace(full_match, str(value))

    return result
```

---

### Phase 3: Enforce Contracts (Day 3) — Higher risk

| Task | File                            | Change                                                | Priority |
| ---- | ------------------------------- | ----------------------------------------------------- | -------- |
| 3.1  | `models/node/create.py`         | **EXTEND** — Add `state_path` to `ContractField`      | P0       |
| 3.2  | `runtime/engine/utils.py`       | **VALIDATE** — Check `InputContract` before resolving | P0       |
| 3.3  | `runtime/base/content_utils.py` | **VALIDATE** — Check template vars against contract   | P1       |
| 3.4  | All node handlers               | **UPDATE** — Use `StateContext`                       | P0       |

#### Task 3.1: Extend ContractField

```python
# File: apps/graph-weave/src/models/node/create.py

class ContractField(BaseModel):
    """Defines a field in a node's input/output contract."""

    name: str
    type: str
    required: bool = True
    description: Optional[str] = None
    default: Optional[Any] = None
    state_path: Optional[str] = None  # NEW: Dot-notation path in state
```

#### Task 3.4: Update Node Handlers

**Example: Agent Handler**

```python
# File: apps/graph-weave/src/adapters/langgraph/nodes/agent/handler.py

from ...runtime.base.input_resolver import resolve_input_mapping

class AgentNodeHandler:
    async def execute(self, node, state, workflow):
        # ... existing setup ...

        # BEFORE (20 lines of duplicated code):
        # input_mapping = get_field("input_mapping", {})
        # if input_mapping:
        #     agent_input_context = {}
        #     for key, path in input_mapping.items():
        #         agent_input_context[key] = self.executor._get_state_value(path, state)
        # else:
        #     agent_input_context = dict(state.get("workflow_state", {}))

        # AFTER (1 line):
        agent_input_context = resolve_input_mapping(
            state,
            input_mapping=get_field("input_mapping"),
            node_input_contract=self.contract,
        )

        # ... rest of handler ...
```

---

### Phase 4: Migration Path (Day 4) — Documentation + tooling

| Task | File                | Change                                   | Priority |
| ---- | ------------------- | ---------------------------------------- | -------- |
| 4.1  | `docs/graph-weave/` | **DOCUMENT** — StateContext usage guide  | P1       |
| 4.2  | `scripts/`          | **NEW** — Auto-generate InputContract    | P2       |
| 4.3  | Tests               | **UPDATE** — Add StateContext unit tests | P0       |

---

## Migration Guide

### For Existing Workflows

Existing workflows will continue to work without changes. The refactoring is backward compatible.

### For New Workflows

Use the new patterns:

```python
# 1. Declare InputContract on your node config
node_config = {
    "type": "agent",
    "input_contract": {
        "required": [
            {"name": "query", "type": "string", "state_path": "user_query"}
        ],
        "optional": [
            {"name": "context", "type": "dict", "state_path": "researcher_output"}
        ]
    },
    "input_mapping": {
        "query": "user_query",
        "context": "researcher_output.summary"
    }
}

# 2. In your node handler, use resolve_input_mapping()
from graph_weave.adapters.langgraph.runtime.base.input_resolver import resolve_input_mapping

async def my_node_handler(node, state, workflow):
    inputs = resolve_input_mapping(
        state,
        input_mapping=node.get("input_mapping"),
        node_input_contract=node.get("input_contract"),
    )

    # Use inputs["query"], inputs["context"]
    ...

# 3. Return standardized NodeResultPayload
from graph_weave.adapters.langgraph.runtime.base.result_schema import NodeResultPayload

return NodeResultPayload(
    node_id="my_node",
    status="completed",
    result=output_data,
)
```

---

## Success Metrics

| Metric                               | Current | Target | How to Measure                               |
| ------------------------------------ | ------- | ------ | -------------------------------------------- |
| `input_mapping` resolution locations | 6       | 1      | Grep for `_get_state_value` in node handlers |
| `get_state_value()` fallback levels  | 5       | 3      | Code review of state_utils.py                |
| Duplicate `interpolate_prompt()`     | 2       | 1      | Grep for function definition                 |
| Duplicate condition evaluators       | 2       | 1      | Grep for `evaluate_condition`                |
| Node result schema compliance        | 0%      | 100%   | Runtime validation                           |
| Missing input detection              | None    | 100%   | Contract validation logs                     |

---

## Risk Assessment

| Phase   | Risk   | Likelihood | Impact | Mitigation                                                          |
| ------- | ------ | ---------- | ------ | ------------------------------------------------------------------- |
| Phase 1 | Low    | Low        | Low    | Backward compatible, existing code still works                      |
| Phase 2 | Medium | Medium     | Medium | Run full test suite after each merge                                |
| Phase 3 | Higher | Medium     | High   | Feature flag for contract validation; log warnings before enforcing |
| Phase 4 | Low    | Low        | Low    | Documentation only                                                  |

### Rollback Strategy

Each phase is independently deployable. If issues arise:

1. **Phase 1**: Remove `StateContext` usage from node handlers, revert to direct `_get_state_value` calls
2. **Phase 2**: Restore duplicate implementations from git history
3. **Phase 3**: Disable contract validation via feature flag
4. **Phase 4**: Remove documentation

---

## References

### Internal Documentation

- [Graph Weave AGENTS.md](../../AGENTS.md) — Engineering standards
- [State Schema Reference](../../../docs/graph-weave/code/state_schema.py) — Design-layer state definition
- [ExecutorState Definition](../../../apps/graph-weave/src/adapters/langgraph/helper/types.py) — Runtime state definition

### External Resources

- [LangGraph State Management](https://langchain-ai-langgraph-40.mintlify.app/concepts/state) — Official docs
- [StateGraph API Reference](https://reference.langchain.com/python/langgraph/graph/state/StateGraph) — API docs
- [Production LangGraph Patterns (2026)](https://callsphere.ai/blog/langgraph-state-machine-architecture-deep-dive-2026) — Best practices
- [Multi-Agent State Management](https://agentmarketcap.ai/blog/2026/04/10/concurrent-multi-agent-state-management) — Concurrency patterns

---

## Appendix A: File Inventory

### Files to Create

| File                             | Purpose                        |
| -------------------------------- | ------------------------------ |
| `runtime/base/state_context.py`  | StateContext class             |
| `runtime/base/input_resolver.py` | Shared resolve_input_mapping() |
| `runtime/base/result_schema.py`  | NodeResultPayload TypedDict    |

### Files to Modify

| File                             | Change                                   |
| -------------------------------- | ---------------------------------------- |
| `runtime/base/state_utils.py`    | Remove Level 4 & 5 fallbacks             |
| `runtime/base/content_utils.py`  | Unify interpolate_prompt()               |
| `runtime/base/workflow_utils.py` | Keep as canonical evaluator              |
| `graph/builder.py`               | Extract shared build logic               |
| `graph/prompts.py`               | Deprecate, use runtime version           |
| `graph/evaluator.py`             | Deprecate, use runtime version           |
| `nodes/agent/handler.py`         | Use StateContext                         |
| `nodes/orchestrator.py`          | Use StateContext, remove double wrapping |
| `nodes/cli.py`                   | Use StateContext                         |
| `runtime/engine/utils.py`        | Use StateContext                         |
| `runtime/engine/handlers.py`     | Use StateContext                         |
| `models/node/create.py`          | Add state_path to ContractField          |

---

## Appendix B: Glossary

| Term                  | Definition                                            |
| --------------------- | ----------------------------------------------------- |
| **StateContext**      | Typed wrapper around ExecutorState with validation    |
| **InputContract**     | Pydantic model declaring what inputs a node expects   |
| **NodeResultPayload** | Standardized TypedDict for node handler return values |
| **input_mapping**     | Dict mapping local parameter names to state dot-paths |
| **workflow_state**    | Accumulated shared state merged from all node outputs |
| **node_results**      | Per-node output cache (isolated by node_id)           |
