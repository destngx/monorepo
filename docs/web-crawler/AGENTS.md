# Web Crawler

This document is the canonical working guide for the Python `apps/web-crawler` service.

## Rules

- Use FastAPI for the HTTP surface.
- Keep browser acquisition behind a service interface so tests can override it.
- Use Playwright for browser control.
- Keep shared HTML classification and browser plumbing in plain Python helpers.
- Keep site-specific parsing and crawl logic under `src/modules/sites/<site>/`.
- Avoid writing crawl state outside the site module until a shared crawl pipeline is needed.
- `apps/web-crawler/project.json` remains the Nx integration point.
