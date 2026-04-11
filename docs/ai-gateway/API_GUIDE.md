# AI Gateway Integration Guide

This guide provides technical specifications for client applications integrating with the AI Gateway.

## Authentication & Base URL

The gateway is typically accessed at: `http://localhost:8080/v1`

**Why `/v1`?**
This gateway uses the `/v1` path prefix for two main reasons:

1.  **Drop-in Compatibility**: Most AI SDKs (like the `openai` Python/JS packages) expect the base URL to end with `/v1`. Following this convention allows the gateway to function as a seamless replacement for existing integrations.
2.  **Versioning**: It provides a stable versioning contract. Any future breaking changes can be introduced under `/v2` without disrupting existing v1 clients.

**Authentication:** Clients do not need to provide provider-specific API keys. These are managed by the gateway server via environment variables.

---

## Required Headers

Every request to the chat completion or models endpoint should include the following:

| Header          | Description                  | Possible Values                           | Default  |
| :-------------- | :--------------------------- | :---------------------------------------- | :------- |
| `X-AI-Provider` | Selects which backend to use | `github`, `openai`, `anthropic`, `ollama` | `github` |
| `Content-Type`  | Standard JSON header         | `application/json`                        | —        |

---

## API Endpoints

### 1. Chat Completions

`POST /v1/chat/completions`

Follows the standard OpenAI Request Body format. Use the `"stream"` property to toggle between response modes.

#### Response Modes

| Mode                | Request Param     | HTTP Status | Content-Type        |
| :------------------ | :---------------- | :---------- | :------------------ |
| **Sync (JSON)**     | `"stream": false` | `200 OK`    | `application/json`  |
| **Streaming (SSE)** | `"stream": true`  | `200 OK`    | `text/event-stream` |

**Example Request:**

```json
{
  "model": "gpt-4o",
  "messages": [
    { "role": "system", "content": "You are a helpful assistant." },
    { "role": "user", "content": "How do I implement SSE in Go?" }
  ],
  "stream": true
}
```

### 2. List Models

`GET /v1/models`

Returns a list of models available from the provider specified in the `X-AI-Provider` header.

### 3. Embeddings

`POST /v1/embeddings`

Converts text inputs into vector representations.

**Example Request:**

```json
{
  "model": "text-embedding-3-small",
  "input": "Semantic search is powerful"
}
```

**Example Request (Batch):**

```json
{
  "model": "text-embedding-3-small",
  "input": ["First sentence", "Second sentence"]
}
```

### 4. Tool / Function Calling

`POST /v1/chat/completions`

The gateway supports transparent passthrough of OpenAI-compatible tool definitions. It also handles the complex translation between OpenAI and Anthropic tool formats. This allows you to use frameworks like **LangGraph** or **Graph-weave** with any backend.

**Example Request (with Tools):**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-AI-Provider: github" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "What is the weather in London?"}],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get current weather",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {"type": "string"}
            }
          }
        }
      }
    ]
  }'
```

The gateway will return a standard OpenAI-compatible `tool_calls` object in the response.

### 5. Service Health

`GET /health`

Returns the global status of the gateway and a list of all currently registered (configured + pingable) providers.

---

## Provider-Specific Context

| Provider      | `model` format examples                         | Notes                                          |
| :------------ | :---------------------------------------------- | :--------------------------------------------- |
| **GitHub**    | `openai/gpt-4.1`, `meta/llama-3.3-70b-instruct` | Requires publisher namespace.                  |
| **OpenAI**    | `gpt-4o`, `gpt-3.5-turbo`                       | Standard OpenAI identifiers.                   |
| **Anthropic** | `claude-3-5-sonnet-20241022`                    | Gateway converts messages to Anthropic format. |
| **Ollama**    | `llama3.2`, `mistral`                           | Proxies to local instance.                     |

---

## Error Handling

| Status Code         | Meaning          | Reason                                                                                        |
| :------------------ | :--------------- | :-------------------------------------------------------------------------------------------- |
| **400 Bad Request** | Provider Error   | Missing `X-AI-Provider` or invalid JSON body.                                                 |
| **404 Not Found**   | Unready Provider | The requested provider is either not configured (missing token) or unreachable (ping failed). |
| **502 Bad Gateway** | Upstream Error   | The AI provider (e.g., GitHub) returned an error or timed out.                                |

---

## Integration Examples

### Python (OpenAI SDK Companion)

You can use existing OpenAI-integrated libraries by simply overriding the base URL:

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-required",
    default_headers={"X-AI-Provider": "anthropic"}
)

response = client.chat.completions.create(
    model="claude-3-5-sonnet-20241022",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

### cURL (Streaming)

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-AI-Provider: github" \
  -d '{
    "model": "openai/gpt-4.1",
    "stream": true,
    "messages": [{"role": "user", "content": "Briefly describe Go channels."}]
  }'
```

### cURL (Models)

```bash
curl -H "X-AI-Provider: ollama" http://localhost:8080/v1/models
```
