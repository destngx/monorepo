import logging
import uuid
from typing import Any, Dict, Callable, Optional
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from ..adapters.schedule import RedisScheduleStore

logger = logging.getLogger(__name__)

class SchedulerService:
    def __init__(self, schedule_store: RedisScheduleStore, execution_handler: Callable):
        self.scheduler = BackgroundScheduler()
        self.schedule_store = schedule_store
        self.execution_handler = execution_handler

    def start(self):
        if not self.scheduler.running:
            self.scheduler.start()
            logger.info("Scheduler service started")

    def shutdown(self):
        if self.scheduler.running:
            self.scheduler.shutdown()
            logger.info("Scheduler service shut down")

    def add_schedule(self, tenant_id: str, schedule_id: str, cron_expression: str, workflow_id: str, input_data: Dict[str, Any]):
        try:
            self.scheduler.add_job(
                func=self.execution_handler,
                trigger=CronTrigger.from_crontab(cron_expression),
                id=f"{tenant_id}:{schedule_id}",
                args=[tenant_id, workflow_id, input_data],
                replace_existing=True
            )
            logger.info(f"Added job for schedule {schedule_id} (tenant {tenant_id}): {cron_expression}")
        except Exception as e:
            logger.error(f"Failed to add job for schedule {schedule_id}: {e}")
            raise

    def remove_schedule(self, tenant_id: str, schedule_id: str):
        job_id = f"{tenant_id}:{schedule_id}"
        if self.scheduler.get_job(job_id):
            self.scheduler.remove_job(job_id)
            logger.info(f"Removed job for schedule {schedule_id} (tenant {tenant_id})")

    def sync_schedules(self, tenant_id: str):
        """Sync all enabled schedules for a tenant from the store to the scheduler."""
        schedules = self.schedule_store.list_for_tenant(tenant_id)
        for s in schedules:
            if s.get("enabled", True):
                self.add_schedule(
                    tenant_id=tenant_id,
                    schedule_id=s["schedule_id"],
                    cron_expression=s["cron_expression"],
                    workflow_id=s["workflow_id"],
                    input_data=s.get("input_data", {})
                )
            else:
                self.remove_schedule(tenant_id, s["schedule_id"])
