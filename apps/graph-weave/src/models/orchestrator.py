from pydantic import BaseModel, Field, field_validator
from typing import Optional, List, Dict, Any


class TraceEntry(BaseModel):
    """A single step in the orchestrator's ReAct trace."""

    type: str = Field(
        ...,
        description="Entry type: 'thought', 'action', 'observation'",
        json_schema_extra={"example": "thought"},
    )
    iteration: int = Field(..., description="Zero-based iteration index")
    content: Any = Field(..., description="Raw text (thought) or tool result (observation)")
    tool_name: Optional[str] = Field(None, description="Tool name when type is 'action'")
    tool_args: Optional[Dict[str, Any]] = Field(None, description="Tool arguments when type is 'action'")


class OrchestratorConfig(BaseModel):
    """Configuration for an orchestrator node in a workflow definition."""

    system_prompt: str = Field(
        ...,
        description="LLM persona and task context for the ReAct loop",
        min_length=1,
    )
    allowed_skills: List[str] = Field(
        ...,
        description="MCP skill names made available to the orchestrator",
        min_length=1,
    )
    max_iterations: int = Field(
        10,
        description="Maximum ReAct iterations before circuit breaker exits (1–50)",
        ge=1,
        le=50,
    )
    output_schema: Optional[Dict[str, Any]] = Field(
        None,
        description="Optional JSON Schema to validate the final_result output object",
    )
    provider: Optional[str] = Field(
        None,
        description="LLM provider (e.g. 'github-copilot', 'openai'). Falls back to executor default.",
        json_schema_extra={"example": "github-copilot"},
    )
    model: Optional[str] = Field(
        None,
        description="Model name override. Falls back to executor default.",
        json_schema_extra={"example": "gpt-4.1"},
    )
    input_mapping: Optional[Dict[str, str]] = Field(
        None,
        description="Maps global state paths to local keys for the orchestrator context",
    )
    user_prompt_template: Optional[str] = Field(
        None,
        description="Optional starting goal or user instructions for the ReAct loop",
    )

    @field_validator("allowed_skills")
    @classmethod
    def validate_allowed_skills(cls, v: List[str]) -> List[str]:
        if not v:
            raise ValueError("allowed_skills must contain at least one skill name")
        for skill in v:
            if not skill or not isinstance(skill, str) or not skill.strip():
                raise ValueError("each entry in allowed_skills must be a non-empty string")
        return [s.strip() for s in v]
