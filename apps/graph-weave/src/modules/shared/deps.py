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
