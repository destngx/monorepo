"""
Workflow Generation Benchmarking

Measures latency, accuracy, and resource usage across patterns.
"""

import time
import psutil
import os
from typing import Dict, List, Any, Callable
from dataclasses import dataclass, asdict
from pydantic import BaseModel, Field
import json


@dataclass
class BenchmarkResult:
    """Result of a single benchmark run."""
    
    pattern: str
    workflow_name: str
    latency_ms: float
    memory_mb: float
    cpu_percent: float
    success: bool
    errors: List[str]
    metadata: Dict[str, Any]


class WorkflowBenchmark:
    """Benchmarking suite for workflow generation patterns."""
    
    def __init__(self):
        """Initialize benchmark suite."""
        self.results: List[BenchmarkResult] = []
        self.process = psutil.Process(os.getpid())
    
    def benchmark_pattern1(
        self,
        intent_factory: Callable,
        num_runs: int = 5
    ) -> List[BenchmarkResult]:
        """
        Benchmark Pattern 1: Intent → Structured Workflow.
        
        Args:
            intent_factory: Function that returns IntentExtraction
            num_runs: Number of benchmark runs
        
        Returns:
            List of BenchmarkResult
        """
        from ..generators import IntentToWorkflowGenerator
        
        results = []
        generator = IntentToWorkflowGenerator()
        
        for run in range(num_runs):
            intent = intent_factory()
            
            # Measure
            start_time = time.time()
            start_memory = self.process.memory_info().rss / 1024 / 1024
            start_cpu = self.process.cpu_percent()
            
            result = generator.generate(intent)
            
            end_time = time.time()
            end_memory = self.process.memory_info().rss / 1024 / 1024
            end_cpu = self.process.cpu_percent()
            
            # Record
            latency_ms = (end_time - start_time) * 1000
            memory_mb = end_memory - start_memory
            
            benchmark_result = BenchmarkResult(
                pattern="Pattern 1: Intent→Workflow",
                workflow_name=intent.goal,
                latency_ms=latency_ms,
                memory_mb=memory_mb,
                cpu_percent=end_cpu,
                success=result["success"],
                errors=result["errors"],
                metadata={
                    "run": run + 1,
                    "action_count": len(intent.actions),
                    "node_count": len(result["workflow"].nodes) if result["workflow"] else 0
                }
            )
            
            results.append(benchmark_result)
            self.results.append(benchmark_result)
        
        return results
    
    def benchmark_pattern2(
        self,
        num_runs: int = 5
    ) -> List[BenchmarkResult]:
        """
        Benchmark Pattern 2: Operator Registry.
        
        Args:
            num_runs: Number of benchmark runs
        
        Returns:
            List of BenchmarkResult
        """
        from ..operators import get_global_registry, OperatorCapability
        
        results = []
        registry = get_global_registry()
        
        for run in range(num_runs):
            # Measure
            start_time = time.time()
            start_memory = self.process.memory_info().rss / 1024 / 1024
            start_cpu = self.process.cpu_percent()
            
            # Perform various registry operations
            all_ops = registry.list_all()
            search_results = registry.search("search")
            capability_results = registry.find_by_capability(OperatorCapability.DATA_FETCH)
            stats = registry.get_stats()
            
            end_time = time.time()
            end_memory = self.process.memory_info().rss / 1024 / 1024
            end_cpu = self.process.cpu_percent()
            
            # Record
            latency_ms = (end_time - start_time) * 1000
            memory_mb = end_memory - start_memory
            
            benchmark_result = BenchmarkResult(
                pattern="Pattern 2: Operator Registry",
                workflow_name="Registry Operations",
                latency_ms=latency_ms,
                memory_mb=memory_mb,
                cpu_percent=end_cpu,
                success=True,
                errors=[],
                metadata={
                    "run": run + 1,
                    "total_operators": len(all_ops),
                    "search_results": len(search_results),
                    "capability_results": len(capability_results)
                }
            )
            
            results.append(benchmark_result)
            self.results.append(benchmark_result)
        
        return results
    
    def benchmark_pattern3(
        self,
        template_factory: Callable,
        variables_factory: Callable,
        num_runs: int = 5
    ) -> List[BenchmarkResult]:
        """
        Benchmark Pattern 3: Template-based Workflows.
        
        Args:
            template_factory: Function that returns WorkflowTemplate
            variables_factory: Function that returns variables dict
            num_runs: Number of benchmark runs
        
        Returns:
            List of BenchmarkResult
        """
        from ..templates import TemplateWorkflowGenerator
        
        results = []
        generator = TemplateWorkflowGenerator()
        
        for run in range(num_runs):
            template = template_factory()
            variables = variables_factory()
            
            # Measure
            start_time = time.time()
            start_memory = self.process.memory_info().rss / 1024 / 1024
            start_cpu = self.process.cpu_percent()
            
            result = generator.generate(template, variables)
            
            end_time = time.time()
            end_memory = self.process.memory_info().rss / 1024 / 1024
            end_cpu = self.process.cpu_percent()
            
            # Record
            latency_ms = (end_time - start_time) * 1000
            memory_mb = end_memory - start_memory
            
            benchmark_result = BenchmarkResult(
                pattern="Pattern 3: Template-based",
                workflow_name=template.name,
                latency_ms=latency_ms,
                memory_mb=memory_mb,
                cpu_percent=end_cpu,
                success=result["success"],
                errors=result["errors"],
                metadata={
                    "run": run + 1,
                    "variable_count": len(template.variables),
                    "node_count": len(template.nodes),
                    "workflow_node_count": len(result["workflow"].nodes) if result["workflow"] else 0
                }
            )
            
            results.append(benchmark_result)
            self.results.append(benchmark_result)
        
        return results
    
    def get_summary(self) -> Dict[str, Any]:
        """Get summary statistics across all benchmarks."""
        if not self.results:
            return {}
        
        # Group by pattern
        by_pattern = {}
        for result in self.results:
            if result.pattern not in by_pattern:
                by_pattern[result.pattern] = []
            by_pattern[result.pattern].append(result)
        
        # Calculate statistics
        summary = {}
        for pattern, results in by_pattern.items():
            latencies = [r.latency_ms for r in results]
            memories = [r.memory_mb for r in results]
            
            summary[pattern] = {
                "runs": len(results),
                "success_rate": sum(1 for r in results if r.success) / len(results),
                "latency": {
                    "min_ms": min(latencies),
                    "max_ms": max(latencies),
                    "avg_ms": sum(latencies) / len(latencies),
                    "median_ms": sorted(latencies)[len(latencies) // 2]
                },
                "memory": {
                    "min_mb": min(memories),
                    "max_mb": max(memories),
                    "avg_mb": sum(memories) / len(memories)
                }
            }
        
        return summary
    
    def to_json(self) -> str:
        """Serialize results to JSON."""
        results_dict = [asdict(r) for r in self.results]
        return json.dumps(results_dict, indent=2)
    
    def print_summary(self):
        """Print summary statistics."""
        summary = self.get_summary()
        
        print("\n" + "=" * 80)
        print("WORKFLOW GENERATION BENCHMARKS")
        print("=" * 80)
        
        for pattern, stats in summary.items():
            print(f"\n{pattern}")
            print("-" * 80)
            print(f"  Runs: {stats['runs']}")
            print(f"  Success Rate: {stats['success_rate']:.1%}")
            print(f"  Latency:")
            print(f"    Min: {stats['latency']['min_ms']:.2f}ms")
            print(f"    Max: {stats['latency']['max_ms']:.2f}ms")
            print(f"    Avg: {stats['latency']['avg_ms']:.2f}ms")
            print(f"    Median: {stats['latency']['median_ms']:.2f}ms")
            print(f"  Memory:")
            print(f"    Min: {stats['memory']['min_mb']:.2f}MB")
            print(f"    Max: {stats['memory']['max_mb']:.2f}MB")
            print(f"    Avg: {stats['memory']['avg_mb']:.2f}MB")
