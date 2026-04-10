"""
Response Schema Validation Tests (TDD RED Phase)

Purpose: Verify all ExecuteResponse and InvalidateResponse schemas are correctly
enforced with proper UUID format, status enums, and required fields before returning
to clients.

Given: Invalid response data
When: Response models attempt to construct with invalid values
Then: Expect ValidationError with field-level details
"""

import pytest
from pydantic import ValidationError
import uuid
from src.models import ExecuteResponse, InvalidateResponse


class TestExecuteResponseValidation:
    """ExecuteResponse field validation tests"""

    def test_execute_response_invalid_run_id_uuid_format(self):
        """
        Given: ExecuteResponse with invalid run_id from quant-research workflow
        When: Creating model instance with run_id="not-a-uuid"
        Then: Raise ValidationError with run_id field error
        """
        with pytest.raises(ValidationError) as exc_info:
            ExecuteResponse(
                run_id="not-a-uuid",
                thread_id=str(uuid.uuid4()),
                status="pending",
                workflow_id="quant-research:v3.0.0",
                tenant_id="hedge_fund_research_desk",
            )
        assert "run_id" in str(exc_info.value)

    def test_execute_response_invalid_thread_id_uuid_format(self):
        """
        Given: ExecuteResponse with invalid thread_id for financial research desk
        When: Creating model instance with thread_id="not-a-uuid"
        Then: Raise ValidationError with thread_id field error
        """
        with pytest.raises(ValidationError) as exc_info:
            ExecuteResponse(
                run_id=str(uuid.uuid4()),
                thread_id="not-a-uuid",
                status="running",
                workflow_id="quant-research:v3.0.0",
                tenant_id="hedge_fund_research_desk",
            )
        assert "thread_id" in str(exc_info.value)

    def test_execute_response_invalid_status_enum(self):
        """
        Given: ExecuteResponse with status not in allowed set
        When: Creating model instance with status="invalid_status" for earnings research
        Then: Raise ValidationError with status field error
        """
        with pytest.raises(ValidationError) as exc_info:
            ExecuteResponse(
                run_id=str(uuid.uuid4()),
                thread_id=str(uuid.uuid4()),
                status="invalid_status",
                workflow_id="quant-research:v3.0.0",
                tenant_id="hedge_fund_research_desk",
            )
        assert "status" in str(exc_info.value)

    def test_execute_response_null_workflow_id(self):
        """
        Given: ExecuteResponse with null workflow_id from research desk
        When: Creating model instance with workflow_id=None
        Then: Raise ValidationError with workflow_id field error
        """
        with pytest.raises(ValidationError) as exc_info:
            ExecuteResponse(
                run_id=str(uuid.uuid4()),
                thread_id=str(uuid.uuid4()),
                status="pending",
                workflow_id=None,
                tenant_id="hedge_fund_research_desk",
            )
        assert "workflow_id" in str(exc_info.value)

    def test_execute_response_valid_all_statuses(self):
        """
        Given: ExecuteResponse with each valid status for quant-research workflow
        When: Creating model instances with status in ["pending", "running", "completed", "failed"]
        Then: All should construct successfully without ValidationError
        """
        valid_statuses = ["pending", "running", "completed", "failed"]
        for status in valid_statuses:
            response = ExecuteResponse(
                run_id=str(uuid.uuid4()),
                thread_id=str(uuid.uuid4()),
                status=status,
                workflow_id="quant-research:v3.0.0",
                tenant_id="hedge_fund_research_desk",
            )
            assert response.status == status


class TestInvalidateResponseValidation:
    """InvalidateResponse field validation tests"""

    def test_invalidate_response_invalid_status_enum(self):
        """
        Given: InvalidateResponse with status not in ["invalidated", "not_found"]
        When: Creating model instance with invalid status for web_search skill
        Then: Raise ValidationError with status field error
        """
        with pytest.raises(ValidationError) as exc_info:
            InvalidateResponse(
                status="invalid",
                tenant_id="hedge_fund_research_desk",
                skill_id="web_search",
                reason="updated_implementation",
            )
        assert "status" in str(exc_info.value)

    def test_invalidate_response_valid_statuses(self):
        """
        Given: InvalidateResponse with valid statuses for skill invalidation
        When: Creating model instances with status in ["invalidated", "not_found"]
        Then: Both should construct successfully without ValidationError
        """
        for status in ["invalidated", "not_found"]:
            response = InvalidateResponse(
                status=status,
                tenant_id="hedge_fund_research_desk",
                skill_id="sql_query_engine",
                reason="schema_change",
            )
            assert response.status == status
