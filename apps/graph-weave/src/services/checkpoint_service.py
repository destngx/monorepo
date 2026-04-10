from typing import Any, Dict, Optional


class CheckpointService:
    def __init__(self, store: Any):
        self._store = store

    def save_checkpoint(
        self, tenant_id: str, thread_id: str, workflow_state: Dict[str, Any]
    ) -> Dict[str, Any]:
        checkpoint = {
            "tenant_id": tenant_id,
            "thread_id": thread_id,
            "workflow_state": workflow_state,
        }
        self._store.set(f"checkpoint:{tenant_id}:{thread_id}:latest", checkpoint)
        return checkpoint

    def load_checkpoint(
        self, tenant_id: str, thread_id: str
    ) -> Optional[Dict[str, Any]]:
        return self._store.get(f"checkpoint:{tenant_id}:{thread_id}:latest")

    def clear_checkpoint(self, tenant_id: str, thread_id: str) -> None:
        self._store.delete(f"checkpoint:{tenant_id}:{thread_id}:latest")
