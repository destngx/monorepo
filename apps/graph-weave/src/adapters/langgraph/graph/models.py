from dataclasses import dataclass
from typing import List, Optional

@dataclass
class NodeConfig:
    """Per-node configuration override."""
    provider: Optional[str] = None
    model: Optional[str] = None
    temperature: Optional[float] = None
    max_tokens: Optional[int] = None
    tools: Optional[List[str]] = None

VALID_PROVIDERS = {"github-copilot", "openai", "anthropic"}
PROVIDER_MODELS = {
    "github-copilot": ["claude-3.5-sonnet", "claude-3-opus", "gpt-5.4-mini"],
    "openai": ["gpt-5.4-mini", "gpt-3.5-turbo"],
    "anthropic": ["claude-3-opus", "claude-3-sonnet"],
}
VALID_TOOLS = {"load_skill", "search", "verify", "bash", "fs", "fetch"}

class WorkflowParseError(Exception):
    pass
