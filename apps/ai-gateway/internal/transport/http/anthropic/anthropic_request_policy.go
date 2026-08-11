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

func detectUnsupportedNativeTools(req anthropic.Request, providerName string) (bool, string) {
	if providerName == domain.ProviderAnthropic {
		return false, ""
	}

	validFunctionToolNames := make(map[string]bool)

	for _, t := range req.Tools {
		toolType := strings.ToLower(t.Type)
		if toolType == "" || toolType == domain.ToolTypeFunction {
			validFunctionToolNames[t.Name] = true
			continue
		}
		return true, fmt.Sprintf("non-function tool type '%s' is not supported", t.Type)
	}

	if req.ToolChoice != nil {
		if m, ok := req.ToolChoice.(map[string]any); ok {
			if t, ok := m["type"].(string); ok && t == "tool" {
				if name, ok := m["name"].(string); ok && !validFunctionToolNames[name] {
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
					if bType != "text" && bType != "tool_use" && bType != "tool_result" {
						return true, fmt.Sprintf("native message content block type '%s' is not supported", bType)
					}
				}
			}
		}
	}

	return false, ""
}
