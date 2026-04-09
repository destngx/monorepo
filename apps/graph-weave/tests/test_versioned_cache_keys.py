"""
Test: GW-DATA-101 - Add versioned cache keys

Verification: Cache adapter uses versioned key format for skill lookups
"""

import pytest
from src.adapters.cache import MockRedisAdapter


@pytest.fixture
def cache():
    return MockRedisAdapter()


class TestVersionedCacheKeys:
    def test_versioned_skill_key_level1(self, cache):
        """
        SCHEMA-01: Versioned key shape
        Given a skill cache set operation with tenant, skill_id, and version,
        When the key is constructed,
        Then it should follow the format: skills:level1:{tenant_id}:{skill_id}:{version}
        """
        tenant_id = "tenant_001"
        skill_id = "calculate"
        version = "1.0.0"

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version,
            value={"name": "calculate", "description": "Math tool"},
        )

        value = cache.get_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version,
        )

        assert value is not None
        assert value["name"] == "calculate"

    def test_versioned_key_format_generated_correctly(self, cache):
        """
        FUNC-01: Key shape used
        Given a versioned key operation,
        When the key is constructed,
        Then the internal key format should be skills:level1:tenant_001:calculate:1.0.0
        """
        tenant_id = "tenant_001"
        skill_id = "calculate"
        version = "1.0.0"
        expected_key = "skills:level1:tenant_001:calculate:1.0.0"

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version,
            value={"tool": "math"},
        )

        assert cache.exists(expected_key)

    def test_versioned_keys_are_separate(self, cache):
        """
        Additional: Different versions should have separate cache entries
        """
        tenant_id = "tenant_001"
        skill_id = "calculate"
        version_1 = "1.0.0"
        version_2 = "2.0.0"

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version_1,
            value={"version": 1},
        )

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version_2,
            value={"version": 2},
        )

        value_1 = cache.get_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version_1,
        )

        value_2 = cache.get_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version=version_2,
        )

        assert value_1["version"] == 1
        assert value_2["version"] == 2

    def test_versioned_key_tenant_isolation(self, cache):
        """
        Additional: Different tenants should have separate entries
        """
        tenant_1 = "tenant_001"
        tenant_2 = "tenant_002"
        skill_id = "calculate"
        version = "1.0.0"

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_1,
            skill_id=skill_id,
            version=version,
            value={"tenant": tenant_1},
        )

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_2,
            skill_id=skill_id,
            version=version,
            value={"tenant": tenant_2},
        )

        value_1 = cache.get_versioned(
            namespace="skills:level1",
            tenant_id=tenant_1,
            skill_id=skill_id,
            version=version,
        )

        value_2 = cache.get_versioned(
            namespace="skills:level1",
            tenant_id=tenant_2,
            skill_id=skill_id,
            version=version,
        )

        assert value_1["tenant"] == tenant_1
        assert value_2["tenant"] == tenant_2
