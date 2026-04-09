# VERIFY-GW-WF-102-DOC: Multi-label rule wording

> **Linked Task** : GW-WF-102 — `docs/graph-weave/specification/workflow-schema/tasks/MOCK/GW-WF-102.md`
> **Verification Type** : DOC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09 00:00
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/README.md` (line 22)
- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` (Traceability section)

## 2. Scope Compliance

- ✅ The multi-label rule stays readable in the workflow docs.

## 3. Type-Specific Criteria

| #      | Criterion    | Expected                             | Actual   | Status |
| ------ | ------------ | ------------------------------------ | -------- | ------ |
| DOC-01 | Rule clarity | The multi-label rule is easy to read | ✅ Clear | Pass   |

## 4. Evidence

**Spec README** (line 22 - clear rule definition):

> "Requirements and features may carry inline phase labels like `[MOCK]`, `[MVP]`, and `[FULL]`; multiple labels are allowed when a requirement spans more than one phase."

The rule is stated plainly:

1. Single labels for phase-specific requirements
2. Multiple labels when spanning phases (explicitly allowed)
3. Clear examples: `[MOCK]`, `[MVP,FULL]`, `[MOCK,MVP,FULL]`

**Workflow SPEC** - No ambiguity in application:

- Requirements clearly show their applicable phases
- Separation is by comma (standard list format)
- No parsing complexity

## 5. Final Decision

| Decision        | Condition                        |
| --------------- | -------------------------------- |
| Pass            | Multi-label wording stays clear  |
| Needs Revision  | Multi-label wording is ambiguous |
| Fail + Rollback | Wording conflicts with spec      |

**Decision:** ✅ Pass - Multi-label rule is clearly stated and consistently applied
