# Web Crawler Architecture

## Goals

- Keep the service small and testable.
- Separate browser acquisition from HTML parsing.
- Preserve a clean HTTP API boundary.

## Package Layout

- `src/main.py`: FastAPI composition root
- `src/config.py`: environment-driven configuration
- `src/routers/`: HTTP handlers
- `src/services/`: browser client, probe service, detector, and other shared helpers
- `src/modules/sites/`: site-specific crawler and parser modules
- `src/modules/shared/deps.py`: singleton service wiring for the app

## Boundary Rules

- Routers do not know about Playwright.
- Parser code does not launch browsers.
- Browser code may persist profile data under the configured profile directory.
- Site-specific crawl logic stays in `modules/sites/<site>/`.
- Shared service wiring stays in `modules/shared/deps.py`.
