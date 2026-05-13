import time
from typing import Any, Dict, Optional, List
from src.app_logging import get_logger

from ..base.executor import BaseLangGraphExecutor
from ..stagnation import StagnationDetector
from ...nodes import AgentNodeHandler, OrchestratorNodeHandler, CLINodeHandler
from ....ai_provider import AIProviderFactory, LLMClient
from ....mcp_router import MCPRouter
from src.adapters.redis import NamespacedRedisClient

from .utils import should_stop, resolve_node_input
from .routing import route_by_edge
from .events import emit_event
from .handlers import (
    handle_exit_node,
    handle_node_failure,
    finalize_execution,
    handle_execution_error
)

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
        default_timeout_seconds: int = 1800,
        config: Optional[Any] = None,
    ):
        super().__init__(ai_provider=ai_provider, config=config)
        self.ai_provider_factory = AIProviderFactory(ai_gateway_client=ai_provider)
        self.mcp_router = mcp_router or MCPRouter()
        self.redis_client = redis_client
        self.checkpoint_service = checkpoint_service
        self.default_timeout_seconds = default_timeout_seconds
        self.execution_events: Dict[str, List[Dict[str, Any]]] = {}
        
        # Initialize specialized node handlers
        self.agent_handler = AgentNodeHandler(self)
        self.orchestrator_handler = OrchestratorNodeHandler(self)
        self.cli_handler = CLINodeHandler(self)

    def _emit_event(self, run_id: str, event_type: str, data: Dict[str, Any]) -> None:
        """Compatibility wrapper for node handlers."""
        emit_event(self, run_id, event_type, data)

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
            emit_event(self, run_id, "request.started", {"tenant_id": tenant_id, "workflow_id": workflow_id})
            current_node_id = self._find_entry_node(workflow)
            if not current_node_id:
                raise ValueError("No entry node found in workflow")

            state["current_node"] = current_node_id
            hop_count = 0

            while True:
                # 1. Check constraints (timeout, kill flag, stagnation)
                if should_stop(self, run_id, tenant_id, start_time, timeout, detector, state):
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
                    handle_exit_node(self, run_id, workflow_id, node, state)
                    break

                # 3. Resolve Node Input & Emit Start Event
                resolved_input = resolve_node_input(self, node, state)
                emit_event(self, run_id, "node.started", {
                    "node_id": current_node_id, 
                    "node_type": node_type, 
                    "input_data": resolved_input
                })

                # 4. Execute Node
                try:
                    node_result = self._dispatch_node_execution(run_id, node, state, workflow)
                    self._logger.debug(f"Node {current_node_id} result keys: {list(node_result.keys())}")
                    state["node_results"][current_node_id] = node_result
                    state["workflow_state"].update(node_result)

                    if self.checkpoint_service:
                        self.checkpoint_service.save_checkpoint(
                            tenant_id=tenant_id, 
                            thread_id=thread_id, 
                            workflow_state=state["workflow_state"]
                        )

                    emit_event(self, run_id, "node.completed", {
                        "node_id": current_node_id, 
                        "node_type": node_type, 
                        "result": node_result, 
                        "result_keys": list(node_result.keys())
                    })

                except Exception as e:
                    handle_node_failure(self, run_id, current_node_id, state, e)
                    # Stop execution if node is not optional (default)
                    node_config = node.get("config", {})
                    if not node_config.get("optional", False):
                        state["status"] = "failed"
                        break

                # 5. Route to Next Node
                next_node_id = route_by_edge(self, run_id, workflow, current_node_id, state)
                if next_node_id is None or next_node_id == current_node_id:
                    break
                current_node_id = next_node_id

            return finalize_execution(self, run_id, thread_id, tenant_id, workflow_id, state, start_time, hop_count)

        except Exception as e:
            return handle_execution_error(self, run_id, thread_id, tenant_id, workflow_id, state, e)

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
        elif node_type in {"cli_node", "bash"}:
            return self.cli_handler.execute(run_id, node, state, workflow)
        else:
            raise ValueError(f"Unknown node type: {node_type}")
