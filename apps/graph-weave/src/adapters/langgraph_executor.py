"""
Mock LangGraph Executor for MOCK phase.

Simulates real workflow execution by:
1. Loading the workflow JSON (nodes and edges)
2. Executing nodes sequentially
3. Evaluating edge conditions to route to next node
4. Calling MockAIProvider for agent_node execution
5. Tracking execution state through CheckpointStore
"""

from typing import Any, Dict, Optional, List
from datetime import datetime
import json
import copy

from .ai_provider import MockAIProvider


class MockLangGraphExecutor:
    """Executes workflows by traversing nodes and edges, calling AI provider for agent work."""

    def __init__(self, ai_provider: Optional[MockAIProvider] = None):
        self.ai_provider = ai_provider or MockAIProvider()
        self.execution_events: Dict[str, List[Dict[str, Any]]] = {}

    def execute(
        self,
        workflow: Dict[str, Any],
        input_data: Dict[str, Any],
        checkpoint_store: Any,
        cache: Any,
    ) -> Dict[str, Any]:
        """
        Execute a workflow from start to finish.

        Args:
            workflow: Workflow JSON (nodes, edges, limits)
            input_data: Initial input data for entry node
            checkpoint_store: CheckpointStore for persisting state
            cache: Cache adapter for storing intermediate results

        Returns:
            Final execution result with status, events, and final state
        """
        run_id = self.execution_events.get("__current_run_id__")
        if not run_id:
            raise ValueError("No current run_id set")

        self.execution_events[run_id] = []
        workflow_id = workflow.get("workflow_id", "unknown")
        tenant_id = workflow.get("metadata", {}).get("tenant_id", "unknown")

        state = {
            "input": input_data,
            "step": 0,
            "current_node": None,
            "node_results": {},
        }

        try:
            current_node_id = self._find_entry_node(workflow)
            state["current_node"] = current_node_id

            limits = workflow.get("limits", {})
            max_hops = limits.get("max_hops", 20)
            hop_count = 0

            while hop_count < max_hops:
                hop_count += 1

                if not current_node_id:
                    self._log_event(run_id, "error", "No current node to execute")
                    break

                node = self._find_node(workflow, current_node_id)
                if not node:
                    self._log_event(
                        run_id, "error", f"Node not found: {current_node_id}"
                    )
                    break

                state["step"] = hop_count
                node_type = node.get("type")

                if node_type == "exit":
                    self._log_event(
                        run_id, "node_exit", f"Reached exit node: {current_node_id}"
                    )
                    state["current_node"] = current_node_id
                    break

                if node_type == "agent_node":
                    result = self._execute_agent_node(
                        run_id, node, state, workflow, checkpoint_store
                    )
                    state["node_results"][current_node_id] = result
                elif node_type == "entry":
                    self._log_event(
                        run_id,
                        "node_entry",
                        f"Processing entry node: {current_node_id}",
                    )
                elif node_type == "branch":
                    self._log_event(
                        run_id,
                        "node_branch",
                        f"Processing branch node: {current_node_id}",
                    )

                next_node_id = self._route_by_edge(
                    run_id, workflow, current_node_id, state
                )

                if next_node_id == current_node_id:
                    self._log_event(run_id, "warning", "Self-loop detected, exiting")
                    break

                current_node_id = next_node_id

            state["current_node"] = current_node_id
            state["status"] = "completed"
            state["hop_count"] = hop_count

            return {
                "run_id": run_id,
                "status": "completed",
                "workflow_id": workflow_id,
                "tenant_id": tenant_id,
                "events": self.execution_events.get(run_id, []),
                "final_state": state,
                "hop_count": hop_count,
            }

        except Exception as e:
            self._log_event(run_id, "error", f"Execution failed: {str(e)}")
            return {
                "run_id": run_id,
                "status": "error",
                "workflow_id": workflow_id,
                "tenant_id": tenant_id,
                "events": self.execution_events.get(run_id, []),
                "error": str(e),
            }

    def set_current_run_id(self, run_id: str) -> None:
        self.execution_events["__current_run_id__"] = run_id

    def get_events(self, run_id: str) -> List[Dict[str, Any]]:
        return self.execution_events.get(run_id, [])

    def _find_entry_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        for node in workflow.get("nodes", []):
            if node.get("type") == "entry":
                return node.get("id")
        return None

    def _find_node(
        self, workflow: Dict[str, Any], node_id: str
    ) -> Optional[Dict[str, Any]]:
        for node in workflow.get("nodes", []):
            if node.get("id") == node_id:
                return node
        return None

    def _execute_agent_node(
        self,
        run_id: str,
        node: Dict[str, Any],
        state: Dict[str, Any],
        workflow: Dict[str, Any],
        checkpoint_store: Any,
    ) -> Dict[str, Any]:
        node_id = node.get("id")
        self._log_event(run_id, "node_execute", f"Executing agent node: {node_id}")

        config = node.get("config", {})
        system_prompt = config.get("system_prompt", "")
        user_prompt_template = config.get("user_prompt_template", "")

        user_prompt = self._interpolate_prompt(user_prompt_template, state)

        ai_response = self.ai_provider.call(
            system_prompt=system_prompt,
            user_prompt=user_prompt,
        )

        self._log_event(
            run_id,
            "agent_response",
            f"Agent {node_id} returned: {ai_response.get('content', '')[:100]}...",
        )

        try:
            result_data = json.loads(ai_response.get("content", "{}"))
        except json.JSONDecodeError:
            result_data = {"raw_response": ai_response.get("content", "")}

        result = {
            "node_id": node_id,
            "status": "completed",
            "result": result_data,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "tokens_used": ai_response.get("tokens_used", 0),
        }

        state["last_result"] = result_data

        return result

    def _route_by_edge(
        self,
        run_id: str,
        workflow: Dict[str, Any],
        current_node_id: str,
        state: Dict[str, Any],
    ) -> Optional[str]:
        edges = workflow.get("edges", [])
        matching_edges = [e for e in edges if e.get("from") == current_node_id]

        if not matching_edges:
            self._log_event(
                run_id,
                "warning",
                f"No outgoing edges from {current_node_id}, stopping",
            )
            exit_node_id = self._find_exit_node(workflow)
            return exit_node_id

        for edge in matching_edges:
            condition = edge.get("condition")
            if not condition or self._evaluate_condition(condition, state):
                next_node_id = edge.get("to")
                self._log_event(
                    run_id,
                    "edge_route",
                    f"Taking edge {current_node_id} -> {next_node_id}",
                )
                return next_node_id

        next_node_id = matching_edges[0].get("to")
        self._log_event(
            run_id,
            "edge_route",
            f"No condition matched, taking first edge to {next_node_id}",
        )
        return next_node_id

    def _find_exit_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        for node in workflow.get("nodes", []):
            if node.get("type") == "exit":
                return node.get("id")
        return None

    def _evaluate_condition(self, condition: str, state: Dict[str, Any]) -> bool:
        if not condition:
            return True

        try:
            if "==" in condition:
                left, right = condition.split("==")
                left = left.strip()
                right = right.strip().strip("'\"")

                state_value = self._get_state_value(left, state)
                return state_value == right

            if "!=" in condition:
                left, right = condition.split("!=")
                left = left.strip()
                right = right.strip().strip("'\"")

                state_value = self._get_state_value(left, state)
                return state_value != right

            if ">" in condition and ">=" not in condition:
                left, right = condition.split(">")
                left = left.strip()
                right = right.strip().strip("'\"")

                try:
                    state_value = float(self._get_state_value(left, state))
                    return state_value > float(right)
                except (ValueError, TypeError):
                    return False

            if "<" in condition and "<=" not in condition:
                left, right = condition.split("<")
                left = left.strip()
                right = right.strip().strip("'\"")

                try:
                    state_value = float(self._get_state_value(left, state))
                    return state_value < float(right)
                except (ValueError, TypeError):
                    return False

            return True

        except Exception:
            return False

    def _get_state_value(self, path: str, state: Dict[str, Any]) -> Any:
        if path.startswith("$."):
            path = path[2:]

        keys = path.split(".")
        current = state

        for key in keys:
            if isinstance(current, dict):
                current = current.get(key)
            else:
                return None

        return current

    def _interpolate_prompt(self, template: str, state: Dict[str, Any]) -> str:
        result = template
        for key, value in state.items():
            placeholder = f"{{{key}}}"
            if placeholder in result:
                result = result.replace(placeholder, str(value))
        return result

    def _log_event(self, run_id: str, event_type: str, message: str) -> None:
        if run_id not in self.execution_events:
            self.execution_events[run_id] = []

        event = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "type": event_type,
            "message": message,
        }
        self.execution_events[run_id].append(event)
