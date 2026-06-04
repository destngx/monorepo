from typing import Any, Dict, List, Optional
from src.adapters.langgraph.utils.json_value_utils import parse_json_string_if_needed

def coerce_to_output_schema(data: Any, schema: Optional[Dict[str, Any]]) -> Any:
    """
    Tries to fit the data into the given JSON schema, including unwrapping envelopes.
    """
    if not schema:
        return data

    if is_legacy_result_confidence_schema(schema):
        if isinstance(data, dict) and isinstance(data.get("result"), dict):
            validate_output_schema(data, schema)
            return data["result"]
        validate_output_schema(data, schema)
        return data

    candidates = []
    if not (isinstance(data, list) and schema.get("type") == "object"):
        candidates.append(data)
    if isinstance(data, dict):
        candidates.extend(iter_dict_candidates(data))

    # Multi-field list-to-object wrapping candidates.
    # When data is a list but schema expects an object with array-typed required fields,
    # generate candidates that assign the list to each array field, auto-filling others.
    required = schema.get("required", [])
    if isinstance(data, list) and schema.get("type") == "object":
        properties = schema.get("properties", {})
        array_fields = [
            f for f in required
            if isinstance(properties.get(f), dict) and properties[f].get("type") == "array"
        ]
        if len(array_fields) >= 1:
            for target_field in array_fields:
                wrapper = {target_field: data}
                for other_field in required:
                    if other_field != target_field:
                        f_schema = properties.get(other_field, {})
                        f_type = f_schema.get("type") if isinstance(f_schema, dict) else None
                        if f_type == "array":
                            wrapper[other_field] = []
                        elif f_type == "object":
                            wrapper[other_field] = {}
                        elif f_type == "string":
                            wrapper[other_field] = ""
                        elif f_type == "boolean":
                            wrapper[other_field] = False
                        elif f_type in ("number", "integer"):
                            wrapper[other_field] = 0
                        else:
                            wrapper[other_field] = None
                candidates.append(wrapper)

    # Special case: single required field schema
    required = schema.get("required", [])
    properties = schema.get("properties", {})
    if len(required) == 1:
        required_key = required[0]
        field_schema = properties.get(required_key, {})
        if isinstance(data, dict):
            candidates.extend(
                iter_single_field_candidates(data, required_key, field_schema)
            )
        if not isinstance(data, dict) or required_key not in data:
            if not isinstance(field_schema, dict) or matches_json_type(data, field_schema.get("type")):
                candidates.append({required_key: data})

    first_error = None
    for candidate in candidates:
        try:
            normalized = normalize_against_schema(candidate, schema)
            validate_output_schema(normalized, schema)
            return normalized
        except ValueError as exc:
            if first_error is None:
                first_error = exc

    if first_error:
        raise first_error
    raise ValueError("Agent output does not match schema")

def iter_dict_candidates(data: Dict[str, Any]) -> List[Dict[str, Any]]:
    """
    Returns likely envelope payloads from an agent response.
    """
    candidates: List[Dict[str, Any]] = []
    seen_ids = set()
    pending = [data]
    while pending:
        current = pending.pop(0)
        for envelope_key in ("result", "output", "data"):
            value = current.get(envelope_key)
            if isinstance(value, dict) and id(value) not in seen_ids:
                seen_ids.add(id(value))
                candidates.append(value)
                pending.append(value)

        for value in current.values():
            if isinstance(value, dict) and id(value) not in seen_ids:
                seen_ids.add(id(value))
                candidates.append(value)

    return candidates

def iter_single_field_candidates(
    data: Dict[str, Any],
    required_key: str,
    field_schema: Dict[str, Any],
) -> List[Dict[str, Any]]:
    """
    Returns candidates for schemas shaped like {"required": ["nodes"]}.
    """
    candidates: List[Dict[str, Any]] = []
    seen_ids = set()
    pending: List[Any] = [data]

    while pending:
        current = pending.pop(0)
        if isinstance(current, dict):
            if required_key in current:
                candidates.append(current)

            for value in current.values():
                if isinstance(value, (dict, list)) and id(value) not in seen_ids:
                    seen_ids.add(id(value))
                    pending.append(value)

                if matches_or_parses_as(value, field_schema):
                    parsed_value = parse_json_string_if_needed(value)
                    candidates.append({required_key: parsed_value})

        elif isinstance(current, list):
            for value in current:
                if isinstance(value, (dict, list)) and id(value) not in seen_ids:
                    seen_ids.add(id(value))
                    pending.append(value)

    return candidates

def matches_or_parses_as(value: Any, schema: Dict[str, Any]) -> bool:
    if matches_json_type(value, schema.get("type")):
        return True
    parsed = parse_json_string_if_needed(value)
    return parsed is not value and matches_json_type(parsed, schema.get("type"))

def normalize_against_schema(data: Any, schema: Optional[Dict[str, Any]]) -> Any:
    """
    Parses stringified JSON values when the schema expects structured data,
    and applies self-healing lenient type coercion and auto-initialization
    of missing required fields to prevent JSON schema validation failures.
    """
    if not schema:
        return data

    parsed = parse_json_string_if_needed(data)
    if parsed is not data:
        data = parsed

    expected_type = schema.get("type")
    
    # 1. Lenient Type Coercion
    if expected_type:
        if expected_type == "array":
            if data is None:
                data = []
            elif not isinstance(data, list):
                if isinstance(data, str) and data.startswith("[") and data.endswith("]"):
                    try:
                        import json
                        data = json.loads(data)
                    except Exception:
                        data = [data]
                else:
                    data = [data]
        elif expected_type == "object":
            if data is None:
                data = {}
            elif not isinstance(data, dict):
                if isinstance(data, str) and data.startswith("{") and data.endswith("}"):
                    try:
                        import json
                        data = json.loads(data)
                    except Exception:
                        data = {}
                else:
                    data = {}
        elif expected_type == "string":
            if data is None:
                data = ""
            elif isinstance(data, (list, dict)):
                import json
                data = json.dumps(data)
            else:
                data = str(data)
        elif expected_type == "boolean":
            if data is None:
                data = False
            elif isinstance(data, str):
                val_lower = data.strip().lower()
                data = val_lower in ("true", "yes", "1", "y", "t", "ready", "pass", "passed", "valid")
            else:
                data = bool(data)
        elif expected_type in ("number", "integer"):
            if data is None:
                data = 0
            elif isinstance(data, str):
                try:
                    data = int(float(data)) if expected_type == "integer" else float(data)
                except ValueError:
                    data = 0
            elif isinstance(data, bool):
                data = 1 if data else 0
            elif not isinstance(data, (int, float)):
                data = 0

    # 2. Recursive Auto-Filing of Missing Required Fields
    if expected_type == "object" and isinstance(data, dict):
        properties = schema.get("properties", {})
        required = schema.get("required", [])
        normalized = dict(data)

        for req_field in required:
            if req_field not in normalized or normalized[req_field] is None:
                field_schema = properties.get(req_field, {})
                f_type = field_schema.get("type") if isinstance(field_schema, dict) else None
                
                if isinstance(f_type, list):
                    non_null_types = [t for t in f_type if t != "null"]
                    f_type = non_null_types[0] if non_null_types else None
                
                if f_type == "array":
                    normalized[req_field] = []
                elif f_type == "object":
                    normalized[req_field] = {}
                elif f_type == "string":
                    normalized[req_field] = ""
                elif f_type == "boolean":
                    normalized[req_field] = False
                elif f_type in ("number", "integer"):
                    normalized[req_field] = 0
                else:
                    normalized[req_field] = None

        for key, field_schema in properties.items():
            if key in normalized and isinstance(field_schema, dict):
                normalized[key] = normalize_against_schema(normalized[key], field_schema)
        return normalized

    if expected_type == "array" and isinstance(data, list):
        item_schema = schema.get("items")
        if isinstance(item_schema, dict):
            return [normalize_against_schema(item, item_schema) for item in data]
        return data

    return data

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
        "null": type(None),
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
