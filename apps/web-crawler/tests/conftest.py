import os
from pathlib import Path

os.environ.setdefault("DEBUG", "false")
os.environ.setdefault("HOST", "127.0.0.1")
os.environ.setdefault("PORT", "8002")
os.environ.setdefault("CHROME_BIN", "")
os.environ.setdefault("BROWSER_HEADLESS", "true")
os.environ.setdefault("BROWSER_TIMEOUT_SECONDS", "45")
os.environ.setdefault("BROWSER_NO_SANDBOX", "false")

_env_path = Path(__file__).resolve().parent.parent / ".env.local"
if _env_path.exists():
    with _env_path.open() as handle:
        for raw_line in handle:
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            os.environ.setdefault(key, value)
