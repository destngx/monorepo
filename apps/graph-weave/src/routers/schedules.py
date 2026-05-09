from fastapi import APIRouter, HTTPException
from typing import Any
from ..app_logging import get_logger
from ..config import GraphWeaveConfig
from ..modules.shared.deps import get_schedule_store, get_scheduler_service
from ..models import (
    ScheduleCreate,
    ScheduleUpdate,
    ScheduleResponse,
    ScheduleSummary,
    ScheduleListResponse,
)

logger = get_logger(__name__)
router = APIRouter()

@router.post(
    "/schedules",
    response_model=ScheduleResponse,
    status_code=201,
    tags=["Schedules"],
)
async def create_schedule(request: ScheduleCreate):
    store = get_schedule_store()
    schedule_data = {
        "workflow_id": request.workflow_id,
        "cron_expression": request.cron_expression,
        "input_data": request.input_data or {},
        "name": request.name,
        "enabled": request.enabled,
    }

    result = store.create(request.tenant_id, schedule_data)

    if request.enabled and GraphWeaveConfig.SCHEDULER_ENABLED:
        scheduler = get_scheduler_service()
        scheduler.add_schedule(
            tenant_id=request.tenant_id,
            schedule_id=result["schedule_id"],
            cron_expression=result["cron_expression"],
            workflow_id=result["workflow_id"],
            input_data=result["input_data"]
        )

    return ScheduleResponse(**result)


@router.get("/schedules", response_model=ScheduleListResponse, tags=["Schedules"])
async def list_schedules(tenant_id: str):
    store = get_schedule_store()
    schedules = store.list_for_tenant(tenant_id)
    return ScheduleListResponse(
        tenant_id=tenant_id,
        count=len(schedules),
        schedules=[ScheduleSummary(**s) for s in schedules],
    )


@router.get("/schedules/{schedule_id}", response_model=ScheduleResponse, tags=["Schedules"])
async def get_schedule(schedule_id: str, tenant_id: str):
    store = get_schedule_store()
    result = store.get(tenant_id, schedule_id)
    if not result:
        raise HTTPException(status_code=404, detail="Schedule not found")
    return ScheduleResponse(**result)


@router.put("/schedules/{schedule_id}", response_model=ScheduleResponse, tags=["Schedules"])
async def update_schedule(schedule_id: str, tenant_id: str, request: ScheduleUpdate):
    store = get_schedule_store()

    updates: dict[str, Any] = {}
    if request.cron_expression is not None:
        updates["cron_expression"] = request.cron_expression
    if request.input_data is not None:
        updates["input_data"] = request.input_data
    if request.name is not None:
        updates["name"] = request.name
    if request.enabled is not None:
        updates["enabled"] = request.enabled

    result = store.update(tenant_id, schedule_id, updates)
    if not result:
        raise HTTPException(status_code=404, detail="Schedule not found")

    if GraphWeaveConfig.SCHEDULER_ENABLED:
        scheduler = get_scheduler_service()
        if result["enabled"]:
            scheduler.add_schedule(
                tenant_id=tenant_id,
                schedule_id=schedule_id,
                cron_expression=result["cron_expression"],
                workflow_id=result["workflow_id"],
                input_data=result["input_data"]
            )
        else:
            scheduler.remove_schedule(tenant_id, schedule_id)

    return ScheduleResponse(**result)


@router.delete("/schedules/{schedule_id}", status_code=204, tags=["Schedules"])
async def delete_schedule(schedule_id: str, tenant_id: str):
    store = get_schedule_store()
    if not store.delete(tenant_id, schedule_id):
        raise HTTPException(status_code=404, detail="Schedule not found")

    if GraphWeaveConfig.SCHEDULER_ENABLED:
        scheduler = get_scheduler_service()
        scheduler.remove_schedule(tenant_id, schedule_id)
