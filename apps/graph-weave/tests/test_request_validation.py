"""
Request Input Validation Tests (TDD RED Phase)

Purpose: Verify all ExecuteRequest and InvalidateRequest fields are validated
with correct field-level rules before business logic execution.

Given: Invalid request payloads
When: Requesting POST /execute or POST /invalidate endpoints
Then: Expect HTTP 422 with validation error details
"""

import pytest
from fastapi.testclient import TestClient
from pydantic import ValidationError
from src.main import app
from src.models import ExecuteRequest, InvalidateRequest


client = TestClient(app)


class TestExecuteRequestValidation:
    """ExecuteRequest field validation tests"""

    def test_execute_request_empty_tenant_id_validation(self):
        """
        Given: ExecuteRequest with empty tenant_id (financial research use case)
        When: Creating model instance with empty tenant_id
        Then: Raise ValidationError with tenant_id field error
        """
        with pytest.raises(ValidationError) as exc_info:
            ExecuteRequest(
                tenant_id="",
                workflow_id="quant-research:v3.0.0",
                input={},
            )
        assert "tenant_id" in str(exc_info.value)

    def test_execute_request_empty_workflow_id_validation(self):
        """
        Given: ExecuteRequest with empty workflow_id from financial research desk
        When: Creating model instance with empty workflow_id
        Then: Raise ValidationError with workflow_id field error
        """
        with pytest.raises(ValidationError) as exc_info:
            ExecuteRequest(
                tenant_id="hedge_fund_research_desk",
                workflow_id="",
                input={},
            )
        assert "workflow_id" in str(exc_info.value)


class TestInvalidateRequestValidation:
    """InvalidateRequest field validation tests"""

    def test_invalidate_request_empty_skill_id_validation(self):
        """
        Given: InvalidateRequest with empty skill_id for web_search invalidation
        When: Creating model instance with empty skill_id
        Then: Raise ValidationError with skill_id field error
        """
        with pytest.raises(ValidationError) as exc_info:
            InvalidateRequest(
                tenant_id="hedge_fund_research_desk",
                skill_id="",
                reason="updated_implementation",
            )
        assert "skill_id" in str(exc_info.value)

    def test_invalidate_request_empty_reason_validation(self):
        """
        Given: InvalidateRequest with empty reason for skill invalidation
        When: Creating model instance with empty reason
        Then: Raise ValidationError with reason field error
        """
        with pytest.raises(ValidationError) as exc_info:
            InvalidateRequest(
                tenant_id="hedge_fund_research_desk",
                skill_id="web_search",
                reason="",
            )
        assert "reason" in str(exc_info.value)

    def test_invalidate_request_reason_max_length_validation(self):
        """
        Given: InvalidateRequest with reason exceeding 256 characters
        When: Creating model instance with oversized reason from quant-research workflow
        Then: Raise ValidationError with reason field error
        """
        with pytest.raises(ValidationError) as exc_info:
            InvalidateRequest(
                tenant_id="hedge_fund_research_desk",
                skill_id="sql_query_engine",
                reason="x" * 257,  # Exceeds 256 char limit
            )
        assert "reason" in str(exc_info.value)


class TestEndpointValidationErrors:
    """HTTP 422 response validation for endpoints"""

    def test_execute_endpoint_returns_422_on_invalid_request(self):
        """
        Given: POST /execute with empty tenant_id from financial research workflow
        When: Making request with invalid data
        Then: Expect HTTP 422 Unprocessable Entity
        """
        response = client.post(
            "/execute",
            json={
                "tenant_id": "",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "query": "Q3 earnings and performance metrics",
                    "stagnation_threshold": 3,
                },
            },
        )
        assert response.status_code == 422
        assert "detail" in response.json() or "error" in response.json()

    def test_invalidate_endpoint_returns_422_on_invalid_request(self):
        """
        Given: POST /invalidate with empty skill_id from hedge fund research desk
        When: Making request with invalid data
        Then: Expect HTTP 422 Unprocessable Entity
        """
        response = client.post(
            "/invalidate",
            json={
                "tenant_id": "hedge_fund_research_desk",
                "skill_id": "",
                "reason": "updated_implementation",
            },
        )
        assert response.status_code == 422
        assert "detail" in response.json() or "error" in response.json()
