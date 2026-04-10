from pydantic import BaseModel, field_validator, Field, model_validator
from enum import Enum
from typing import Any, Dict, Optional, List
from datetime import datetime
import re
from .validation import validate_resource_id, validate_optional_uuid


class ExecuteRequest(BaseModel):
    tenant_id: str = Field(
        ...,
        description="Tenant identifier (e.g., hedge_fund_research_desk)",
        json_schema_extra={"example": "hedge_fund_research_desk"},
        min_length=1,
        max_length=128,
    )
    workflow_id: str = Field(
        ...,
        description="Workflow identifier with optional version (e.g., quant-research:v3.0.0)",
        json_schema_extra={"example": "quant-research:v3.0.0"},
        min_length=1,
        max_length=128,
    )
    input: Dict[str, Any] = Field(
        ...,
        description="Workflow input data. Example: search query, SQL parameters, or research objectives",
        json_schema_extra={
            "example": {
                "query": "Q3 earnings and performance metrics",
                "stagnation_threshold": 3,
                "data_sources": ["web_search", "sql_warehouse"],
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


class StatusEnum(str, Enum):
    queued = "queued"
    validating = "validating"
    pending = "pending"
    running = "running"
    completed = "completed"
    failed = "failed"
    cancelled = "cancelled"


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
        json_schema_extra={"example": "quant-research:v3.0.0"},
    )
    tenant_id: str = Field(
        ...,
        description="Tenant identifier echo from request",
        json_schema_extra={"example": "hedge_fund_research_desk"},
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


class InvalidateRequest(BaseModel):
    tenant_id: str = Field(
        ...,
        description="Tenant identifier",
        json_schema_extra={"example": "hedge_fund_research_desk"},
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
        json_schema_extra={"example": "hedge_fund_research_desk"},
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


# Workflow Management Models


def validate_workflow_id_format(v: str) -> str:
    """Validate workflow_id format: name:vX.Y.Z"""
    if not re.match(r"^[a-z][a-z0-9_-]*:v\d+\.\d+\.\d+$", v):
        raise ValueError(
            f"workflow_id must be in format 'name:vX.Y.Z' (e.g., quant-research:v3.0.0), got '{v}'"
        )
    return v


def validate_semantic_version(v: str) -> str:
    """Validate semantic versioning X.Y.Z"""
    if not re.match(r"^\d+\.\d+\.\d+$", v):
        raise ValueError(
            f"version must be semantic versioning X.Y.Z (e.g., 3.0.0), got '{v}'"
        )
    return v


class WorkflowCreate(BaseModel):
    """Request model for creating a new workflow"""

    tenant_id: str = Field(
        ...,
        description="Tenant identifier (e.g., hedge_fund_research_desk)",
        json_schema_extra={"example": "hedge_fund_research_desk"},
        min_length=1,
        max_length=128,
    )
    workflow_id: str = Field(
        ...,
        description="Workflow identifier in format 'name:vX.Y.Z' (e.g., quant-research:v3.0.0)",
        json_schema_extra={"example": "quant-research:v3.0.0"},
        min_length=1,
        max_length=128,
    )
    name: str = Field(
        ...,
        description="Human-readable workflow name",
        json_schema_extra={"example": "Quantitative Research Pipeline"},
        min_length=1,
        max_length=256,
    )
    version: str = Field(
        ...,
        description="Semantic versioning X.Y.Z (must match version in workflow_id)",
        json_schema_extra={"example": "3.0.0"},
    )
    description: Optional[str] = Field(
        None,
        description="Workflow description (markdown supported)",
        json_schema_extra={
            "example": "Comprehensive quantitative research workflow for equities analysis"
        },
        max_length=1000,
    )
    tags: Optional[List[str]] = Field(
        None,
        description="Searchable tags",
        json_schema_extra={"example": ["research", "equities", "earnings"]},
    )
    owner: str = Field(
        ...,
        description="Workflow owner/creator",
        json_schema_extra={"example": "research_team"},
        min_length=1,
        max_length=128,
    )
    definition: Dict[str, Any] = Field(
        ...,
        description="Complete workflow JSON definition per WORKFLOW_JSON_SPEC",
    )

    @field_validator("tenant_id")
    @classmethod
    def validate_tenant_id(cls, v: str) -> str:
        return validate_resource_id(v, "tenant_id")

    @field_validator("workflow_id")
    @classmethod
    def validate_workflow_id(cls, v: str) -> str:
        return validate_workflow_id_format(v)

    @field_validator("version")
    @classmethod
    def validate_version(cls, v: str) -> str:
        return validate_semantic_version(v)

    @field_validator("name")
    @classmethod
    def validate_name(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("name cannot be empty or whitespace-only")
        return v.strip()

    @field_validator("owner")
    @classmethod
    def validate_owner(cls, v: str) -> str:
        return validate_resource_id(v, "owner")

    @field_validator("definition")
    @classmethod
    def validate_definition(cls, v: Dict[str, Any]) -> Dict[str, Any]:
        if not isinstance(v, dict):
            raise ValueError("definition must be a valid JSON object")
        if not v:
            raise ValueError("definition cannot be empty")
        required_fields = {"nodes", "edges", "entry_point", "exit_point"}
        missing_fields = required_fields - set(v.keys())
        if missing_fields:
            raise ValueError(
                f"workflow definition missing required fields: {', '.join(sorted(missing_fields))}"
            )
        return v

    @model_validator(mode="after")
    def validate_workflow_id_matches_version(self):
        workflow_id = self.workflow_id
        version = self.version
        id_version = workflow_id.split(":")[-1][1:]
        if id_version != version:
            raise ValueError(
                f"workflow_id version (v{id_version}) must match version field ({version})"
            )
        return self


class WorkflowResponse(BaseModel):
    """Response model for workflow operations (without definition)"""

    workflow_id: str = Field(
        ...,
        description="Workflow identifier (e.g., quant-research:v3.0.0)",
        json_schema_extra={"example": "quant-research:v3.0.0"},
    )
    tenant_id: str = Field(
        ...,
        description="Tenant identifier",
        json_schema_extra={"example": "hedge_fund_research_desk"},
    )
    name: str = Field(
        ...,
        description="Human-readable workflow name",
        json_schema_extra={"example": "Quantitative Research Pipeline"},
    )
    version: str = Field(
        ...,
        description="Semantic version",
        json_schema_extra={"example": "3.0.0"},
    )
    description: Optional[str] = Field(
        None,
        description="Workflow description",
        json_schema_extra={
            "example": "Comprehensive quantitative research workflow for equities analysis"
        },
    )
    tags: Optional[List[str]] = Field(
        None,
        description="Searchable tags",
        json_schema_extra={"example": ["research", "equities", "earnings"]},
    )
    owner: str = Field(
        ...,
        description="Workflow owner",
        json_schema_extra={"example": "research_team"},
    )
    status: str = Field(
        ...,
        description="Workflow status: active, archived, or draft",
        json_schema_extra={"example": "active"},
    )
    created_at: str = Field(
        ...,
        description="ISO 8601 creation timestamp",
        json_schema_extra={"example": "2026-04-09T14:30:00Z"},
    )
    updated_at: str = Field(
        ...,
        description="ISO 8601 last update timestamp",
        json_schema_extra={"example": "2026-04-09T14:30:00Z"},
    )

    @field_validator("status")
    @classmethod
    def validate_status(cls, v: str) -> str:
        valid_statuses = ["active", "archived", "draft"]
        if v not in valid_statuses:
            raise ValueError(f"status must be one of {valid_statuses}, got '{v}'")
        return v


class WorkflowDetailResponse(WorkflowResponse):
    """Response model that includes the full workflow definition"""

    definition: Dict[str, Any] = Field(
        ...,
        description="Complete workflow JSON definition",
    )


class WorkflowSummary(BaseModel):
    """Lightweight workflow summary for list operations"""

    workflow_id: str
    name: str
    version: str
    description: Optional[str] = None
    tags: Optional[List[str]] = None
    owner: str
    status: str
    created_at: str
    updated_at: str


class WorkflowListResponse(BaseModel):
    """Response model for listing workflows"""

    tenant_id: str = Field(
        ...,
        description="Tenant identifier",
        json_schema_extra={"example": "hedge_fund_research_desk"},
    )
    workflows: List[WorkflowSummary] = Field(
        default_factory=list,
        description="List of workflow summaries (without definitions)",
    )
    count: int = Field(
        ...,
        description="Number of workflows in the list",
        json_schema_extra={"example": 1},
    )


class UpdateWorkflowRequest(BaseModel):
    """Request model for updating a workflow (all fields optional)"""

    name: Optional[str] = Field(
        None,
        description="Updated workflow name",
        min_length=1,
        max_length=256,
    )
    description: Optional[str] = Field(
        None,
        description="Updated workflow description",
        max_length=1000,
    )
    tags: Optional[List[str]] = Field(
        None,
        description="Updated tags",
    )
    owner: Optional[str] = Field(
        None,
        description="Updated owner",
        min_length=1,
        max_length=128,
    )
    status: Optional[str] = Field(
        None,
        description="Updated status (active, archived, or draft)",
    )
    definition: Optional[Dict[str, Any]] = Field(
        None,
        description="Updated workflow definition",
    )

    @field_validator("name", mode="before")
    @classmethod
    def validate_name_if_present(cls, v: Optional[str]) -> Optional[str]:
        if v is not None:
            if not v.strip():
                raise ValueError("name cannot be empty or whitespace-only")
            return v.strip()
        return v

    @field_validator("owner", mode="before")
    @classmethod
    def validate_owner_if_present(cls, v: Optional[str]) -> Optional[str]:
        if v is not None:
            return validate_resource_id(v, "owner")
        return v

    @field_validator("status", mode="before")
    @classmethod
    def validate_status_if_present(cls, v: Optional[str]) -> Optional[str]:
        if v is not None:
            valid_statuses = ["active", "archived", "draft"]
            if v not in valid_statuses:
                raise ValueError(f"status must be one of {valid_statuses}, got '{v}'")
            return v
        return v

    @field_validator("definition", mode="before")
    @classmethod
    def validate_definition_if_present(
        cls, v: Optional[Dict[str, Any]]
    ) -> Optional[Dict[str, Any]]:
        if v is not None:
            if not isinstance(v, dict):
                raise ValueError("definition must be a valid JSON object")
            if not v:
                raise ValueError("definition cannot be empty")
            required_fields = {"nodes", "edges", "entry_point", "exit_point"}
            missing_fields = required_fields - set(v.keys())
            if missing_fields:
                raise ValueError(
                    f"workflow definition missing required fields: {', '.join(sorted(missing_fields))}"
                )
        return v

    @field_validator("*", mode="before")
    @classmethod
    def at_least_one_field(cls, v, info):
        """Ensure at least one field is provided"""
        if info.context and info.context.get("mode") == "validation":
            return v
        return v

    def has_updates(self) -> bool:
        """Check if any fields are being updated"""
        return any(
            [
                self.name is not None,
                self.description is not None,
                self.tags is not None,
                self.owner is not None,
                self.status is not None,
                self.definition is not None,
            ]
        )
