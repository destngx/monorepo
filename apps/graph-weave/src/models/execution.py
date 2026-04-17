from pydantic import BaseModel, field_validator, Field
from enum import Enum
from typing import Dict, Any, Optional
from ..validation import validate_resource_id, validate_optional_uuid


class StatusEnum(str, Enum):
    queued = "queued"
    validating = "validating"
    pending = "pending"
    running = "running"
    completed = "completed"
    failed = "failed"
    cancelled = "cancelled"


class ExecuteRequest(BaseModel):
    tenant_id: str = Field(
        default="default",
        description="Tenant identifier (e.g., default)",
        json_schema_extra={"example": "default"},
        min_length=1,
        max_length=128,
    )
    workflow_id: str = Field(
        ...,
        description="Workflow identifier with optional version (e.g., workflow-generator:v1.0.0)",
        json_schema_extra={"example": "workflow-generator:v1.0.0"},
        min_length=1,
        max_length=128,
    )
    input: Dict[str, Any] = Field(
        ...,
        description="Workflow input data. Example: intent, domain, or research objectives",
        json_schema_extra={
            "example": {
                "intent": "When an EKS pod enters CrashLoopBackOff state, fetch logs and alert Slack.",
                "domain": "devops",
                "correction_attempts": 0,
            }
        },
    )

    @field_validator("tenant_id")
    @classmethod
    def validate_tenant_id(cls, v: str) -> str:
        return validate_resource_id(v, "tenant_id")

    @field_validator("workflow_id")
    @classmethod
    def validate_workflow_id(cls, v: str) -> str:
        return validate_resource_id(v, "workflow_id")


class ExecuteResponse(BaseModel):
    run_id: str = Field(
        ...,
        description="Unique run identifier (UUID) - stable across reruns for the same workflow instance",
        json_schema_extra={"example": "1e3f5aa7-32be-4e2a-bdc6-88b3fd461ee5"},
    )
    thread_id: str = Field(
        ...,
        description="Thread identifier (UUID) - changes on each execution within a run",
        json_schema_extra={"example": "a7e2c9f1-4b6d-11ee-be56-0242ac120002"},
    )
    status: StatusEnum = Field(
        ...,
        description="Execution status: queued, validating, pending, running, completed, failed, or cancelled",
        json_schema_extra={"example": "queued"},
    )
    workflow_id: str = Field(
        ...,
        description="Workflow identifier echo from request",
        json_schema_extra={"example": "workflow-generator:v1.0.0"},
    )
    tenant_id: str = Field(
        ...,
        description="Tenant identifier echo from request",
        json_schema_extra={"example": "default"},
    )

    @field_validator("run_id", mode="before")
    @classmethod
    def validate_run_id(cls, v: str) -> str:
        return validate_optional_uuid(v, "run_id") or v

    @field_validator("thread_id", mode="before")
    @classmethod
    def validate_thread_id(cls, v: str) -> str:
        return validate_optional_uuid(v, "thread_id") or v

    @field_validator("status")
    @classmethod
    def validate_status(cls, v: str) -> str:
        valid_statuses = [s.value for s in StatusEnum]
        if v not in valid_statuses:
            raise ValueError(f"status must be one of {valid_statuses}, got '{v}'")
        return v


class RecoveryRequest(BaseModel):
    thread_id: str = Field(..., min_length=1, max_length=128)


class CancelResponse(BaseModel):
    run_id: str
    status: StatusEnum
    thread_id: str
