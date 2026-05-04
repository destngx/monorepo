import os
import json
import time
import httpx
import pytest

# Configuration
API_URL = "http://localhost:8001"
TENANT_ID = "persona"
GENERATOR_ID = "workflow-generator:v1.0.0"
VAULT_ROOT = "/Users/destnguyxn/projects/obsidian-vaults"

@pytest.fixture
def client():
    return httpx.Client(base_url=API_URL, timeout=120.0)

def test_inbox_ingest_generator_to_execution_real_server(client):
    """
    E2E test against the real server (localhost:8001):
    1. Run workflow-generator with intent
    2. Register the generated workflow
    3. Execute the generated workflow
    4. Verify results
    """
    
    # 1. Define the intent
    intent = """
Process a preprocessed inbox file:

INPUT (required):
- file_path: string
- file_content: string
- existing_tags: array of strings (optional, default [])

STEP 1 — classify_and_extract:
Analyze file_content and return JSON:
- source_type: one of [article, book, video, paper, fleeting, journal]
- title: string
- authors: array of strings
- url: string (empty if none)
- published_at: string YYYY-MM-DD (empty if unknown)
- summary: array of 3-5 strings
- claims: array of strings
- tags: array of 3-5 lowercase strings

STEP 2 — create_source_card:
IF source_type is "fleeting" OR "journal" → SKIP, set source_card_path = null
ELSE call:
  bash Persona/90_meta/02_scripts/agent/create_source_card.sh \
    --id src-YYYYMMDD-NNN \
    --source-kind <source_type> \
    --title <title> \
    --url <url> \
    --authors <authors joined by comma> \
    --published-at <published_at> \
    --summary <summary joined by newline> \
    --claims <claims joined by newline> \
    --tags <tags joined by comma>
Output: source_card_path

STEP 3 — create_draft:
Call:
  bash Persona/90_meta/02_scripts/agent/create_draft.sh \
    --title <title> \
    --origin-type <source_type> \
    --source-refs <source_card_path or empty> \
    --summary <summary joined by newline> \
    --claims <claims joined by newline> \
    --tags <tags joined by comma>
Output: draft_path

STEP 4 — update_original:
Call:
  bash Persona/90_meta/02_scripts/agent/update_original.sh \
    --file <file_path> \
    --summary <summary[0]> \
    --tags <tags joined by comma> \
    --source-card <source_card_path or empty> \
    --draft <draft_path>
Output: status ("processed" or "failed")

OUTPUT:
- status: "processed" | "failed"
- source_card_path: string | null
- draft_path: string
- summary: array
- tags: array
- claims: array

ERROR HANDLING:
- if any script returns non-zero exit code → set status = "failed"
- if source_card creation fails → skip, set source_card_path = null, continue
- if draft creation fails → set status = "failed", stop
- if update_original fails → set status = "failed", stop
"""

    # 2. Run workflow-generator
    print("\n[Step 1] Running workflow-generator on real server...")
    response = client.post(
        "/execute",
        json={
            "tenant_id": TENANT_ID,
            "workflow_id": GENERATOR_ID,
            "input": {
                "intent": intent,
                "domain": "pkm",
                "correction_attempts": 0
            }
        }
    )
    response.raise_for_status()
    run_id = response.json()["run_id"]
    print(f"Generator run started: {run_id}")
    
    # Poll for completion
    max_retries = 150 # 5 minutes
    status = "pending"
    generated_workflow = None
    data = {}

    time.sleep(2) # Initial delay
    
    for i in range(max_retries):
        status_resp = client.get(f"/execute/{run_id}/status")
        status_resp.raise_for_status()
        data = status_resp.json()
        status = data["status"]
        print(f"Polling Generator... Status: {status}")
        if status in ["completed", "failed"]:
            if status == "completed":
                generated_workflow = data.get("workflow_state", {}).get("generated_workflow")
            break
        time.sleep(2)
    
    assert status == "completed", f"Generator failed with status {status}. Errors: {data.get('errors')}\nFull State: {json.dumps(data.get('workflow_state'), indent=2)}"
    
    workflow_state = data.get("workflow_state", {})
    is_valid = workflow_state.get("is_valid")
    generated_workflow = workflow_state.get("generated_workflow")
    
    if not generated_workflow:
        print(f"DEBUG: Generator State: {json.dumps(workflow_state, indent=2)}")
        
    assert is_valid is True, f"Generator marked workflow as invalid: {workflow_state.get('validation_errors')}"
    
    # Dump generated workflow to file for inspection
    with open("generated_workflow_debug.json", "w") as f:
        json.dump(generated_workflow, f, indent=2)
        
    assert generated_workflow is not None, "Generator did not produce a workflow"
    print(f"[Step 1] Success! Generated nodes: {len(generated_workflow.get('nodes', []))}")

    # 3. Register the generated workflow
    new_workflow_id = "inbox-ingest-generated:v1.0.0"
    print(f"[Step 2] Registering generated workflow as {new_workflow_id}...")
    reg_response = client.post(
        "/workflows",
        json={
            "tenant_id": TENANT_ID,
            "workflow_id": new_workflow_id,
            "name": "Inbox Ingest (Generated)",
            "version": "1.0.0",
            "description": "Generated by E2E test",
            "owner": "test",
            "definition": generated_workflow
        }
    )
    # Check if created or already exists
    assert reg_response.status_code in [201, 200, 409], f"Registration failed: {reg_response.text}"
    print("[Step 2] Success!")

    # 4. Execute the new workflow
    print(f"[Step 3] Executing {new_workflow_id}...")
    test_file_path = f"{VAULT_ROOT}/Persona/00_inbox/_ready/test_generator_real.md"
    
    # Ensure dir exists
    os.makedirs(os.path.dirname(test_file_path), exist_ok=True)
    with open(test_file_path, "w") as f:
        f.write("# Real Server Test\nBy Automation\n\nTesting the full pipeline on real server.")

    try:
        exec_response = client.post(
            "/execute",
            json={
                "tenant_id": TENANT_ID,
                "workflow_id": new_workflow_id,
                "input": {
                    "query": {
                        "file_path": test_file_path,
                        "file_content": "# Real Server Test\nBy Automation\n\nTesting the full pipeline on real server."
                    }
                }
            }
        )
        exec_response.raise_for_status()
        exec_run_id = exec_response.json()["run_id"]
        
        # Poll for completion
        for i in range(30):
            status_resp = client.get(f"/execute/{exec_run_id}/status")
            status_resp.raise_for_status()
            data = status_resp.json()
            status = data["status"]
            print(f"Polling Execution... Status: {status}")
            if status in ["completed", "failed"]:
                break
            time.sleep(2)
            
        assert status == "completed", f"Execution failed: {data.get('errors')}. State: {json.dumps(data.get('workflow_state'), indent=2)}"
        print("[Step 3] Success!")
        
        # 5. Detailed Verification and Logging
        print("\n=== EXECUTION AUDIT LOG ===")
        events = data.get("events", [])
        
        # DEBUG: Print first bash event to see structure
        bash_events = [e for e in events if "tool" in str(e).lower() and "bash" in str(e).lower()]
        if bash_events:
            print(f"DEBUG: Sample Bash Event: {json.dumps(bash_events[0], indent=2)}")

        bash_calls_started = [e for e in events if e.get("type") == "tool.started" and e.get("data", {}).get("tool") == "bash"]
        bash_calls_completed = [e for e in events if e.get("type") == "tool.completed" and e.get("data", {}).get("tool") == "bash"]
        
        node_executions = [e for e in events if e.get("type") == "node.completed"]
        
        for node in node_executions:
            node_id = node.get("data", {}).get("node_id")
            print(f"\n[Node: {node_id}]")
        
        print("\n--- Tool Calls Detail ---")
        for i, start_event in enumerate(bash_calls_started):
            tool_data = start_event.get("data", {})
            # On some servers, arguments might be in 'input' or 'arguments'
            args = tool_data.get('arguments') or tool_data.get('input') or {}
            print(f"\nTool Call #{i+1}: {tool_data.get('tool')}")
            print(f"Command: {args.get('command')}")
            
            # Find corresponding completion
            call_id = tool_data.get('call_id')
            completion = next((e for e in bash_calls_completed if e.get("data", {}).get("call_id") == call_id), None)
            if completion:
                result = completion.get("data", {}).get("result", {})
                print(f"Exit Code: {result.get('exit_code')}")
                if result.get('stdout'):
                    print(f"Stdout: {result.get('stdout')[:200]}...")
                if result.get('stderr'):
                    print(f"Stderr: {result.get('stderr')}")
            else:
                print("Status: Result not found in events")

        print(f"\n[Step 4] Verification: Found {len(bash_calls_started)} bash tool calls.")
        assert len(bash_calls_started) >= 1, "No bash tool calls detected"
        
        print("\nReal Server Full Cycle E2E Test Success!")

    finally:
        if os.path.exists(test_file_path):
            os.remove(test_file_path)

if __name__ == "__main__":
    pytest.main([__file__, "-s"])
