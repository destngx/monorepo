"""
Unit tests for OrchestratorReAct module.

Tests are isolated: LLM + MCP router are fully mocked.

Slices:
  1. Happy-path run  → orchestrator_trace + final_result returned
  2. Circuit breaker → exits after max_iterations; trace length == max_iterations
  3. Streaming events → orchestrator.thought / orchestrator.tool_called emitted each iteration
  4. Context trim    → _trim_context keeps window to max_context_messages
"""

import json
import pytest
from typing import Any, Dict, List, Optional
from unittest.mock import MagicMock

from src.models import OrchestratorConfig
from src.modules.orchestrator_react import OrchestratorReAct


# ---------------------------------------------------------------------------
# Helpers / shared fixtures
# ---------------------------------------------------------------------------

def _make_config(**kwargs) -> OrchestratorConfig:
    defaults = {
        "system_prompt": "You are an SRE incident responder.",
        "allowed_skills": ["search", "load_skill"],
        "max_iterations": 5,
    }
    defaults.update(kwargs)
    return OrchestratorConfig(**defaults)


def _finish_response(content: str = '{"summary": "done"}') -> Dict[str, Any]:
    """LLM response with no tool_calls → tells the loop to finish."""
    return {
        "choices": [{"message": {"role": "assistant", "content": content}}],
        "usage": {"total_tokens": 10},
        "model": "gpt-4.1",
    }


def _tool_call_response(tool_name: str, args: Dict) -> Dict[str, Any]:
    """LLM response requesting a tool call."""
    return {
        "choices": [{
            "message": {
                "role": "assistant",
                "content": None,
                "tool_calls": [{
                    "id": f"call_{tool_name}_0",
                    "type": "function",
                    "function": {
                        "name": tool_name,
                        "arguments": json.dumps(args),
                    },
                }],
            }
        }],
        "usage": {"total_tokens": 12},
        "model": "gpt-4.1",
    }


# ---------------------------------------------------------------------------
# Slice 1 · Happy path
# ---------------------------------------------------------------------------

class TestOrchestratorHappyPath:
    """LLM decides to finish on iteration 0 (no tool calls needed)."""

    def test_returns_orchestrator_trace_and_final_result(self):
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _finish_response('{"answer": "all clear"}')

        mock_mcp = MagicMock()
        emitted: List[Dict] = []

        react = OrchestratorReAct(
            client=mock_client,
            mcp_router=mock_mcp,
            emit=lambda etype, data: emitted.append({"type": etype, **data}),
        )
        config = _make_config(max_iterations=3)

        result = react.run(
            run_id="run-001",
            node_id="node-orch",
            config=config,
            workflow_state={"incident": "high CPU on prod"},
        )

        assert "orchestrator_trace" in result
        assert "final_result" in result
        assert result["final_result"] == {"answer": "all clear"}

    def test_trace_contains_thought_entry_on_finish(self):
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _finish_response('{"answer": "resolved"}')
        mock_mcp = MagicMock()

        react = OrchestratorReAct(client=mock_client, mcp_router=mock_mcp, emit=lambda *a: None)
        result = react.run("r1", "n1", _make_config(max_iterations=3), {})

        trace = result["orchestrator_trace"]
        assert any(entry["type"] == "thought" for entry in trace)

    def test_single_tool_call_then_finish(self):
        """LLM calls 'search' once, then finishes on next turn."""
        responses = [
            _tool_call_response("search", {"query": "check grafana"}),
            _finish_response('{"found": "high error rate"}'),
        ]
        mock_client = MagicMock()
        mock_client.chat_completion.side_effect = responses

        mock_mcp = MagicMock()
        mock_mcp.execute_tool.return_value = {"results": ["error_rate=5%"]}

        react = OrchestratorReAct(client=mock_client, mcp_router=mock_mcp, emit=lambda *a: None)
        result = react.run("r1", "n1", _make_config(max_iterations=5), {"env": "prod"})

        assert result["final_result"] == {"found": "high error rate"}
        # Tool should have been called
        mock_mcp.execute_tool.assert_called_once_with("search", {"query": "check grafana"})
        # Trace should have action + observation entries
        types = [e["type"] for e in result["orchestrator_trace"]]
        assert "action" in types
        assert "observation" in types


# ---------------------------------------------------------------------------
# Slice 2 · Circuit breaker
# ---------------------------------------------------------------------------

class TestOrchestratorCircuitBreaker:
    """LLM never finishes → loop must exit after max_iterations."""

    def test_exits_after_max_iterations(self):
        # LLM always requests a tool call, never finishes
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _tool_call_response("search", {"query": "still looking"})

        mock_mcp = MagicMock()
        mock_mcp.execute_tool.return_value = {"results": ["no answer yet"]}

        react = OrchestratorReAct(client=mock_client, mcp_router=mock_mcp, emit=lambda *a: None)
        config = _make_config(max_iterations=3)

        result = react.run("r2", "n2", config, {})

        # Must return something — not raise
        assert "orchestrator_trace" in result
        assert "final_result" in result
        # LLM was called at most max_iterations times
        assert mock_client.chat_completion.call_count <= config.max_iterations

    def test_final_result_signals_circuit_breaker(self):
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _tool_call_response("search", {"query": "looping"})
        mock_mcp = MagicMock()
        mock_mcp.execute_tool.return_value = {"results": []}

        react = OrchestratorReAct(client=mock_client, mcp_router=mock_mcp, emit=lambda *a: None)
        result = react.run("r2", "n2", _make_config(max_iterations=2), {})

        # final_result should carry a circuit-breaker indicator
        assert result["final_result"].get("error") == "max_iterations_exceeded"

    def test_trace_length_matches_iterations_used(self):
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _tool_call_response("load_skill", {"skill_name": "aws"})
        mock_mcp = MagicMock()
        mock_mcp.execute_tool.return_value = {}

        max_iter = 4
        react = OrchestratorReAct(client=mock_client, mcp_router=mock_mcp, emit=lambda *a: None)
        result = react.run("r2", "n3", _make_config(max_iterations=max_iter), {})

        # Each iteration adds at least one trace entry
        assert len(result["orchestrator_trace"]) >= max_iter


# ---------------------------------------------------------------------------
# Slice 3 · Streaming events
# ---------------------------------------------------------------------------

class TestOrchestratorStreamingEvents:
    """orchestrator.thought and orchestrator.tool_called emitted after each iteration."""

    def test_thought_event_emitted_on_finish(self):
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _finish_response('{"ok": true}')
        mock_mcp = MagicMock()

        emitted: List[Dict] = []
        react = OrchestratorReAct(
            client=mock_client,
            mcp_router=mock_mcp,
            emit=lambda etype, data: emitted.append({"type": etype, **data}),
        )
        react.run("r3", "n3", _make_config(max_iterations=3), {})

        thought_events = [e for e in emitted if e["type"] == "orchestrator.thought"]
        assert len(thought_events) >= 1
        assert "node_id" in thought_events[0]
        assert "iteration" in thought_events[0]

    def test_tool_called_event_emitted_on_tool_use(self):
        responses = [
            _tool_call_response("search", {"query": "check db"}),
            _finish_response('{"resolved": true}'),
        ]
        mock_client = MagicMock()
        mock_client.chat_completion.side_effect = responses
        mock_mcp = MagicMock()
        mock_mcp.execute_tool.return_value = {"results": ["ok"]}

        emitted: List[Dict] = []
        react = OrchestratorReAct(
            client=mock_client,
            mcp_router=mock_mcp,
            emit=lambda etype, data: emitted.append({"type": etype, **data}),
        )
        react.run("r3", "n4", _make_config(max_iterations=5), {})

        tool_events = [e for e in emitted if e["type"] == "orchestrator.tool_called"]
        assert len(tool_events) == 2
        assert tool_events[0]["tool_name"] == "search"
        assert "result" not in tool_events[0]
        assert tool_events[1]["tool_name"] == "search"
        assert "result" in tool_events[1]

    def test_finished_event_emitted_at_end(self):
        mock_client = MagicMock()
        mock_client.chat_completion.return_value = _finish_response('{}')
        mock_mcp = MagicMock()

        emitted: List[Dict] = []
        react = OrchestratorReAct(
            client=mock_client,
            mcp_router=mock_mcp,
            emit=lambda etype, data: emitted.append({"type": etype, **data}),
        )
        react.run("r3", "n5", _make_config(), {})

        finished = [e for e in emitted if e["type"] == "orchestrator.finished"]
        assert len(finished) == 1
        assert "iterations_used" in finished[0]
        assert "status" in finished[0]


# ---------------------------------------------------------------------------
# Slice 4 · Context window trimming
# ---------------------------------------------------------------------------

class TestOrchestratorContextTrim:
    """_trim_context must keep the window to max_context_messages."""

    def test_trim_keeps_system_message(self):
        react = OrchestratorReAct(
            client=MagicMock(),
            mcp_router=MagicMock(),
            emit=lambda *a: None,
            max_context_messages=4,
        )
        messages = [
            {"role": "system", "content": "You are an SRE."},
            {"role": "user", "content": "msg1"},
            {"role": "assistant", "content": "msg2"},
            {"role": "user", "content": "msg3"},
            {"role": "assistant", "content": "msg4"},
            {"role": "user", "content": "msg5"},  # oldest non-system should drop
        ]
        trimmed = react._trim_context(messages)

        assert trimmed[0]["role"] == "system"
        assert len(trimmed) <= 4

    def test_trim_no_op_within_window(self):
        react = OrchestratorReAct(
            client=MagicMock(),
            mcp_router=MagicMock(),
            emit=lambda *a: None,
            max_context_messages=10,
        )
        messages = [
            {"role": "system", "content": "sys"},
            {"role": "user", "content": "u1"},
            {"role": "assistant", "content": "a1"},
        ]
        trimmed = react._trim_context(messages)
        assert trimmed == messages

    def test_trim_preserves_recency(self):
        """After trimming, the LATEST messages should survive."""
        react = OrchestratorReAct(
            client=MagicMock(),
            mcp_router=MagicMock(),
            emit=lambda *a: None,
            max_context_messages=3,
        )
        messages = [
            {"role": "system", "content": "sys"},
            {"role": "user", "content": "old1"},
            {"role": "assistant", "content": "old2"},
            {"role": "user", "content": "recent1"},
            {"role": "assistant", "content": "recent2"},
        ]
        trimmed = react._trim_context(messages)

        contents = [m["content"] for m in trimmed]
        assert "recent2" in contents
        assert "old1" not in contents
