"""
Benchmark Tests for Workflow Generation Patterns

Tests latency, accuracy, and resource usage.
"""

import pytest
from src.models import (
    IntentExtraction,
    IntentAction,
    IntentParameter,
)
from src.services import (
    WorkflowTemplate,
    TemplateVariable,
    TemplateNode,
)
from src.services.benchmarks import WorkflowBenchmark


class TestBenchmarks:
    """Benchmark tests for all patterns."""
    
    @pytest.fixture
    def benchmark_suite(self):
        """Create benchmark suite."""
        return WorkflowBenchmark()
    
    def create_simple_intent(self):
        """Create a simple intent for benchmarking."""
        return IntentExtraction(
            goal="Simple workflow",
            description="A simple linear workflow",
            actions=[
                IntentAction(
                    id="action1",
                    name="Action 1",
                    description="First action",
                    operator="op1"
                ),
                IntentAction(
                    id="action2",
                    name="Action 2",
                    description="Second action",
                    operator="op2",
                    dependencies=["action1"]
                )
            ]
        )
    
    def create_complex_intent(self):
        """Create a complex intent for benchmarking."""
        return IntentExtraction(
            goal="Complex research workflow",
            description="Multi-step research with parallel operations",
            actions=[
                IntentAction(
                    id="search_web",
                    name="Search Web",
                    description="Search the web",
                    operator="web_search"
                ),
                IntentAction(
                    id="search_academic",
                    name="Search Academic",
                    description="Search academic databases",
                    operator="academic_search"
                ),
                IntentAction(
                    id="fetch_results",
                    name="Fetch Results",
                    description="Fetch full content",
                    operator="fetch_url",
                    dependencies=["search_web", "search_academic"]
                ),
                IntentAction(
                    id="analyze",
                    name="Analyze",
                    description="Analyze content",
                    operator="llm_call",
                    dependencies=["fetch_results"]
                ),
                IntentAction(
                    id="summarize",
                    name="Summarize",
                    description="Create summary",
                    operator="llm_call",
                    dependencies=["analyze"]
                )
            ]
        )
    
    def create_simple_template(self):
        """Create a simple template for benchmarking."""
        return WorkflowTemplate(
            id="simple_template",
            name="Simple Template",
            description="A simple template",
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
                    description="Search",
                    operator="web_search",
                    config={"query": "{{query}}"}
                )
            ],
            start_node_id="search",
            end_node_ids=["search"]
        )
    
    def test_pattern1_simple_latency(self, benchmark_suite):
        """Test Pattern 1 latency with simple workflow."""
        results = benchmark_suite.benchmark_pattern1(
            intent_factory=self.create_simple_intent,
            num_runs=3
        )
        
        assert len(results) == 3
        assert all(r.success for r in results)
        
        # Latency should be reasonable (< 100ms)
        avg_latency = sum(r.latency_ms for r in results) / len(results)
        assert avg_latency < 100
    
    def test_pattern1_complex_latency(self, benchmark_suite):
        """Test Pattern 1 latency with complex workflow."""
        results = benchmark_suite.benchmark_pattern1(
            intent_factory=self.create_complex_intent,
            num_runs=3
        )
        
        assert len(results) == 3
        assert all(r.success for r in results)
        
        # Complex workflows should still be fast
        avg_latency = sum(r.latency_ms for r in results) / len(results)
        assert avg_latency < 200
    
    def test_pattern2_registry_latency(self, benchmark_suite):
        """Test Pattern 2 registry operations latency."""
        results = benchmark_suite.benchmark_pattern2(num_runs=3)
        
        assert len(results) == 3
        assert all(r.success for r in results)
        
        # Registry operations should be very fast
        avg_latency = sum(r.latency_ms for r in results) / len(results)
        assert avg_latency < 50
    
    def test_pattern3_template_latency(self, benchmark_suite):
        """Test Pattern 3 template generation latency."""
        results = benchmark_suite.benchmark_pattern3(
            template_factory=self.create_simple_template,
            variables_factory=lambda: {"query": "test"},
            num_runs=3
        )
        
        assert len(results) == 3
        assert all(r.success for r in results)
        
        # Template generation should be fast
        avg_latency = sum(r.latency_ms for r in results) / len(results)
        assert avg_latency < 100
    
    def test_benchmark_summary(self, benchmark_suite):
        """Test benchmark summary generation."""
        # Run benchmarks
        benchmark_suite.benchmark_pattern1(
            intent_factory=self.create_simple_intent,
            num_runs=2
        )
        benchmark_suite.benchmark_pattern2(num_runs=2)
        benchmark_suite.benchmark_pattern3(
            template_factory=self.create_simple_template,
            variables_factory=lambda: {"query": "test"},
            num_runs=2
        )
        
        # Get summary
        summary = benchmark_suite.get_summary()
        
        assert len(summary) == 3
        assert all("latency" in stats for stats in summary.values())
        assert all("memory" in stats for stats in summary.values())
        assert all("success_rate" in stats for stats in summary.values())


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
