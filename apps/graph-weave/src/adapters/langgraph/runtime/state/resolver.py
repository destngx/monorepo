from __future__ import annotations

import logging
import re
from typing import Any, Dict, Mapping, Optional, Tuple

from .schema import ExecutorState
from .transforms import TRANSFORMS

logger = logging.getLogger(__name__)


class StateResolutionError(ValueError):
    pass


class MissingStatePathError(StateResolutionError):
    pass


class InvalidStatePathError(StateResolutionError):
    pass


class StateResolver:
    def __init__(self, state: Mapping[str, Any]):
        self.state = state

    def resolve(self, path: str, *, required: bool = True) -> Any:
        spec = self._parse_path(path)
        value = self._resolve_namespace_path(spec)
        if value is None:
            if required:
                raise MissingStatePathError(f"State path not found: {path}")
            return None
        return value

    def resolve_mapping(self, mapping: Mapping[str, Any]) -> Dict[str, Any]:
        resolved: Dict[str, Any] = {}
        for key, spec in mapping.items():
            resolved[key] = self._resolve_mapping_value(spec)
        return resolved

    def _resolve_mapping_value(self, spec: Any) -> Any:
        if isinstance(spec, str):
            return self.resolve(spec)

        if not isinstance(spec, dict):
            raise InvalidStatePathError(
                f"Mapping spec must be a string or object, got {type(spec).__name__}"
            )

        path = spec.get("path")
        if not path:
            if spec.get("required", True):
                raise MissingStatePathError("Mapping spec missing required 'path'")
            return spec.get("default")

        required = spec.get("required", True)
        value = self.resolve(path, required=required)
        if value is None:
            return spec.get("default")

        transform = spec.get("transform")
        if transform:
            value = self._apply_transform(value, transform, spec)
        return value

    def _parse_path(self, path: str) -> Tuple[str, list[str]]:
        if not isinstance(path, str) or not path.strip():
            raise InvalidStatePathError("State path must be a non-empty string")

        clean_path = path.strip()
        if clean_path.startswith("$."):
            clean_path = clean_path[2:]
        elif clean_path.startswith("$"):
            clean_path = clean_path[1:]
            if clean_path.startswith("."):
                clean_path = clean_path[1:]
        else:
            raise InvalidStatePathError(
                f"State path must start with '$.', got {path!r}"
            )

        if not clean_path:
            raise InvalidStatePathError("State path must not be empty")

        parts = clean_path.split(".")
        namespace = parts[0]
        return namespace, parts[1:]

    def _resolve_namespace_path(self, spec: Tuple[str, list[str]]) -> Any:
        namespace, keys = spec
        current: Any = self._namespace_root(namespace)
        if current is None:
            return None

        return self._resolve_path(current, keys)

    def _namespace_root(self, namespace: str) -> Any:
        if namespace in self.state:
            return self.state[namespace]

        # Legacy compatibility for direct root lookups during migration.
        if namespace in {"input", "workflow", "nodes", "runtime", "errors"}:
            return self.state.get(namespace)

        if namespace in self.state:
            return self.state[namespace]

        return None

    def _resolve_path(self, current: Any, keys: list[str]) -> Any:
        for key in keys:
            if current is None:
                return None

            bracket_index_match = re.match(r"\[(\d+)\]", key)
            if bracket_index_match:
                index = int(bracket_index_match.group(1))
                if isinstance(current, list) and index < len(current):
                    current = current[index]
                    continue
                return None

            array_match = re.match(r"([^\[]+)\[(\d+)\]", key)
            if array_match:
                name, index = array_match.groups()
                index = int(index)
                if isinstance(current, dict) and name in current:
                    current = current[name]
                    if isinstance(current, list) and index < len(current):
                        current = current[index]
                        continue
                return None

            if isinstance(current, dict):
                if key in current:
                    current = current[key]
                    continue
                return None

            if isinstance(current, list) and key.isdigit():
                idx = int(key)
                if idx < len(current):
                    current = current[idx]
                    continue
                return None

            return None
        return current

    def _apply_transform(self, value: Any, transform: str, spec: Dict[str, Any]) -> Any:
        transform_fn = TRANSFORMS.get(transform)
        if not transform_fn:
            raise InvalidStatePathError(f"Unknown transform: {transform}")

        if transform == "join":
            return transform_fn(value, separator=spec.get("separator", ", "))
        return transform_fn(value)
