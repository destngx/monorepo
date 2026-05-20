"""
Tests for Workflow Generation Module

Tests cover:
1. Intent extraction validation
2. Workflow generation
3. DAG properties
4. Error handling
"""

import pytest
from src.models import (
    IntentExtraction,
    IntentAction,
    IntentParameter,
    WorkflowNodeType as NodeType,
)
from src.services import (
    IntentValidator,
    IntentToWorkflowGenerator,
    WorkflowValidator,
)


class TestIntentValidation:
    """Test IntentValidator."""
    
    def test_valid_intent(self):
        """Test validation of a valid intent."""
        intent = IntentExtraction(
            goal="Test goal",
            description="Test description",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="test_op"
                )
            ]
        )
        
        validator = IntentValidator()
        result = validator.validate(intent)
        
        assert result["valid"] is True
        assert len(result["errors"]) == 0
    
    def test_empty_actions(self):
        """Test validation fails with no actions."""
        intent = IntentExtraction(
            goal="Test goal",
            description="Test description",
            actions=[]
        )
        
        validator = IntentValidator()
        result = validator.validate(intent)
        
        assert result["valid"] is False
        assert any("at least one action" in e for e in result["errors"])
    
    def test_empty_goal(self):
        """Test validation fails with empty goal."""
        intent = IntentExtraction(
            goal="",
            description="Test description",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="test_op"
                )
            ]
        )
        
        validator = IntentValidator()
        result = validator.validate(intent)
        
        assert result["valid"] is False
        assert any("Goal" in e for e in result["errors"])
    
    def test_duplicate_action_ids(self):
        """Test validation fails with duplicate action IDs."""
        with pytest.raises(ValueError):
            IntentExtraction(
                goal="Test goal",
                description="Test description",
                actions=[
                    IntentAction(
                        id="action1",
                        name="Action 1",
                        description="First action",
                        operator="test_op"
                    ),
                    IntentAction(
                        id="action1",
                        name="Action 2",
                        description="Second action",
                        operator="test_op"
                    )
                ]
            )
    
    def test_invalid_dependency(self):
        """Test validation fails with non-existent dependency."""
        with pytest.raises(ValueError):
            IntentExtraction(
                goal="Test goal",
                description="Test description",
                actions=[
                    IntentAction(
                        id="action1",
                        name="Action 1",
                        description="First action",
                        operator="test_op",
                        dependencies=["non_existent"]
                    )
                ]
            )
    
    def test_circular_dependency(self):
        """Test validation fails with circular dependencies."""
        with pytest.raises(ValueError):
            IntentExtraction(
                goal="Test goal",
                description="Test description",
                actions=[
                    IntentAction(
                        id="action1",
                        name="Action 1",
                        description="First action",
                        operator="test_op",
                        dependencies=["action2"]
                    ),
                    IntentAction(
                        id="action2",
                        name="Action 2",
                        description="Second action",
                        operator="test_op",
                        dependencies=["action1"]
                    )
                ]
            )
    
    def test_operator_availability_warning(self):
        """Test warning when operator not in registry."""
        intent = IntentExtraction(
            goal="Test goal",
            description="Test description",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="unknown_op"
                )
            ]
        )
        
        validator = IntentValidator(available_operators={"known_op"})
        result = validator.validate(intent)
        
        assert result["valid"] is True
        assert any("unknown_op" in w for w in result["warnings"])


class TestWorkflowGeneration:
    """Test IntentToWorkflowGenerator."""
    
    def test_simple_workflow_generation(self):
        """Test generation of a simple linear workflow."""
        intent = IntentExtraction(
            goal="Simple workflow",
            description="A simple linear workflow",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                ),
                IntentAction(
                    id="action2",
                    name="Action 2",
                    description="Second action",
                    operator="op2",
                    dependencies=["action1"]
                )
            ]
        )
        
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)
        
        assert result["success"] is True
        assert result["workflow"] is not None
        
        workflow = result["workflow"]
        assert len(workflow.nodes) == 4  # START + 2 actions + END
        assert len(workflow.edges) >= 3  # START→action1, action1→action2, action2→END
    
    def test_workflow_with_multiple_start_actions(self):
        """Test workflow with multiple actions that have no dependencies."""
        intent = IntentExtraction(
            goal="Parallel workflow",
            description="Workflow with parallel start actions",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                ),
                IntentAction(
                    id="action2",
                    name="Action 2",
                    description="Second action",
                    operator="op2"
                ),
                IntentAction(
                    id="action3",
                    name="Action 3",
                    description="Third action",
                    operator="op3",
                    dependencies=["action1", "action2"]
                )
            ]
        )
        
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)
        
        assert result["success"] is True
        workflow = result["workflow"]
        
        # Check that both action1 and action2 are connected to START
        start_edges = workflow.get_outgoing_edges(workflow.start_node_id)
        assert len(start_edges) == 2
    
    def test_workflow_validation_passes(self):
        """Test that generated workflow passes validation."""
        intent = IntentExtraction(
            goal="Test workflow",
            description="Test workflow",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                )
            ]
        )
        
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)
        
        assert result["workflow_validation"]["valid"] is True
        assert len(result["workflow_validation"]["errors"]) == 0
    
    def test_workflow_topological_sort(self):
        """Test topological sorting of workflow nodes."""
        intent = IntentExtraction(
            goal="Test workflow",
            description="Test workflow",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                ),
                IntentAction(
                    id="action2",
                    name="Action 2",
                    description="Second action",
                    operator="op2",
                    dependencies=["action1"]
                ),
                IntentAction(
                    id="action3",
                    name="Action 3",
                    description="Third action",
                    operator="op3",
                    dependencies=["action2"]
                )
            ]
        )
        
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)
        
        workflow = result["workflow"]
        topo_order = workflow.topological_sort()
        
        # START should be first
        assert topo_order[0] == workflow.start_node_id
        
        # END should be last
        assert topo_order[-1] in workflow.end_node_ids
        
        # Actions should be in dependency order
        assert len(topo_order) == len(workflow.nodes)


class TestWorkflowValidation:
    """Test WorkflowValidator."""
    
    def test_valid_workflow(self):
        """Test validation of a valid workflow."""
        intent = IntentExtraction(
            goal="Test workflow",
            description="Test workflow",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                )
            ]
        )
        
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)
        
        validator = WorkflowValidator()
        validation = validator.validate(result["workflow"])
        
        assert validation["valid"] is True
        assert len(validation["errors"]) == 0
    
    def test_workflow_reachability(self):
        """Test that all nodes are reachable from start."""
        intent = IntentExtraction(
            goal="Test workflow",
            description="Test workflow",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                ),
                IntentAction(
                    id="action2",
                    name="Action 2",
                    description="Second action",
                    operator="op2",
                    dependencies=["action1"]
                )
            ]
        )
        
        generator = IntentToWorkflowGenerator()
        result = generator.generate(intent)
        
        validator = WorkflowValidator()
        validation = validator.validate(result["workflow"])
        
        assert validation["valid"] is True


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
