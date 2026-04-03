# Use case: DevOps Log Analyzer

1. Tenant: Platform engineering team
Trigger: Alert from monitoring system
Workflow: `devops-log-analyzer:v1.0.0`

Execution Flow
- Orchestrator loads Kubernetes + Datadog skills
- Subagent queries logs
- Detects anomaly patterns
- Returns summarized root cause
Edge Cases
- Infinite retry loops → stagnation detector
- API failure → subagent error handling

2. Tenant: `cloudcorp`
Trigger: Webhook from CloudWatch → workflow_id: "log_analyzer"
Workflow ID: `cloudcorp/log_analyzer/v1.2.0`

Execution Flow:

  - Input: 50MB of ERROR-level logs (truncated to last 1000 lines)
  - Orchestrator detects anomaly pattern → routes to SubAgent_anomaly_detector
  - Subagent runs k-means clustering on log vectors
  - Returns summary: {"anomaly_type": "throttling", "p95_latency": "2.3s"}
  - Orchestrator decides SubAgent_ticket_creator → creates PagerDuty alert
  - Circuit breaker check: If 5+ alerts in 1 min, kill switch engages

3. Tenant Profile: Internal SRE team for a Fintech application.

Trigger: PagerDuty alert indicating high latency on a Kubernetes ingress pod.

Workflow ID: `incident-analyzer:v1.0.5`

Execution:

  - Orchestrator immediately calls k8s_agent to pull recent pod logs via MCP.
  - The subagent summarizes a specific CrashLoopBackOff error.
  - Orchestrator loads `datadog_api` skill and commands metrics_agent to correlate CPU spikes.
  - Orchestrator synthesizes a root cause report and posts to Slack.

Edge Case Handled: PII in logs. OutputGuardrailNode redacts user email addresses accidentally dumped in the Kubernetes stack trace before the Slack API tool fires.
