from typing import Optional

from typing import List, Optional

from ..models.node import (
    NodeCreate,
    NodeUpdate,
    NodeResponse,
    NodeListResponse,
)
from ..models.node.validators import NodeIdFormatError
from .redis.namespaced import NamespacedRedisClient


class ConflictError(Exception):
    pass


class NotFoundError(Exception):
    pass


class RedisNodeStore:
    def __init__(self, redis: NamespacedRedisClient):
        self.redis = redis

    def _node_key(self, node_id: str) -> str:
        return self.redis.node_key(node_id)

    def _index_key(self) -> str:
        return self.redis.node_index_key()

    def _name_key(self, node_name: str) -> str:
        return self.redis.node_name_key(node_name)

    def _tag_key(self, tag: str) -> str:
        return self.redis.node_tag_key(tag)

    async def create(self, node: NodeCreate) -> NodeResponse:
        key = self._node_key(node.node_id)

        if await self.redis.exists(key):
            raise ConflictError(f"Node {node.node_id} already exists")

        node_dict = node.dict()
        node_dict["created_at"] = ""
        node_dict["immutable_fields"] = [
            "config",
            "input_contract",
            "output_contract",
            "type",
        ]

        await self.redis.hset(key, field="data", value=node_dict)
        await self.redis.sadd(self._index_key(), node.node_id)
        await self.redis.sadd(self._name_key(node.node_name), node.node_id)

        for tag in node.tags:
            await self.redis.sadd(self._tag_key(tag), node.node_id)

        return NodeResponse(**node_dict)

    async def get(self, node_id: str) -> Optional[NodeResponse]:
        key = self._node_key(node_id)
        data = await self.redis.hget(key, "data")
        if not data:
            return None
        if isinstance(data, str):
            import json
            data = json.loads(data)
        return NodeResponse(**data)

    async def list(
        self,
        tags: Optional[list[str]] = None,
        node_name: Optional[str] = None,
    ) -> NodeListResponse:
        if node_name:
            node_ids = await self.redis.smembers(self._name_key(node_name))
        elif tags:
            tag_keys = [self._tag_key(t) for t in tags]
            node_ids = await self.redis.sinter(*tag_keys)
        else:
            node_ids = await self.redis.smembers(self._index_key())

        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node:
                nodes.append(node)

        return NodeListResponse(nodes=nodes, total=len(nodes))

    async def update(self, node_id: str, update: NodeUpdate) -> NodeResponse:
        node = await self.get(node_id)
        if not node:
            raise NotFoundError(f"Node {node_id} not found")

        update_data = update.dict(exclude_unset=True)
        node_dict = node.dict()
        for field in update_data:
            if field in ("name", "description", "tags"):
                node_dict[field] = update_data[field]

        key = self._node_key(node_id)
        await self.redis.hset(key, field="data", value=node_dict)

        if "tags" in update_data:
            for tag in node.tags:
                await self.redis.srem(self._tag_key(tag), node_id)
            for tag in update_data["tags"]:
                await self.redis.sadd(self._tag_key(tag), node_id)

        return NodeResponse(**node_dict)

    async def delete(self, node_id: str) -> bool:
        node = await self.get(node_id)
        if not node:
            return False

        key = self._node_key(node_id)
        await self.redis.srem(self._index_key(), node_id)
        await self.redis.srem(self._name_key(node.node_name), node_id)

        for tag in node.tags:
            await self.redis.srem(self._tag_key(tag), node_id)

        await self.redis.delete(key)
        return True

    async def exists(self, node_id: str) -> bool:
        return await self.redis.exists(self._node_key(node_id))

    async def find_by_name(self, node_name: str) -> List[NodeResponse]:
        node_ids = await self.redis.smembers(self._name_key(node_name))
        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node:
                nodes.append(node)
        return nodes
