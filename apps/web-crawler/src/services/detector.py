from bs4 import BeautifulSoup

# Matched against the page title and dedicated challenge markup only. Keywords
# like "cloudflare" or "challenge" also appear on normal pages (Cloudflare
# injects /cdn-cgi/challenge-platform/ scripts) and inside user posts.
CHALLENGE_TITLE_MARKERS = (
    "just a moment",
    "attention required",
    "verify you are human",
    "checking your browser",
    "security check",
)
CHALLENGE_SELECTOR = (
    "#challenge-form, "
    "#challenge-stage, "
    "#challenge-running, "
    "#cf-challenge-running, "
    "form[action*='__cf_chl']"
)


def classify_page(status_code: int, soup: BeautifulSoup) -> str:
    if status_code == 429:
        return "rate_limited"
    if status_code == 403:
        return "blocked"
    if status_code >= 500:
        return "server_error"
    if _looks_like_challenge(soup):
        return "challenge"
    return "ok"


def _looks_like_challenge(soup: BeautifulSoup) -> bool:
    title = soup.title.get_text(" ", strip=True).lower() if soup.title else ""
    if any(marker in title for marker in CHALLENGE_TITLE_MARKERS):
        return True
    return soup.select_one(CHALLENGE_SELECTOR) is not None
