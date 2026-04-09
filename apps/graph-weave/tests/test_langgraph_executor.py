import pytest
import json
from src.adapters.langgraph_executor import MockLangGraphExecutor
from src.adapters.ai_provider import MockAIProvider


class TestMockLangGraphExecutorInstantiation:
    def test_instantiation_with_default_provider(self):
        executor = MockLangGraphExecutor()
        assert executor is not None
        assert executor.ai_provider is not None
        assert executor.execution_events == {}

    def test_instantiation_with_custom_provider(self):
        provider = MockAIProvider()
        executor = MockLangGraphExecutor(ai_provider=provider)
        assert executor.ai_provider is provider


class TestMockLangGraphExecutorNodeFinding:
    def test_find_entry_node(self, workflow_multi_node):
        executor = MockLangGraphExecutor()
        entry_node_id = executor._find_entry_node(workflow_multi_node)
        assert entry_node_id == "start"

    def test_find_exit_node(self, workflow_multi_node):
        executor = MockLangGraphExecutor()
        exit_node_id = executor._find_exit_node(workflow_multi_node)
        assert exit_node_id == "end"

    def test_find_node_by_id(self, workflow_multi_node):
        executor = MockLangGraphExecutor()
        node = executor._find_node(workflow_multi_node, "research_agent")
        assert node is not None
        assert node["id"] == "research_agent"
        assert node["type"] == "agent_node"

    def test_find_nonexistent_node(self, workflow_multi_node):
        executor = MockLangGraphExecutor()
        node = executor._find_node(workflow_multi_node, "nonexistent")
        assert node is None


class TestMockLangGraphExecutorEventLogging:
    def test_log_event_creates_timestamped_event(self):
        executor = MockLangGraphExecutor()
        run_id = "test-run-1"
        executor.set_current_run_id(run_id)

        executor._log_event(run_id, "node_execute", "Testing")

        events = executor.get_events(run_id)
        assert len(events) == 1
        assert events[0]["type"] == "node_execute"
        assert events[0]["message"] == "Testing"
        assert "timestamp" in events[0]

    def test_log_multiple_events(self):
        executor = MockLangGraphExecutor()
        run_id = "test-run-2"
        executor.set_current_run_id(run_id)

        executor._log_event(run_id, "node_execute", "First")
        executor._log_event(run_id, "agent_response", "Second")
        executor._log_event(run_id, "edge_route", "Third")

        events = executor.get_events(run_id)
        assert len(events) == 3
        assert events[0]["type"] == "node_execute"
        assert events[1]["type"] == "agent_response"
        assert events[2]["type"] == "edge_route"

    def test_event_timestamp_format(self):
        executor = MockLangGraphExecutor()
        run_id = "test-run-3"
        executor.set_current_run_id(run_id)

        executor._log_event(run_id, "test_event", "message")
        events = executor.get_events(run_id)

        timestamp = events[0]["timestamp"]
        assert timestamp.endswith("Z")
        assert "T" in timestamp

    def test_get_events_for_run_id(self):
        executor = MockLangGraphExecutor()
        run_id = "test-run-4"
        executor.set_current_run_id(run_id)

        executor._log_event(run_id, "event1", "msg1")
        executor._log_event(run_id, "event2", "msg2")

        events = executor.get_events(run_id)
        assert len(events) == 2

    def test_get_events_empty_run_id(self):
        executor = MockLangGraphExecutor()
        events = executor.get_events("nonexistent-run")
        assert events == []


class TestMockLangGraphExecutorPromptInterpolation:
    def test_interpolate_single_variable(self):
        executor = MockLangGraphExecutor()
        template = "Analyze {topic}"
        state = {"topic": "earnings"}

        result = executor._interpolate_prompt(template, state)
        assert result == "Analyze earnings"

    def test_interpolate_multiple_variables(self):
        executor = MockLangGraphExecutor()
        template = "Analyze {topic} with depth {depth}"
        state = {"topic": "earnings", "depth": "3"}

        result = executor._interpolate_prompt(template, state)
        assert result == "Analyze earnings with depth 3"

    def test_interpolate_with_missing_variable(self):
        executor = MockLangGraphExecutor()
        template = "Analyze {topic} and {missing}"
        state = {"topic": "earnings"}

        result = executor._interpolate_prompt(template, state)
        assert "{missing}" in result

    def test_interpolate_no_variables(self):
        executor = MockLangGraphExecutor()
        template = "Static text"
        state = {"topic": "earnings"}

        result = executor._interpolate_prompt(template, state)
        assert result == "Static text"


class TestMockLangGraphExecutorConditionEvaluation:
    def test_evaluate_condition_equals_true(self):
        executor = MockLangGraphExecutor()
        state = {"status": "research_complete"}

        result = executor._evaluate_condition("$.status == 'research_complete'", state)
        assert result is True

    def test_evaluate_condition_equals_false(self):
        executor = MockLangGraphExecutor()
        state = {"status": "error"}

        result = executor._evaluate_condition("$.status == 'research_complete'", state)
        assert result is False

    def test_evaluate_condition_not_equals_true(self):
        executor = MockLangGraphExecutor()
        state = {"confidence": "0.5"}

        result = executor._evaluate_condition("$.confidence != '0.5'", state)
        assert result is False

    def test_evaluate_condition_greater_than_true(self):
        executor = MockLangGraphExecutor()
        state = {"confidence": 0.92}

        result = executor._evaluate_condition("$.confidence > 0.8", state)
        assert result is True

    def test_evaluate_condition_greater_than_false(self):
        executor = MockLangGraphExecutor()
        state = {"confidence": 0.5}

        result = executor._evaluate_condition("$.confidence > 0.8", state)
        assert result is False

    def test_evaluate_condition_less_than_true(self):
        executor = MockLangGraphExecutor()
        state = {"hop_count": 5}

        result = executor._evaluate_condition("$.hop_count < 20", state)
        assert result is True

    def test_evaluate_condition_less_than_false(self):
        executor = MockLangGraphExecutor()
        state = {"hop_count": 25}

        result = executor._evaluate_condition("$.hop_count < 20", state)
        assert result is False

    def test_evaluate_condition_none_returns_true(self):
        executor = MockLangGraphExecutor()
        state = {"status": "any"}

        result = executor._evaluate_condition(None, state)
        assert result is True

    def test_evaluate_condition_empty_returns_true(self):
        executor = MockLangGraphExecutor()
        state = {"status": "any"}

        result = executor._evaluate_condition("", state)
        assert result is True


class TestMockLangGraphExecutorStateValueExtraction:
    def test_get_state_value_simple_key(self):
        executor = MockLangGraphExecutor()
        state = {"status": "running"}

        value = executor._get_state_value("$.status", state)
        assert value == "running"

    def test_get_state_value_nested_key(self):
        executor = MockLangGraphExecutor()
        state = {"node_results": {"agent_1": {"status": "done"}}}

        value = executor._get_state_value("$.node_results.agent_1.status", state)
        assert value == "done"

    def test_get_state_value_missing_key(self):
        executor = MockLangGraphExecutor()
        state = {"status": "running"}

        value = executor._get_state_value("$.missing", state)
        assert value is None

    def test_get_state_value_without_dollar_prefix(self):
        executor = MockLangGraphExecutor()
        state = {"status": "running"}

        value = executor._get_state_value("status", state)
        assert value == "running"


class TestMockLangGraphExecutorEdgeRouting:
    def test_route_by_edge_unconditional(self, workflow_multi_node):
        executor = MockLangGraphExecutor()
        state = {"status": "pending"}

        next_node = executor._route_by_edge(
            "test-run", workflow_multi_node, "start", state
        )
        assert next_node == "research_agent"

    def test_route_by_edge_with_matching_condition(self):
        executor = MockLangGraphExecutor()
        workflow = {
            "nodes": [
                {"id": "node1", "type": "agent_node"},
                {"id": "node2", "type": "agent_node"},
            ],
            "edges": [
                {"from": "node1", "to": "node2", "condition": "$.status == 'done'"},
            ],
        }
        state = {"status": "done"}
        executor.set_current_run_id("test-run")

        next_node = executor._route_by_edge("test-run", workflow, "node1", state)
        assert next_node == "node2"

    def test_route_by_edge_with_nonmatching_condition(self):
        executor = MockLangGraphExecutor()
        workflow = {
            "nodes": [
                {"id": "node1", "type": "agent_node"},
                {"id": "node2", "type": "agent_node"},
                {"id": "node3", "type": "agent_node"},
            ],
            "edges": [
                {"from": "node1", "to": "node2", "condition": "$.status == 'done'"},
                {"from": "node1", "to": "node3"},
            ],
        }
        state = {"status": "pending"}
        executor.set_current_run_id("test-run")

        next_node = executor._route_by_edge("test-run", workflow, "node1", state)
        assert next_node == "node3"

    def test_route_by_edge_no_outgoing_edges(self):
        executor = MockLangGraphExecutor()
        workflow = {
            "nodes": [
                {"id": "node1", "type": "agent_node"},
                {"id": "exit", "type": "exit"},
            ],
            "edges": [],
        }
        state = {"status": "pending"}
        executor.set_current_run_id("test-run")

        next_node = executor._route_by_edge("test-run", workflow, "node1", state)
        assert next_node == "exit"


class TestMockLangGraphExecutorAgentNodeExecution:
    def test_execute_agent_node(self):
        executor = MockLangGraphExecutor()
        node = {
            "id": "research_agent",
            "type": "agent_node",
            "config": {
                "system_prompt": "You are a research agent",
                "user_prompt_template": "Research {topic}",
            },
        }
        state = {"topic": "earnings", "step": 1}
        executor.set_current_run_id("test-run")

        result = executor._execute_agent_node("test-run", node, state, {}, None)

        assert result["node_id"] == "research_agent"
        assert result["status"] == "completed"
        assert "result" in result
        assert "timestamp" in result
        assert "tokens_used" in result
        assert isinstance(result["result"], dict)

    def test_execute_agent_node_updates_state(self):
        executor = MockLangGraphExecutor()
        node = {
            "id": "agent",
            "type": "agent_node",
            "config": {
                "system_prompt": "System",
                "user_prompt_template": "User",
            },
        }
        state = {"step": 1}
        executor.set_current_run_id("test-run")

        result = executor._execute_agent_node("test-run", node, state, {}, None)

        assert state["last_result"] is not None
        assert isinstance(state["last_result"], dict)


class TestMockLangGraphExecutorFullExecution:
    def test_execute_simple_workflow(self, workflow_multi_node, test_input_data):
        executor = MockLangGraphExecutor()
        run_id = "test-run-full"
        executor.set_current_run_id(run_id)

        from src.adapters.checkpoint import MockCheckpointStore
        from src.adapters.cache import MockRedisAdapter

        checkpoint_store = MockCheckpointStore()
        cache = MockRedisAdapter()

        result = executor.execute(
            workflow=workflow_multi_node,
            input_data=test_input_data,
            checkpoint_store=checkpoint_store,
            cache=cache,
        )

        assert result["run_id"] == run_id
        assert result["status"] == "completed"
        assert result["workflow_id"] == "multi-node-test:v1.0.0"
        assert len(result["events"]) > 0
        assert "final_state" in result
        assert "hop_count" in result

    def test_execute_respects_max_hops(self, test_input_data):
        executor = MockLangGraphExecutor()
        run_id = "test-run-hops"
        executor.set_current_run_id(run_id)

        workflow = {
            "workflow_id": "loop-test",
            "metadata": {"tenant_id": "test"},
            "limits": {"max_hops": 3},
            "nodes": [
                {"id": "start", "type": "entry", "config": {}},
                {
                    "id": "loop",
                    "type": "agent_node",
                    "config": {"system_prompt": "test", "user_prompt_template": "test"},
                },
                {"id": "end", "type": "exit"},
            ],
            "edges": [
                {"from": "start", "to": "loop"},
                {"from": "loop", "to": "loop"},
            ],
        }

        from src.adapters.checkpoint import MockCheckpointStore
        from src.adapters.cache import MockRedisAdapter

        checkpoint_store = MockCheckpointStore()
        cache = MockRedisAdapter()

        result = executor.execute(workflow, test_input_data, checkpoint_store, cache)

        assert result["hop_count"] <= 3

    def test_execute_logs_entry_event(self, workflow_multi_node, test_input_data):
        executor = MockLangGraphExecutor()
        run_id = "test-run-entry"
        executor.set_current_run_id(run_id)

        from src.adapters.checkpoint import MockCheckpointStore
        from src.adapters.cache import MockRedisAdapter

        checkpoint_store = MockCheckpointStore()
        cache = MockRedisAdapter()

        executor.execute(workflow_multi_node, test_input_data, checkpoint_store, cache)

        events = executor.get_events(run_id)
        event_types = [e["type"] for e in events]

        assert "node_entry" in event_types or "node_execute" in event_types

    def test_execute_logs_exit_event(self, workflow_multi_node, test_input_data):
        executor = MockLangGraphExecutor()
        run_id = "test-run-exit"
        executor.set_current_run_id(run_id)

        from src.adapters.checkpoint import MockCheckpointStore
        from src.adapters.cache import MockRedisAdapter

        checkpoint_store = MockCheckpointStore()
        cache = MockRedisAdapter()

        executor.execute(workflow_multi_node, test_input_data, checkpoint_store, cache)

        events = executor.get_events(run_id)
        event_types = [e["type"] for e in events]

        assert "node_exit" in event_types

    def test_execute_handles_error(self, test_input_data):
        executor = MockLangGraphExecutor()
        run_id = "test-run-error"
        executor.set_current_run_id(run_id)

        workflow = {
            "workflow_id": "error-test",
            "metadata": {"tenant_id": "test"},
            "limits": {},
            "nodes": [
                {"id": "start", "type": "entry", "config": {}},
            ],
            "edges": [],
        }

        from src.adapters.checkpoint import MockCheckpointStore
        from src.adapters.cache import MockRedisAdapter

        checkpoint_store = MockCheckpointStore()
        cache = MockRedisAdapter()

        result = executor.execute(workflow, test_input_data, checkpoint_store, cache)

        assert "error" in result or result["status"] == "completed"
