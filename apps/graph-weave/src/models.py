from pydantic import BaseModel
from typing import Any, Dict, Optional
import uuid


class ExecuteRequest(BaseModel):
    tenant_id: str
    workflow_id: str
    input: Dict[str, Any]
    run_id: Optional[str] = None


class ExecuteResponse(BaseModel):
    run_id: str
    thread_id: str
    status: str
    workflow_id: str
    tenant_id: str


class InvalidateRequest(BaseModel):
    tenant_id: str
    skill_id: str
    reason: str


class InvalidateResponse(BaseModel):
    status: str
    tenant_id: str
    skill_id: str
    reason: str
