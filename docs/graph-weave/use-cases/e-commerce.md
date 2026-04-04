# Use cases: E-Commerce Customer Support

Tenant: retail SaaS / Shopify-like support desk
Trigger: customer asks about an order and refund
Workflow: `customer-support:v2.1.0`

Execution flow:

- Input guardrail validates the request
- Graph initializer loads the workflow definition and Tier 1 summaries
- Orchestrator routes first to shipping/order lookup, then to billing/refund
- Skill loader fetches the full MCP schema only for the selected skill
- Subagents execute in isolation and return summaries only
- Output guardrail checks the final response for PII and policy violations
- SSE streams the final answer back to the client

Example path:

- Order lookup agent queries shipping status
- Billing agent issues the refund if the package is lost or policy allows it
- The orchestrator folds both summaries into one customer-facing answer

Edge cases:

- Refund loop -> stagnation detection exits with the last good answer after repeated intent
- Malicious input -> guardrail block
- Stripe or FedEx timeout -> circuit breaker fallback and safe retry handling
- Token overflow -> forced exit
