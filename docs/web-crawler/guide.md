# Web Crawler Guide

This is the practical guide for running the `apps/web-crawler` service.

## Prerequisites

- Install dependencies for `apps/web-crawler`.
- Install Chrome or Chromium locally.
- Set `CHROME_BIN` if the default path does not match your machine.

## Start

1. Run `bunx nx run web-crawler:serve`.
2. Confirm `GET /health` returns `200`.
3. Use `POST /v1/pages/probe/browser` to probe a URL.

Example:

```bash
curl -X POST 'http://127.0.0.1:8002/v1/pages/probe/browser' \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://voz.vn/t/example-thread"
  }'
```

## Crawl VOZ

1. Send `POST /v1/threads/crawl/browser` with a VOZ thread URL. It returns a `job_id` right away.
2. Poll `GET /v1/threads/crawl/jobs/<job_id>` until `status` is no longer `running`. Use `DELETE` on the same path to stop a crawl.
3. Use `fresh_profile=true` if a profile becomes blocked.
4. Send the same request again later to pick up new replies: only the last saved page and newer pages are fetched. Use `resume_from_page` to start at a specific page, or `restart=true` to crawl everything again.
5. Inspect the output JSON under `data/crawls/voz.vn/<thread_id>/`.

Example:

```bash
job_id=$(curl -s -X POST 'http://127.0.0.1:8002/v1/threads/crawl/browser' \
  -H 'Content-Type: application/json' \
  -d '{"url": "https://voz.vn/t/example-thread.123/", "max_pages": 5}' | jq -r .job.job_id)
curl -s "http://127.0.0.1:8002/v1/threads/crawl/jobs/$job_id" | jq .job.status
```

## Checks

- `bunx nx run web-crawler:test`
- `bunx nx run web-crawler:test-e2e`
- `bunx nx run web-crawler:lint`
