import json
from datetime import datetime
from typing import Any, Dict, Optional, List
from src.app_logging import get_logger
from .base_executor import BaseLangGraphExecutor
from .types import ExecutorState
from ..ai_provider import AIProviderFactory, LLMClient
from ..mcp_router import MCPRouter, ProviderConfigError

logger = get_logger(__name__)

class MockLangGraphExecutor(BaseLangGraphExecutor):
    """Executes workflows by traversing nodes and edges, calling AI provider for agent work."""

    def __init__(
        self,
        ai_provider: Optional[LLMClient] = None,
        mcp_router: Optional[MCPRouter] = None,
    ):
        self.ai_provider_factory = AIProviderFactory(ai_gateway_client=ai_provider)
        self.mcp_router = mcp_router or MCPRouter()
        self._current_run_id: Optional[str] = None
        self.execution_events: Dict[str, List[Dict[str, Any]]] = {}

    def execute(
        self,
        workflow: Dict[str, Any],
        input_data: Dict[str, Any],
        checkpoint_store: Any,
        cache: Any,
    ) -> Dict[str, Any]:
        run_id = self._current_run_id
        if not run_id:
            raise ValueError("No current run_id set")

        self.execution_events[run_id] = []
        workflow_id = workflow.get("workflow_id", "unknown")
        tenant_id = workflow.get("metadata", {}).get("tenant_id", "unknown")

        workflow = self._normalize_workflow(workflow)

        state: ExecutorState = {
            "input": input_data,
            "step": 0,
            "current_node": None,
            "node_results": {},
            "workflow_state": dict(input_data),
            "status": None,
            "hop_count": 0,
            "last_result": None,
            "errors": []
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

                if node_type in {"agent_node", "agent"}:
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
                "hop_count": hop_count,
                "events": self.execution_events.get(run_id, []),
                "final_state": state["node_results"],
                "workflow_state": state,
            }

        except Exception as e:
            self._log_event(run_id, "error", f"Execution failed: {str(e)}")
            return {
                "run_id": run_id,
                "status": "failed",
                "workflow_id": workflow_id,
                "tenant_id": tenant_id,
                "hop_count": hop_count,
                "error": str(e),
                "events": self.execution_events.get(run_id, []),
                "final_state": state["node_results"],
                "workflow_state": state,
            }

    def set_current_run_id(self, run_id: str) -> None:
        self._current_run_id = run_id

    def get_events(self, run_id: str) -> List[Dict[str, Any]]:
        return self.execution_events.get(run_id, [])

    def _execute_agent_node(
        self,
        run_id: str,
        node: Dict[str, Any],
        state: ExecutorState,
        workflow: Dict[str, Any],
        checkpoint_store: Any,
    ) -> Dict[str, Any]:
        node_id = node.get("id")
        self._log_event(run_id, "node_execute", f"Executing agent node: {node_id}")

        config = node.get("config", {})
        system_prompt = config.get("system_prompt", "")
        user_prompt_template = config.get("user_prompt_template", "")

        input_mapping = config.get("input_mapping", {})
        agent_input_context = {}
        if input_mapping:
            for key, path in input_mapping.items():
                agent_input_context[key] = self._get_state_value(path, state)

        user_prompt = self._interpolate_prompt(user_prompt_template, state, local_context=agent_input_context)

        provider = config.get("provider", "github-copilot")
        model = config.get("model")
        temperature = config.get("temperature", 0.7)
        max_tokens = config.get("max_tokens", 2000)
        allowed_tools = config.get("tools")

        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt},
        ]

        try:
            provider_client = self.ai_provider_factory.get_provider_client(
                provider or "openai", model
            )
            response = provider_client.chat_completion(
                messages=messages,
                provider=provider or "openai",
                model=model or "gpt-4.1",
                temperature=temperature,
                max_tokens=max_tokens,
                tools=allowed_tools,
            )

            content = response["choices"][0]["message"].get("content", "")
            tokens_used = response.get("usage", {}).get("total_tokens", 0)

            ai_response = {
                "content": content,
                "tokens_used": tokens_used,
                "model": response.get("model", model),
            }
        except ProviderConfigError as e:
            self._log_event(run_id, "error", f"Provider config error: {str(e)}")
            return {
                "node_id": node_id,
                "status": "failed",
                "result": {"error": str(e)},
                "timestamp": datetime.utcnow().isoformat() + "Z",
                "tokens_used": 0,
            }

        content = str(ai_response.get("content", ""))
        tool_calls = self.mcp_router.parse_tool_calls(content, allowed_tools)
        
        if tool_calls:
            self._log_event(
                run_id,
                "agent_tool_calls",
                f"Agent {node_id} made {len(tool_calls)} tool calls",
            )

        self._log_event(
            run_id,
            "agent_response",
            f"Agent {node_id} returned: {content[:100]}...",
        )

        try:
            result_data = json.loads(content)
        except json.JSONDecodeError:
            result_data = {"raw_response": content}

        result = {
            "node_id": node_id,
            "status": "completed",
            "result": result_data,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "tokens_used": ai_response.get("tokens_used", 0),
            "tool_calls": tool_calls,
        }

        output_key = config.get("output_key")
        if output_key:
            result[output_key] = result_data

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
            return self._find_exit_node(workflow)

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

    def _log_event(self, run_id: str, event_type: str, message: str) -> None:
        if run_id not in self.execution_events:
            self.execution_events[run_id] = []

        event = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "type": event_type,
            "message": message,
        }
        self.execution_events[run_id].append(event)
