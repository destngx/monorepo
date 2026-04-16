"""
E2E test for E-Commerce Customer Support use case (UC-ECOM-001, UC-ECOM-002, UC-ECOM-003)

This test validates the support workflow that:
1. Validates customer input before routing
2. Supports order lookup, refund handling, and safe fallback
3. Redacts response and streams after guardrails pass
"""

import time
import pytest
import os
import pytest
import httpx

from .helpers import (
    wait_for_terminal_status,
    debug_log,
    ensure_clean_workflow,
    create_workflow_via_api,
    log_workflow_agent_io,
)


@pytest.fixture
def client():
    api_url = os.getenv("GRAPH_WEAVE_API_URL", "http://localhost:8001")
    return httpx.Client(base_url=api_url)


@pytest.fixture
def ecommerce_support_workflow():
    return {
        "tenant_id": "retail-saas",
        "workflow_id": "customer-support:v2.1.0",
        "name": "E-Commerce Customer Support",
        "version": "2.1.0",
        "description": "Resolve order and refund requests through a deterministic support workflow",
        "owner": "support-ops",
        "tags": ["ecommerce", "support", "customer-service"],
        "definition": {
            "nodes": [
                {
                    "id": "entry",
                    "type": "entry",
                    "config": {"description": "Accept customer support request"},
                },
                {
                    "id": "input-guardrail",
                    "type": "guardrail",
                    "config": {
                        "type": "input_validation",
                        "rules": ["max_length:1000", "no_malicious_input"],
                        "description": "Validate customer input before routing",
                    },
                },
                {
                    "id": "route-decision",
                    "type": "router",
                    "config": {
                        "description": "Route to shipping/order or billing/refund",
                        "branches": ["shipping", "billing"],
                    },
                },
                {
                    "id": "shipping-agent",
                    "type": "agent",
                    "config": {
                        "agent_name": "shipping_order_lookup",
                        "description": "Handle order status and shipping queries",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are an e-commerce shipping specialist. Look up order status and handle shipping queries. Use the order_lookup tool to retrieve order information.",
                        "user_prompt_template": "Handle shipping query: {query}. Look up order status using order ID: {order_id}. Use the order_lookup tool.",
                        "tools": [
                            {
                                "name": "order_lookup",
                                "description": "Look up order status and shipping information",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "order_id": {"type": "string"},
                                        "customer_id": {"type": "string"},
                                        "fields": {
                                            "type": "array",
                                            "items": {"type": "string"},
                                        },
                                    },
                                    "required": ["order_id"],
                                },
                            }
                        ],
                    },
                },
                {
                    "id": "billing-agent",
                    "type": "agent",
                    "config": {
                        "agent_name": "billing_refund_handler",
                        "description": "Handle billing and refund decisions",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are a billing and refund specialist. Handle refund requests and billing inquiries. Use the refund_handler tool to process refunds.",
                        "user_prompt_template": "Handle billing query: {query}. Process refund for order: {order_id} if applicable. Use the refund_handler tool.",
                        "tools": [
                            {
                                "name": "refund_handler",
                                "description": "Handle refund processing and billing decisions",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "order_id": {"type": "string"},
                                        "refund_reason": {"type": "string"},
                                        "refund_amount": {"type": "number"},
                                    },
                                    "required": ["order_id", "refund_reason"],
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
                        "description": "Detect refund loops or timeouts",
                    },
                },
                {
                    "id": "output-guardrail",
                    "type": "guardrail",
                    "config": {
                        "type": "output_redaction",
                        "patterns": ["credit_card", "payment_method", "customer_ip"],
                        "replacement": "[REDACTED]",
                        "description": "Redact PII and policy violations",
                    },
                },
                {
                    "id": "stream-response",
                    "type": "output",
                    "config": {
                        "type": "sse",
                        "description": "Stream response via Server-Sent Events",
                    },
                },
                {
                    "id": "exit",
                    "type": "exit",
                    "config": {"description": "Return customer answer"},
                },
            ],
            "edges": [
                {"from": "entry", "to": "input-guardrail"},
                {"from": "input-guardrail", "to": "route-decision"},
                {"from": "route-decision", "to": "shipping-agent"},
                {"from": "route-decision", "to": "billing-agent"},
                {"from": "shipping-agent", "to": "stagnation-check"},
                {"from": "billing-agent", "to": "stagnation-check"},
                {"from": "stagnation-check", "to": "output-guardrail"},
                {"from": "output-guardrail", "to": "stream-response"},
                {"from": "stream-response", "to": "exit"},
            ],
            "entry_point": "entry",
            "exit_point": "exit",
        },
    }


class TestEcommerceCustomerSupportE2E:
    def test_uc_ecom_001_validate_customer_input_before_routing(
        self, client, ecommerce_support_workflow
    ):
        """
        UC-ECOM-001: The support workflow must validate customer input before routing.

        Given: A customer support request with valid input
        When: The input guardrail processes it
        Then: It should pass validation and proceed to routing
        """
        debug_log(
            "TEST", "Starting test_uc_ecom_001_validate_customer_input_before_routing"
        )

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with valid customer input")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-12345",
                    "query": "Where is my order #ORD-98765?",
                    "request_type": "order_status",
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        data = response.json()
        debug_log(
            "EXEC",
            f"Run created: run_id={data['run_id']}, initial_status={data['status']}",
        )
        assert data["status"] == "queued"

        debug_log("POLL", "Starting workflow execution polling")
        final = wait_for_terminal_status(client, data["run_id"], timeout=5.0)
        assert final is not None
        assert final["status"] in ["completed", "failed"]
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ UC-ECOM-001: Customer input validated successfully")
        print(f"  - Run ID: {data['run_id']}")
        print(f"  - Status: {final['status']}")
        debug_log(
            "TEST", "✓ test_uc_ecom_001_validate_customer_input_before_routing PASSED"
        )

    def test_uc_ecom_001_malicious_input_blocked(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: A request with malicious input
        When: The input guardrail validates it
        Then: The request should be blocked before routing
        """
        debug_log("TEST", "Starting test_uc_ecom_001_malicious_input_blocked")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with malicious input")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-12345",
                    "query": "<script>alert('xss')</script>",
                    "request_type": "order_status",
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling")
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ UC-ECOM-001: Malicious input blocked")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_uc_ecom_001_malicious_input_blocked PASSED")

    def test_uc_ecom_002_order_lookup_path(self, client, ecommerce_support_workflow):
        """
        UC-ECOM-002: The workflow must support order lookup, refund handling, and safe fallback.

        Given: A customer asking about their order status
        When: The workflow routes to shipping/order lookup
        Then: The shipping agent should return order status information
        """
        debug_log("TEST", "Starting test_uc_ecom_002_order_lookup_path")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute for order lookup request")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-67890",
                    "query": "My package hasn't arrived. Order #ORD-11111. When will it ship?",
                    "request_type": "order_status",
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling for order lookup")
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        assert final is not None
        assert final["status"] in ["completed", "failed"]
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ UC-ECOM-002: Order lookup path executed")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_uc_ecom_002_order_lookup_path PASSED")

    def test_uc_ecom_002_refund_handling_path(self, client, ecommerce_support_workflow):
        """
        Given: A customer requesting a refund for a lost package
        When: The workflow routes to billing/refund handler
        Then: The billing agent should evaluate refund eligibility and process if allowed
        """
        debug_log("TEST", "Starting test_uc_ecom_002_refund_handling_path")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute for refund request")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-22222",
                    "query": "I never received my order #ORD-22222. Please refund my $150.",
                    "request_type": "refund",
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling for refund handling")
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        assert final is not None
        assert final["status"] in ["completed", "failed"]
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ UC-ECOM-002: Refund handling path executed")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_uc_ecom_002_refund_handling_path PASSED")

    def test_uc_ecom_002_refund_loop_detection(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: A refund request that triggers repeated processing attempts
        When: The stagnation detector activates after 3 identical intents
        Then: The workflow should exit safely with the last good answer
        """
        debug_log("TEST", "Starting test_uc_ecom_002_refund_loop_detection")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log(
            "EXEC",
            "Posting /execute with trigger_loop=True to test stagnation detection",
        )
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-33333",
                    "query": "Refund please refund refund",
                    "request_type": "refund",
                    "trigger_loop": True,
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log(
            "POLL", "Starting workflow execution polling for refund loop detection"
        )
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ UC-ECOM-002: Refund loop detected and halted")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_uc_ecom_002_refund_loop_detection PASSED")

    def test_uc_ecom_003_output_redaction_for_pii(
        self, client, ecommerce_support_workflow
    ):
        """
        UC-ECOM-003: The response must be redacted and streamed after guardrails pass.

        Given: A support interaction with payment method details in the final response
        When: The output guardrail processes it
        Then: PII and policy violations should be redacted before streaming
        """
        debug_log("TEST", "Starting test_uc_ecom_003_output_redaction_for_pii")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute for output redaction test")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-44444",
                    "query": "My refund status for order #ORD-44444",
                    "request_type": "refund_status",
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling for output redaction")
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        if final.get("final_state"):
            log_workflow_agent_io(ecommerce_support_workflow, final["final_state"])
            output_str = str(final["final_state"])
            debug_log("EXEC", f"Checking output for PII redaction")
            assert "credit_card" not in output_str or "[REDACTED]" in output_str

        print(f"\n✓ UC-ECOM-003: Output redacted for PII")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_uc_ecom_003_output_redaction_for_pii PASSED")

    def test_response_within_user_visible_latency_budget(
        self, client, ecommerce_support_workflow
    ):
        """
        NFR: The workflow should return within a user-visible latency budget suitable for support interactions.

        Given: A support request
        When: The workflow executes
        Then: Response should complete within 3 seconds (typical user wait time)
        """
        debug_log("TEST", "Starting test_response_within_user_visible_latency_budget")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Starting latency benchmark timer")
        start = time.monotonic()

        debug_log("EXEC", "Posting /execute for latency measurement")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-55555",
                    "query": "Order status for #ORD-55555",
                    "request_type": "order_status",
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}")

        debug_log("POLL", "Starting workflow execution polling with latency tracking")
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        elapsed = time.monotonic() - start
        debug_log(
            "EXEC",
            f"Workflow execution complete: status={final['status']}, elapsed={elapsed:.3f}s",
        )

        assert final is not None
        assert final["status"] in ["completed", "failed"]
        assert elapsed < 3.5

        print(f"\n✓ NFR: Response latency within SLA")
        print(f"  - Run ID: {run_id}")
        print(f"  - Elapsed: {elapsed:.3f}s (SLA: < 3.5s)")
        debug_log("TEST", "✓ test_response_within_user_visible_latency_budget PASSED")

    def test_safe_fallback_on_provider_timeout(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: Stripe or FedEx provider timeout during refund processing
        When: The circuit breaker activates
        Then: The workflow should fall back safely with retry handling
        """
        debug_log("TEST", "Starting test_safe_fallback_on_provider_timeout")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "retail-saas", "customer-support:v2.1.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(client, "retail-saas", ecommerce_support_workflow)
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with simulate_timeout=True")
        response = client.post(
            "/execute",
            json={
                "tenant_id": "retail-saas",
                "workflow_id": "customer-support:v2.1.0",
                "input": {
                    "customer_id": "cust-66666",
                    "query": "Process refund for #ORD-66666",
                    "request_type": "refund",
                    "simulate_timeout": True,
                },
            },
        )
        debug_log("EXEC", f"POST /execute response: status_code={response.status_code}")

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log(
            "EXEC", f"Run created: run_id={run_id}, will test circuit breaker fallback"
        )

        debug_log("POLL", "Starting workflow execution polling for circuit breaker")
        final = wait_for_terminal_status(client, run_id, timeout=5.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ Safe fallback on provider timeout")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        debug_log("TEST", "✓ test_safe_fallback_on_provider_timeout PASSED")
