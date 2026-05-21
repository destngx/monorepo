from typing import Any, Dict
from src.models import OrchestratorConfig
from src.modules.orchestrator import OrchestratorReAct
from ..runtime.state import StateResolver

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
            orchestrator_context = StateResolver(state).resolve_mapping(input_mapping)
        else:
            orchestrator_context = dict(state.get("workflow", {}))

        config.system_prompt = self.executor._interpolate_prompt(config.system_prompt, state)
        user_prompt = self.executor._interpolate_prompt(config.user_prompt_template, state) if config.user_prompt_template else None
        
        # Support dynamic provider/model/reasoning_effort overrides
        if config.provider:
            config.provider = self.executor._interpolate_prompt(config.provider, state)
        if config.model:
            config.model = self.executor._interpolate_prompt(config.model, state)
        if config.reasoning_effort:
            config.reasoning_effort = self.executor._interpolate_prompt(config.reasoning_effort, state)
        else:
            config.reasoning_effort = self.executor.config.DEFAULT_REASONING_EFFORT

        react = OrchestratorReAct(
            client=self.executor.ai_provider_factory.get_provider_client(
                config.provider or self.executor.config.DEFAULT_PROVIDER,
                config.model or self.executor.config.DEFAULT_MODEL,
            ),
            mcp_router=self.executor.mcp_router,
            emit=lambda etype, data: self.executor._emit_event(run_id, etype, data),
            default_provider=self.executor.config.DEFAULT_PROVIDER,
            default_model=self.executor.config.DEFAULT_MODEL,
        )

        result = react.run(
            run_id=run_id,
            node_id=node_id,
            config=config,
            workflow_context=orchestrator_context,
            user_prompt=user_prompt,
        )

        return {
            "status": "completed",
            "result": result.get("final_result", {}),
            "outputs": {
                "orchestrator_trace": result.get("orchestrator_trace", []),
                "final_result": result.get("final_result", {}),
            },
            "metadata": {
                "node_id": node_id,
                "iterations": len(result.get("orchestrator_trace", [])),
            },
        }
