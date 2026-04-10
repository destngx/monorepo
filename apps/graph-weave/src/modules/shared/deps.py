"""
Shared dependencies for modules.
"""

from typing import Optional
from src.adapters.cache import RedisAdapter
from src.config import GraphWeaveConfig
from src.services.checkpoint_service import CheckpointService
from src.services.thread_lifecycle_service import ThreadLifecycleService
from src.adapters.workflow import MockWorkflowStore
from src.adapters.checkpoint import MockCheckpointStore


class Services:
    def __init__(self):
        if (
            not GraphWeaveConfig.UPSTASH_REDIS_REST_URL
            or not GraphWeaveConfig.UPSTASH_REDIS_REST_TOKEN
        ):
            raise RuntimeError(
                "UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN are required"
            )
        self.cache = RedisAdapter.from_env(
            GraphWeaveConfig.UPSTASH_REDIS_REST_URL,
            GraphWeaveConfig.UPSTASH_REDIS_REST_TOKEN,
        )
        self.workflow_store = MockWorkflowStore()
        self.checkpoint_service = CheckpointService(self.cache)
        self.thread_lifecycle_service = ThreadLifecycleService(self.cache)


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
