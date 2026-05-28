# Web Crawler Overview

This page describes the service at a high level.

## What Exists

- FastAPI server at `apps/web-crawler/src/main.py`
- Health endpoint for bootstrap verification
- Browser probe endpoint backed by Playwright
- Persistent browser profile support for session reuse
- HTML classification and site-specific parser helpers
- VOZ site module for crawling and checkpointed output
- Pytest coverage for bootstrap, parser, browser probe, and crawl wiring

## Current Scope

- The current MVP is a browser-probe service with site modules, not a full generic crawl orchestrator yet.
- The service renders a target page and reports whether the returned HTML looks normal or challenged.
- The browser probe keeps a persistent profile by default so cookies and local storage can survive between runs.
- Crawl state persistence is implemented for the VOZ site module and can be reused by later modules.

## Current Entry Points

- `GET /health`
- `POST /v1/pages/probe/browser`
