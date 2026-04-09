"""
Test: GW-RUNTIME-101 - Preserve run_id on rerun

Verification: run_id stays stable when rerunning, while thread_id is always new
"""

import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestStableRunId:
    def test_rerun_preserves_run_id(self, client):
        """
        FUNC-01: Stable identity
        Given a first execution request,
        When we rerun with the same run_id,
        Then the returned run_id should match the original
        """
        response1 = client.post(
            "/execute",
            json={
                "tenant_id": "tenant_001",
                "workflow_id": "workflow_001",
                "input": {"data": "test"},
            },
        )
        assert response1.status_code == 200
        run_id_first = response1.json()["run_id"]
        thread_id_first = response1.json()["thread_id"]

        response2 = client.post(
            "/execute",
            json={
                "tenant_id": "tenant_001",
                "workflow_id": "workflow_001",
                "input": {"data": "test"},
                "run_id": run_id_first,
            },
        )
        assert response2.status_code == 200
        run_id_second = response2.json()["run_id"]
        thread_id_second = response2.json()["thread_id"]

        assert run_id_second == run_id_first
        assert thread_id_second != thread_id_first

    def test_new_run_generates_new_run_id(self, client):
        """
        Verify that normal executions still generate new run_ids
        """
        response1 = client.post(
            "/execute",
            json={
                "tenant_id": "tenant_001",
                "workflow_id": "workflow_001",
                "input": {"data": "test"},
            },
        )
        run_id_first = response1.json()["run_id"]

        response2 = client.post(
            "/execute",
            json={
                "tenant_id": "tenant_001",
                "workflow_id": "workflow_001",
                "input": {"data": "test"},
            },
        )
        run_id_second = response2.json()["run_id"]

        assert run_id_first != run_id_second

    def test_rerun_response_structure(self, client):
        """
        Verify response structure is correct on rerun
        """
        response1 = client.post(
            "/execute",
            json={
                "tenant_id": "tenant_001",
                "workflow_id": "workflow_001",
                "input": {"data": "test"},
            },
        )
        run_id = response1.json()["run_id"]

        response2 = client.post(
            "/execute",
            json={
                "tenant_id": "tenant_001",
                "workflow_id": "workflow_001",
                "input": {"data": "test"},
                "run_id": run_id,
            },
        )
        assert response2.status_code == 200
        data = response2.json()

        assert "run_id" in data
        assert "thread_id" in data
        assert "status" in data
        assert "workflow_id" in data
        assert "tenant_id" in data
        assert data["run_id"] == run_id
        assert data["status"] == "pending"
