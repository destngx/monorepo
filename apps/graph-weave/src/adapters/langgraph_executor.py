"""
Compatibility shim for the LangGraph executor package.

The real implementation lives in `src.adapters.langgraph`.
"""

from .langgraph import (
    ExecutorState,
    BaseLangGraphExecutor,
    RealLangGraphExecutor,
)

__all__ = [
    "ExecutorState",
    "BaseLangGraphExecutor",
    "RealLangGraphExecutor",
]
