from typing import Any, Dict
from src.models import OrchestratorConfig
from src.modules.orchestrator_react import OrchestratorReAct

class OrchestratorNodeHandler:
    """
    Handles the execution of orchestrator nodes by delegating to OrchestratorReAct.
    """
    
    def __init__(self, executor: Any):
        self.executor = executor

    def execute(
        self,
        run_id: str,
        node: Dict[str, Any],
        state: Dict[str, Any],
        workflow: Dict[str, Any],
    ) -> Dict[str, Any]:
        node_id = node.get("id")
        raw_config = node.get("config", {}) or node

        try:
            config = OrchestratorConfig(**raw_config)
        except Exception as e:
            raise ValueError(f"Invalid orchestrator config on node '{node_id}': {e}")

        input_mapping = config.input_mapping
        if input_mapping:
            orchestrator_context = {}
            for key, path in input_mapping.items():
                orchestrator_context[key] = self.executor._get_state_value(path, state)
        else:
            orchestrator_context = dict(state.get("workflow_state", {}))

        config.system_prompt = self.executor._interpolate_prompt(config.system_prompt, state)
        user_prompt = self.executor._interpolate_prompt(config.user_prompt_template, state) if config.user_prompt_template else None

        react = OrchestratorReAct(
            client=self.executor.ai_provider_factory.get_provider_client(
                config.provider or "github-copilot",
                config.model or "gpt-4.1",
            ),
            mcp_router=self.executor.mcp_router,
            emit=lambda etype, data: self.executor._emit_event(run_id, etype, data),
        )

        result = react.run(
            run_id=run_id,
            node_id=node_id,
            config=config,
            workflow_state=orchestrator_context,
            user_prompt=user_prompt,
        )

        return {
            **result,
            "orchestrator_result": result.get("final_result", {}),
            "node_id": node_id,
        }
