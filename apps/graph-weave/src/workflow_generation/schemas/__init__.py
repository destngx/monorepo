"""
Workflow Generation Schemas

Pattern 1: Intent → Structured Workflow
- Intent extraction from natural language
- Structured workflow representation
- Deterministic DAG construction
"""

from .intent import IntentExtraction, IntentAction, IntentParameter
from .workflow import WorkflowSpec, WorkflowNode, WorkflowEdge, NodeType

__all__ = [
    "IntentExtraction",
    "IntentAction",
    "IntentParameter",
    "WorkflowSpec",
    "WorkflowNode",
    "WorkflowEdge",
    "NodeType",
]
