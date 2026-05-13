from src.adapters.langgraph.agent_node import AgentNodeHandler


class DummyExecutor:
    class Config:
        DEFAULT_PROVIDER = "test"
        DEFAULT_MODEL = "test-model"

    config = Config()

    def __init__(self):
        self.events = []
        self.mcp_router = DummyRouter()
        self.ai_provider_factory = DummyProviderFactory()

    def _get_state_value(self, path, state):
        return None

    def _interpolate_prompt(self, template, state, local_context=None):
        return template

    def _clean_filler(self, content):
        return content or ""

    def _emit_event(self, run_id, event_type, data):
        self.events.append({"run_id": run_id, "type": event_type, "data": data})


class DummyRouter:
    def get_tool_definitions(self, allowed_tools):
        return [{"type": "function", "function": {"name": "bash"}}]

    def execute_tool(self, name, args):
        return {
            "tool": name,
            "status": "success",
            "stdout": "CREATED: /workspace/output.txt\n",
            "stderr": "",
        }


class DummyProviderFactory:
    def __init__(self, client=None):
        self.client = client or DummyClient()

    def get_provider_client(self, provider, model):
        return self.client


class DummyClient:
    def __init__(self):
        self.calls = 0

    def chat_completion(self, **kwargs):
        self.calls += 1
        if self.calls > 1:
            return {
                "choices": [{"message": {"content": '{"unexpected": true}'}}],
                "usage": {"total_tokens": 1},
            }
        return {
            "choices": [
                {
                    "message": {
                        "content": "",
                        "tool_calls": [
                            {
                                "id": "call-1",
                                "function": {
                                    "name": "bash",
                                    "arguments": '{"command": "echo created"}',
                                },
                            }
                        ],
                    }
                }
            ],
            "usage": {"total_tokens": 1},
        }


class EnvelopeClient:
    def chat_completion(self, **kwargs):
        return {
            "choices": [
                {
                    "message": {
                        "content": '{"result": {"nodes": [{"id": "entry", "type": "entry"}]}, "confidence": 1}'
                    }
                }
            ],
            "usage": {"total_tokens": 1},
        }


class ErrorRouter(DummyRouter):
    def execute_tool(self, name, args):
        return {
            "tool": name,
            "command": args.get("command"),
            "status": "error",
            "exit_code": 2,
            "stdout": "",
            "stderr": "script failed",
            "error": None,
        }


def test_infer_generic_fields_from_successful_bash_stdout():
    handler = AgentNodeHandler(DummyExecutor())

    result = handler._infer_result_from_tools(
        [
            {
                "tool": "bash",
                "status": "success",
                "stdout": "CREATED: /workspace/output.txt\nID: abc123\n",
            }
        ],
        "",
    )

    assert result["status"] == "success"
    assert result["stdout_fields"] == {
        "created": "/workspace/output.txt",
        "id": "abc123",
    }
    assert result["tool_stdout"] == "CREATED: /workspace/output.txt\nID: abc123\n"


def test_applies_declared_tool_output_mapping():
    handler = AgentNodeHandler(DummyExecutor())

    result = handler._infer_result_from_tools(
        [
            {
                "tool": "bash",
                "status": "success",
                "stdout": "CREATED: /workspace/output.txt\n",
            }
        ],
        "",
        {
            "artifact_path": "$.stdout_fields.created",
            "artifact_paths": {"type": "array", "value": "$.stdout_fields.created"},
            "domain_status": {"type": "constant", "value": "processed"},
        },
    )

    assert result["artifact_path"] == "/workspace/output.txt"
    assert result["artifact_paths"] == ["/workspace/output.txt"]
    assert result["domain_status"] == "processed"


def test_complete_after_tool_calls_returns_mapped_tool_result_without_extra_llm_turn():
    executor = DummyExecutor()
    handler = AgentNodeHandler(executor)

    result = handler.execute(
        "run-1",
        {
            "id": "script_step",
            "type": "agent_node",
            "config": {
                "system_prompt": "Use bash.",
                "user_prompt_template": "Run the script.",
                "tools": ["bash"],
                "complete_after_tool_calls": True,
                "tool_output_mapping": {
                    "artifact_path": "$.stdout_fields.created",
                },
            },
        },
        {"workflow_state": {}, "node_results": {}},
        {},
    )

    assert executor.ai_provider_factory.client.calls == 1
    assert result["result"]["artifact_path"] == "/workspace/output.txt"


def test_complete_after_tool_calls_fails_error_tool_result_by_default():
    executor = DummyExecutor()
    executor.mcp_router = ErrorRouter()
    handler = AgentNodeHandler(executor)

    try:
        handler.execute(
            "run-1",
            {
                "id": "script_step",
                "type": "agent_node",
                "config": {
                    "system_prompt": "Use bash.",
                    "user_prompt_template": "Run the script.",
                    "tools": ["bash"],
                    "complete_after_tool_calls": True,
                },
            },
            {"workflow_state": {}, "node_results": {}},
            {},
        )
    except ValueError as exc:
        assert "Tool execution failed" in str(exc)
        assert "stderr=script failed" in str(exc)
    else:
        raise AssertionError("Expected error tool result to fail the agent node")


def test_allow_tool_errors_keeps_error_tool_result_structured():
    executor = DummyExecutor()
    executor.mcp_router = ErrorRouter()
    handler = AgentNodeHandler(executor)

    result = handler.execute(
        "run-1",
        {
            "id": "script_step",
            "type": "agent_node",
            "config": {
                "system_prompt": "Use bash.",
                "user_prompt_template": "Run the script.",
                "tools": ["bash"],
                "complete_after_tool_calls": True,
                "allow_tool_errors": True,
            },
        },
        {"workflow_state": {}, "node_results": {}},
        {},
    )

    assert result["result"]["status"] == "error"
    assert result["result"]["tool_results"][0]["stderr"] == "script failed"


def test_tool_args_with_unresolved_template_placeholder_fail_before_execution():
    handler = AgentNodeHandler(DummyExecutor())

    try:
        handler._validate_tool_args_resolved(
            "bash",
            {
                "command": "bash script.sh --claims \"{ideas_newline_joined}\"",
                "cwd": "/workspace",
            },
        )
    except ValueError as exc:
        assert "unresolved template placeholders" in str(exc)
        assert "{ideas_newline_joined}" in str(exc)
    else:
        raise AssertionError("Expected unresolved tool placeholder to fail")


def test_tool_args_allow_non_template_braces():
    handler = AgentNodeHandler(DummyExecutor())

    handler._validate_tool_args_resolved(
        "bash",
        {
            "command": "echo '{\"status\":\"processed\"}'",
        },
    )


def test_extract_json_from_prefaced_content():
    handler = AgentNodeHandler(DummyExecutor())

    result = handler._extract_json('First, assemble it.\n{"is_valid": true, "generated_workflow": {}}')

    assert result == {"is_valid": True, "generated_workflow": {}}


class RepairClient:
    def __init__(self, repaired_content):
        self.calls = 0
        self.repaired_content = repaired_content

    def chat_completion(self, **kwargs):
        self.calls += 1
        content = "I will reason about this first." if self.calls == 1 else self.repaired_content
        return {
            "choices": [{"message": {"content": content}}],
            "usage": {"total_tokens": 1},
        }


def test_schema_bound_agent_repairs_non_json_once():
    client = RepairClient('{"is_valid": true, "generated_workflow": {}}')
    executor = DummyExecutor()
    executor.ai_provider_factory = DummyProviderFactory(client)
    handler = AgentNodeHandler(executor)

    result = handler.execute(
        "run-1",
        {
            "id": "assembler",
            "type": "agent_node",
            "config": {
                "system_prompt": "Return JSON.",
                "user_prompt_template": "Assemble.",
                "output_schema": {
                    "type": "object",
                    "properties": {
                        "is_valid": {"type": "boolean"},
                        "generated_workflow": {"type": "object"},
                    },
                    "required": ["is_valid", "generated_workflow"],
                },
            },
        },
        {"workflow_state": {}, "node_results": {}},
        {},
    )

    assert client.calls == 2
    assert result["result"] == {"is_valid": True, "generated_workflow": {}}


def test_schema_bound_agent_fails_when_repair_missing_required_field():
    client = RepairClient('{"is_valid": true}')
    executor = DummyExecutor()
    executor.ai_provider_factory = DummyProviderFactory(client)
    handler = AgentNodeHandler(executor)

    try:
        handler.execute(
            "run-1",
            {
                "id": "assembler",
                "type": "agent_node",
                "config": {
                    "system_prompt": "Return JSON.",
                    "user_prompt_template": "Assemble.",
                    "output_schema": {
                        "type": "object",
                        "properties": {
                            "is_valid": {"type": "boolean"},
                            "generated_workflow": {"type": "object"},
                        },
                        "required": ["is_valid", "generated_workflow"],
                    },
                },
            },
            {"workflow_state": {}, "node_results": {}},
            {},
        )
    except ValueError as exc:
        assert "missing required schema field 'generated_workflow'" in str(exc)
    else:
        raise AssertionError("Expected schema-bound agent to fail invalid repaired JSON")


def test_legacy_result_confidence_schema_accepts_flat_json():
    handler = AgentNodeHandler(DummyExecutor())

    handler._validate_output_schema(
        {
            "file_path": "/tmp/inbox.md",
            "file_content": "body",
            "existing_tags": [],
        },
        {
            "type": "object",
            "properties": {
                "result": {"type": "object"},
                "confidence": {"type": "number", "minimum": 0, "maximum": 1},
            },
            "required": ["result", "confidence"],
        },
    )


def test_coerce_output_schema_unwraps_result_envelope():
    handler = AgentNodeHandler(DummyExecutor())

    result = handler._coerce_to_output_schema(
        {"result": {"nodes": [{"id": "entry", "type": "entry"}]}, "confidence": 1},
        {
            "type": "object",
            "properties": {"nodes": {"type": "array"}},
            "required": ["nodes"],
        },
    )

    assert result == {"nodes": [{"id": "entry", "type": "entry"}]}


def test_agent_execute_accepts_result_envelope_when_inner_object_matches_schema():
    executor = DummyExecutor()
    executor.ai_provider_factory = DummyProviderFactory(EnvelopeClient())
    handler = AgentNodeHandler(executor)

    result = handler.execute(
        "run-1",
        {
            "id": "node_builder",
            "type": "agent_node",
            "config": {
                "system_prompt": "Return JSON",
                "user_prompt_template": "Build nodes",
                "output_key": "nodes_definition",
                "output_schema": {
                    "type": "object",
                    "properties": {"nodes": {"type": "array"}},
                    "required": ["nodes"],
                },
            },
        },
        {"workflow_state": {}, "node_results": {}},
        {},
    )

    assert result["result"] == {"nodes": [{"id": "entry", "type": "entry"}]}
    assert result["nodes_definition"] == {"nodes": [{"id": "entry", "type": "entry"}]}


def test_output_schema_rejects_array_items_with_wrong_type():
    handler = AgentNodeHandler(DummyExecutor())

    try:
        handler._validate_output_schema(
            {"edges": ["entry -> exit"]},
            {
                "type": "object",
                "properties": {
                    "edges": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "from": {"type": "string"},
                                "to": {"type": "string"},
                            },
                            "required": ["from", "to"],
                        },
                    }
                },
                "required": ["edges"],
            },
        )
    except ValueError as exc:
        assert "edges[0]" in str(exc)
        assert "object" in str(exc)
    else:
        raise AssertionError("Expected schema validation to reject string edge items")


def test_domain_schema_still_rejects_missing_required_field():
    handler = AgentNodeHandler(DummyExecutor())

    try:
        handler._validate_output_schema(
            {"is_valid": True},
            {
                "type": "object",
                "properties": {
                    "is_valid": {"type": "boolean"},
                    "generated_workflow": {"type": "object"},
                },
                "required": ["is_valid", "generated_workflow"],
            },
        )
    except ValueError as exc:
        assert "missing required schema field 'generated_workflow'" in str(exc)
    else:
        raise AssertionError("Expected domain schema to reject missing required field")
