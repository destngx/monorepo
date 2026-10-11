---
title: web-crawler
summary: A FastAPI service that drives a local Chrome through Playwright to probe pages for challenges and blocks, and crawls VOZ forum threads into resumable JSON checkpoints.
year: 2026
status: experiment
stack: [Python, FastAPI, Playwright, BeautifulSoup, lxml, Pydantic]
repo: https://github.com/destngx/monorepo/tree/main/apps/web-crawler
featured: false
order: 7
sfx: ザワ
---

## The problem

Some sites block plain HTTP clients or serve challenge pages, and long forum threads take many page loads. I wanted a local service that loads pages in a real browser, reports when a request has been blocked, and can resume a long crawl from where it stopped. The first target was VOZ (voz.vn), a XenForo forum.

## What it does

- `GET /health` for bootstrap checks.
- `POST /v1/pages/probe/browser` loads a URL and returns diagnostics: status code, final URL, byte count, and a `kind` such as `ok` or `challenge`.
- `POST /v1/threads/crawl/browser` crawls a VOZ thread page by page. It accepts `max_pages`, `resume_from_page`, and `fresh_profile`.
- Writes `data/crawls/<domain>/<thread_id>/<thread_id>.json` plus a `.checkpoint.json`. Each reply keeps only `author`, `created_at`, and `text`.
- Caller-supplied cookies are passed to the browser in memory. The API docs state they are never written to disk.

## How it's built

I used FastAPI for the HTTP layer and Pydantic models for requests and responses. `src/main.py` is the composition root: its lifespan hook validates config and initializes the services.

Playwright drives the locally installed Chrome, located through `CHROME_BIN`. It uses `launch_persistent_context` by default so cookies and local storage survive between runs. BeautifulSoup with the lxml parser extracts posts from the XenForo markup.

The architecture doc sets the boundaries I followed: routers do not know about Playwright, parsers do not launch browsers, and site logic lives under `src/modules/sites/<site>/`. Tests are pytest unit tests plus a small e2e smoke test.

## Interesting bits

- Page completion is judged from posts actually found in the HTML, not from the HTTP status alone.
- The detector maps 429 to `rate_limited`, 403 to `blocked`, 5xx to `server_error`, and markers such as "cloudflare" or "captcha" to `challenge`.
- On the first `403 blocked`, the crawler writes a checkpoint that points at the last good page and retries once with a fresh browser profile. A second block stops the crawl and keeps the checkpoint.
- Page-to-page delays are random, drawn between 5 and 30 seconds, rather than fixed.
- A browser profile is kept on disk by default. Setting `BROWSER_PERSISTENT_PROFILE=false` gives an ephemeral context instead.

## What's next

The overview describes the service as an MVP: a browser-probe endpoint plus site modules, not a generic crawl orchestrator yet. The VOZ strategy doc notes that thread parsing is tied to the current XenForo layout and does not yet cover other forum engines or content types.
