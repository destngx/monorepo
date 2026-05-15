from typing import Optional, List, Tuple
import inspect
import re

from ..models.node import (
    NodeCreate,
    NodeUpdate,
    NodeResponse,
    NodeListResponse,
)
from ..models.node.validators import NodeIdFormatError, parse_node_id
from .redis.namespaced import NamespacedRedisClient


class ConflictError(Exception):
    pass


class NotFoundError(Exception):
    pass


class VersionIncompatibleError(Exception):
    pass


def parse_version(version: str) -> Tuple[int, int, int]:
    """Parse version string to tuple for comparison."""
    match = re.match(r"v?(\d+)\.(\d+)\.(\d+)", version)
    if not match:
        raise ValueError(f"Invalid version format: {version}")
    return (int(match.group(1)), int(match.group(2)), int(match.group(3)))


def is_version_compatible(required: str, available: str) -> bool:
    """Check if available version is compatible with required.
    
    Compatible means:
    - Same major version
    - Available minor >= required minor
    - Available patch >= required patch (if minor is same)
    """
    req_major, req_minor, req_patch = parse_version(required)
    avail_major, avail_minor, avail_patch = parse_version(available)
    
    if req_major != avail_major:
        return False
    if avail_minor < req_minor:
        return False
    if avail_minor == req_minor and avail_patch < req_patch:
        return False
    return True


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

    async def _redis_call(self, method_name: str, *args, **kwargs):
        result = getattr(self.redis, method_name)(*args, **kwargs)
        if inspect.isawaitable(result):
            return await result
        return result

    async def create(self, node: NodeCreate) -> NodeResponse:
        key = self._node_key(node.node_id)

        if await self._redis_call("exists", key):
            raise ConflictError(f"Node {node.node_id} already exists")

        node_dict = node.model_dump()
        node_dict["created_at"] = ""
        node_dict["immutable_fields"] = [
            "config",
            "input_contract",
            "output_contract",
            "type",
        ]

        await self._redis_call("hset", key, field="data", value=node_dict)
        await self._redis_call("sadd", self._index_key(), node.node_id)
        await self._redis_call("sadd", self._name_key(node.node_name), node.node_id)

        for tag in node.tags:
            await self._redis_call("sadd", self._tag_key(tag), node.node_id)

        return NodeResponse(**node_dict)

    async def get(self, node_id: str) -> Optional[NodeResponse]:
        key = self._node_key(node_id)
        data = await self._redis_call("hget", key, "data")
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
        page: int = 1,
        page_size: int = 20,
    ) -> NodeListResponse:
        if node_name:
            node_ids = await self._redis_call("smembers", self._name_key(node_name))
        elif tags:
            tag_keys = [self._tag_key(t) for t in tags]
            node_ids = await self._redis_call("sinter", *tag_keys)
        else:
            node_ids = await self._redis_call("smembers", self._index_key())

        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node:
                nodes.append(node)

        # Apply pagination
        total = len(nodes)
        start = (page - 1) * page_size
        end = start + page_size
        paginated_nodes = nodes[start:end]

        return NodeListResponse(
            nodes=paginated_nodes,
            total=total,
            page=page,
            page_size=page_size,
        )

    async def update(self, node_id: str, update: NodeUpdate) -> NodeResponse:
        node = await self.get(node_id)
        if not node:
            raise NotFoundError(f"Node {node_id} not found")

        update_data = update.model_dump(exclude_unset=True)
        node_dict = node.model_dump()
        for field in update_data:
            if field in ("name", "description", "tags"):
                node_dict[field] = update_data[field]

        key = self._node_key(node_id)
        await self._redis_call("hset", key, field="data", value=node_dict)

        if "tags" in update_data:
            for tag in node.tags:
                await self._redis_call("srem", self._tag_key(tag), node_id)
            for tag in update_data["tags"]:
                await self._redis_call("sadd", self._tag_key(tag), node_id)

        return NodeResponse(**node_dict)

    async def delete(self, node_id: str) -> bool:
        node = await self.get(node_id)
        if not node:
            return False

        key = self._node_key(node_id)
        await self._redis_call("srem", self._index_key(), node_id)
        await self._redis_call("srem", self._name_key(node.node_name), node_id)

        for tag in node.tags:
            await self._redis_call("srem", self._tag_key(tag), node_id)

        await self._redis_call("delete", key)
        return True

    async def exists(self, node_id: str) -> bool:
        return await self._redis_call("exists", self._node_key(node_id))

    async def find_by_name(self, node_name: str) -> List[NodeResponse]:
        node_ids = await self._redis_call("smembers", self._name_key(node_name))
        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node:
                nodes.append(node)
        return nodes

    async def find_compatible(
        self, node_name: str, required_version: str = None
    ) -> Optional[NodeResponse]:
        """Find latest compatible version of a node."""
        nodes = await self.find_by_name(node_name)
        if not nodes:
            return None
        
        # Sort by version descending
        nodes.sort(key=lambda n: parse_version(n.version), reverse=True)
        
        if required_version:
            for node in nodes:
                if is_version_compatible(required_version, node.version):
                    return node
            return None
        
        # Return latest version
        return nodes[0]

    async def find_latest(self, node_name: str) -> Optional[NodeResponse]:
        """Find latest version of a node."""
        nodes = await self.find_by_name(node_name)
        if not nodes:
            return None
        
        # Sort by version descending
        nodes.sort(key=lambda n: parse_version(n.version), reverse=True)
        return nodes[0]

    async def count_references(self, node_id: str) -> int:
        """Count how many workflows reference this node."""
        # This would require scanning workflow definitions
        # For now, return 0 (implement with proper indexing later)
        return 0

    async def soft_delete(self, node_id: str) -> bool:
        """Soft delete a node (mark as deprecated)."""
        node = await self.get(node_id)
        if not node:
            return False
        
        node_dict = node.model_dump()
        node_dict["status"] = "deprecated"
        
        key = self._node_key(node_id)
        await self._redis_call("hset", key, field="data", value=node_dict)
        return True
