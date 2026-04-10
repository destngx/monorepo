from typing import Any, Dict, Optional


class MockThreadLifecycleStore:
    def __init__(self):
        self._store: Dict[str, Any] = {}

    def set(self, key: str, value: Any) -> None:
        self._store[key] = value

    def get(self, key: str) -> Optional[Any]:
        return self._store.get(key)

    def delete(self, key: str) -> None:
        if key in self._store:
            del self._store[key]

    def exists(self, key: str) -> bool:
        return key in self._store

    def add_active_thread(
        self, tenant_id: str, thread_id: str, workflow_id: str
    ) -> Dict[str, Any]:
        record = {
            "tenant_id": tenant_id,
            "thread_id": thread_id,
            "workflow_id": workflow_id,
        }
        self.set(f"active_threads:{tenant_id}:{thread_id}", record)
        return record

    def remove_active_thread(self, tenant_id: str, thread_id: str) -> None:
        self.delete(f"active_threads:{tenant_id}:{thread_id}")

    def is_active(self, tenant_id: str, thread_id: str) -> bool:
        return self.exists(f"active_threads:{tenant_id}:{thread_id}")
