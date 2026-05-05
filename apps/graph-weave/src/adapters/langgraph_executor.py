"""
Compatibility shim for the LangGraph executor package.

The real implementation lives in `src.adapters.langgraph`.
"""

from .langgraph import (
    ExecutorState,
    BaseLangGraphExecutor,
    MockLangGraphExecutor,
    RealLangGraphExecutor,
)

__all__ = [
    "ExecutorState",
    "BaseLangGraphExecutor",
    "MockLangGraphExecutor",
    "RealLangGraphExecutor",
]
