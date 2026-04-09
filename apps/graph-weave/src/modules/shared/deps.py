"""
Shared dependencies for modules.
"""

from typing import Optional
from src.adapters.cache import MockRedisAdapter
from src.adapters.workflow import MockWorkflowStore
from src.adapters.checkpoint import MockCheckpointStore


class Services:
    def __init__(self):
        self.cache = MockRedisAdapter()
        self.workflow_store = MockWorkflowStore()
        self.checkpoint_store = MockCheckpointStore()


_services: Optional[Services] = None


def init_services() -> None:
    global _services
    _services = Services()


def get_services() -> Services:
    if _services is None:
        init_services()
    return _services


def get_cache() -> MockRedisAdapter:
    return get_services().cache


def get_workflow_store() -> MockWorkflowStore:
    return get_services().workflow_store


def get_checkpoint_store() -> MockCheckpointStore:
    return get_services().checkpoint_store
