from pydantic import BaseModel, Field


class BrowserProbeRequest(BaseModel):
    url: str = Field(..., description="Forum thread or page URL")
    page_url: str | None = Field(default=None, description="Optional exact page URL")
    cookie: str | None = Field(
        default=None,
        description="Optional Cookie header string from an authorized browser session",
    )
    timeout_seconds: int | None = Field(
        default=None, ge=1, le=120, description="Timeout for the browser probe"
    )


class BrowserProbeStep(BaseModel):
    name: str
    renderer: str
    url: str
    final_url: str = ""
    status_code: int = 0
    content_type: str = ""
    kind: str = ""
    bytes: int = 0
    post_count: int = 0
    last_page: int = 1
    error: str = ""


class BrowserProbeResponse(BaseModel):
    step: BrowserProbeStep
