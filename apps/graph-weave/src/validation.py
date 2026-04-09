"""Request validation helpers for GraphWeave"""

from typing import Any, Optional
import uuid


def validate_resource_id(value: Any, field_name: str, max_len: int = 128) -> str:
    """
    Validate resource identifiers (tenant_id, workflow_id, skill_id).

    Args:
        value: The value to validate
        field_name: Field name for error messages
        max_len: Maximum allowed length (default 128)

    Returns:
        str: Validated and sanitized value

    Raises:
        ValueError: If validation fails
    """
    if value is None:
        raise ValueError(f"{field_name} cannot be None")

    if not isinstance(value, str):
        raise ValueError(f"{field_name} must be string, got {type(value).__name__}")

    stripped = value.strip()

    if not stripped:
        raise ValueError(f"{field_name} cannot be empty or whitespace-only")

    if len(stripped) > max_len:
        raise ValueError(f"{field_name} cannot exceed {max_len} characters")

    return stripped


def validate_optional_uuid(value: Optional[str], field_name: str) -> Optional[str]:
    """
    Validate optional UUID format fields.

    Args:
        value: The value to validate (can be None)
        field_name: Field name for error messages

    Returns:
        str: Valid UUID string, or None if value is None

    Raises:
        ValueError: If value is not valid UUID format
    """
    if value is None:
        return None

    if not isinstance(value, str):
        raise ValueError(
            f"{field_name} must be string or None, got {type(value).__name__}"
        )

    try:
        uuid.UUID(value)
        return value
    except ValueError:
        raise ValueError(f"{field_name} must be valid UUID format, got '{value}'")
