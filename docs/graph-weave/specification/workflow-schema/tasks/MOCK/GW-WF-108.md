# GW-WF-108: Workflow schema validation

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Workflow validation system
- **Feature** : Comprehensive validation of workflow JSON definitions against WORKFLOW_JSON_SPEC
- **Rationale** : All workflow create/update operations must validate that the definition conforms to the canonical schema before storage

---

### Input

- **Data / Files** : `[[../../WORKFLOW_JSON_SPEC.md]]`, `[[../../WORKFLOW_MANAGEMENT_API.md]]`, `[[../../../architecture/system-architecture.md]]`
- **Dependencies** : GW-WF-102 (workflow schema foundation), GW-WF-103 (create endpoint)
- **External Systems**: None (mock phase)

---

### Scope

- **In Scope** :
  - Create WorkflowValidator class in src/validation.py (or extend existing validation.py)
  - Validate required workflow JSON fields: nodes, edges, entry_point, exit_point
  - Validate nodes array: each node must have id, type (entry|agent_node|branch|exit), config
  - Validate edges array: each edge must have from, to, and optional condition
  - Validate node types exist and have required configuration:
    - entry_node: minimal config
    - agent_node: must have system_prompt and user_prompt_template
    - branch: condition expression allowed but optional
    - exit_node: output_mapping required
  - Validate all edge from/to node IDs exist in nodes array
  - Validate no unreachable nodes (all non-entry nodes must have incoming edge)
  - Validate entry_point and exit_point IDs exist and have correct types
  - Return validation error with specific field and reason if any validation fails
  - Raise ValueError with descriptive message for invalid schemas

- **Out of Scope**:
  - Runtime execution validation (condition evaluation, prompt template syntax)
  - State schema validation (covered by state-schema.py)
  - LangGraph compilation validation

- **Max Increment**: WorkflowValidator class with comprehensive schema validation

---

### Approach

1. Create or extend src/validation.py with WorkflowValidator class
2. Implement validate_workflow_definition(definition: Dict[str, Any]) method
3. Validate structure: nodes (list), edges (list), entry_point (str), exit_point (str)
4. Validate each node: id (str), type (enum), config (dict)
5. Validate node type-specific config:
   - agent_node: system_prompt (str), user_prompt_template (str)
   - exit_node: output_mapping (dict or str)
6. Validate edges: from (str), to (str), optional condition
7. Validate node reachability: all non-entry nodes have incoming edge, no dangling nodes
8. Validate entry/exit points exist and have correct types
9. Integration: call validator in CreateWorkflowRequest and UpdateWorkflowRequest validators
10. Write comprehensive tests covering: valid schema, missing required fields, invalid node types, unreachable nodes, circular loops, invalid edge references

**Files to Modify/Create**:

- `apps/graph-weave/src/validation.py` — Add WorkflowValidator class and validate_workflow_definition function
- `apps/graph-weave/src/models.py` — Add validators to WorkflowCreate and UpdateWorkflowRequest that call validate_workflow_definition
- `apps/graph-weave/tests/test_workflow_validation.py` — Test WorkflowValidator (minimum 10 tests)

---

### Implementation Notes

- **API Integration**: WorkflowValidator is called from Pydantic validators in WorkflowCreate and UpdateWorkflowRequest models
- **API Tags**: Validation errors are returned from endpoints decorated with `tags=["Workflows"]` per `[[../plan/api-organization.md]]`
- **OpenAPI Schema**: Validation errors contribute to 422 Unprocessable Entity responses documented in WORKFLOW_MANAGEMENT_API.md

---

### Expected Output

- **Deliverable** : WorkflowValidator class with full schema validation
- **Format** : Python validator class callable from Pydantic models
- **Example Success** :
  - Input: Valid workflow JSON per WORKFLOW_JSON_SPEC
  - Output: No exception, validation passes

- **Example Errors** :
  - Missing nodes array: ValueError("Workflow definition must include 'nodes' array")
  - Invalid node type: ValueError("Unknown node type 'unknown_type' in node 'step_1'")
  - Unreachable node: ValueError("Node 'step_2' has no incoming edge (not reachable)")
  - Invalid edge reference: ValueError("Edge references non-existent node 'step_99'")

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-108-SCHEMA.md]]
[[../../verification/MOCK/VERIFY-GW-WF-108-DOC.md]]

---

### References

[[../../WORKFLOW_JSON_SPEC.md]] - Authoritative workflow schema specification
[[../../WORKFLOW_MANAGEMENT_API.md]] - Validation requirements for create/update endpoints
[[../../../architecture/system-architecture.md]] - Validation architecture role
