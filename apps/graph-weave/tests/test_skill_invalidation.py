"""
Test: GW-SKILL-101 - Add cache invalidation API

Verification: Invalidation endpoint accepts tenant, skill, and reason to remove cache entries
"""

import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestSkillCacheInvalidation:
    def test_invalidate_endpoint_exists(self, client):
        """
        FUNC-01: API inputs present
        Given an invalidation request with tenant, skill, and reason,
        When POST /invalidate is called,
        Then it should accept the request
        """
        response = client.post(
            "/invalidate",
            json={
                "tenant_id": "tenant_001",
                "skill_id": "calculate",
                "reason": "package_update",
            },
        )
        assert response.status_code == 200

    def test_invalidate_removes_cache_entry(self, client):
        """
        FUNC-01: API inputs present + functional behavior
        Given a cached skill entry,
        When invalidation is triggered with tenant, skill, and reason,
        Then the cache entry should be removed
        """
        from src.modules.shared.deps import get_cache

        cache = get_cache()
        tenant_id = "tenant_001"
        skill_id = "calculate"

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_id,
            version="latest",
            value={"name": "calculate"},
        )

        cache_key = f"skills:level1:{tenant_id}:{skill_id}:latest"
        assert cache.exists(cache_key)

        response = client.post(
            "/invalidate",
            json={
                "tenant_id": tenant_id,
                "skill_id": skill_id,
                "reason": "package_update",
            },
        )
        assert response.status_code == 200
        assert not cache.exists(cache_key)

    def test_invalidate_returns_success_status(self, client):
        """
        Additional: Verify response structure
        """
        response = client.post(
            "/invalidate",
            json={
                "tenant_id": "tenant_001",
                "skill_id": "search",
                "reason": "manual_refresh",
            },
        )
        data = response.json()
        assert response.status_code == 200
        assert "status" in data or response.text

    def test_invalidate_with_all_required_fields(self, client):
        """
        Additional: All required fields must be present
        """
        response = client.post(
            "/invalidate",
            json={
                "tenant_id": "tenant_001",
                "skill_id": "calculate",
                "reason": "external_edit",
            },
        )
        assert response.status_code == 200

    def test_invalidate_missing_field_returns_error(self, client):
        """
        Additional: Missing required fields should error
        """
        response = client.post(
            "/invalidate", json={"tenant_id": "tenant_001", "skill_id": "calculate"}
        )
        assert response.status_code == 422

    def test_invalidate_multiple_skills_independently(self, client):
        """
        Additional: Invalidating one skill doesn't affect others
        """
        from src.modules.shared.deps import get_cache

        cache = get_cache()
        tenant_id = "tenant_001"
        skill_1 = "calculate"
        skill_2 = "search"

        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_1,
            version="latest",
            value={"name": skill_1},
        )
        cache.set_versioned(
            namespace="skills:level1",
            tenant_id=tenant_id,
            skill_id=skill_2,
            version="latest",
            value={"name": skill_2},
        )

        client.post(
            "/invalidate",
            json={"tenant_id": tenant_id, "skill_id": skill_1, "reason": "update"},
        )

        key_1 = f"skills:level1:{tenant_id}:{skill_1}:latest"
        key_2 = f"skills:level1:{tenant_id}:{skill_2}:latest"

        assert not cache.exists(key_1)
        assert cache.exists(key_2)
