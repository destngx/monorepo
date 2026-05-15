from typing import List, Optional

from pydantic import BaseModel, Field


class NodeUpdate(BaseModel):
    name: Optional[str] = None
    description: Optional[str] = None
    tags: Optional[List[str]] = None
