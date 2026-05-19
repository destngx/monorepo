import re
from typing import List, Optional
from enum import Enum

from pydantic import BaseModel, Field, field_validator

from .validators import validate_node_id


class NodeType(str, Enum):
    AGENT_NODE = "agent_node"
    CLI_NODE = "cli_node"
    ORCHESTRATOR = "orchestrator"
    BRANCH = "branch"
    ENTRY = "entry"
    EXIT = "exit"
    GUARDRAIL = "guardrail"
    SKILL_LOADER = "skill_loader"


class ContractField(BaseModel):
    name: str
    type: str
    description: str = ""
    state_path: str = ""


class InputContract(BaseModel):
    required: List[ContractField] = Field(default_factory=list)
    optional: List[ContractField] = Field(default_factory=list)


class OutputContract(BaseModel):
    produced: List[ContractField] = Field(default_factory=list)


class NodeConfig(BaseModel):
    system_prompt: str = ""
    user_prompt_template: str = ""
    input_mapping: dict = Field(default_factory=dict)
    output_schema: dict = Field(default_factory=dict)
    output_key: str = ""
    command: str = ""
    provider: str = ""
    model: str = ""
    temperature: float = 0.0

    @field_validator("output_key")
    @classmethod
    def validate_output_key(cls, v: str) -> str:
        if v and not re.match(r"^[a-z_][a-z0-9_]*$", v):
            raise ValueError(
                f"Invalid output_key format: '{v}'. Must be lowercase snake_case."
            )
        return v


class Provenance(BaseModel):
    generated_by_workflow: Optional[str] = None
    generated_by_run_id: Optional[str] = None
    generated_at: Optional[str] = None
    source_intent: Optional[str] = None


class NodeCreate(BaseModel):
    tenant_id: str
    node_id: str
    node_name: str
    version: str
    name: str
    type: NodeType
    description: str = ""
    config: NodeConfig
    input_contract: InputContract
    output_contract: OutputContract
    capabilities: List[str] = Field(default_factory=list)
    tags: List[str] = Field(default_factory=list)
    owner: str = "system"
    status: str = "active"
    reuse_eligible: bool = True
    provenance: Provenance = Field(default_factory=Provenance)

    @field_validator("status")
    @classmethod
    def validate_status(cls, v: str) -> str:
        allowed = {"active", "deprecated", "inactive"}
        if v not in allowed:
            raise ValueError(f"Invalid status: '{v}'. Must be one of {allowed}")
        return v

    @field_validator("tags")
    @classmethod
    def validate_tags(cls, v: List[str]) -> List[str]:
        tag_pattern = re.compile(r"^[a-z0-9-]+$")
        for tag in v:
            if not tag_pattern.match(tag):
                raise ValueError(
                    f"Invalid tag format: '{tag}'. Must be lowercase kebab-case."
                )
        return v

    def validate_node_id_format(self) -> None:
        validate_node_id(self.node_id)
