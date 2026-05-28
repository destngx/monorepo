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

## Response Semantics

- `kind=ok` means the browser returned HTML that does not look like a challenge page.
- `kind=challenge` means the page loaded but the HTML looks blocked or challenged.
- `kind=network_error` means the browser could not launch or navigate successfully.
- `cookie` is treated as an in-memory Cookie header string and is never written to disk.
- `BROWSER_PERSISTENT_PROFILE=true` uses the configured `BROWSER_PROFILE_DIR` on disk; set it to `false` to force a fresh ephemeral context.
