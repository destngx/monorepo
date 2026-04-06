# Use case: HR Onboarding Automation

## 1. Objective

- What: Automate onboarding across HR, IT, and facilities while keeping the workflow auditable.
- Why: Reduce manual work and ensure every onboarding step completes reliably.
- Who: Enterprise HR operations teams.

## 2. Scope

- In scope: hire validation, account creation, payroll/benefits enrollment, contract generation, and email dispatch.
- Out of scope: manual one-off onboarding tasks and untracked side effects.

## 3. Specification

- The workflow must validate the hire payload before any downstream action.
- Subagents for IT, HR, and facilities must remain isolated.
- Mandatory steps must be verified before finish.
- Large document generation must be constrained by token watchdog behavior.

## 4. Technical Plan

- Fan out onboarding responsibilities across specialized subagents.
- Use doc-generation and email dispatch as separate steps.
- Stop gracefully if HRIS timing or token budgets are exceeded.

## 5. Tasks

- [ ] Validate the incoming hire record.
- [ ] Execute IT/HR/facilities onboarding branches.
- [ ] Generate contract and send the onboarding packet.

## 6. Verification

- Given a new hire, when the workflow runs, then the onboarding packet should be produced and dispatched.
- Given missing employee data, when validation fails, then the workflow must fall back safely.
- Given a token budget breach, when document generation expands too far, then execution must halt cleanly.

Tenant: enterprise HR platform
Trigger: a new hire record is created in Workday
Workflow: `hr-onboarding:v1.1.0`

Execution flow:

- Input guardrail validates the hire payload
- Graph initializer loads the workflow and Tier 1 HR/document skills
- Orchestrator fans out to isolated subagents
  - IT: create accounts and access
  - HR: enroll payroll and benefits
  - Facilities: assign desk and equipment
- Doc generation creates the contract and onboarding packet
- Email subagent dispatches the package
- Orchestrator aggregates summaries and verifies every mandatory step before finish

Example path:

- A new hire arrives in Workday
- The workflow generates the contract, provisions tools, and sends the onboarding packet automatically

Edge cases:

- Missing employee data -> guardrail fallback
- HRIS timeout -> retry with circuit breaker protection
- Large document generation -> max token watchdog exit before budget blowout
