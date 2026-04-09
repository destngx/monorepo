# VERIFY-GW-WF-108-DOC: Workflow schema validation documentation

> **Linked Task** : GW-WF-108
> **Verification Type** : DOC
> **Phase ID** : MOCK

---

## Validator Documentation

### WorkflowValidator Class

Comprehensive validation of workflow JSON definitions against WORKFLOW_JSON_SPEC.

**Method**: validate_workflow_definition(definition: Dict[str, Any]) -> None

- Raises ValueError with specific error message if validation fails
- Returns None if validation succeeds

### Error Messages

Clear, actionable error messages:

- "Workflow definition must include 'nodes' array"
- "Node 'agent_1' has no incoming edge (not reachable)"
- "Unknown node type 'unknown_type' in node 'step_1'"
- "Entry point node 'entry' must have type='entry_node'"
- "Agent node 'agent_1' missing required system_prompt"

### Integration Points

1. WorkflowCreate model validator calls validate_workflow_definition
2. UpdateWorkflowRequest validator calls validate_workflow_definition if definition provided
3. Both return 422 Validation error if schema invalid

---

## Status

✅ Validation documentation defined and ready for implementation
