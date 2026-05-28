from typing import Optional, List, Tuple
import inspect
import json
import logging
import os
import re

from ..models.node import (
    NodeCreate,
    NodeUpdate,
    NodeResponse,
    NodeListResponse,
)
from ..models.node.validators import NodeIdFormatError, parse_node_id
from .redis.namespaced import NamespacedRedisClient

_logger = logging.getLogger(__name__)

# Directory where bundled system node JSON definitions live
_NODES_RESOURCE_DIR = os.path.join(os.path.dirname(__file__), "..", "resources", "nodes")


def _find_node_resource(node_id: str) -> Optional[dict]:
    """Scan resources/nodes/ for a JSON file whose node_id matches. Returns raw dict or None."""
    if not os.path.isdir(_NODES_RESOURCE_DIR):
        return None
    for fname in os.listdir(_NODES_RESOURCE_DIR):
        if not fname.endswith(".json"):
            continue
        fpath = os.path.join(_NODES_RESOURCE_DIR, fname)
        try:
            with open(fpath, "r") as f:
                data = json.load(f)
            if data.get("node_id") == node_id:
                return data
        except Exception:
            continue
    return None


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

    def _capability_key(self, capability: str) -> str:
        return f"nodes:{self.redis.tenant_id}:capabilities:{capability}"

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

        for capability in node.capabilities:
            await self._redis_call("sadd", self._capability_key(capability), node.node_id)

        return NodeResponse(**node_dict)

    async def get(self, node_id: str) -> Optional[NodeResponse]:
        key = self._node_key(node_id)
        data = await self._redis_call("hget", key, "data")

        # Lazy-load from bundled resources on a Redis miss
        if not data:
            raw = _find_node_resource(node_id)
            if raw:
                _logger.info(f"Lazy-loading system node from resources: {node_id}")
                # Build a complete node dict with safe defaults for optional fields
                node_dict = {
                    "tenant_id": raw.get("tenant_id", "system"),
                    "node_id": raw.get("node_id", node_id),
                    "node_name": raw.get("node_name") or node_id,
                    "version": raw.get("version", "1.0.0"),
                    "name": raw.get("name") or raw.get("display_name") or node_id,
                    "type": raw.get("type", "agent_node"),
                    "description": raw.get("description", ""),
                    "config": raw.get("config") or {},
                    "input_contract": raw.get("input_contract") or {"required": [], "optional": []},
                    "output_contract": raw.get("output_contract") or {"produced": []},
                    "capabilities": raw.get("capabilities") or [],
                    "tags": raw.get("tags") or [],
                    "owner": raw.get("owner", "system"),
                    "status": raw.get("status", "active"),
                    "reuse_eligible": raw.get("reuse_eligible", True),
                    "provenance": raw.get("provenance") or {},
                    "created_at": "",
                    "immutable_fields": ["config", "input_contract", "output_contract", "type"],
                }
                # Cache to Redis so next lookup is instant
                try:
                    await self._redis_call("hset", key, field="data", value=node_dict)
                    await self._redis_call("sadd", self._index_key(), node_dict["node_id"])
                except Exception as cache_err:
                    _logger.debug(f"Could not cache lazy-loaded node {node_id}: {cache_err}")
                return NodeResponse(**node_dict)
            return None

        if isinstance(data, str):
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
            if field in ("name", "description", "tags", "config"):
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

        for capability in node.capabilities:
            await self._redis_call("srem", self._capability_key(capability), node_id)

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

    async def find_by_capability(self, capability: str) -> List[NodeResponse]:
        """Find all active catalog nodes dynamically with a given capability."""
        # Try fetching from set index first
        node_ids = await self._redis_call("smembers", self._capability_key(capability))
        if not node_ids:
            # Fallback: scan all nodes and check capabilities (robust fallback for non-indexed nodes)
            all_node_ids = await self._redis_call("smembers", self._index_key())
            nodes = []
            for node_id in all_node_ids:
                node = await self.get(node_id)
                if node and node.status == "active" and capability in getattr(node, "capabilities", []):
                    nodes.append(node)
            return nodes
        
        nodes = []
        for node_id in node_ids:
            node = await self.get(node_id)
            if node and node.status == "active":
                nodes.append(node)
        return nodes

    async def sync_predefined_nodes(self, tenant_id: str) -> None:
        """Synchronizes all pre-defined nodes into the node store."""
        import os
        import json
        from ..models.node import NodeCreate, NodeUpdate
        
        nodes_dir = os.path.join(
            os.path.dirname(__file__), "..", "resources", "nodes"
        )
        if not os.path.exists(nodes_dir):
            return
            
        import logging
        logger = logging.getLogger(__name__)
        logger.info(f"Syncing predefined nodes for tenant: {tenant_id}")
        
        for file_name in os.listdir(nodes_dir):
            if not file_name.endswith(".json"):
                continue
            file_path = os.path.join(nodes_dir, file_name)
            try:
                with open(file_path, "r") as f:
                    node_data = json.load(f)
                
                node_id = node_data.get("node_id") or file_name[:-5]
                node_name = node_data.get("node_name") or node_id.split(":")[0]
                
                node_create = NodeCreate(
                    node_id=node_id,
                    node_name=node_name,
                    name=node_data.get("name") or node_data.get("display_name") or node_name,
                    type=node_data.get("type", "agent_node"),
                    tags=node_data.get("tags") or [],
                    capabilities=node_data.get("capabilities") or [],
                    description=node_data.get("description", ""),
                    config=node_data.get("config") or {},
                )
                
                try:
                    await self.create(node_create)
                    logger.info(f"  ✓ Seeded predefined node: {node_id}")
                except Exception:
                    # Update to keep it fresh
                    node_update = NodeUpdate(
                        name=node_create.name,
                        description=node_create.description,
                        tags=node_create.tags,
                        config=node_create.config,
                    )
                    await self.update(node_id, node_update)
            except Exception as e:
                logger.debug(f"Failed to seed node {file_name}: {e}")
