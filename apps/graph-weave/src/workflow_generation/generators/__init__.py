"""
Workflow Generators

Deterministic conversion from IntentExtraction → WorkflowSpec.
Based on research: A²Flow (operator discovery), Template-based generation.
"""

from .intent_to_workflow import IntentToWorkflowGenerator

__all__ = ["IntentToWorkflowGenerator"]
