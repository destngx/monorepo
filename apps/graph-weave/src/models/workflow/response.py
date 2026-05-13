from pydantic import BaseModel, Field, field_validator
from typing import Optional, List, Dict, Any

class WorkflowResponse(BaseModel):
    """Response model for workflow operations (without definition)"""

    workflow_id: str = Field(
        ...,
        description="Workflow identifier (e.g., workflow-generator:v1.0.0)",
        json_schema_extra={"example": "workflow-generator:v1.0.0"},
    )
    tenant_id: str = Field(
        ...,
        description="Tenant identifier",
        json_schema_extra={"example": "default"},
    )
    name: str = Field(
        ...,
        description="Human-readable workflow name",
        json_schema_extra={"example": "Workflow Generator"},
    )
    version: str = Field(
        ...,
        description="Semantic version",
        json_schema_extra={"example": "1.0.0"},
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
        json_schema_extra={"example": "default"},
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

class TenantListResponse(BaseModel):
    """Response model for listing all tenants with workflows"""
    tenants: List[str] = Field(
        default_factory=list,
        description="List of all tenant IDs that have workflows registered",
        json_schema_extra={"example": ["default", "system", "research_team"]},
    )
    count: int = Field(
        ...,
        description="Number of tenants in the list",
        json_schema_extra={"example": 3},
    )
