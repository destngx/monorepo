# Browser Probe Test Plan

This plan validates the Python browser probe endpoint for VOZ.

## Scope

- `GET /health`
- `POST /v1/pages/probe/browser`
- VOZ target thread: `https://voz.vn/t/cao-rau-doi-khi-nhung-thu-tot-nhat-lai-la-nhung-thu-re-tien.913440/`

## Prerequisites

- Python dependencies installed under `apps/web-crawler/.venv`
- Chrome or Chromium installed on the host
- `CHROME_BIN` points to the browser executable
- `cookie` may be supplied in the request body for an authorized browser session
- persistent profile is enabled by default and stores browser state under `./data/browser-profile`

## Test Cases

### 1. Bootstrap

Run the unit suite and confirm the app imports and boots.

Expected:

- `GET /health` returns HTTP `200`
- OpenAPI is available at `/openapi.json`

### 2. Browser Probe With Fake Browser Service

Use a test override so the endpoint is exercised without a real browser process.

Expected:

- HTTP `200`
- `step.renderer == "playwright"`
- `step.kind == "ok"`
- `step.post_count > 0`
- `step.last_page >= 1`

### 3. Browser Probe With Real Chrome

Start the server with `CHROME_BIN` set to the local Chrome executable and call the endpoint.

Expected:

- The API responds without crashing
- `kind` is either `ok` or `challenge`
- the returned payload includes `bytes`, `post_count`, and `last_page`
- `network_error` is only returned when the browser cannot launch or navigate
- optional cookies are kept in memory only and are not logged or persisted

### 4. Browser Probe With Persistent Profile

Run the endpoint twice with the same profile directory and confirm the second request can reuse browser state created by the first one.

Expected:

- the profile directory exists on disk after the first run
- the second run uses the same profile directory
- login/session state can survive between runs when the site accepts it

## Execution Order

1. Run `bunx nx run web-crawler:test`.
2. Run `bunx nx run web-crawler:test-e2e`.
3. Start the API with `bunx nx run web-crawler:serve`.
4. Call `/v1/pages/probe/browser` with `curl`.
5. Record `kind`, `bytes`, `post_count`, and `last_page`.

## Acceptance Criteria

- unit tests pass
- the API smoke test passes
- the real browser probe returns structured diagnostics

## Execution Results

### Unit and Smoke Tests

- `bunx nx run web-crawler:test` passed
- `bunx nx run web-crawler:test-e2e` passed

### Live Browser Probe

- Chrome binary used: `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
- HTTP status: `200`
- `step.kind`: `challenge`
- `step.renderer`: `playwright`
- `step.bytes`: `237284`
- `step.post_count`: `44`
- `step.last_page`: `112`
- `step.error`: empty

### Interpretation

- The new Python endpoint can launch a real browser and return rendered HTML diagnostics.
- VOZ still classifies the page as a challenge in this environment, so browser launch success is not the same as content access success.
