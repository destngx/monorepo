# AI Gateway Integration Guide

This guide provides technical specifications for client applications integrating with the AI Gateway.

## Authentication & Base URL

The gateway is typically accessed at: `http://localhost:8080/v1`

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

### 3. Service Health

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
