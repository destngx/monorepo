package fiber

import (
	"strings"
)

func toTagFromPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "default"
	}

	segments := strings.Split(trimmed, "/")
	if len(segments) >= 2 && segments[0] == "api" && segments[1] != "" {
		tag := segments[1]
		if tag == "external" && len(segments) >= 3 && segments[2] != "" {
			return segments[2]
		}
		return tag
	}
	if segments[0] != "" {
		return segments[0]
	}
	return "default"
}

func shouldExcludeFromOpenAPI(path string) bool {
	return path == "/api/docs" || path == "/api/openapi.json"
}

func toOpenAPIPath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			param := strings.TrimPrefix(segment, ":")
			segments[i] = "{" + param + "}"
			continue
		}
		if strings.HasPrefix(segment, "*") {
			param := strings.TrimPrefix(segment, "*")
			if param == "" {
				param = "wildcard"
			}
			segments[i] = "{" + param + "}"
		}
	}
	return strings.Join(segments, "/")
}

func toPathParameters(params []string) []map[string]any {
	parameters := make([]map[string]any, 0, len(params))
	for _, param := range params {
		parameters = append(parameters, map[string]any{
			"name":     param,
			"in":       "path",
			"required": true,
			"schema": map[string]any{
				"type": "string",
			},
		})
	}
	return parameters
}

func toOperationID(method string, path string) string {
	normalized := strings.ToLower(method + "_" + path)
	replacer := strings.NewReplacer("/", "_", "{", "", "}", "", ":", "", "-", "_")
	normalized = replacer.Replace(normalized)
	normalized = strings.Trim(normalized, "_")
	return normalized
}
