from typing import Any, Dict, List, Optional


class EventEmitter:
    def __init__(self, store: Optional[Dict[str, List[Dict[str, Any]]]] = None):
        self._store = store if store is not None else {}

    def emit(
        self, tenant_id: str, run_id: str, event_type: str, payload: Dict[str, Any]
    ) -> Dict[str, Any]:
        event = {
            "type": event_type,
            "tenant_id": tenant_id,
            "run_id": run_id,
            **payload,
        }
        key = f"run:{tenant_id}:{run_id}:events"
        self._store.setdefault(key, []).append(event)
        return event

    def get_events(self, tenant_id: str, run_id: str) -> List[Dict[str, Any]]:
        return self._store.get(f"run:{tenant_id}:{run_id}:events", [])
