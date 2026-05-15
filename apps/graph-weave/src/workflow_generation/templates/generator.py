"""
Template-based Workflow Generator

Converts WorkflowTemplate → WorkflowSpec with variable substitution.
"""

from typing import Dict, Any
from .template import WorkflowTemplate
from ..schemas.workflow import WorkflowSpec, WorkflowNode, WorkflowEdge, NodeType


class TemplateWorkflowGenerator:
    """Generates WorkflowSpec from WorkflowTemplate."""
    
    def generate(
        self,
        template: WorkflowTemplate,
        variables: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Generate WorkflowSpec from template with variable substitution.
        
        Returns:
            {
                "workflow": WorkflowSpec | None,
                "template_validation": Dict,
                "success": bool,
                "errors": List[str]
            }
        """
        # Validate variables
        validation = template.validate_variables(variables)
        
        if not validation["valid"]:
            return {
                "workflow": None,
                "template_validation": validation,
                "success": False,
                "errors": validation["errors"]
            }
        
        # Instantiate template
        try:
            instantiated = template.instantiate(variables)
        except Exception as e:
            return {
                "workflow": None,
                "template_validation": validation,
                "success": False,
                "errors": [f"Template instantiation failed: {str(e)}"]
            }
        
        # Convert to WorkflowSpec
        try:
            workflow = self._template_to_workflow(instantiated)
        except Exception as e:
            return {
                "workflow": None,
                "template_validation": validation,
                "success": False,
                "errors": [f"Workflow conversion failed: {str(e)}"]
            }
        
        return {
            "workflow": workflow,
            "template_validation": validation,
            "success": True,
            "errors": []
        }
    
    def _template_to_workflow(self, template: WorkflowTemplate) -> WorkflowSpec:
        """Convert instantiated template to WorkflowSpec."""
        
        # Create nodes
        nodes = []
        
        # Add START node
        start_node_id = f"{template.id}_start"
        nodes.append(WorkflowNode(
            id=start_node_id,
            type=NodeType.START,
            name="Start",
            description="Workflow start",
            tags=["system"]
        ))
        
        # Convert template nodes
        for tnode in template.nodes:
            nodes.append(WorkflowNode(
                id=tnode.id,
                type=NodeType.TOOL_CALL,
                name=tnode.name,
                description=tnode.description,
                operator=tnode.operator,
                config=tnode.config,
                retry_policy=tnode.retry_policy,
                timeout_seconds=tnode.timeout_seconds,
                tags=["action"]
            ))
        
        # Add END node
        end_node_id = f"{template.id}_end"
        nodes.append(WorkflowNode(
            id=end_node_id,
            type=NodeType.END,
            name="End",
            description="Workflow end",
            tags=["system"]
        ))
        
        # Create edges
        edges = []
        
        # Connect START to first nodes
        first_nodes = [n for n in template.nodes if not n.dependencies]
        for node in first_nodes:
            edges.append(WorkflowEdge(
                source=start_node_id,
                target=node.id,
                condition="success",
                label="start"
            ))
        
        # Connect nodes based on dependencies
        for node in template.nodes:
            for dep_id in node.dependencies:
                edges.append(WorkflowEdge(
                    source=dep_id,
                    target=node.id,
                    condition="success",
                    label=f"after_{dep_id}"
                ))
        
        # Connect last nodes to END
        last_nodes = [
            n for n in template.nodes
            if not any(n.id in other.dependencies for other in template.nodes)
        ]
        for node in last_nodes:
            edges.append(WorkflowEdge(
                source=node.id,
                target=end_node_id,
                condition="success",
                label="complete"
            ))
        
        # Create WorkflowSpec
        workflow = WorkflowSpec(
            id=template.id,
            name=template.name,
            description=template.description,
            version=template.version,
            nodes=nodes,
            edges=edges,
            start_node_id=start_node_id,
            end_node_ids=[end_node_id],
            tags=template.tags + ["template-based"],
            metadata={"generated_from_template": True}
        )
        
        return workflow
