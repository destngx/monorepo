# Streaming & Token Tracking

This document explains how the AI Gateway handles real-time streaming (SSE) and tracks token usage across different AI providers.

## Core Mechanism: SSE Proxying

The gateway uses a "pipe and parse" strategy for streaming. When a client requests a streamed response (`stream: true`), the gateway:

1. Opens a persistent connection to the provider.
2. Forwards incoming data chunks to the client immediately (using `http.Flusher` for low latency).
3. In the background, parses every chunk to track generated content.

### SSE Handler Flow

The logic is encapsulated in `providers/stream.go` through the `streamSSEAndCountTokens` function.

```mermaid
sequenceDiagram
    participant P as AI Provider
    participant GW as AI Gateway
    participant C as Client App

    P->>GW: data: {"choices": [{"delta": {"content": "Hello"}}]}
    GW->>C: data: {"choices": [{"delta": {"content": "Hello"}}]}
    Note right of GW: Parse content: "Hello" (+1 token)

    P->>GW: data: [DONE]
    Note right of GW: Check if Provider sent Usage stats

    alt Provider sent Usage (e.g., OpenAI)
        GW->>C: data: [DONE]
    else Provider sent NO Usage (e.g., Ollama)
        GW->>C: data: {"usage": {"completion_tokens": 10...}}
        GW->>C: data: [DONE]
    end
```

---

## Token Estimation Strategy

Because different providers use different tokenizers (Tiktoken, SentencePiece, etc.), the Gateway uses a **standardized character-based estimation** when native usage data is missing.

### Estimation Rule

`1 Token ≈ 4 Characters (English)`

This provides a consistent metric across all models, preventing "Free" usage of local or smaller models that don't report stats.

### Tool Call Tracking

The gateway also tracks `tool_calls` in streams. It extracts the `arguments` delta and adds them to the `completion_tokens` count, ensuring that complex function calling remains measurable.

---

## Client Compatibility & State Tracking

Stream-based tools and CLIs (like **Claude Code**) are often very strict about partial JSON parsing. The Gateway implements several state-tracking mechanisms to prevent client-side "Drops":

- **Mandatory Field Initialization**: In `message_start` events, the Gateway explicitly initializes required fields (e.g., `content: []`, `stop_reason: null`). This prevents strict JSON parsers from failing on missing fields.
- **Strict Block Delimitation**: The Gateway uses a state machine (`textBlockStarted`, `activeToolIndex`) to ensure that Anthropic event streams are properly structured.
  - Text blocks are opened with `content_block_start` the moment text is received.
  - If a tool delta arrives while a text block is open, the text block is explicitly closed with `content_block_stop` before the tool block begins.
- **Continuous Block Indexing**: The Gateway maintains a serial `blockIndex` that increments across every new content block (text or tool). This ensures that clients like **Claude Code** can accurately reconstruct the message sequence without index collisions.

---

## Error Handling in Streams

Errors can occur at two stages of a stream:

### 1. Connection Initialization

If the [Rate Limit](./RATE_LIMITING.md) is exceeded before the stream begins, the Gateway returns a standard **HTTP 429** response.

### 2. Mid-Stream Failure

If the provider fails _after_ the stream has started, the Gateway cannot change the HTTP status code. The handling depends on whether data has already been emitted:

- **Before First Event**: If the upstream fails before any events are successfully converted, the Gateway emits a final Anthropic-style `error` event:
  ```json
  event: error
  data: {"error": {"type": "api_error", "message": "Upstream timeout"}}
  ```
- **During Data Flow**: If events have already been emitted, the Gateway relies on the standard SSE termination (or an explicit `message_delta` with an error reason) to signal the failure.

### Request ID Correlation

A unique **Request ID** is injected into every stream log. Use this ID to match client-side errors with specific failure points in the server logs.

### Proxy Support (`X-Accel-Buffering`)

For streams behind Nginx or Cloudflare, the Gateway sets `X-Accel-Buffering: no`. This disables downstream proxy buffering, ensuring that tokens reach the user as they are generated, rather than being batched into larger (higher latency) chunks. This is critical for real-time interactivity.

---

## Performance Considerations

- **Buffered Scanning**: Uses `bufio.Scanner` with a 64KB buffer to handle ultra-long JSON payloads (common in code generation).
- **Zero-Latency**: The `Unmarshal` logic occurs _after_ the raw bytes have been forwarded to the client, ensuring the Gateway adds zero perceptible latency to the stream.
