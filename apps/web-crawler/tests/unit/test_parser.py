from src.modules.sites.voz.parser import VozParser

HTML = """
<html>
  <body>
    <a rel="next" href="/t/example.913440/page-2/">Next</a>
    <div class="pageNav-main">of 112</div>
    <article id="post-1" class="message">one</article>
    <article id="post-2" class="message">two</article>
  </body>
</html>
"""


def test_parse_last_page():
    parser = VozParser()
    assert parser.parse_last_page(HTML) == 112


def test_count_posts():
    parser = VozParser()
    assert parser.count_posts(HTML) == 2


def test_count_posts_ignores_xenforo_ad_placeholders():
    html = """
    <html>
      <body>
        <article class="message message--post js-post" data-content="post-1" id="js-post-1"></article>
        <div class="message message--post"><pubtag data-ad-slot="ad"></pubtag></div>
        <article class="message message--post js-post" data-content="post-2" id="js-post-2"></article>
      </body>
    </html>
    """

    parser = VozParser()

    assert parser.count_posts(html) == 2


def test_parse_last_page_from_xenforo_page_jump_max():
    html = """
    <html>
      <body>
        <input class="js-pageJumpPage" value="4" min="1" max="112" />
        <a href="/t/example.913440/page-2" class="pageNav-jump--next">Next</a>
      </body>
    </html>
    """

    parser = VozParser()

    assert parser.parse_last_page(html) == 112


def test_extract_posts_from_xenforo_thread_html():
    html = """
    <html>
      <body>
        <article class="message message--post js-post" data-author="alice" data-content="post-3001" id="js-post-3001">
          <h4 class="message-name"><a href="/u/alice.42/" class="username" data-user-id="42">alice</a></h4>
          <time class="u-dt" datetime="2024-01-26T11:38:20+0700" data-timestamp="1706243900"></time>
          <ul class="message-attribution-opposite"><li><a href="/t/example.913440/post-3001">#1</a></li></ul>
          <article class="message-body"><div class="bbWrapper">hello<br />world<img data-src="/attachments/a.png" /></div></article>
        </article>
      </body>
    </html>
    """

    parser = VozParser()
    posts = parser.extract_posts(
        html,
        page=1,
        page_url="https://voz.vn/t/example.913440/",
    )

    assert len(posts) == 1
    assert posts[0].thread_id == "913440"
    assert posts[0].post_id == "3001"
    assert posts[0].post_number == 1
    assert posts[0].author == "alice"
    assert posts[0].author_id == "42"
    assert posts[0].created_timestamp == 1706243900
    assert "hello" in posts[0].text
    assert posts[0].images == ["https://voz.vn/attachments/a.png"]


def test_builds_xenforo_page_urls():
    parser = VozParser()
    url = "https://voz.vn/t/example.913440/"

    assert parser.page_url(url, 1) == url
    assert parser.page_url(url, 2) == "https://voz.vn/t/example.913440/page-2"


def test_canonical_thread_url_strips_page_and_fragment():
    parser = VozParser()
    url = "https://voz.vn/t/ca-phe-tai-gia-2025.1048163/page-239#post-41417669"

    assert parser.canonical_thread_url(url) == "https://voz.vn/t/ca-phe-tai-gia-2025.1048163/"
    assert parser.thread_id_from_url(url) == "1048163"
    assert parser.page_url(url, 2) == "https://voz.vn/t/ca-phe-tai-gia-2025.1048163/page-2"
