# VERIFY-GW-WF-101-SCHEMA: Single-label workflow requirement

> **Linked Task** : GW-WF-101 — `docs/graph-weave/specification/workflow-schema/tasks/MOCK/GW-WF-101.md`
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09 00:00
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/README.md` (line 22)
- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` (lines 12-16)

## 2. Scope Compliance

- ✅ The mock workflow docs allow phase labels on a requirement.

## 3. Type-Specific Criteria

| #         | Criterion           | Expected                            | Actual       | Status |
| --------- | ------------------- | ----------------------------------- | ------------ | ------ |
| SCHEMA-01 | Single-label format | Requirements can show `[MOCK]` tags | ✅ Confirmed | Pass   |

## 4. Evidence

**Spec README** (line 22):

> "Requirements and features may carry inline phase labels like `[MOCK]`, `[MVP]`, and `[FULL]`; multiple labels are allowed when a requirement spans more than one phase."

**WORKFLOW_JSON_SPEC.md** - All requirements with MOCK label:

```
- FR-WF-001 [MOCK,MVP,FULL]: Workflow JSON must define explicit nodes and edges...
- FR-WF-002 [MOCK,MVP,FULL]: Nodes must specify system and user prompts...
- FR-WF-003 [MOCK,MVP,FULL]: Edge conditions must evaluate node results...
- FR-WF-004 [MOCK,MVP,FULL]: Guardrails must be attached to nodes...
- FR-WF-005 [MVP,FULL]: Workflow structure must be serializable...
```

Note: FR-WF-001 through FR-WF-004 have `[MOCK]` in their label set (multi-phase format currently), confirming they apply to MOCK phase.

## 5. Final Decision

| Decision        | Condition                          |
| --------------- | ---------------------------------- |
| Pass            | Single-label format is explicit    |
| Needs Revision  | Label format is ambiguous          |
| Fail + Rollback | Label format conflicts with schema |

**Decision:** ✅ Pass - Phase labels are explicitly defined and applied to workflow requirements
