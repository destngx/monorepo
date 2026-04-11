from typing import Dict, Any, Optional, List
from datetime import datetime


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
            return None
        return self._workflows[tenant_id].get(workflow_id)

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
