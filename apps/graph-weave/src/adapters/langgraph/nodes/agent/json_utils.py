import json
import re
from typing import Any, Dict, List, Optional

def repair_split_arrays(content: str) -> str:
    """
    Repairs malformed JSON arrays that are split into multiple arrays (e.g. `["a"], ["b"]` instead of `["a", "b"]`)
    by merging them when they occur inside an object and outside of other arrays.
    """
    if not isinstance(content, str):
        return content

    result = []
    in_string = False
    escape = False
    brace_depth = 0
    bracket_depth = 0

    i = 0
    n = len(content)
    while i < n:
        char = content[i]

        if in_string:
            if escape:
                escape = False
            elif char == '\\':
                escape = True
            elif char == '"':
                in_string = False
            result.append(char)
            i += 1
            continue

        # Not in string
        if char == '"':
            in_string = True
            escape = False
            result.append(char)
            i += 1
            continue

        if char == '{':
            brace_depth += 1
            result.append(char)
            i += 1
            continue
        elif char == '}':
            brace_depth = max(0, brace_depth - 1)
            result.append(char)
            i += 1
            continue
        elif char == '[':
            bracket_depth += 1
            result.append(char)
            i += 1
            continue
        elif char == ']':
            # Check if this is a split array boundary
            if bracket_depth == 1 and brace_depth > 0:
                # Look ahead for optional whitespace, optional single comma, optional whitespace, and '['
                j = i + 1
                seen_comma = False
                has_match = False
                while j < n:
                    c = content[j]
                    if c.isspace():
                        j += 1
                    elif c == ',' and not seen_comma:
                        seen_comma = True
                        j += 1
                    elif c == '[':
                        has_match = True
                        break
                    else:
                        break
                
                if has_match:
                    # Merge by replacing the range from i to j (from ']' to '[') with ', '
                    result.append(', ')
                    i = j + 1
                    continue

            bracket_depth = max(0, bracket_depth - 1)
            result.append(char)
            i += 1
            continue

        result.append(char)
        i += 1

    return "".join(result)

def extract_json(content: Any) -> Optional[Any]:
    """
    Robust JSON extraction: try code blocks first, then any JSON-like structure.
    """
    if not isinstance(content, str) or not content.strip():
        return None

    repaired = repair_split_arrays(content)
    return _extract_json_internal(repaired)

def _extract_json_internal(content: str) -> Optional[Any]:
    # Try to extract using a greedy regex (matches from first ```json to last ```)
    # This prevents truncation on inner markdown code blocks within the JSON string values
    json_match = re.search(r"```(?:json)?\s*(.*)\s*```", content, re.DOTALL)
    if json_match:
        parsed = loads_json(json_match.group(1))
        if parsed is not None:
            return parsed

    # Fallback to non-greedy extract from markdown code blocks
    json_match_non_greedy = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
    if json_match_non_greedy:
        parsed = loads_json(json_match_non_greedy.group(1))
        if parsed is not None:
            return parsed

    # Try to parse the entire content
    parsed = loads_json(content)
    if parsed is not None:
        return parsed

    # Fallback: scan for first '{' or '[' and try raw decode
    decoder = json.JSONDecoder()
    for index, char in enumerate(content):
        if char not in "{[":
            continue
        try:
            value, _ = decoder.raw_decode(content[index:])
            return value
        except json.JSONDecodeError:
            continue

    return None

def loads_json(content: str) -> Optional[Any]:
    """
    Safely loads JSON string.
    """
    try:
        return json.loads(content)
    except (TypeError, json.JSONDecodeError):
        return None

def repair_schema_json(
    client: Any,
    messages: List[Dict[str, Any]],
    provider: str,
    model: str,
    temperature: float,
    max_tokens: int,
    reasoning_effort: Optional[str],
    output_schema: Dict[str, Any],
    bad_content: str,
    validation_error: Optional[str] = None,
) -> Any:
    """
    Asks the LLM to repair a JSON response that didn't match the schema.
    """
    error_context = f"\n\nValidation error:\n{validation_error}" if validation_error else ""
    repair_messages = messages + [
        {
            "role": "user",
            "content": (
                "Your previous response was not valid JSON for the required output schema. "
                "Return only one raw JSON value. Do not include markdown, commentary, or reasoning.\n\n"
                f"Required output schema:\n{json.dumps(output_schema, indent=2)}\n\n"
                f"Previous response:\n{bad_content}"
                f"{error_context}"
            ),
        }
    ]

    response = client.chat_completion(
        messages=repair_messages,
        provider=provider,
        model=model,
        tools=None,
        temperature=0,
        max_tokens=max_tokens,
        reasoning_effort=reasoning_effort,
    )
    repaired_content = response["choices"][0]["message"].get("content", "")
    repaired = extract_json(repaired_content)
    if repaired is None:
        raise ValueError("Schema-bound agent returned non-JSON output and JSON repair failed")
    return repaired
