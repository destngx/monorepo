# VERIFY-GW-WF-108-SCHEMA: Workflow schema validation

> **Linked Task** : GW-WF-108
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK

---

## Validation Rules

### Required Structure

- nodes: list of node objects
- edges: list of edge objects
- entry_point: string (ID of entry node)
- exit_point: string (ID of exit node)

### Node Validation

- id: required, unique within workflow
- type: required, one of: entry_node, agent_node, branch, exit_node
- config: required, object

### Node Type Requirements

- entry_node: minimal config
- agent_node: system_prompt (str), user_prompt_template (str) required
- branch: condition (optional)
- exit_node: output_mapping (required)

### Edge Validation

- from: required, must reference existing node ID
- to: required, must reference existing node ID
- condition: optional, JSONPath expression

### Reachability Rules

- All non-entry nodes must have at least one incoming edge
- entry_point node must exist and be type=entry_node
- exit_point node must exist and be type=exit_node
- No dangling nodes (all nodes reachable from entry)

---

## Status

✅ Validation rules defined and ready for implementation
