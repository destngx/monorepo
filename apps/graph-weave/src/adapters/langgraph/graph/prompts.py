import re
from typing import Any, Dict
from .models import WorkflowParseError

def interpolate_prompt(template: str, context: Dict[str, Any]) -> str:
    """Interpolate variables in prompt template. Supports dot-notation (e.g. {metadata.title})."""
    result = template
    pattern = r"\{([^}]+)\}"

    for match in re.finditer(pattern, template):
        var_name = match.group(1)
        val = None

        if "." in var_name:
            keys = var_name.split(".")
            current = context
            found = True
            for key in keys:
                if isinstance(current, dict) and key in current:
                    current = current[key]
                else:
                    found = False
                    break
            if found and current is not None:
                val = current
            else:
                val = context.get(var_name)
        else:
            val = context.get(var_name)

        if val is None:
            raise WorkflowParseError(f"Missing template variable: {var_name}")

        result = result.replace(f"{{{var_name}}}", str(val))

    return result
