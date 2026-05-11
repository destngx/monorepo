package github_copilot

import "strings"

func sanitizeParameters(v any) any {
	return sanitizeNode(v, true)
}

func sanitizeNode(v any, normalizeSelf bool) any {
	switch node := v.(type) {
	case map[string]any:
		if !normalizeSelf {
			out := make(map[string]any, len(node))
			for key, value := range node {
				if value == nil {
					continue
				}
				out[key] = sanitizeNode(value, true)
			}
			return out
		}

		out := make(map[string]any, len(node))
		for key, value := range node {
			if value == nil {
				continue
			}
			if shouldDropKey(key) {
				continue
			}
			out[key] = sanitizeNode(value, !isSchemaContainerKey(key))
		}
		normalizeSchemaMap(out)
		return out
	case []any:
		out := make([]any, 0, len(node))
		for _, value := range node {
			if value == nil {
				continue
			}
			out = append(out, sanitizeNode(value, true))
		}
		return out
	default:
		return v
	}
}

func shouldDropKey(key string) bool {
	switch key {
	case "$schema", "$id", "default":
		return true
	default:
		return false
	}
}

func isSchemaContainerKey(key string) bool {
	switch key {
	case "properties", "patternProperties", "$defs", "definitions", "dependentSchemas":
		return true
	default:
		return false
	}
}

func normalizeSchemaMap(schema map[string]any) {
	if typeValue, ok := schema["type"].(string); ok {
		if strings.EqualFold(typeValue, "none") {
			schema["type"] = "object"
		}
		return
	}

	switch {
	case hasKey(schema, "properties"),
		hasKey(schema, "required"),
		hasKey(schema, "additionalProperties"),
		hasKey(schema, "propertyNames"):
		schema["type"] = "object"
	case hasKey(schema, "items"):
		schema["type"] = "array"
	}
}

func hasKey(schema map[string]any, key string) bool {
	_, ok := schema[key]
	return ok
}
