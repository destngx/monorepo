from importlib import import_module


ai_provider = import_module("src.adapters.ai_provider")
GitHubCopilotProvider = ai_provider.GitHubCopilotProvider
MockAIProvider = ai_provider.MockAIProvider
create_ai_provider = ai_provider.create_ai_provider


def test_create_ai_provider_defaults_to_mock(monkeypatch):
    monkeypatch.delenv("GITHUB_TOKEN", raising=False)

    import pytest

    with pytest.raises(RuntimeError, match="GITHUB_TOKEN is required"):
        create_ai_provider()


def test_create_ai_provider_uses_github_token(monkeypatch):
    monkeypatch.setenv("GITHUB_TOKEN", "gho_test_token")

    provider = create_ai_provider()

    assert isinstance(provider, GitHubCopilotProvider)


def test_github_provider_defaults_to_gpt_4_1(monkeypatch):
    monkeypatch.setenv("GITHUB_TOKEN", "gho_test_token")

    provider = GitHubCopilotProvider(token="gho_test_token")

    captured = {}

    class DummyResponse:
        def raise_for_status(self):
            return None

        def json(self):
            return {
                "choices": [{"message": {"content": "ok"}}],
                "usage": {"total_tokens": 1},
                "model": "gpt-4.1",
            }

    def fake_post(url, json=None, headers=None, timeout=None):
        captured["json"] = json
        return DummyResponse()

    import httpx

    monkeypatch.setattr(httpx, "post", fake_post)

    provider.call("system", "user")

    assert captured["json"]["model"] == "gpt-4.1"
