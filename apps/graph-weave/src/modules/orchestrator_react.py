"""
OrchestratorReAct — core ReAct loop module for GraphWeave's Orchestrator Node.

Implements:
  - Reason → Act → Observe loop with configurable max_iterations
  - Sliding-window context trimming to prevent token overflow
  - Structured output: {orchestrator_trace, final_result}
  - Event emission: orchestrator.thought, orchestrator.tool_called, orchestrator.finished
  - Circuit breaker: exits after max_iterations with error indicator
"""

from __future__ import annotations

import json
import logging
from typing import Any, Callable, Dict, List, Optional

from src.app_logging import get_logger
from src.models import OrchestratorConfig, TraceEntry

logger = get_logger(__name__)


class OrchestratorReAct:
    """
    Self-contained ReAct (Reason + Act) loop for the Orchestrator node type.

    From the graph's perspective this is a black box: it receives workflow_state
    and returns a structured result dict.  Internally it iterates:
      1. Build truncated message window
      2. Call LLM with available tool definitions
      3. If LLM requests tool calls  → execute each, record observations
      4. If LLM returns text content → finish (or hit max_iterations)
    """

    def __init__(
        self,
        client: Any,          # LLMClient — kept as Any to avoid circular import
        mcp_router: Any,      # MCPRouter
        emit: Callable[[str, Dict[str, Any]], None],
        max_context_messages: int = 20,
    ) -> None:
        self.client = client
        self.mcp_router = mcp_router
        self.emit = emit
        self.max_context_messages = max_context_messages

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def run(
        self,
        run_id: str,
        node_id: str,
        config: OrchestratorConfig,
        workflow_state: Dict[str, Any],
        user_prompt: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        Execute the ReAct loop.

        Returns:
            {
                "orchestrator_trace": [TraceEntry ...],
                "final_result": {...}   # dict from LLM or circuit-breaker marker
            }
        """
        trace: List[Dict[str, Any]] = []
        messages: List[Dict[str, Any]] = self._build_initial_messages(config, workflow_state, user_prompt)

        # Fetch tool definitions for allowed_skills
        tools = self._get_tools(config.allowed_skills)

        final_result: Dict[str, Any] = {}
        iterations_used = 0
        status = "completed"

        for iteration in range(config.max_iterations):
            iterations_used = iteration + 1

            # Trim context window before each LLM call
            messages = self._trim_context(messages)

            response = self.client.chat_completion(
                messages=messages,
                provider=config.provider or "github-copilot",
                model=config.model or "gpt-4.1",
                tools=tools if tools else None,
                temperature=0.2,
                max_tokens=2000,
            )

            choice = response["choices"][0]
            message = choice["message"]
            tool_calls = message.get("tool_calls")

            if tool_calls:
                # ---- ACT phase ----
                # Add the assistant's tool-request to history
                messages.append(message)

                for tc in tool_calls:
                    tool_name = tc["function"]["name"]
                    try:
                        tool_args = json.loads(tc["function"]["arguments"])
                    except (json.JSONDecodeError, KeyError):
                        tool_args = {}
                    tool_call_id = tc.get("id", f"call_{tool_name}_{iteration}")

                    # Emit streaming event
                    self.emit(
                        "orchestrator.tool_called",
                        {
                            "node_id": node_id,
                            "iteration": iteration,
                            "tool_name": tool_name,
                            "args": tool_args,
                        },
                    )

                    # Record action in trace
                    trace.append(
                        TraceEntry(
                            type="action",
                            iteration=iteration,
                            content=tool_args,
                            tool_name=tool_name,
                            tool_args=tool_args,
                        ).model_dump()
                    )

                    # Execute tool
                    try:
                        tool_result = self.mcp_router.execute_tool(tool_name, tool_args)
                    except Exception as exc:
                        logger.warning(f"Orchestrator tool '{tool_name}' failed: {exc}")
                        tool_result = {"error": str(exc)}

                    # Emit streaming event with result
                    self.emit(
                        "orchestrator.tool_called",
                        {
                            "node_id": node_id,
                            "iteration": iteration,
                            "tool_name": tool_name,
                            "args": tool_args,
                            "result": tool_result,
                        },
                    )

                    # Append observation to history
                    messages.append(
                        {
                            "role": "tool",
                            "tool_call_id": tool_call_id,
                            "content": json.dumps(tool_result),
                        }
                    )

                    # Record observation in trace
                    trace.append(
                        TraceEntry(
                            type="observation",
                            iteration=iteration,
                            content=tool_result,
                            tool_name=tool_name,
                        ).model_dump()
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

    # ------------------------------------------------------------------
    # Context management
    # ------------------------------------------------------------------

    def _trim_context(self, messages: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Keep the message list within max_context_messages.
        Always preserves the system message (index 0) and drops the
        oldest non-system messages when over the limit.
        """
        if len(messages) <= self.max_context_messages:
            return messages

        system = [m for m in messages if m.get("role") == "system"]
        non_system = [m for m in messages if m.get("role") != "system"]

        keep_non_system = self.max_context_messages - len(system)
        if keep_non_system <= 0:
            return system

        # Keep only the most recent non-system messages
        trimmed_non_system = non_system[-keep_non_system:]
        return system + trimmed_non_system

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _build_initial_messages(
        self,
        config: OrchestratorConfig,
        workflow_state: Dict[str, Any],
        user_prompt: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        context_summary = json.dumps(workflow_state, default=str)
        
        # If no explicit prompt provided, use the hardcoded default
        final_user_prompt = user_prompt or (
            "Use the available tools to investigate, then return your final conclusion "
            "as a JSON object when you are done."
        )

        return [
            {"role": "system", "content": config.system_prompt},
            {
                "role": "user",
                "content": (
                    f"Current workflow context:\n{context_summary}\n\n"
                    f"{final_user_prompt}"
                ),
            },
        ]

    def _get_tools(self, allowed_skills: List[str]) -> List[Dict[str, Any]]:
        """Fetch OpenAI-compatible tool definitions from MCPRouter."""
        try:
            return self.mcp_router.get_tool_definitions(allowed_skills)
        except Exception as exc:
            logger.warning(f"Could not fetch tool definitions: {exc}")
            return []
