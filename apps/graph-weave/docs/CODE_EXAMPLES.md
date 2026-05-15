# Code Examples & Integration Guide

**Date**: May 15, 2026  
**Scope**: All three workflow generation patterns

---

## Quick Start

### Pattern 1: Intent → Structured Workflow

```python
from workflow_generation import (
    IntentExtraction,
    IntentAction,
    IntentParameter,
    IntentToWorkflowGenerator,
)

# Create intent
intent = IntentExtraction(
    goal="Research a topic",
    description="Search, fetch, and summarize",
    actions=[
        IntentAction(
            id="search",
            name="Search",
            description="Search the web",
            operator="web_search"
        ),
        IntentAction(
            id="fetch",
            name="Fetch",
            description="Fetch content",
            operator="fetch_url",
            dependencies=["search"]
        ),
        IntentAction(
            id="summarize",
            name="Summarize",
            description="Create summary",
            operator="llm_call",
            dependencies=["fetch"]
        )
    ]
)

# Generate workflow
generator = IntentToWorkflowGenerator()
result = generator.generate(intent)

if result["success"]:
    workflow = result["workflow"]
    print(f"✓ Workflow: {workflow.id}")
    print(f"  Nodes: {len(workflow.nodes)}")
    print(f"  Execution order: {workflow.topological_sort()}")
else:
    print(f"✗ Error: {result['errors']}")
```

### Pattern 2: Operator Registry

```python
from workflow_generation.operators import (
    get_global_registry,
    OperatorCapability,
)

# Get registry
registry = get_global_registry()

# List all operators
all_ops = registry.list_all()
print(f"Available operators: {len(all_ops)}")

# Find by capability
search_ops = registry.find_by_capability(OperatorCapability.SEARCH)
print(f"Search operators: {[op.name for op in search_ops]}")

# Search by name
web_search = registry.search("web")
print(f"Web search: {web_search[0].name if web_search else 'Not found'}")

# Get statistics
stats = registry.get_stats()
print(f"Statistics: {stats}")
```

### Pattern 3: Template-based Workflows

```python
from workflow_generation.templates import (
    WorkflowTemplate,
    TemplateVariable,
    TemplateNode,
    TemplateWorkflowGenerator,
)

# Create template
template = WorkflowTemplate(
    id="search_template",
    name="Search Template",
    description="A search workflow template",
    variables=[
        TemplateVariable(
            name="query",
            type="string",
            description="Search query",
            required=True
        )
    ],
    nodes=[
        TemplateNode(
            id="search",
            name="Search",
            description="Search the web",
            operator="web_search",
            config={"query": "{{query}}"}
        )
    ],
    start_node_id="search",
    end_node_ids=["search"]
)

# Generate workflow from template
generator = TemplateWorkflowGenerator()
result = generator.generate(template, {"query": "machine learning"})

if result["success"]:
    workflow = result["workflow"]
    print(f"✓ Workflow from template: {workflow.id}")
else:
    print(f"✗ Error: {result['errors']}")
```

---

## Advanced Examples

### Example 1: Complex Research Workflow

```python
from workflow_generation import IntentExtraction, IntentAction, IntentParameter

# Create complex intent with parallel operations
intent = IntentExtraction(
    goal="Comprehensive research analysis",
    description="Multi-source research with parallel data collection",
    actions=[
        # Parallel start: search multiple sources
        IntentAction(
            id="search_web",
            name="Search Web",
            operator="web_search",
            parameters=[
                IntentParameter(name="query", type="string", required=True),
                IntentParameter(name="max_results", type="number", default=10)
            ]
        ),
        IntentAction(
            id="search_academic",
            name="Search Academic",
            operator="academic_search",
            parameters=[
                IntentParameter(name="query", type="string", required=True)
            ]
        ),
        # Parallel fetch
        IntentAction(
            id="fetch_web",
            name="Fetch Web Results",
            operator="fetch_url",
            dependencies=["search_web"],
            timeout_seconds=120,
            retry_policy={"max_attempts": 3}
        ),
        IntentAction(
            id="fetch_academic",
            name="Fetch Academic Results",
            operator="fetch_url",
            dependencies=["search_academic"],
            timeout_seconds=120
        ),
        # Sequential analysis
        IntentAction(
            id="analyze",
            name="Analyze All Sources",
            operator="llm_call",
            dependencies=["fetch_web", "fetch_academic"],
            timeout_seconds=60
        ),
        # Final summary
        IntentAction(
            id="summarize",
            name="Create Summary",
            operator="llm_call",
            dependencies=["analyze"],
            timeout_seconds=45
        )
    ],
    success_criteria=[
        "All sources fetched",
        "Analysis complete",
        "Summary generated"
    ],
    constraints={
        "max_cost": 1.00,
        "max_duration": 600,
        "required_tools": ["web_search", "academic_search", "fetch_url", "llm_call"]
    }
)

# Generate and inspect
generator = IntentToWorkflowGenerator()
result = generator.generate(intent)

if result["success"]:
    workflow = result["workflow"]

    # Inspect structure
    print("Workflow Structure:")
    for node in workflow.nodes:
        incoming = workflow.get_incoming_edges(node.id)
        outgoing = workflow.get_outgoing_edges(node.id)
        print(f"  {node.id}: {len(incoming)} incoming, {len(outgoing)} outgoing")

    # Get execution order
    print("\nExecution Order:")
    for i, node_id in enumerate(workflow.topological_sort(), 1):
        print(f"  {i}. {node_id}")
```

### Example 2: Operator Discovery & Composition

```python
from workflow_generation.operators import (
    get_global_registry,
    OperatorCapability,
    OperatorDefinition,
)

registry = get_global_registry()

# Discover operators by capability
print("Available Capabilities:")
for capability in registry.list_capabilities():
    ops = registry.find_by_capability(capability)
    print(f"  {capability.value}: {len(ops)} operators")
    for op in ops:
        print(f"    - {op.name} (timeout: {op.timeout_seconds}s, cost: ${op.cost_per_call})")

# Find deterministic operators (safe for reproducible workflows)
all_ops = registry.list_all()
deterministic = [op for op in all_ops if op.is_deterministic]
print(f"\nDeterministic operators: {len(deterministic)}")
for op in deterministic:
    print(f"  - {op.name}")

# Find operators requiring authentication
auth_required = [op for op in all_ops if op.requires_auth]
print(f"\nOperators requiring auth: {len(auth_required)}")

# Estimate workflow cost
def estimate_workflow_cost(workflow_spec):
    """Estimate total cost of a workflow."""
    total_cost = 0
    for node in workflow_spec.nodes:
        if node.operator:
            op = registry.get(node.operator)
            if op and op.cost_per_call:
                total_cost += op.cost_per_call
    return total_cost

# Example usage
from workflow_generation import IntentToWorkflowGenerator
intent = ...  # Create intent
result = IntentToWorkflowGenerator().generate(intent)
if result["success"]:
    cost = estimate_workflow_cost(result["workflow"])
    print(f"\nEstimated workflow cost: ${cost:.2f}")
```

### Example 3: Template Composition

```python
from workflow_generation.templates import (
    WorkflowTemplate,
    TemplateVariable,
    TemplateNode,
    TemplateWorkflowGenerator,
)

# Create reusable templates
search_template = WorkflowTemplate(
    id="search",
    name="Search",
    description="Search template",
    variables=[
        TemplateVariable(name="query", type="string", required=True),
        TemplateVariable(name="max_results", type="number", default=10)
    ],
    nodes=[
        TemplateNode(
            id="search",
            name="Search",
            operator="web_search",
            config={"query": "{{query}}", "max_results": "{{max_results}}"}
        )
    ],
    start_node_id="search",
    end_node_ids=["search"]
)

fetch_template = WorkflowTemplate(
    id="fetch",
    name="Fetch",
    description="Fetch template",
    variables=[
        TemplateVariable(name="urls", type="array", required=True)
    ],
    nodes=[
        TemplateNode(
            id="fetch",
            name="Fetch",
            operator="fetch_url",
            config={"urls": "{{urls}}"}
        )
    ],
    start_node_id="fetch",
    end_node_ids=["fetch"]
)

# Use templates
generator = TemplateWorkflowGenerator()

# Generate search workflow
search_result = generator.generate(
    search_template,
    {"query": "machine learning", "max_results": 20}
)

# Generate fetch workflow
fetch_result = generator.generate(
    fetch_template,
    {"urls": ["http://example.com/1", "http://example.com/2"]}
)

print(f"Search workflow: {search_result['workflow'].id}")
print(f"Fetch workflow: {fetch_result['workflow'].id}")
```

---

## Integration Patterns

### With FastAPI

```python
from fastapi import FastAPI, HTTPException
from workflow_generation import IntentToWorkflowGenerator

app = FastAPI()

@app.post("/workflows/generate")
async def generate_workflow(intent_data: dict):
    """Generate workflow from intent."""
    try:
        # Parse intent
        intent = IntentExtraction(**intent_data)

        # Generate workflow
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)

        if not result["success"]:
            raise HTTPException(status_code=400, detail=result["errors"])

        return {
            "workflow_id": result["workflow"].id,
            "nodes": len(result["workflow"].nodes),
            "edges": len(result["workflow"].edges)
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

### With Pydantic

```python
from pydantic import BaseModel, validator
from workflow_generation import IntentExtraction

class WorkflowRequest(BaseModel):
    """Request to generate workflow."""
    goal: str
    description: str
    actions: list

    @validator("goal")
    def goal_not_empty(cls, v):
        if not v or not v.strip():
            raise ValueError("Goal cannot be empty")
        return v

    def to_intent(self) -> IntentExtraction:
        """Convert to IntentExtraction."""
        return IntentExtraction(
            goal=self.goal,
            description=self.description,
            actions=[IntentAction(**a) for a in self.actions]
        )
```

### With Logging

```python
import logging
from workflow_generation import IntentToWorkflowGenerator

logger = logging.getLogger(__name__)

def generate_workflow_with_logging(intent):
    """Generate workflow with audit logging."""
    logger.info(f"Generating workflow for goal: {intent.goal}")

    generator = IntentToWorkflowGenerator()
    result = generator.generate(intent)

    if result["success"]:
        logger.info(f"✓ Workflow generated: {result['workflow'].id}")
        logger.debug(f"  Nodes: {len(result['workflow'].nodes)}")
        logger.debug(f"  Edges: {len(result['workflow'].edges)}")
    else:
        logger.error(f"✗ Workflow generation failed: {result['errors']}")

    return result
```

---

## Testing Examples

### Unit Tests

```python
import pytest
from workflow_generation import IntentExtraction, IntentAction

def test_simple_workflow():
    """Test simple workflow generation."""
    intent = IntentExtraction(
        goal="Test",
        description="Test workflow",
        actions=[
            IntentAction(id="a1", name="A1", operator="op1"),
            IntentAction(id="a2", name="A2", operator="op2", dependencies=["a1"])
        ]
    )

    generator = IntentToWorkflowGenerator()
    result = generator.generate(intent)

    assert result["success"]
    assert len(result["workflow"].nodes) == 4  # START + 2 actions + END
    assert len(result["workflow"].edges) >= 3

def test_circular_dependency_detection():
    """Test circular dependency detection."""
    with pytest.raises(ValueError):
        IntentExtraction(
            goal="Test",
            description="Test",
            actions=[
                IntentAction(id="a1", dependencies=["a2"]),
                IntentAction(id="a2", dependencies=["a1"])
            ]
        )
```

### Integration Tests

```python
def test_end_to_end_workflow():
    """Test end-to-end workflow generation and execution."""
    # Create intent
    intent = create_research_intent()

    # Generate workflow
    generator = IntentToWorkflowGenerator()
    result = generator.generate(intent)
    assert result["success"]

    # Validate workflow
    workflow = result["workflow"]
    assert workflow.topological_sort()  # Should not raise

    # Check structure
    assert workflow.start_node_id in [n.id for n in workflow.nodes]
    assert all(end_id in [n.id for n in workflow.nodes] for end_id in workflow.end_node_ids)
```

---

## Performance Optimization

### Caching

```python
from functools import lru_cache
from workflow_generation import IntentToWorkflowGenerator

class CachedGenerator:
    """Generator with caching."""

    def __init__(self):
        self.generator = IntentToWorkflowGenerator()
        self.cache = {}

    def generate(self, intent):
        """Generate with caching."""
        # Create cache key from intent
        key = hash(intent.model_dump_json())

        if key in self.cache:
            return self.cache[key]

        result = self.generator.generate(intent)
        self.cache[key] = result
        return result
```

### Parallel Processing

```python
from concurrent.futures import ThreadPoolExecutor
from workflow_generation import IntentToWorkflowGenerator

def generate_workflows_parallel(intents, max_workers=4):
    """Generate multiple workflows in parallel."""
    generator = IntentToWorkflowGenerator()

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        results = list(executor.map(generator.generate, intents))

    return results
```

---

## Conclusion

These examples demonstrate:

- ✅ Basic usage of all three patterns
- ✅ Advanced workflow composition
- ✅ Integration with FastAPI and Pydantic
- ✅ Logging and audit trails
- ✅ Testing strategies
- ✅ Performance optimization

For more examples, see:

- `examples/pattern1_langgraph_integration.py`
- `examples/pattern3_template_examples.yaml`
- `tests/test_workflow_generation.py`
- `tests/test_operator_registry.py`
- `tests/test_template_workflow.py`
