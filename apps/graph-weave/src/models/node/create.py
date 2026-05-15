from typing import List, Optional

from pydantic import BaseModel, Field

from .validators import validate_node_id


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
    cwd: str = ""
    provider: str = ""
    model: str = ""
    temperature: float = 0.0


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
    type: str
    description: str = ""
    config: NodeConfig
    input_contract: InputContract
    output_contract: OutputContract
    tags: List[str] = Field(default_factory=list)
    owner: str = "system"
    status: str = "active"
    reuse_eligible: bool = True
    provenance: Provenance = Field(default_factory=Provenance)

    def validate_node_id_format(self) -> None:
        validate_node_id(self.node_id)

    def validate_type(self) -> None:
        if self.type not in ("agent_node", "cli_node"):
            raise ValueError(
                f"Invalid type: {self.type}. Must be agent_node or cli_node"
            )
