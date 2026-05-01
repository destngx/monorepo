"""
Shared dependencies for modules.
"""

from typing import Optional
from src.adapters.cache import RedisAdapter, MockRedisAdapter
from src.config import GraphWeaveConfig
from src.services.checkpoint_service import CheckpointService
from src.services.thread_lifecycle_service import ThreadLifecycleService
from src.adapters.workflow import MockWorkflowStore
from src.adapters.checkpoint import MockCheckpointStore
from src.adapters.redis_circuit_breaker import NamespacedRedisClient, FallbackStorage
from src.adapters.langgraph_executor import RealLangGraphExecutor
from src.adapters.mcp_router import MCPRouter
from src.adapters.schedule import RedisScheduleStore
from src.services.scheduler_service import SchedulerService


class Services:
    def __init__(self):
        if (
            not GraphWeaveConfig.UPSTASH_REDIS_REST_URL
            or not GraphWeaveConfig.UPSTASH_REDIS_REST_TOKEN
        ):
            # Default to MockRedisAdapter for local dev/testing if env vars are missing
            # This prevents RuntimeError during module import (test collection)
            self.cache = MockRedisAdapter()
        else:
            self.cache = RedisAdapter.from_env(
                GraphWeaveConfig.UPSTASH_REDIS_REST_URL,
                GraphWeaveConfig.UPSTASH_REDIS_REST_TOKEN,
            )
        self.redis_client = NamespacedRedisClient(
            redis_client=self.cache,
            fallback_storage=FallbackStorage()
        )
        self.workflow_store = MockWorkflowStore()
        self.checkpoint_service = CheckpointService(self.redis_client)
        self.thread_lifecycle_service = ThreadLifecycleService(self.redis_client)
        
        self.mcp_router = MCPRouter()
        self.executor = RealLangGraphExecutor(
            mcp_router=self.mcp_router,
            redis_client=self.redis_client
        )
        self.schedule_store = RedisScheduleStore(self.redis_client)
        
        # Execution handler for the scheduler
        def scheduler_execution_handler(tenant_id, workflow_id, input_data):
            # This will be called by APScheduler
            import uuid
            run_id = str(uuid.uuid4())
            thread_id = str(uuid.uuid4())
            workflow = self.workflow_store.get(tenant_id, workflow_id)
            if workflow:
                from src.services.status_service import StatusService
                # We need a status service instance. In main.py it's global.
                # Here we might need a way to access it or just trigger the background execute.
                # For now, let's just log. Real implementation should trigger background execute.
                pass

        self.scheduler_service = SchedulerService(
            self.schedule_store, 
            execution_handler=None # Will be set in main.py to avoid circular dependency or use a better way
        )


_services: Optional[Services] = None


def init_services() -> None:
    global _services
    _services = Services()


def get_services() -> Services:
    if _services is None:
        init_services()
    return _services


def get_cache() -> RedisAdapter:
    return get_services().cache


def get_workflow_store() -> MockWorkflowStore:
    return get_services().workflow_store


def get_checkpoint_store() -> MockCheckpointStore:
    return MockCheckpointStore()


def get_checkpoint_service() -> CheckpointService:
    return get_services().checkpoint_service


def get_thread_lifecycle_service() -> ThreadLifecycleService:
    return get_services().thread_lifecycle_service


def get_redis_client() -> NamespacedRedisClient:
    return get_services().redis_client

def get_executor() -> RealLangGraphExecutor:
    return get_services().executor

def get_mcp_router() -> MCPRouter:
    return get_services().mcp_router

def get_schedule_store() -> RedisScheduleStore:
    return get_services().schedule_store

def get_scheduler_service() -> SchedulerService:
    return get_services().scheduler_service
