"""
Tests for Operator Registry

Tests cover:
1. Operator registration
2. Operator discovery
3. Capability indexing
4. Tag-based search
"""

import pytest
from src.models import OperatorCapability
from src.services import (
    OperatorRegistry,
    OperatorDefinition,
    get_global_registry,
)


class TestOperatorRegistry:
    """Test OperatorRegistry."""
    
    def test_register_operator(self):
        """Test registering an operator."""
        registry = OperatorRegistry()
        
        operator = OperatorDefinition(
            id="test_op",
            name="Test Operator",
            description="A test operator",
            capability=OperatorCapability.DATA_FETCH
        )
        
        registry.register(operator)
        
        assert registry.get("test_op") is not None
        assert registry.get("test_op").name == "Test Operator"
    
    def test_find_by_capability(self):
        """Test finding operators by capability."""
        registry = OperatorRegistry()
        
        op1 = OperatorDefinition(
            id="fetch_op",
            name="Fetch",
            description="Fetch data",
            capability=OperatorCapability.DATA_FETCH
        )
        
        op2 = OperatorDefinition(
            id="process_op",
            name="Process",
            description="Process data",
            capability=OperatorCapability.DATA_PROCESS
        )
        
        registry.register(op1)
        registry.register(op2)
        
        fetch_ops = registry.find_by_capability(OperatorCapability.DATA_FETCH)
        assert len(fetch_ops) == 1
        assert fetch_ops[0].id == "fetch_op"
    
    def test_find_by_tag(self):
        """Test finding operators by tag."""
        registry = OperatorRegistry()
        
        operator = OperatorDefinition(
            id="test_op",
            name="Test",
            description="Test operator",
            capability=OperatorCapability.DATA_FETCH,
            tags=["external", "http"]
        )
        
        registry.register(operator)
        
        results = registry.find_by_tag("external")
        assert len(results) == 1
        assert results[0].id == "test_op"
    
    def test_search_by_name(self):
        """Test searching operators by name."""
        registry = OperatorRegistry()
        
        operator = OperatorDefinition(
            id="web_search",
            name="Web Search",
            description="Search the web",
            capability=OperatorCapability.SEARCH
        )
        
        registry.register(operator)
        
        results = registry.search("web")
        assert len(results) == 1
        assert results[0].id == "web_search"
    
    def test_search_by_description(self):
        """Test searching operators by description."""
        registry = OperatorRegistry()
        
        operator = OperatorDefinition(
            id="fetch_op",
            name="Fetch",
            description="Fetch content from URLs",
            capability=OperatorCapability.DATA_FETCH
        )
        
        registry.register(operator)
        
        results = registry.search("content")
        assert len(results) == 1
        assert results[0].id == "fetch_op"
    
    def test_search_with_capability_filter(self):
        """Test searching with capability filter."""
        registry = OperatorRegistry()
        
        op1 = OperatorDefinition(
            id="fetch_op",
            name="Fetch",
            description="Fetch data",
            capability=OperatorCapability.DATA_FETCH
        )
        
        op2 = OperatorDefinition(
            id="process_op",
            name="Process",
            description="Process data",
            capability=OperatorCapability.DATA_PROCESS
        )
        
        registry.register(op1)
        registry.register(op2)
        
        results = registry.search(
            "data",
            capability=OperatorCapability.DATA_FETCH
        )
        assert len(results) == 1
        assert results[0].id == "fetch_op"
    
    def test_list_all(self):
        """Test listing all operators."""
        registry = OperatorRegistry()
        
        op1 = OperatorDefinition(
            id="op1",
            name="Op 1",
            description="First",
            capability=OperatorCapability.DATA_FETCH
        )
        
        op2 = OperatorDefinition(
            id="op2",
            name="Op 2",
            description="Second",
            capability=OperatorCapability.DATA_PROCESS
        )
        
        registry.register(op1)
        registry.register(op2)
        
        all_ops = registry.list_all()
        assert len(all_ops) == 2
    
    def test_get_stats(self):
        """Test getting registry statistics."""
        registry = OperatorRegistry()
        
        operator = OperatorDefinition(
            id="test_op",
            name="Test",
            description="Test",
            capability=OperatorCapability.DATA_FETCH,
            tags=["external"]
        )
        
        registry.register(operator)
        
        stats = registry.get_stats()
        assert stats["total_operators"] == 1
        assert OperatorCapability.DATA_FETCH.value in stats["capabilities"]
        assert "external" in stats["tags"]
    
    def test_global_registry(self):
        """Test global registry instance."""
        registry = get_global_registry()
        
        # Should have built-in operators
        all_ops = registry.list_all()
        assert len(all_ops) > 0
        
        # Should have web_search
        web_search = registry.get("web_search")
        assert web_search is not None
        assert web_search.name == "Web Search"


class TestBuiltinOperators:
    """Test built-in operators."""
    
    def test_web_search_operator(self):
        """Test web_search operator definition."""
        registry = get_global_registry()
        op = registry.get("web_search")
        
        assert op is not None
        assert op.capability == OperatorCapability.SEARCH
        assert "search" in op.tags
        assert op.timeout_seconds == 30
    
    def test_fetch_url_operator(self):
        """Test fetch_url operator definition."""
        registry = get_global_registry()
        op = registry.get("fetch_url")
        
        assert op is not None
        assert op.capability == OperatorCapability.DATA_FETCH
        assert "http" in op.tags
        assert op.max_retries == 3
    
    def test_llm_call_operator(self):
        """Test llm_call operator definition."""
        registry = get_global_registry()
        op = registry.get("llm_call")
        
        assert op is not None
        assert op.capability == OperatorCapability.LLM_CALL
        assert not op.is_deterministic
    
    def test_python_script_operator(self):
        """Test python_script operator definition."""
        registry = get_global_registry()
        op = registry.get("python_script")
        
        assert op is not None
        assert op.capability == OperatorCapability.COMPUTE
        assert op.is_deterministic
        assert op.cost_per_call == 0.0


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
