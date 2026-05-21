from typing import Dict, Any, Optional, List
from datetime import datetime
import json
import os
import logging
from src.adapters.redis import NamespacedRedisClient
from src.adapters.node import RedisNodeStore
from src.services.node_compiler import WorkflowCompiler

logger = logging.getLogger(__name__)

PREDEFINED_WORKFLOWS_DIR = os.path.join(
    os.path.dirname(__file__), "..", "resources", "workflows"
)

PREDEFINED_WORKFLOWS = {
    "workflow-generator:v1.0.0": "workflow-generator:v1.0.0.json",
    "workflow-generator:v2.0.0": "workflow-generator:v2.0.0.json",
    "create-node:v1.0.0": "create-node:v1.0.0.json",
}


def load_workflow_definition(resource_path: str) -> Dict[str, Any]:
    with open(resource_path, "r") as f:
        definition = json.load(f)
    return _resolve_workflow_components(definition, os.path.dirname(resource_path))


def _resolve_workflow_components(definition: Dict[str, Any], base_dir: str) -> Dict[str, Any]:
    components = definition.get("components")
    if not isinstance(components, dict):
        return definition

    node_refs = components.get("nodes") or []
    if not node_refs:
        return definition

    components_base_dir = components.get("base_dir") or ""
    if components_base_dir:
        base_dir = os.path.join(base_dir, components_base_dir)

    resolved_nodes: List[Dict[str, Any]] = []
    for node_ref in node_refs:
        if isinstance(node_ref, str):
            node_path = os.path.join(base_dir, node_ref)
            node_alias = None
        elif isinstance(node_ref, dict):
            node_path = os.path.join(base_dir, node_ref.get("file", ""))
            node_alias = node_ref.get("id")
        else:
            continue

        if not node_path or not os.path.exists(node_path):
            raise FileNotFoundError(f"Workflow component not found: {node_path}")

        with open(node_path, "r") as f:
            node_definition = json.load(f)

        if node_alias and "id" not in node_definition:
            node_definition["id"] = node_alias

        resolved_nodes.append(
            _resolve_workflow_components(node_definition, os.path.dirname(node_path))
        )

    resolved_definition = {
        **{k: v for k, v in definition.items() if k != "components"},
        "nodes": resolved_nodes,
    }
    return resolved_definition


class RedisWorkflowStore:
    def __init__(self, redis_client: NamespacedRedisClient, node_store: RedisNodeStore = None):
        self.redis_client = redis_client
        self.node_store = node_store
        self.compiler = WorkflowCompiler(node_store) if node_store else None

    async def get_compiled(self, tenant_id: str, workflow_id: str) -> Optional[Dict[str, Any]]:
        workflow = self.get(tenant_id, workflow_id)
        if not workflow:
            return None
        if self.compiler:
            return await self.compiler.compile(workflow.get("definition", workflow))
        return workflow.get("definition", workflow)

    def sync_predefined_workflows(self, tenant_id: str) -> None:
        """
        Synchronizes all predefined workflows into the tenant store.
        Ensures they are immediately available without lazy-loading.
        """
        import logging
        logger = logging.getLogger(__name__)
        logger.info(f"Syncing predefined workflows for tenant: {tenant_id}")
        for workflow_id in PREDEFINED_WORKFLOWS:
            self.get(tenant_id, workflow_id)

    def clear(self) -> None:
        """Clear all workflows from the store (for testing)."""
        self.redis_client.clear()

    def create(self, tenant_id: str, workflow: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        workflow_id = workflow["workflow_id"]
        
        # 1. Enforce global uniqueness against pre-defined workflows
        if workflow_id in PREDEFINED_WORKFLOWS:
            logger.warning(f"Attempt to create workflow shadowing pre-defined ID: {workflow_id}")
            return None

        # 2. Check if workflow already exists for this tenant
        tenant_workflows_key = f"workflows:tenant:{tenant_id}"
        if self.redis_client.hsetnx(tenant_workflows_key, workflow_id, "placeholder") == 0:
            logger.warning(f"Workflow ID already exists for tenant: {workflow_id}")
            return None

        # 3. Create the workflow data
        timestamp = datetime.utcnow().isoformat() + "Z"
        workflow_data = {
            **workflow,
            "status": workflow.get("status", "active"),
            "created_at": timestamp,
            "updated_at": timestamp,
        }
        if "tags" not in workflow_data or workflow_data["tags"] is None:
            workflow_data["tags"] = []

        # 4. Persist to Redis (individual key for detail)
        key = self.redis_client.workflow_key(workflow_id, tenant_id)
        self.redis_client.set(key, workflow_data)
        
        # 5. Update tenant index with full data
        self.redis_client.hset(tenant_workflows_key, workflow_id, workflow_data)
        
        return workflow_data

    def get(self, tenant_id: str, workflow_id: str) -> Optional[Dict[str, Any]]:
        key = self.redis_client.workflow_key(workflow_id, tenant_id)

        # Check pre-defined workflows
        if workflow_id in PREDEFINED_WORKFLOWS:
            resource_file = PREDEFINED_WORKFLOWS[workflow_id]
            resource_path = os.path.join(PREDEFINED_WORKFLOWS_DIR, resource_file)
            if os.path.isabs(resource_file):
                resource_path = resource_file

            if os.path.exists(resource_path):
                try:
                    logger.info(f"Loading pre-defined workflow '{workflow_id}' for tenant '{tenant_id}' from bundled resource")
                    definition = load_workflow_definition(resource_path)
                    
                    timestamp = datetime.utcnow().isoformat() + "Z"
                    existing_workflow = self.redis_client.get(key) or {}
                    workflow_data = {
                        **existing_workflow,
                        "tenant_id": tenant_id,
                        "workflow_id": workflow_id,
                        "name": definition.get("name", "Pre-defined Workflow"),
                        "version": definition.get("version", "1.0.0"),
                        "description": definition.get("description", ""),
                        "owner": "system",
                        "tags": definition.get("metadata", {}).get("tags", []) + ["pre-defined"],
                        "definition": definition,
                        "status": "active",
                        "created_at": timestamp,
                        "updated_at": timestamp,
                    }
                    
                    self.redis_client.set(key, workflow_data)
                    # Also update tenant index
                    tenant_workflows_key = f"workflows:tenant:{tenant_id}"
                    self.redis_client.hset(tenant_workflows_key, workflow_id, workflow_data)
                    return workflow_data
                except Exception as e:
                    logger.error(f"Failed to lazy-load pre-defined workflow '{workflow_id}': {e}")

        workflow = self.redis_client.get(key)
        if workflow:
            return workflow
            
        return None

    def update(self, tenant_id: str, workflow_id: str, updates: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        workflow = self.get(tenant_id, workflow_id)
        if not workflow:
            return None

        timestamp = datetime.utcnow().isoformat() + "Z"
        for key, value in updates.items():
            if key not in ["workflow_id", "version", "created_at", "tenant_id"]:
                workflow[key] = value

        if "tags" in workflow and workflow["tags"] is None:
            workflow["tags"] = []

        workflow["updated_at"] = timestamp
        
        key = self.redis_client.workflow_key(workflow_id, tenant_id)
        self.redis_client.set(key, workflow)
        
        # Update tenant index
        tenant_workflows_key = f"workflows:tenant:{tenant_id}"
        self.redis_client.hset(tenant_workflows_key, workflow_id, workflow)
        
        return workflow

    def delete(self, tenant_id: str, workflow_id: str) -> bool:
        key = self.redis_client.workflow_key(workflow_id, tenant_id)
        
        if self.redis_client.delete(key):
            # Remove from tenant index
            tenant_workflows_key = f"workflows:tenant:{tenant_id}"
            self.redis_client.hdel(tenant_workflows_key, workflow_id)
            return True
        return False


    def list_tenants(self) -> List[str]:
        """List all tenant IDs that have workflows registered."""
        keys = self.redis_client.keys("workflows:tenant:*")
        tenants = []
        for key in keys:
            # key is "workflows:tenant:TENANT_ID"
            parts = key.split(":")
            if len(parts) >= 3:
                tenants.append(parts[2])
        return sorted(list(set(tenants)))

    def list_for_tenant(self, tenant_id: str, status: Optional[str] = None, tags: Optional[List[str]] = None, owner: Optional[str] = None) -> List[Dict[str, Any]]:
        tenant_workflows_key = f"workflows:tenant:{tenant_id}"
        workflows_dict = self.redis_client.hgetall(tenant_workflows_key)
        
        workflows = list(workflows_dict.values())
        
        # Filter out placeholders (if any)
        workflows = [w for w in workflows if isinstance(w, dict)]
        
        if status:
            workflows = [w for w in workflows if w.get("status") == status]
        if owner:
            workflows = [w for w in workflows if w.get("owner") == owner]
        if tags:
            workflows = [w for w in workflows if any(t in w.get("tags", []) for t in tags)]
            
        return workflows
