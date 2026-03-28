# API Specification: AI Intelligence Briefing (POST /api/ai/intelligence-briefing)

## 1. Executive Summary

The **Intelligence Briefing API** generates a personalized, narrative-driven financial summary (similar to an institutional investment briefing) for the user. It synthesizes real-time accounts, recent transactions, and active budget targets to deliver a "Daily Financial Status" report, surfacing critical alerts, savings rate velocity, and net worth milestones.

---

## 2. API Details

- **Endpoint**: `POST /api/ai/intelligence-briefing`
- **Authentication**: Institutional Session Required.

### 2.1 Input (JSON Payload)

```json
{
  "accounts": [{ "balance": 500000000 }],
  "transactions": [{ "date": "2024-03-21", "deposit": 15000000 }],
  "budget": [],
  "loans": [],
  "modelId": "github-gpt-4o"
}
```

### 2.2 Output (JSON Response Format)

```json
{
  "briefing": "Your net worth has stabilized this month. Savings rate is at 35%, primarily driven by March bonuses...",
  "alerts": [{ "type": "warning", "message": "High spending in 'Entertainment' detected (150% of budget)." }],
  "highlights": ["Net Worth Milestone Reached", "Debt Reduction Progress"]
}
```

---

## 3. Logic & Process Flow

### 3.1 Synthesis Pipeline

1.  **Aggregator**: Computes `totalAssets`, `totalLiabilities`, `netWorth`, and current-month `cashFlow`.
2.  **Prompt Engineering**: Loading the `intelligence-briefing` task template and replacing placeholders with computed financial metrics.
3.  **LLM Generation**: Uses the `generateText` SDK to produce a narrative report based on the provided JSON snapshots of limited accounts and transactions.
4.  **JSON Sanitization**: Applies a robust regex-based cleaning to the AI output to fix common formatting errors (e.g., thousands separators like `1.234.567` which break standard `JSON.parse`).

---

## 4. Technical Requirements

### 4.1 Heuristic Computations

- **Savings Rate**: `((Income - Expense) / Income) * 100`.
- **Cash Flow**: Difference between current month deposits and payments.

### 4.2 Data Sampling (Optimization)

To prevent prompt token overflow:

- **Accounts**: Sliced to top 5.
- **Transactions**: Sliced to most recent 10.
- **Budget**: Sliced to top 5 categories.

---

## 5. Edge Cases & Resilience

### 5.1 Zero Cash Flow

- If income is 0, the savings rate defaults to `0` to prevent division-by-zero errors.
- The AI briefing is instructed to prioritize "Cost Cutting" advice in zero-income scenarios.

### 5.2 Formatting Resilience

- **VND Separators**: Since Vietnamese LLMs often output numbers with dots (e.g., `5.000.000`), the API implements a post-processing cleaning step to ensure numeric values are valid for JSON parsing (`5000000`).

### 5.3 Fallback Narrative

- If the AI fails to generate a valid response or if the JSON is unrepairable, the API returns a generic, safe response: _"Your financial dashboard is ready. Review your assets and liabilities to stay on track."_

---

## 6. Non-Functional Requirements (NFR)

### 6.1 Performance & Resource Cap

- **CPU Time**: Heavy regex cleaning and LLM token processing.
- **Duration**: `< 10,000ms`.
- **Latency Masking**: Designed for use in dashboard headers where initial hydration occurs before the AI briefing is fully visible.
