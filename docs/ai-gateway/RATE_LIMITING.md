# Provider Rate Limiting

The AI Gateway provides built-in rate limiting to protect upstream provider tokens (GitHub PATs, OpenAI Keys) and internal infrastructure.

## Core Algorithm: Token Bucket

We use a **Token Bucket** algorithm (via `golang.org/x/time/rate`) which allows for both a steady-state request rate and a configurable "burst" for handling sudden spikes.

### Definitions

- **RPM (Requests Per Minute)**: The sustained rate allowed over time.
- **Burst**: The maximum number of requests that can be handled simultaneously if tokens have accumulated in the bucket.

## Configuration

Rate limits are configured independently for each provider via environment variables.

| Provider      | RPM Env Var          | Burst Env Var          |
| :------------ | :------------------- | :--------------------- |
| **GitHub**    | `GITHUB_RATE_RPM`    | `GITHUB_RATE_BURST`    |
| **OpenAI**    | `OPENAI_RATE_RPM`    | `OPENAI_RATE_BURST`    |
| **Anthropic** | `ANTHROPIC_RATE_RPM` | `ANTHROPIC_RATE_BURST` |
| **Ollama**    | `OLLAMA_RATE_RPM`    | `OLLAMA_RATE_BURST`    |

### Example `.env.local`

```bash
# Allow 5000 requests per hour (approx 83 per minute) for GitHub
GITHUB_RATE_RPM=83
GITHUB_RATE_BURST=10

# Allow 10 requests per minute for local Ollama
OLLAMA_RATE_RPM=10
OLLAMA_RATE_BURST=2
```

---

## 429 Too Many Requests

When a local gateway limit is exceeded, the Gateway returns an **HTTP 429** status code.

### JSON Response Body

The response includes the specific provider that hit the limit and a stack trace for debugging internal orchestrator logic.

```json
{
  "error": "gateway rate limit exceeded for provider \"github\"",
  "stack": "...",
  "status": 429
}
```

### Response Headers

Unlike standard 429 responses, the Gateway currently enforces a **local** bucket. It does not yet include a `Retry-After` header, as the token bucket refilling is continuous (sub-second granularity). Clients should implement **Exponential Backoff** for retries.

---

## Implementation Detail (Decorator Pattern)

The rate limiter is implemented as a **Provider Decorator**. This means the underlying provider logic remains unaware of the rate limiter, allowing for clean separation of concerns.

```go
// router.go logic
r.register(providers.NewRateLimitedProvider(
    providers.NewGitHub(cfg.GitHubToken),
    cfg.GitHubRate.RPM, cfg.GitHubRate.Burst,
))
```
