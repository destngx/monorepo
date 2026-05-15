"""
Benchmarking Suite for Workflow Generation Patterns

Measures:
- Latency: Time to generate workflow
- Accuracy: Correctness of generated workflow
- Resource Usage: Memory and CPU
"""

from .benchmark import WorkflowBenchmark, BenchmarkResult

__all__ = ["WorkflowBenchmark", "BenchmarkResult"]
