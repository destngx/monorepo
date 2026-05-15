"""
Template-based Workflow Generation

Pattern 3: Template-based workflow generation with variable substitution.

Supports:
- YAML/JSON workflow templates
- Variable substitution
- Template validation
- Template composition
"""

from .template import WorkflowTemplate, TemplateVariable, TemplateNode
from .generator import TemplateWorkflowGenerator

__all__ = ["WorkflowTemplate", "TemplateVariable", "TemplateNode", "TemplateWorkflowGenerator"]
