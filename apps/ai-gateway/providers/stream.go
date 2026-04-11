package providers

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"apps/ai-gateway/types"
)

// streamSSEAndCountTokens pipes SSE bytes to w and returns token usage.
// It handles both providers that embed usage in the last chunk (OpenAI-style)
// and those that return usage only on [DONE].
func streamSSEAndCountTokens(body io.Reader, w io.Writer) (types.Usage, error) {
	var usage types.Usage
	var completionTokens int

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	for scanner.Scan() {
		line := scanner.Text()

		// Forward every line to the client as-is (zero-copy passthrough)
		if _, err := io.WriteString(w, line+"\n"); err != nil {
			return usage, err
		}

		// Flush if the writer supports it (http.ResponseWriter with Flusher)
		if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}

		// Parse only data lines for token extraction
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		// Parse the chunk to extract usage and count completion tokens
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *types.Usage `json:"usage,omitempty"`
		}

		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}

		// Accumulate completion tokens from delta content length (approximate)
		if len(chunk.Choices) > 0 {
			completionTokens += estimateTokens(chunk.Choices[0].Delta.Content)
		}

		// Use exact usage if provider embeds it in stream (OpenAI does this on last chunk)
		if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
			usage = *chunk.Usage
		}
	}

	// If provider didn't embed usage, use our estimate
	if usage.TotalTokens == 0 {
		usage.CompletionTokens = completionTokens
		// PromptTokens are estimated by callers or left at 0 for stream mode
	}

	return usage, scanner.Err()
}

// estimateTokens provides a rough ~4 chars/token estimate when exact counts
// are unavailable (e.g., streaming providers that omit usage in chunks).
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// Approximate: 1 token ≈ 4 characters (English text average)
	count := len([]rune(text)) / 4
	if count == 0 {
		return 1
	}
	return count
}

// injectUsageChunk appends a synthetic SSE usage chunk after [DONE].
// Used when the upstream omits usage data.
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

// MergeUsage combines two Usage structs (for multi-call aggregation).
func MergeUsage(a, b types.Usage) types.Usage {
	return types.Usage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		TotalTokens:      a.TotalTokens + b.TotalTokens,
	}
}
