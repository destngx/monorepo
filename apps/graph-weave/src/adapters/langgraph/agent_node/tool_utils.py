import re
from typing import Any, Dict, List, Optional

def format_tool_error(inferred_result: Dict[str, Any]) -> str:
    """
    Formats a human-readable error message from inferred tool results.
    """
    tool_results = inferred_result.get("tool_results", [])
    latest = tool_results[-1] if tool_results else {}
    parts = [
        "Tool execution failed",
        f"tool={latest.get('tool')}" if latest.get("tool") else "",
        f"command={latest.get('command')}" if latest.get("command") else "",
        f"exit_code={latest.get('exit_code')}" if latest.get("exit_code") is not None else "",
        f"error={latest.get('error')}" if latest.get("error") else "",
        f"stderr={latest.get('stderr')}" if latest.get("stderr") else "",
        f"stdout={latest.get('stdout')}" if latest.get("stdout") else "",
    ]
    return "; ".join(part for part in parts if part)

def infer_result_from_tools(
    tool_results: List[Dict[str, Any]],
    final_content: str,
    tool_output_mapping: Optional[Dict[str, Any]] = None,
) -> Optional[Dict[str, Any]]:
    """
    Infers a structured result from the output of executed tools.
    """
    if not tool_results:
        return None

    successful = [result for result in tool_results if result.get("status") == "success"]
    latest = successful[-1] if successful else tool_results[-1]
    stdout = str(latest.get("stdout") or "")
    stderr = str(latest.get("stderr") or "")
    parsed_stdout = parse_labeled_stdout(stdout)

    output: Dict[str, Any] = {
        "status": "success" if successful else "error",
        "tool_results": tool_results,
        "tool_stdout": stdout,
        "tool_stderr": stderr,
        "stdout_fields": parsed_stdout,
        "raw_response": final_content or stdout,
    }

    for key, spec in (tool_output_mapping or {}).items():
        output[key] = resolve_tool_output_spec(spec, output)

    return output

def parse_labeled_stdout(text: str) -> Dict[str, Any]:
    """
    Parses "Key: Value" lines from stdout.
    """
    fields: Dict[str, Any] = {}
    for line in text.splitlines():
        stripped = line.strip()
        match = re.match(r"^([A-Za-z][A-Za-z0-9 _-]*):\s*(.*)$", stripped)
        if not match:
            continue
        key = re.sub(r"[^a-z0-9]+", "_", match.group(1).lower()).strip("_")
        value = match.group(2).strip()
        if not key or value == "":
            continue
        existing = fields.get(key)
        if existing is None:
            fields[key] = value
        elif isinstance(existing, list):
            existing.append(value)
        else:
            fields[key] = [existing, value]
    return fields

def resolve_tool_output_spec(spec: Any, output: Dict[str, Any]) -> Any:
    """
    Resolves a tool output mapping specification against the collected output.
    """
    if isinstance(spec, str):
        if spec.startswith("$."):
            return resolve_dict_path(output, spec[2:].split("."))
        if spec in output:
            return output[spec]
        return output.get("stdout_fields", {}).get(spec)

    if not isinstance(spec, dict):
        return spec

    transform = spec.get("type")
    if transform == "constant":
        return spec.get("value")
    if transform == "array":
        value = resolve_tool_output_spec(spec.get("value"), output)
        if value is None:
            return []
        return value if isinstance(value, list) else [value]

    return None

def resolve_dict_path(current: Any, keys: List[str]) -> Any:
    """
    Resolves a nested path (list of keys) against a dictionary/list.
    """
    for key in keys:
        if isinstance(current, dict) and key in current:
            current = current[key]
        elif isinstance(current, list) and key.isdigit():
            index = int(key)
            current = current[index] if index < len(current) else None
        else:
            return None
    return current
