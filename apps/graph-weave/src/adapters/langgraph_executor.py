"""
Real LangGraph Executor with event emission and stagnation detection.

Implements:
1. StateGraph execution loop with node traversal
2. Event emission (node.started, node.completed, node.failed)
3. Per-node configuration (provider, model, temperature, max_tokens, tools)
4. Stagnation detection (max 20 hops)
5. Timeout enforcement (300s)
6. Circuit breaker kill flag checking
7. State accumulation (merge all node outputs)
8. Error handling and recovery
"""

from typing import Any, Dict, Optional, List, TypedDict, cast, Mapping
from datetime import datetime, timezone
import json
import time
import logging
from src.app_logging import get_logger

from .ai_gateway_adapter import AIGatewayClient
from .mcp_router import MCPRouter, ProviderConfigError, ToolExecutionError, LLMClient
from .stagnation_detector import StagnationDetector
from .redis_circuit_breaker import NamespacedRedisClient

logger = get_logger(__name__)


class ExecutorState(TypedDict):
    input: Dict[str, Any]
    step: int
    current_node: Optional[str]
    node_results: Dict[str, Dict[str, Any]]
    status: Optional[str]
    hop_count: int
    last_result: Optional[Dict[str, Any]]


class MockLangGraphExecutor:
    """Executes workflows by traversing nodes and edges, calling AI provider for agent work."""

    def __init__(
        self,
        ai_provider: Optional[LLMClient] = None,
        mcp_router: Optional[MCPRouter] = None,
    ):
        self.ai_provider = ai_provider
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
            "status": None,
            "hop_count": 0,
            "last_result": None,
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

    def _normalize_workflow(self, workflow: Dict[str, Any]) -> Dict[str, Any]:
        """
        Normalize workflow to ensure nodes, edges, and limits are at top level.
        Handles both nested (definition) and flat structures.
        """
        if "nodes" in workflow:
            return workflow

        definition = workflow.get("definition", {})
        if "nodes" in definition:
            workflow["nodes"] = definition.get("nodes", [])
            workflow["edges"] = definition.get("edges", [])
            workflow["limits"] = definition.get("limits", workflow.get("limits", {}))
            workflow["entry_point"] = definition.get("entry_point")
            workflow["exit_point"] = definition.get("exit_point")

        return workflow

    def get_events(self, run_id: str) -> List[Dict[str, Any]]:
        return self.execution_events.get(run_id, [])

    def _find_entry_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        definition = workflow.get("definition", {})
        entry_point = definition.get("entry_point")
        if entry_point:
            return entry_point
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
        state: ExecutorState,
        workflow: Dict[str, Any],
        checkpoint_store: Any,
    ) -> Dict[str, Any]:
        node_id = node.get("id")
        self._log_event(run_id, "node_execute", f"Executing agent node: {node_id}")

        config = node.get("config", {})
        system_prompt = config.get("system_prompt", "")
        user_prompt_template = config.get("user_prompt_template", "")

        user_prompt = self._interpolate_prompt(user_prompt_template, state)

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
            # Always route through Gateway via mcp_router
            provider_client = self.mcp_router.get_provider_client(provider or "openai", model)
            response = provider_client.chat_completion(
                messages=messages,
                provider=provider or "openai",
                model=model or "gpt-4.1",
                temperature=temperature,
                max_tokens=max_tokens,
            )
            
            # Extract content from OpenAI-compatible structure
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

        state["last_result"] = result_data

        return result

    def _route_by_edge(
        self,
        run_id: str,
        workflow: Dict[str, Any],
        current_node_id: str,
        state: Mapping[str, Any],
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
        definition = workflow.get("definition", {})
        exit_point = definition.get("exit_point")
        if exit_point:
            return exit_point
        for node in workflow.get("nodes", []):
            if node.get("type") == "exit":
                return node.get("id")
        return None

    def _evaluate_condition(self, condition: str, state: Mapping[str, Any]) -> bool:
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

    def _get_state_value(self, path: str, state: Mapping[str, Any]) -> Any:
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

    def _interpolate_prompt(self, template: str, state: Mapping[str, Any]) -> str:
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


class RealLangGraphExecutor:
    """
    Real LangGraph executor with full event emission, stagnation detection,
    timeout enforcement, and circuit breaker support.

    Implements:
    - StateGraph execution loop with node traversal
    - Event emission after each node (node.started, node.completed, node.failed)
    - Per-node configuration (provider, model, temperature, max_tokens, tools)
    - Stagnation detection (prevents infinite loops)
    - Timeout enforcement (configurable, default 300s)
    - Circuit breaker (checks kill flag after each node)
    - State accumulation (merges all node outputs)
    - Error handling and graceful failure
    """

    _logger = get_logger(__name__)

    def __init__(
        self,
        ai_provider: Optional[LLMClient] = None,
        mcp_router: Optional[MCPRouter] = None,
        redis_client: Optional[NamespacedRedisClient] = None,
        default_timeout_seconds: int = 300,
    ):
        """
        Initialize RealLangGraphExecutor.

        Args:
            ai_provider: LLM provider client (optional, usually provided via mcp_router)
            mcp_router: MCP router for tool/provider routing
            redis_client: Redis client for circuit breaker
            default_timeout_seconds: Default execution timeout in seconds (default 300)
        """
        self.ai_provider = ai_provider
        self.mcp_router = mcp_router or MCPRouter()
        self.redis_client = redis_client
        self.default_timeout_seconds = default_timeout_seconds
        self.execution_events: Dict[str, List[Dict[str, Any]]] = {}

    def execute(
        self,
        run_id: str,
        thread_id: str,
        tenant_id: str,
        workflow: Dict[str, Any],
        input_data: Dict[str, Any],
        timeout_seconds: Optional[int] = None,
    ) -> Dict[str, Any]:
        """
        Execute a real LangGraph workflow with full event emission and safeguards.

        Args:
            run_id: Unique execution ID
            thread_id: Thread ID for checkpoint storage
            tenant_id: Tenant ID for multi-tenancy
            workflow: Workflow JSON definition
            input_data: Initial input data
            timeout_seconds: Optional timeout override

        Returns:
            Execution result with status, events, final state, and metadata
        """
        workflow_id = workflow.get("workflow_id", "unknown")
        timeout = timeout_seconds or self.default_timeout_seconds
        start_time = time.monotonic()
        detector = StagnationDetector(
            max_hops=workflow.get("limits", {}).get("max_hops", 20)
        )

        self.execution_events[run_id] = []

        # Normalize workflow format
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
            self._emit_event(
                run_id,
                "request.started",
                {"tenant_id": tenant_id, "workflow_id": workflow_id},
            )

            current_node_id = self._find_entry_node(workflow)
            if not current_node_id:
                raise ValueError("No entry node found in workflow")

            state["current_node"] = current_node_id
            hop_count = 0

            while True:
                elapsed = time.monotonic() - start_time
                if elapsed > timeout:
                    self._emit_event(
                        run_id, "execution.timeout", {"elapsed_seconds": elapsed}
                    )
                    state["status"] = "timeout"
                    break

                if self._check_kill_flag(run_id, tenant_id):
                    self._emit_event(
                        run_id, "execution.killed", {"elapsed_seconds": elapsed}
                    )
                    state["status"] = "killed"
                    break

                if detector.is_stagnated():
                    self._emit_event(
                        run_id,
                        "execution.stagnated",
                        {
                            "visits": detector.get_summary(),
                            "max_hops": detector.max_hops,
                        },
                    )
                    state["status"] = "stagnated"
                    break

                hop_count += 1
                detector.track_node_visit(current_node_id)

                node = self._find_node(workflow, current_node_id)
                if not node:
                    raise ValueError(f"Node not found: {current_node_id}")

                node_type = node.get("type")
                state["current_node"] = current_node_id
                state["step"] = hop_count

                if node_type == "exit":
                    self._emit_event(
                        run_id,
                        "node.completed",
                        {"node_id": current_node_id, "node_type": "exit"},
                    )
                    break

                self._emit_event(
                    run_id,
                    "node.started",
                    {"node_id": current_node_id, "node_type": node_type},
                )

                try:
                    if node_type == "agent_node":
                        node_result = self._execute_agent_node(
                            run_id, node, state, workflow
                        )
                    elif node_type == "entry":
                        node_result = self._execute_entry_node(state, node)
                    elif node_type == "branch":
                        node_result = self._execute_branch_node(state, node)
                    else:
                        raise ValueError(f"Unknown node type: {node_type}")

                    state["node_results"][current_node_id] = node_result
                    state["workflow_state"].update(node_result)

                    self._emit_event(
                        run_id,
                        "node.completed",
                        {
                            "node_id": current_node_id,
                            "node_type": node_type,
                            "result_keys": list(node_result.keys()),
                        },
                    )

                except Exception as e:
                    self._emit_event(
                        run_id,
                        "node.failed",
                        {"node_id": current_node_id, "error": str(e)},
                    )
                    state["errors"].append(
                        {
                            "node_id": current_node_id,
                            "error": str(e),
                            "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
                        }
                    )

                next_node_id = self._route_by_edge(workflow, current_node_id, state)
                if next_node_id is None or next_node_id == current_node_id:
                    break

                current_node_id = next_node_id

            state["status"] = state.get("status") or "completed"
            state["hop_count"] = hop_count

            self._emit_event(
                run_id,
                "request.completed",
                {
                    "status": state["status"],
                    "hop_count": hop_count,
                    "elapsed_seconds": time.monotonic() - start_time,
                },
            )

            return {
                "run_id": run_id,
                "thread_id": thread_id,
                "tenant_id": tenant_id,
                "workflow_id": workflow_id,
                "status": state["status"],
                "hop_count": hop_count,
                "events": self.execution_events.get(run_id, []),
                "final_state": state,
                "workflow_state": state["workflow_state"],
                "elapsed_seconds": time.monotonic() - start_time,
            }

        except Exception as e:
            self._logger.exception(f"Execution error: {e}")
            state["status"] = "failed"
            self._emit_event(run_id, "request.failed", {"error": str(e)})
            return {
                "run_id": run_id,
                "thread_id": thread_id,
                "tenant_id": tenant_id,
                "workflow_id": workflow_id,
                "status": "failed",
                "error": str(e),
                "events": self.execution_events.get(run_id, []),
                "final_state": state,
            }

    def _execute_entry_node(
        self, state: Dict[str, Any], node: Dict[str, Any]
    ) -> Dict[str, Any]:
        return dict(state.get("input", {}))

    def _execute_branch_node(
        self, state: Dict[str, Any], node: Dict[str, Any]
    ) -> Dict[str, Any]:
        return {"branch_result": "true"}

    def _execute_agent_node(
        self,
        run_id: str,
        node: Dict[str, Any],
        state: Dict[str, Any],
        workflow: Dict[str, Any],
    ) -> Dict[str, Any]:
        node_id = node.get("id")
        config = node.get("config", {})

        system_prompt = config.get("system_prompt", "You are a helpful assistant.")
        user_prompt_template = config.get("user_prompt_template", "")
        user_prompt = self._interpolate_prompt(user_prompt_template, state)

        provider = config.get("provider", "github-copilot")
        model = config.get("model", "gpt-4")
        temperature = config.get("temperature", 0.7)
        max_tokens = config.get("max_tokens", 2000)
        allowed_tools = config.get("tools", [])

        try:
            # Use Gateway client for unified interaction
            client = cast(AIGatewayClient, self.mcp_router.get_provider_client(provider, model))
            
            # Initial state for the tool loop
            messages = [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ]
            
            # Extract tool definitions from MCPRouter
            tools = self.mcp_router.get_tool_definitions(allowed_tools)
            
            self._emit_event(run_id, "agent.request", {
                "node_id": node_id,
                "provider": provider,
                "model": model,
                "tool_count": len(tools)
            })

            # The "Ping-Pong" Tool Loop
            max_turns = 5
            turns = 0
            final_content = ""
            total_tokens = 0
            
            while turns < max_turns:
                turns += 1
                
                response = client.chat_completion(
                    messages=messages,
                    provider=provider,
                    model=model,
                    tools=tools if tools else None,
                    temperature=temperature,
                    max_tokens=max_tokens
                )
                
                total_tokens += response.get("usage", {}).get("total_tokens", 0)
                choice = response["choices"][0]
                message = choice["message"]
                
                # Check for tool_calls in the structured response
                tool_calls = message.get("tool_calls")
                
                if not tool_calls:
                    final_content = message.get("content", "")
                    break
                
                # Add assistant message (with tool_calls) to history
                messages.append(message)
                
                self._emit_event(run_id, "agent.tool_calls", {
                    "node_id": node_id,
                    "count": len(tool_calls)
                })

                # Execute all requested tools
                for tool_call in tool_calls:
                    tool_id = tool_call["id"]
                    tool_name = tool_call["function"]["name"]
                    tool_args = json.loads(tool_call["function"]["arguments"])
                    
                    try:
                        self._emit_event(run_id, "tool.started", {"tool": tool_name, "id": tool_id})
                        tool_result = self.mcp_router.execute_tool(tool_name, tool_args)
                        
                        # Add tool result to history
                        messages.append({
                            "role": "tool",
                            "tool_call_id": tool_id,
                            "content": json.dumps(tool_result)
                        })
                        self._emit_event(run_id, "tool.completed", {"tool": tool_name, "id": tool_id})
                    except Exception as e:
                        self._logger.error(f"Tool {tool_name} failed: {e}")
                        messages.append({
                            "role": "tool",
                            "tool_call_id": tool_id,
                            "content": json.dumps({"error": str(e)})
                        })
                        self._emit_event(run_id, "tool.failed", {"tool": tool_name, "id": tool_id, "error": str(e)})

            # Finalize node results
            try:
                result_data = json.loads(final_content)
            except json.JSONDecodeError:
                result_data = {"raw_response": final_content}

            return {
                f"{node_id}_output": result_data,
                f"{node_id}_status": "completed",
                "tokens_used": total_tokens,
                "turns": turns
            }

        except ProviderConfigError as e:
            raise ValueError(f"Provider configuration error: {e}")
        except Exception as e:
            raise ValueError(f"Agent node execution failed: {e}")

    def _route_by_edge(
        self,
        workflow: Dict[str, Any],
        current_node_id: str,
        state: Dict[str, Any],
    ) -> Optional[str]:
        edges = workflow.get("edges", [])
        matching_edges = [e for e in edges if e.get("from") == current_node_id]

        if not matching_edges:
            return self._find_exit_node(workflow)

        for edge in matching_edges:
            condition = edge.get("condition")
            if not condition or self._evaluate_condition(condition, state):
                return edge.get("to")

        return matching_edges[0].get("to") if matching_edges else None

    def _find_entry_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        for node in workflow.get("nodes", []):
            if node.get("type") == "entry":
                return node.get("id")
        return None

    def _find_exit_node(self, workflow: Dict[str, Any]) -> Optional[str]:
        for node in workflow.get("nodes", []):
            if node.get("type") == "exit":
                return node.get("id")
        return None

    def _find_node(
        self, workflow: Dict[str, Any], node_id: str
    ) -> Optional[Dict[str, Any]]:
        for node in workflow.get("nodes", []):
            if node.get("id") == node_id:
                return node
        return None

    def _normalize_workflow(self, workflow: Dict[str, Any]) -> Dict[str, Any]:
        """Normalize workflow format to ensure 'nodes' and 'edges' are available."""
        if "definition" in workflow:
            definition = workflow["definition"]
            # Create a copy to avoid mutating original
            normalized = workflow.copy()
            normalized["nodes"] = definition.get("nodes", [])
            normalized["edges"] = definition.get("edges", [])
            normalized["entry_point"] = definition.get("entry_point")
            normalized["exit_point"] = definition.get("exit_point")
            return normalized
        return workflow

    def _evaluate_condition(self, condition: str, state: Dict[str, Any]) -> bool:
        if not condition:
            return True
        try:
            for op in ["==", "!=", ">=", "<=", ">", "<"]:
                if op in condition:
                    left, right = condition.split(op, 1)
                    left_val = self._get_state_value(left.strip(), state)
                    right_val = right.strip().strip("'\"")
                    if op == "==":
                        return left_val == right_val
                    elif op == "!=":
                        return left_val != right_val
                    elif op == ">=":
                        return float(left_val) >= float(right_val)
                    elif op == "<=":
                        return float(left_val) <= float(right_val)
                    elif op == ">":
                        return float(left_val) > float(right_val)
                    elif op == "<":
                        return float(left_val) < float(right_val)
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

    def _check_kill_flag(self, run_id: str, tenant_id: str) -> bool:
        if not self.redis_client:
            return False
        try:
            kill_key = f"kill:{tenant_id}:{run_id}"
            return self.redis_client.exists(kill_key)
        except Exception:
            return False

    def _emit_event(self, run_id: str, event_type: str, data: Dict[str, Any]) -> None:
        if run_id not in self.execution_events:
            self.execution_events[run_id] = []

        event = {
            "type": event_type,
            "run_id": run_id,
            "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            "data": data,
        }
        self.execution_events[run_id].append(event)

        if self.redis_client:
            try:
                event_key = f"event:{run_id}"
                self.redis_client.rpush(event_key, json.dumps(event))
            except Exception as e:
                self._logger.warning(f"Failed to persist event to Redis: {e}")
