import json
import re
from typing import Any, Dict, List, Optional
from src.app_logging import get_logger
from ..mcp_router import ProviderConfigError

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
        
        provider = get_field("provider", "github-copilot")
        model = get_field("model", "gpt-4.1")
        temperature = get_field("temperature", 0.7)
        max_tokens = get_field("max_tokens", 4000)
        allowed_tools = get_field("tools", [])

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
                tool_calls = message.get("tool_calls")

                if not tool_calls:
                    final_content = self.executor._clean_filler(message.get("content", ""))
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

                    if tool_name == "bash" and "cwd" in config:
                        if "cwd" not in tool_args:
                            tool_args["cwd"] = config["cwd"]

                    try:
                        self.executor._emit_event(run_id, "tool.started", {"tool": tool_name, "id": tool_id, "input": tool_args})
                        tool_result = self.executor.mcp_router.execute_tool(tool_name, tool_args)
                        messages.append({"role": "tool", "tool_call_id": tool_id, "content": json.dumps(tool_result)})
                        self.executor._emit_event(run_id, "tool.completed", {"tool": tool_name, "id": tool_id, "result": tool_result})
                    except Exception as e:
                        self._logger.error(f"Tool {tool_name} failed: {e}")
                        messages.append({"role": "tool", "tool_call_id": tool_id, "content": json.dumps({"error": str(e)})})
                        self.executor._emit_event(run_id, "tool.failed", {"tool": tool_name, "id": tool_id, "error": str(e)})

            cleaned_content = final_content
            if isinstance(cleaned_content, str) and "```" in cleaned_content:
                json_match = re.search(r"```(?:json)?\s*(.*?)\s*```", cleaned_content, re.DOTALL)
                if json_match:
                    cleaned_content = json_match.group(1)

            try:
                result_data = json.loads(cleaned_content or final_content)
                print(f"[AGENT] Parsed JSON from {node_id}", flush=True)
            except json.JSONDecodeError:
                result_data = {"raw_response": final_content}
                print(f"[AGENT] Failed to parse JSON from {node_id}, using raw_response", flush=True)

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
