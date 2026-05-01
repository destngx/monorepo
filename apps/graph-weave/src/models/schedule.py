from pydantic import BaseModel, Field, field_validator
from typing import Optional, List, Dict, Any
import re
from datetime import datetime

def validate_cron(v: str) -> str:
    """Very basic cron validation. Detailed validation done by APScheduler/croniter."""
    if not v:
        raise ValueError("cron_expression cannot be empty")
    # Basic check for 5 or 6 fields
    parts = v.split()
    if len(parts) not in [5, 6]:
        raise ValueError("cron_expression must have 5 or 6 parts")
    return v

class ScheduleCreate(BaseModel):
    """Request model for creating a new schedule"""
    tenant_id: str = Field(..., description="Tenant identifier")
    workflow_id: str = Field(..., description="Workflow ID to execute")
    cron_expression: str = Field(..., description="Cron expression (e.g., '0 0 * * *')")
    input_data: Optional[Dict[str, Any]] = Field(default_factory=dict, description="Input data for the workflow")
    name: Optional[str] = Field(None, description="Human-readable name for the schedule")
    enabled: bool = Field(default=True, description="Whether the schedule is active")

    @field_validator("cron_expression")
    @classmethod
    def validate_cron_expr(cls, v: str) -> str:
        return validate_cron(v)

class ScheduleUpdate(BaseModel):
    """Request model for updating an existing schedule"""
    cron_expression: Optional[str] = Field(None, description="Cron expression")
    input_data: Optional[Dict[str, Any]] = Field(None, description="Input data for the workflow")
    name: Optional[str] = Field(None, description="Human-readable name")
    enabled: Optional[bool] = Field(None, description="Whether the schedule is active")

    @field_validator("cron_expression")
    @classmethod
    def validate_cron_expr_if_present(cls, v: Optional[str]) -> Optional[str]:
        if v is not None:
            return validate_cron(v)
        return v

class ScheduleResponse(BaseModel):
    """Response model for schedule details"""
    schedule_id: str
    tenant_id: str
    workflow_id: str
    cron_expression: str
    input_data: Dict[str, Any]
    name: Optional[str]
    enabled: bool
    created_at: str
    updated_at: str

class ScheduleSummary(BaseModel):
    """Lightweight schedule summary for list operations"""
    schedule_id: str
    workflow_id: str
    cron_expression: str
    name: Optional[str]
    enabled: bool

class ScheduleListResponse(BaseModel):
    """Response model for listing schedules"""
    tenant_id: str
    count: int
    schedules: List[ScheduleSummary]
