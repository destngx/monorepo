# API Specification: AI Transaction Review (POST /api/ai/transaction-review)

## 1. Executive Summary

The **Transaction Review API** is a specialized auditing engine that analyzes recent spending patterns. It identifies anomalous transactions, categorizes unexpected expenses, and provides "Insight Cards" regarding the user's spending habits. It serves as a granular financial monitor, transforming a raw list of ledger entries into actionable behavioral intelligence.

---

## 2. API Details

- **Endpoint**: `POST /api/ai/transaction-review`
- **Authentication**: Institutional Session Required.

### 2.1 Input (JSON Payload)

```json
{
  "transactions": [{ "date": "2024-03-21", "payee": "Starbucks", "payment": 65000, "category": "Food" }]
}
```

### 2.2 Output (JSON Response Format)

```json
{
  "review": {
    "title": "Unusual Food Spending",
    "observation": "Average daily 'Food' spending is up 15%. Multiple coffee transactions noted.",
    "recommendation": "Consider consolidating daily coffee purchases to reduce per-unit overhead.",
    "type": "ADVISORY"
  }
}
```

---

## 3. Logic & Process Flow

### 3.1 Audit Pipeline

1.  **Ingestion**: Receives a JSON array of recent transactions.
2.  **Context Construction**: Loads the `transaction-review` task template and injects the raw transaction data.
3.  **Prompt Enrichment**: Appends the `STRUCTURED_INSIGHT_FORMAT_INSTRUCTION` to ensure the AI outputs exactly one of the known "Insight" JSON schemas.
4.  **LLM Generation**: Uses the high-reasoning `github-gpt-4o` class models for behavioral analysis.
5.  **Extraction**: Calls `extractAndParseJSON` to isolate the JSON object from the AI's natural language explanation.

---

## 4. Technical Requirements

### 4.1 AI Orchestration

- **Model**: `github-gpt-4.1` (or equivalent high-reasoning class).
- **Structure**: Enforces `StructuredInsight` type matching, including `title`, `observation`, `recommendation`, and `type`.

### 4.2 Error Mitigation

- **Regex Extraction**: Protects against model "Chatter" by isolating the `{}` JSON block regardless of surrounding text.
- **Fail-safe Logic**: If the AI output is completely unparseable, the API returns the raw AI `text` as the `review` string fallback.

---

## 5. Edge Cases & Resilience

### 5.1 Volume & Noise

- **Overwhelming Transaction Lists**: The system prioritizes high-value ($>500k$ VND) and recurring transactions for analytical priority to reduce computational noise.
- **Ambiguous Payees**: Payees with generic names (e.g., "Transfer") are scrutinized against historical "Memo" strings to identify hidden expense types.

### 5.2 Empty Histories

- If zero transactions are provided, the API returns a status 400 `AppError` suggesting "Provide recent transactions to start the review."

---

## 6. Non-Functional Requirements (NFR)

### 6.1 Performance & Security

- **Auditing Latency**: `< 8,000ms`.
- **Privacy Standard**: Zero persistence of raw transaction strings in the backend logs; data is forwarded to the LLM over an encrypted channel and purged from the function memory upon response.
