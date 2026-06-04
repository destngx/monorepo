import pytest
from src.adapters.langgraph.runtime.engine.routing import route_by_edge
from src.adapters.langgraph.runtime.base.executor import BaseLangGraphExecutor
from src.adapters.langgraph.runtime.state.resolver import StateResolver

class DummyExecutor(BaseLangGraphExecutor):
    def __init__(self):
        super().__init__()
        self.execution_events = {}
        self.redis_client = None

    def _emit_event(self, run_id: str, event_type: str, data: dict):
        pass


@pytest.fixture
def workflow_spec():
    return {
        "workflow_id": "test-pkm-ingest:v1.0.0",
        "entry_point": "entry",
        "exit_point": "exit",
        "nodes": [
            {"id": "entry", "type": "entry"},
            {"id": "normalize_input", "type": "agent_node"},
            {"id": "fetch_url_content", "type": "agent_node"},
            {"id": "process_media", "type": "cli_node"},
            {"id": "extract_and_format", "type": "agent_node"},
            {"id": "exit", "type": "exit"}
        ],
        "edges": [
            {"from": "entry", "to": "normalize_input"},
            {"from": "normalize_input", "to": "fetch_url_content", "condition": "$.nodes.normalize_input.result.has_urls == true"},
            {"from": "normalize_input", "to": "process_media"}, # fallback if has_urls is false
            {"from": "fetch_url_content", "to": "process_media", "condition": "$.nodes.normalize_input.result.has_media == true"},
            {"from": "fetch_url_content", "to": "extract_and_format"}, # fallback if has_media is false
            {"from": "process_media", "to": "extract_and_format"},
            {"from": "extract_and_format", "to": "exit"}
        ]
    }


def test_routing_scenario_both_skipped(workflow_spec):
    """Scenario 1: No URLs, No Media (Both skipped)"""
    executor = DummyExecutor()
    state = {
        "input": {"file_path": "/notes/test.md", "file_content": "No URL and no media"},
        "nodes": {
            "entry": {"result": {}},
            "normalize_input": {
                "status": "completed",
                "result": {"has_urls": False, "has_media": False, "file_content": "No URL and no media"}
            }
        }
    }
    
    # 1. Routing from normalize_input
    # has_urls == true is False -> should evaluate next edge which points to process_media
    next_node = route_by_edge(executor, "run-1", workflow_spec, "normalize_input", state)
    assert next_node == "process_media"
    
    # 2. Routing from process_media
    # Unconditional or falls through to extract_and_format because it's not meeting the condition to stay or loop
    state["nodes"]["process_media"] = {"status": "skipped", "result": {}}
    next_node = route_by_edge(executor, "run-1", workflow_spec, "process_media", state)
    assert next_node == "extract_and_format"


def test_routing_scenario_only_fetch(workflow_spec):
    """Scenario 2: Has URLs, No Media (Fetch runs, OCR skipped)"""
    executor = DummyExecutor()
    state = {
        "nodes": {
            "normalize_input": {
                "status": "completed",
                "result": {"has_urls": True, "has_media": False}
            }
        }
    }
    
    next_node = route_by_edge(executor, "run-2", workflow_spec, "normalize_input", state)
    assert next_node == "fetch_url_content"
    
    # Once fetch_url_content finishes, we check path to process_media
    state["nodes"]["fetch_url_content"] = {
        "status": "completed",
        "result": {"file_content": "Augmented content with fetched URL"}
    }
    
    next_node = route_by_edge(executor, "run-2", workflow_spec, "fetch_url_content", state)
    assert next_node == "extract_and_format" # has_media is False, so skip process_media


def test_routing_scenario_only_media(workflow_spec):
    """Scenario 3: No URLs, Has Media (Fetch skipped, OCR runs)"""
    executor = DummyExecutor()
    state = {
        "nodes": {
            "normalize_input": {
                "status": "completed",
                "result": {"has_urls": False, "has_media": True}
            }
        }
    }
    
    next_node = route_by_edge(executor, "run-3", workflow_spec, "normalize_input", state)
    assert next_node == "process_media"


def test_routing_scenario_both_run(workflow_spec):
    """Scenario 4: Has URLs, Has Media (Both run)"""
    executor = DummyExecutor()
    state = {
        "nodes": {
            "normalize_input": {
                "status": "completed",
                "result": {"has_urls": True, "has_media": True}
            }
        }
    }
    
    next_node = route_by_edge(executor, "run-4", workflow_spec, "normalize_input", state)
    assert next_node == "fetch_url_content"
    
    state["nodes"]["fetch_url_content"] = {
        "status": "completed",
        "result": {"file_content": "Fetched URL"}
    }
    
    next_node = route_by_edge(executor, "run-4", workflow_spec, "fetch_url_content", state)
    assert next_node == "process_media"


def test_state_resolution_fallback_coalescing():
    """Verify that wildcard resolution resolves from whichever optional node ran successfully"""
    # Case A: Both skipped -> falls back to normalize_input
    state_a = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}}
        }
    }
    resolver_a = StateResolver(state_a)
    assert resolver_a.resolve("$.nodes.*.result.file_content") == "normalized"

    # Case B: Fetch url completed -> fetches winning value
    state_b = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}},
            "fetch_url_content": {"result": {"file_content": "fetched_content"}}
        }
    }
    resolver_b = StateResolver(state_b)
    assert resolver_b.resolve("$.nodes.*.result.file_content") == "fetched_content"

    # Case C: Media runs after fetch -> media content wins
    state_c = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}},
            "fetch_url_content": {"result": {"file_content": "fetched_content"}},
            "process_media": {"result": {"file_content": "media_extracted"}}
        }
    }
    resolver_c = StateResolver(state_c)
    assert resolver_c.resolve("$.nodes.*.result.file_content") == "media_extracted"


def test_file_content_propagation_and_coalescing():
    """Verify that downstream nodes receive the correct, latest file_content input value."""
    # Downstream input mapping format used by extract_and_format node
    input_mapping_spec = [
        "$.nodes.process_media.result.file_content",
        "$.nodes.fetch_url_content.result.file_content",
        "$.nodes.normalize_input.result.file_content",
        "$.input.file_content"
    ]

    # Scenario 1: Both fetch and media skipped.
    # extract_and_format should get "normalized" content.
    state_1 = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}}
        }
    }
    resolver_1 = StateResolver(state_1)
    resolved_1 = resolver_1._resolve_mapping_value(input_mapping_spec)
    assert resolved_1 == "normalized"

    # Scenario 2: fetch URL executes but media skipped.
    # extract_and_format should get "fetched_content".
    state_2 = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}},
            "fetch_url_content": {"result": {"file_content": "fetched_content"}}
        }
    }
    resolver_2 = StateResolver(state_2)
    resolved_2 = resolver_2._resolve_mapping_value(input_mapping_spec)
    assert resolved_2 == "fetched_content"

    # Scenario 3: both fetch URL and media execute.
    # extract_and_format should get "media_extracted".
    state_3 = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}},
            "fetch_url_content": {"result": {"file_content": "fetched_content"}},
            "process_media": {"result": {"file_content": "media_extracted"}}
        }
    }
    resolver_3 = StateResolver(state_3)
    resolved_3 = resolver_3._resolve_mapping_value(input_mapping_spec)
    assert resolved_3 == "media_extracted"

    # Scenario 4: fetch URL skipped, but media executes.
    # extract_and_format should get "media_extracted".
    state_4 = {
        "input": {"file_content": "original"},
        "nodes": {
            "normalize_input": {"result": {"file_content": "normalized"}},
            "process_media": {"result": {"file_content": "media_extracted"}}
        }
    }
    resolver_4 = StateResolver(state_4)
    resolved_4 = resolver_4._resolve_mapping_value(input_mapping_spec)
    assert resolved_4 == "media_extracted"


