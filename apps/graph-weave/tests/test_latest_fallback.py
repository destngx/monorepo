"""
Test: GW-DATA-102 - Add latest skill fallback

Verification: Cache adapter uses 'latest' as default when version is not provided
"""

import pytest
from src.adapters.cache import MockRedisAdapter


@pytest.fixture
def cache():
    return MockRedisAdapter()


class TestLatestFallback:
    def test_get_versioned_without_version_resolves_to_latest(self, cache):
        """
        SCHEMA-01: Latest fallback rule
        Given a skill cached at version 'latest',
        When a lookup is requested without a version parameter,
        Then it should resolve to the 'latest' entry
        """
        tenant_id = "tenant_001"
        skill_id = "calculate"
        namespace = "skills:level1"

        cache.set_versioned(
            namespace=namespace,
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="latest",
            value={"name": "calculate", "version": "1.0.0"},
        )

        value = cache.get_versioned_with_fallback(
            namespace=namespace, tenant_id=tenant_id, skill_id=skill_id, version=None
        )

        assert value is not None
        assert value["name"] == "calculate"

    def test_latest_fallback_used_when_version_missing(self, cache):
        """
        FUNC-01: Latest fallback used
        Given a skill with a 'latest' entry in cache,
        When requesting without version,
        Then it should return the latest value
        """
        tenant_id = "tenant_001"
        skill_id = "search"
        namespace = "skills:level1"

        cache.set_versioned(
            namespace=namespace,
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="latest",
            value={"tool": "search", "status": "active"},
        )

        value = cache.get_versioned_with_fallback(
            namespace=namespace, tenant_id=tenant_id, skill_id=skill_id, version=None
        )

        assert value["tool"] == "search"
        assert value["status"] == "active"

    def test_specific_version_takes_precedence_over_latest(self, cache):
        """
        Additional: Specific version should be used if provided and exists
        """
        tenant_id = "tenant_001"
        skill_id = "calculate"
        namespace = "skills:level1"

        cache.set_versioned(
            namespace=namespace,
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="latest",
            value={"version": "1.0.0"},
        )

        cache.set_versioned(
            namespace=namespace,
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="2.0.0",
            value={"version": "2.0.0"},
        )

        value = cache.get_versioned_with_fallback(
            namespace=namespace, tenant_id=tenant_id, skill_id=skill_id, version="2.0.0"
        )

        assert value["version"] == "2.0.0"

    def test_latest_fallback_returns_none_if_not_found(self, cache):
        """
        Additional: Returns None if latest entry doesn't exist
        """
        tenant_id = "tenant_001"
        skill_id = "nonexistent"
        namespace = "skills:level1"

        value = cache.get_versioned_with_fallback(
            namespace=namespace, tenant_id=tenant_id, skill_id=skill_id, version=None
        )

        assert value is None

    def test_latest_fallback_with_multiple_versions(self, cache):
        """
        Additional: Only 'latest' is used when version is None
        """
        tenant_id = "tenant_001"
        skill_id = "calculate"
        namespace = "skills:level1"

        cache.set_versioned(
            namespace=namespace,
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="1.0.0",
            value={"version": "1.0.0"},
        )

        cache.set_versioned(
            namespace=namespace,
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="latest",
            value={"version": "latest"},
        )

        value = cache.get_versioned_with_fallback(
            namespace=namespace, tenant_id=tenant_id, skill_id=skill_id, version=None
        )

        assert value["version"] == "latest"
