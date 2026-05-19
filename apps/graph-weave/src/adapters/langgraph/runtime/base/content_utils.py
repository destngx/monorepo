import re
import json
import logging
from typing import Any, Dict, Mapping, Optional

logger = logging.getLogger(__name__)

def clean_filler(content: str) -> str:
    """
    Removes common LLM filler/preamble text from responses.
    """
    if not content or not isinstance(content, str):
        return content or ""
    
    # Strip common filler patterns
    lines = content.strip().split('\n')
    skip_prefixes = (
        "Sure", "Here is", "Here's", "Certainly", "Of course",
        "I'll", "Let me", "Below is", "The following", "Based on",
        "OK", "Okay",
    )
    
    # Skip leading filler lines
    start = 0
    for i, line in enumerate(lines):
        stripped = line.strip()
        if stripped:
            if any(stripped.lower().startswith(p.lower()) for p in skip_prefixes) and not stripped.startswith("```"):
                continue
            start = i
            break
    
    # Skip trailing filler
    end = len(lines)
    for i in range(len(lines) - 1, start, -1):
        stripped = lines[i].strip()
        if stripped:
            if any(stripped.lower().startswith(p.lower()) for p in ("let me know", "feel free", "i hope", "hope this")):
                continue
            end = i + 1
            break
    
    return '\n'.join(lines[start:end]).strip()

def interpolate_prompt(
    template: str, 
    state: Mapping[str, Any], 
    get_state_value_cb,
    local_context: Optional[Dict[str, Any]] = None
) -> str:
    """
    Interpolates variables into a prompt template using the execution state.
    """
    if not template or not isinstance(template, str):
        return template

    result = template
    placeholders = re.findall(r"\{+([^{}]+)\}+", template)

    # Merge context for existence check
    context = dict(state.get("workflow_state", {}))
    context.update({k: v for k, v in state.items() if k != "workflow_state"})
    if local_context:
        context.update(local_context)

    # Collect the set of known variable names
    known_vars = set()
    if local_context:
        known_vars.update(local_context.keys())

    for p in placeholders:
        lookup_p = p
        is_shell = False
        if p.lower().endswith("_shell"):
            is_shell = True
            lookup_p = p[:-6]

        val = get_state_value_cb(lookup_p, state)
        
        if val is None and local_context and lookup_p in local_context:
            val = local_context[lookup_p]
            
        if val is None and lookup_p in context:
            val = context[lookup_p]

        if val is None and lookup_p not in known_vars and lookup_p not in context:
            continue

        val_str = ""
        if val is not None:
            if isinstance(val, dict):
                if "raw_response" in val:
                    val = val["raw_response"]
                elif "result" in val:
                    val = val["result"]

            if isinstance(val, list):
                if lookup_p.lower().endswith("_json"):
                    try:
                        val_str = json.dumps(val)
                    except (TypeError, ValueError):
                        val_str = str(val)
                elif any(hint in lookup_p.lower() for hint in ["tag", "author", "name"]):
                    val_str = ", ".join(str(i) for i in val)
                else:
                    val_str = "\n".join(str(i) for i in val)
            elif isinstance(val, dict):
                try:
                    val_str = json.dumps(val, indent=2)
                except (TypeError, ValueError):
                    val_str = str(val)
            else:
                val_str = str(val)

            if is_shell:
                val_str = val_str.replace("'", "'\\''")

        result = result.replace(f"{{{{{p}}}}}", val_str)
        result = result.replace(f"{{{p}}}", val_str)

    return result
