# Web Crawler API

The service exposes a minimal JSON API for bootstrap checks and browser probing.

## Endpoints

### `GET /health`

Returns service health and browser configuration status.

Example response:

```json
{
  "status": "ok",
  "service": "web-crawler",
  "browser_configured": true
}
```

### `POST /v1/pages/probe/browser`

Runs a Playwright-backed browser probe against a VOZ URL and returns diagnostics.
The browser probe uses a persistent Playwright profile by default so cookies and local storage can be reused across runs.

Request body:

```json
{
  "url": "https://voz.vn/t/cao-rau-doi-khi-nhung-thu-tot-nhat-lai-la-nhung-thu-re-tien.913440/",
  "page_url": "https://voz.vn/t/cao-rau-doi-khi-nhung-thu-tot-nhat-lai-la-nhung-thu-re-tien.913440/page-2/",
  "cookie": "voz_session=abc123; theme=dark",
  "timeout_seconds": 60
}
```

Response body:

```json
{
  "step": {
    "name": "browser_probe",
    "renderer": "playwright",
    "url": "https://voz.vn/t/cao-rau-doi-khi-nhung-thu-tot-nhat-lai-la-nhung-thu-re-tien.913440/",
    "final_url": "...",
    "status_code": 200,
    "content_type": "text/html",
    "kind": "ok",
    "bytes": 12345,
    "post_count": 3,
    "last_page": 112,
    "error": ""
  }
}
```

### `POST /v1/threads/crawl/browser`

Starts a background crawl of a VOZ thread and returns `202 Accepted` with the job immediately. Poll the job endpoint for progress and the final result.

Request body:

```json
{
  "url": "https://voz.vn/t/cao-rau-doi-khi-nhung-thu-tot-nhat-lai-la-nhung-thu-re-tien.913440/",
  "cookie": "voz_session=abc123; theme=dark",
  "timeout_seconds": 60,
  "max_pages": 20,
  "delay_seconds": 5,
  "fresh_profile": false,
  "resume_from_page": 71
}
```

Response body:

```json
{
  "job": {
    "job_id": "3f2b8c...",
    "status": "running",
    "url": "https://voz.vn/t/...913440/",
    "created_at": "2026-10-13T08:00:00Z",
    "finished_at": null,
    "progress": {"current_page": 0, "last_page": 1, "pages_crawled": 0, "posts_written": 0},
    "result": null,
    "error": ""
  }
}
```

Returns `409 Conflict` with the running `job_id` if a crawl for the same thread is already in progress.

### `GET /v1/threads/crawl/jobs/{job_id}`

Returns the job. `status` is `running`, `succeeded`, `failed` or `cancelled`. `progress` updates after every saved page; `result` holds the `ThreadCrawlResult` once the job succeeds, and `error` holds the failure message.

### `GET /v1/threads/crawl/jobs`

Lists known jobs, newest first.

### `DELETE /v1/threads/crawl/jobs/{job_id}`

Requests cancellation and returns `202 Accepted`. The status moves to `cancelled` once the current page fetch stops. Pages saved before cancellation stay on disk and can be resumed.

Job state is in memory only: it is lost on restart and the 100 most recent finished jobs are kept. The on-disk checkpoint is what makes a crawl resumable.

## Response Semantics

- `kind=ok` means the browser returned HTML that does not look like a challenge page.
- `kind=challenge` means the page loaded but its title or markup is a bot-challenge interstitial (for example Cloudflare's "Just a moment..."). Words like "captcha" inside posts do not count.
- `kind=blocked`, `kind=rate_limited` and `kind=server_error` come from HTTP `403`, `429` and `5xx`.
- `kind=network_error` means the browser could not launch or navigate successfully.
- `cookie` is treated as an in-memory Cookie header string and is never written to disk.
- `BROWSER_BLOCK_RESOURCES=true` aborts image, media and font requests to save bandwidth and load time. It is off by default because it makes the browser look slightly less like a real visitor.
- `BROWSER_PERSISTENT_PROFILE=true` uses the configured `BROWSER_PROFILE_DIR` on disk; set it to `false` to force a fresh ephemeral context.
