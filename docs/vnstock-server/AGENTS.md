# Vnstock Server: Engineering & Agent Standards

This document defines the core standards for the `vnstock-server` wrapper (Python).

## 🐍 Architecture: FastAPI & Market Data

The server provides a high-performance REST API wrapper around the `vnstock` library for the Vietnam stock market.

### 📁 Stack & Tooling

- **Language**: Python 3.x
- **Framework**: FastAPI
- **Dependency Management**: `uv` (Fast and reliable Python package management).
- **Caching**: Upstash Redis (Essential for market data latency and rate-limit mitigation).

## ⚡ Caching Policy

All endpoints must respect the following default caching TTLs to ensure data freshness while reducing upstream load:

- **Quotes**: 1 minute
- **Listings**: 1 hour
- **Historical**: 1 hour
- **Financial Statements**: 24 hours

## 🧪 Testing & Workflow

- **Runner**: `uv run pytest`
- **Environment**: Always use `.env.local` for local development. Never commit secrets like `VNSTOCK_API_KEY`.
- **Nx Integration**: Prefer running tasks via Nx: `bunx nx run vnstock-server:test`.

## 🛠️ Implementation Rules

- Always use typed Pydantic models for request and response validation.
- Ensure error responses are descriptive and handle provider-specific exceptions gracefully.
