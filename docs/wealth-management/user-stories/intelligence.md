# User Stories: Intelligence Center (AI Synthesis)

## 1. Persona Definition: AI-First Investor

- **Role**: Specialized Wealth Manager focused on natural language query/response for data retrieval and complex synthesis.
- **Needs**: Fast, contextual chat, accurate summarization, data-driven answers, and "Tools" (API access).

---

## 2. Natural Language Querying (Chat)

### US-INT-001: Direct Ledger Querying (RAG)

- **Story**: As an AI-First Investor, I want to ask "How much did I spend at Grab last month?" in natural language, so that I don't have to manually apply multiple filters in the transaction ledger.
- **Acceptance Criteria**:
  - AI must have current context of the user's transactions for the specified period.
  - Response must include the absolute VND total and the # of transactions.

### US-INT-002: Real-time Stock Research Query

- **Story**: As an AI-First Investor, I want to ask "Tell me about the current sentiment for HPG," so that the AI can fetch the latest news and summarize the consensus for me.
- **Acceptance Criteria**:
  - AI must trigger a "Tool Call" to the News Analyze API.
  - Summary must include: Sentiment Score, Key News, and Potential Catalyst.

---

## 3. Daily Synthesis (Briefing)

### US-INT-003: Morning Commanders Briefing

- **Story**: As an AI-First Investor, I want an automated summary of my portfolio growth, budget remaining, and market status when I first log in, so that I map out my financial day in under 30 seconds.
- **Acceptance Criteria**:
  - Must be generated as a 3-bullet list (Portfolio, Budget, Market).
  - Must use actual API-driven totals from the system state.

### US-INT-004: Ticker-Specific Intelligence Analysis

- **Story**: As an AI-First Investor, I want to ask "Should I sell FPT now or wait?" and have the AI analyze (1) my current cost basis, (2) the current market trend, and (3) the technical indicators, so that I have a multi-layered decision support.
- **Acceptance Criteria**:
  - AI must analyze the user's _actual holding_ cost (from Accounts/Transactions).
  - AI must analyze the _live price_ and _technicals_ (from Investments).
  - AI must offer a "Neutral / Bullish / Bearish" suggestion based on (1) & (2).

---

## 4. Interaction & UX

### US-INT-005: Interactive Markdown UI Cards

- **Story**: As an AI-First Investor, I want the AI to render a "Ticker Card" or "Category Pie Chart" directly inside the chat bubbles, so that I have a visual representation of what I'm asking for.
- **Acceptance Criteria**:
  - Supports Mermaid diagrams or custom React component rendering (e.g., Ticker Price chip).
  - Charts must be interactive (e.g., hoverable).

---

## 5. Security & Accuracy

### US-INT-006: Numerical Hallucination Guard

- **Story**: As an AI-First Investor, I want the system to double-check any numerical totals against the raw API data before the AI presents them to me, so that I can trust the advisor with my capital decisions.
- **Acceptance Criteria**:
  - The AI prompt must emphasize the use of "Tools" for _all_ calculations.
  - Any answer without a backing "Tool Call" for numbers should be flagged or corrected.

---

## 6. Financial Analysis & Projections

### US-INT-007: AI Budget Optimization & Direct Implementation

- **Story**: As an AI-First Investor, I want to ask "Optimize my budget based on my last 3 months of spending," and have the AI produce a refined plan with an **"Apply New Budget"** button directly in the chat, so that I don't have to navigate to another page to execute the advice.
- **Acceptance Criteria**:
  - AI must aggregate last 90 days of transaction data per category to identify "realistic" spending floors.
  - Response must output a Markdown comparison table (Category | Current | Suggested | Reason).
  - Chat bubble must render an interactive **"Approve & Sync"** button that triggers the Budget API and updates persistent storage.
