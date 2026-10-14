from bs4 import BeautifulSoup

from src.services.detector import classify_page


def _soup(html: str) -> BeautifulSoup:
    return BeautifulSoup(html, "lxml")


def test_forum_page_mentioning_challenge_words_is_ok():
    html = """
    <html>
      <head>
        <title>Captcha challenge thread | VOZ</title>
        <script src="/cdn-cgi/challenge-platform/scripts/jsd/main.js"></script>
      </head>
      <body><article class="message-body">Cloudflare captcha challenge again</article></body>
    </html>
    """

    assert classify_page(200, _soup(html)) == "ok"


def test_cloudflare_interstitial_title_is_challenge():
    html = "<html><head><title>Just a moment...</title></head><body></body></html>"

    assert classify_page(200, _soup(html)) == "challenge"


def test_challenge_form_markup_is_challenge():
    html = """
    <html><body>
      <form id="challenge-form" action="/t/x.1/?__cf_chl_f_tk=abc"></form>
    </body></html>
    """

    assert classify_page(200, _soup(html)) == "challenge"


def test_status_codes_take_precedence():
    soup = _soup("<html><head><title>Just a moment...</title></head></html>")

    assert classify_page(403, soup) == "blocked"
    assert classify_page(429, soup) == "rate_limited"
    assert classify_page(503, soup) == "server_error"
