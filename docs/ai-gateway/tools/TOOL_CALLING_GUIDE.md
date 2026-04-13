# Tool Calling Guide

Tool Calling (also known as Function Calling) allows the LLM to request the execution of custom logic—such as database queries, web searches, or calculation services. The AI Gateway provides a standard, OpenAI-compatible proxy that enables tool calling across multiple providers (GitHub, OpenAI, Anthropic, Ollama).

## Architecture

The AI Gateway operates in **Proxy Mode (Stateless)**. This means the Gateway handles the communication and translation, while your application retains full control over the execution of the tools.

### System Architecture

The following diagram illustrates how the AI Gateway sits between the orchestrator (e.g., Wealth Engine) and the AI providers during a tool calling operation.

[View System Architecture](diagrams/system-architecture.md)

### Sequence Flow

Tool calling is a multi-step "ping-pong" interaction. The LLM identifies the need for a tool, the client executes it, and the results are sent back for a final answer.

[View Sequence Flow](diagrams/sequence-flow.md)

---

## Technical Details

### 1. Request Format

Send tool definitions in the standard OpenAI `tools` array within your chat request.

```json
{
  "model": "gpt-4o",
  "messages": [{ "role": "user", "content": "What is the stock price of AAPL?" }],
  "tools": [
    {
      "type": "function",
      "function": {
        "name": "get_stock_price",
        "parameters": {
          "type": "object",
          "properties": { "symbol": { "type": "string" } }
        }
      }
    }
  ]
}
```

### 2. cURL Example (The Tool Loop)

Tool calling requires a two-step "loop". Here is how to perform it using `curl`.

#### Step 1: Request the Tool Call

The client asks a question and provides the tool definition.

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-AI-Provider: github" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "What is the stock price of AAPL?"}],
    "tools": [{
      "type": "function",
      "function": {
        "name": "get_stock_price",
        "parameters": {
          "type": "object",
          "properties": {"symbol": {"type": "string"}}
        }
      }
    }]
  }'
```

#### Step 2: Submit the Tool Result

After your app executes the logic (e.g., fetching a stock price), send the result back to get the final answer. Note the `role: "tool"` and the `tool_call_id`.

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "X-AI-Provider: github" \
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "What is the stock price of AAPL?"},
      {
        "role": "assistant",
        "tool_calls": [{
          "id": "call_123",
          "type": "function",
          "function": {"name": "get_stock_price", "arguments": "{\"symbol\":\"AAPL\"}"}
        }]
      },
      {
        "role": "tool",
        "tool_call_id": "call_123",
        "content": "AAPL is currently trading at $185.20"
      }
    ]
  }'
```

---

### 3. Provider Translation (Anthropic & GitHub Copilot)

The AI Gateway performs deep schema normalization when mapping between Anthropic and OpenAI/GitHub Copilot formats:

- **Recursive Schema Sanitization (Gemini/Copilot)**: Providers like Gemini and GitHub Copilot are extremely strict about JSON Schema compliance. The Gateway recursively traverses every schema node to strip unsupported keys like `$schema`, `$id`, `default`, and **`additionalProperties`**.
- **Intelligent Type Injection (De-Pollution)**: To prevent schema validation errors, the Gateway only adds `"type": "object"` or `"type": "array"` to nodes that contain structural indicators. It specifically avoids "polluting" container keys like `properties` with extraneous types.
- **Tool Choice Mapping**:
  - Anthropic's forced tool use (`"type": "any"`) is automatically translated to the OpenAI/Copilot equivalent (`"required"`).
  - Specific named tool selection (`"type": "tool"`) is mapped to the OpenAI function selection format.
  - This ensures that clients like the **Claude Code CLI** work seamlessly even when forcing specific tool selection.
- **Role Translation**: Maps OpenAI `assistant` (with tool calls) and `tool` (results) roles to Anthropic's compliant message blocks.
- **Streaming Fragments**: Translates Anthropic's streaming tool delta events into standard OpenAI-compatible chunks, ensuring real-time UI updates for tool arguments.

### 3. Integration with Orchestrators

This gateway is optimized for use with stateful orchestrators like **LangGraph** or **Graph-weave**.

- **Statefulness**: Orchestrators maintain the chat history and checkpointer (e.g., in Redis).
- **Control**: By using the Gateway in Proxy Mode, the orchestrator has full visibility and control over tool execution results.

---

## Best Practices

1. **Descriptions Matter**: The LLM relies on the `description` field to decide _when_ to call a tool. Be specific.
2. **Schema Validation**: Ensure your `parameters` follow the JSON Schema standard strictly.
3. **Stateless Gateway**: Do not rely on the Gateway to maintain session state; always pass the full message history in each request.
