from pydantic import BaseModel, Field


class ThreadCrawlRequest(BaseModel):
    url: str = Field(..., description="VOZ thread URL")
    cookie: str | None = Field(
        default=None,
        description="Optional Cookie header string from an authorized browser session",
    )
    timeout_seconds: int | None = Field(
        default=None, ge=1, le=120, description="Timeout per page fetch"
    )
    max_pages: int | None = Field(
        default=None,
        ge=1,
        le=500,
        description="Optional cap for test runs or partial crawls",
    )
    delay_seconds: float | None = Field(
        default=None,
        ge=0,
        le=30,
        description="Delay between page fetches",
    )
    fresh_profile: bool = Field(
        default=False,
        description="Use a fresh ephemeral browser profile for this crawl",
    )
    resume_from_page: int | None = Field(
        default=None,
        ge=1,
        le=500,
        description="Optional page to resume crawling from",
    )


class CrawledPost(BaseModel):
    thread_id: str
    page: int
    post_id: str
    post_number: int | None = None
    author: str = ""
    author_id: str = ""
    created_at: str = ""
    created_timestamp: int | None = None
    text: str = ""
    html: str = ""
    images: list[str] = Field(default_factory=list)
    url: str = ""


class CrawledPage(BaseModel):
    page: int
    url: str
    final_url: str = ""
    status_code: int = 0
    kind: str = ""
    bytes: int = 0
    post_count: int = 0
    last_page: int = 1
    error: str = ""


class ThreadCrawlResult(BaseModel):
    thread_id: str
    url: str
    output_path: str
    checkpoint_path: str
    pages_requested: int
    pages_crawled: int
    posts_written: int
    last_page: int
    resume_from_page: int
    stopped_reason: str = ""
    pages: list[CrawledPage] = Field(default_factory=list)


class CrawlCheckpoint(BaseModel):
    thread_id: str
    url: str
    last_successful_page: int
    last_page: int
    pages_requested: int
    posts_written: int
    output_path: str

    @classmethod
    def for_page(
        cls,
        *,
        thread_id: str,
        url: str,
        last_successful_page: int,
        last_page: int,
        pages_requested: int,
        posts_written: int,
        output_path: str,
    ) -> "CrawlCheckpoint":
        return cls(
            thread_id=thread_id,
            url=url,
            last_successful_page=last_successful_page,
            last_page=last_page,
            pages_requested=pages_requested,
            posts_written=posts_written,
            output_path=output_path,
        )


class ThreadCrawlResponse(BaseModel):
    result: ThreadCrawlResult
