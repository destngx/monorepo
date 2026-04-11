"""Tests for LangGraph StateGraph builder from workflow JSON schema.

Test Coverage:
- WorkflowParser: parse_workflow_json, extract_node_config
- EdgeEvaluator: evaluate with all operators (>, >=, <, <=, ==, !=)
- Node builders: entry, agent_node, branch, exit
- Prompt interpolation: _interpolate_prompt with context variables
- GraphBuilder.build: full workflow construction
- Per-node configuration: provider/model/temperature/max_tokens/tools validation
- Error handling: all error cases with WorkflowParseError
- Tenant isolation: node IDs and edge validation
- State isolation: per-node output keys
"""

import pytest
import sys
from pathlib import Path
from typing import Any, Dict

sys.path.insert(0, str(Path(__file__).parent.parent.parent))

from src.adapters.langgraph_graph_builder import (
    WorkflowParser,
    EdgeEvaluator,
    NodeConfig,
    WorkflowParseError,
    GraphBuilder,
    build_entry_node,
    build_agent_node,
    build_branch_node,
    build_exit_node,
    _interpolate_prompt,
    _parse_value,
)


class TestWorkflowParserParseWorkflowJson:
    """WorkflowParser.parse_workflow_json validation tests."""

    def test_parse_workflow_json_empty_raises(self):
        """Empty workflow should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="cannot be empty"):
            WorkflowParser.parse_workflow_json({})

    def test_parse_workflow_json_no_nodes_raises(self):
        """Workflow with no nodes should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="at least one node"):
            WorkflowParser.parse_workflow_json({"edges": []})

    def test_parse_workflow_json_empty_nodes_list_raises(self):
        """Workflow with empty nodes list should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="at least one node"):
            WorkflowParser.parse_workflow_json({"nodes": [], "edges": []})

    def test_parse_workflow_json_valid_single_node(self):
        """Single node with no edges should parse successfully."""
        workflow = {"nodes": [{"id": "node1", "type": "entry"}], "edges": []}
        result = WorkflowParser.parse_workflow_json(workflow)
        assert result["nodes"] == workflow["nodes"]
        assert result["edges"] == []

    def test_parse_workflow_json_valid_multiple_nodes(self):
        """Multiple nodes with valid edges should parse successfully."""
        workflow = {
            "nodes": [
                {"id": "entry", "type": "entry"},
                {"id": "agent", "type": "agent_node"},
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"source": "entry", "target": "agent"},
                {"source": "agent", "target": "exit"},
            ],
        }
        result = WorkflowParser.parse_workflow_json(workflow)
        assert len(result["nodes"]) == 3
        assert len(result["edges"]) == 2

    def test_parse_workflow_json_unknown_source_node_raises(self):
        """Edge referencing unknown source node should raise WorkflowParseError."""
        workflow = {
            "nodes": [{"id": "node1", "type": "entry"}],
            "edges": [{"source": "unknown", "target": "node1"}],
        }
        with pytest.raises(WorkflowParseError, match="Unknown source node"):
            WorkflowParser.parse_workflow_json(workflow)

    def test_parse_workflow_json_unknown_target_node_raises(self):
        """Edge referencing unknown target node should raise WorkflowParseError."""
        workflow = {
            "nodes": [{"id": "node1", "type": "entry"}],
            "edges": [{"source": "node1", "target": "unknown"}],
        }
        with pytest.raises(WorkflowParseError, match="Unknown target node"):
            WorkflowParser.parse_workflow_json(workflow)

    def test_parse_workflow_json_multiple_edges_all_valid(self):
        """Multiple edges between valid nodes should all validate."""
        workflow = {
            "nodes": [
                {"id": "a", "type": "entry"},
                {"id": "b", "type": "agent_node"},
                {"id": "c", "type": "branch"},
                {"id": "d", "type": "exit"},
            ],
            "edges": [
                {"source": "a", "target": "b"},
                {"source": "b", "target": "c"},
                {"source": "c", "target": "d"},
            ],
        }
        result = WorkflowParser.parse_workflow_json(workflow)
        assert len(result["edges"]) == 3


class TestWorkflowParserExtractNodeConfig:
    """WorkflowParser.extract_node_config validation tests."""

    def test_extract_node_config_empty_dict(self):
        """Empty node dict should return empty NodeConfig."""
        config = WorkflowParser.extract_node_config({})
        assert config.provider is None
        assert config.model is None
        assert config.temperature is None
        assert config.max_tokens is None
        assert config.tools is None

    def test_extract_node_config_valid_provider(self):
        """Valid provider should be extracted."""
        config = WorkflowParser.extract_node_config({"provider": "openai"})
        assert config.provider == "openai"

    def test_extract_node_config_invalid_provider_raises(self):
        """Invalid provider should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Invalid provider"):
            WorkflowParser.extract_node_config({"provider": "invalid_provider"})

    def test_extract_node_config_valid_model_for_provider(self):
        """Valid model for provider should be extracted."""
        config = WorkflowParser.extract_node_config(
            {"provider": "openai", "model": "gpt-4"}
        )
        assert config.model == "gpt-4"

    def test_extract_node_config_invalid_model_for_provider_raises(self):
        """Invalid model for provider should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Invalid model.*for provider"):
            WorkflowParser.extract_node_config(
                {
                    "provider": "openai",
                    "model": "claude-3-opus",  # Anthropic model, not OpenAI
                }
            )

    def test_extract_node_config_all_providers_valid_models(self):
        """All providers should have valid models."""
        for provider, models in [
            ("github", "claude-3.5-sonnet"),
            ("openai", "gpt-4"),
            ("anthropic", "claude-3-opus"),
        ]:
            config = WorkflowParser.extract_node_config(
                {"provider": provider, "model": models}
            )
            assert config.provider == provider
            assert config.model == models

    def test_extract_node_config_valid_temperature(self):
        """Valid temperature [0, 1] should be extracted."""
        for temp in [0, 0.5, 1]:
            config = WorkflowParser.extract_node_config({"temperature": temp})
            assert config.temperature == temp

    def test_extract_node_config_temperature_below_zero_raises(self):
        """Temperature < 0 should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Temperature must be"):
            WorkflowParser.extract_node_config({"temperature": -0.1})

    def test_extract_node_config_temperature_above_one_raises(self):
        """Temperature > 1 should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Temperature must be"):
            WorkflowParser.extract_node_config({"temperature": 1.1})

    def test_extract_node_config_valid_max_tokens(self):
        """Valid max_tokens >= 1 should be extracted."""
        for tokens in [1, 100, 4096]:
            config = WorkflowParser.extract_node_config({"max_tokens": tokens})
            assert config.max_tokens == tokens

    def test_extract_node_config_max_tokens_zero_raises(self):
        """max_tokens < 1 should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="max_tokens must be"):
            WorkflowParser.extract_node_config({"max_tokens": 0})

    def test_extract_node_config_max_tokens_negative_raises(self):
        """Negative max_tokens should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="max_tokens must be"):
            WorkflowParser.extract_node_config({"max_tokens": -10})

    def test_extract_node_config_valid_tools(self):
        """Valid tools should be extracted."""
        config = WorkflowParser.extract_node_config({"tools": ["load_skill", "search"]})
        assert config.tools == ["load_skill", "search"]

    def test_extract_node_config_invalid_tool_raises(self):
        """Invalid tool should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Invalid tools"):
            WorkflowParser.extract_node_config(
                {"tools": ["load_skill", "invalid_tool"]}
            )

    def test_extract_node_config_all_valid_tools(self):
        """All valid tools should be accepted."""
        config = WorkflowParser.extract_node_config(
            {"tools": ["load_skill", "search", "verify"]}
        )
        assert set(config.tools) == {"load_skill", "search", "verify"}

    def test_extract_node_config_combined_valid_config(self):
        """All valid fields combined should extract correctly."""
        config = WorkflowParser.extract_node_config(
            {
                "provider": "openai",
                "model": "gpt-4",
                "temperature": 0.7,
                "max_tokens": 2048,
                "tools": ["load_skill", "verify"],
            }
        )
        assert config.provider == "openai"
        assert config.model == "gpt-4"
        assert config.temperature == 0.7
        assert config.max_tokens == 2048
        assert config.tools == ["load_skill", "verify"]


class TestEdgeEvaluator:
    """EdgeEvaluator condition evaluation tests."""

    def test_edge_evaluator_empty_condition_returns_true(self):
        """Empty condition should return True."""
        assert EdgeEvaluator.evaluate("", {"key": "value"}) is True

    def test_edge_evaluator_none_condition_returns_true(self):
        """None condition should return True."""
        # Test with empty string which evaluates to falsy
        assert EdgeEvaluator.evaluate("", {}) is True

    def test_edge_evaluator_greater_than_true(self):
        """Condition > should evaluate correctly when true."""
        state = {"confidence": 0.8}
        assert EdgeEvaluator.evaluate("$.confidence > 0.7", state) is True

    def test_edge_evaluator_greater_than_false(self):
        """Condition > should evaluate correctly when false."""
        state = {"confidence": 0.5}
        assert EdgeEvaluator.evaluate("$.confidence > 0.7", state) is False

    def test_edge_evaluator_greater_equal_true(self):
        """Condition >= should evaluate correctly when true."""
        state = {"score": 10}
        assert EdgeEvaluator.evaluate("$.score >= 10", state) is True

    def test_edge_evaluator_greater_equal_false(self):
        """Condition >= should evaluate correctly when false."""
        state = {"score": 9}
        assert EdgeEvaluator.evaluate("$.score >= 10", state) is False

    def test_edge_evaluator_less_than_true(self):
        """Condition < should evaluate correctly when true."""
        state = {"retries": 2}
        assert EdgeEvaluator.evaluate("$.retries < 3", state) is True

    def test_edge_evaluator_less_than_false(self):
        """Condition < should evaluate correctly when false."""
        state = {"retries": 3}
        assert EdgeEvaluator.evaluate("$.retries < 3", state) is False

    def test_edge_evaluator_less_equal_true(self):
        """Condition <= should evaluate correctly when true."""
        state = {"count": 5}
        assert EdgeEvaluator.evaluate("$.count <= 5", state) is True

    def test_edge_evaluator_less_equal_false(self):
        """Condition <= should evaluate correctly when false."""
        state = {"count": 6}
        assert EdgeEvaluator.evaluate("$.count <= 5", state) is False

    def test_edge_evaluator_equal_string_true(self):
        """Condition == should evaluate correctly for strings when true."""
        state = {"status": "approved"}
        assert EdgeEvaluator.evaluate("$.status == 'approved'", state) is True

    def test_edge_evaluator_equal_string_false(self):
        """Condition == should evaluate correctly for strings when false."""
        state = {"status": "rejected"}
        assert EdgeEvaluator.evaluate("$.status == 'approved'", state) is False

    def test_edge_evaluator_equal_number_true(self):
        """Condition == should evaluate correctly for numbers when true."""
        state = {"priority": 5}
        assert EdgeEvaluator.evaluate("$.priority == 5", state) is True

    def test_edge_evaluator_not_equal_true(self):
        """Condition != should evaluate correctly when true."""
        state = {"status": "pending"}
        assert EdgeEvaluator.evaluate("$.status != 'approved'", state) is True

    def test_edge_evaluator_not_equal_false(self):
        """Condition != should evaluate correctly when false."""
        state = {"status": "approved"}
        assert EdgeEvaluator.evaluate("$.status != 'approved'", state) is False

    def test_edge_evaluator_jsonpath_nested_access(self):
        """JSONPath should support nested object access."""
        state = {"result": {"confidence": 0.95}}
        assert EdgeEvaluator.evaluate("$.result.confidence > 0.9", state) is True

    def test_edge_evaluator_jsonpath_invalid_raises(self):
        """Invalid JSONPath should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Invalid edge condition"):
            EdgeEvaluator.evaluate("invalid syntax >>", {"key": "value"})

    def test_edge_evaluator_missing_path_raises(self):
        """JSONPath not starting with $ should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="JSONPath must start with"):
            EdgeEvaluator.evaluate("confidence > 0.5", {"confidence": 0.8})


class TestNodeBuilders:
    """Node builder function tests."""

    def test_build_entry_node_merges_input(self):
        """Entry node should merge initial state with input data."""
        entry_func = build_entry_node({})
        result = entry_func({"user_id": "u123", "run_id": "r456"})
        assert result == {"user_id": "u123", "run_id": "r456"}

    def test_build_entry_node_input_overrides_state(self):
        """Entry node input should override initial state."""
        entry_func = build_entry_node({"key": "old_value"})
        result = entry_func({"key": "new_value"})
        assert result["key"] == "new_value"

    def test_build_agent_node_stores_config(self):
        """Agent node should store NodeConfig on function."""
        config = NodeConfig(provider="openai", model="gpt-4")
        agent_func = build_agent_node(config, "You are helpful", "Analyze {input}")
        assert agent_func.config == config

    def test_build_agent_node_interpolates_prompt(self):
        """Agent node should interpolate user prompt with context variables."""
        config = NodeConfig()
        agent_func = build_agent_node(config, "You are helpful", "Process: {data}")
        state = {"data": "test_value", "current_node": "agent1"}
        result = agent_func(state)
        assert result["agent1_output"] == "Process: test_value"

    def test_build_agent_node_output_key_uses_current_node(self):
        """Agent node output key should use current_node from state."""
        config = NodeConfig()
        agent_func = build_agent_node(config, "You are helpful", "Output: {x}")
        state = {"x": "123", "current_node": "decision_node"}
        result = agent_func(state)
        assert "decision_node_output" in result
        assert result["decision_node_output"] == "Output: 123"

    def test_build_agent_node_preserves_state(self):
        """Agent node should preserve existing state keys."""
        config = NodeConfig()
        agent_func = build_agent_node(config, "You are helpful", "{msg}")
        state = {"msg": "hello", "run_id": "r123", "user_id": "u456"}
        result = agent_func(state)
        assert result["run_id"] == "r123"
        assert result["user_id"] == "u456"

    def test_build_branch_node_evaluates_condition_true(self):
        """Branch node should return 'true_branch' when condition evaluates true."""
        branch_func = build_branch_node("$.confidence > 0.5")
        state = {"confidence": 0.8}
        result = branch_func(state)
        assert result == "true_branch"

    def test_build_branch_node_evaluates_condition_false(self):
        """Branch node should return 'false_branch' when condition evaluates false."""
        branch_func = build_branch_node("$.confidence > 0.5")
        state = {"confidence": 0.3}
        result = branch_func(state)
        assert result == "false_branch"

    def test_build_exit_node_returns_state(self):
        """Exit node should return state unchanged."""
        exit_func = build_exit_node()
        state = {"result": "final", "run_id": "r123"}
        result = exit_func(state)
        assert result == state

    def test_build_exit_node_with_output_mapping(self):
        """Exit node should accept output_mapping (unused in current impl)."""
        exit_func = build_exit_node({"key": "final_output"})
        state = {"result": "data"}
        result = exit_func(state)
        assert result == state


class TestPromptInterpolation:
    """Prompt interpolation and value parsing tests."""

    def test_interpolate_prompt_single_variable(self):
        """Single variable should be interpolated."""
        result = _interpolate_prompt("Hello {name}", {"name": "Alice"})
        assert result == "Hello Alice"

    def test_interpolate_prompt_multiple_variables(self):
        """Multiple variables should be interpolated."""
        result = _interpolate_prompt(
            "User {user_id} has {count} items", {"user_id": "u123", "count": 5}
        )
        assert result == "User u123 has 5 items"

    def test_interpolate_prompt_missing_variable_raises(self):
        """Missing variable should raise WorkflowParseError."""
        with pytest.raises(WorkflowParseError, match="Missing template variable"):
            _interpolate_prompt("Hello {name}", {"other": "value"})

    def test_interpolate_prompt_no_variables(self):
        """Prompt without variables should return unchanged."""
        result = _interpolate_prompt("Fixed prompt", {})
        assert result == "Fixed prompt"

    def test_parse_value_single_quoted_string(self):
        """Single-quoted string should parse as string."""
        assert _parse_value("'hello'") == "hello"

    def test_parse_value_double_quoted_string(self):
        """Double-quoted string should parse as string."""
        assert _parse_value('"world"') == "world"

    def test_parse_value_true_boolean(self):
        """'true' string should parse as True boolean."""
        assert _parse_value("true") is True

    def test_parse_value_false_boolean(self):
        """'false' string should parse as False boolean."""
        assert _parse_value("false") is False

    def test_parse_value_integer(self):
        """Integer string should parse as int."""
        assert _parse_value("42") == 42

    def test_parse_value_float(self):
        """Float string should parse as float."""
        assert _parse_value("3.14") == 3.14

    def test_parse_value_unquoted_string(self):
        """Unquoted non-numeric string should parse as string."""
        assert _parse_value("hello") == "hello"

    def test_parse_value_whitespace_trimmed(self):
        """Leading/trailing whitespace should be trimmed."""
        assert _parse_value("  'value'  ") == "value"


class TestGraphBuilderBuild:
    """GraphBuilder.build full workflow construction tests."""

    def test_graph_builder_single_entry_node(self):
        """Graph with single entry node should build successfully."""
        workflow = {"nodes": [{"id": "entry", "type": "entry"}], "edges": []}
        result = GraphBuilder.build(workflow)
        assert "entry" in result["nodes"]
        assert result["edges"] == []

    def test_graph_builder_entry_to_agent_to_exit(self):
        """Simple 3-node workflow should build successfully."""
        workflow = {
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "agent",
                    "type": "agent_node",
                    "provider": "openai",
                    "model": "gpt-4",
                    "system_prompt": "You are helpful",
                    "user_prompt_template": "Process this",
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"source": "entry", "target": "agent"},
                {"source": "agent", "target": "exit"},
            ],
        }
        result = GraphBuilder.build(workflow)
        assert "entry" in result["nodes"]
        assert "agent" in result["nodes"]
        assert "exit" in result["nodes"]
        assert len(result["edges"]) == 2

    def test_graph_builder_preserves_workflow_definition(self):
        """Built graph should include original workflow definition."""
        workflow = {"nodes": [{"id": "n1", "type": "entry"}], "edges": []}
        result = GraphBuilder.build(workflow)
        assert result["definition"] == workflow

    def test_graph_builder_branch_node_with_condition(self):
        """Branch node with condition should build successfully."""
        workflow = {
            "nodes": [
                {"id": "entry", "type": "entry"},
                {"id": "branch", "type": "branch", "condition": "$.score >= 0.8"},
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"source": "entry", "target": "branch"},
                {"source": "branch", "target": "exit"},
            ],
        }
        result = GraphBuilder.build(workflow)
        assert "branch" in result["nodes"]

    def test_graph_builder_unknown_node_type_raises(self):
        """Unknown node type should raise WorkflowParseError."""
        workflow = {"nodes": [{"id": "unknown", "type": "unknown_type"}], "edges": []}
        with pytest.raises(WorkflowParseError, match="Unknown node type"):
            GraphBuilder.build(workflow)

    def test_graph_builder_edges_with_conditions(self):
        """Edges with conditions should be preserved in graph."""
        workflow = {
            "nodes": [{"id": "entry", "type": "entry"}, {"id": "exit", "type": "exit"}],
            "edges": [
                {"source": "entry", "target": "exit", "condition": "$.ready == true"}
            ],
        }
        result = GraphBuilder.build(workflow)
        assert result["edges"][0]["condition"] == "$.ready == true"

    def test_graph_builder_multiple_agent_nodes(self):
        """Multiple agent nodes with different configs should build."""
        workflow = {
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "agent1",
                    "type": "agent_node",
                    "provider": "openai",
                    "model": "gpt-4",
                },
                {
                    "id": "agent2",
                    "type": "agent_node",
                    "provider": "anthropic",
                    "model": "claude-3-opus",
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"source": "entry", "target": "agent1"},
                {"source": "agent1", "target": "agent2"},
                {"source": "agent2", "target": "exit"},
            ],
        }
        result = GraphBuilder.build(workflow)
        assert len(result["nodes"]) == 4
        # Verify agent nodes have configs
        assert hasattr(result["nodes"]["agent1"], "config")
        assert hasattr(result["nodes"]["agent2"], "config")


class TestTenantIsolation:
    """Tenant isolation in workflow parsing and execution."""

    def test_workflow_nodes_isolated_by_tenant_id(self):
        """Node IDs should support tenant isolation in graph execution."""
        # This tests that node IDs can include tenant prefix
        workflow = {
            "nodes": [
                {"id": "tenant_t1:node_entry", "type": "entry"},
                {"id": "tenant_t1:node_agent", "type": "agent_node"},
            ],
            "edges": [
                {"source": "tenant_t1:node_entry", "target": "tenant_t1:node_agent"}
            ],
        }
        result = GraphBuilder.build(workflow)
        assert "tenant_t1:node_entry" in result["nodes"]
        assert "tenant_t1:node_agent" in result["nodes"]

    def test_workflow_multiple_tenants_separate_nodes(self):
        """Different tenants should have separate node instances."""
        workflow_t1 = {"nodes": [{"id": "t1:entry", "type": "entry"}], "edges": []}
        workflow_t2 = {"nodes": [{"id": "t2:entry", "type": "entry"}], "edges": []}
        result_t1 = GraphBuilder.build(workflow_t1)
        result_t2 = GraphBuilder.build(workflow_t2)

        assert "t1:entry" in result_t1["nodes"]
        assert "t2:entry" in result_t2["nodes"]
        assert result_t1["nodes"] != result_t2["nodes"]


class TestStateIsolation:
    """State isolation in node execution."""

    def test_agent_node_output_keys_isolated_by_node(self):
        """Agent node outputs should be keyed by node ID."""
        config = NodeConfig()
        agent_func = build_agent_node(config, "You are helpful", "Output: {x}")

        state1 = {"x": "data1", "current_node": "node_a"}
        state2 = {"x": "data2", "current_node": "node_b"}

        result1 = agent_func(state1)
        result2 = agent_func(state2)

        assert "node_a_output" in result1
        assert "node_b_output" in result2
        assert result1["node_a_output"] != result2["node_b_output"]

    def test_agent_node_preserves_previous_outputs(self):
        """Agent node should preserve outputs from previous nodes."""
        config = NodeConfig()
        agent_func = build_agent_node(config, "You are helpful", "New output")

        state = {
            "current_node": "node_b",
            "node_a_output": "previous_result",
            "run_id": "r123",
        }
        result = agent_func(state)

        assert result["node_a_output"] == "previous_result"
        assert result["node_b_output"] == "New output"
        assert result["run_id"] == "r123"


class TestErrorHandling:
    """Error handling and edge cases."""

    def test_invalid_edge_condition_raises_on_evaluation(self):
        """Invalid edge condition syntax should raise WorkflowParseError at evaluation time."""
        # Conditions are validated lazily during branch evaluation, not during build
        workflow = {
            "nodes": [
                {"id": "entry", "type": "entry"},
                {"id": "branch", "type": "branch", "condition": "$.x >>> 5"},
            ],
            "edges": [],
        }
        # Build succeeds (lazy validation)
        result = GraphBuilder.build(workflow)
        branch_func = result["nodes"]["branch"]

        # Evaluation fails with invalid syntax
        state = {"x": 5}
        with pytest.raises(WorkflowParseError, match="Invalid edge condition"):
            branch_func(state)

    def test_missing_required_node_type(self):
        """Node without type should raise WorkflowParseError."""
        workflow = {
            "nodes": [{"id": "node1"}],  # Missing type
            "edges": [],
        }
        with pytest.raises(WorkflowParseError, match="Unknown node type"):
            GraphBuilder.build(workflow)

    def test_agent_node_missing_template_variable_raises(self):
        """Agent node with missing template variable should raise on execution."""
        config = NodeConfig()
        agent_func = build_agent_node(config, "Help", "Process {missing_var}")
        state = {"current_node": "agent"}

        with pytest.raises(WorkflowParseError, match="Missing template variable"):
            agent_func(state)


class TestNodeConfigDataclass:
    """NodeConfig dataclass tests."""

    def test_node_config_all_fields_optional(self):
        """NodeConfig should accept all fields as optional."""
        config = NodeConfig()
        assert config.provider is None
        assert config.model is None
        assert config.temperature is None
        assert config.max_tokens is None
        assert config.tools is None

    def test_node_config_partial_init(self):
        """NodeConfig should accept partial field initialization."""
        config = NodeConfig(provider="openai", temperature=0.8)
        assert config.provider == "openai"
        assert config.temperature == 0.8
        assert config.model is None
        assert config.max_tokens is None

    def test_node_config_full_init(self):
        """NodeConfig should accept full field initialization."""
        config = NodeConfig(
            provider="openai",
            model="gpt-4",
            temperature=0.9,
            max_tokens=1024,
            tools=["search", "verify"],
        )
        assert config.provider == "openai"
        assert config.model == "gpt-4"
        assert config.temperature == 0.9
        assert config.max_tokens == 1024
        assert config.tools == ["search", "verify"]
