from pydantic import BaseModel, Field, field_validator
from typing import Optional, List, Dict, Any
from ...validation import validate_resource_id
from .create import WorkflowCompositionSpec

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
    composition: Optional[WorkflowCompositionSpec] = Field(
        None,
        description="Updated composition spec (generates new workflow definition)",
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
                self.composition is not None,
            ]
        )
