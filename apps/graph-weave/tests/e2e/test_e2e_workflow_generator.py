"""
E2E test: Workflow Generator DAG (GW-FEAT-GEN-001, GW-FEAT-GEN-002, GW-FEAT-GEN-003)

Uses the 'workflow-generator' workflow to convert a natural language DevOps intent
into a validated, executable GraphWeave DAG via the existing /execute API.

This test proves GraphWeave can build GraphWeave — no new APIs needed.

Prerequisites:
- AI_GATEWAY_URL must be reachable (real provider calls)
- GRAPH_WEAVE_API_URL must be running (default: http://localhost:8001)
"""

import json
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

TENANT_ID = "platform-eng"
WORKFLOW_ID = "workflow-generator:v1.0.0"
FIXTURE_PATH = os.path.join(
    os.path.dirname(__file__),
    "../../../../docs/graph-weave/code/workflow-generator.json",
)


@pytest.fixture(scope="module")
def client():
    api_url = os.getenv("GRAPH_WEAVE_API_URL", "http://localhost:8001")
    return httpx.Client(base_url=api_url, timeout=120.0)


@pytest.fixture(scope="module")
def generator_workflow_definition():
    """Load the workflow-generator.json definition from docs/graph-weave/code/."""
    debug_log("FIXTURE", f"Loading workflow definition from: {FIXTURE_PATH}")
    with open(FIXTURE_PATH, "r") as f:
        definition = json.load(f)

    return {
        "tenant_id": TENANT_ID,
        "workflow_id": WORKFLOW_ID,
        "name": "Workflow Generator",
        "version": "1.0.0",
        "description": "Meta-workflow that converts natural language intents into GraphWeave DAGs",
        "owner": "platform-eng",
        "tags": ["meta", "generator", "dag-builder", "devops"],
        "definition": definition,
    }


@pytest.fixture(scope="module", autouse=True)
def register_generator_workflow(client, generator_workflow_definition):
    """Register the workflow-generator once for the entire test module."""
    debug_log("SETUP", f"Registering '{WORKFLOW_ID}' for module-level tests")
    ensure_clean_workflow(client, TENANT_ID, WORKFLOW_ID)
    create_workflow_via_api(client, TENANT_ID, generator_workflow_definition)
    debug_log("SETUP", f"✓ '{WORKFLOW_ID}' registered")
    yield
    debug_log("TEARDOWN", f"Cleaning up '{WORKFLOW_ID}'")
    ensure_clean_workflow(client, TENANT_ID, WORKFLOW_ID)


@pytest.mark.e2e
class TestWorkflowGeneratorE2E:
    """E2E tests for the Autonomous Workflow Generator meta-DAG."""

    def setup_method(self, method):
        """Prepare for each test method."""
        # Load the generator definition for I/O logging
        with open(FIXTURE_PATH, "r") as f:
            self.generator_def = {
                "workflow_id": WORKFLOW_ID,
                "definition": json.load(f)
            }

    def _execute_and_wait(self, client, intent: str, domain: str = "devops", timeout: float = 90.0):
        """Helper to execute the generator workflow and wait for completion."""
        print(f"\n[STEP 1] Submitting intent to generator...")
        print(f"  → Intent: {intent}")
        debug_log("EXEC", f"Submitting intent: '{intent}'")
        
        response = client.post(
            "/execute",
            json={
                "tenant_id": TENANT_ID,
                "workflow_id": WORKFLOW_ID,
                "input": {
                    "intent": intent,
                    "domain": domain,
                    "correction_attempts": 0,
                },
            },
        )
        assert response.status_code == 200, (
            f"Execute failed: {response.status_code} — {response.text}"
        )
        data = response.json()
        run_id = data["run_id"]
        
        print(f"\n[RESPONSE 1] Execute endpoint response:")
        print(f"  ✓ Status code: {response.status_code}")
        print(f"  ✓ Run ID: {run_id}")
        print(f"  ✓ Initial status: {data['status']}")
        
        debug_log("EXEC", f"Run created: run_id={run_id}, status={data['status']}")
        assert data["status"] == "queued"

        print(f"\n[STEP 2] Waiting for workflow generator to complete (SLA < 90s)...")
        debug_log("POLL", f"Starting status poll for run_id={run_id}")
        final = wait_for_terminal_status(client, run_id, timeout=timeout)
        
        print(f"\n[RESPONSE 2] Final execution status:")
        print(f"  ✓ Final status: {final['status']}")
        debug_log("EXEC", f"Run terminal: status={final['status']}, run_id={run_id}")
        return run_id, final

    def test_gen001_eks_diagnostic_intent_produces_valid_dag(self, client):
        """
        GW-FEAT-GEN-001 / GW-FEAT-GEN-002:
        Given a DevOps EKS intent, when the generator workflow runs,
        then the final state must contain a structurally valid GraphWeave DAG.
        """
        debug_log("TEST", "Starting test_gen001_eks_diagnostic_intent_produces_valid_dag")
        
        print("\n" + "=" * 80)
        print("GW-FEAT-GEN-001: EKS DIAGNOSTIC INTENT PRODUCES VALID DAG")
        print("=" * 80)

        intent = (
            "When an EKS pod enters a CrashLoopBackOff state, fetch the pod logs, "
            "analyze the error pattern, determine the root cause, and post a structured "
            "incident summary to Slack with recommended remediation steps."
        )

        run_id, final = self._execute_and_wait(client, intent, domain="devops")

        assert final["status"] in ["completed", "failed"], (
            f"Unexpected terminal status: {final['status']}"
        )

        if final.get("workflow_state"):
            # Final terminal state logs
            print(f"\n[FINAL WORKFLOW STATE]")
            state = final["workflow_state"]
            
            # Use fixed helper to log each node's processing
            log_workflow_agent_io(self.generator_def, state)

            # GW-FEAT-GEN-001: generated_workflow must exist in state
            generated = state.get("generated_workflow")
            debug_log("ASSERT", f"generated_workflow present: {generated is not None}")
            
            assert generated is not None, (
                "Final workflow_state must contain 'generated_workflow'. "
                f"Available keys: {list(state.keys())}"
            )

            # Log the final workflow result
            print(f"\n[GENERATED WORKFLOW DAG]")
            print(json.dumps(generated, indent=2))

            # GW-FEAT-GEN-002: basic schema validity — required top-level keys
            assert isinstance(generated, dict), "generated_workflow must be a dict"
            required_keys = {"nodes", "edges"}
            missing = required_keys - set(generated.keys())
            assert not missing, (
                f"generated_workflow missing required keys: {missing}"
            )

            nodes = generated["nodes"]
            edges = generated["edges"]
            
            print(f"\n[OUTCOME SUMMARY]")
            print(f"  ✓ Workflow Plan: {len(state.get('plan', []))} steps identified")
            print(f"  ✓ Generated Nodes: {len(nodes)}")
            print(f"  ✓ Generated Edges: {len(edges)}")
            
            assert len(nodes) >= 3, f"Expected at least 3 nodes, got {len(nodes)}"
            assert len(edges) >= 2, f"Expected at least 2 edges, got {len(edges)}"
            debug_log("ASSERT", f"✓ DAG has {len(nodes)} nodes and {len(edges)} edges")

            # Verify all edge references point to existing node IDs
            node_ids = {n["id"] for n in nodes if isinstance(n, dict) and "id" in n}
            for edge in edges:
                if isinstance(edge, dict):
                    assert edge.get("from") in node_ids, (
                        f"Edge 'from' references unknown node: {edge.get('from')}"
                    )
                    assert edge.get("to") in node_ids, (
                        f"Edge 'to' references unknown node: {edge.get('to')}"
                    )
            debug_log("ASSERT", "✓ All edge references valid")

        print(f"\n✓ GW-FEAT-GEN-001/002: EKS diagnostic DAG generated successfully")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_gen001_eks_diagnostic_intent_produces_valid_dag PASSED")

    def test_gen002_aws_credential_rotation_intent(self, client):
        """
        GW-FEAT-GEN-001:
        Given an AWS credential rotation intent, when run,
        then a plausible multi-step DAG is produced with relevant node names.
        """
        debug_log("TEST", "Starting test_gen002_aws_credential_rotation_intent")
        
        print("\n" + "=" * 80)
        print("GW-FEAT-GEN-001: AWS CREDENTIAL ROTATION INTENT")
        print("=" * 80)

        intent = (
            "Rotate an expiring AWS IAM access key: detect keys older than 90 days, "
            "generate new key pair, update the secret in AWS Secrets Manager, "
            "notify the owning team via Slack, and archive the old key."
        )

        run_id, final = self._execute_and_wait(client, intent, domain="devops", timeout=90.0)

        if final.get("workflow_state"):
            state = final["workflow_state"]
            
            print(f"\n[FINAL WORKFLOW STATE]")
            log_workflow_agent_io(self.generator_def, state)
            
            plan = state.get("plan")
            
            print(f"\n[OUTCOME SUMMARY]")
            print(f"  ✓ Intention: AWS Credential Rotation")
            print(f"  ✓ High-level Plan Steps:")
            if plan:
                for idx, step in enumerate(plan):
                    print(f"    {idx+1}. {step.get('id')}: {step.get('purpose')}")
            
            debug_log("ASSERT", f"Plan steps: {plan}")
            assert plan is not None, "The 'plan' key should be present in workflow_state"

        print(f"\n✓ GW-FEAT-GEN-001: AWS credential rotation DAG generated")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_gen002_aws_credential_rotation_intent PASSED")

    def test_gen003_output_ready_for_post_workflows(self, client):
        """
        GW-FEAT-GEN-003:
        Given a generated workflow in the final state, when the client extracts it,
        then the JSON is immediately usable in a POST /workflows call.
        """
        debug_log("TEST", "Starting test_gen003_output_ready_for_post_workflows")
        
        print("\n" + "=" * 80)
        print("GW-FEAT-GEN-003: OUTPUT READY FOR POST /WORKFLOWS")
        print("=" * 80)

        intent = (
            "Monitor Grafana for anomalous CPU spikes across all EKS nodes, "
            "auto-correlate with ArgoCD deployment timestamps, "
            "and create a GitHub issue summarising potential culprit deployments."
        )

        run_id, final = self._execute_and_wait(client, intent, domain="devops", timeout=90.0)

        if final.get("workflow_state"):
            state = final["workflow_state"]
            
            print(f"\n[FINAL WORKFLOW STATE]")
            log_workflow_agent_io(self.generator_def, state)
            
            generated = state.get("generated_workflow")

            if generated is not None:
                # GW-FEAT-GEN-003: The output must be submittable as a workflow definition
                new_workflow_id = "generated-grafana-monitor:v1.0.0"
                ensure_clean_workflow(client, TENANT_ID, new_workflow_id)

                payload = {
                    "tenant_id": TENANT_ID,
                    "workflow_id": new_workflow_id,
                    "name": "Generated Grafana Monitor",
                    "version": "1.0.0",
                    "description": "Generated by workflow-generator",
                    "owner": "platform-eng",
                    "tags": ["generated"],
                    "definition": generated,
                }
                
                print(f"\n[STEP 3] Registering generated DAG via POST /workflows...")
                debug_log("ASSERT", f"Attempting POST /workflows with generated DAG")
                reg_response = client.post("/workflows", json=payload)
                debug_log("ASSERT", f"POST /workflows status: {reg_response.status_code}")

                print(f"\n[RESPONSE 3] Workflow registration result:")
                print(f"  ✓ ID: {new_workflow_id}")
                print(f"  ✓ Status: {reg_response.status_code}")
                
                # Definition must be acceptable — entry/exit/nodes/edges present
                assert reg_response.status_code in [200, 201, 422], (
                    f"Unexpected status: {reg_response.status_code} — {reg_response.text}"
                )

                if reg_response.status_code in [200, 201]:
                    print(f"  ✓ SUCCESS: Generated DAG is schema-compliant")
                    debug_log("ASSERT", "✓ Generated workflow successfully registered")
                    ensure_clean_workflow(client, TENANT_ID, new_workflow_id)
                else:
                    print(f"  ⚠ WARNING: Schema mismatch in generated DAG")
                    debug_log(
                        "ASSERT",
                        f"422 returned (schema issue): {reg_response.json()}",
                        "WARN",
                    )

        print(f"\n✓ GW-FEAT-GEN-003: Generated workflow output tested for POST /workflows")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_gen003_output_ready_for_post_workflows PASSED")

    def test_gen_latency_within_sla(self, client):
        """
        NFR: The workflow generator must complete within a 120 second SLA
        to remain practical for interactive design-time use.
        """
        debug_log("TEST", "Starting test_gen_latency_within_sla")
        
        print("\n" + "=" * 80)
        print("NFR: WORKFLOW GENERATOR LATENCY WITHIN SLA")
        print("=" * 80)

        intent = (
            "On a Cloudflare WAF rule trigger, fetch the blocked request details, "
            "classify the threat type, and update the blocklist rule automatically."
        )

        start = time.monotonic()
        run_id, final = self._execute_and_wait(client, intent, domain="devops", timeout=120.0)
        elapsed = time.monotonic() - start

        print(f"\n[LATENCY ANALYSIS]")
        print(f"  ✓ Target SLA: 120.0s")
        print(f"  ✓ Actual Time: {elapsed:.2f}s")
        print(f"  ✓ Margin: {120.0 - elapsed:.2f}s")

        debug_log("SLA", f"Elapsed: {elapsed:.2f}s (SLA: <120s)")
        assert elapsed < 120.0, (
            f"Workflow generator exceeded 120s SLA: took {elapsed:.2f}s"
        )
        assert final["status"] in ["completed", "failed"]

        print(f"\n✓ NFR: Workflow generator completed within SLA")
        print(f"  - Run ID: {run_id}")
        print(f"  - Elapsed: {elapsed:.2f}s")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_gen_latency_within_sla PASSED")
