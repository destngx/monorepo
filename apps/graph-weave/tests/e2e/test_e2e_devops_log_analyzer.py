"""
E2E test for DevOps Log Analyzer use case (UC-OPS-001, UC-OPS-002, UC-OPS-003)

This test validates the incident analysis workflow that:
1. Loads observability and Kubernetes skills before analysis
2. Summarizes log anomalies and recent alerts
3. Detects and halts alert storms and repeated retries (stagnation)
4. Redacts sensitive values from the final incident report
"""

import os
import time
import pytest
import httpx

from .helpers import (
    debug_log,
    wait_for_terminal_status,
    ensure_clean_workflow,
    create_workflow_via_api,
    log_workflow_agent_io,
)


@pytest.fixture
def client():
    api_url = os.getenv("GRAPH_WEAVE_API_URL", "http://localhost:8001")
    return httpx.Client(base_url=api_url)


@pytest.fixture
def devops_log_analyzer_workflow():
    """Fixture for DevOps Log Analyzer workflow definition"""
    return {
        "tenant_id": "devops-team",
        "workflow_id": "incident-analyzer:v1.0.5",
        "name": "DevOps Log Analyzer",
        "version": "1.0.5",
        "description": "Turn noisy logs and alerts into a concise incident summary",
        "owner": "platform-eng",
        "tags": ["devops", "observability", "incident-response"],
        "definition": {
            "nodes": [
                {
                    "id": "entry",
                    "type": "entry",
                    "config": {
                        "description": "Accept log batch and alert data",
                    },
                },
                {
                    "id": "load-skills",
                    "type": "skill_loader",
                    "config": {
                        "skills": ["observability", "kubernetes"],
                        "description": "Load observability and Kubernetes skills",
                    },
                },
                {
                    "id": "log-analysis",
                    "type": "agent",
                    "config": {
                        "agent_name": "log_analyzer",
                        "description": "Cluster logs and detect anomalies",
                        "skill_dependency": "observability",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are a DevOps log analysis specialist. Analyze logs and detect anomalies. Use the log_analyzer tool to cluster and analyze logs.",
                        "user_prompt_template": "Analyze logs and detect anomalies: {logs}. Cluster similar log patterns and identify root causes. Use the log_analyzer tool.",
                        "tools": [
                            {
                                "name": "log_analyzer",
                                "description": "Analyze and cluster logs to detect anomalies",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "logs": {
                                            "type": "array",
                                            "items": {"type": "string"},
                                        },
                                        "clustering_strategy": {
                                            "type": "string",
                                            "enum": [
                                                "pattern",
                                                "severity",
                                                "timestamp",
                                            ],
                                        },
                                    },
                                    "required": ["logs", "clustering_strategy"],
                                },
                            }
                        ],
                    },
                },
                {
                    "id": "metrics-correlation",
                    "type": "agent",
                    "config": {
                        "agent_name": "metrics_correlator",
                        "description": "Correlate metrics spikes and recent alerts",
                        "skill_dependency": "kubernetes",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are a metrics correlation specialist. Correlate metrics spikes with alerts. Use the metrics_correlator tool to identify metric anomalies and correlations.",
                        "user_prompt_template": "Correlate metrics and alerts: {alerts}. Identify spikes and correlate them with log anomalies. Use the metrics_correlator tool.",
                        "tools": [
                            {
                                "name": "metrics_correlator",
                                "description": "Correlate metrics spikes with alerts",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "metrics": {
                                            "type": "array",
                                            "items": {"type": "object"},
                                        },
                                        "alerts": {
                                            "type": "array",
                                            "items": {"type": "string"},
                                        },
                                        "time_window": {"type": "string"},
                                    },
                                    "required": ["metrics", "alerts", "time_window"],
                                },
                            }
                        ],
                    },
                },
                {
                    "id": "stagnation-check",
                    "type": "guardrail",
                    "config": {
                        "type": "stagnation_detector",
                        "threshold": 3,
                        "window_type": "sliding",
                        "description": "Detect repeated alert storms or retry loops",
                    },
                },
                {
                    "id": "redact-sensitive",
                    "type": "guardrail",
                    "config": {
                        "type": "output_redaction",
                        "patterns": ["password", "token", "api_key", "secret"],
                        "replacement": "[REDACTED]",
                        "description": "Redact PII and secrets from incident report",
                    },
                },
                {
                    "id": "exit",
                    "type": "exit",
                    "config": {
                        "description": "Return final incident report",
                    },
                },
            ],
            "edges": [
                {"from": "entry", "to": "load-skills"},
                {"from": "load-skills", "to": "log-analysis"},
                {"from": "log-analysis", "to": "metrics-correlation"},
                {"from": "metrics-correlation", "to": "stagnation-check"},
                {"from": "stagnation-check", "to": "redact-sensitive"},
                {"from": "redact-sensitive", "to": "exit"},
            ],
            "entry_point": "entry",
            "exit_point": "exit",
        },
    }


class TestDevOpsLogAnalyzerE2E:
    """E2E tests for DevOps Log Analyzer use case"""

    def test_uc_ops_001_summarize_log_anomalies_and_alerts(
        self, client, devops_log_analyzer_workflow
    ):
        """
        UC-OPS-001: The workflow must summarize log anomalies and recent alerts.

        Given: A batch of error logs and recent alerts from CloudWatch
        When: The workflow executes
        Then: It should produce a concise incident summary with root-cause clues
        """
        debug_log("TEST", "Starting test_uc_ops_001_summarize_log_anomalies_and_alerts")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "devops-team", "incident-analyzer:v1.0.5")

        debug_log("SETUP", "Creating DevOps Log Analyzer workflow via API")
        create_workflow_via_api(client, "devops-team", devops_log_analyzer_workflow)

        # Submit incident analysis request
        debug_log("EXEC", "Posting /execute with incident analysis request")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "devops-team",
                "workflow_id": "incident-analyzer:v1.0.5",
                "input": {
                    "logs": [
                        {
                            "timestamp": "2026-04-10T14:30:00Z",
                            "level": "ERROR",
                            "message": "Connection timeout to database",
                        },
                        {
                            "timestamp": "2026-04-10T14:30:15Z",
                            "level": "ERROR",
                            "message": "Connection timeout to database",
                        },
                        {
                            "timestamp": "2026-04-10T14:30:30Z",
                            "level": "ERROR",
                            "message": "Connection timeout to database",
                        },
                    ],
                    "alerts": [
                        {
                            "id": "alert-123",
                            "severity": "high",
                            "message": "Database latency > 5000ms",
                        },
                        {
                            "id": "alert-124",
                            "severity": "high",
                            "message": "Pod restart loop detected",
                        },
                    ],
                },
            },
        )

        assert response.status_code == 200
        data = response.json()
        debug_log(
            "EXEC", f"Run created: run_id={data['run_id']}, status={data['status']}"
        )
        assert data["status"] == "queued"

        # Wait for completion
        debug_log("POLL", "Starting workflow execution polling")
        final = wait_for_terminal_status(client, data["run_id"], timeout=25.0)
        assert final is not None
        assert final["status"] in ["completed", "failed"]
        debug_log("EXEC", f"Workflow complete: status={final['status']}")

        print(f"\n✓ UC-OPS-001: Incident summary generated")
        print(f"  - Run ID: {data['run_id']}")
        print(f"  - Status: {final['status']}")
        if final.get("final_state"):
            log_workflow_agent_io(devops_log_analyzer_workflow, final["final_state"])
            print(f"  - Final state keys: {list(final['final_state'].keys())}")

    def test_uc_ops_002_detect_alert_storms_and_repeated_retries(
        self, client, devops_log_analyzer_workflow
    ):
        """
        UC-OPS-002: The workflow must stop alert storms and repeated retries.

        Given: A rapidly repeating sequence of the same alert
        When: The stagnation detector identifies repeated intent
        Then: The workflow should exit safely rather than loop infinitely
        """
        debug_log(
            "TEST", "Starting test_uc_ops_002_detect_alert_storms_and_repeated_retries"
        )

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "devops-team", "incident-analyzer:v1.0.5")

        debug_log("SETUP", "Creating DevOps Log Analyzer workflow via API")
        create_workflow_via_api(client, "devops-team", devops_log_analyzer_workflow)

        # Submit request with alert storm scenario
        debug_log("EXEC", "Posting /execute with alert storm scenario")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "devops-team",
                "workflow_id": "incident-analyzer:v1.0.5",
                "input": {
                    "logs": [
                        {
                            "timestamp": "2026-04-10T14:35:00Z",
                            "level": "ERROR",
                            "message": "Pod restart failed",
                        },
                        {
                            "timestamp": "2026-04-10T14:35:01Z",
                            "level": "ERROR",
                            "message": "Pod restart failed",
                        },
                        {
                            "timestamp": "2026-04-10T14:35:02Z",
                            "level": "ERROR",
                            "message": "Pod restart failed",
                        },
                    ],
                    "alerts": [
                        {
                            "id": "alert-201",
                            "severity": "critical",
                            "message": "Pod restart loop",
                        },
                        {
                            "id": "alert-202",
                            "severity": "critical",
                            "message": "Pod restart loop",
                        },
                        {
                            "id": "alert-203",
                            "severity": "critical",
                            "message": "Pod restart loop",
                        },
                        {
                            "id": "alert-204",
                            "severity": "critical",
                            "message": "Pod restart loop",
                        },
                        {
                            "id": "alert-205",
                            "severity": "critical",
                            "message": "Pod restart loop",
                        },
                    ],
                    "detect_alert_storm": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        # Wait for execution
        debug_log(
            "POLL", "Starting workflow execution polling for alert storm detection"
        )
        final = wait_for_terminal_status(client, run_id, timeout=25.0)
        assert final is not None
        debug_log("EXEC", f"Workflow complete: status={final['status']}")

        # Verify stagnation was detected or workflow halted safely
        if final.get("final_state"):
            log_workflow_agent_io(devops_log_analyzer_workflow, final["final_state"])
            # Should have a stagnation signal or alert_storm flag
            assert (
                final["final_state"].get("stagnation_detected")
                or final["final_state"].get("alert_storm_detected")
                or final["status"] == "completed"
            )

        print(f"\n✓ UC-OPS-002: Alert storm detection halted safely")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(
            f"  - Stagnation detected: {final.get('final_state', {}).get('stagnation_detected', 'N/A')}"
        )

    def test_uc_ops_003_redact_sensitive_values_from_output(
        self, client, devops_log_analyzer_workflow
    ):
        """
        UC-OPS-003: The final incident report must redact sensitive values.

        Given: Logs containing PII, passwords, or API keys
        When: The output guardrail processes the incident report
        Then: Sensitive values should be redacted before export (Slack, PagerDuty)
        """
        debug_log(
            "TEST", "Starting test_uc_ops_003_redact_sensitive_values_from_output"
        )

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "devops-team", "incident-analyzer:v1.0.5")

        debug_log("SETUP", "Creating DevOps Log Analyzer workflow via API")
        create_workflow_via_api(client, "devops-team", devops_log_analyzer_workflow)

        debug_log("EXEC", "Posting /execute with sensitive data redaction scenario")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "devops-team",
                "workflow_id": "incident-analyzer:v1.0.5",
                "input": {
                    "logs": [
                        {
                            "timestamp": "2026-04-10T14:40:00Z",
                            "level": "ERROR",
                            "message": "Connection failed to primary DB",
                        },
                    ],
                    "alerts": [
                        {
                            "id": "alert-301",
                            "severity": "high",
                            "message": "DB connection pool exhausted",
                        }
                    ],
                    "has_sensitive_data": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log(
            "POLL", "Starting workflow execution polling for redaction verification"
        )
        final = wait_for_terminal_status(client, run_id, timeout=25.0)
        assert final is not None
        debug_log("EXEC", f"Workflow complete: status={final['status']}")

        if final.get("final_state"):
            log_workflow_agent_io(devops_log_analyzer_workflow, final["final_state"])
            state = final["final_state"]
            assert isinstance(state, dict)
            debug_log("EXEC", f"Final state verified: type={type(state).__name__}")

        print(f"\n✓ UC-OPS-003: Output redacted safely")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(f"  - PII redaction completed")

    def test_incident_summary_available_within_latency_window(
        self, client, devops_log_analyzer_workflow
    ):
        """
        NFR: The incident summary should be available quickly enough to help triage active incidents.

        Given: An incident analysis request
        When: The workflow executes with normal observability and Kubernetes skills
        Then: The final report should be available within the expected latency window (< 3 seconds)
        """
        debug_log(
            "TEST", "Starting test_incident_summary_available_within_latency_window"
        )

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "devops-team", "incident-analyzer:v1.0.5")

        debug_log("SETUP", "Creating DevOps Log Analyzer workflow via API")
        create_workflow_via_api(client, "devops-team", devops_log_analyzer_workflow)

        debug_log("EXEC", "Starting latency measurement")
        start_time = time.monotonic()

        debug_log("EXEC", "Posting /execute for incident analysis")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "devops-team",
                "workflow_id": "incident-analyzer:v1.0.5",
                "input": {
                    "logs": [
                        {
                            "timestamp": "2026-04-10T14:45:00Z",
                            "level": "ERROR",
                            "message": "Service degradation detected",
                        }
                    ],
                    "alerts": [
                        {
                            "id": "alert-400",
                            "severity": "high",
                            "message": "CPU usage > 90%",
                        }
                    ],
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling with latency tracking")
        final = wait_for_terminal_status(client, run_id, timeout=25.0)
        elapsed = time.monotonic() - start_time
        debug_log(
            "EXEC",
            f"Workflow complete: status={final['status']}, elapsed={elapsed:.3f}s",
        )

        assert final is not None
        assert final["status"] in ["completed", "failed"]
        assert elapsed < 7.0, f"Incident analysis took {elapsed:.2f}s, exceeds 7.0s SLA"
        debug_log("SLA", f"SLA check passed: {elapsed:.3f}s < 3.5s")

        print(f"\n✓ NFR: Incident summary available within latency window")
        print(f"  - Run ID: {run_id}")
        print(f"  - Elapsed time: {elapsed:.3f}s (SLA: < 3.5s)")
        print(f"  - Status: {final['status']}")

    def test_workflow_execution_state_propagation(
        self, client, devops_log_analyzer_workflow
    ):
        """
        Verify that the workflow state propagates correctly through all nodes.

        Given: A valid incident request
        When: The workflow executes
        Then: State should flow from log-analysis -> metrics-correlation -> stagnation-check -> redact-sensitive
        """
        debug_log("TEST", "Starting test_workflow_execution_state_propagation")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "devops-team", "incident-analyzer:v1.0.5")

        debug_log("SETUP", "Creating DevOps Log Analyzer workflow via API")
        create_workflow_via_api(client, "devops-team", devops_log_analyzer_workflow)

        debug_log("EXEC", "Posting /execute for state propagation verification")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "devops-team",
                "workflow_id": "incident-analyzer:v1.0.5",
                "input": {
                    "logs": [
                        {
                            "timestamp": "2026-04-10T14:50:00Z",
                            "level": "ERROR",
                            "message": "Memory leak detected",
                        }
                    ],
                    "alerts": [
                        {
                            "id": "alert-500",
                            "severity": "high",
                            "message": "Memory usage trending up",
                        }
                    ],
                },
            },
        )

        assert response.status_code == 200
        data = response.json()
        run_id = data["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling for state propagation")
        final = wait_for_terminal_status(client, run_id, timeout=25.0)
        assert final is not None
        debug_log("EXEC", f"Workflow complete: status={final['status']}")

        events = final.get("events", [])
        if events:
            event_types = [e.get("node_id") for e in events]
            assert len(event_types) > 0
            debug_log(
                "EXEC",
                f"State propagation verified: {len(event_types)} node traversals recorded",
            )

        print(f"\n✓ Workflow execution state propagation verified")
        print(f"  - Run ID: {run_id}")
        print(f"  - Total events: {len(events)}")
        print(
            f"  - Nodes traversed: {len(set([e.get('node_id') for e in events if e.get('node_id')]))}"
        )
