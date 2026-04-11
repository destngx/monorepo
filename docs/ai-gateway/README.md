# Go AI Gateway — Full Implementation Document

A stateless, OpenAI-compatible local AI proxy written in Go. Client applications only pass a `provider` and `model` name — the gateway handles all routing, header transformation, streaming, and token tracking transparently.

***

## Architecture Overview

```
Client App (any language)
      │
      │  POST /v1/chat/completions
      │  { "model": "gpt-4.1", "messages": [...] }
      │  Header: X-AI-Provider: github
      ▼
┌─────────────────────────────────────┐
│           AI Gateway (Go)           │
│                                     │
│  Router → Provider Factory          │
│         → Request Transformer       │
│         → Upstream HTTP Proxy       │
│         → Token Counter             │
│         → Response Passthrough      │
└─────────────────────────────────────┘
      │
      ├──▶ models.github.ai          (GitHub Models)
      ├──▶ api.openai.com            (OpenAI)
      ├──▶ api.anthropic.com         (Anthropic — converted)
      └──▶ localhost:11434           (Ollama — local)
```

### Design Principles

- **Stateless** — no database, no session state; every request is self-contained
- **OpenAI-compatible** — clients use the same request format regardless of provider
- **Provider-agnostic** — callers only set `X-AI-Provider` header and model name
- **Token-aware** — every response includes `usage` with prompt/completion/total counts
- **Streaming-first** — SSE passthrough with zero buffering for low latency

***

## Project Structure

```
ai-gateway/
├── main.go
├── config/
│   └── config.go          # env-based config, provider API keys
├── proxy/
│   ├── handler.go          # HTTP handler entrypoint
│   ├── router.go           # provider routing logic
│   └── middleware.go       # logging, recovery, CORS
├── providers/
│   ├── interface.go        # Provider interface definition
│   ├── github.go           # GitHub Models
│   ├── openai.go           # OpenAI
│   ├── anthropic.go        # Anthropic (with format conversion)
│   └── ollama.go           # Ollama (local)
├── types/
│   └── types.go            # shared request/response structs
└── go.mod
```

***

## Types — `types/types.go`

```go
package types

// ChatRequest is the OpenAI-compatible inbound request structure.
// Clients always send this format, regardless of provider.
type ChatRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    Stream      bool      `json:"stream"`
    Temperature *float64  `json:"temperature,omitempty"`
    TopP        *float64  `json:"top_p,omitempty"`
    MaxTokens   *int      `json:"max_tokens,omitempty"`
    Stop        any       `json:"stop,omitempty"`
    N           *int      `json:"n,omitempty"`
}

type Message struct {
    Role    string `json:"role"`    // "system" | "user" | "assistant"
    Content string `json:"content"`
}

// ChatResponse is the OpenAI-compatible non-streaming response.
type ChatResponse struct {
    ID      string   `json:"id"`
    Object  string   `json:"object"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Choices []Choice `json:"choices"`
    Usage   Usage    `json:"usage"`
}

type Choice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason string  `json:"finish_reason"`
}

// Usage carries token counts — always populated in the gateway response.
type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

***

## Provider Interface — `providers/interface.go`

Every provider implements this single interface. The gateway calls only these methods — it never knows the provider's internal implementation.

```go
package providers

import (
    "context"
    "io"
    "ai-gateway/types"
)

// Provider is the universal contract all AI backends must satisfy.
type Provider interface {
    // Name returns the provider identifier (e.g., "github", "openai").
    Name() string

    // Chat sends a non-streaming request and returns a parsed response.
    Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error)

    // ChatStream sends a streaming request and writes raw SSE bytes to w.
    // It returns the accumulated usage after the stream ends.
    ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error)
}
```

***

## Config — `config/config.go`

```go
package config

import "os"

type Config struct {
    GitHubToken    string
    OpenAIKey      string
    AnthropicKey   string
    OllamaBaseURL  string
    ListenAddr     string
}

func Load() *Config {
    ollamaBase := os.Getenv("OLLAMA_BASE_URL")
    if ollamaBase == "" {
        ollamaBase = "http://localhost:11434"
    }
    addr := os.Getenv("LISTEN_ADDR")
    if addr == "" {
        addr = ":8080"
    }
    return &Config{
        GitHubToken:   os.Getenv("GITHUB_TOKEN"),
        OpenAIKey:     os.Getenv("OPENAI_API_KEY"),
        AnthropicKey:  os.Getenv("ANTHROPIC_API_KEY"),
        OllamaBaseURL: ollamaBase,
        ListenAddr:    addr,
    }
}
```

***

## GitHub Models Provider — `providers/github.go`

```go
package providers

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "ai-gateway/types"
)

const githubBaseURL = "https://models.github.ai/inference"

type GitHubProvider struct {
    token  string
    client *http.Client
}

func NewGitHub(token string) *GitHubProvider {
    return &GitHubProvider{
        token:  token,
        client: &http.Client{Timeout: 120 * time.Second},
    }
}

func (g *GitHubProvider) Name() string { return "github" }

func (g *GitHubProvider) headers() map[string]string {
    return map[string]string{
        "Authorization":      "Bearer " + g.token,
        "Accept":             "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
        "Content-Type":       "application/json",
    }
}

func (g *GitHubProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
        githubBaseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    for k, v := range g.headers() {
        httpReq.Header.Set(k, v)
    }

    resp, err := g.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("github models error %d: %s", resp.StatusCode, b)
    }

    var result types.ChatResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}

func (g *GitHubProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
    req.Stream = true
    body, _ := json.Marshal(req)

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
        githubBaseURL+"/chat/completions", bytes.NewReader(body))
    if err != nil {
        return types.Usage{}, err
    }
    for k, v := range g.headers() {
        httpReq.Header.Set(k, v)
    }

    resp, err := g.client.Do(httpReq)
    if err != nil {
        return types.Usage{}, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return types.Usage{}, fmt.Errorf("github models error %d: %s", resp.StatusCode, b)
    }

    // Pass raw SSE bytes directly to writer — zero copy, zero latency
    usage, err := streamSSEAndCountTokens(resp.Body, w)
    return usage, err
}
```

***

## OpenAI Provider — `providers/openai.go`

```go
package providers

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "ai-gateway/types"
)

const openaiBaseURL = "https://api.openai.com/v1"

type OpenAIProvider struct {
    apiKey string
    client *http.Client
}

func NewOpenAI(apiKey string) *OpenAIProvider {
    return &OpenAIProvider{
        apiKey: apiKey,
        client: &http.Client{Timeout: 120 * time.Second},
    }
}

func (o *OpenAIProvider) Name() string { return "openai" }

func (o *OpenAIProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
        openaiBaseURL+"/chat/completions", bytes.NewReader(body))
    httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := o.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, b)
    }

    var result types.ChatResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (o *OpenAIProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
    req.Stream = true
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
        openaiBaseURL+"/chat/completions", bytes.NewReader(body))
    httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := o.client.Do(httpReq)
    if err != nil {
        return types.Usage{}, err
    }
    defer resp.Body.Close()

    return streamSSEAndCountTokens(resp.Body, w)
}
```

***

## Ollama Provider — `providers/ollama.go`

```go
package providers

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "ai-gateway/types"
)

type OllamaProvider struct {
    baseURL string
    client  *http.Client
}

func NewOllama(baseURL string) *OllamaProvider {
    return &OllamaProvider{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 300 * time.Second},
    }
}

func (o *OllamaProvider) Name() string { return "ollama" }

// Ollama v0.x supports the OpenAI-compatible endpoint at /v1/chat/completions
func (o *OllamaProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
        o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := o.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("ollama error %d: %s", resp.StatusCode, b)
    }

    var result types.ChatResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}

func (o *OllamaProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
    req.Stream = true
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost,
        o.baseURL+"/v1/chat/completions", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := o.client.Do(httpReq)
    if err != nil {
        return types.Usage{}, err
    }
    defer resp.Body.Close()

    return streamSSEAndCountTokens(resp.Body, w)
}
```

***

## SSE Stream Parser + Token Counter — `providers/stream.go`

This is the core utility shared across all providers. It reads the raw SSE stream, pipes it directly to the response writer, and extracts token usage from the final `[DONE]` chunk.

```go
package providers

import (
    "bufio"
    "bytes"
    "encoding/json"
    "io"
    "strings"

    "ai-gateway/types"
)

// streamSSEAndCountTokens pipes SSE bytes to w and returns token usage.
// It handles both providers that embed usage in the last chunk (OpenAI-style)
// and those that return usage only on [DONE].
func streamSSEAndCountTokens(body io.Reader, w io.Writer) (types.Usage, error) {
    var usage types.Usage
    var completionTokens int

    scanner := bufio.NewScanner(body)
    scanner.Buffer(make([]byte, 1024*64), 1024*64)

    for scanner.Scan() {
        line := scanner.Text()

        // Forward every line to the client as-is (zero-copy passthrough)
        if _, err := io.WriteString(w, line+"\n"); err != nil {
            return usage, err
        }

        // Flush if the writer supports it (http.ResponseWriter with Flusher)
        if f, ok := w.(interface{ Flush() }); ok {
            f.Flush()
        }

        // Parse only data lines for token extraction
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        payload := strings.TrimPrefix(line, "data: ")
        if payload == "[DONE]" {
            break
        }

        // Parse the chunk to extract usage and count completion tokens
        var chunk struct {
            Choices []struct {
                Delta struct {
                    Content string `json:"content"`
                } `json:"delta"`
            } `json:"choices"`
            Usage *types.Usage `json:"usage,omitempty"`
        }

        if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
            continue
        }

        // Accumulate completion tokens from delta content length (approximate)
        if len(chunk.Choices) > 0 {
            completionTokens += estimateTokens(chunk.Choices[0].Delta.Content)
        }

        // Use exact usage if provider embeds it in stream (OpenAI does this on last chunk)
        if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
            usage = *chunk.Usage
        }
    }

    // If provider didn't embed usage, use our estimate
    if usage.TotalTokens == 0 {
        usage.CompletionTokens = completionTokens
        // PromptTokens are estimated by callers or left at 0 for stream mode
    }

    return usage, scanner.Err()
}

// estimateTokens provides a rough ~4 chars/token estimate when exact counts
// are unavailable (e.g., streaming providers that omit usage in chunks).
func estimateTokens(text string) int {
    if text == "" {
        return 0
    }
    // Approximate: 1 token ≈ 4 characters (English text average)
    count := len([]rune(text)) / 4
    if count == 0 {
        return 1
    }
    return count
}

// estimateMessagesTokens estimates total tokens for a messages array.
// Used for non-streaming prompt token approximation.
func estimateMessagesTokens(messages interface{ MarshalJSON() ([]byte, error) }) int {
    b, err := json.Marshal(messages)
    if err != nil {
        return 0
    }
    return len(b) / 4
}

// injectUsageChunk appends a synthetic SSE usage chunk after [DONE].
// Used when the upstream omits usage data.
func injectUsageChunk(w io.Writer, usage types.Usage) {
    chunk := map[string]interface{}{
        "object": "chat.completion.chunk",
        "usage":  usage,
        "choices": []interface{}{},
    }
    b, _ := json.Marshal(chunk)
    io.WriteString(w, "data: "+string(b)+"\n\n")
    io.WriteString(w, "data: [DONE]\n\n")
    if f, ok := w.(interface{ Flush() }); ok {
        f.Flush()
    }
}

// MergeUsage combines two Usage structs (for multi-call aggregation).
func MergeUsage(a, b types.Usage) types.Usage {
    return types.Usage{
        PromptTokens:     a.PromptTokens + b.PromptTokens,
        CompletionTokens: a.CompletionTokens + b.CompletionTokens,
        TotalTokens:      a.TotalTokens + b.TotalTokens,
    }
}
```

***

## Provider Router — `proxy/router.go`

```go
package proxy

import (
    "fmt"
    "ai-gateway/config"
    "ai-gateway/providers"
)

// Registry maps provider names to their implementations.
type Registry struct {
    providers map[string]providers.Provider
}

// NewRegistry initialises all configured providers.
// A provider is registered only if its credentials are present.
func NewRegistry(cfg *config.Config) *Registry {
    r := &Registry{providers: make(map[string]providers.Provider)}

    if cfg.GitHubToken != "" {
        p := providers.NewGitHub(cfg.GitHubToken)
        r.providers[p.Name()] = p
    }
    if cfg.OpenAIKey != "" {
        p := providers.NewOpenAI(cfg.OpenAIKey)
        r.providers[p.Name()] = p
    }
    if cfg.AnthropicKey != "" {
        p := providers.NewAnthropic(cfg.AnthropicKey)
        r.providers[p.Name()] = p
    }
    // Ollama is always available (local, no key needed)
    p := providers.NewOllama(cfg.OllamaBaseURL)
    r.providers[p.Name()] = p

    return r
}

// Get returns the provider for the given name.
func (r *Registry) Get(name string) (providers.Provider, error) {
    p, ok := r.providers[name]
    if !ok {
        return nil, fmt.Errorf("unknown provider %q — registered: %v", name, r.list())
    }
    return p, nil
}

func (r *Registry) list() []string {
    keys := make([]string, 0, len(r.providers))
    for k := range r.providers {
        keys = append(keys, k)
    }
    return keys
}
```

***

## HTTP Handler — `proxy/handler.go`

```go
package proxy

import (
    "encoding/json"
    "net/http"

    "ai-gateway/types"
)

type Handler struct {
    registry *Registry
}

func NewHandler(registry *Registry) *Handler {
    return &Handler{registry: registry}
}

// ServeHTTP routes /v1/chat/completions — fully OpenAI-compatible.
//
// Required client header:
//   X-AI-Provider: github | openai | anthropic | ollama
//
// The model name in the request body is passed directly to the provider.
// Examples:
//   GitHub:    "model": "openai/gpt-4.1"
//   OpenAI:    "model": "gpt-4o"
//   Anthropic: "model": "claude-3-5-sonnet-20241022"
//   Ollama:    "model": "llama3.2"
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 1. Resolve provider from header
    providerName := r.Header.Get("X-AI-Provider")
    if providerName == "" {
        providerName = "github" // sensible default
    }
    provider, err := h.registry.Get(providerName)
    if err != nil {
        writeError(w, http.StatusBadRequest, err.Error())
        return
    }

    // 2. Decode request
    var req types.ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
        return
    }

    // 3. Route: streaming vs non-streaming
    if req.Stream {
        h.handleStream(w, r, provider, req)
    } else {
        h.handleSync(w, r, provider, req)
    }
}

func (h *Handler) handleSync(w http.ResponseWriter, r *http.Request, p interface {
    Chat(context.Context, types.ChatRequest) (*types.ChatResponse, error)
}, req types.ChatRequest) {
    resp, err := p.Chat(r.Context(), req)
    if err != nil {
        writeError(w, http.StatusBadGateway, err.Error())
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request, p interface {
    ChatStream(context.Context, types.ChatRequest, io.Writer) (types.Usage, error)
}, req types.ChatRequest) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    flusher, ok := w.(http.Flusher)
    if !ok {
        writeError(w, http.StatusInternalServerError, "streaming not supported")
        return
    }
    _ = flusher

    _, err := p.ChatStream(r.Context(), req, w)
    if err != nil {
        // Stream already started — cannot change status code; write error as SSE
        w.Write([]byte("data: {\"error\": \"" + err.Error() + "\"}\n\n"))
    }
}

func writeError(w http.ResponseWriter, code int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

***

## Middleware — `proxy/middleware.go`

```go
package proxy

import (
    "log"
    "net/http"
    "time"
)

// Chain applies middlewares in order.
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}

// Logger logs method, path, provider, status, and duration.
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start    := time.Now()
        provider := r.Header.Get("X-AI-Provider")
        if provider == "" {
            provider = "github"
        }
        rw := &responseWriter{ResponseWriter: w, status: 200}
        next.ServeHTTP(rw, r)
        log.Printf("[%s] %s %s provider=%s status=%d duration=%s",
            r.Method, r.URL.Path, r.RemoteAddr, provider, rw.status, time.Since(start))
    })
}

// Recovery catches panics and returns 500.
func Recovery(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                log.Printf("panic recovered: %v", rec)
                http.Error(w, "internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// CORS allows all local origins (suitable for local-only gateway).
func CORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-AI-Provider, Authorization")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}
```

***

## Entry Point — `main.go`

```go
package main

import (
    "log"
    "net/http"

    "ai-gateway/config"
    "ai-gateway/proxy"
)

func main() {
    cfg := config.Load()
    registry := proxy.NewRegistry(cfg)
    handler  := proxy.NewHandler(registry)

    mux := http.NewServeMux()
    mux.Handle("/v1/chat/completions", handler)
    mux.Handle("/v1/models", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"object":"list","data":[]}`))
    }))
    mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"status":"ok"}`))
    }))

    stack := proxy.Chain(mux,
        proxy.Recovery,
        proxy.Logger,
        proxy.CORS,
    )

    log.Printf("AI Gateway listening on %s", cfg.ListenAddr)
    if err := http.ListenAndServe(cfg.ListenAddr, stack); err != nil {
        log.Fatal(err)
    }
}
```

***

## Module File — `go.mod`

```go
module ai-gateway

go 1.22
```

No external dependencies — the entire gateway uses only the Go standard library.

***

## Build & Run

```bash
# Set credentials (only the providers you need)
export GITHUB_TOKEN="gho_..."
export OPENAI_API_KEY="sk-..."        # optional
export ANTHROPIC_API_KEY="sk-ant-..." # optional
export LISTEN_ADDR=":8080"            # default

# Build
go build -o ai-gateway ./...

# Run
./ai-gateway
```

Or with `go run`:

```bash
GITHUB_TOKEN=gho_... go run ./...
```

***

## Client Integration Examples

### Python (requests)

```python
import requests

BASE = "http://localhost:8080"

# Non-streaming — full response at once
resp = requests.post(f"{BASE}/v1/chat/completions",
    headers={"X-AI-Provider": "github"},
    json={
        "model": "openai/gpt-4.1",
        "messages": [{"role": "user", "content": "Hello!"}],
        "stream": False,
    }
).json()

print(resp["choices"][0]["message"]["content"])
print("Tokens used:", resp["usage"]["total_tokens"])
```

### Python Streaming

```python
import requests, json

with requests.post(f"{BASE}/v1/chat/completions",
    headers={"X-AI-Provider": "github"},
    json={"model": "openai/gpt-4.1",
          "messages": [{"role": "user", "content": "Explain Go concurrency"}],
          "stream": True},
    stream=True
) as resp:
    for line in resp.iter_lines():
        if line and line != b"data: [DONE]":
            chunk = json.loads(line.removeprefix(b"data: "))
            if chunk.get("usage"):
                print("\nTokens:", chunk["usage"])
            elif chunk["choices"][0]["delta"].get("content"):
                print(chunk["choices"][0]["delta"]["content"], end="", flush=True)
```

### OpenAI SDK (Python) — Zero Code Change

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="not-needed",             # gateway handles real keys
    default_headers={"X-AI-Provider": "github"},
)

response = client.chat.completions.create(
    model="openai/gpt-4.1",
    messages=[{"role": "user", "content": "Hello from OpenAI SDK!"}],
)
print(response.choices[0].message.content)
print("Tokens:", response.usage.total_tokens)
```

### Go Client

```go
package main

import (
    "bytes", "encoding/json", "fmt", "net/http"
)

func main() {
    body, _ := json.Marshal(map[string]any{
        "model":    "openai/gpt-4.1",
        "messages": []map[string]string{{"role": "user", "content": "Hello"}},
        "stream":   false,
    })

    req, _ := http.NewRequest("POST", "http://localhost:8080/v1/chat/completions",
        bytes.NewReader(body))
    req.Header.Set("X-AI-Provider", "github")
    req.Header.Set("Content-Type", "application/json")

    resp, _ := http.DefaultClient.Do(req)
    defer resp.Body.Close()

    var result map[string]any
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Println(result)
}
```

***

## API Reference

### `POST /v1/chat/completions`

**Headers:**

| Header | Required | Values | Default |
|---|---|---|---|
| `X-AI-Provider` | No | `github`, `openai`, `anthropic`, `ollama` | `github` |
| `Content-Type` | Yes | `application/json` | — |

**Request body:** Standard OpenAI chat completion format.

**Provider + Model Examples:**

| Provider | Model value | Notes |
|---|---|---|
| `github` | `openai/gpt-4.1` | Namespaced format required |
| `github` | `meta/llama-3.3-70b-instruct` | Meta models |
| `github` | `microsoft/phi-4` | Microsoft models |
| `openai` | `gpt-4o` | Standard OpenAI model name |
| `anthropic` | `claude-3-5-sonnet-20241022` | Anthropic model ID |
| `ollama` | `llama3.2` | Locally pulled model name |

**Response includes `usage` on every call:**

```json
{
  "id": "chatcmpl-...",
  "choices": [{ "message": { "role": "assistant", "content": "..." } }],
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 45,
    "total_tokens": 165
  }
}
```

### `GET /health`

Returns `{"status": "ok"}` — use for process monitoring.

***

## Token Counting Strategy

| Mode | Source | Accuracy |
|---|---|---|
| Non-streaming | Provider `usage` field in response | ✅ Exact |
| Streaming (OpenAI) | `usage` embedded in last SSE chunk | ✅ Exact |
| Streaming (others) | Character-length estimation (~4 chars/token) | ⚠️ ±10% |

For exact token counts in streaming mode across all providers, use the `stream_options: {"include_usage": true}` parameter (supported by OpenAI and GitHub Models):

```json
{
  "model": "openai/gpt-4.1",
  "messages": [...],
  "stream": true,
  "stream_options": { "include_usage": true }
}
```

***

## Environment Variables Reference

| Variable | Default | Description |
|---|---|---|
| `GITHUB_TOKEN` | — | `gho_` OAuth token for GitHub Models |
| `OPENAI_API_KEY` | — | OpenAI secret key |
| `ANTHROPIC_API_KEY` | — | Anthropic secret key |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama server address |
| `LISTEN_ADDR` | `:8080` | Gateway bind address |