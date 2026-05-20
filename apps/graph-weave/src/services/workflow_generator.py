"""
Intent to Workflow Generator

Deterministic conversion from IntentExtraction → WorkflowSpec.
Relocated to core services layer and harmonized with the entry_point / exit_point DAG format.
"""

import uuid
from typing import Dict, List, Optional, Any, Set
from ..models import (
    IntentExtraction,
    IntentAction,
    WorkflowSpec,
    WorkflowNode,
    WorkflowEdge,
    WorkflowNodeType,
)
from .intent_validator import IntentValidator
from .workflow_validator import WorkflowValidator


class IntentToWorkflowGenerator:
    """Generates WorkflowSpec from IntentExtraction."""
    
    def __init__(
        self,
        available_operators: Optional[Set[str]] = None,
        workflow_id_prefix: str = "wf",
        node_store: Optional[Any] = None,
    ):
        """
        Initialize generator.
        
        Args:
            available_operators: Set of available operator names. 
                                 If provided, will override the dynamically resolved capabilities.
            workflow_id_prefix: Prefix for generated workflow IDs
            node_store: RedisNodeStore instance to dynamically resolve capability availability.
        """
        self.intent_validator = IntentValidator(
            available_operators=available_operators,
            node_store=node_store
        )
        self.workflow_validator = WorkflowValidator()
        self.workflow_id_prefix = workflow_id_prefix
    
    def generate(self, intent: IntentExtraction) -> Dict[str, Any]:
        """
        Generate WorkflowSpec from IntentExtraction synchronously.
        
        Backward compatible synchronous wrapper running the async pipeline.
        """
        import asyncio
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = None
            
        if loop and loop.is_running():
            import concurrent.futures
            with concurrent.futures.ThreadPoolExecutor() as pool:
                return pool.submit(lambda: asyncio.run(self.generate_async(intent))).result()
        else:
            return asyncio.run(self.generate_async(intent))

    async def generate_async(self, intent: IntentExtraction) -> Dict[str, Any]:
        """
        Generate WorkflowSpec from IntentExtraction asynchronously.
        
        This method is async to support dynamic capability lookups against Redis.
        
        Returns:
            {
                "workflow": WorkflowSpec | None,
                "intent_validation": Dict,
                "workflow_validation": Dict,
                "success": bool,
                "errors": List[str]
            }
        """
        # Stage 1: Validate intent (async capability check)
        intent_validation = await self.intent_validator.validate_async(intent)
        
        if not intent_validation["valid"]:
            return {
                "workflow": None,
                "intent_validation": intent_validation,
                "workflow_validation": None,
                "success": False,
                "errors": intent_validation["errors"]
            }
        
        # Stage 2: Generate workflow
        try:
            workflow = self._generate_workflow(intent)
        except Exception as e:
            return {
                "workflow": None,
                "intent_validation": intent_validation,
                "workflow_validation": None,
                "success": False,
                "errors": [f"Workflow generation failed: {str(e)}"]
            }
        
        # Stage 3: Validate workflow
        workflow_validation = self.workflow_validator.validate(workflow)
        
        return {
            "workflow": workflow if workflow_validation["valid"] else None,
            "intent_validation": intent_validation,
            "workflow_validation": workflow_validation,
            "success": workflow_validation["valid"],
            "errors": workflow_validation["errors"]
        }
    
    def _generate_workflow(self, intent: IntentExtraction) -> WorkflowSpec:
        """Generate WorkflowSpec from validated IntentExtraction."""
        
        # Generate workflow ID
        workflow_id = f"{self.workflow_id_prefix}_{uuid.uuid4().hex[:8]}"
        
        # Create nodes
        nodes = []
        action_to_node_id = {}
        
        # Add ENTRY node
        entry_node_id = f"{workflow_id}_entry"
        nodes.append(WorkflowNode(
            id=entry_node_id,
            type=WorkflowNodeType.ENTRY,
            name="Start",
            description="Workflow start",
            tags=["system"]
        ))
        
        # Convert actions to nodes
        for action in intent.actions:
            node_id = f"{workflow_id}_{action.id}"
            action_to_node_id[action.id] = node_id
            
            nodes.append(WorkflowNode(
                id=node_id,
                type=WorkflowNodeType.TOOL_CALL,
                name=action.name,
                description=action.description,
                operator=action.operator,
                config={
                    "parameters": [p.model_dump() for p in action.parameters]
                },
                retry_policy=action.retry_policy,
                timeout_seconds=action.timeout_seconds,
                tags=["action"]
            ))
        
        # Add EXIT node
        exit_node_id = f"{workflow_id}_exit"
        nodes.append(WorkflowNode(
            id=exit_node_id,
            type=WorkflowNodeType.EXIT,
            name="End",
            description="Workflow end",
            tags=["system"]
        ))
        
        # Create edges
        edges = []
        
        # Connect ENTRY to first actions (those with no dependencies)
        first_actions = [a for a in intent.actions if not a.dependencies]
        for action in first_actions:
            edges.append(WorkflowEdge(
                source=entry_node_id,
                target=action_to_node_id[action.id],
                condition="success",
                label="start"
            ))
        
        # Connect actions based on dependencies
        for action in intent.actions:
            for dep_id in action.dependencies:
                edges.append(WorkflowEdge(
                    source=action_to_node_id[dep_id],
                    target=action_to_node_id[action.id],
                    condition="success",
                    label=f"after_{dep_id}"
                ))
        
        # Connect last actions to EXIT
        last_actions = [
            a for a in intent.actions
            if not any(a.id in other.dependencies for other in intent.actions)
        ]
        for action in last_actions:
            edges.append(WorkflowEdge(
                source=action_to_node_id[action.id],
                target=exit_node_id,
                condition="success",
                label="complete"
            ))
        
        # Create WorkflowSpec
        workflow = WorkflowSpec(
            id=workflow_id,
            name=intent.goal,
            description=intent.description,
            version="1.0.0",
            nodes=nodes,
            edges=edges,
            entry_point=entry_node_id,
            exit_point=exit_node_id,
            tags=["generated", "intent-driven"],
            constraints=intent.constraints,
            success_criteria=intent.success_criteria,
            metadata={
                "generated_from_intent": True,
                "intent_metadata": intent.metadata
            }
        )
        
        return workflow
