# Use cases: E-Commerce Customer Support

## 1. Objective

- What: Resolve order and refund requests through a deterministic support workflow.
- Why: Give the support desk a repeatable path from customer question to safe resolution.
- Who: Retail SaaS support operations.

## Traceability

- UC-ECOM-001: The support workflow must validate customer input before routing.
- UC-ECOM-002: The workflow must support order lookup, refund handling, and safe fallback.
- UC-ECOM-003: The response must be redacted and streamed after guardrails pass.

## 2. Scope

- In scope: order status lookup, refund decisions, guarded responses, and streamed replies.
- Out of scope: manual agent routing and provider-specific implementation details.

## 2.1 Success Definition

- A customer gets a safe, consistent answer without exposing private data or creating a manual support loop.

## 3. Specification

- Input requests must pass through the guardrail before routing.
- The orchestrator must decide between shipping/order lookup and billing/refund paths.
- Subagents must return summarized results only.
- Output must be redacted for PII and policy violations before delivery.
- The runtime contract must include concrete client-facing endpoints for support requests and streaming.
- NFR: the workflow should return within a user-visible latency budget suitable for support interactions.

## 4. Technical Plan

- Load Level 1 frontmatter first, then fetch the Level 2 body only for the selected branch and open Level 3 linked files only if needed.
- Keep shipping/order and billing/refund work isolated in agent nodes.
- Stream the final answer over SSE once guardrails pass.

## 5. Tasks

- [ ] Validate customer input and route to the correct support branch.
- [ ] Execute shipping and billing subagent flows in isolation.
- [ ] Apply output guardrails before streaming the response.
- [ ] Record test cases for happy-path refund, missing order, malicious input, and timeout fallback.

## 6. Verification

- Given a valid order/refund request, when it is processed, then the workflow must return a safe customer answer.
- Given malicious input, when it reaches the guardrail, then the request must be blocked.
- Given a refund loop or timeout, when thresholds are exceeded, then the workflow must stop safely.
- Given a normal request, when the flow completes, then it must satisfy the documented latency and safety constraints.

Tenant: retail SaaS / Shopify-like support desk
Trigger: customer asks about an order and refund
Workflow: `customer-support:v2.1.0`

Execution flow:

- Input guardrail validates the request
- Graph initializer loads the workflow definition and Level 1 frontmatter
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
