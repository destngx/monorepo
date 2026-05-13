import re
import hashlib
from functools import wraps
from typing import Any, Dict, List, Optional, Callable

def _tool_response_cache(func: Callable[..., Any]) -> Callable[..., Any]:
    """Decorator for caching tool responses based on input hash."""
    cache: Dict[str, Any] = {}

    @wraps(func)
    def wrapper(*args: Any, **kwargs: Any) -> Any:
        cache_key = f"{func.__name__}:{str(args)}:{str(kwargs)}"
        cache_hash = hashlib.md5(cache_key.encode()).hexdigest()

        if cache_hash in cache:
            return cache[cache_hash]

        result = func(*args, **kwargs)
        cache[cache_hash] = result
        return result

    return wrapper

def format_prompt(template: str, context: Dict[str, Any]) -> str:
    """Format a prompt template with variable interpolation.
    Supports dot notation for nested objects (e.g. {metadata.title}).
    """
    def resolver(match):
        path = match.group(1)
        keys = path.split('.')
        current = context
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current[key]
            else:
                return f"{{{path}}}" # Return original if not found
        return str(current)

    return re.sub(r'\{([a-zA-Z0-9_.]+)\}', resolver, template)

def filter_allowed_tools(
    all_tools: List[str], allowed_tools: Optional[List[str]] = None
) -> List[str]:
    """Filter tools list to only allowed tools."""
    if allowed_tools is None:
        return all_tools
    return [t for t in all_tools if t in allowed_tools]
