import json
import re
from typing import Any, Dict, List, Optional

def extract_json(content: Any) -> Optional[Any]:
    """
    Robust JSON extraction: try code blocks first, then any JSON-like structure.
    """
    if not isinstance(content, str) or not content.strip():
        return None

    # Try to extract from markdown code blocks
    json_match = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
    if json_match:
        parsed = loads_json(json_match.group(1))
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
