import os

import pytest

from src.adapters.ai_provider import GitHubCopilotProvider


def test_live_github_provider_smoke():
    token = os.getenv("GITHUB_TOKEN")
    if not token or token.startswith("gho_test"):
        pytest.skip("Live GitHub token is not available")

    provider = GitHubCopilotProvider(token=token)
    result = provider.call(
        "You are a helper", "Say hello", model="gpt-4.1", max_tokens=32
    )

    assert isinstance(result, dict)
    assert "content" in result
    assert result["model"] == "gpt-4.1"
