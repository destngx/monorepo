# Use cases: E-Commerce Customer Support

## 1. Objective

- What: Resolve order and refund requests through a deterministic support workflow.
- Why: Give the support desk a repeatable path from customer question to safe resolution.
- Who: Retail SaaS support operations.

## 2. Scope

- In scope: order status lookup, refund decisions, guarded responses, and streamed replies.
- Out of scope: manual agent routing and provider-specific implementation details.

## 3. Specification

- Input requests must pass through the guardrail before routing.
- The orchestrator must decide between shipping/order lookup and billing/refund paths.
- Subagents must return summarized results only.
- Output must be redacted for PII and policy violations before delivery.

## 4. Technical Plan

- Load Tier 1 summaries first, then fetch Tier 2 MCP schema only for the selected branch.
- Keep shipping/order and billing/refund work isolated in subagents.
- Stream the final answer over SSE once guardrails pass.

## 5. Tasks

- [ ] Validate customer input and route to the correct support branch.
- [ ] Execute shipping and billing subagent flows in isolation.
- [ ] Apply output guardrails before streaming the response.

## 6. Verification

- Given a valid order/refund request, when it is processed, then the workflow must return a safe customer answer.
- Given malicious input, when it reaches the guardrail, then the request must be blocked.
- Given a refund loop or timeout, when thresholds are exceeded, then the workflow must stop safely.

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
