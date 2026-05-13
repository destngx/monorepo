from src.adapters.mcp import ToolRegistry
from src.services.status_service import StatusService
from src.models import StatusEnum


def test_status_service_records_transition_payload():
    service = StatusService()
    service.set_status("tenant-1", "run-1", StatusEnum.queued)

    record = service.transition_status(
        "tenant-1",
        "run-1",
        StatusEnum.running,
        {"events": [{"type": "status.changed"}]},
    )

    assert record["status"] == "running"
    assert record["events"] == [{"type": "status.changed"}]


def test_registry_search_tool_remains_deterministic():
    registry = ToolRegistry()
    result = registry.call_tool("search", {"query": "graph weave"})

    assert result == {"results": ["Result for: graph weave"]}
