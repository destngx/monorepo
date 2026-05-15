# Pattern Selection Guide: Decision Tree for Workflow Generation

**Date**: May 15, 2026  
**Purpose**: Help developers choose the right workflow generation pattern based on use case constraints

---

## Quick Decision Tree

```
START: What is your primary constraint?
│
├─→ [A] Need deterministic, reproducible workflows from natural language?
│   └─→ YES → Pattern 1: Intent → Structured Workflow ✓
│   └─→ NO  → Continue to [B]
│
├─→ [B] Need runtime operator discovery and dynamic capability matching?
│   └─→ YES → Pattern 2: Operator Registry ✓
│   └─→ NO  → Continue to [C]
│
├─→ [C] Need pre-defined workflow templates with variable substitution?
│   └─→ YES → Pattern 3: Template-based Workflows ✓
│   └─→ NO  → Consider hybrid approach (see below)
│
└─→ [D] Multiple constraints? → See Hybrid Patterns section
```

---

## Pattern Selection Matrix

| Constraint       | Pattern 1       | Pattern 2          | Pattern 3            | Hybrid |
| ---------------- | --------------- | ------------------ | -------------------- | ------ |
| **Determinism**  | ✓✓ (LLM-once)   | ✓ (registry-based) | ✓✓ (template-based)  | ✓      |
| **Latency**      | 6ms avg         | 3ms avg ⭐         | 5ms avg              | 3-6ms  |
| **Memory**       | <1MB            | <1MB ⭐            | <1MB                 | <1MB   |
| **Flexibility**  | ✓✓ (NL input)   | ✓✓ (dynamic)       | ✓ (template vars)    | ✓✓✓    |
| **Auditability** | ✓✓ (full trace) | ✓ (operator log)   | ✓✓ (template + vars) | ✓✓     |
| **Scalability**  | ✓ (LLM cost)    | ✓✓ (O(1) lookup)   | ✓✓ (O(n) vars)       | ✓      |
| **Compliance**   | ✓ (GDPR/SOC2)   | ✓ (GDPR/SOC2)      | ✓ (GDPR/SOC2)        | ✓      |

---

## Detailed Pattern Profiles

### Pattern 1: Intent → Structured Workflow

**Best For**:

- Natural language workflow specifications
- Research/exploration tasks with dynamic step discovery
- Systems requiring full auditability of intent→workflow mapping
- One-time or infrequent workflow generation

**Use Case Examples**:

- "Research a topic and summarize findings"
- "Analyze market data and generate report"
- "Debug system issue and create runbook"

**Strengths**:

- ✓ Deterministic (LLM called once, result cached)
- ✓ Full audit trail (intent → validation → DAG → execution)
- ✓ Circular dependency detection
- ✓ 4-stage validation (schema → semantic → operator → constraint)
- ✓ Topological sort for execution order

**Weaknesses**:

- ✗ LLM cost per workflow generation
- ✗ Requires valid operator registry
- ✗ Latency depends on LLM response time (6ms avg)

**Integration Points**:

- LangGraph: StateGraph with workflow nodes as state transitions
- FastAPI: POST `/workflows/generate` endpoint
- Logging: Full intent→workflow trace for compliance

**Example**:

```python
from workflow_generation import IntentToWorkflowGenerator

generator = IntentToWorkflowGenerator()
result = generator.generate(intent)
workflow = result["workflow"]
execution_order = workflow.topological_sort()
```

---

### Pattern 2: Operator Registry

**Best For**:

- Runtime operator discovery and capability matching
- Systems with dynamic operator sets (plugins, extensions)
- High-throughput workflow composition
- Operator-centric architectures (A²Flow style)

**Use Case Examples**:

- Plugin system with dynamic operator loading
- Microservice mesh with capability-based routing
- Agent framework with tool discovery
- MCP server with dynamic tool registration

**Strengths**:

- ✓ Fastest pattern (3ms avg latency)
- ✓ O(1) operator lookup by name/capability
- ✓ Extensible (add operators at runtime)
- ✓ Minimal memory footprint
- ✓ Global singleton for consistency

**Weaknesses**:

- ✗ Requires pre-registration of operators
- ✗ No automatic workflow generation from intent
- ✗ Manual workflow composition needed

**Integration Points**:

- MCP: Operators → MCP Tools, registry → MCP Resources
- FastAPI: GET `/operators`, POST `/operators/search`
- Logging: Operator usage statistics, capability matching

**Example**:

```python
from workflow_generation.operators import get_global_registry, OperatorCapability

registry = get_global_registry()
search_ops = registry.find_by_capability(OperatorCapability.SEARCH)
web_search = registry.search("web")[0]
```

---

### Pattern 3: Template-based Workflows

**Best For**:

- Pre-defined workflow patterns with variable substitution
- Standardized processes (ETL, data pipelines, CI/CD)
- Configuration-driven workflows
- Teams with non-technical workflow designers

**Use Case Examples**:

- ETL pipeline templates (extract → transform → load)
- Data processing workflows (ingest → process → export)
- Research pipelines (search → fetch → analyze → report)
- CI/CD workflows (build → test → deploy)

**Strengths**:

- ✓ No LLM cost (template-based)
- ✓ Deterministic (template + variables)
- ✓ Easy to version control (YAML/JSON)
- ✓ Non-technical users can modify variables
- ✓ Type-safe variable substitution

**Weaknesses**:

- ✗ Limited flexibility (bound by template structure)
- ✗ Requires pre-defined templates
- ✗ Variable substitution only (no logic)

**Integration Points**:

- YAML/JSON: Template storage and versioning
- FastAPI: POST `/workflows/instantiate` with variables
- Logging: Template + variable audit trail

**Example**:

```python
from workflow_generation.templates import TemplateWorkflowGenerator

generator = TemplateWorkflowGenerator()
workflow = generator.generate(template, variables)
```

---

## Decision Algorithm

### Step 1: Identify Primary Constraint

```python
def select_pattern(constraints: dict) -> str:
    """
    constraints = {
        "determinism": bool,           # Must be reproducible?
        "latency_critical": bool,      # <5ms required?
        "natural_language": bool,      # NL input needed?
        "dynamic_operators": bool,     # Runtime operator changes?
        "pre_defined_templates": bool, # Templates available?
        "cost_sensitive": bool,        # Minimize LLM calls?
    }
    """

    # Rule 1: Natural language input → Pattern 1
    if constraints.get("natural_language"):
        return "Pattern 1: Intent → Structured Workflow"

    # Rule 2: Dynamic operators + latency critical → Pattern 2
    if constraints.get("dynamic_operators") and constraints.get("latency_critical"):
        return "Pattern 2: Operator Registry"

    # Rule 3: Pre-defined templates + cost sensitive → Pattern 3
    if constraints.get("pre_defined_templates") and constraints.get("cost_sensitive"):
        return "Pattern 3: Template-based Workflows"

    # Rule 4: Determinism + auditability → Pattern 1
    if constraints.get("determinism"):
        return "Pattern 1: Intent → Structured Workflow"

    # Default: Pattern 3 (safest, no LLM cost)
    return "Pattern 3: Template-based Workflows"
```

### Step 2: Validate Against Constraints

```python
def validate_pattern_fit(pattern: str, constraints: dict) -> tuple[bool, list[str]]:
    """Returns (is_valid, warnings)"""

    warnings = []

    if pattern == "Pattern 1":
        if constraints.get("cost_sensitive"):
            warnings.append("⚠ Pattern 1 requires LLM calls (cost impact)")
        if not constraints.get("determinism"):
            warnings.append("⚠ Pattern 1 is deterministic; consider Pattern 2 for flexibility")

    elif pattern == "Pattern 2":
        if constraints.get("natural_language"):
            warnings.append("⚠ Pattern 2 doesn't support NL input; use Pattern 1")
        if not constraints.get("dynamic_operators"):
            warnings.append("⚠ Pattern 2 overhead unnecessary for static operators")

    elif pattern == "Pattern 3":
        if not constraints.get("pre_defined_templates"):
            warnings.append("⚠ Pattern 3 requires pre-defined templates")
        if constraints.get("natural_language"):
            warnings.append("⚠ Pattern 3 doesn't support NL input; use Pattern 1")

    return len(warnings) == 0, warnings
```

---

## Hybrid Patterns

### Hybrid 1: Intent → Registry → Execution

**Use Case**: Natural language input with dynamic operator discovery

```
Intent (NL) → Pattern 1 (generate workflow) → Pattern 2 (resolve operators) → Execute
```

**Implementation**:

```python
# Step 1: Generate workflow from intent
generator1 = IntentToWorkflowGenerator()
workflow = generator1.generate(intent)["workflow"]

# Step 2: Resolve operators from registry
registry = get_global_registry()
for node in workflow.nodes:
    operator = registry.search(node.operator_name)[0]
    node.operator_definition = operator
```

---

### Hybrid 2: Template → Intent → Execution

**Use Case**: Template-based workflows with intent-driven customization

```
Template (YAML) → Pattern 3 (instantiate) → Pattern 1 (customize) → Execute
```

**Implementation**:

```python
# Step 1: Instantiate template
template_gen = TemplateWorkflowGenerator()
workflow = template_gen.generate(template, variables)

# Step 2: Customize with intent
intent_gen = IntentToWorkflowGenerator()
customized = intent_gen.customize(workflow, additional_intent)
```

---

### Hybrid 3: Registry → Template → Execution

**Use Case**: Operator-driven template generation

```
Operators (Registry) → Pattern 2 (discover) → Pattern 3 (template) → Execute
```

**Implementation**:

```python
# Step 1: Discover operators
registry = get_global_registry()
search_ops = registry.find_by_capability(OperatorCapability.SEARCH)

# Step 2: Generate template from operators
template = create_template_from_operators(search_ops)

# Step 3: Instantiate
gen = TemplateWorkflowGenerator()
workflow = gen.generate(template, variables)
```

---

## Selection Flowchart (ASCII)

```
┌─────────────────────────────────────────┐
│  START: Workflow Generation Pattern     │
│         Selection                       │
└────────────────┬────────────────────────┘
                 │
                 ▼
        ┌────────────────┐
        │ Natural        │
        │ Language       │
        │ Input?         │
        └────┬───────┬───┘
             │       │
            YES      NO
             │       │
             ▼       ▼
        ┌────────┐  ┌──────────────┐
        │Pattern │  │ Dynamic      │
        │   1    │  │ Operators?   │
        │        │  └────┬────┬────┘
        └────────┘       │    │
                        YES   NO
                         │    │
                         ▼    ▼
                    ┌────────┐ ┌──────────────┐
                    │Pattern │ │ Pre-defined  │
                    │   2    │ │ Templates?   │
                    │        │ └────┬────┬────┘
                    └────────┘      │    │
                                   YES   NO
                                    │    │
                                    ▼    ▼
                               ┌────────┐ ┌──────────┐
                               │Pattern │ │ Hybrid   │
                               │   3    │ │ Pattern  │
                               │        │ │ (see     │
                               └────────┘ │ section) │
                                          └──────────┘
```

---

## Quick Reference Table

| Scenario                                  | Pattern | Reason              |
| ----------------------------------------- | ------- | ------------------- |
| "Generate workflow from user description" | 1       | NL input support    |
| "Find operators by capability"            | 2       | Registry lookup     |
| "Run ETL pipeline with config"            | 3       | Template-based      |
| "Plugin system with dynamic tools"        | 2       | Runtime discovery   |
| "Research task with steps"                | 1       | Intent-driven       |
| "Standardized data pipeline"              | 3       | Template reuse      |
| "Microservice routing"                    | 2       | Capability matching |
| "Compliance audit trail"                  | 1       | Full traceability   |
| "Cost-sensitive production"               | 3       | No LLM calls        |
| "Flexible, exploratory"                   | 1       | Dynamic composition |

---

## Implementation Checklist

### Before Selecting a Pattern

- [ ] Define primary constraint (determinism, latency, cost, flexibility)
- [ ] List available operators/templates
- [ ] Identify audit/compliance requirements
- [ ] Estimate workflow generation frequency
- [ ] Check LLM cost budget (if using Pattern 1)
- [ ] Verify operator registry availability (if using Pattern 2)
- [ ] Prepare templates (if using Pattern 3)

### After Selecting a Pattern

- [ ] Review pattern-specific documentation
- [ ] Run pattern examples
- [ ] Benchmark latency/memory for your use case
- [ ] Set up logging/audit trail
- [ ] Configure error handling
- [ ] Test with production-like data
- [ ] Document pattern choice and rationale

---

## References

- **Pattern 1 Details**: See `PATTERN_1_INTENT_TO_WORKFLOW.md`
- **Benchmarking**: See `BENCHMARKING_REPORT.md`
- **Security**: See `SECURITY_AND_AUDITABILITY.md`
- **Code Examples**: See `CODE_EXAMPLES.md`
- **Implementation**: See `IMPLEMENTATION_SUMMARY.md`
