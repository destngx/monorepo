from pydantic import BaseModel, Field, field_validator, model_validator
from typing import Optional, List, Dict, Any
from ...validation import validate_resource_id
from .validators import validate_workflow_id_format, validate_semantic_version

class WorkflowCreate(BaseModel):
    """Request model for creating a new workflow"""

    tenant_id: str = Field(
        default="default",
        description="Tenant identifier (e.g., default)",
        json_schema_extra={"example": "default"},
        min_length=1,
        max_length=128,
    )
    workflow_id: str = Field(
        ...,
        description="Workflow identifier in format 'name:vX.Y.Z' (e.g., workflow-generator:v1.0.0)",
        json_schema_extra={"example": "workflow-generator:v1.0.0"},
        min_length=1,
        max_length=128,
    )
    name: str = Field(
        ...,
        description="Human-readable workflow name",
        json_schema_extra={"example": "Workflow Generator"},
        min_length=1,
        max_length=256,
    )
    version: str = Field(
        ...,
        description="Semantic versioning X.Y.Z (must match version in workflow_id)",
        json_schema_extra={"example": "1.0.0"},
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
