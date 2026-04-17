from pydantic import BaseModel, Field, field_validator
from typing import Optional
from ..validation import validate_resource_id


class InvalidateRequest(BaseModel):
    tenant_id: str = Field(
        default="default",
        description="Tenant identifier",
        json_schema_extra={"example": "default"},
        min_length=1,
        max_length=128,
    )
    skill_id: str = Field(
        ...,
        description="Skill identifier to invalidate cache for (e.g., web_search, sql_query_engine)",
        json_schema_extra={"example": "web_search"},
        min_length=1,
        max_length=128,
    )
    reason: str = Field(
        ...,
        description="Reason for invalidation (e.g., updated_implementation, schema_change, test_refresh)",
        json_schema_extra={"example": "updated_implementation"},
        min_length=1,
        max_length=256,
    )

    @field_validator("tenant_id")
    @classmethod
    def validate_tenant_id(cls, v: str) -> str:
        return validate_resource_id(v, "tenant_id")

    @field_validator("skill_id")
    @classmethod
    def validate_skill_id(cls, v: str) -> str:
        return validate_resource_id(v, "skill_id")

    @field_validator("reason")
    @classmethod
    def validate_reason(cls, v: str) -> str:
        if v is None:
            raise ValueError("reason cannot be None")
        if not isinstance(v, str):
            raise ValueError(f"reason must be string, got {type(v).__name__}")
        stripped = v.strip()
        if not stripped:
            raise ValueError("reason cannot be empty or whitespace-only")
        if len(stripped) > 256:
            raise ValueError("reason cannot exceed 256 characters")
        return stripped


class InvalidateResponse(BaseModel):
    status: str = Field(
        ...,
        description="Invalidation result: invalidated (cache entry removed) or not_found (entry didn't exist)",
        json_schema_extra={"example": "invalidated"},
    )
    tenant_id: str = Field(
        ...,
        description="Tenant identifier echo from request",
        json_schema_extra={"example": "default"},
    )
    skill_id: str = Field(
        ...,
        description="Skill identifier echo from request",
        json_schema_extra={"example": "web_search"},
    )
    reason: str = Field(
        ...,
        description="Reason echo from request",
        json_schema_extra={"example": "updated_implementation"},
    )

    @field_validator("status")
    @classmethod
    def validate_status(cls, v: str) -> str:
        valid_statuses = ["invalidated", "not_found"]
        if v not in valid_statuses:
            raise ValueError(f"status must be one of {valid_statuses}, got '{v}'")
        return v
