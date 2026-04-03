# Use case: Financial Research Pipeline Stagnation Scenario

1. Tenant: Hedge fund
Trigger: Analyst query
Workflow: `financial-research:v2.1.0`

Flow
- Web search agent → SQL agent → synthesis
- Orchestrator loops between sources
- Stagnation Case
- Same query repeated across tools
- Detector triggers after threshold
- Forces exit with partial answer

2. Tenant: `finbank`
Workflow ID: `finbank/10k_analyzer/v3.0.0`

Stagnation Scenario:

- Loop 1: Orchestrator routes to SubAgent_extract_risk_factors
- Loop 2: Same routing decision (repetition)
- Loop 3: Same again → `StagnationDetector` triggers
    - Sliding window: last 3 orchestrator outputs identical
    - Increments stagnation counter to 3 → FORCE_EXIT

Recovery: Return last successful summary + {"stagnation": true, "partial_results": true}

3. Tenant Profile: Hedge fund quantitative research desk.

Trigger: Analyst requests a synthesis of Q3 earnings vs. historical SQL performance data.

Workflow ID: `quant-research:v3.0.0`

Execution:

  - Orchestrator directs web_search_agent to fetch Q3 transcripts.
  - Orchestrator directs sql_agent to query the internal PostgreSQL data warehouse.
  - The sql_agent repeatedly fails to write a correct JOIN query.

Edge Case Handled: Stagnation. The Orchestrator continually requests the sql_agent to try again. StagnationDetectorNode registers 3 repetitive routing intents, flags stagnation_detected=True, forces the graph to output_guardrail, and returns partial data with a warning.
