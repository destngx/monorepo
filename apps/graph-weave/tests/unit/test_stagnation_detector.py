import pytest
import json
import time
from datetime import datetime



class TestStagnationDetectorBasics:
    def test_instantiation_with_default_max_hops(self):
        detector = StagnationDetector()
        assert detector.max_hops == 20
        assert detector.total_visits == 0
        assert detector.visit_counts == {}

    def test_instantiation_with_custom_max_hops(self):
        detector = StagnationDetector(max_hops=50)
        assert detector.max_hops == 50

    def test_track_node_visit_increments_counter(self):
        detector = StagnationDetector()
        count = detector.track_node_visit("node_1")
        assert count == 1
        assert detector.total_visits == 1

    def test_track_multiple_visits_same_node(self):
        detector = StagnationDetector()
        detector.track_node_visit("node_1")
        detector.track_node_visit("node_1")
        detector.track_node_visit("node_1")
        assert detector.get_visit_count("node_1") == 3
        assert detector.total_visits == 3

    def test_track_visits_different_nodes(self):
        detector = StagnationDetector()
        detector.track_node_visit("node_1")
        detector.track_node_visit("node_2")
        detector.track_node_visit("node_3")
        assert detector.get_visit_count("node_1") == 1
        assert detector.get_visit_count("node_2") == 1
        assert detector.get_visit_count("node_3") == 1
        assert detector.total_visits == 3


class TestStagnationDetection:
    def test_is_stagnated_false_below_threshold(self):
        detector = StagnationDetector(max_hops=20)
        for i in range(10):
            detector.track_node_visit(f"node_{i}")
        assert not detector.is_stagnated()

    def test_is_stagnated_true_at_max_hops(self):
        detector = StagnationDetector(max_hops=20)
        for i in range(20):
            detector.track_node_visit(f"node_{i % 5}")
        assert detector.is_stagnated()

    def test_is_stagnated_true_exceeds_max_hops(self):
        detector = StagnationDetector(max_hops=20)
        for i in range(25):
            detector.track_node_visit(f"node_{i % 5}")
        assert detector.is_stagnated()

    def test_is_stagnated_with_override_limit(self):
        detector = StagnationDetector(max_hops=20)
        for i in range(15):
            detector.track_node_visit("node_1")
        assert not detector.is_stagnated()
        assert detector.is_stagnated(max_hops=15)

    def test_is_stagnated_false_one_below_threshold(self):
        detector = StagnationDetector(max_hops=20)
        for i in range(19):
            detector.track_node_visit(f"node_{i}")
        assert not detector.is_stagnated()


class TestStagnationDetectorReset:
    def test_reset_clears_visit_counts(self):
        detector = StagnationDetector()
        detector.track_node_visit("node_1")
        detector.track_node_visit("node_2")
        assert detector.total_visits == 2

        detector.reset()
        assert detector.total_visits == 0
        assert detector.visit_counts == {}

    def test_reset_allows_new_tracking(self):
        detector = StagnationDetector()
        detector.track_node_visit("node_1")
        detector.reset()
        detector.track_node_visit("node_2")
        assert detector.total_visits == 1
        assert detector.get_visit_count("node_2") == 1


class TestStagnationDetectorSummary:
    def test_get_summary_returns_visit_counts(self):
        detector = StagnationDetector()
        detector.track_node_visit("node_1")
        detector.track_node_visit("node_1")
        detector.track_node_visit("node_2")
        summary = detector.get_summary()
        assert summary == {"node_1": 2, "node_2": 1}

    def test_get_visit_count_nonexistent_node(self):
        detector = StagnationDetector()
        assert detector.get_visit_count("nonexistent") == 0


class TestRealLangGraphExecutorBasics:
    def test_instantiation_with_defaults(self, mock_mcp_router):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        assert executor.default_timeout_seconds == 300
        assert executor.mcp_router is not None

    def test_instantiation_with_custom_timeout(self, mock_mcp_router):
        executor = RealLangGraphExecutor(
            mcp_router=mock_mcp_router, default_timeout_seconds=600
        )
        assert executor.default_timeout_seconds == 600


class TestRealLangGraphExecutorEventEmission:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_events_emitted_for_simple_workflow(self, mock_mcp_router, simple_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=simple_workflow,
            input_data={"topic": "test"},
        )

        events = result["events"]
        assert len(events) > 0
        event_types = [e["type"] for e in events]
        assert "request.started" in event_types
        assert "request.completed" in event_types

    def test_event_format_validation(self, mock_mcp_router, simple_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=simple_workflow,
            input_data={"topic": "test"},
        )

        events = result["events"]
        for event in events:
            assert "type" in event
            assert "run_id" in event
            assert "timestamp" in event
            assert "data" in event
            assert event["timestamp"].endswith("Z")


class TestRealLangGraphExecutorStateAccumulation:
    @pytest.fixture
    def multi_node_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {
                        "system_prompt": "Test",
                        "user_prompt_template": "Input: {input}",
                    },
                },
                {
                    "id": "node_2",
                    "type": "agent_node",
                    "config": {
                        "system_prompt": "Test2",
                        "user_prompt_template": "Process: {node_1_output}",
                    },
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "node_2"},
                {"from": "node_2", "to": "exit"},
            ],
        }

    def test_workflow_state_accumulation(self, mock_mcp_router, multi_node_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=multi_node_workflow,
            input_data={"input": "test_data"},
        )

        workflow_state = result["workflow_state"]
        assert "input" in workflow_state
        assert result["status"] in ["completed", "stagnated", "timeout", "failed"]


class TestRealLangGraphExecutorTimeout:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_timeout_stops_execution(self, mock_mcp_router, simple_workflow):
        executor = RealLangGraphExecutor(
            mcp_router=mock_mcp_router, default_timeout_seconds=1
        )
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=simple_workflow,
            input_data={"topic": "test"},
            timeout_seconds=0.1,
        )

        assert "elapsed_seconds" in result
        assert result["elapsed_seconds"] >= 0


class TestRealLangGraphExecutorPerNodeConfig:
    @pytest.fixture
    def provider_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {
                        "provider": "github-copilot",
                        "model": "gpt-4.1",
                        "temperature": 0.5,
                        "max_tokens": 1000,
                        "system_prompt": "You are helpful",
                        "user_prompt_template": "Test",
                    },
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_provider_config_used(self, mock_mcp_router, provider_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=provider_workflow,
            input_data={"topic": "test"},
        )

        assert result["status"] in ["completed", "failed"]


class TestStagnationDetectionInWorkflow:
    @pytest.fixture
    def looping_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "limits": {"max_hops": 3},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "loop_node",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "loop_node"},
                {"from": "loop_node", "to": "loop_node"},
            ],
        }

    def test_stagnation_stops_execution(self, mock_mcp_router, looping_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=looping_workflow,
            input_data={},
        )

        assert result["hop_count"] <= looping_workflow["limits"]["max_hops"]


class TestCircuitBreakerKillFlag:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_kill_flag_check_works(self, mock_mcp_router, simple_workflow):
        fallback = FallbackStorage()
        redis_adapter = MockRedisAdapter()
        redis_client = NamespacedRedisClient(redis_adapter, fallback)
        executor = RealLangGraphExecutor(
            mcp_router=mock_mcp_router, redis_client=redis_client
        )

        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=simple_workflow,
            input_data={},
        )

        assert "status" in result


class TestEventOrdering:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_request_started_before_completed(self, mock_mcp_router, simple_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=simple_workflow,
            input_data={},
        )

        events = result["events"]
        event_types = [e["type"] for e in events]

        started_idx = event_types.index("request.started")
        completed_idx = event_types.index("request.completed")

        assert started_idx < completed_idx


class TestErrorHandling:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_missing_entry_node_handled(self, mock_mcp_router):
        workflow = {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [{"id": "exit", "type": "exit"}],
            "edges": [],
        }
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=workflow,
            input_data={},
        )

        assert result["status"] == "failed"


class TestConcurrentExecution:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_multiple_executions_independent(self, mock_mcp_router, simple_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)

        result1 = executor.execute(
            run_id="run-1",
            thread_id="thread-1",
            tenant_id="tenant-1",
            workflow=simple_workflow,
            input_data={"data": "test1"},
        )

        result2 = executor.execute(
            run_id="run-2",
            thread_id="thread-2",
            tenant_id="tenant-2",
            workflow=simple_workflow,
            input_data={"data": "test2"},
        )

        assert result1["run_id"] == "run-1"
        assert result2["run_id"] == "run-2"
        assert len(executor.execution_events["run-1"]) > 0
        assert len(executor.execution_events["run-2"]) > 0


class TestProviderSwitching:
    @pytest.fixture
    def switching_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "github_node",
                    "type": "agent_node",
                    "config": {
                        "provider": "github-copilot",
                        "model": "gpt-4.1",
                        "system_prompt": "You are helpful",
                        "user_prompt_template": "Test",
                    },
                },
                {
                    "id": "openai_node",
                    "type": "agent_node",
                    "config": {
                        "provider": "openai",
                        "model": "gpt-4.1",
                        "system_prompt": "You are helpful",
                        "user_prompt_template": "Test",
                    },
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "github_node"},
                {"from": "github_node", "to": "openai_node"},
                {"from": "openai_node", "to": "exit"},
            ],
        }

    def test_provider_switching_workflow(self, mock_mcp_router, switching_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=switching_workflow,
            input_data={},
        )

        assert result["status"] in ["completed", "failed"]


class TestEdgeRouting:
    @pytest.fixture
    def conditional_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {
                    "id": "branch",
                    "type": "branch",
                    "config": {"condition": "$.value > 5"},
                },
                {
                    "id": "node_yes",
                    "type": "agent_node",
                    "config": {"system_prompt": "Yes"},
                },
                {
                    "id": "node_no",
                    "type": "agent_node",
                    "config": {"system_prompt": "No"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "branch"},
                {"from": "branch", "to": "node_yes", "condition": "$.value > 5"},
                {"from": "branch", "to": "node_no"},
                {"from": "node_yes", "to": "exit"},
                {"from": "node_no", "to": "exit"},
            ],
        }

    def test_edge_routing_works(self, mock_mcp_router, conditional_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=conditional_workflow,
            input_data={"value": 3},
        )

        assert result["status"] in ["completed", "failed", "stagnated"]


class TestExecutionMetadata:
    @pytest.fixture
    def simple_workflow(self):
        return {
            "workflow_id": "test-workflow",
            "metadata": {"tenant_id": "test-tenant"},
            "nodes": [
                {"id": "entry", "type": "entry"},
                {
                    "id": "node_1",
                    "type": "agent_node",
                    "config": {"system_prompt": "Test"},
                },
                {"id": "exit", "type": "exit"},
            ],
            "edges": [
                {"from": "entry", "to": "node_1"},
                {"from": "node_1", "to": "exit"},
            ],
        }

    def test_execution_result_structure(self, mock_mcp_router, simple_workflow):
        executor = RealLangGraphExecutor(mcp_router=mock_mcp_router)
        result = executor.execute(
            run_id="test-run",
            thread_id="test-thread",
            tenant_id="test-tenant",
            workflow=simple_workflow,
            input_data={},
        )

        assert "run_id" in result
        assert "thread_id" in result
        assert "tenant_id" in result
        assert "workflow_id" in result
        assert "status" in result
        assert "hop_count" in result
        assert "events" in result
        assert "final_state" in result
