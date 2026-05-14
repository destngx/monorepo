import json
from typing import Any


def parse_json_string_if_needed(value: Any) -> Any:
    if not isinstance(value, str):
        return value

    stripped = value.strip()
    if not stripped or stripped[0] not in "[{":
        return value

    try:
        return json.loads(stripped)
    except json.JSONDecodeError:
        return value


def json_stringify_normalized(value: Any) -> str:
    parsed = parse_json_string_if_needed(value)
    return json.dumps(parsed)


def structured_arg_string(value: Any) -> str:
    parsed = parse_json_string_if_needed(value)
    if isinstance(parsed, (dict, list)):
        return json.dumps(parsed)
    return str(value)
