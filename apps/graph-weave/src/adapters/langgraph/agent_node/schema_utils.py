from typing import Any, Dict, List, Optional

def coerce_to_output_schema(data: Any, schema: Optional[Dict[str, Any]]) -> Any:
    """
    Tries to fit the data into the given JSON schema, including unwrapping envelopes.
    """
    if not schema:
        return data

    candidates = [data]
    if isinstance(data, dict):
        for envelope_key in ("result", "output", "data"):
            value = data.get(envelope_key)
            if isinstance(value, dict):
                candidates.append(value)

    # Special case: single required field schema
    required = schema.get("required", [])
    properties = schema.get("properties", {})
    if len(required) == 1:
        required_key = required[0]
        if not isinstance(data, dict) or required_key not in data:
            field_schema = properties.get(required_key, {})
            if not isinstance(field_schema, dict) or matches_json_type(data, field_schema.get("type")):
                candidates.append({required_key: data})

    first_error = None
    for candidate in candidates:
        try:
            validate_output_schema(candidate, schema)
            return candidate
        except ValueError as exc:
            if first_error is None:
                first_error = exc

    if first_error:
        raise first_error
    raise ValueError("Agent output does not match schema")

def validate_output_schema(data: Any, schema: Optional[Dict[str, Any]]) -> None:
    """
    Validates that the data matches the provided JSON schema.
    """
    if not schema:
        return

    if is_legacy_result_confidence_schema(schema) and isinstance(data, dict) and "result" not in data:
        return

    expected_type = schema.get("type")
    if expected_type and not matches_json_type(data, expected_type):
        raise ValueError(f"Agent output does not match schema type '{expected_type}'")

    if isinstance(data, dict):
        for key in schema.get("required", []):
            if key not in data:
                raise ValueError(f"Agent output missing required schema field '{key}'")

        properties = schema.get("properties", {})
        for key, field_schema in properties.items():
            if key not in data or not isinstance(field_schema, dict):
                continue
            validate_value_schema(data[key], field_schema, key)

def validate_value_schema(value: Any, schema: Dict[str, Any], path: str) -> None:
    expected_type = schema.get("type")
    if expected_type and not matches_json_type(value, expected_type):
        raise ValueError(f"Agent output field '{path}' does not match schema type '{expected_type}'")

    if isinstance(value, dict):
        for key in schema.get("required", []):
            if key not in value:
                raise ValueError(f"Agent output field '{path}' missing required schema field '{key}'")
        properties = schema.get("properties", {})
        for key, field_schema in properties.items():
            if key not in value or not isinstance(field_schema, dict):
                continue
            validate_value_schema(value[key], field_schema, f"{path}.{key}")

    if isinstance(value, list) and isinstance(schema.get("items"), dict):
        item_schema = schema["items"]
        for index, item in enumerate(value):
            validate_value_schema(item, item_schema, f"{path}[{index}]")

def matches_json_type(value: Any, expected_type: Any) -> bool:
    """
    Checks if a Python value matches a JSON schema type.
    """
    if isinstance(expected_type, list):
        return any(matches_json_type(value, item) for item in expected_type)

    type_map = {
        "object": dict,
        "array": list,
        "string": str,
        "boolean": bool,
        "integer": int,
        "number": (int, float),
    }
    python_type = type_map.get(expected_type)
    if python_type is None:
        return True
    if expected_type in {"integer", "number"} and isinstance(value, bool):
        return False
    return isinstance(value, python_type)

def is_legacy_result_confidence_schema(schema: Dict[str, Any]) -> bool:
    """
    Detects if the schema is the legacy {result: object, confidence: number} format.
    """
    required = set(schema.get("required", []))
    properties = schema.get("properties", {})
    result_schema = properties.get("result", {})
    confidence_schema = properties.get("confidence", {})

    return (
        schema.get("type") == "object"
        and required.issubset({"result", "confidence"})
        and "result" in properties
        and "confidence" in properties
        and result_schema.get("type") == "object"
        and confidence_schema.get("type") == "number"
    )
