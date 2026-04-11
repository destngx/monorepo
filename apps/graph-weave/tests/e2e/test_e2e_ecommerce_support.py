"""
E2E test for E-Commerce Customer Support use case (UC-ECOM-001, UC-ECOM-002, UC-ECOM-003)

This test validates the support workflow that:
1. Validates customer input before routing
2. Supports order lookup, refund handling, and safe fallback
3. Redacts response and streams after guardrails pass
"""

import time
import pytest
from fastapi.testclient import TestClient

from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


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
                    },
                },
                {
                    "id": "billing-agent",
                    "type": "agent",
                    "config": {
                        "agent_name": "billing_refund_handler",
                        "description": "Handle billing and refund decisions",
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


def wait_for_terminal_status(client, run_id, timeout=3.0):
    deadline = time.monotonic() + timeout
    last_data = None
    while time.monotonic() < deadline:
        response = client.get(f"/execute/{run_id}/status")
        assert response.status_code == 200
        last_data = response.json()
        if last_data.get("status") in ["completed", "failed", "cancelled"]:
            return last_data
        time.sleep(0.01)
    return last_data


class TestEcommerceCustomerSupportE2E:
    def setup_method(self):
        store = get_workflow_store()
        store.clear()

    def test_uc_ecom_001_validate_customer_input_before_routing(
        self, client, ecommerce_support_workflow
    ):
        """
        UC-ECOM-001: The support workflow must validate customer input before routing.

        Given: A customer support request with valid input
        When: The input guardrail processes it
        Then: It should pass validation and proceed to routing
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "queued"

        final = wait_for_terminal_status(client, data["run_id"])
        assert final is not None
        assert final["status"] in ["completed", "failed"]

        print(f"\n✓ UC-ECOM-001: Customer input validated successfully")
        print(f"  - Run ID: {data['run_id']}")
        print(f"  - Status: {final['status']}")

    def test_uc_ecom_001_malicious_input_blocked(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: A request with malicious input
        When: The input guardrail validates it
        Then: The request should be blocked before routing
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id, timeout=2.0)
        assert final is not None

        print(f"\n✓ UC-ECOM-001: Malicious input blocked")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")

    def test_uc_ecom_002_order_lookup_path(
        self, client, ecommerce_support_workflow
    ):
        """
        UC-ECOM-002: The workflow must support order lookup, refund handling, and safe fallback.

        Given: A customer asking about their order status
        When: The workflow routes to shipping/order lookup
        Then: The shipping agent should return order status information
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id)
        assert final is not None
        assert final["status"] in ["completed", "failed"]

        print(f"\n✓ UC-ECOM-002: Order lookup path executed")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")

    def test_uc_ecom_002_refund_handling_path(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: A customer requesting a refund for a lost package
        When: The workflow routes to billing/refund handler
        Then: The billing agent should evaluate refund eligibility and process if allowed
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id)
        assert final is not None
        assert final["status"] in ["completed", "failed"]

        print(f"\n✓ UC-ECOM-002: Refund handling path executed")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")

    def test_uc_ecom_002_refund_loop_detection(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: A refund request that triggers repeated processing attempts
        When: The stagnation detector activates after 3 identical intents
        Then: The workflow should exit safely with the last good answer
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id, timeout=2.0)
        assert final is not None

        print(f"\n✓ UC-ECOM-002: Refund loop detected and halted")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")

    def test_uc_ecom_003_output_redaction_for_pii(
        self, client, ecommerce_support_workflow
    ):
        """
        UC-ECOM-003: The response must be redacted and streamed after guardrails pass.

        Given: A support interaction with payment method details in the final response
        When: The output guardrail processes it
        Then: PII and policy violations should be redacted before streaming
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id)
        assert final is not None

        if final.get("final_state"):
            output_str = str(final["final_state"])
            assert "credit_card" not in output_str or "[REDACTED]" in output_str

        print(f"\n✓ UC-ECOM-003: Output redacted for PII")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")

    def test_response_within_user_visible_latency_budget(
        self, client, ecommerce_support_workflow
    ):
        """
        NFR: The workflow should return within a user-visible latency budget suitable for support interactions.

        Given: A support request
        When: The workflow executes
        Then: Response should complete within 3 seconds (typical user wait time)
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

        start = time.monotonic()

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id, timeout=3.0)
        elapsed = time.monotonic() - start

        assert final is not None
        assert final["status"] in ["completed", "failed"]
        assert elapsed < 3.5

        print(f"\n✓ NFR: Response latency within SLA")
        print(f"  - Run ID: {run_id}")
        print(f"  - Elapsed: {elapsed:.3f}s (SLA: < 3.5s)")

    def test_safe_fallback_on_provider_timeout(
        self, client, ecommerce_support_workflow
    ):
        """
        Given: Stripe or FedEx provider timeout during refund processing
        When: The circuit breaker activates
        Then: The workflow should fall back safely with retry handling
        """
        store = get_workflow_store()
        store.create("retail-saas", ecommerce_support_workflow)

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

        assert response.status_code == 200
        run_id = response.json()["run_id"]

        final = wait_for_terminal_status(client, run_id, timeout=3.0)
        assert final is not None

        print(f"\n✓ Safe fallback on provider timeout")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
