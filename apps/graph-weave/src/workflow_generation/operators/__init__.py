"""
Operator Registry

A²Flow-style operator discovery and dynamic composition.
Enables runtime operator selection and composition.
"""

from .registry import (
    OperatorRegistry,
    OperatorDefinition,
    OperatorCapability,
    get_global_registry,
    register_operator,
)

# Import built-in operators to auto-register them
from . import builtin

__all__ = [
    "OperatorRegistry",
    "OperatorDefinition",
    "OperatorCapability",
    "get_global_registry",
    "register_operator",
]
