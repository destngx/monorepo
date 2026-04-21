"""
E2E Test Helpers for Graph-Weave

Shared utilities for E2E tests including workflow cleanup and status polling.
"""

import time


def debug_log(step, message, level="INFO"):
    """
    Print structured debug log with timestamp and level.

    Args:
        step: Step identifier (e.g., "CLEANUP", "POLL", "ASSERT")
        message: Log message
        level: Log level (INFO, DEBUG, WARN, ERROR)
    """
    timestamp = time.strftime("%H:%M:%S")
    prefix = f"[{timestamp}] [{level:5s}] [{step:10s}]"
    print(f"{prefix} {message}")


def render_template(template, context):
    result = template
    for key, value in context.items():
        result = result.replace(f"{{{key}}}", str(value))
    return result


def get_node_config(workflow_definition, node_id):
    for node in workflow_definition.get("definition", {}).get("nodes", []):
        if node.get("id") == node_id:
            return node.get("config", {})
    return {}


def log_node_trace(events):
    """Print request/response style logs for each agent node."""
    node_events = [
        event
        for event in events
        if any(
            marker in str(event.get("message", ""))
            or marker in str(event.get("data", {}))
            for marker in ["agent", "node", "tool"]
            for marker in ["agent", "node", "tool"]
        )
    ]

    for event in node_events:
        message = event.get("message") or event.get("type")
        data = event.get("data", {})
        node_id = data.get("node_id", data.get("tool", "unknown"))
        debug_log("NODE_REQ", f"{node_id}: {message}")
        if data:
            debug_log("NODE_RES", f"{node_id}: {data}")


def _execution_input(state):
    if not isinstance(state, dict):
        return {}

    for key in ("input", "inputs", "request", "payload", "execution_input"):
        value = state.get(key)
        if isinstance(value, dict):
            return value

    if "node_results" not in state and "events" not in state:
        return state

    return {}


def log_agent_node_io(node_id, node_config, state):
    """Print the resolved input, tool usage, and output for a single agent node."""
    agent_name = node_config.get("agent_name", node_id)
    user_template = node_config.get("user_prompt_template", "")
    tools = node_config.get("tools", [])
    execution_input = _execution_input(state)
    input_text = render_template(user_template, execution_input)

    print(f"\n[AGENT NODE] {agent_name} ({node_id})")
    print(f"  → input: {input_text}")
    if tools:
        print(f"  → tools: {', '.join(tool.get('name', 'unknown') for tool in tools)}")

    node_results = state.get("node_results", {}) if isinstance(state, dict) else {}
    node_result = (
        node_results.get(node_id, {}) if isinstance(node_results, dict) else {}
    )
    if isinstance(node_result, dict) and node_result:
        tool_calls = node_result.get("tool_calls", [])
        if tool_calls:
            print("  → tool usage:")
            for tool_call in tool_calls:
                tool_name = tool_call.get("name") or tool_call.get("tool") or "unknown"
                tool_args = tool_call.get("args", tool_call.get("arguments", {}))
                print(f"      - {tool_name}: {tool_args}")

        output_value = node_result.get("result", node_result.get("output", node_result))
        print(f"  ← output: {output_value}")
        return

    output_value = state.get(f"{node_id}_output") if isinstance(state, dict) else None
    print(f"  ← output: {output_value if output_value is not None else 'N/A'}")


def log_workflow_agent_io(workflow_definition, state, node_ids=None):
    agent_nodes = [
        node
        for node in workflow_definition.get("definition", {}).get("nodes", [])
        if node.get("type") in ["agent", "agent_node"]
        and (node_ids is None or node.get("id") in set(node_ids))
    ]

    if not agent_nodes:
        debug_log("AGENT", "No agent nodes found for workflow", "WARN")
        return

    print(f"\n[AGENT WORKFLOW] {workflow_definition.get('workflow_id', 'unknown')}")
    for node in agent_nodes:
        log_agent_node_io(node.get("id", "unknown"), node.get("config", {}), state)


def wait_for_terminal_status(client, run_id, timeout=180.0, debug=True):
    """
    Wait for workflow execution to reach terminal status.

    Args:
        client: HTTP client instance for API calls (httpx.Client or TestClient)
        run_id: Workflow run ID to monitor
        timeout: Maximum time to wait in seconds (default: 30.0)
        debug: Enable debug logging (default: True)

    Returns:
        dict: Final status response or last known state

    Raises:
        AssertionError: If status polling fails
    """
    if debug:
        debug_log(
            "POLL", f"Starting status poll for run_id={run_id}, timeout={timeout}s"
        )

    deadline = time.monotonic() + timeout
    last_data = None
    poll_count = 0
    status = "unknown"

    while time.monotonic() < deadline:
        poll_count += 1
        try:
            response = client.get(f"/execute/{run_id}/status")

            if response.status_code != 200:
                debug_log(
                    "POLL",
                    f"Poll #{poll_count}: Status code {response.status_code}",
                    "WARN",
                )
                assert response.status_code == 200

            last_data = response.json()
            status = last_data.get("status")

            if debug:
                debug_log("POLL", f"Poll #{poll_count}: status={status}")

            if status in ["completed", "failed", "cancelled"]:
                if debug:
                    debug_log(
                        "POLL",
                        f"Terminal status reached: {status} (after {poll_count} polls)",
                    )
                return last_data

            time.sleep(1.0)  # Poll more frequently to keep e2e runtime bounded
        except Exception as e:
            debug_log("POLL", f"Poll #{poll_count}: Exception - {str(e)}", "ERROR")
            raise

    if debug:
        debug_log(
            "POLL",
            f"Timeout reached after {poll_count} polls without terminal status",
            "ERROR",
        )
    raise AssertionError(
        f"Workflow run {run_id} failed to reach terminal status within {timeout}s. Last status: {last_data.get('status') if isinstance(last_data, dict) else status}"
    )


def ensure_clean_workflow(client, tenant_id, workflow_id, debug=True):
    """
    Delete workflow if it exists, then prepare for fresh creation.

    This helper ensures test isolation by removing any existing workflow
    before creating a new one. It handles missing workflows gracefully.

    Args:
        client: httpx.Client instance for API calls
        tenant_id: Tenant ID for the workflow
        workflow_id: Workflow ID to clean up
        debug: Enable debug logging (default: True)

    Side Effects:
        Logs cleanup actions to stdout (for test visibility)
    """
    if debug:
        debug_log("CLEANUP", f"Starting cleanup for workflow_id={workflow_id}")

    try:
        if debug:
            debug_log("CLEANUP", f"Sending DELETE request for workflow")

        delete_response = client.delete(
            f"/workflows/{workflow_id}", params={"tenant_id": tenant_id}
        )

        if delete_response.status_code in [200, 204]:
            if debug:
                debug_log("CLEANUP", f"✓ Deleted existing workflow: {workflow_id}")
            print(f"  ✓ Deleted existing workflow: {workflow_id}")
        elif delete_response.status_code == 404:
            if debug:
                debug_log("CLEANUP", f"Workflow not found (404) - this is OK", "INFO")
        else:
            debug_log(
                "CLEANUP",
                f"Unexpected status code: {delete_response.status_code}",
                "WARN",
            )

    except Exception as e:
        debug_log("CLEANUP", f"Exception during cleanup: {str(e)}", "WARN")
        print(f"  ℹ Note: Could not delete existing workflow (may not exist): {e}")

    if debug:
        debug_log("CLEANUP", f"✓ Workflow cleaned up and ready for fresh creation")
    print(f"  ✓ Workflow cleaned up and ready for fresh creation")


def create_workflow_via_api(client, tenant_id, workflow_definition, debug=True):
    """
    Create a workflow definition via API.

    Args:
        client: httpx.Client instance
        tenant_id: Tenant ID
        workflow_definition: Workflow definition dict
        debug: Enable debug logging (default: True)

    Returns:
        dict: API response data

    Raises:
        AssertionError: If creation fails
    """
    if debug:
        workflow_id = workflow_definition.get("workflow_id", "unknown")
        debug_log("SETUP", f"Creating workflow via API: {workflow_id}")

    response = client.post(
        "/workflows",
        json=workflow_definition,
        params={"tenant_id": tenant_id},
    )

    assert response.status_code == 201 or response.status_code == 200, (
        f"Failed to create workflow: {response.status_code} - {response.text}"
    )

    data = response.json()
    if debug:
        debug_log("SETUP", f"✓ Workflow created: {data.get('workflow_id')}")
    print(f"  ✓ Workflow created: {data.get('workflow_id')}")

    return data
