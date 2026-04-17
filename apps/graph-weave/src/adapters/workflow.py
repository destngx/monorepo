from typing import Dict, Any, Optional, List
from datetime import datetime
import json
import os
import logging

logger = logging.getLogger(__name__)

# Directory where built-in workflows are bundled
PREDEFINED_WORKFLOWS_DIR = os.path.join(
    os.path.dirname(__file__), "..", "resources", "workflows"
)

# Registry of pre-defined workflow IDs to their resource filenames
PREDEFINED_WORKFLOWS = {
    "workflow-generator:v1.0.0": "workflow-generator:v1.0.0.json",
}


class MockWorkflowStore:
    def __init__(self):
        self._workflows: Dict[str, Dict[str, Dict[str, Any]]] = {}

    def create(
        self, tenant_id: str, workflow: Dict[str, Any]
    ) -> Optional[Dict[str, Any]]:
        if tenant_id not in self._workflows:
            self._workflows[tenant_id] = {}

        workflow_id = workflow["workflow_id"]
        if workflow_id in self._workflows[tenant_id]:
            return None

        timestamp = datetime.utcnow().isoformat() + "Z"
        workflow_data = {
            **workflow,
            "status": workflow.get("status", "active"),
            "created_at": timestamp,
            "updated_at": timestamp,
        }

        if "tags" not in workflow_data or workflow_data["tags"] is None:
            workflow_data["tags"] = []

        self._workflows[tenant_id][workflow_id] = workflow_data
        return workflow_data

    def get(self, tenant_id: str, workflow_id: str) -> Optional[Dict[str, Any]]:
        if tenant_id not in self._workflows:
            self._workflows[tenant_id] = {}

        # 1. Check in-memory cache
        workflow = self._workflows[tenant_id].get(workflow_id)
        if workflow:
            return workflow

        # 2. Check if it's a pre-defined workflow (Lazy Load)
        if workflow_id in PREDEFINED_WORKFLOWS:
            resource_file = PREDEFINED_WORKFLOWS[workflow_id]
            resource_path = os.path.join(PREDEFINED_WORKFLOWS_DIR, resource_file)

            if os.path.exists(resource_path):
                try:
                    logger.info(f"Lazy-loading pre-defined workflow '{workflow_id}' for tenant '{tenant_id}'")
                    with open(resource_path, "r") as f:
                        definition = json.load(f)
                    
                    # Create the workflow record in memory (simulating registration)
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
                    self._workflows[tenant_id][workflow_id] = workflow_data
                    return workflow_data
                except Exception as e:
                    logger.error(f"Failed to lazy-load pre-defined workflow '{workflow_id}': {e}")
            else:
                logger.warning(f"Pre-defined workflow resource not found: {resource_path}")

        return None

    def update(
        self, tenant_id: str, workflow_id: str, updates: Dict[str, Any]
    ) -> Optional[Dict[str, Any]]:
        if (
            tenant_id not in self._workflows
            or workflow_id not in self._workflows[tenant_id]
        ):
            return None

        workflow = self._workflows[tenant_id][workflow_id]
        timestamp = datetime.utcnow().isoformat() + "Z"

        for key, value in updates.items():
            if key not in ["workflow_id", "version", "created_at", "tenant_id"]:
                workflow[key] = value

        if "tags" in workflow and workflow["tags"] is None:
            workflow["tags"] = []

        workflow["updated_at"] = timestamp
        return workflow

    def delete(self, tenant_id: str, workflow_id: str) -> bool:
        if (
            tenant_id not in self._workflows
            or workflow_id not in self._workflows[tenant_id]
        ):
            return False

        del self._workflows[tenant_id][workflow_id]
        return True

    def list_for_tenant(
        self,
        tenant_id: str,
        status: Optional[str] = None,
        tags: Optional[List[str]] = None,
        owner: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        if tenant_id not in self._workflows:
            return []

        workflows = list(self._workflows[tenant_id].values())

        if status:
            workflows = [w for w in workflows if w.get("status") == status]

        if owner:
            workflows = [w for w in workflows if w.get("owner") == owner]

        if tags:
            workflows = [
                w for w in workflows if any(t in w.get("tags", []) for t in tags)
            ]

        return workflows

    def clear(self) -> None:
        self._workflows.clear()
