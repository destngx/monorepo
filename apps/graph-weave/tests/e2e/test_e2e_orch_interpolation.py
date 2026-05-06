import pytest
import requests
import json
import time

API_URL = "http://localhost:8001"
TENANT_ID = "test-tenant"

@pytest.fixture
def client():
    return requests.Session()

def test_orchestrator_interpolation(client):
    """
    Verify that Orchestrator system_prompt and user_prompt_template 
    correctly interpolate variables from the workflow state.
    """
    workflow_id = f"test-orch-interp-{int(time.time())}"
    
    definition = {
        "nodes": [
            {
                "id": "entry",
                "type": "entry"
            },
            {
                "id": "orch",
                "type": "orchestrator",
                "config": {
                    "system_prompt": "You are a specialized agent helping with {service_name}.",
                    "user_prompt_template": "Focus your investigation on the {subcomponent} area.",
                    "allowed_skills": ["web_search"], # Just a dummy skill
                    "max_iterations": 2
                }
            },
            {
                "id": "exit",
                "type": "exit"
            }
        ],
        "edges": [
            {"from": "entry", "to": "orch"},
            {"from": "orch", "to": "exit"}
        ],
        "entry_point": "entry",
        "exit_point": "exit"
    }

    # Register workflow
    resp = client.post(
        f"{API_URL}/workflows",
        json={
            "tenant_id": TENANT_ID,
            "workflow_id": f"{workflow_id}:v1.0.0",
            "name": "Test Orch Interpolation",
            "version": "1.0.0",
            "definition": definition,
            "owner": "test-user"
        }
    )
    if resp.status_code == 422:
        print(f"Validation Error: {resp.text}")
    assert resp.status_code == 201

    # Execute workflow
    resp = client.post(
        f"{API_URL}/execute",
        json={
            "tenant_id": TENANT_ID,
            "workflow_id": f"{workflow_id}:v1.0.0",
            "input": {
                "service_name": "Kubernetes",
                "subcomponent": "etcd"
            }
        }
    )
    assert resp.status_code == 200
    run_id = resp.json()["run_id"]

    # Poll for completion
    for _ in range(15):
        resp = client.get(f"{API_URL}/execute/{run_id}/status")
        if resp.status_code != 200:
            time.sleep(1)
            continue
        status = resp.json()["status"]
        if status in ["completed", "failed"]:
            break
        time.sleep(2)
    
    final = resp.json()
    assert final["status"] == "completed"
    
    state = final.get("workflow_state", {})
    assert "orchestrator_result" in state
