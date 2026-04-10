import json
from importlib import import_module

import pytest


GitHubCopilotProvider = import_module("src.adapters.ai_provider").GitHubCopilotProvider


def test_github_copilot_provider_uses_github_token_and_returns_response(monkeypatch):
    provider = GitHubCopilotProvider(token="gho_test_token")

    captured = {}

    class DummyResponse:
        def raise_for_status(self):
            return None

        def json(self):
            return {
                "choices": [
                    {"message": {"content": json.dumps({"status": "completed"})}}
                ],
                "usage": {"total_tokens": 42},
                "model": "gpt-4.1",
            }

    def fake_post(url, json=None, headers=None, timeout=None):
        captured["url"] = url
        captured["json"] = json
        captured["headers"] = headers
        captured["timeout"] = timeout
        return DummyResponse()

    monkeypatch.setattr("httpx.post", fake_post)

    result = provider.call(
        system_prompt="You are a helpful assistant",
        user_prompt="Return JSON only",
        model="gpt-4.1",
        temperature=0.2,
        max_tokens=256,
    )

    assert captured["url"].startswith("https://api.githubcopilot.com")
    assert captured["headers"]["Authorization"] == "Bearer gho_test_token"
    assert captured["json"]["model"] == "gpt-4.1"
    assert result["model"] == "gpt-4.1"
    assert result["tokens_used"] == 42
    assert result["content"] == json.dumps({"status": "completed"})
