from typing import Any, Dict, List


class ThreadLifecycleService:
    def __init__(self, store: Any):
        self._store = store

    def add_active_thread(
        self, tenant_id: str, thread_id: str, workflow_id: str
    ) -> Dict[str, Any]:
        record = {
            "tenant_id": tenant_id,
            "thread_id": thread_id,
            "workflow_id": workflow_id,
        }
        self._store.set(f"active_threads:{tenant_id}:{thread_id}", record)
        return record

    def remove_active_thread(self, tenant_id: str, thread_id: str) -> None:
        self._store.delete(f"active_threads:{tenant_id}:{thread_id}")

    def is_active(self, tenant_id: str, thread_id: str) -> bool:
        return self._store.exists(f"active_threads:{tenant_id}:{thread_id}")

    def list_active_threads(self, tenant_id: str) -> List[Dict[str, Any]]:
        prefix = f"active_threads:{tenant_id}:"
        return [
            value
            for key, value in getattr(self._store, "_store", {}).items()
            if key.startswith(prefix)
        ]

    def set_kill_flag(self, tenant_id: str, thread_id: str) -> None:
        self._store.set(f"circuit_breaker:{tenant_id}:{thread_id}:kill", True)
