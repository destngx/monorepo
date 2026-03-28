# Specification: Wealth Intelligence Center (The AI CFO)

**Author**: Product Owner / AI Strategist  
**Intended Audience**: Engineering (AI/Backend)  
**Status**: Mid-Stage Implementation  
**Keywords**: GPT-4o, Tool Selection, Financial Context, Synthesis Engine

---

## 1. Product Intent

To transform raw numerical data into human-readable strategic insights via an AI-first interaction layer. The Intelligence Center represents the "Advisor" level of the ecosystem.

---

## 2. Intelligence Briefing (Daily Command)

**BA Requirement**: Provide a prioritized "Morning Brief" that synthesizes global market context with private portfolio state.

### 2.1 Narrative Generation

- **Feature**: 3 Actionable Bullet Points (Gains/Losses, Rebalancing Needs, Budget Alerts).
- **Logic**: Uses GPT-4o to analyze accounts, budgets, and ticker drift.
- **AC**: Brief MUST be updated at first user login of the day.

---

## 3. Conversational AI Interface (Chat)

### 3.1 Intent-Based Tool Execution

- **Feature**: Natural Language query for financial lookup.
- **AC**: MUST accurately choose between `getAccounts`, `getTransactions`, or `getMarketIntel` tools based on the user's question.

### 3.2 Streaming & Responsiveness

- **AC**: **TTFT** (Time to First Token) < 1.0s.
- **Visual**: Glassmorphism bubbles with streaming responses.

---

## 4. Financial Health (Holistic Score 0-100)

**PO Goal**: Gamify the journey toward Financial Freedom through diagnostic ratios.

- **Metric 1**: Savings Rate (> 30%).
- **Metric 2**: Emergency Fund Coverage (6 months target).
- **Metric 3**: Multi-Portfolio Diversification Ratio.
- **AC**: AI must provide a "Doctor's Recommendation" based on the health score analysis.

---

## 5. Technical Requirements for Engineering

- **Context Injection**: Financial metadata (Account Names, Currencies) must be injected into the LLM system prompt securely.
- **PII Filtering**: No private bank IDs or real account numbers in the prompt; use masked titles.
- **NDJSON**: Stream responses via Next.js 13+ streaming patterns for better UX.

---

## 6. Engineering Success Criteria

- **Quality**: 95% relevance of AI suggestions (determined by user feedback loops).
- **Speed**: Full briefing generated in < 5 seconds.
- **Parity**: AI math (e.g., "What was my spend on Coffee last week?") must match the ledger counts exactly.
