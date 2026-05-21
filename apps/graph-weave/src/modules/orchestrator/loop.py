from __future__ import annotations
import json
import logging
from typing import Any, Callable, Dict, List, Optional

from src.app_logging import get_logger
from src.models import OrchestratorConfig, TraceEntry
from src.adapters.ai_provider import LLMClient

from .context import build_initial_messages, trim_context
from .tools import get_tools, execute_tool_calls

logger = get_logger(__name__)

class OrchestratorReAct:
    """
    Self-contained ReAct (Reason + Act) loop for the Orchestrator node type.
    """

    def __init__(
        self,
        client: LLMClient,
        mcp_router: Any,      # MCPRouter
        emit: Callable[[str, Dict[str, Any]], None],
        max_context_messages: int = 20,
        default_provider: str = "github-copilot",
        default_model: str = "gpt-5.4-mini",
    ) -> None:
        self.client = client
        self.mcp_router = mcp_router
        self.emit = emit
        self.max_context_messages = max_context_messages
        self.default_provider = default_provider
        self.default_model = default_model

    def run(
        self,
        run_id: str,
        node_id: str,
        config: OrchestratorConfig,
        workflow_context: Dict[str, Any],
        user_prompt: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Execute the ReAct loop."""
        trace: List[Dict[str, Any]] = []
        messages: List[Dict[str, Any]] = build_initial_messages(config, workflow_context, user_prompt)

        # Fetch tool definitions for allowed_skills
        tools = get_tools(self.mcp_router, config.allowed_skills)

        final_result: Dict[str, Any] = {}
        iterations_used = 0
        status = "completed"

        for iteration in range(config.max_iterations):
            iterations_used = iteration + 1

            # Trim context window before each LLM call
            messages = trim_context(messages, self.max_context_messages)

            response = self.client.chat_completion(
                messages=messages,
                provider=config.provider or self.default_provider,
                model=config.model or self.default_model,
                tools=tools if tools else None,
                temperature=0.2,
                max_tokens=8000,
                reasoning_effort=config.reasoning_effort,
            )

            choice = response["choices"][0]
            message = choice["message"]
            tool_calls = message.get("tool_calls")

            if tool_calls:
                # ---- ACT phase ----
                messages.append(message)
                execute_tool_calls(
                    self.mcp_router,
                    tool_calls,
                    iteration,
                    node_id,
                    self.emit,
                    trace,
                    messages
                )
            else:
                # ---- FINISH phase ----
                thought_text = message.get("content") or ""

                # Emit thought event
                self.emit(
                    "orchestrator.thought",
                    {
                        "node_id": node_id,
                        "iteration": iteration,
                        "thought": thought_text,
                    },
                )

                # Record thought in trace
                trace.append(
                    TraceEntry(
                        type="thought",
                        iteration=iteration,
                        content=thought_text,
                    ).model_dump()
                )

                # Parse structured output
                try:
                    final_result = json.loads(thought_text)
                    if not isinstance(final_result, dict):
                        final_result = {"raw_response": thought_text}
                except json.JSONDecodeError:
                    final_result = {"raw_response": thought_text}

                # Add to history and break
                messages.append({"role": "assistant", "content": thought_text})
                break

        else:
            # Loop exhausted without a finish decision → circuit breaker
            status = "max_iterations_exceeded"
            final_result = {"error": "max_iterations_exceeded"}

        self.emit(
            "orchestrator.finished",
            {
                "node_id": node_id,
                "iterations_used": iterations_used,
                "status": status,
            },
        )

        return {
            "orchestrator_trace": trace,
            "final_result": final_result,
        }

    def _trim_context(self, messages: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        return trim_context(messages, self.max_context_messages)

    def _build_initial_messages(
        self,
        config: OrchestratorConfig,
        workflow_context: Dict[str, Any],
        user_prompt: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        return build_initial_messages(config, workflow_context, user_prompt)
