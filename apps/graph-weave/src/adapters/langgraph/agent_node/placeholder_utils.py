import re
from typing import Any, Set

def validate_tool_args_resolved(tool_name: str, tool_args: Any) -> None:
    """
    Validates that all placeholders in tool arguments have been resolved.
    """
    unresolved = find_unresolved_placeholders(tool_args)
    if unresolved:
        raise ValueError(
            f"Tool '{tool_name}' arguments contain unresolved template placeholders: "
            + ", ".join(sorted(unresolved))
        )

def find_unresolved_placeholders(value: Any) -> Set[str]:
    """
    Recursively finds unresolved {placeholder} strings in the given value.
    """
    found: Set[str] = set()
    if isinstance(value, str):
        # Match {variable} or {nested.variable}
        for match in re.findall(r"\{[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*\}", value):
            found.add(match)
    elif isinstance(value, dict):
        for item in value.values():
            found.update(find_unresolved_placeholders(item))
    elif isinstance(value, list):
        for item in value:
            found.update(find_unresolved_placeholders(item))
    return found
