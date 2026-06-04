"""
Unit tests for wildcard (*) state path resolution in StateResolver.

Covers:
- Basic wildcard resolution across multiple executed nodes
- Reverse chronological precedence (latest node wins)
- Fallback to $.input when no nodes produce the field
- Lenient None return when all fallback paths miss
- Integration with exit-style output_mapping using wildcard + input fallback
"""

import pytest
from src.adapters.langgraph.runtime.state.resolver import (
    StateResolver,
    MissingStatePathError,
)


# ── Helpers ──────────────────────────────────────────────────────────────

def _make_state(nodes: dict, input_data: dict | None = None) -> dict:
    """Build a minimal executor state dict with nodes and optional input."""
    state: dict = {"nodes": nodes}
    if input_data is not None:
        state["input"] = input_data
    return state


# ── Wildcard Resolution ─────────────────────────────────────────────────

class TestWildcardResolution:

    def test_wildcard_returns_value_from_single_node(self):
        state = _make_state({
            "normalize_input": {"result": {"file_content": "hello"}}
        })
        resolver = StateResolver(state)
        assert resolver.resolve("$.nodes.*.result.file_content") == "hello"

    def test_wildcard_returns_latest_node_value(self):
        """When multiple nodes produce the same field, the last inserted (latest) wins."""
        # Python dicts preserve insertion order; latest key is last
        state = _make_state({
            "normalize_input": {"result": {"file_content": "original"}},
            "fetch_url_content": {"result": {"file_content": "enriched"}},
        })
        resolver = StateResolver(state)
        # Reverse iteration means fetch_url_content (last inserted) is checked first
        assert resolver.resolve("$.nodes.*.result.file_content") == "enriched"

    def test_wildcard_skips_nodes_without_field(self):
        """If a later node doesn't have the field, fall back to an earlier one."""
        state = _make_state({
            "normalize_input": {"result": {"file_content": "from_normalizer"}},
            "extract_and_format": {"result": {"sources_markdown": "## Sources"}},
        })
        resolver = StateResolver(state)
        assert resolver.resolve("$.nodes.*.result.file_content") == "from_normalizer"

    def test_wildcard_returns_none_when_no_match(self):
        state = _make_state({
            "normalize_input": {"result": {"other_field": "x"}},
        })
        resolver = StateResolver(state)
        assert resolver.resolve("$.nodes.*.result.file_content", required=False) is None

    def test_wildcard_raises_when_required_and_no_match(self):
        state = _make_state({
            "normalize_input": {"result": {"other_field": "x"}},
        })
        resolver = StateResolver(state)
        with pytest.raises(MissingStatePathError):
            resolver.resolve("$.nodes.*.result.file_content", required=True)

    def test_wildcard_with_empty_nodes(self):
        state = _make_state({})
        resolver = StateResolver(state)
        assert resolver.resolve("$.nodes.*.result.file_content", required=False) is None


# ── Lenient List Fallback ────────────────────────────────────────────────

class TestLenientListFallback:

    def test_list_fallback_returns_first_non_null(self):
        state = _make_state(
            {"node_a": {"result": {"val": "from_a"}}},
            input_data={"val": "from_input"},
        )
        resolver = StateResolver(state)
        result = resolver._resolve_mapping_value([
            "$.nodes.node_a.result.val",
            "$.input.val",
        ])
        assert result == "from_a"

    def test_list_fallback_skips_missing_to_input(self):
        state = _make_state(
            {},  # no nodes executed
            input_data={"val": "from_input"},
        )
        resolver = StateResolver(state)
        result = resolver._resolve_mapping_value([
            "$.nodes.*.result.val",
            "$.input.val",
        ])
        assert result == "from_input"

    def test_list_fallback_returns_none_when_all_miss(self):
        """Critical fix: must NOT raise MissingStatePathError here."""
        state = _make_state({}, input_data={})
        resolver = StateResolver(state)
        result = resolver._resolve_mapping_value([
            "$.nodes.*.result.val",
            "$.input.val",
        ])
        assert result is None

    def test_list_fallback_wildcard_then_specific_then_input(self):
        state = _make_state(
            {"process_media": {"result": {"file_content": "ocr_output"}}},
            input_data={"file_content": "raw_input"},
        )
        resolver = StateResolver(state)
        result = resolver._resolve_mapping_value([
            "$.nodes.*.result.file_content",
            "$.input.file_content",
        ])
        assert result == "ocr_output"


# ── Exit Mapping Integration ────────────────────────────────────────────

class TestExitMappingIntegration:
    """Simulate how handle_exit_node calls resolve_mapping with wildcard mappings."""

    def test_exit_mapping_resolves_with_all_nodes_executed(self):
        state = _make_state(
            {
                "normalize_input": {"result": {"file_content": "normalized"}},
                "fetch_url_content": {"result": {"file_content": "fetched"}},
                "process_media": {
                    "result": {"file_content": "ocr"},
                    "outputs": {"stdout": "ok", "stderr": "", "exit_code": 0},
                },
                "extract_and_format": {
                    "result": {
                        "sources_markdown": "## Sources",
                        "drafts_markdown": "## Drafts",
                    }
                },
            },
            input_data={"file_content": "raw_input", "file_path": "/notes/test.md"},
        )

        mapping = {
            "file_content": ["$.nodes.*.result.file_content", "$.input.file_content"],
            "sources_markdown": ["$.nodes.*.result.sources_markdown", "$.input.sources_markdown"],
            "drafts_markdown": ["$.nodes.*.result.drafts_markdown", "$.input.drafts_markdown"],
            "stdout": "$.nodes.*.outputs.stdout",
            "stderr": "$.nodes.*.outputs.stderr",
            "exit_code": "$.nodes.*.outputs.exit_code",
        }

        resolver = StateResolver(state)
        result = resolver.resolve_mapping(mapping)

        # Latest node producing each field wins
        assert result["sources_markdown"] == "## Sources"
        assert result["drafts_markdown"] == "## Drafts"
        # extract_and_format is last and has file_content? No — it doesn't.
        # process_media is last node with file_content → "ocr" should be returned
        # Wait: dict iteration in reverse. extract_and_format is last in insertion order.
        # extract_and_format.result has no file_content → skip → process_media has it → "ocr"
        assert result["file_content"] == "ocr"
        assert result["stdout"] == "ok"
        assert result["stderr"] == ""
        assert result["exit_code"] == 0

    def test_exit_mapping_with_optional_nodes_skipped(self):
        """When fetch_url_content and process_media are skipped (conditional edges),
        the exit mapping should still resolve correctly without crashing."""
        state = _make_state(
            {
                "normalize_input": {"result": {"file_content": "normalized"}},
                # fetch_url_content: SKIPPED
                # process_media: SKIPPED
                "extract_and_format": {
                    "result": {
                        "sources_markdown": "## Sources",
                        "drafts_markdown": "## Drafts",
                    }
                },
            },
            input_data={"file_content": "raw_input", "file_path": "/notes/test.md"},
        )

        mapping = {
            "file_content": ["$.nodes.*.result.file_content", "$.input.file_content"],
            "sources_markdown": ["$.nodes.*.result.sources_markdown", "$.input.sources_markdown"],
            "drafts_markdown": ["$.nodes.*.result.drafts_markdown", "$.input.drafts_markdown"],
            "stdout": "$.nodes.*.outputs.stdout",
            "stderr": "$.nodes.*.outputs.stderr",
            "exit_code": "$.nodes.*.outputs.exit_code",
        }

        resolver = StateResolver(state)
        result = resolver.resolve_mapping(mapping)

        assert result["file_content"] == "normalized"
        assert result["sources_markdown"] == "## Sources"
        assert result["drafts_markdown"] == "## Drafts"
        # No CLI nodes executed, so stdout/stderr/exit_code resolve to None
        assert result["stdout"] is None
        assert result["stderr"] is None
        assert result["exit_code"] is None

    def test_exit_mapping_all_optional_skipped_falls_to_input(self):
        """When ALL intermediate nodes are skipped, the exit mapping should
        fall back to $.input fields."""
        state = _make_state(
            {},  # No nodes executed at all
            input_data={
                "file_content": "raw_input",
                "sources_markdown": "input_sources",
            },
        )

        mapping = {
            "file_content": ["$.nodes.*.result.file_content", "$.input.file_content"],
            "sources_markdown": ["$.nodes.*.result.sources_markdown", "$.input.sources_markdown"],
        }

        resolver = StateResolver(state)
        result = resolver.resolve_mapping(mapping)

        assert result["file_content"] == "raw_input"
        assert result["sources_markdown"] == "input_sources"
