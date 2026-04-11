package providers

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"apps/ai-gateway/types"
)

// streamSSEAndCountTokens proxies Server-Sent Events (SSE) from a provider to a client
// while simultaneously tracking and estimating token usage.
//
// Modern AI providers vary in how they report usage in streams:
// 1. OpenAI/GitHub: Embed a usage object in the final data chunk.
// 2. Others: Only signal completion with [DONE], providing no usage data.
//
// This function handles both cases by parsing incoming chunks for native usage data
// and falling back to a character-based estimation if none is provided.
func streamSSEAndCountTokens(body io.Reader, w io.Writer) (types.Usage, error) {
	var usage types.Usage
	var completionTokens int

	scanner := bufio.NewScanner(body)
	// Use a 64KB buffer to handle large JSON payloads in a single line
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	for scanner.Scan() {
		line := scanner.Text()

		// Pass through non-data lines (comments, heartbeats) directly
		if !strings.HasPrefix(line, "data: ") {
			if _, err := io.WriteString(w, line+"\n"); err != nil {
				return usage, err
			}
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		// Forward the raw data line to the client immediately for low latency
		if _, err := io.WriteString(w, line+"\n"); err != nil {
			return usage, err
		}

		// Flush ensure the client receives the chunk without buffering delays
		if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}

		// Background: Parse the chunk to extract usage and count completion tokens
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string           `json:"content"`
					ToolCalls []types.ToolCall `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *types.Usage `json:"usage,omitempty"`
		}

		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Ignore non-standard JSON or malformed chunks during count
			continue
		}

		// Track generated content and tool arguments for token estimation
		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			completionTokens += estimateTokens(delta.Content)
			for _, tc := range delta.ToolCalls {
				completionTokens += estimateTokens(tc.Function.Arguments)
			}
		}

		// Capture native usage data if the provider includes it (e.g. OpenAI)
		if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
			usage = *chunk.Usage
		}
	}

	// Finalization: Ensure usage data is always sent to the client
	if usage.TotalTokens == 0 {
		// Fallback to estimation for providers like Ollama or Anthropic
		usage.CompletionTokens = completionTokens
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		injectUsageChunk(w, usage)
	} else {
		// Signal [DONE] if it wasn't already handled by the usage chunk
		io.WriteString(w, "data: [DONE]\n\n")
		if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}
	}

	return usage, scanner.Err()
}

// estimateTokens provides a conservative token count estimate (~4 chars per token).
// This is used as a fallback when the upstream provider does not provide usage stats.
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	count := len([]rune(text)) / 4
	if count == 0 {
		return 1
	}
	return count
}

// injectUsageChunk appends a synthetic OpenAI-compatible usage chunk to the stream.
func injectUsageChunk(w io.Writer, usage types.Usage) {
	chunk := map[string]interface{}{
		"object":  "chat.completion.chunk",
		"usage":   usage,
		"choices": []interface{}{},
	}
	b, _ := json.Marshal(chunk)
	io.WriteString(w, "data: "+string(b)+"\n\n")
	io.WriteString(w, "data: [DONE]\n\n")
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
}

// MergeUsage accumulates usage stats across multiple tool-loop iterations.
func MergeUsage(a, b types.Usage) types.Usage {
	return types.Usage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		TotalTokens:      a.TotalTokens + b.TotalTokens,
	}
}
