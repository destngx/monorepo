"""
Workflow Generation Module

Pattern 1: Intent → Structured Workflow

This module implements deterministic workflow generation from natural language intent.

Key principles:
1. LLM extracts intent ONCE (single invocation)
2. All subsequent steps are deterministic validation and DAG construction
3. Full auditability: every decision is traceable
4. No LLM in the loop after intent extraction

Architecture:
- IntentExtraction: Pydantic schema for LLM output
- IntentValidator: 4-stage validation (schema, semantic, operator, constraint)
- IntentToWorkflowGenerator: Deterministic DAG construction
- WorkflowSpec: Validated, executable workflow representation
- WorkflowValidator: Structural integrity checks

References:
- Compiled AI (arXiv 2604.05150): 4-stage validation pattern
- A²Flow (AAAI 2026): Operator discovery and dynamic composition
- FlowMind (arXiv 2602.11782): Execute-summarize pattern
"""

from .schemas import (
    IntentExtraction,
    IntentAction,
    IntentParameter,
    WorkflowSpec,
    WorkflowNode,
    WorkflowEdge,
    NodeType,
)

from .validators import (
    IntentValidator,
    WorkflowValidator,
)

from .generators import (
    IntentToWorkflowGenerator,
)

__all__ = [
    # Schemas
    "IntentExtraction",
    "IntentAction",
    "IntentParameter",
    "WorkflowSpec",
    "WorkflowNode",
    "WorkflowEdge",
    "NodeType",
    # Validators
    "IntentValidator",
    "WorkflowValidator",
    # Generators
    "IntentToWorkflowGenerator",
]
