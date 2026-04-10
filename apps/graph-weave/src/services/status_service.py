import os

from typing import Any, Dict, Optional, Union

from ..adapters.cache import RedisAdapter
from ..models import StatusEnum
from .event_emitter import EventEmitter


class StatusService:
    def __init__(self, store: Optional[Any] = None):
        self._store = (
            store
            if store is not None
            else RedisAdapter.from_env(
                os.getenv("UPSTASH_REDIS_REST_URL", ""),
                os.getenv("UPSTASH_REDIS_REST_TOKEN", ""),
            )
        )
        self._events = EventEmitter(getattr(self._store, "_store", {}))

    def _status_key(self, tenant_id: str, run_id: str) -> str:
        return f"run:{tenant_id}:{run_id}:status"

    def _run_index_key(self, run_id: str) -> str:
        return f"run_index:{run_id}"

    def set_status(
        self,
        tenant_id: str,
        run_id: str,
        status: Union[StatusEnum, str],
        payload: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        normalized_status = (
            status if isinstance(status, StatusEnum) else StatusEnum(status)
        )
        record = {
            "tenant_id": tenant_id,
            "run_id": run_id,
            "status": normalized_status.value,
            "events": payload.get("events", []) if payload else [],
            "final_state": payload.get("final_state") if payload else None,
            "hop_count": payload.get("hop_count", 0) if payload else 0,
            "workflow_id": payload.get("workflow_id") if payload else None,
        }
        if hasattr(self._store, "set"):
            self._store.set(self._status_key(tenant_id, run_id), record)
            self._store.set(self._run_index_key(run_id), tenant_id)
        else:
            self._store[f"{tenant_id}:{run_id}"] = record
        if payload and payload.get("events"):
            for event in payload["events"]:
                self._events.emit(
                    tenant_id, run_id, event.get("type", "status.changed"), event
                )
        return record

    def get_status(self, tenant_id: str, run_id: str) -> Optional[Dict[str, Any]]:
        if hasattr(self._store, "get"):
            record = self._store.get(self._status_key(tenant_id, run_id))
            if record is not None:
                return record
            indexed_tenant = self._store.get(self._run_index_key(run_id))
            if indexed_tenant is not None:
                return self._store.get(self._status_key(indexed_tenant, run_id))
            return None
        return self._store.get(f"{tenant_id}:{run_id}")

    def transition_status(
        self,
        tenant_id: str,
        run_id: str,
        status: Union[StatusEnum, str],
        payload: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        event_payload = payload or {}
        event_payload.setdefault(
            "events",
            [
                {
                    "type": "status.changed",
                    "status": (
                        status.value if isinstance(status, StatusEnum) else status
                    ),
                }
            ],
        )
        return self.set_status(tenant_id, run_id, status, event_payload)
