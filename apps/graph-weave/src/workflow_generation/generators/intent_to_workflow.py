"""
Intent to Workflow Generator

Deterministic conversion from IntentExtraction → WorkflowSpec.

Algorithm:
1. Validate intent (IntentValidator)
2. Map actions to workflow nodes
3. Construct DAG edges from dependencies
4. Add START/END nodes
5. Validate workflow (WorkflowValidator)
"""

import uuid
from typing import Dict, List, Optional, Any
from ..schemas.intent import IntentExtraction, IntentAction
from ..schemas.workflow import WorkflowSpec, WorkflowNode, WorkflowEdge, NodeType
from ..validators.intent_validator import IntentValidator
from ..validators.workflow_validator import WorkflowValidator


class IntentToWorkflowGenerator:
    """Generates WorkflowSpec from IntentExtraction."""
    
    def __init__(
        self,
        available_operators: Optional[set] = None,
        workflow_id_prefix: str = "wf"
    ):
        """
        Initialize generator.
        
        Args:
            available_operators: Set of available operator names
            workflow_id_prefix: Prefix for generated workflow IDs
        """
        self.intent_validator = IntentValidator(available_operators)
        self.workflow_validator = WorkflowValidator()
        self.workflow_id_prefix = workflow_id_prefix
    
    def generate(self, intent: IntentExtraction) -> Dict[str, Any]:
        """
        Generate WorkflowSpec from IntentExtraction.
        
        Returns:
            {
                "workflow": WorkflowSpec | None,
                "intent_validation": Dict,
                "workflow_validation": Dict,
                "success": bool,
                "errors": List[str]
            }
        """
        # Stage 1: Validate intent
        intent_validation = self.intent_validator.validate(intent)
        
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
        
        # Add START node
        start_node_id = f"{workflow_id}_start"
        nodes.append(WorkflowNode(
            id=start_node_id,
            type=NodeType.START,
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
                type=NodeType.TOOL_CALL,
                name=action.name,
                description=action.description,
                operator=action.operator,
                config={
                    "parameters": [p.dict() for p in action.parameters]
                },
                retry_policy=action.retry_policy,
                timeout_seconds=action.timeout_seconds,
                tags=["action"]
            ))
        
        # Add END node
        end_node_id = f"{workflow_id}_end"
        nodes.append(WorkflowNode(
            id=end_node_id,
            type=NodeType.END,
            name="End",
            description="Workflow end",
            tags=["system"]
        ))
        
        # Create edges
        edges = []
        
        # Connect START to first actions (those with no dependencies)
        first_actions = [a for a in intent.actions if not a.dependencies]
        for action in first_actions:
            edges.append(WorkflowEdge(
                source=start_node_id,
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
        
        # Connect last actions to END
        last_actions = [
            a for a in intent.actions
            if not any(a.id in other.dependencies for other in intent.actions)
        ]
        for action in last_actions:
            edges.append(WorkflowEdge(
                source=action_to_node_id[action.id],
                target=end_node_id,
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
            start_node_id=start_node_id,
            end_node_ids=[end_node_id],
            tags=["generated", "intent-driven"],
            constraints=intent.constraints,
            success_criteria=intent.success_criteria,
            metadata={
                "generated_from_intent": True,
                "intent_metadata": intent.metadata
            }
        )
        
        return workflow
