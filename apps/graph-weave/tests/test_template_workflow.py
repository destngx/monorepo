"""
Tests for Template-based Workflow Generation

Tests cover:
1. Template validation
2. Variable substitution
3. Template instantiation
4. Workflow generation from templates
"""

import pytest
from src.services import (
    WorkflowTemplate,
    TemplateVariable,
    TemplateNode,
    TemplateWorkflowGenerator,
)


class TestTemplateVariable:
    """Test TemplateVariable."""
    
    def test_validate_string(self):
        """Test string variable validation."""
        var = TemplateVariable(
            name="query",
            type="string",
            description="Search query",
            required=True
        )
        
        assert var.validate_value("test query") is True
        assert var.validate_value(123) is False
    
    def test_validate_number(self):
        """Test number variable validation."""
        var = TemplateVariable(
            name="max_results",
            type="number",
            description="Max results",
            required=False,
            default=10
        )
        
        assert var.validate_value(10) is True
        assert var.validate_value(10.5) is True
        assert var.validate_value("10") is False
    
    def test_validate_pattern(self):
        """Test pattern validation."""
        var = TemplateVariable(
            name="email",
            type="string",
            description="Email address",
            pattern=r"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$"
        )
        
        assert var.validate_value("test@example.com") is True
        assert var.validate_value("invalid-email") is False
    
    def test_validate_required(self):
        """Test required field validation."""
        var = TemplateVariable(
            name="required_field",
            type="string",
            description="A required field",
            required=True
        )
        
        assert var.validate_value("value") is True
        assert var.validate_value(None) is False


class TestWorkflowTemplate:
    """Test WorkflowTemplate."""
    
    def test_create_template(self):
        """Test creating a workflow template."""
        template = WorkflowTemplate(
            id="search_template",
            name="Search Template",
            description="A search workflow template",
            variables=[
                TemplateVariable(
                    name="query",
                    type="string",
                    description="Search query",
                    required=True
                )
            ],
            nodes=[
                TemplateNode(
                    id="search",
                    name="Search",
                    description="Search the web",
                    operator="web_search",
                    config={"query": "{{query}}"}
                )
            ],
            start_node_id="search",
            end_node_ids=["search"]
        )
        
        assert template.id == "search_template"
        assert len(template.variables) == 1
        assert len(template.nodes) == 1
    
    def test_validate_variables(self):
        """Test variable validation."""
        template = WorkflowTemplate(
            id="test",
            name="Test",
            description="Test",
            variables=[
                TemplateVariable(
                    name="query",
                    type="string",
                    description="Search query",
                    required=True
                )
            ],
            nodes=[
                TemplateNode(
                    id="node1",
                    name="Node 1",
                    description="Test node",
                    operator="test_op"
                )
            ],
            start_node_id="node1",
            end_node_ids=["node1"]
        )
        
        # Valid
        result = template.validate_variables({"query": "test"})
        assert result["valid"] is True
        
        # Missing required
        result = template.validate_variables({})
        assert result["valid"] is False
        assert any("Required" in e for e in result["errors"])
    
    def test_instantiate_template(self):
        """Test template instantiation with variable substitution."""
        template = WorkflowTemplate(
            id="search_template",
            name="Search Template",
            description="A search workflow template",
            variables=[
                TemplateVariable(
                    name="query",
                    type="string",
                    description="Search query",
                    required=True
                ),
                TemplateVariable(
                    name="max_results",
                    type="number",
                    description="Max results",
                    required=False,
                    default=10
                )
            ],
            nodes=[
                TemplateNode(
                    id="search",
                    name="Search",
                    description="Search the web",
                    operator="web_search",
                    config={
                        "query": "{{query}}",
                        "max_results": "{{max_results}}"
                    }
                )
            ],
            start_node_id="search",
            end_node_ids=["search"]
        )
        
        # Instantiate
        instance = template.instantiate({
            "query": "machine learning",
            "max_results": 20
        })
        
        # Check substitution
        search_node = instance.nodes[0]
        assert search_node.config["query"] == "machine learning"
        assert search_node.config["max_results"] == 20
    
    def test_get_required_variables(self):
        """Test getting required variables."""
        template = WorkflowTemplate(
            id="test",
            name="Test",
            description="Test",
            variables=[
                TemplateVariable(
                    name="required_var",
                    type="string",
                    description="A required variable",
                    required=True
                ),
                TemplateVariable(
                    name="optional_var",
                    type="string",
                    description="An optional variable",
                    required=False
                )
            ],
            nodes=[
                TemplateNode(
                    id="node1",
                    name="Node 1",
                    description="Test",
                    operator="test_op"
                )
            ],
            start_node_id="node1",
            end_node_ids=["node1"]
        )
        
        required = template.get_required_variables()
        assert len(required) == 1
        assert required[0].name == "required_var"


class TestTemplateWorkflowGenerator:
    """Test TemplateWorkflowGenerator."""
    
    def test_generate_from_template(self):
        """Test generating workflow from template."""
        template = WorkflowTemplate(
            id="search_template",
            name="Search Template",
            description="A search workflow template",
            variables=[
                TemplateVariable(
                    name="query",
                    type="string",
                    description="Search query",
                    required=True
                )
            ],
            nodes=[
                TemplateNode(
                    id="search",
                    name="Search",
                    description="Search the web",
                    operator="web_search",
                    config={"query": "{{query}}"}
                )
            ],
            start_node_id="search",
            end_node_ids=["search"]
        )
        
        generator = TemplateWorkflowGenerator()
        result = generator.generate(template, {"query": "test"})
        
        assert result["success"] is True
        assert result["workflow"] is not None
        
        workflow = result["workflow"]
        assert len(workflow.nodes) == 3  # START + search + END
    
    def test_generate_with_dependencies(self):
        """Test generating workflow with node dependencies."""
        template = WorkflowTemplate(
            id="pipeline",
            name="Pipeline",
            description="A pipeline template",
            variables=[],
            nodes=[
                TemplateNode(
                    id="fetch",
                    name="Fetch",
                    description="Fetch data",
                    operator="fetch_url"
                ),
                TemplateNode(
                    id="process",
                    name="Process",
                    description="Process data",
                    operator="python_script",
                    dependencies=["fetch"]
                )
            ],
            start_node_id="fetch",
            end_node_ids=["process"]
        )
        
        generator = TemplateWorkflowGenerator()
        result = generator.generate(template, {})
        
        assert result["success"] is True
        workflow = result["workflow"]
        
        # Check edges
        process_edges = workflow.get_incoming_edges("process")
        assert len(process_edges) > 0
    
    def test_generate_with_invalid_variables(self):
        """Test generation fails with invalid variables."""
        template = WorkflowTemplate(
            id="test",
            name="Test",
            description="Test",
            variables=[
                TemplateVariable(
                    name="query",
                    type="string",
                    description="Search query",
                    required=True
                )
            ],
            nodes=[
                TemplateNode(
                    id="node1",
                    name="Node 1",
                    description="Test",
                    operator="test_op"
                )
            ],
            start_node_id="node1",
            end_node_ids=["node1"]
        )
        
        generator = TemplateWorkflowGenerator()
        result = generator.generate(template, {})  # Missing required variable
        
        assert result["success"] is False
        assert len(result["errors"]) > 0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
