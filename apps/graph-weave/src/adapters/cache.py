from typing import Optional, Dict, Any


class MockRedisAdapter:
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

    def clear(self) -> None:
        self._store.clear()

    def _build_versioned_key(
        self, namespace: str, tenant_id: str, skill_id: str, version: str
    ) -> str:
        return f"{namespace}:{tenant_id}:{skill_id}:{version}"

    def set_versioned(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
        value: Any,
    ) -> None:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        self.set(key, value)

    def get_versioned(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
    ) -> Optional[Any]:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        return self.get(key)

    def get_versioned_with_fallback(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: Optional[str],
    ) -> Optional[Any]:
        if version is not None:
            return self.get_versioned(namespace, tenant_id, skill_id, version)
        return self.get_versioned(namespace, tenant_id, skill_id, "latest")

    def get_versioned_with_rebuild(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
        source_loader: Any,
    ) -> Any:
        cached = self.get_versioned(namespace, tenant_id, skill_id, version)
        if cached is not None:
            return cached

        entry = source_loader.load_from_source(tenant_id, skill_id, version)
        self.set_versioned(namespace, tenant_id, skill_id, version, entry)
        return entry
