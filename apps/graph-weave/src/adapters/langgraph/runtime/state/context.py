from __future__ import annotations

from typing import Any, Dict, Mapping, Optional

from .resolver import StateResolver


class StateContext:
    def __init__(
        self,
        state: Mapping[str, Any],
        node_input_contract: Optional[Any] = None,
    ):
        self.state = state
        self.contract = node_input_contract
        self.resolver = StateResolver(state)

    def resolve(self, path: str, *, required: bool = True) -> Any:
        return self.resolver.resolve(path, required=required)

    def resolve_mapping(self, mapping: Mapping[str, Any]) -> Dict[str, Any]:
        return self.resolver.resolve_mapping(mapping)
