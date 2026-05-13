import json
from typing import Any, Dict, List, Optional
from src.app_logging import get_logger
from ...mcp_router import ProviderConfigError

from .placeholder_utils import validate_tool_args_resolved
from .json_utils import extract_json, repair_schema_json
from .schema_utils import coerce_to_output_schema, validate_output_schema
from .tool_utils import infer_result_from_tools, format_tool_error

logger = get_logger(__name__)

class AgentNodeHandler:
    """
    Handles the execution of agent nodes, including the tool loop.
    """
    
    def __init__(self, executor: Any):
        self.executor = executor
        self._logger = logger

    def execute(
        self,
        run_id: str,
        node: Dict[str, Any],
        state: Dict[str, Any],
        workflow: Dict[str, Any],
    ) -> Dict[str, Any]:
        node_id = node.get("id")
        config = node.get("config", {})
        
        def get_field(name, default=None):
            return config.get(name) or node.get(name) or default

        system_prompt = get_field("system_prompt", "You are a helpful assistant.")
        user_prompt_template = get_field("user_prompt_template", "")
        
        input_mapping = get_field("input_mapping", {})
        if input_mapping:
            agent_input_context = {}
            for key, path in input_mapping.items():
                agent_input_context[key] = self.executor._get_state_value(path, state)
        else:
            agent_input_context = dict(state.get("workflow_state", {}))
        
        user_prompt = self.executor._interpolate_prompt(user_prompt_template, state, local_context=agent_input_context)
        system_prompt = self.executor._interpolate_prompt(system_prompt, state, local_context=agent_input_context)
        
        provider_raw = get_field("provider", self.executor.config.DEFAULT_PROVIDER)
        model_raw = get_field("model", self.executor.config.DEFAULT_MODEL)
        reasoning_effort_raw = get_field("reasoning_effort")
        
        provider = self.executor._interpolate_prompt(provider_raw, state, local_context=agent_input_context)
        model = self.executor._interpolate_prompt(model_raw, state, local_context=agent_input_context)
        reasoning_effort = self.executor._interpolate_prompt(reasoning_effort_raw, state, local_context=agent_input_context) if reasoning_effort_raw else None
        
        temperature = get_field("temperature", 0.7)
        max_tokens = get_field("max_tokens", 8000)
        allowed_tools = get_field("tools", [])
        complete_after_tool_calls = bool(get_field("complete_after_tool_calls", False))
        allow_tool_errors = bool(get_field("allow_tool_errors", False))
        output_schema = get_field("output_schema")

        try:
            client = self.executor.ai_provider_factory.get_provider_client(provider, model)
            messages = [
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt},
            ]
            tools = self.executor.mcp_router.get_tool_definitions(allowed_tools)

            self.executor._emit_event(
                run_id,
                "agent.request",
                {
                    "node_id": node_id,
                    "provider": provider,
                    "model": model,
                    "tool_count": len(tools),
                },
            )

            max_turns = 5
            turns = 0
            final_content = ""
            total_tokens = 0
            all_tool_calls = []
            tool_results = []

            while turns < max_turns:
                turns += 1
                response = client.chat_completion(
                    messages=messages,
                    provider=provider,
                    model=model,
                    tools=tools if tools else None,
                    temperature=temperature,
                    max_tokens=max_tokens,
                    reasoning_effort=reasoning_effort,
                )

                total_tokens += response.get("usage", {}).get("total_tokens", 0)
                choice = response["choices"][0]
                message = choice["message"]
                
                # Log raw LLM response for debugging tool-use issues
                self._logger.info(f"[AGENT] {node_id} response: {message.get('content', '')}")
                if message.get("tool_calls"):
                    self._logger.info(f"[AGENT] {node_id} requested tools: {[tc['function']['name'] for tc in message['tool_calls']]}")

                tool_calls = message.get("tool_calls")
                reasoning = message.get("reasoning_content", "")

                if reasoning:
                    self._logger.info(f"[AGENT] {node_id} reasoning: {reasoning[:200]}...")

                if not tool_calls:
                    content = message.get("content", "")
                    if not content and reasoning and not output_schema:
                        self._logger.info(f"[AGENT] {node_id} content is empty, using reasoning as fallback")
                        content = reasoning
                    
                    final_content = self.executor._clean_filler(content)
                    break
                
                all_tool_calls.extend(tool_calls)
                messages.append(message)

                self.executor._emit_event(
                    run_id,
                    "agent.tool_calls",
                    {"node_id": node_id, "count": len(tool_calls)},
                )

                for tool_call in tool_calls:
                    tool_id = tool_call["id"]
                    tool_name = tool_call["function"]["name"]
                    tool_args_raw = tool_call["function"]["arguments"]
                    tool_args_interpolated = self.executor._interpolate_prompt(tool_args_raw, state, local_context=agent_input_context)
                    try:
                        tool_args = json.loads(tool_args_interpolated)
                    except json.JSONDecodeError:
                        tool_args = json.loads(tool_args_raw)
                    
                    validate_tool_args_resolved(tool_name, tool_args)

                    if tool_name == "bash" and "cwd" in config:
                        if "cwd" not in tool_args:
                            tool_args["cwd"] = config["cwd"]

                    try:
                        self.executor._emit_event(run_id, "tool.started", {"tool": tool_name, "id": tool_id, "input": tool_args})
                        tool_result = self.executor.mcp_router.execute_tool(tool_name, tool_args)
                        tool_results.append(tool_result)
                        messages.append({"role": "tool", "tool_call_id": tool_id, "content": json.dumps(tool_result)})
                        self.executor._emit_event(run_id, "tool.completed", {"tool": tool_name, "id": tool_id, "result": tool_result})
                    except Exception as e:
                        self._logger.error(f"Tool {tool_name} failed: {e}")
                        tool_result = {"tool": tool_name, "status": "error", "error": str(e)}
                        tool_results.append(tool_result)
                        messages.append({"role": "tool", "tool_call_id": tool_id, "content": json.dumps(tool_result)})
                        self.executor._emit_event(run_id, "tool.failed", {"tool": tool_name, "id": tool_id, "error": str(e)})

                if complete_after_tool_calls:
                    final_content = ""
                    break

            cleaned_content = final_content
            # Robust JSON extraction: try code blocks first, then any JSON-like structure
            json_data = extract_json(cleaned_content)

            if json_data is not None:
                try:
                    result_data = coerce_to_output_schema(json_data, output_schema)
                    self._logger.debug(f"[AGENT] Successfully parsed JSON from {node_id}")
                except ValueError as validation_error:
                    if not output_schema:
                        raise
                    repaired = repair_schema_json(
                        client=client,
                        messages=messages,
                        provider=provider,
                        model=model,
                        temperature=temperature,
                        max_tokens=max_tokens,
                        reasoning_effort=reasoning_effort,
                        output_schema=output_schema,
                        bad_content=final_content,
                        validation_error=str(validation_error),
                    )
                    result_data = coerce_to_output_schema(repaired, output_schema)
            else:
                inferred_result = infer_result_from_tools(
                    tool_results,
                    final_content,
                    get_field("tool_output_mapping", {}),
                )
                if inferred_result is not None:
                    if inferred_result.get("status") == "error" and not allow_tool_errors:
                        raise ValueError(format_tool_error(inferred_result))
                    result_data = inferred_result
                    self._logger.info(f"[AGENT] Inferred structured result for {node_id} from tool output")
                elif output_schema:
                    repaired = repair_schema_json(
                        client=client,
                        messages=messages,
                        provider=provider,
                        model=model,
                        temperature=temperature,
                        max_tokens=max_tokens,
                        reasoning_effort=reasoning_effort,
                        output_schema=output_schema,
                        bad_content=final_content,
                    )
                    result_data = coerce_to_output_schema(repaired, output_schema)
                else:
                    result_data = {"raw_response": final_content}
                    self._logger.warning(f"[AGENT] Failed to parse JSON from {node_id}, using raw_response")

            output_key = get_field("output_key")
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

    def _format_tool_error(self, inferred_result: Dict[str, Any]) -> str:
        return format_tool_error(inferred_result)

    def _validate_tool_args_resolved(self, tool_name: str, tool_args: Any) -> None:
        return validate_tool_args_resolved(tool_name, tool_args)

    def _extract_json(self, content: Any) -> Optional[Any]:
        return extract_json(content)

    def _repair_schema_json(self, **kwargs) -> Any:
        return repair_schema_json(**kwargs)

    def _coerce_to_output_schema(self, data: Any, schema: Optional[Dict[str, Any]]) -> Any:
        return coerce_to_output_schema(data, schema)

    def _validate_output_schema(self, data: Any, schema: Optional[Dict[str, Any]]) -> None:
        return validate_output_schema(data, schema)

    def _infer_result_from_tools(
        self,
        tool_results: List[Dict[str, Any]],
        final_content: str,
        tool_output_mapping: Optional[Dict[str, Any]] = None,
    ) -> Optional[Dict[str, Any]]:
        return infer_result_from_tools(tool_results, final_content, tool_output_mapping)
