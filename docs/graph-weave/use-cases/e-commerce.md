# Use cases: E-Commerce Customer Support

1. Tenant: Shopify-like SaaS
Trigger: Customer chat message
Workflow: `customer-support:v1.2.0`

Execution Flow
- InputGuardrail validates request
- GraphInitializer loads Tier 1 skills
- Orchestrator selects:
  - billing_system
  - shipping_api
- SkillLoader loads billing schema
- Subagent executes Stripe query
- Summary returned
- Orchestrator routes to shipping_agent
- Final response generated

Edge Cases
- Refund loop → stagnation detection triggers exit
- Malicious input → blocked by guardrail
- Token overflow → forced exit

2. Tenant: retailco (100K DAU)
Trigger: `POST /api/v1/execute with {"workflow_id": "customer_support", "input": "Where's my order #ORD-12345?"}`
Workflow ID: `retailco/customer_support/v2.1.0`

Step-by-Step Execution:

  - InputGuardrail: Validate input (<10K chars, no injection patterns)
  - GraphInitializer: Load workflow JSON from Redis `graphweave:workflow:retailco:customer_support:v2.1.0`
  - SkillLoader: Load Tier 1 (200 tokens): ["order_lookup_desc", "return_policy_desc", "escalate_desc"]
  - Orchestrator: LLM outputs:
    ``` json
    {
      "reasoning": "User needs order status. Requires order_lookup tool.",
      "routing_directive": "SubAgent_order_tools",
      "subagent_payload": {
        "objective": "Retrieve status for ORD-12345",
        "required_tools": ["order_api"]
      }
    }
    ```
  - SubAgentExecutor: Isolated subagent calls order API → returns {"status": "shipped", "eta": "2025-01-15"}
  - Orchestrator (2nd pass): Summarizes to user-facing response
  - OutputGuardrail: Validate no PII leakage
  - SSE Stream: Final response delivered

Edge Cases:
  - Guardrail violation → FORCE_EXIT with sanitized error
  - Stagnation (3 identical intents) → Kill switch, return last good response
  - Hop limit exceeded (max 10) → FORCE_EXIT, log escalation
  - Kill switch active → interrupt() immediately

3. Tenant Profile: Mid-market retail SaaS platform.

Trigger: Customer submits a ticket via web widget regarding an unreceived package and a requested refund.

Workflow ID: `support-resolution:v2.1.0`

Execution:
  - Orchestrator reads input and Tier 1 summaries. Routes directive to LOAD_SKILL:shipping_api.
  - SkillLoaderNode fetches the full Shopify/FedEx MCP schema.
  - Orchestrator routes to shipping_agent. The subagent queries the API, discovers the package was lost in transit, and summarizes the finding.
  - Orchestrator reads the summary, sets directive LOAD_SKILL:billing_system.
  - billing_agent executes a Stripe refund tool.
  - Orchestrator generates the final response, validated by OutputGuardrailNode.

Edge Case Handled: If Stripe API times out, the CircuitBreakerWatchdogNode prevents an infinite retry loop, triggering a graceful fallback message.
