import re
from urllib.parse import urljoin, urlsplit, urlunsplit

from bs4 import BeautifulSoup

from src.models.crawl import CrawledPost

PAGE_RE = re.compile(r"page-(\d+)", re.IGNORECASE)
OF_RE = re.compile(r"\bof\s+(\d+)\b", re.IGNORECASE)
# The id ends the thread segment: "/t/slug.123/" or a bare "/t/123/".
THREAD_ID_RE = re.compile(r"[./](\d+)/$")
POST_NUMBER_RE = re.compile(r"#(\d+)")
USER_ID_RE = re.compile(r"\.(\d+)/?$")
ITEM_POST_ID_RE = re.compile(r"/p/(\d+)/?")

MAX_PAGE = 10_000
PAGE_JUMP_SELECTOR = "input.js-pageJumpPage[max]"
PAGE_NAV_LINK_SELECTOR = (
    ".pageNav a[href], .pageNav-main a[href], .pageNavSimple a[href]"
)
PAGE_NAV_TEXT_SELECTOR = ".pageNav-main, .pageNavSimple-el--current"
POST_SELECTOR = (
    "article.message--post.js-post, "
    "article.js-post[id^='js-post-'], "
    "article[data-content^='post-'], "
    "article[itemtype='https://schema.org/Comment'], "
    "article[id^='post-']"
)


class VozParser:
    def parse_html(self, html: str) -> BeautifulSoup:
        return BeautifulSoup(html, "lxml")

    def canonical_thread_url(self, url: str) -> str:
        parsed = urlsplit(url.strip())
        path = re.sub(r"/page-\d+/?$", "/", parsed.path)
        path = re.sub(r"/+$", "/", path)
        if not path.endswith("/"):
            path = f"{path}/"
        return urlunsplit((parsed.scheme, parsed.netloc, path, "", ""))

    def parse_last_page(self, soup: BeautifulSoup) -> int:
        # Only the XenForo page navigation is trusted: post bodies routinely
        # contain text like "page-300" or "5 of 300".
        candidates: set[int] = set()

        for node in soup.select(PAGE_JUMP_SELECTOR):
            self._add_page_candidate(candidates, str(node.get("max")))

        for link in soup.select(PAGE_NAV_LINK_SELECTOR):
            for match in PAGE_RE.finditer(str(link.get("href"))):
                self._add_page_candidate(candidates, match.group(1))

        for node in soup.select(PAGE_NAV_TEXT_SELECTOR):
            for match in OF_RE.finditer(node.get_text(" ", strip=True)):
                self._add_page_candidate(candidates, match.group(1))

        return max(candidates, default=1)

    def count_posts(self, soup: BeautifulSoup) -> int:
        return len(self._post_nodes(soup))

    def extract_posts(
        self,
        soup: BeautifulSoup,
        page: int,
        page_url: str,
        thread_id: str | None = None,
    ) -> list[CrawledPost]:
        resolved_thread_id = thread_id or self.thread_id_from_url(page_url)
        posts: list[CrawledPost] = []

        for node in self._post_nodes(soup):
            post_id = self._post_id(node)
            if not post_id:
                continue

            author_node = node.select_one(".message-name .username, a.username")
            time_node = node.select_one("time.u-dt")
            body_node = node.select_one(".message-body")
            post_link_node = node.select_one(
                ".message-attribution-opposite a[href*='/post-']"
            )

            post_url = page_url
            if post_link_node and post_link_node.get("href"):
                post_url = urljoin(page_url, str(post_link_node.get("href")))

            posts.append(
                CrawledPost(
                    thread_id=resolved_thread_id,
                    page=page,
                    post_id=post_id,
                    post_number=self._post_number(node),
                    author=(
                        author_node.get_text(" ", strip=True)
                        if author_node
                        else str(node.get("data-author") or "")
                    ),
                    author_id=self._author_id(author_node),
                    created_at=(
                        str(time_node.get("datetime") or "") if time_node else ""
                    ),
                    created_timestamp=self._timestamp(time_node),
                    text=body_node.get_text("\n", strip=True) if body_node else "",
                    images=self._images(node, page_url),
                    url=post_url,
                )
            )

        return posts

    def thread_id_from_url(self, url: str) -> str:
        match = THREAD_ID_RE.search(self.canonical_thread_url(url))
        if match:
            return match.group(1)
        return "thread"

    def page_url(self, thread_url: str, page: int) -> str:
        base = self.canonical_thread_url(thread_url)
        if page <= 1:
            return base
        return f"{base.rstrip('/')}/page-{page}"

    def _post_nodes(self, soup: BeautifulSoup) -> list:
        seen: set[str] = set()
        nodes = []

        for node in soup.select(POST_SELECTOR):
            node_id = (
                node.get("data-content")
                or node.get("itemid")
                or node.get("id")
                or f"anon-{id(node)}"
            )
            if node_id in seen:
                continue
            seen.add(node_id)
            nodes.append(node)

        return nodes

    def _post_id(self, node) -> str:
        content = str(node.get("data-content") or "")
        if content.startswith("post-"):
            return content.removeprefix("post-")

        node_id = str(node.get("id") or "")
        if node_id.startswith("js-post-"):
            return node_id.removeprefix("js-post-")
        if node_id.startswith("post-"):
            return node_id.removeprefix("post-")

        item_id = str(node.get("itemid") or "")
        match = ITEM_POST_ID_RE.search(item_id)
        if match:
            return match.group(1)

        return ""

    def _post_number(self, node) -> int | None:
        for link in node.select(".message-attribution-opposite a"):
            text = link.get_text(" ", strip=True)
            match = POST_NUMBER_RE.search(text)
            if match:
                return int(match.group(1))
        return None

    def _author_id(self, author_node) -> str:
        if not author_node:
            return ""

        data_user_id = author_node.get("data-user-id")
        if data_user_id:
            return str(data_user_id)

        href = str(author_node.get("href") or "")
        match = USER_ID_RE.search(href)
        if match:
            return match.group(1)

        return ""

    def _timestamp(self, time_node) -> int | None:
        if not time_node:
            return None

        raw = time_node.get("data-timestamp")
        if not raw:
            return None

        try:
            return int(str(raw))
        except ValueError:
            return None

    def _add_page_candidate(self, candidates: set[int], raw: str) -> None:
        if raw.isdigit() and 1 <= int(raw) <= MAX_PAGE:
            candidates.add(int(raw))

    def _images(self, node, page_url: str) -> list[str]:
        images: list[str] = []
        seen: set[str] = set()

        for image in node.select("img"):
            raw = image.get("data-src") or image.get("src")
            if not raw:
                continue
            src = urljoin(page_url, str(raw))
            if src in seen or src.startswith("data:"):
                continue
            seen.add(src)
            images.append(src)

        return images
