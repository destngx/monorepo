package anthropic

import (
	"fmt"
	"strings"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/anthropic"
)

func normalizeRequestForRoute(req *domain.ChatRequest, anthroReq anthropic.Request, route AnthropicRoute) {
	if route.Provider.Name() != domain.ProviderOpenAI {
		return
	}

	req.MaxTokens = nil
	req.MaxCompletionTokens = nil
}

// isWebSearchTool identifies Anthropic's versioned web-search tool names
// without coupling routing to a specific dated API revision.
func isWebSearchTool(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "web_search" || strings.HasPrefix(normalized, "web_search_")
}

func detectUnsupportedNativeTools(req anthropic.Request, providerName string) (bool, string) {
	if providerName == domain.ProviderAnthropic {
		return false, ""
	}

	validFunctionToolNames := make(map[string]bool)

	for _, t := range req.Tools {
		toolType := strings.ToLower(t.Type)
		if isWebSearchTool(toolType) || isWebSearchTool(t.Name) {
			// Temporary compatibility path: allow Claude web-search tools through
			// for sidecar/OpenAI web-search testing.
			continue
		}
		if toolType == "" || toolType == domain.ToolTypeFunction {
			validFunctionToolNames[t.Name] = true
			continue
		}
		return true, fmt.Sprintf("non-function tool type '%s' is not supported", t.Type)
	}

	if req.ToolChoice != nil {
		if m, ok := req.ToolChoice.(map[string]any); ok {
			if t, ok := m["type"].(string); ok && t == "tool" {
				if name, ok := m["name"].(string); ok && !isWebSearchTool(name) && !validFunctionToolNames[name] {
					return true, fmt.Sprintf("tool_choice targets unmapped or native tool '%s'", name)
				}
			}
		}
	}

	for _, msg := range req.Messages {
		if blocks, ok := msg.Content.([]any); ok {
			for _, block := range blocks {
				if b, ok := block.(map[string]any); ok {
					bType, _ := b["type"].(string)
					blockName, _ := b["name"].(string)
					if isWebSearchTool(bType) || isWebSearchTool(blockName) {
						continue
					}
					if bType != "text" && bType != "tool_use" && bType != "tool_result" {
						return true, fmt.Sprintf("native message content block type '%s' is not supported", bType)
					}
				}
			}
		}
	}

	return false, ""
}
