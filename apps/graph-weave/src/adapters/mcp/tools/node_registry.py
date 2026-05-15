import re
from typing import Dict, Any, List, Optional
from src.app_logging import get_logger
from ..infra.exceptions import ToolExecutionError
from src.adapters.node import RedisNodeStore

logger = get_logger(__name__)

def handle_node_registry(node_registry_tool: 'NodeRegistryTool', operation: str, **kwargs) -> Dict[str, Any]:
    """Execute a node registry operation using the NodeRegistryTool."""
    try:
        if not hasattr(node_registry_tool, operation):
            return {
                "tool": "node_registry",
                "operation": operation,
                "status": "error",
                "error": f"Unknown Node Registry operation: {operation}"
            }
            
        method = getattr(node_registry_tool, operation)
        result = method(**kwargs)
        
        return {
            "tool": "node_registry",
            "operation": operation,
            "status": "success" if result.get("success", False) else "error",
            **result
        }
    except Exception as e:
        raise ToolExecutionError(f"Failed to execute node registry operation '{operation}': {str(e)}")


class NodeRegistryTool:
    """
    Tool for interacting with the Node Registry (Redis-backed).
    Allows searching, retrieving, and resolving node definitions.
    """

    def __init__(self, node_store: RedisNodeStore):
        """
        Initialize the NodeRegistryTool.
        
        Args:
            node_store: The RedisNodeStore instance to use.
        """
        self.node_store = node_store

    def search_nodes(self, query: Optional[str] = None, tags: Optional[List[str]] = None, node_name: Optional[str] = None) -> Dict[str, Any]:
        """Search for nodes in the registry."""
        import asyncio
        
        # Since this tool is called from a synchronous context in MCPRouter, 
        # but RedisNodeStore is async, we need to handle the event loop.
        # However, MCPRouter's execute_tool is currently sync.
        # Let's check how other async-ish tools are handled.
        # Actually, RedisNodeStore methods are async.
        
        # I'll use a helper to run async in sync if needed, 
        # but let's see if I can make the tool methods async and have MCPRouter handle it.
        # MCPRouter.execute_tool is sync.
        
        return self._run_async(self._search_nodes(query, tags, node_name))

    def list_current(self, page_size: int = 200) -> Dict[str, Any]:
        """Return the current node inventory."""
        return self._run_async(self._list_current(page_size))

    async def _list_current(self, page_size: int = 200) -> Dict[str, Any]:
        try:
            result = await self.node_store.list(page_size=page_size)
            return {
                "success": True,
                "nodes": [n.model_dump() for n in result.nodes],
                "total": result.total,
            }
        except Exception as e:
            return {"success": False, "error": str(e)}

    def resolve_steps(self, steps: List[Dict[str, Any]], page_size: int = 200) -> Dict[str, Any]:
        """Resolve intent-analysis steps to existing nodes or creation suggestions."""
        return self._run_async(self._resolve_steps(steps, page_size))

    async def _resolve_steps(self, steps: List[Dict[str, Any]], page_size: int = 200) -> Dict[str, Any]:
        try:
            inventory = await self.node_store.list(page_size=page_size)
            existing_nodes = [n.model_dump() for n in inventory.nodes]
            resolved = []

            for step in steps or []:
                if not isinstance(step, dict):
                    continue
                alias = step.get("step_id") or step.get("alias")
                if not alias:
                    continue

                match = self._find_step_match(step, existing_nodes)
                if match:
                    resolved.append(
                        {
                            "alias": alias,
                            "status": "exists",
                            "node_id": match["node_id"],
                            "suggestion": None,
                        }
                    )
                    continue

                resolved.append(
                    {
                        "alias": alias,
                        "status": "missing",
                        "node_id": None,
                        "suggestion": {
                            "node_name": self._suggest_node_name(step),
                            "description": step.get("description") or step.get("purpose") or "",
                            "type": step.get("type") or "agent_node",
                            "capabilities": step.get("capabilities") or [],
                            "intent_context": step.get("purpose") or step.get("description") or "",
                        },
                    }
                )

            return {
                "success": True,
                "nodes": resolved,
                "total": len(resolved),
                "registry_total": inventory.total,
            }
        except Exception as e:
            return {"success": False, "error": str(e)}

    def _find_step_match(self, step: Dict[str, Any], existing_nodes: List[Dict[str, Any]]) -> Optional[Dict[str, Any]]:
        aliases = self._candidate_names(step)
        step_capabilities = {self._normalize_token(c) for c in step.get("capabilities") or []}
        step_type = step.get("type")

        best_match = None
        best_score = 0
        for node in existing_nodes:
            node_name = self._normalize_token(node.get("node_name") or node.get("name") or "")
            node_id_name = self._normalize_token((node.get("node_id") or "").split(":")[0])
            node_type = node.get("type")
            node_tags = {self._normalize_token(t) for t in node.get("tags") or []}

            score = 0
            if node_type == step_type:
                score += 2
            if node_name in aliases or node_id_name in aliases:
                score += 6
            if aliases & {node_name.removesuffix("_node"), node_id_name.removesuffix("_node")}:
                score += 4
            if step_capabilities and node_tags:
                score += len(step_capabilities & node_tags)

            if score > best_score:
                best_score = score
                best_match = node

        return best_match if best_score >= 3 else None

    def _candidate_names(self, step: Dict[str, Any]) -> set[str]:
        raw = [
            step.get("step_id") or "",
            step.get("alias") or "",
            step.get("node_name") or "",
        ]
        names = set()
        for value in raw:
            normalized = self._normalize_token(value)
            if not normalized:
                continue
            names.add(normalized)
            names.add(re.sub(r"^step_\d+_", "", normalized))
            names.add(normalized.removesuffix("_node"))
        return names

    def _suggest_node_name(self, step: Dict[str, Any]) -> str:
        aliases = sorted(self._candidate_names(step), key=len)
        return aliases[0] if aliases else "generated_node"

    def _normalize_token(self, value: Any) -> str:
        return re.sub(r"[^a-z0-9_]+", "_", str(value or "").lower()).strip("_")

    async def _search_nodes(self, query: Optional[str] = None, tags: Optional[List[str]] = None, node_name: Optional[str] = None) -> Dict[str, Any]:
        try:
            # RedisNodeStore.list handles tags and node_name
            result = await self.node_store.list(tags=tags, node_name=node_name)
            
            nodes = result.nodes
            if query:
                query_lower = query.lower()
                nodes = [
                    n for n in nodes 
                    if query_lower in n.name.lower() or query_lower in n.description.lower()
                ]
            
            return {
                "success": True,
                "nodes": [n.model_dump() for n in nodes],
                "total": len(nodes)
            }
        except Exception as e:
            return {"success": False, "error": str(e)}

    def get_node(self, node_id: str) -> Dict[str, Any]:
        """Get a specific node definition."""
        return self._run_async(self._get_node(node_id))

    async def _get_node(self, node_id: str) -> Dict[str, Any]:
        try:
            node = await self.node_store.get(node_id)
            if not node:
                return {"success": False, "error": f"Node {node_id} not found"}
            
            return {
                "success": True,
                "node": node.model_dump()
            }
        except Exception as e:
            return {"success": False, "error": str(e)}

    def find_compatible(self, node_name: Optional[str] = None, required_version: Optional[str] = None) -> Dict[str, Any]:
        """Find the latest compatible version of a node."""
        if not node_name:
            return {
                "success": False,
                "error": "node_name is required for find_compatible",
            }
        return self._run_async(self._find_compatible(node_name, required_version))

    async def _find_compatible(self, node_name: str, required_version: Optional[str] = None) -> Dict[str, Any]:
        try:
            node = await self.node_store.find_compatible(node_name, required_version)
            if not node:
                return {"success": False, "error": f"No compatible version found for {node_name}"}
            
            return {
                "success": True,
                "node": node.model_dump()
            }
        except Exception as e:
            return {"success": False, "error": str(e)}

    def _run_async(self, coro):
        """Helper to run async coroutine in sync context."""
        import asyncio
        try:
            loop = asyncio.get_event_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            
        if loop.is_running():
            # If we are already in an event loop (e.g. FastAPI/Uvicorn), 
            # we should really be using await.
            # But the tool interface is currently sync.
            # This is a bit of a hack but necessary if we can't change the caller.
            import nest_asyncio
            nest_asyncio.apply()
            
        return loop.run_until_complete(coro)

    def create_node(self, node_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new node in the registry."""
        return self._run_async(self._create_node(node_data))

    async def _create_node(self, node_data: Dict[str, Any]) -> Dict[str, Any]:
        try:
            from src.models.node import NodeCreate
            node_create = NodeCreate(**node_data)
            node = await self.node_store.create(node_create)
            return {
                "success": True,
                "node": node.model_dump()
            }
        except Exception as e:
            return {"success": False, "error": str(e)}
