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
            json_data = self._extract_json(cleaned_content)

            if json_data is not None:
                self._validate_output_schema(json_data, output_schema)
                result_data = json_data
                self._logger.debug(f"[AGENT] Successfully parsed JSON from {node_id}")
            else:
                inferred_result = self._infer_result_from_tools(
                    tool_results,
                    final_content,
                    get_field("tool_output_mapping", {}),
                )
                if inferred_result is not None:
                    result_data = inferred_result
                    self._logger.info(f"[AGENT] Inferred structured result for {node_id} from tool output")
                elif output_schema:
                    repaired = self._repair_schema_json(
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
                    self._validate_output_schema(repaired, output_schema)
                    result_data = repaired
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

    def _extract_json(self, content: Any) -> Optional[Any]:
        if not isinstance(content, str) or not content.strip():
            return None

        json_match = re.search(r"```(?:json)?\s*(.*?)\s*```", content, re.DOTALL)
        if json_match:
            parsed = self._loads_json(json_match.group(1))
            if parsed is not None:
                return parsed

        parsed = self._loads_json(content)
        if parsed is not None:
            return parsed

        decoder = json.JSONDecoder()
        for index, char in enumerate(content):
            if char not in "{[":
                continue
            try:
                value, _ = decoder.raw_decode(content[index:])
                return value
            except json.JSONDecodeError:
                continue

        return None

    def _loads_json(self, content: str) -> Optional[Any]:
        try:
            return json.loads(content)
        except (TypeError, json.JSONDecodeError):
            return None

    def _repair_schema_json(
        self,
        client: Any,
        messages: List[Dict[str, Any]],
        provider: str,
        model: str,
        temperature: float,
        max_tokens: int,
        reasoning_effort: Optional[str],
        output_schema: Dict[str, Any],
        bad_content: str,
    ) -> Any:
        repair_messages = messages + [
            {
                "role": "user",
                "content": (
                    "Your previous response was not valid JSON for the required output schema. "
                    "Return only one raw JSON value. Do not include markdown, commentary, or reasoning.\n\n"
                    f"Required output schema:\n{json.dumps(output_schema, indent=2)}\n\n"
                    f"Previous response:\n{bad_content}"
                ),
            }
        ]

        response = client.chat_completion(
            messages=repair_messages,
            provider=provider,
            model=model,
            tools=None,
            temperature=0,
            max_tokens=max_tokens,
            reasoning_effort=reasoning_effort,
        )
        repaired_content = response["choices"][0]["message"].get("content", "")
        repaired = self._extract_json(repaired_content)
        if repaired is None:
            raise ValueError("Schema-bound agent returned non-JSON output and JSON repair failed")
        return repaired

    def _validate_output_schema(self, data: Any, schema: Optional[Dict[str, Any]]) -> None:
        if not schema:
            return

        if self._is_legacy_result_confidence_schema(schema) and isinstance(data, dict) and "result" not in data:
            return

        expected_type = schema.get("type")
        if expected_type and not self._matches_json_type(data, expected_type):
            raise ValueError(f"Agent output does not match schema type '{expected_type}'")

        if isinstance(data, dict):
            for key in schema.get("required", []):
                if key not in data:
                    raise ValueError(f"Agent output missing required schema field '{key}'")

            properties = schema.get("properties", {})
            for key, field_schema in properties.items():
                if key not in data or not isinstance(field_schema, dict):
                    continue
                field_type = field_schema.get("type")
                if field_type and not self._matches_json_type(data[key], field_type):
                    raise ValueError(f"Agent output field '{key}' does not match schema type '{field_type}'")

    def _matches_json_type(self, value: Any, expected_type: Any) -> bool:
        if isinstance(expected_type, list):
            return any(self._matches_json_type(value, item) for item in expected_type)

        type_map = {
            "object": dict,
            "array": list,
            "string": str,
            "boolean": bool,
            "integer": int,
            "number": (int, float),
        }
        python_type = type_map.get(expected_type)
        if python_type is None:
            return True
        if expected_type in {"integer", "number"} and isinstance(value, bool):
            return False
        return isinstance(value, python_type)

    def _is_legacy_result_confidence_schema(self, schema: Dict[str, Any]) -> bool:
        required = set(schema.get("required", []))
        properties = schema.get("properties", {})
        result_schema = properties.get("result", {})
        confidence_schema = properties.get("confidence", {})

        return (
            schema.get("type") == "object"
            and required.issubset({"result", "confidence"})
            and "result" in properties
            and "confidence" in properties
            and result_schema.get("type") == "object"
            and confidence_schema.get("type") == "number"
        )

    def _infer_result_from_tools(
        self,
        tool_results: List[Dict[str, Any]],
        final_content: str,
        tool_output_mapping: Optional[Dict[str, Any]] = None,
    ) -> Optional[Dict[str, Any]]:
        if not tool_results:
            return None

        successful = [result for result in tool_results if result.get("status") == "success"]
        latest = successful[-1] if successful else tool_results[-1]
        stdout = str(latest.get("stdout") or "")
        stderr = str(latest.get("stderr") or "")
        parsed_stdout = self._parse_labeled_stdout(stdout)

        output: Dict[str, Any] = {
            "status": "success" if successful else "error",
            "tool_results": tool_results,
            "tool_stdout": stdout,
            "tool_stderr": stderr,
            "stdout_fields": parsed_stdout,
            "raw_response": final_content or stdout,
        }

        for key, spec in (tool_output_mapping or {}).items():
            output[key] = self._resolve_tool_output_spec(spec, output)

        return output

    def _parse_labeled_stdout(self, text: str) -> Dict[str, Any]:
        fields: Dict[str, Any] = {}
        for line in text.splitlines():
            stripped = line.strip()
            match = re.match(r"^([A-Za-z][A-Za-z0-9 _-]*):\s*(.*)$", stripped)
            if not match:
                continue
            key = re.sub(r"[^a-z0-9]+", "_", match.group(1).lower()).strip("_")
            value = match.group(2).strip()
            if not key or value == "":
                continue
            existing = fields.get(key)
            if existing is None:
                fields[key] = value
            elif isinstance(existing, list):
                existing.append(value)
            else:
                fields[key] = [existing, value]
        return fields

    def _resolve_tool_output_spec(self, spec: Any, output: Dict[str, Any]) -> Any:
        if isinstance(spec, str):
            if spec.startswith("$."):
                return self._resolve_dict_path(output, spec[2:].split("."))
            if spec in output:
                return output[spec]
            return output.get("stdout_fields", {}).get(spec)

        if not isinstance(spec, dict):
            return spec

        transform = spec.get("type")
        if transform == "constant":
            return spec.get("value")
        if transform == "array":
            value = self._resolve_tool_output_spec(spec.get("value"), output)
            if value is None:
                return []
            return value if isinstance(value, list) else [value]

        return None

    def _resolve_dict_path(self, current: Any, keys: List[str]) -> Any:
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current[key]
            elif isinstance(current, list) and key.isdigit():
                index = int(key)
                current = current[index] if index < len(current) else None
            else:
                return None
        return current
