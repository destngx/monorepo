from typing import Dict, Any, Optional, List
from datetime import datetime
import json
import os
import logging
from ..adapters.redis_circuit_breaker import NamespacedRedisClient

logger = logging.getLogger(__name__)

# Directory where built-in workflows are bundled
PREDEFINED_WORKFLOWS_DIR = os.path.join(
    os.path.dirname(__file__), "..", "resources", "workflows"
)

# Registry of pre-defined workflow IDs to their resource filenames
PREDEFINED_WORKFLOWS = {
    "workflow-generator:v1.0.0": "workflow-generator:v1.0.0.json",
}


class RedisWorkflowStore:
    """Production workflow store backed by Redis with global uniqueness enforcement."""

    GLOBAL_REGISTRY_KEY = "workflows:global:registry"

    def __init__(self, redis_client: NamespacedRedisClient):
        self.redis_client = redis_client

    def create(self, tenant_id: str, workflow: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        workflow_id = workflow["workflow_id"]
        
        # 1. Enforce global uniqueness against pre-defined workflows
        if workflow_id in PREDEFINED_WORKFLOWS:
            logger.warning(f"Attempt to create workflow shadowing pre-defined ID: {workflow_id}")
            return None

        # 2. Enforce global uniqueness in Redis registry
        # HSETNX returns 1 if field is new, 0 if it exists
        if self.redis_client.hsetnx(self.GLOBAL_REGISTRY_KEY, workflow_id, tenant_id) == 0:
            # Check if it's the same tenant (might be a retry)
            owner = self.redis_client.hget(self.GLOBAL_REGISTRY_KEY, workflow_id)
            if owner != tenant_id:
                logger.warning(f"Workflow ID collision: {workflow_id} already owned by {owner}")
                return None
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
        version = workflow.get("version", "1.0.0")
        key = self.redis_client.workflow_key(workflow_id, tenant_id, version)
        self.redis_client.set(key, workflow_data)
        
        # 5. Update tenant index (for listing)
        tenant_workflows_key = f"workflows:tenant:{tenant_id}"
        self.redis_client.hset(tenant_workflows_key, workflow_id, workflow_data)
        
        return workflow_data

    def get(self, tenant_id: str, workflow_id: str) -> Optional[Dict[str, Any]]:
        # Reconstruct key using version from workflow_id
        version = workflow_id.split(":")[-1].lstrip("v") if ":" in workflow_id else "1.0.0"
        key = self.redis_client.workflow_key(workflow_id, tenant_id, version)
        workflow = self.redis_client.get(key)
        
        if workflow:
            return workflow

        # Check pre-defined workflows
        if workflow_id in PREDEFINED_WORKFLOWS:
            resource_file = PREDEFINED_WORKFLOWS[workflow_id]
            resource_path = os.path.join(PREDEFINED_WORKFLOWS_DIR, resource_file)

            if os.path.exists(resource_path):
                try:
                    logger.info(f"Lazy-loading pre-defined workflow '{workflow_id}' for tenant '{tenant_id}' to Redis")
                    with open(resource_path, "r") as f:
                        definition = json.load(f)
                    
                    timestamp = datetime.utcnow().isoformat() + "Z"
                    workflow_data = {
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
        
        version = workflow_id.split(":")[-1].lstrip("v") if ":" in workflow_id else "1.0.0"
        key = self.redis_client.workflow_key(workflow_id, tenant_id, version)
        self.redis_client.set(key, workflow)
        
        # Update tenant index
        tenant_workflows_key = f"workflows:tenant:{tenant_id}"
        self.redis_client.hset(tenant_workflows_key, workflow_id, workflow)
        
        return workflow

    def delete(self, tenant_id: str, workflow_id: str) -> bool:
        version = workflow_id.split(":")[-1].lstrip("v") if ":" in workflow_id else "1.0.0"
        key = self.redis_client.workflow_key(workflow_id, tenant_id, version)
        
        if self.redis_client.delete(key):
            # 1. Remove from global registry
            self.redis_client.hdel(self.GLOBAL_REGISTRY_KEY, workflow_id)
            # 2. Remove from tenant index
            tenant_workflows_key = f"workflows:tenant:{tenant_id}"
            self.redis_client.hdel(tenant_workflows_key, workflow_id)
            return True
        return False

    def list_for_tenant(self, tenant_id: str, status: Optional[str] = None, tags: Optional[List[str]] = None, owner: Optional[str] = None) -> List[Dict[str, Any]]:
        tenant_workflows_key = f"workflows:tenant:{tenant_id}"
        workflows_dict = self.redis_client.hgetall(tenant_workflows_key)
        
        workflows = list(workflows_dict.values())
        
        if status:
            workflows = [w for w in workflows if w.get("status") == status]
        if owner:
            workflows = [w for w in workflows if w.get("owner") == owner]
        if tags:
            workflows = [w for w in workflows if any(t in w.get("tags", []) for t in tags)]
            
        return workflows
