import pytest
from src.adapters.langgraph.runtime.state.resolver import (
    StateResolver,
    MissingStatePathError,
    InvalidStatePathError,
)
from src.adapters.langgraph.runtime.base.state_utils import get_state_value


@pytest.fixture
def sample_state():
    return {
        "input": {
            "file_path": "/Users/test/file name.txt",  # Space triggers shell quoting
            "file_content": "Hello world from input!",
            "nested": {
                "deep": "deep value"
            }
        },
        "workflow": {
            "shared_config": "config_val"
        },
        "nodes": {
            "entry": {
                "result": {
                    "file_path": "/Users/test/file name.txt"
                }
            },
            "normalize_input": {
                "result": {
                    "file_content": "Normalized text content",
                    "parsed_data_json": '{"parsed": "json content"}',  # Only _json key exists
                    "tags": ["wealth", "market", "data"]
                }
            },
            "fetch_url_content": {
                "result": {
                    "file_content": "Extracted website content"
                }
            }
        }
    }


class TestStateResolverUnified:
    def test_canonical_paths(self, sample_state):
        resolver = StateResolver(sample_state)
        # Canonical input path
        assert resolver.resolve("$.input.file_path") == "/Users/test/file name.txt"
        # Canonical node result path
        assert resolver.resolve("$.nodes.normalize_input.result.file_content") == "Normalized text content"

    def test_lenient_bare_paths(self, sample_state):
        resolver = StateResolver(sample_state)
        # Accepts path without $ prefix
        assert resolver.resolve("input.file_path") == "/Users/test/file name.txt"
        assert resolver.resolve("nodes.normalize_input.result.file_content") == "Normalized text content"

    def test_llm_entry_input_schema_normalization(self, sample_state):
        resolver = StateResolver(sample_state)
        # Normalizes entry.input_schema -> input
        assert resolver.resolve("$entry.input_schema.file_path") == "/Users/test/file name.txt"
        assert resolver.resolve("entry.input_schema.nested.deep") == "deep value"

    def test_llm_output_schema_normalization(self, sample_state):
        resolver = StateResolver(sample_state)
        # Normalizes $<node_id>.output_schema.<field> -> $.nodes.<node_id>.result.<field>
        assert resolver.resolve("$normalize_input.output_schema.file_content") == "Normalized text content"
        assert resolver.resolve("fetch_url_content.output_schema.file_content") == "Extracted website content"

    def test_llm_input_schema_normalization(self, sample_state):
        resolver = StateResolver(sample_state)
        # Normalizes $<node_id>.input_schema.<field> -> $.nodes.<node_id>.result.<field>
        assert resolver.resolve("$normalize_input.input_schema.file_content") == "Normalized text content"

    def test_lenient_namespace_node_lookup_fallback(self, sample_state):
        resolver = StateResolver(sample_state)
        # Bare node lookup - searches in nodes state first
        assert resolver.resolve("$normalize_input.result.file_content") == "Normalized text content"
        assert resolver.resolve("normalize_input.result.file_content") == "Normalized text content"

    def test_json_key_variants(self, sample_state):
        resolver = StateResolver(sample_state)
        # Automatically parses json string if root key_json exists and we request root key (parsed_data)
        assert resolver.resolve("normalize_input.result.parsed_data.parsed") == "json content"

    def test_inline_virtual_transforms(self, sample_state):
        resolver = StateResolver(sample_state)
        # Test _joined transform
        assert resolver.resolve("normalize_input.result.tags_joined") == "wealth, market, data"
        # Test _first transform
        assert resolver.resolve("normalize_input.result.tags_first") == "wealth"
        # Test sh_quote transform
        assert resolver.resolve("input.file_path_shell") == "'/Users/test/file name.txt'"

    def test_get_state_value_wrapper(self, sample_state):
        # Verify that get_state_value wraps StateResolver correctly
        assert get_state_value("$.input.file_path", sample_state) == "/Users/test/file name.txt"
        assert get_state_value("$normalize_input.output_schema.file_content", sample_state) == "Normalized text content"
        assert get_state_value("normalize_input.result.tags_first", sample_state) == "wealth"
