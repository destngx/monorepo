# Engineering Standards - AI Gateway

This document outlines the core engineering principles and coding conventions for the AI Gateway. Adhering to these standards ensures the codebase remains maintainable, scalable, and readable.

## 1. Zero Hardcoded Strings Policy

We maintain a strict **no hardcoded strings** policy in all application logic. Literals (headers, log formats, error messages, content types, etc.) must be extracted into meaningful constants.

### Why?

- **Maintainability**: Changing a header name or log pattern once is easier and safer than doing a find-and-replace across multiple files.
- **Readability**: Constants provide semantic meaning to what would otherwise be opaque string literals.
- **Type Safety**: Using constants reduces the risk of runtime errors caused by typos or inconsistent casing.

### Implementation Guidelines

- Define constants at the **top of the file** within a `const (...)` block.
- For strings shared across multiple files within the same package, use a package-level **`constants.go`** file.
- Give constants **meaningful, contextual names** (e.g., `HeaderAIProvider` instead of just `ProviderHeader`).

## 2. Naming Conventions & Namespacing

In Go, constants in the same package must have unique names. To avoid collisions while maintaining semantic clarity, use **prefix-based namespacing**.

### Patterns

- **Log Formats**: Prefix with `LogFormat` or `LogMsg` followed by the context.
  - `LogFormatAnthroRequest`
  - `LogMsgOpenAIEnteringSync`
- **Headers**: Prefix with `Header`.
  - `HeaderAuthorization`
  - `HeaderXAccelBuffering`
- **Error Messages**: Prefix with `ErrMsg`.
  - `ErrMsgMethodNotAllowed`
  - `ErrMsgInvalidAnthroBody`

## 3. Architecture: Modular Internal Structure

The codebase follows **Clean Architecture** principles by segregating concerns into the `internal/` directory.

### Structural Layers

1.  [**`domain`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/domain): Zero-dependency layer containing core business entities (Request/Response models) and interfaces.
2.  [**`providers`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/providers): The Adapter layer. Each provider (Anthropic, OpenAI, etc.) is isolated. They translate between the internal `domain` contracts and external upstream APIs.
3.  [**`service`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/service): The Orchestration layer. Contains the `Registry` (managing provider instances) and `Mapper` (handling model routing logic).
4.  [**`transport`**](file:///Users/ez2/projects/personal/monorepo/apps/ai-gateway/internal/transport): The Interface layer. Handles HTTP protocol specifics, SSE streaming delivery, and cross-cutting middleware.

### Cross-Layer Communication

- **Dependency Inversion**: Outer layers (Transport, Providers) depend on inner layers (Domain, Service).
- **Communication by Contract**: Handlers interact with the `Service` layer via interfaces, and services interact with `Providers` via the `Provider` interface.

## 4. Error Handling Philosophy

- **Typed Errors**: Prefer using custom error types (e.g., `ErrRateLimitExceeded`) when specific upstream handling is required.
- **Consistent Responses**: All HTTP errors MUST use the shared error utility to ensure a uniform JSON format across the gateway.
- **Contextual Logging**: Always include the `RequestID` in error logs to facilitate troubleshooting.
