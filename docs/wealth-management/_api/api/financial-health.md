# API Specification: AI Financial Health Analysis (POST /api/ai/financial-health)

## 1. Executive Summary

The **Financial Health API** is a high-level cognitive engine that synthesizes a user's entire financial state—including Net Worth, Assets, Liabilities, Loans, and Historical Spending—into a professional, quantified "Financial Physical" report. It leverages LLMs to provide objective grading and strategic advice for wealth optimization, effectively acting as an automated Chief Financial Officer (CFO).

---

## 2. API Details

- **Endpoint**: `POST /api/ai/financial-health`
- **Authentication**: Institutional Session Required.

### 2.1 Input (JSON Payload)

```json
{
  "accounts": [{ "name": "Main Bank", "balance": 500000000, "category": "Bank" }],
  "months": [{ "month": "Jan", "spent": 15000000, "income": 50000000 }],
  "loans": [{ "name": "Mortgage", "yearlyRemaining": 1200000000 }]
}
```

### 2.2 Output (JSON Response Format)

```json
{
  "overallScore": 76,
  "netWorthTrend": "positive",
  "keyInsights": [
    {
      "area": "Liquidity",
      "status": "Healthy",
      "note": "Emergency fund is fully funded (6+ months expenses)."
    }
  ],
  "recommendations": [
    "Increase allocation to equities to improve long-term yield.",
    "Refinance mortgage to capture lower interest rate environment."
  ],
  "healthGauge": {
    "savingsRate": 0.35,
    "debtToIncome": 0.12
  }
}
```

---

## 3. Logic & Process Flow

### 3.1 Financial Synthesis Pipeline

1.  **Metric Aggregation**: Computes Net Worth, Total Assets, and Total Liabilities from the provided account and loan arrays.
2.  **Instruction Building**: Calls `buildFinancialHealthPrompt` to construct a data-dense task for the AI, integrating historical spending trends.
3.  **Action Loading**: Retrieves the `financial-health` specialized persona prompt from the `AIOrchestrator` registry.
4.  **LLM Execution**: Runs `AIOrchestrator.runJson` to generate a structured, schema-validated report.

---

## 4. Technical Requirements

### 4.1 Computational Logic

- **Net Worth Calculation**: `(Total positive balances) - (Total negative balances + Total loan debt)`.
- **Debt-to-Asset Ratio**: Derived in the prompt-building phase to give the AI context on leverage risks.

### 4.2 AI Orchestration

- Uses the centralized `AIOrchestrator` with specialized system prompts to ensure a "Professional Financial Advisor" tone and zero hallucination of numeric data.

---

## 5. Edge Cases & Resilience

### 5.1 Missing Data

- **Incomplete History**: If `months[]` is empty, the AI is instructed to skip "Trend Analysis" and focus on the "Static Snapshot" of Net Worth only.
- **Zero Net Worth**: The system handles negative net worth (Debt-heavy) by triggering the "Debt Rehabilitation" branch of the system prompt.

### 5.2 Input Validation

- **Negative Balances**: Successfully maps credit cards (negative balance) as Liabilities rather than failing the asset calculation.
- **Empty Accounts**: If no accounts are provided, the API returns a standard `AppError` before executing the expensive LLM.

---

## 6. Non-Functional Requirements (NFR)

### 6.1 Performance & Security

- **LLM Latency**: `< 12,000ms` for a full financial synthesis.
- **Data Privacy**: The AI processing is transient; raw financial data is sent to the LLM (GPT-4o) via a stateless API call without persistent storage in the AI's memory.
