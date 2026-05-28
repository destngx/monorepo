CHALLENGE_MARKERS = (
    "captcha",
    "cloudflare",
    "attention required",
    "challenge-platform",
    "verify you are human",
    "cf-challenge",
    "challenge",
)


def classify_html(status_code: int, body: str) -> str:
    lowered = body.lower()
    if status_code == 429:
        return "rate_limited"
    if status_code == 403:
        return "blocked"
    if status_code >= 500:
        return "server_error"
    if any(marker in lowered for marker in CHALLENGE_MARKERS):
        return "challenge"
    if 200 <= status_code < 300:
        return "ok"
    return "ok"
