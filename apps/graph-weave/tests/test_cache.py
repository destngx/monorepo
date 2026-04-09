import pytest
from src.adapters.cache import MockRedisAdapter


@pytest.fixture
def cache_adapter():
    return MockRedisAdapter()


class TestMockRedisAdapter:
    def test_set_and_get_basic(self, cache_adapter):
        cache_adapter.set("test:key", "test_value")
        value = cache_adapter.get("test:key")
        assert value == "test_value"

    def test_get_nonexistent_key(self, cache_adapter):
        value = cache_adapter.get("nonexistent:key")
        assert value is None

    def test_delete_key(self, cache_adapter):
        cache_adapter.set("test:key", "test_value")
        cache_adapter.delete("test:key")
        value = cache_adapter.get("test:key")
        assert value is None

    def test_set_overwrites_previous(self, cache_adapter):
        cache_adapter.set("test:key", "value1")
        cache_adapter.set("test:key", "value2")
        value = cache_adapter.get("test:key")
        assert value == "value2"

    def test_namespace_isolation(self, cache_adapter):
        cache_adapter.set("namespace1:key", "value1")
        cache_adapter.set("namespace2:key", "value2")

        assert cache_adapter.get("namespace1:key") == "value1"
        assert cache_adapter.get("namespace2:key") == "value2"

    def test_exists_method(self, cache_adapter):
        cache_adapter.set("test:key", "test_value")
        assert cache_adapter.exists("test:key") is True
        assert cache_adapter.exists("nonexistent:key") is False

    def test_clear_all(self, cache_adapter):
        cache_adapter.set("key1", "value1")
        cache_adapter.set("key2", "value2")
        cache_adapter.clear()

        assert cache_adapter.get("key1") is None
        assert cache_adapter.get("key2") is None
