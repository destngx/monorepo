# Specification: AI Orchestration & Tooling (Internal Intelligence)

**Author**: Product Owner / AI Strategist  
**Intended Audience**: Engineering Team (AI/Client-side)  
**Status**: Approved for Implementation  
**Keywords**: GPT-4o, Tool Selection, Financial Context, Tool Registry

---

## 1. Business Objective

To humanize financial data by transforming raw transactional numbers into actionable, narrative-driven insights. The AI acts as the user's "Chief Financial Officer" (CFO).

---

## 2. Core Operational Flow

**BA Requirement**: AI must have **Low Latency** (TTFT < 1s) and **Contextual Memory** (User Profile, Accounts, Market Context).

### 2.1 The "Think Tank" Strategy

- **Phase 1 (Intent Extraction)**: Analyze user prompt (e.g., "Analyze my crypto risk") and map to specific tools.
- **Phase 2 (Tool Execution)**: Parallel fetch of private account data and public market telemetry (Binance, VNStock).
- **Phase 3 (Synthesis)**: Orchestrate a multi-perspective response (Long-term Analyst vs. Tactical Trader).

---

## 3. Tool Capability Registry (BA/PO Scope)

The dev team must implement and maintain a registry of the following tool capabilities:

| Tool Category   | Purpose                                     | Business Case                            |
| :-------------- | :------------------------------------------ | :--------------------------------------- |
| **Data Pull**   | Fetch `Accounts`, `Transactions`, `Budget`. | Foundational state knowledge.            |
| **Market Scan** | Browse news, verify stock prices.           | External context for alpha generation.   |
| **Logic/Calc**  | Project goals, calculate net worth.         | Error-free mathematical synthesis.       |
| **Persistence** | Update `Budget`, save `Transactions`.       | Direct operational impact (Quick Entry). |

---

## 4. Prompt Engineering & Quality Guidelines

**PO Expectation**: Consistent persona across all touchpoints (Greeting, Chat, Briefing).

1.  **Persona**: Professional, sophisticated, analytical, yet encouraging.
2.  **Safety First**: Never provide direct investment "Advice"; always frame as "Analysis" or "Observations."
3.  **Data Privacy**: PII must never be sent to the model; only normalized financial strings and masked account titles.

---

## 5. Performance & UX Requirements (NFRs)

- **Streaming Implementation**: NDJSON streaming for real-time responsiveness.
- **Model Fallback**:
  - The engine calls one OpenAI-compatible provider with a model per use case (`AI_MODEL`, `AI_MODEL_*`) and retries once with `AI_FALLBACK_MODEL` ([REQUIREMENTS.md §10.1](../../REQUIREMENTS.md#101-architecture-decided)).
- **Effort and order** (WM-AI-16): each feature sets its reasoning effort and queue priority in `service/ai/schedule.go`. At most 4 model calls run at once, and identical cached requests share one call.
- **Tool Concurrency**: Parallelize tool execution whenever multiple sources are required (e.g., Accounts + Market Data).

---

## 6. Engineering Success Criteria

- **TTFT (Time to First Token)**: < 1.0s.
- **Tool Integrity**: 98% accuracy in mapping prompt intent to the correct registry tool.
- **Stability**: Graceful degradation if external market APIs are offline.
