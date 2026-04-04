# Use case: DevOps Log Analyzer

Tenant: platform engineering / SRE team
Trigger: alert, webhook, or pager duty incident
Workflow: `incident-analyzer:v1.0.5`

Execution flow:

- Orchestrator loads Kubernetes and observability skills
- A log subagent fetches and truncates large log sets
- Anomaly detection summarizes the root cause from logs and metrics
- A metrics subagent correlates spikes and recent alerts
- Orchestrator posts the final report to Slack or PagerDuty

Example path:

- CloudWatch or PagerDuty triggers the workflow
- The analyzer clusters recent ERROR logs and extracts a probable incident type
- If alert volume spikes, the watchdog can stop repeated escalations before they spam operators

Edge cases:

- Infinite retries -> stagnation detector
- Many alerts in a short window -> circuit breaker engages (for example, 5+ alerts per minute)
- PII in logs -> output guardrail redaction before export
