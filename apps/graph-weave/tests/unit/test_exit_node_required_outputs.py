import pytest

from src.adapters.langgraph.runtime.engine.handlers import handle_exit_node


class DummyExecutor:
    def __init__(self):
        self.execution_events = {}
        self.redis_client = None

    def _get_state_value(self, path, state):
        if path == "$.draft_paths":
            return state["workflow_state"].get("draft_paths")
        if path == "$.status":
            return state["workflow_state"].get("status")
        return None


def test_exit_node_fails_when_required_output_is_missing():
    executor = DummyExecutor()
    state = {
        "workflow_state": {
            "status": "success",
            "draft_paths": None,
        },
        # Add root-level status so $.status resolves but $.draft_paths stays None
        "status": "success",
    }
    node = {
        "id": "exit",
        "type": "exit",
        "config": {
            "output_mapping": {
                "status": "$.status",
                "draft_paths": "$.draft_paths",
            },
            "required_outputs": ["status", "draft_paths"],
        },
    }

    with pytest.raises(ValueError) as exc:
        handle_exit_node(executor, "run-1", "workflow-1", node, state)

    assert "Exit node missing required outputs: draft_paths" in str(exc.value)


def test_exit_node_accepts_required_outputs_when_present():
    executor = DummyExecutor()
    state = {
        "workflow_state": {
            "status": "success",
            "draft_paths": ["/draft.md"],
        }
    }
    node = {
        "id": "exit",
        "type": "exit",
        "config": {
            "output_mapping": {
                "status": "$.status",
                "draft_paths": "$.draft_paths",
            },
            "required_outputs": ["status", "draft_paths"],
        },
    }

    handle_exit_node(executor, "run-1", "workflow-1", node, state)

    assert state["status"] == "completed"
    assert state["workflow_state"]["draft_paths"] == ["/draft.md"]
