from typing import List

from pydantic import BaseModel, Field

from .create import NodeConfig, InputContract, OutputContract, Provenance, NodeType


class NodeResponse(BaseModel):
    tenant_id: str
    node_id: str
    node_name: str
    version: str
    name: str
    type: NodeType
    description: str
    config: NodeConfig
    input_contract: InputContract
    output_contract: OutputContract
    capabilities: List[str] = Field(default_factory=list)
    tags: List[str]
    owner: str
    status: str
    reuse_eligible: bool
    provenance: Provenance
    created_at: str = ""
    immutable_fields: List[str] = Field(
        default_factory=lambda: ["config", "input_contract", "output_contract", "type"]
    )


class NodeListResponse(BaseModel):
    nodes: List[NodeResponse]
    total: int
    page: int = 1
    page_size: int = 20
