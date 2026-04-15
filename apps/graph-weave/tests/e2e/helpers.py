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


def wait_for_terminal_status(client, run_id, timeout=60.0, debug=True):
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
        f"Workflow run {run_id} failed to reach terminal status within {timeout}s. Last status: {status}"
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

        if delete_response.status_code == 200:
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
