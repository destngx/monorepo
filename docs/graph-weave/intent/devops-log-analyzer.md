# Use case: DevOps Log Analyzer

## 1. Objective

- What: Turn noisy logs and alerts into a concise incident summary.
- Why: Help SRE teams detect root cause quickly without manual log hunting.
- Who: Platform engineering and SRE teams.

## 2. Scope

- In scope: log analysis, anomaly summarization, alert correlation, and incident reporting.
- Out of scope: manual triage tooling and provider-specific log parsing internals.

## 3. Specification

- The orchestrator must load observability and Kubernetes skills before analysis.
- Large log sets must be truncated before subagent analysis.
- Repeated alert loops must trip stagnation or circuit-breaker controls.
- Output must redact PII before any export target such as Slack.

## 4. Technical Plan

- Use a log subagent for clustering and summarization.
- Use a metrics subagent to correlate spikes and recent alerts.
- Stop repeated escalations when alert volume grows too quickly.

## 5. Tasks

- [ ] Load observability skills and fetch log data.
- [ ] Summarize anomaly type and root cause.
- [ ] Correlate metrics and route a final incident report.

## 6. Verification

- Given a new incident, when the workflow runs, then it should produce a concise incident summary.
- Given repeated retries or alert storms, when thresholds are exceeded, then the workflow must stop or fall back.
- Given log PII, when it is detected, then it must be redacted before export.

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
