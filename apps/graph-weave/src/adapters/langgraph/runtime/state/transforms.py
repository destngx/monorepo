import json
import shlex
from typing import Any, Iterable, List

from src.adapters.langgraph.utils.json_value_utils import (
    json_stringify_normalized,
    structured_arg_string,
)


def shell_quote(value: Any) -> str:
    return shlex.quote(structured_arg_string(value))


def json_stringify(value: Any) -> str:
    return json_stringify_normalized(value)


def join(value: Any, separator: str = ", ") -> str:
    if isinstance(value, list):
        return separator.join(str(item) for item in value)
    if value is None:
        return ""
    return str(value)


def first(value: Any) -> Any:
    if isinstance(value, list) and value:
        return value[0]
    return value


TRANSFORMS = {
    "shell_quote": shell_quote,
    "json_stringify": json_stringify,
    "join": join,
    "first": first,
}
