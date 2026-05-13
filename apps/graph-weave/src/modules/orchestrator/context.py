import json
from typing import Any, Dict, List, Optional
from src.models import OrchestratorConfig

def trim_context(messages: List[Dict[str, Any]], max_context_messages: int) -> List[Dict[str, Any]]:
    """
    Keep the message list within max_context_messages.
    Always preserves the system message (index 0) and drops the
    oldest non-system messages when over the limit.
    """
    if len(messages) <= max_context_messages:
        return messages

    system = [m for m in messages if m.get("role") == "system"]
    non_system = [m for m in messages if m.get("role") != "system"]

    keep_non_system = max_context_messages - len(system)
    if keep_non_system <= 0:
        return system

    # Keep only the most recent non-system messages
    trimmed_non_system = non_system[-keep_non_system:]
    return system + trimmed_non_system

def build_initial_messages(
    config: OrchestratorConfig,
    workflow_state: Dict[str, Any],
    user_prompt: Optional[str] = None,
) -> List[Dict[str, Any]]:
    context_summary = json.dumps(workflow_state, default=str)
    
    # If no explicit prompt provided, use the hardcoded default
    final_user_prompt = user_prompt or (
        "Use the available tools to investigate, then return your final conclusion "
        "as a JSON object when you are done."
    )

    return [
        {"role": "system", "content": config.system_prompt},
        {
            "role": "user",
            "content": (
                f"Current workflow context:\n{context_summary}\n\n"
                f"{final_user_prompt}"
            ),
        },
    ]
