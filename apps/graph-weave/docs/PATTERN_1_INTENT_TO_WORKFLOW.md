# Pattern 1: Intent → Structured Workflow

## Overview

**Pattern 1** implements deterministic workflow generation from natural language intent, following the **Compiled AI** paradigm (arXiv 2604.05150).

### Key Principle

```
LLM (Single Invocation) → IntentExtraction
                              ↓
                    [Deterministic Validators]
                              ↓
                         WorkflowSpec (DAG)
                              ↓
                    [LangGraph StateGraph]
                              ↓
                         Executable Workflow
```

**Critical**: The LLM is invoked **exactly once** to extract intent. All subsequent steps are deterministic, auditable, and reproducible.

---

## Architecture

### 1. Intent Extraction (`IntentExtraction`)

**Input**: Natural language user request  
**Output**: Structured Pydantic model

```python
from workflow_generation import IntentExtraction, IntentAction, IntentParameter

intent = IntentExtraction(
    goal="Research and summarize a topic",
    description="Search for information, fetch articles, and create a summary",

    actions=[
        IntentAction(
            id="search_topic",
            name="Search for Topic",
            description="Search the web for information",
            operator="web_search",
            parameters=[
                IntentParameter(
                    name="query",
                    type="string",
                    description="Search query",
                    required=True
                )
            ],
            timeout_seconds=30
        ),

        IntentAction(
            id="fetch_articles",
            name="Fetch Articles",
            description="Fetch full content from search results",
            operator="fetch_url",
            parameters=[...],
            dependencies=["search_topic"],
            timeout_seconds=60,
            retry_policy={"max_attempts": 3, "backoff_strategy": "exponential"}
        ),

        IntentAction(
            id="summarize",
            name="Summarize Findings",
            description="Create a comprehensive summary",
            operator="llm_call",
            parameters=[...],
            dependencies=["fetch_articles"],
            timeout_seconds=45
        )
    ],

    success_criteria=[
        "All articles fetched successfully",
        "Summary generated without errors",
        "Summary is between 100-500 words"
    ],

    constraints={
        "max_cost": 0.50,
        "max_duration": 300,  # 5 minutes
        "required_tools": ["web_search", "fetch_url", "llm_call"]
    }
)
```

### 2. Intent Validation (`IntentValidator`)

**4-Stage Validation** (Compiled AI pattern):

1. **Schema Validation**: Pydantic enforces structure
2. **Semantic Validation**: Goal/description coherence, action uniqueness
3. **Operator Availability**: Check tool registry
4. **Constraint Satisfaction**: Resource limits, required tools

### 3. Workflow Generation (`IntentToWorkflowGenerator`)

**Deterministic DAG Construction**:

1. Map actions → workflow nodes
2. Construct edges from dependencies
3. Add START/END nodes
4. Validate workflow structure

### 4. Workflow Specification (`WorkflowSpec`)

**Output**: Validated, executable DAG

---

## Testing

### Run Tests

```bash
cd apps/graph-weave
.venv/bin/python -m pytest tests/test_workflow_generation.py -v
```

### Test Coverage

- ✓ Intent validation (7 tests)
- ✓ Workflow generation (4 tests)
- ✓ Workflow validation (2 tests)
- **Total**: 13 tests, 100% pass rate

---

## References

### Academic Papers

- **Compiled AI** (arXiv 2604.05150): 4-stage validation pattern
- **A²Flow** (AAAI 2026): Operator discovery and dynamic composition
- **FlowMind** (arXiv 2602.11782): Execute-summarize pattern
- **BayesFlow** (ACL 2026 Findings): Bayesian workflow generation

### Implementation

- **Location**: `apps/graph-weave/src/workflow_generation/`
- **Schemas**: `schemas/intent.py`, `schemas/workflow.py`
- **Validators**: `validators/intent_validator.py`, `validators/workflow_validator.py`
- **Generator**: `generators/intent_to_workflow.py`
- **Tests**: `tests/test_workflow_generation.py`
- **Examples**: `src/workflow_generation/example_usage.py`
