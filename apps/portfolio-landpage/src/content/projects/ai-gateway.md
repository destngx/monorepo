---
title: AI Gateway
summary: A stateless Go proxy with OpenAI- and Anthropic-compatible endpoints in front of Copilot, OpenAI, Anthropic, Bedrock, Ollama, and more, with per-provider rate limits and failover.
year: 2026
status: active
stack: [Go, net/http, AWS SDK for Go v2, golang.org/x/time/rate, Swagger]
repo: https://github.com/destngx/monorepo/tree/main/apps/ai-gateway
featured: true
order: 2
sfx: シュッ
---

## The problem

Clients such as Claude Code, my wealth dashboard, and GraphWeave needed to call whichever model was available. Without a shared layer, each client would hold provider keys and handle each vendor's wire format, including streaming and tool calls.

## What it does

- Serves OpenAI-style `/v1/chat/completions`, `/v1/responses`, `/v1/embeddings`, and `/v1/models`, plus Anthropic-style `/v1/messages` and `/v1/messages/count_tokens`.
- Routes each request to a provider by model name. Claude model families are mapped to gateway equivalents in one table.
- Falls back to a second provider when the primary hits a plan limit.
- Exposes `/health`, `/v1/usage`, `/metrics` with a dashboard, and Swagger docs at `/docs/`.

## How it's built

- Layered `internal/` packages: `domain` holds the shared OpenAI-compatible types, `providers` has one adapter per backend behind a common `Provider` interface, `service` holds the registry and model mapper, and `transport/http` holds handlers, SSE, and middleware.
- Routing uses the standard library `net/http`. Bedrock is called through the AWS SDK for Go v2.
- Every provider is wrapped in a rate-limit decorator that uses `golang.org/x/time/rate` token buckets.
- The gateway is stateless. It has no database and no session store, so it can run as a sidecar.
- Providers that do not report usage get an estimate of about four characters per token.

## Interesting bits

- Failover is a decorator: `NewFailoverProvider(primary, fallback)` switches over when an error is classified as a provider limit error, then keeps the fallback active for a 5-hour cooldown.
- At startup, each provider is checked in two phases, token presence and then a network ping, so only configured and reachable providers receive traffic.
- Bedrock is registered only if its AWS config loads. I used the Converse API instead of InvokeModel so one request shape works across Bedrock models.
- Requests for Anthropic models are translated from the OpenAI format into Anthropic's Messages API, with stream events mapped back, so OpenAI-style clients can reach Claude models.

## What's next

- Rate-limited responses do not include a `Retry-After` header yet. Clients have to back off on their own.
- The Bedrock model IDs in `models.go` still need to be confirmed against the account and region before use.
