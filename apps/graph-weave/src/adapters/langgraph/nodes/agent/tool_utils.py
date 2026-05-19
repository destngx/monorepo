import re
from typing import Any, Dict, List, Optional

def format_tool_error(inferred_result: Dict[str, Any]) -> str:
    """
    Formats a human-readable error message from inferred tool results.
    """
    tool_results = inferred_result.get("tool_results", [])
    latest = tool_results[-1] if tool_results else {}
    missing_mapped_fields = inferred_result.get("missing_mapped_fields") or []
    parts = [
        "Tool execution failed",
        f"tool={latest.get('tool')}" if latest.get("tool") else "",
        f"command={latest.get('command')}" if latest.get("command") else "",
        f"exit_code={latest.get('exit_code')}" if latest.get("exit_code") is not None else "",
        f"missing_mapped_fields={','.join(missing_mapped_fields)}" if missing_mapped_fields else "",
        f"error={latest.get('error')}" if latest.get("error") else "",
        f"stderr={latest.get('stderr')}" if latest.get("stderr") else "",
        f"stdout={latest.get('stdout')}" if latest.get("stdout") else "",
    ]
    return "; ".join(part for part in parts if part)

def validate_command_contract(command: str, contract: Optional[Dict[str, Any]], node_id: str = "") -> None:
    """
    Validates a shell command against a generic workflow-defined contract.
    """
    if not contract:
        return

    command = str(command or "")
    if not command:
        return

    forbidden_commands = contract.get("forbidden_commands") or []
    for forbidden_command in forbidden_commands:
        pattern = rf"(^|[;&|]\s*){re.escape(str(forbidden_command))}\b"
        if re.search(pattern, command.strip()):
            raise ValueError(
                f"Node '{node_id}' command uses forbidden command '{forbidden_command}'. "
                f"Got command: {command}"
            )

    forbidden_patterns = contract.get("forbidden_command_patterns") or []
    for pattern in forbidden_patterns:
        if re.search(pattern, command):
            raise ValueError(
                f"Node '{node_id}' command matches forbidden pattern '{pattern}'. "
                f"Got command: {command}"
            )

    required_substrings = contract.get("required_command_substrings") or []
    missing = [value for value in required_substrings if value not in command]
    if missing:
        raise ValueError(
            f"Node '{node_id}' command is missing required substrings: {', '.join(missing)}. "
            f"Got command: {command}"
        )

    forbidden_substrings = contract.get("forbidden_command_substrings") or []
    present = [value for value in forbidden_substrings if value in command]
    if present:
        raise ValueError(
            f"Node '{node_id}' command contains forbidden substrings: {', '.join(present)}. "
            f"Got command: {command}"
        )

    required_non_empty_flags = contract.get("required_non_empty_flags") or []
    if required_non_empty_flags:
        missing_values = []
        for flag in required_non_empty_flags:
            flag = str(flag)
            value = extract_shell_flag_value(command, flag)
            if value is None or value == "" or value.startswith("--"):
                missing_values.append(flag)

        if missing_values:
            raise ValueError(
                f"Node '{node_id}' command has empty required flag values: {', '.join(missing_values)}. "
                f"Got command: {command}"
            )

def extract_shell_flag_value(command: str, flag: str) -> Optional[str]:
    """
    Extracts a shell flag value without parsing the entire command.

    This intentionally avoids shlex.split(command), because generated commands may
    contain complex quoting in unrelated arguments. Contract validation should
    only validate the local flag it cares about.
    """
    flag_pattern = re.escape(flag)
    match = re.search(
        rf"(^|\s){flag_pattern}(?:=(?P<eq>\S*)|\s+(?P<value>\"[^\"]*\"|'[^']*'|\S+))",
        command,
        re.DOTALL,
    )
    if not match:
        return None

    value = match.group("eq") if match.group("eq") is not None else match.group("value")
    if value is None:
        return None

    value = value.strip()
    if len(value) >= 2 and value[0] == value[-1] and value[0] in {"'", '"'}:
        return value[1:-1]
    return value

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
    execution_status = "success" if successful else "error"
    stdout = str(latest.get("stdout") or "")
    stderr = str(latest.get("stderr") or "")
    parsed_stdout = parse_labeled_stdout(stdout)

    output: Dict[str, Any] = {
        "status": execution_status,
        "tool_status": execution_status,
        "tool_results": tool_results,
        "tool_stdout": stdout,
        "tool_stderr": stderr,
        "stdout_fields": parsed_stdout,
        "raw_response": final_content or stdout,
    }

    # Expose structured keys from MCP tool results at the top level.
    # CLI tools use stdout_fields; MCP tools return structured dicts directly.
    _meta_keys = {"tool", "status", "stdout", "stderr", "error", "exit_code", "command", "success"}
    if isinstance(latest, dict):
        for key, value in latest.items():
            if key not in _meta_keys and key not in output:
                output[key] = value

    missing_mapped_fields: List[str] = []
    for key, spec in (tool_output_mapping or {}).items():
        value = resolve_tool_output_spec(spec, output)
        output[key] = value
        if value is None and is_required_tool_output_mapping(spec):
            missing_mapped_fields.append(key)

    if missing_mapped_fields:
        output["missing_mapped_fields"] = missing_mapped_fields
        output["status"] = "error"
    elif execution_status == "error":
        output["status"] = "error"

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

def is_required_tool_output_mapping(spec: Any) -> bool:
    """
    Returns whether a tool output mapping must resolve to a real value.
    """
    if isinstance(spec, str):
        return spec.startswith("$.")
    if not isinstance(spec, dict):
        return False
    if spec.get("type") == "constant":
        return False
    if spec.get("required") is False:
        return False
    return True

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
