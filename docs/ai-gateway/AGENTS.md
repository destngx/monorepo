# AI Gateway: Engineering & Agent Standards

This document defines the core engineering principles for the AI Gateway (Go).

## 🏛️ Architecture: Clean & Modular

The codebase segregates concerns into several specialized layers within the `internal/` directory.

### 📁 Structural Layers

1. **`domain/`**: Zero-dependency layer containing core business entities (Request/Response models) and interfaces.
2. **`providers/`**: The Adapter layer. Translations between internal `domain` contracts and external upstream APIs (Anthropic, OpenAI, etc.).
3. **`service/`**: The Orchestration layer (Registry, Mapper).
4. **`transport/`**: The Interface layer handling HTTP protocol, SSE streaming, and middleware.

## 📜 Coding Policies

### 🚫 Zero Hardcoded Strings

We maintain a strict **no hardcoded strings** policy in application logic.

- All literals (headers, log formats, error messages, content types) must be extracted into package-level constants or `constants.go`.
- Use **prefix-based namespacing**: `Header...`, `LogFormat...`, `ErrMsg...`.

### ⚠️ Error Handling

- **Typed Errors**: Prefer custom error types for upstream-specific logic.
- **Shared Utils**: Always use the shared error utility for HTTP responses to ensure uniform JSON formatting.
- **Traceability**: Always include `RequestID` in error logs.

## 🔌 Adding New Providers

- Implement the `Provider` interface located in the `domain` layer.
- Ensure all provider-specific constants follow the naming conventions.
- Map external errors to internal `domain` error types where possible.
