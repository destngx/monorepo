#!/usr/bin/env python3
import os
import sys
import json
import time
import httpx
from pathlib import Path
from typing import Optional


def update_env_var(key: str, value: str, env_path: Optional[str] = None) -> None:
    if env_path is None:
        env_path = Path.cwd() / ".env.local"
    else:
        env_path = Path(env_path)

    env_path.parent.mkdir(parents=True, exist_ok=True)

    env_content = ""
    if env_path.exists():
        env_content = env_path.read_text()

    import re

    regex = re.compile(f"^{key}=.*$", re.MULTILINE)
    if regex.search(env_content):
        env_content = regex.sub(f"{key}={value}", env_content)
    else:
        env_content += f"\n{key}={value}\n"

    env_path.write_text(env_content.strip() + "\n")


def print_section(title: str) -> None:
    print("\n" + "─" * 50)
    print(f"🤖 {title}")
    print("─" * 50 + "\n")


def github_device_flow(is_enterprise: bool = False) -> str:
    print_section("GitHub Copilot OAuth Setup")

    client_id = "01ab8ac9400c4e429b23"
    scope = "read:user user:email"

    code_response = httpx.post(
        "https://github.com/login/device/code",
        headers={"Accept": "application/json", "Content-Type": "application/json"},
        json={"client_id": client_id, "scope": scope},
    )

    if code_response.status_code != 200:
        raise RuntimeError(
            f"Failed to request device code: {code_response.status_code}"
        )

    code_data = code_response.json()

    print(f"1. Open your browser to: {code_data['verification_uri']}")
    print(
        f"2. Enter the following code: \033[32m\033[1m{code_data['user_code']}\033[0m\n"
    )
    print("Waiting for authorization (this may take up to 15 minutes)...\n")

    interval_seconds = code_data.get("interval", 5)
    token_data = None

    while not token_data:
        time.sleep(interval_seconds)

        poll_response = httpx.post(
            "https://github.com/login/oauth/access_token",
            headers={"Accept": "application/json", "Content-Type": "application/json"},
            json={
                "client_id": client_id,
                "device_code": code_data["device_code"],
                "grant_type": "urn:ietf:params:oauth:grant-type:device_code",
            },
        )

        poll_data = poll_response.json()

        if "error" in poll_data:
            error = poll_data["error"]
            if error == "authorization_pending":
                sys.stdout.write(".")
                sys.stdout.flush()
            elif error == "slow_down":
                time.sleep(poll_data.get("interval", 5))
            elif error == "expired_token":
                print("\n\n❌ Device code expired. Please run the script again.")
                sys.exit(1)
            elif error == "access_denied":
                print("\n\n❌ Authorization was denied.")
                sys.exit(1)
            else:
                print(f"\n\n❌ Unexpected error: {error}")
                sys.exit(1)
        elif "access_token" in poll_data:
            token_data = poll_data

    print("\n\n✅ Successfully authenticated with GitHub!")
    token = token_data["access_token"]
    update_env_var("GITHUB_TOKEN", token)
    print("🔑 Saved GITHUB_TOKEN to .env.local")

    print("\nFetching latest supported GitHub Copilot models...")
    try:
        models_response = httpx.get(
            "https://models.inference.ai.azure.com/models",
            headers={"Authorization": f"Bearer {token}"},
        )
        if models_response.status_code == 200:
            models_data = models_response.json()
            print("\nAvailable Models from your Copilot Account:")
            text_models = [
                m for m in models_data if m.get("task") in ["chat", "text-generation"]
            ]
            for model in text_models:
                print(f" - {model['name']} ({model.get('friendly_name', 'N/A')})")
            print(
                "\n💡 Note: You have access to GPT-4o, GPT-o1, and other models in your Copilot account!"
            )
        else:
            print("\nCould not fetch models directly, but authorization succeeded.")
    except Exception as e:
        print(f"\nCould not fetch models list: {e}")

    return token


def main():
    print("\n┌  Add credential\n│")

    providers = {
        "1": (
            "github-copilot",
            "GitHub Copilot (GPT-4o, o1, Grok - Free with Pro/Teams/Enterprise)",
        ),
        "2": ("openai", "OpenAI"),
        "3": ("google", "Google (Gemini)"),
        "4": ("anthropic", "Anthropic (Claude)"),
    }

    print("Select provider:")
    for key, (_, title) in providers.items():
        print(f"  {key}. {title}")

    choice = input("\nEnter choice (1-4): ").strip()

    if choice not in providers:
        print("Canceled.")
        sys.exit(0)

    provider_type, _ = providers[choice]

    if provider_type == "github-copilot":
        print("\nSelect GitHub deployment type:")
        print("  1. GitHub.com (Public)")
        print("  2. GitHub Enterprise")

        deployment_choice = input("\nEnter choice (1-2): ").strip()

        if deployment_choice == "2":
            token = input(
                "Enter your GitHub Enterprise Personal Access Token: "
            ).strip()
            if token:
                update_env_var("GITHUB_TOKEN", token)
                print("🔑 Saved GITHUB_TOKEN to .env.local")
        elif deployment_choice == "1":
            github_device_flow()
        else:
            sys.exit(0)
    else:
        key_name = ""
        url = ""

        if provider_type == "openai":
            key_name = "OPENAI_API_KEY"
            url = "https://platform.openai.com/api-keys"
        elif provider_type == "google":
            key_name = "GOOGLE_GENERATIVE_AI_API_KEY"
            url = "https://aistudio.google.com/app/apikey"
        elif provider_type == "anthropic":
            key_name = "ANTHROPIC_API_KEY"
            url = "https://console.anthropic.com/settings/keys"

        print(f"\nTo get your key, visit: {url}")
        api_key = input(f"Enter your {provider_type} API Key: ").strip()

        if api_key:
            update_env_var(key_name, api_key)
            print(f"🔑 Saved {key_name} to .env.local")
        else:
            print("Skipped saving key.")

    print("\n🚀 Setup complete! Please restart your dev server if it is running.")


if __name__ == "__main__":
    main()
