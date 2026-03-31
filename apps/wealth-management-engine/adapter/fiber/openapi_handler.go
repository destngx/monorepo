package fiber

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type OpenAPIHandler struct {
	app *fiber.App
}

func NewOpenAPIHandler(app *fiber.App) *OpenAPIHandler {
	return &OpenAPIHandler{app: app}
}

func (h *OpenAPIHandler) Spec(c *fiber.Ctx) error {
	scheme := forwardedProto(c)
	host := c.Hostname()
	serverURL := fmt.Sprintf("%s://%s", scheme, host)
	spec := buildOpenAPISpec(h.app, serverURL)
	return c.JSON(spec)
}

func (h *OpenAPIHandler) Docs(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Wealth Management Engine API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/api/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true
    });
  </script>
</body>
</html>`)
}

func buildOpenAPISpec(app *fiber.App, serverURL string) map[string]any {
	paths := map[string]any{}
	if app != nil {
		routes := app.GetRoutes(true)
		sort.Slice(routes, func(i, j int) bool {
			if routes[i].Path == routes[j].Path {
				return routes[i].Method < routes[j].Method
			}
			return routes[i].Path < routes[j].Path
		})

		for _, route := range routes {
			if !strings.HasPrefix(route.Path, "/api/") {
				continue
			}
			if shouldExcludeFromOpenAPI(route.Path) {
				continue
			}

			method := strings.ToLower(route.Method)
			if method == "head" {
				continue
			}

			path := toOpenAPIPath(route.Path)
			if _, ok := paths[path]; !ok {
				paths[path] = map[string]any{}
			}

			operation := map[string]any{
				"summary":       fmt.Sprintf("%s %s", route.Method, route.Path),
				"operationId":   toOperationID(route.Method, route.Path),
				"tags":          []string{toTagFromPath(route.Path)},
				"responses":     map[string]any{"200": map[string]any{"description": "OK"}},
				"x-handlerName": route.Name,
			}
			if len(route.Params) > 0 {
				operation["parameters"] = toPathParameters(route.Params)
			}

			pathItem := paths[path].(map[string]any)
			pathItem[method] = operation
		}
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Wealth Management Engine API",
			"version":     "0.1.0",
			"description": "Code-generated OpenAPI specification",
		},
		"servers": []map[string]any{
			{"url": serverURL},
		},
		"paths": paths,
	}
}

func toTagFromPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "default"
	}

	segments := strings.Split(trimmed, "/")
	if len(segments) >= 2 && segments[0] == "api" && segments[1] != "" {
		return segments[1]
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

func forwardedProto(c *fiber.Ctx) string {
	if proto := strings.TrimSpace(c.Get("X-Forwarded-Proto")); proto != "" {
		return strings.ToLower(proto)
	}
	return c.Protocol()
}
