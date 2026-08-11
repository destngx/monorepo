package anthropic

import (
	"encoding/json"
	"strings"
)

func extractToolResultText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		var result strings.Builder
		for _, item := range v {
			result.WriteString(extractToolResultText(item))
		}
		return result.String()
	case map[string]any:
		if text, ok := v["text"].(string); ok {
			return text
		}
		if content, ok := v["content"]; ok {
			return extractToolResultText(content)
		}
	}
	return ""
}

func normalizeStreamToolArguments(arguments string) (string, bool) {
	var value map[string]any
	if json.Unmarshal([]byte(arguments), &value) != nil {
		return arguments, false
	}
	if pages, ok := value["pages"].(string); ok && pages == "" {
		delete(value, "pages")
		encoded, err := json.Marshal(value)
		if err == nil {
			return string(encoded), true
		}
	}
	return arguments, false
}

func normalizeAnthropicToolInput(name string, input any) any {
	values, ok := input.(map[string]any)
	if !ok || name != "Read" {
		return input
	}
	if pages, exists := values["pages"]; exists {
		if pageString, ok := pages.(string); ok && pageString == "" {
			delete(values, "pages")
		}
	}
	return values
}
