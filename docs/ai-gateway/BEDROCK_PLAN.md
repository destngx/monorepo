Here is the complete implementation plan, all files are in the downloadable archive above.

---

## Project Structure

```
bedrock-go/
├── cmd/main.go                    ← entry point — runs all 3 demo flows
├── internal/
│   ├── awsconfig/config.go        ← credential loading, zero keys
│   └── bedrock/
│       ├── client.go              ← BedrockRuntime wrapper
│       ├── converse.go            ← synchronous Converse API
│       ├── stream.go              ← streaming ConverseStream API
│       └── models.go              ← model ID constants
├── .env.example                   ← copy → .env, gitignore
├── Makefile
└── go.mod                         ← requires Go 1.24, AWS SDK Go v2
```

---

## Auth Flow (End-to-End)

```
Your Go App
     │
     ▼
awsconfig.Load()   ←── config.LoadDefaultConfig()
     │    Scans credential chain in priority order:
     │    1. AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars
     │    2. ~/.aws/config  (SSO profile — recommended)
     │    3. ~/.aws/credentials (static keys — fallback)
     │    4. EC2/ECS/Lambda instance metadata (IMDS)
     │
     ▼
aws.Config { Credentials: CredentialsProvider }
     │
     ▼
bedrock.NewClient(cfg, modelID)
     │   bedrockruntime.NewFromConfig(cfg)
     ▼
client.Converse() or client.ConverseStream()
     │
     │  SDK Middleware Pipeline (automatic):
     │  ① Serialize → JSON body
     │  ② Resolve endpoint: bedrock-runtime.<region>.amazonaws.com
     │  ③ Retrieve creds from provider (auto-refresh if expired)
     │  ④ Sign request with SigV4 (HMAC-SHA256)
     │  ⑤ HTTPS POST /model/{modelId}/converse
     │
     ▼
Amazon Bedrock Runtime
     │  ✔ Validates SigV4 signature
     │  ✔ Checks IAM: bedrock:InvokeModel
     │  ✔ Routes to Claude Sonnet 4.6 / Opus 4.6
     ▼
ConverseResponse { Text, InputTokens, OutputTokens }
```

---

## Quick Start

```bash
# 1. Login with your existing SSO profile
aws sso login --profile bedrock-dev

# 2. Attach Bedrock permission to your existing role (one-time)
# Add: bedrock:InvokeModel + bedrock:InvokeModelWithResponseStream
# On: arn:aws:bedrock:ap-southeast-1::foundation-model/*

# 3. Run
cd bedrock-go
go mod tidy
AWS_PROFILE=bedrock-dev go run ./cmd
```

---

## Three Request Patterns Implemented

| Pattern                        | File          | Use Case                                                                                                                                                                                          |
| ------------------------------ | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Synchronous** `Converse`     | `converse.go` | Single Q&A, batch jobs, structured output [docs.aws.amazon](https://docs.aws.amazon.com/code-library/latest/ug/go_2_bedrock_code_examples.html)                                                   |
| **Streaming** `ConverseStream` | `stream.go`   | Chat UI, real-time token-by-token display [docs.aws.amazon](https://docs.aws.amazon.com/code-library/latest/ug/go_2_bedrock-runtime_code_examples.html)                                           |
| **Multi-turn chat**            | `main.go`     | Conversation history with context accumulation [docs.aws.amazon](https://docs.aws.amazon.com/bedrock/latest/userguide/bedrock-runtime_example_bedrock-runtime_Scenario_InvokeModels_section.html) |

---

## Key Design Decisions

- **`Converse` API over `InvokeModel`** — `Converse` is model-agnostic and consistent across all Bedrock models; you can swap `ModelSonnet46` → `ModelOpus46` in one line with no request format changes [pkg.go](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/bedrockruntime)
- **Cross-region inference profile IDs** (`us.anthropic.*`) — automatically routes to the nearest available region for lower latency and higher quota [docs.aws.amazon](https://docs.aws.amazon.com/code-library/latest/ug/go_2_bedrock_code_examples.html)
- **`awsconfig.Load()` validates credentials at startup** — fails fast with a clear `run 'aws sso login'` error message instead of a cryptic HTTP 403 mid-request [docs.aws.amazon](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-auth.html)
- **`client.WithModel()`** — returns a lightweight new client reusing the same HTTP connection pool, safe for concurrent use with different models per request [pkg.go](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/bedrockruntime)

> ⚠️ **Verify model IDs**: The IDs in `models.go` follow the cross-region inference profile pattern. Confirm the exact IDs for your region/account in the AWS Console → Bedrock → Model access page, as they can vary by availability date.
