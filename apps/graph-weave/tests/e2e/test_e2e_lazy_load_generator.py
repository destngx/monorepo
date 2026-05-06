import requests
import time
import pytest

API_URL = "http://localhost:8001"

def test_lazy_load_workflow_generator():
    """
    Verify that the workflow-generator:v1.0.0 can be fetched and used
    for a brand new tenant without any manual registration.
    """
    tenant_id = f"lazy-tenant-{int(time.time())}"
    workflow_id = "workflow-generator:v1.0.0"

    # 1. Attempt to execute the generator immediately
    # This will trigger a 'get' in the WorkflowStore, which should lazy-load it.
    resp = requests.post(
        f"{API_URL}/execute",
        json={
            "tenant_id": tenant_id,
            "workflow_id": workflow_id,
            "input": {
                "intent": "Test lazy-loading",
                "domain": "test"
            }
        }
    )
    
    # It should succeed because 'get()' lazy-loads it into the store.
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "queued"
    print(f"✓ Successfully triggered lazy-loaded workflow execution. Run ID: {data['run_id']}")

    # 2. Verify it now appears in the tenant's workflow list
    # (Since our lazy-load caches it in the tenant's memory space)
    resp = requests.get(f"{API_URL}/workflows?tenant_id={tenant_id}")
    assert resp.status_code == 200
    workflows = resp.json().get("workflows", [])
    
    found = any(w["workflow_id"] == workflow_id for w in workflows)
    assert found, f"Workflow {workflow_id} should be visible in list after lazy-loading"
    print(f"✓ Workflow {workflow_id} is now cached and visible for tenant {tenant_id}")

if __name__ == "__main__":
    test_lazy_load_workflow_generator()
