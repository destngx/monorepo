# Use case: HR Onboarding Automation

1. Tenant: Enterprise HR system
Trigger: New employee created
Workflow: `hr-onboarding:v1.0.0`

Flow
- Generate documents
- Call HRIS APIs
- Send onboarding email
Edge Cases
- Missing employee data → guardrail fallback
- API timeout → retry + circuit breaker

2.Tenant: `hrflow`
Workflow ID: `hrflow/onboarding/v1.0.0`

Step-by-Step:

Input: `{"new_hire": "John Doe", "role": "engineer", "start_date": "2025-02-01"}`

Orchestrator parallel routes to 3 subagents (each isolated):

- SubAgent_IT: Create GitHub, Slack, Gmail accounts
- SubAgent_HR: Add to payroll, benefits enrollment
- SubAgent_Facilities: Assign desk, order laptop

Each returns summary; orchestrator aggregates

Guardrail: Validate all mandatory steps completed before FINISH

3. Tenant Profile: Enterprise Human Resources department.

Trigger: New hire record created in Workday.

Workflow ID: `hr-onboarding:v1.1.0`

Execution:
  - Orchestrator loads hris_system and doc_gen skills.
  - Orchestrator tells doc_agent to generate a customized PDF contract.
  - Orchestrator tells email_agent to dispatch the contract via SES.

Edge Case Handled: Token Budget. Generating massive documents consumes heavy tokens. CircuitBreakerWatchdogNode halts execution gracefully via interrupt() if the max_tokens config is breached, preventing budget blowout.
