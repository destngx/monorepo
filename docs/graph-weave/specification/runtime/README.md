# Runtime Living Spec

Use these docs for execution flow, control loops, guardrails, and Redis-backed runtime contracts.

## AI Integration

AI interactions are strictly routed through the `ai-gateway` proxy. This ensures provider abstraction, unified tool-calling translation, and centralized observability.

- **Primary Guide**: `[[../../ai-gateway/API_GUIDE]]`
- **Tool Calling**: `[[../../ai-gateway/tools/TOOL_CALLING_GUIDE]]`

See also: `[[plan/README]]`, `[[tasks/README]]`, `[[verification/README]]`.
