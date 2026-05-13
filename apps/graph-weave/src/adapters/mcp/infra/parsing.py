import json
import re
from typing import Any, Dict, List, Optional
from .constants import VALID_TOOLS

def parse_tool_calls(
    response_text: str, allowed_tools: Optional[List[str]] = None
) -> List[Dict[str, Any]]:
    """
    Parses potential tool calls from the LLM's raw response text.
    """
    tool_calls = []

    try:
        parsed = json.loads(response_text)
        if isinstance(parsed, dict):
            if "tool" in parsed and "arguments" in parsed:
                tool_call = {
                    "tool": parsed["tool"],
                    "arguments": parsed["arguments"],
                }
                if validate_tool_call(tool_call, allowed_tools):
                    tool_calls.append(tool_call)
        elif isinstance(parsed, list):
            for item in parsed:
                if (
                    isinstance(item, dict)
                    and "tool" in item
                    and "arguments" in item
                ):
                    if validate_tool_call(item, allowed_tools):
                        tool_calls.append(item)
    except json.JSONDecodeError:
        pattern = r"```(?:tool_call)?\s*\n?(.*?)\n?```"
        for match in re.finditer(pattern, response_text, re.DOTALL):
            try:
                parsed = json.loads(match.group(1))
                if "tool" in parsed and "arguments" in parsed:
                    if validate_tool_call(parsed, allowed_tools):
                        tool_calls.append(parsed)
            except json.JSONDecodeError:
                continue

    return tool_calls

def validate_tool_call(
    tool_call: Dict[str, Any], allowed_tools: Optional[List[str]] = None
) -> bool:
    """Validate a single tool call."""
    if not isinstance(tool_call, dict):
        return False

    tool_name = tool_call.get("tool")
    if not tool_name or not isinstance(tool_name, str):
        return False

    if tool_name not in VALID_TOOLS:
        return False

    if allowed_tools and tool_name not in allowed_tools:
        return False

    arguments = tool_call.get("arguments")
    if not isinstance(arguments, dict):
        return False

    return True
