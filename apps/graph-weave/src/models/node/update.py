from typing import List, Optional

from pydantic import BaseModel

from .create import NodeConfig


class NodeUpdate(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    tags: Optional[List[str]] = None
    config: Optional[NodeConfig] = None
