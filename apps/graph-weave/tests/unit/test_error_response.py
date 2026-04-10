"""
Error Response Standardization Tests (TDD RED Phase)

Purpose: Verify error responses follow standardized format with consistent
HTTP status codes, error types, and structured error details.

Given: Error conditions across endpoints
When: Errors occur during request handling
Then: Expect standardized error response format with correct HTTP status
"""

import pytest
from fastapi.testclient import TestClient
from src.main import app


client = TestClient(app)


class TestValidationErrorResponse:
    """Validation error response format tests"""

    def test_validation_error_returns_422_status(self):
        """
        Given: POST /execute with empty tenant_id from hedge fund research desk
        When: Making invalid request for earnings research
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

    def test_validation_error_has_standard_format(self):
        """
        Given: Invalid request to endpoint for financial research workflow
        When: Making POST /invalidate with empty skill_id (web_search invalidation)
        Then: Error response includes error, message, details, status_code fields
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
        body = response.json()
        assert "error" in body
        assert body["error"] == "ValidationError"
        assert "message" in body
        assert "details" in body
        assert "status_code" in body
        assert body["status_code"] == 422

    def test_validation_error_details_include_field_info(self):
        """
        Given: Invalid request with multiple field errors for quant-research workflow
        When: Making POST /execute with empty tenant_id and workflow_id
        Then: Details array includes field information
        """
        response = client.post(
            "/execute",
            json={
                "tenant_id": "",
                "workflow_id": "",
                "input": {
                    "query": "Q3 earnings and performance metrics",
                    "stagnation_threshold": 3,
                },
            },
        )
        assert response.status_code == 422
        body = response.json()
        details = body.get("details", [])
        assert isinstance(details, list)
        assert len(details) > 0
        for error in details:
            assert "field" in error or "message" in error


class TestMissingFieldErrorResponse:
    """Missing required field error tests"""

    def test_missing_required_field_returns_422(self):
        """
        Given: POST /execute without required workflow_id from research desk
        When: Missing workflow_id in request for quant-research
        Then: Expect HTTP 422 Unprocessable Entity
        """
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge_fund_research_desk",
                "input": {
                    "query": "Q3 earnings and performance metrics",
                    "stagnation_threshold": 3,
                },
            },
        )
        assert response.status_code == 422

    def test_missing_field_error_format(self):
        """
        Given: POST /invalidate missing required skill_id for web_search invalidation
        When: Making request without skill_id field
        Then: Error response includes field name in details
        """
        response = client.post(
            "/invalidate",
            json={
                "tenant_id": "hedge_fund_research_desk",
                "reason": "updated_implementation",
            },
        )
        assert response.status_code == 422
        body = response.json()
        assert body["error"] == "ValidationError"
        assert body["status_code"] == 422


class TestErrorResponseConsistency:
    """Error response consistency across endpoints"""

    def test_all_validation_errors_use_422_status(self):
        """
        Given: Multiple invalid requests for financial research workflow
        When: Posting to /execute and /invalidate with invalid data
        Then: All return HTTP 422
        """
        response1 = client.post(
            "/execute",
            json={
                "tenant_id": "",
                "workflow_id": "quant-research:v3.0.0",
                "input": {"query": "Q3 earnings"},
            },
        )
        response2 = client.post(
            "/invalidate",
            json={
                "tenant_id": "hedge_fund_research_desk",
                "skill_id": "",
                "reason": "updated_implementation",
            },
        )
        assert response1.status_code == 422
        assert response2.status_code == 422

    def test_all_errors_have_required_fields(self):
        """
        Given: Error responses from different endpoints for research desk
        When: Collecting error responses from /execute and /invalidate
        Then: All have error, message, details, status_code fields
        """
        response1 = client.post(
            "/execute",
            json={
                "tenant_id": "",
                "workflow_id": "quant-research:v3.0.0",
                "input": {"query": "Q3 earnings"},
            },
        )
        response2 = client.post(
            "/invalidate",
            json={
                "tenant_id": "hedge_fund_research_desk",
                "skill_id": "",
                "reason": "updated_implementation",
            },
        )
        for response in [response1, response2]:
            body = response.json()
            assert "error" in body
            assert "message" in body
            assert "details" in body
            assert "status_code" in body

    def test_invalid_json_returns_422(self):
        """
        Given: POST request with malformed JSON for earnings research
        When: Sending invalid JSON to /execute
        Then: Expect HTTP 422 Unprocessable Entity
        """
        response = client.post(
            "/execute",
            content="{invalid json}",
            headers={"Content-Type": "application/json"},
        )
        assert response.status_code == 422
