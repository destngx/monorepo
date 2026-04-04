# Use case: HR Onboarding Automation

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
