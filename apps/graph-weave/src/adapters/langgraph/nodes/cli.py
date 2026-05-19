import json
from typing import Any, Dict, Optional
from src.app_logging import get_logger
from .agent.tool_utils import infer_result_from_tools, format_tool_error, validate_command_contract

logger = get_logger(__name__)

class CLINodeHandler:
    """
    Handles the execution of CLI nodes, which trigger predefined bash commands
    without using an LLM. This is cost-effective for deterministic tasks.
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
        
        # Determine the command to run
        command_template = config.get("command") or node.get("command")
        if not command_template:
            raise ValueError(f"CLI node {node_id} is missing 'command' in config")
            
        # 1. Resolve input mapping if present
        input_mapping = config.get("input_mapping") or node.get("input_mapping", {})
        cli_input_context = {}
        if input_mapping:
            for key, path in input_mapping.items():
                cli_input_context[key] = self.executor._get_state_value(path, state)
        else:
            # Default to full workflow state if no mapping
            cli_input_context = dict(state.get("workflow_state", {}))
            
        # 2. Interpolate command
        command = self.executor._interpolate_prompt(command_template, state, local_context=cli_input_context)
        validate_command_contract(command, config.get("command_contract"), node_id or "")

        self.executor._emit_event(
            run_id,
            "cli.started",
            {
                "node_id": node_id,
                "command": command,
            },
        )

        try:
            # 3. Execute the command via MCPRouter's bash tool
            # This ensures we follow the same security/path restrictions as the LLM-driven bash tool
            result = self.executor.mcp_router.execute_tool("bash", {"command": command})
            
            # 4. Process the result
            status = "completed" if result.get("status") == "success" else "failed"
            exit_code = result.get("exit_code", -1)
            tool_output_mapping = config.get("tool_output_mapping") or node.get("tool_output_mapping")
            allow_errors = bool(config.get("allow_errors", False) or node.get("allow_errors", False))
            fail_on_error = bool(config.get("fail_on_error", False) or node.get("fail_on_error", False))
            
            # Try to parse stdout as JSON if possible, otherwise use raw string
            stdout = result.get("stdout", "")
            inferred_result = infer_result_from_tools([result], stdout, tool_output_mapping)
            if inferred_result and inferred_result.get("status") == "error" and fail_on_error and not allow_errors:
                raise ValueError(format_tool_error(inferred_result))

            if tool_output_mapping:
                output_data = inferred_result or {}
            else:
                try:
                    # Only try to parse if it looks like JSON
                    if stdout.strip().startswith(("{", "[")):
                        output_data = json.loads(stdout)
                    else:
                        output_data = {"stdout": stdout}
                except:
                    output_data = {"stdout": stdout}
                
            # Add stderr and exit_code to output if they exist
            if isinstance(output_data, dict):
                output_data["stderr"] = result.get("stderr", "")
                output_data["exit_code"] = exit_code
                output_data["success"] = result.get("status") == "success"

            self.executor._emit_event(
                run_id,
                "cli.completed",
                {
                    "node_id": node_id,
                    "status": status,
                    "exit_code": exit_code,
                },
            )

            output_key = config.get("output_key") or node.get("output_key")
            actual_result = {output_key: output_data} if output_key else output_data

            node_result_payload = {
                "node_id": node_id,
                "status": status,
                "result": output_data,
                f"{node_id}_output": output_data,
                f"{node_id}_status": status,
                "exit_code": exit_code,
            }
            
            if isinstance(actual_result, dict):
                for key, value in actual_result.items():
                    if key not in {"node_id", "status", "result", f"{node_id}_status"}:
                        node_result_payload[key] = value
                
            return node_result_payload

        except Exception as e:
            self._logger.error(f"CLI node {node_id} execution failed: {e}")
            self.executor._emit_event(
                run_id,
                "cli.failed",
                {
                    "node_id": node_id,
                    "error": str(e),
                },
            )
            raise ValueError(f"CLI node execution failed: {e}")
