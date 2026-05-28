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

## Crawl VOZ

1. Send `POST /v1/threads/crawl/browser` with a VOZ thread URL.
2. Use `fresh_profile=true` if a profile becomes blocked.
3. Use `resume_from_page` to continue from a checkpoint.
4. Inspect the output JSON under `data/crawls/voz.vn/<thread_id>/`.

## Checks

- `bunx nx run web-crawler:test`
- `bunx nx run web-crawler:test-e2e`
- `bunx nx run web-crawler:lint`
