from __future__ import annotations

import logging
import re
import shlex
from typing import Any, Dict, Mapping, Optional, Tuple

from .schema import ExecutorState
from .transforms import TRANSFORMS
from src.adapters.langgraph.utils.json_value_utils import (
    json_stringify_normalized,
    parse_json_string_if_needed,
    structured_arg_string,
)

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

    def resolve(self, path: Any, *, required: bool = True) -> Any:
        if not path:
            return None

        # Standard functional resolvers also accept mapping specs inside resolve()
        if isinstance(path, dict):
            # If it's a function mapping dict
            if path.get("type") == "function":
                return None  # Let handle_function_mapping in state_utils handle it if called as functional
            return self._resolve_mapping_value(path)

        if not isinstance(path, str):
            return None

        # Clean path and extract inline virtual transforms
        clean_path, virtual_transform, original_clean_path = self._parse_virtual_transforms(path)

        spec = self._parse_path(clean_path)
        value = self._resolve_namespace_path(spec)
        if value is None:
            if required:
                raise MissingStatePathError(f"State path not found: {path}")
            return None

        # Apply inline virtual transform if parsed
        if virtual_transform is not None:
            if virtual_transform == "joined" and isinstance(value, list):
                sep_key = original_clean_path.lower()
                sep = ", " if any(k in sep_key for k in ["tag", "author", "name"]) else "\n"
                value = sep.join(str(i) for i in value)
            elif virtual_transform == "first" and isinstance(value, list) and len(value) > 0:
                value = value[0]
            elif virtual_transform == "sh_quote":
                value = shlex.quote(structured_arg_string(value))
            elif virtual_transform == "json_quote":
                value = json_stringify_normalized(value)

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

    def _parse_virtual_transforms(self, path: str) -> Tuple[str, Optional[str], str]:
        # Strip $. prefix or $ prefix if it exists
        clean_path = path.strip()
        if clean_path.startswith("$."):
            clean_path = clean_path[2:]
        elif clean_path.startswith("$"):
            clean_path = clean_path[1:]
            if clean_path.startswith("."):
                clean_path = clean_path[1:]

        original_clean_path = clean_path
        virtual_transform = None

        if ".join(" in clean_path:
            clean_path = clean_path.split(".join(")[0]
            virtual_transform = "joined"
        elif clean_path.endswith("_joined"):
            clean_path = clean_path[:-7]
            virtual_transform = "joined"
        elif ".first(" in clean_path or clean_path.endswith(".first"):
            clean_path = clean_path.split(".first")[0]
            virtual_transform = "first"
        elif clean_path.endswith("_first"):
            clean_path = clean_path[:-6]
            virtual_transform = "first"
        elif ".sh_quote(" in clean_path or clean_path.endswith(".sh_quote"):
            clean_path = clean_path.split(".sh_quote")[0]
            virtual_transform = "sh_quote"
        elif clean_path.endswith("_shell"):
            clean_path = clean_path[:-6]
            virtual_transform = "sh_quote"
        elif ".json_quote(" in clean_path or clean_path.endswith(".json_quote"):
            clean_path = clean_path.split(".json_quote")[0]
            virtual_transform = "json_quote"
        elif ".json_escape(" in clean_path or clean_path.endswith(".json_escape"):
            clean_path = clean_path.split(".json_escape")[0]
            virtual_transform = "json_quote"
        elif clean_path.endswith("_json"):
            clean_path = clean_path[:-5]
            virtual_transform = "json_quote"
        elif ".shell(" in clean_path or clean_path.endswith(".shell"):
            clean_path = clean_path.split(".shell")[0]
            virtual_transform = "sh_quote"

        return clean_path, virtual_transform, original_clean_path

    def _parse_path(self, path: str) -> Tuple[str, list[str]]:
        if not isinstance(path, str) or not path.strip():
            raise InvalidStatePathError("State path must be a non-empty string")

        clean_path = path.strip()
        # Already stripped $ and $. in parse_virtual_transforms, but support passing clean_path directly too
        if clean_path.startswith("$."):
            clean_path = clean_path[2:]
        elif clean_path.startswith("$"):
            clean_path = clean_path[1:]
            if clean_path.startswith("."):
                clean_path = clean_path[1:]

        if not clean_path:
            raise InvalidStatePathError("State path must not be empty")

        # 1. Normalize LLM-generated entry node input schema path references to $.input.*
        if clean_path.startswith("entry.input_schema."):
            clean_path = "input." + clean_path[len("entry.input_schema."):]

        # 2. Normalize LLM-generated $<node_id>.output_schema.<field> patterns
        # Identify if path starts with <node_id>.output_schema. where node_id matches typical node ID patterns
        # e.g., "normalize_input.output_schema.file_content" -> "nodes.normalize_input.result.file_content"
        output_schema_match = re.match(r"^([A-Za-z0-9_-]+)\.output_schema\.(.+)$", clean_path)
        if output_schema_match:
            node_id, field = output_schema_match.groups()
            clean_path = f"nodes.{node_id}.result.{field}"
        else:
            # 3. Normalize LLM-generated $<node_id>.input_schema.<field> patterns
            input_schema_match = re.match(r"^([A-Za-z0-9_-]+)\.input_schema\.(.+)$", clean_path)
            if input_schema_match:
                node_id, field = input_schema_match.groups()
                clean_path = f"nodes.{node_id}.result.{field}"

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

        # Check in workflow_state if present (common in unit tests/legacy formats)
        if "workflow_state" in self.state and isinstance(self.state["workflow_state"], dict):
            if namespace in self.state["workflow_state"]:
                return self.state["workflow_state"][namespace]

        # If namespace references entry node specifically, fallback to nodes.entry
        if namespace == "entry" and "nodes" in self.state and isinstance(self.state["nodes"], dict):
            if "entry" in self.state["nodes"]:
                return self.state["nodes"]["entry"]

        # Legacy compatibility for direct root lookups during migration.
        if namespace in {"input", "workflow", "nodes", "node_results", "runtime", "errors"}:
            return self.state.get(namespace)

        # Unified Resolver Fallback: Search in state["nodes"] directly.
        # This matches the highly lenient get_state_value lookups and makes bare node_id paths work 100%.
        if "nodes" in self.state and isinstance(self.state["nodes"], dict):
            if namespace in self.state["nodes"]:
                return self.state["nodes"][namespace]

        # Fallback for Mock/Test environments that use "node_results" instead of "nodes"
        if "node_results" in self.state and isinstance(self.state["node_results"], dict):
            if namespace in self.state["node_results"]:
                return self.state["node_results"][namespace]

        # Check in workflow state as secondary backup
        if "workflow" in self.state and isinstance(self.state["workflow"], dict):
            if namespace in self.state["workflow"]:
                return self.state["workflow"][namespace]

        return None

    def _resolve_path(self, current: Any, keys: list[str]) -> Any:
        for key in keys:
            if current is None:
                return None

            # Virtual suffixes handling for first part if we have keys
            virtual_transform = None
            first_part = key
            if first_part.endswith("_joined"):
                virtual_transform = "joined"
                first_part = first_part[:-7]
            elif first_part.endswith("_first"):
                virtual_transform = "first"
                first_part = first_part[:-6]
            elif first_part.endswith("_shell"):
                virtual_transform = "sh_quote"
                first_part = first_part[:-6]
            elif first_part.endswith("_json"):
                virtual_transform = "json_quote"
                first_part = first_part[:-5]

            bracket_index_match = re.match(r"\[(\d+)\]", first_part)
            if bracket_index_match:
                index = int(bracket_index_match.group(1))
                if isinstance(current, list) and index < len(current):
                    current = current[index]
                    continue
                return None

            array_match = re.match(r"([^\[]+)\[(\d+)\]", first_part)
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
                if first_part in current:
                    current = current[first_part]
                elif f"{first_part}_json" in current:
                    current = parse_json_string_if_needed(current[f"{first_part}_json"])
                elif first_part.endswith("_json") and first_part[:-5] in current:
                    current = json_stringify_normalized(current[first_part[:-5]])
                else:
                    return None
            elif isinstance(current, list) and first_part.isdigit():
                idx = int(first_part)
                if idx < len(current):
                    current = current[idx]
                else:
                    return None
            else:
                return None

            # Apply virtual transforms at nested paths
            if virtual_transform is not None:
                if virtual_transform == "joined" and isinstance(current, list):
                    current = ", ".join(str(i) for i in current)
                elif virtual_transform == "first" and isinstance(current, list) and len(current) > 0:
                    current = current[0]
                elif virtual_transform == "sh_quote":
                    current = shlex.quote(structured_arg_string(current))
                elif virtual_transform == "json_quote":
                    current = json_stringify_normalized(current)

        return current

    def _apply_transform(self, value: Any, transform: str, spec: Dict[str, Any]) -> Any:
        transform_fn = TRANSFORMS.get(transform)
        if not transform_fn:
            raise InvalidStatePathError(f"Unknown transform: {transform}")

        if transform == "join":
            return transform_fn(value, separator=spec.get("separator", ", "))
        return transform_fn(value)

