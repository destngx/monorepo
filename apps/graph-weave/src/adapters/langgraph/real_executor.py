import json
import time
from datetime import datetime, timezone
from typing import Any, Dict, Optional, List
from src.app_logging import get_logger

from .base_executor import BaseLangGraphExecutor
from .stagnation_detector import StagnationDetector
from .agent_node import AgentNodeHandler
from .orchestrator_node import OrchestratorNodeHandler
from ..ai_provider import AIProviderFactory, LLMClient
from ..mcp_router import MCPRouter
from ..redis_circuit_breaker import NamespacedRedisClient

logger = get_logger(__name__)

class RealLangGraphExecutor(BaseLangGraphExecutor):
    """
    Real LangGraph executor with full event emission, stagnation detection,
    timeout enforcement, and circuit breaker support.
    """

    _logger = get_logger(__name__)

    def __init__(
        self,
        ai_provider: Optional[LLMClient] = None,
        mcp_router: Optional[MCPRouter] = None,
        redis_client: Optional[NamespacedRedisClient] = None,
        checkpoint_service: Optional[Any] = None,
        default_timeout_seconds: int = 300,
    ):
        self.ai_provider_factory = AIProviderFactory(ai_gateway_client=ai_provider)
        self.mcp_router = mcp_router or MCPRouter()
        self.redis_client = redis_client
        self.checkpoint_service = checkpoint_service
        self.default_timeout_seconds = default_timeout_seconds
        self.execution_events: Dict[str, List[Dict[str, Any]]] = {}
        
        # Initialize specialized node handlers
        self.agent_handler = AgentNodeHandler(self)
        self.orchestrator_handler = OrchestratorNodeHandler(self)

    def execute(
        self,
        run_id: str,
        thread_id: str,
        tenant_id: str,
        workflow: Dict[str, Any],
        input_data: Dict[str, Any],
        timeout_seconds: Optional[int] = None,
    ) -> Dict[str, Any]:
        self._logger.info(f"Executing workflow: {workflow.get('name')} {workflow.get('version')}")
        workflow_id = workflow.get("workflow_id", "unknown")
        timeout = timeout_seconds or self.default_timeout_seconds
        start_time = time.monotonic()
        detector = StagnationDetector(
            max_hops=workflow.get("limits", {}).get("max_hops", 20)
        )

        self.execution_events[run_id] = []
        workflow = self._normalize_workflow(workflow)

        state: Dict[str, Any] = {
            "input": input_data,
            "step": 0,
            "current_node": None,
            "node_results": {},
            "workflow_state": dict(input_data),
            "status": None,
            "hop_count": 0,
            "errors": [],
        }

        try:
            self._emit_event(run_id, "request.started", {"tenant_id": tenant_id, "workflow_id": workflow_id})
            current_node_id = self._find_entry_node(workflow)
            if not current_node_id:
                raise ValueError("No entry node found in workflow")

            state["current_node"] = current_node_id
            hop_count = 0

            while True:
                # 1. Check constraints (timeout, kill flag, stagnation)
                if self._should_stop(run_id, tenant_id, start_time, timeout, detector, state):
                    break

                hop_count += 1
                detector.track_node_visit(current_node_id)
                node = self._find_node(workflow, current_node_id)
                if not node:
                    raise ValueError(f"Node not found: {current_node_id}")

                node_type = node.get("type")
                state["current_node"] = current_node_id
                state["step"] = hop_count

                # 2. Handle Exit Node
                if node_type == "exit":
                    self._handle_exit_node(run_id, workflow_id, node, state)
                    break

                # 3. Resolve Node Input & Emit Start Event
                resolved_input = self._resolve_node_input(node, state)
                self._emit_event(run_id, "node.started", {"node_id": current_node_id, "node_type": node_type, "input_data": resolved_input})

                # 4. Execute Node
                try:
                    node_result = self.agent_handler.executor._dispatch_node_execution(run_id, node, state, workflow)
                    self._logger.debug(f"Node {current_node_id} result keys: {list(node_result.keys())}")
                    state["node_results"][current_node_id] = node_result
                    state["workflow_state"].update(node_result)

                    if self.checkpoint_service:
                        self.checkpoint_service.save_checkpoint(tenant_id=tenant_id, thread_id=thread_id, workflow_state=state["workflow_state"])

                    self._emit_event(run_id, "node.completed", {"node_id": current_node_id, "node_type": node_type, "result": node_result, "result_keys": list(node_result.keys())})

                except Exception as e:
                    self._handle_node_failure(run_id, current_node_id, state, e)
                    # Stop execution if node is not optional (default)
                    node_config = node.get("config", {})
                    if not node_config.get("optional", False):
                        state["status"] = "failed"
                        break

                # 5. Route to Next Node
                next_node_id = self._route_by_edge(run_id, workflow, current_node_id, state)
                if next_node_id is None or next_node_id == current_node_id:
                    break
                current_node_id = next_node_id

            return self._finalize_execution(run_id, thread_id, tenant_id, workflow_id, state, start_time, hop_count)

        except Exception as e:
            return self._handle_execution_error(run_id, thread_id, tenant_id, workflow_id, state, e)

    def _should_stop(self, run_id: str, tenant_id: str, start_time: float, timeout: int, detector: StagnationDetector, state: Dict[str, Any]) -> bool:
        elapsed = time.monotonic() - start_time
        if elapsed > timeout:
            self._emit_event(run_id, "execution.timeout", {"elapsed_seconds": elapsed})
            state["status"] = "timeout"
            return True

        if self._check_kill_flag(run_id, tenant_id):
            self._emit_event(run_id, "execution.killed", {"elapsed_seconds": elapsed})
            state["status"] = "killed"
            return True

        if detector.is_stagnated():
            self._emit_event(run_id, "execution.stagnated", {"visits": detector.get_summary(), "max_hops": detector.max_hops})
            state["status"] = "stagnated"
            return True
        return False

    def _handle_exit_node(self, run_id: str, workflow_id: str, node: Dict[str, Any], state: Dict[str, Any]) -> None:
        config = node.get("config", {})
        mapping = node.get("output_mapping") or config.get("output_mapping", {})
        if mapping:
            final_output = {}
            for key, path in mapping.items():
                val = self._get_state_value(path, state)
                self._logger.debug(f"Mapping {key} -> {path} resolved to: {type(val).__name__}")
                final_output[key] = val
            state["workflow_state"] = final_output
        
        self._emit_event(run_id, "workflow.completed", {"run_id": run_id, "workflow_id": workflow_id, "status": "completed", "final_state_keys": list(state["workflow_state"].keys())})
        state["status"] = "completed"

    def _resolve_node_input(self, node: Dict[str, Any], state: Dict[str, Any]) -> Dict[str, Any]:
        node_type = node.get("type")
        if node_type in {"agent_node", "agent", "orchestrator"}:
            config = node.get("config", {})
            input_mapping = node.get("input_mapping") or config.get("input_mapping", {})
            if input_mapping:
                resolved = {}
                for key, path in input_mapping.items():
                    resolved[key] = self._get_state_value(path, state)
                return resolved
        elif node_type == "entry":
            return state.get("input", {})
        return {}

    def _dispatch_node_execution(self, run_id: str, node: Dict[str, Any], state: Dict[str, Any], workflow: Dict[str, Any]) -> Dict[str, Any]:
        node_type = node.get("type")
        if node_type in {"agent_node", "agent"}:
            return self.agent_handler.execute(run_id, node, state, workflow)
        elif node_type == "entry":
            return dict(state.get("input", {}))
        elif node_type == "branch":
            return {"branch_result": "true"}
        elif node_type == "guardrail":
            return {"guardrail_passed": True, "node_id": node.get("id")}
        elif node_type == "skill_loader":
            return {"skills_loaded": True, "node_id": node.get("id")}
        elif node_type == "orchestrator":
            return self.orchestrator_handler.execute(run_id, node, state, workflow)
        else:
            raise ValueError(f"Unknown node type: {node_type}")

    def _handle_node_failure(self, run_id: str, node_id: str, state: Dict[str, Any], error: Exception) -> None:
        self._logger.error(f"Node {node_id} failed: {error}", exc_info=True)
        self._emit_event(run_id, "node.failed", {"node_id": node_id, "error": str(error)})
        state["errors"].append({
            "node_id": node_id,
            "error": str(error),
            "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        })

    def _finalize_execution(self, run_id: str, thread_id: str, tenant_id: str, workflow_id: str, state: Dict[str, Any], start_time: float, hop_count: int) -> Dict[str, Any]:
        state["status"] = state.get("status") or "completed"
        state["hop_count"] = hop_count
        elapsed = time.monotonic() - start_time
        self._emit_event(run_id, "request.completed", {"status": state["status"], "hop_count": hop_count, "elapsed_seconds": elapsed})
        return {
            "run_id": run_id, "thread_id": thread_id, "tenant_id": tenant_id, "workflow_id": workflow_id,
            "status": state["status"], "hop_count": hop_count, "events": self.execution_events.get(run_id, []),
            "final_state": state, "workflow_state": state["workflow_state"], "elapsed_seconds": elapsed,
        }

    def _handle_execution_error(self, run_id: str, thread_id: str, tenant_id: str, workflow_id: str, state: Dict[str, Any], error: Exception) -> Dict[str, Any]:
        self._logger.exception(f"Execution error: {error}")
        state["status"] = "failed"
        self._emit_event(run_id, "request.failed", {"error": str(error)})
        return {
            "run_id": run_id, "thread_id": thread_id, "tenant_id": tenant_id, "workflow_id": workflow_id,
            "status": "failed", "error": str(error), "events": self.execution_events.get(run_id, []),
            "final_state": state, "workflow_state": state.get("workflow_state", {}),
        }

    def _route_by_edge(self, run_id: str, workflow: Dict[str, Any], current_node_id: str, state: Dict[str, Any]) -> Optional[str]:
        edges = workflow.get("edges", [])
        matching_edges = [e for e in edges if e.get("from") == current_node_id]
        if not matching_edges:
            return self._find_exit_node(workflow)

        target_node_id = None
        routing_edge = None
        for edge in matching_edges:
            condition = edge.get("condition")
            if not condition or self._evaluate_condition(condition, state):
                target_node_id = edge.get("to")
                routing_edge = edge
                break

        if not target_node_id and matching_edges:
            target_node_id = matching_edges[0].get("to")
            routing_edge = matching_edges[0]

        if target_node_id:
            self._emit_event(run_id, "edge_route", {"from": current_node_id, "to": target_node_id, "condition": routing_edge.get("condition") if routing_edge else None})
        return target_node_id

    def _check_kill_flag(self, run_id: str, tenant_id: str) -> bool:
        if not self.redis_client: return False
        try:
            return self.redis_client.exists(f"kill:{tenant_id}:{run_id}")
        except Exception: return False

    def _emit_event(self, run_id: str, event_type: str, data: Dict[str, Any]) -> None:
        if run_id not in self.execution_events:
            self.execution_events[run_id] = []
        event = {"type": event_type, "run_id": run_id, "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"), "data": data}
        self.execution_events[run_id].append(event)
        if self.redis_client:
            try:
                self.redis_client.rpush(f"event:{run_id}", json.dumps(event))
            except Exception as e:
                self._logger.warning(f"Failed to persist event to Redis: {e}")
