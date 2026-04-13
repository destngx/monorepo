# Go AI Gateway

A stateless, OpenAI-compatible local AI proxy written in Go. The gateway acts as a unified abstraction layer for multiple AI providers (GitHub Models, OpenAI, Anthropic, Ollama), allowing client applications to switch between LLMs using a single header without handling individual API keys or specific provider formats.

## The "Why": Philosophy

### 1. Provider Agnostic Integration

Clients shouldn't need to learn the nuances of the Anthropic "Messages" API vs. OpenAI's format. The gateway provides a unified, **streaming-first OpenAI-compatible interface** for all backends.

### 2. Secure Token Delegation

Application secrets stay on the server. Client applications (especially local tools or frontend apps) just talk to the gateway, which manages the upstream authentication transparently.

### 3. Statelesness & Portability

The gateway is entirely stateless. No database, no session management. It is designed to be a high-speed, lightweight sidecar or middle-man that can be deployed anywhere with zero external dependencies.

### 4. Robust Readiness

Using a **2-phase check** (Token presence + Network Ping), the gateway ensures you only attempt to call providers that are actually healthy and configured, preventing unexpected upstream failures from leaking into application logic.

### 5. Primary Compatibility Targets

The Gateway is specifically optimized and tested against strict protocol clients like:

- **Claude Code CLI**: Full support for real-time tool calling, prompt caching headers, and complex stream parsing.
- **Wealth Dashboard**: Optimized for multi-agent tool loops and financial data visualization.

---

## The "How": Architecture

The application is built following **Clean Architecture** and **Hexagonal** principles. It has recently been refactored from a monolithic `proxy` package into a modular `internal/` structure to ensure strict separation of concerns.

### 1. Structure

- [**`internal/domain`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/domain) — Core data structures and interface definitions. This is the "OpenAI-compatible" contract that the rest of the app speaks.
- [**`internal/providers`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/providers) — Adapter layer. Contains specific implementations for Anthropic, OpenAI, GitHub Models, and Ollama. Each implements the common `Provider` interface.
- [**`internal/service`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/service) — The business logic layer. Manages the `Registry` (provider lifecycle) and `Mapper` (intelligent model routing).
- [**`internal/transport`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/transport) — The delivery layer. Contains HTTP handlers, SSE logic, and middleware (CORS, Logger, Recovery).

### 2. Engineering Conventions

To maintain a high-quality, readable codebase, we follow strict engineering standards:

- **Zero Hardcoded Strings**: All literals (headers, log patterns, error messages) are extracted into contextual constants.
- **Meaningful Namespacing**: Constants are named to reflect their domain and scoped to prevent collisions.

See [**ENGINEERING.md**](./ENGINEERING.md) for the full breakdown of our coding standards.

### 3. Intelligent Format Conversion

For providers that don't natively support the OpenAI format (like Anthropic), the gateway performs real-time message transformation, SSE stream mapping, and character-based token estimation at the provider-adapter level.

### 3. Integrated into the Monorepo

The app is fully integrated with **Nx** and follows the monorepo's Go standards:

- **Hot-Reloading**: Integrated with `air` for a fast development loop.
- **Dependency Management**: Uses `godotenv` for local configuration.
- **Standard Testing**: High-coverage unit tests using `httptest`.

### 4. Stack

- **Language**: Go 1.26+ (Standard Library only for the core proxy).
- **Tooling**: Nx (Task Runner), Air (Live Reloading).
- **Packages**: `joho/godotenv` (Configuration).

---

## Development Setup

### 1. Configuration

Create a `.env` file in `apps/ai-gateway/`:

```bash
GITHUB_TOKEN=gho_...
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
OLLAMA_BASE_URL=http://localhost:11434
```

### 2. Available Commands

From the monorepo root:

- **Serve (with hot-reload)**: `bun nx serve ai-gateway`
- **Test**: `bun nx test ai-gateway`
- **Format**: `bun nx format ai-gateway`
- **Tidy Dependencies**: `bun nx tidy ai-gateway`

---

## Documentation

- [**API Integration Guide**](./API_GUIDE.md) — Technical reference for developers.
