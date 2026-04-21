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
from src.models import OrchestratorConfig
from src.modules.orchestrator_react import OrchestratorReAct

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

        # Resolve prompts using the FULL state + the local mapped context
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
            # Always route through Gateway via mcp_router
            provider_client = self.mcp_router.get_provider_client(
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

        # GW-FEAT-GEN-004: Ensure result is also packed into output_key for state propagation
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
        """
        Extract a value from the state based on a JSONPath-like string.
        Logic order:
        1. Node results (if path starts with a node_id, e.g., 'entry.intent')
        2. workflow_state (the application data pool)
        3. Root state (metadata like 'step', 'hop_count')
        """
        if not path:
            return None
        if path.startswith("$."):
            path = path[2:]

        keys = path.split(".")
        first_key = keys[0]

        # 1. Try finding in node_results first (e.g. $.entry.query)
        node_results = state.get("node_results", {})
        if first_key in node_results:
            current = node_results[first_key]
            if len(keys) == 1:
                return current
            
            found = True
            for key in keys[1:]:
                if isinstance(current, dict) and key in current:
                    current = current.get(key)
                else:
                    found = False
                    break
            if found:
                return current

        # 2. Try finding in workflow_state next (flat pool)
        workflow_state = state.get("workflow_state", {})
        if first_key in workflow_state:
            current = workflow_state
            found = True
            for key in keys:
                if isinstance(current, dict) and key in current:
                    current = current.get(key)
                else:
                    found = False
                    break
            if found:
                return current

        # 3. Fallback: Root state
        current = state
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current.get(key)
            else:
                return None

        return current

    def _interpolate_prompt(
        self,
        template: str,
        state: Mapping[str, Any],
        local_context: Optional[Dict[str, Any]] = None,
    ) -> str:
        """
        Interpolate variables into the prompt template.
        Supports:
        1. Local context (from input_mapping) - HIGHEST PRIORITY
        2. Root-level state keys like {step}, {hop_count}
        3. workflow_state keys like {query}, {subquestions}
        4. JSONPath-like keys starting with $.
        
        Includes 'Smart Content Extraction': if a value is a dict with 'raw_response' or 'result',
        it uses the content of that key instead of the full JSON object.
        """
        import re
        result = template
        
        # Support both {var} and {{var}}
        # This regex matches the content inside any number of curly braces
        placeholders = re.findall(r"\{+([^{}]+)\}+", template)
        
        # Merge context: local_context > workflow_state > root state
        context = dict(state.get("workflow_state", {}))
        context.update({k: v for k, v in state.items() if k != "workflow_state"})
        if local_context:
            context.update(local_context)
        
        for p in placeholders:
            val = None
            if p.startswith("$."):
                val = self._get_state_value(p, state)
            else:
                # Try direct lookup in merged context
                val = context.get(p)
            
            if val is not None:
                # GW-FEAT-GEN-005: Smart Content Extraction
                # If we have a dict that looks like an agent result, extract the content
                if isinstance(val, dict):
                    if "raw_response" in val:
                        val = val["raw_response"]
                    elif "result" in val:
                        val = val["result"]
                
                # Convert complex objects to pretty JSON strings if they are still dicts/lists
                if isinstance(val, (dict, list)):
                    val_str = json.dumps(val, indent=2)
                else:
                    val_str = str(val)
                
                # Replace ALL variations: {{p}} and {p}
                result = result.replace(f"{{{{{p}}}}}", val_str)
                result = result.replace(f"{{{p}}}", val_str)
                
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
                    # GW-FEAT-GEN-003: Apply output_mapping if present
                    config = node.get("config", {})
                    mapping = config.get("output_mapping", {})
                    if mapping:
                        final_output = {}
                        for key, path in mapping.items():
                            val = self._get_state_value(path, state)
                            final_output[key] = val
                        # Finalize the workflow_state with the mapping results
                        state["workflow_state"] = final_output
                    
                    self._emit_event(
                        run_id,
                        "workflow.completed",
                        {
                            "run_id": run_id,
                            "workflow_id": workflow_id,
                            "status": "completed",
                            "final_state_keys": list(state["workflow_state"].keys()),
                        },
                    )
                    state["status"] = "completed"
                    break

                # GW-FEAT-GEN-002: Capture resolved input for the node
                resolved_input = {}
                if node_type in {"agent_node", "agent", "orchestrator"}:
                    config = node.get("config", {})
                    input_mapping = config.get("input_mapping", {})
                    if input_mapping:
                        for key, path in input_mapping.items():
                            resolved_input[key] = self._get_state_value(path, state)
                elif node_type == "entry":
                    resolved_input = state.get("input", {})

                self._emit_event(
                    run_id,
                    "node.started",
                    {
                        "node_id": current_node_id,
                        "node_type": node_type,
                        "input_data": resolved_input
                    },
                )

                try:
                    if node_type in {"agent_node", "agent"}:
                        node_result = self._execute_agent_node(
                            run_id, node, state, workflow
                        )
                    elif node_type == "entry":
                        node_result = self._execute_entry_node(state, node)
                    elif node_type == "branch":
                        node_result = self._execute_branch_node(state, node)
                    elif node_type == "guardrail":
                        # Basic passthrough for guardrails
                        self._emit_event(
                            run_id,
                            "node.completed",
                            {"node_id": current_node_id, "node_type": "guardrail", "status": "passthrough"},
                        )
                        node_result = {"guardrail_passed": True, "node_id": current_node_id}
                    elif node_type == "skill_loader":
                        # Basic passthrough for skill_loader
                        self._emit_event(
                            run_id,
                            "node.completed",
                            {"node_id": current_node_id, "node_type": "skill_loader", "status": "passthrough"},
                        )
                        node_result = {"skills_loaded": True, "node_id": current_node_id}
                    elif node_type == "orchestrator":
                        node_result = self._execute_orchestrator_node(
                            run_id, node, state, workflow
                        )
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
                            "result": node_result,
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
                            "timestamp": datetime.now(timezone.utc)
                            .isoformat()
                            .replace("+00:00", "Z"),
                        }
                    )

                next_node_id = self._route_by_edge(run_id, workflow, current_node_id, state)
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
                "workflow_state": state.get("workflow_state", {}),
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
        
        # Support input_mapping if present, otherwise use workflow_state
        input_mapping = config.get("input_mapping", {})
        if input_mapping:
            # If mapping is specified, slice the state
            agent_input_context = {}
            for key, path in input_mapping.items():
                agent_input_context[key] = self._get_state_value(path, state)
        else:
            # Fallback to entire workflow_state
            agent_input_context = dict(state["workflow_state"])
        
        # Resolve prompts using the FULL state + the local mapped context
        # This ensures {user_query} from input_mapping is used instead of just global {query}
        user_prompt = self._interpolate_prompt(user_prompt_template, state, local_context=agent_input_context)
        
        provider = config.get("provider", "github-copilot")
        model = config.get("model", "gpt-4.1")
        temperature = config.get("temperature", 0.7)
        max_tokens = config.get("max_tokens", 2000)
        allowed_tools = config.get("tools", [])

        try:
            # Use Gateway client for unified interaction
            client = cast(
                AIGatewayClient, self.mcp_router.get_provider_client(provider, model)
            )

            # Initial state for the tool loop
            messages = [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ]

            # Extract tool definitions from MCPRouter
            tools = self.mcp_router.get_tool_definitions(allowed_tools)

            # Use agent_input_context for the actual agent call
            mcp_context = agent_input_context

            self._emit_event(
                run_id,
                "agent.request",
                {
                    "node_id": node_id,
                    "provider": provider,
                    "model": model,
                    "tool_count": len(tools),
                },
            )

            # The "Ping-Pong" Tool Loop
            max_turns = 5
            turns = 0
            final_content = ""
            total_tokens = 0
            all_tool_calls = []

            while turns < max_turns:
                turns += 1

                response = client.chat_completion(
                    messages=messages,
                    provider=provider,
                    model=model,
                    tools=tools if tools else None,
                    temperature=temperature,
                    max_tokens=max_tokens,
                )

                total_tokens += response.get("usage", {}).get("total_tokens", 0)
                choice = response["choices"][0]
                message = choice["message"]

                # Check for tool_calls in the structured response
                tool_calls = message.get("tool_calls")

                if not tool_calls:
                    final_content = self._clean_filler(message.get("content", ""))
                    break
                
                # Track tool calls for the final result
                all_tool_calls.extend(tool_calls)

                # Add assistant message (with tool_calls) to history
                messages.append(message)

                self._emit_event(
                    run_id,
                    "agent.tool_calls",
                    {"node_id": node_id, "count": len(tool_calls)},
                )

                # Execute all requested tools
                for tool_call in tool_calls:
                    tool_id = tool_call["id"]
                    tool_name = tool_call["function"]["name"]
                    tool_args = json.loads(tool_call["function"]["arguments"])

                    try:
                        self._emit_event(
                            run_id, "tool.started", {"tool": tool_name, "id": tool_id}
                        )
                        tool_result = self.mcp_router.execute_tool(tool_name, tool_args)

                        # Add tool result to history
                        messages.append(
                            {
                                "role": "tool",
                                "tool_call_id": tool_id,
                                "content": json.dumps(tool_result),
                            }
                        )
                        self._emit_event(
                            run_id, "tool.completed", {"tool": tool_name, "id": tool_id}
                        )
                    except Exception as e:
                        self._logger.error(f"Tool {tool_name} failed: {e}")
                        messages.append(
                            {
                                "role": "tool",
                                "tool_call_id": tool_id,
                                "content": json.dumps({"error": str(e)}),
                            }
                        )
                        self._emit_event(
                            run_id,
                            "tool.failed",
                            {"tool": tool_name, "id": tool_id, "error": str(e)},
                        )

            # Finalize node results
            try:
                result_data = json.loads(final_content)
            except json.JSONDecodeError:
                result_data = {"raw_response": final_content}

            # Support output_key: wrap result if specified
            output_key = config.get("output_key")
            actual_result = {output_key: result_data} if output_key else result_data

            node_result_payload = {
                "node_id": node_id,
                "status": "completed",
                "result": result_data,
                "tool_calls": all_tool_calls,
                f"{node_id}_output": result_data,
                f"{node_id}_status": "completed",
                "tokens_used": total_tokens,
                "turns": turns,
            }
            
            if isinstance(actual_result, dict):
                node_result_payload.update(actual_result)
                
            return node_result_payload

        except ProviderConfigError as e:
            raise ValueError(f"Provider configuration error: {e}")
        except Exception as e:
            raise ValueError(f"Agent node execution failed: {e}")

    def _execute_orchestrator_node(
        self,
        run_id: str,
        node: Dict[str, Any],
        state: Dict[str, Any],
        workflow: Dict[str, Any],
    ) -> Dict[str, Any]:
        """
        Delegate to OrchestratorReAct for the internal ReAct loop.

        The result is a structured dict with 'orchestrator_trace' and 'final_result'
        which the executor merges into workflow_state via the normal node-result path.
        """
        node_id = node.get("id")
        raw_config = node.get("config", {})

        try:
            config = OrchestratorConfig(**raw_config)
        except Exception as e:
            raise ValueError(f"Invalid orchestrator config on node '{node_id}': {e}")

        # 1. Resolve input_mapping if present, otherwise use workflow_state
        input_mapping = config.input_mapping
        if input_mapping:
            # If mapping is specified, slice the state
            orchestrator_context = {}
            for key, path in input_mapping.items():
                orchestrator_context[key] = self._get_state_value(path, state)
        else:
            # Fallback to entire workflow_state
            orchestrator_context = dict(state.get("workflow_state", {}))

        # 2. Resolve prompts using the FULL state (allowing {step}, {current_node}, or {$.plan})
        config.system_prompt = self._interpolate_prompt(config.system_prompt, state)
        
        user_prompt = None
        if config.user_prompt_template:
            user_prompt = self._interpolate_prompt(config.user_prompt_template, state)

        react = OrchestratorReAct(
            client=self.mcp_router.get_provider_client(
                config.provider or "github-copilot",
                config.model or "gpt-4.1",
            ),
            mcp_router=self.mcp_router,
            emit=lambda etype, data: self._emit_event(run_id, etype, data),
        )

        result = react.run(
            run_id=run_id,
            node_id=node_id,
            config=config,
            workflow_state=orchestrator_context,
            user_prompt=user_prompt,
        )

        # Expose final_result at a top-level key so downstream edge conditions
        # can reference it as $.orchestrator_result
        return {
            **result,
            "orchestrator_result": result.get("final_result", {}),
            "node_id": node_id,
        }

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
            self._emit_event(
                run_id,
                "edge_route",
                {
                    "from": current_node_id,
                    "to": target_node_id,
                    "condition": routing_edge.get("condition") if routing_edge else None,
                },
            )

        return target_node_id

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
        """
        Extract a value from the state based on a JSONPath-like string.
        Logic order:
        1. Node results (if path starts with a node_id, e.g., 'entry.intent')
        2. workflow_state (the application data pool)
        3. Root state (metadata like 'step', 'hop_count')
        """
        if not path:
            return None
        if path.startswith("$."):
            path = path[2:]

        keys = path.split(".")
        first_key = keys[0]

        # 1. Try finding in node_results first (e.g. $.entry.query)
        node_results = state.get("node_results", {})
        if first_key in node_results:
            current = node_results[first_key]
            if len(keys) == 1:
                return current
            
            found = True
            for key in keys[1:]:
                if isinstance(current, dict) and key in current:
                    current = current.get(key)
                else:
                    found = False
                    break
            if found:
                return current

        # 2. Try finding in workflow_state next (flat pool)
        workflow_state = state.get("workflow_state", {})
        if first_key in workflow_state:
            current = workflow_state
            found = True
            for key in keys:
                if isinstance(current, dict) and key in current:
                    current = current.get(key)
                else:
                    found = False
                    break
            if found:
                return current

        # 3. Fallback: Root state
        current = state
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current.get(key)
            else:
                return None

        return current

    def _interpolate_prompt(
        self,
        template: str,
        state: Dict[str, Any],
        local_context: Optional[Dict[str, Any]] = None,
    ) -> str:
        """
        Interpolate variables into the prompt template.
        Supports:
        1. Local context (from input_mapping) - HIGHEST PRIORITY
        2. Root-level state keys like {step}, {hop_count}
        3. workflow_state keys like {query}, {subquestions}
        4. JSONPath-like keys starting with $.

        Includes 'Smart Content Extraction': if a value is a dict with 'raw_response' or 'result',
        it uses the content of that key instead of the full JSON object.
        """
        if not template:
            return ""

        import re
        result = template

        # Support both {var} and {{var}}
        placeholders = re.findall(r"\{+([^{}]+)\}+", template)

        # Merge context: local_context > workflow_state > root state
        context = dict(state.get("workflow_state", {}))
        context.update({k: v for k, v in state.items() if k != "workflow_state"})
        if local_context:
            context.update(local_context)

        for p in placeholders:
            val = None
            if p.startswith("$."):
                val = self._get_state_value(p, state)
            else:
                # Try direct lookup in merged context
                val = context.get(p)

            if val is not None:
                # GW-FEAT-GEN-005: Smart Content Extraction
                if isinstance(val, dict):
                    if "raw_response" in val:
                        val = val["raw_response"]
                    elif "result" in val:
                        val = val["result"]

                # Convert complex objects to pretty JSON strings if they are still dicts/lists
                if isinstance(val, (dict, list)):
                    try:
                        val_str = json.dumps(val, indent=2)
                    except (TypeError, ValueError):
                        val_str = str(val)
                else:
                    val_str = str(val)

                # Replace ALL variations: {{p}} and {p}
                result = result.replace(f"{{{{{p}}}}}", val_str)
                result = result.replace(f"{{{p}}}", val_str)

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
    def _clean_filler(self, text: str) -> str:
        """
        Strip common LLM conversational filler and greetings from a response.
        """
        if not text:
            return ""
        
        # Remove common markdown code block wrappers
        text = text.strip()
        if text.startswith("```"):
            lines = text.split("\n")
            if lines[0].startswith("```"):
                lines = lines[1:]
            if lines and lines[-1].strip() == "```":
                lines = lines[:-1]
            text = "\n".join(lines).strip()

        # Remove common greetings and conversational lead-ins
        fillers = [
            "Hello!", "Certainly!", "Here is the result:", "Sure,", "I am an AI",
            "How can I assist", "I'd be happy to", "The requested output is:",
            "Here's the JSON", "As an AI assistant", "I have generated"
        ]
        
        lines = text.split("\n")
        cleaned_lines = []
        for line in lines:
            trimmed = line.strip()
            is_filler = False
            for f in fillers:
                if trimmed.lower().startswith(f.lower()):
                    is_filler = True
                    break
            if not is_filler:
                cleaned_lines.append(line)
        
        return "\n".join(cleaned_lines).strip()
