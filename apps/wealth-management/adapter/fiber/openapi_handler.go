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
				"summary":       fmt.Sprintf("%s %s", route.Method, toOpenAPIPath(route.Path)),
				"operationId":   toOperationID(route.Method, route.Path),
				"tags":          []string{toTagFromPath(route.Path)},
				"responses":     responsesForRoute(route.Method, route.Path),
				"x-handlerName": route.Name,
			}
			if meta, ok := lookupRouteMeta(route.Method, route.Path); ok {
				if meta.Summary != "" {
					operation["summary"] = meta.Summary
				}
				if len(meta.Tags) > 0 {
					operation["tags"] = meta.Tags
				}
				if len(meta.PathParams) > 0 || len(meta.QueryParams) > 0 {
					operation["parameters"] = append(toParameters(meta.PathParams), toParameters(meta.QueryParams)...)
				}
				if meta.RequestBody != nil {
					operation["requestBody"] = meta.RequestBody
				}
				if meta.Responses != nil {
					operation["responses"] = meta.Responses
				}
			} else {
				if requestBody := requestBodyForRoute(route.Method, route.Path); requestBody != nil {
					operation["requestBody"] = requestBody
				}
				if queryParams := queryParametersForRoute(route.Path); len(queryParams) > 0 {
					if existing, ok := operation["parameters"].([]map[string]any); ok {
						operation["parameters"] = append(existing, queryParams...)
					} else {
						operation["parameters"] = queryParams
					}
				}
				if len(route.Params) > 0 {
					pathParams := toPathParameters(route.Params)
					if existing, ok := operation["parameters"].([]map[string]any); ok {
						operation["parameters"] = append(existing, pathParams...)
					} else {
						operation["parameters"] = pathParams
					}
				}
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

func queryParametersForRoute(path string) []map[string]any {
	switch path {
	case "/api/accounts", "/api/transactions", "/api/categories", "/api/goals", "/api/notifications", "/api/investments/assets":
		return []map[string]any{queryParameter("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.")}
	default:
		return nil
	}
}

func requestBodyForRoute(method string, path string) map[string]any {
	switch {
	case method == fiber.MethodPost && path == "/api/transactions":
		return jsonRequestBody(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"accountName":     map[string]any{"type": "string"},
				"date":            map[string]any{"type": "string"},
				"payee":           map[string]any{"type": "string"},
				"category":        map[string]any{"type": "string"},
				"tags":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"cleared":         map[string]any{"type": "boolean"},
				"payment":         map[string]any{"type": "number"},
				"deposit":         map[string]any{"type": "number"},
				"memo":            map[string]any{"type": "string"},
				"referenceNumber": map[string]any{"type": "string"},
			},
			"required": []string{"accountName", "date", "payee", "category"},
		})
	case method == fiber.MethodPatch && path == "/api/notifications":
		return jsonRequestBody(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"rowNumbers": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
			},
			"required": []string{"rowNumbers"},
		})
	default:
		return nil
	}
}

func responsesForRoute(method string, path string) map[string]any {
	responses := map[string]any{"200": map[string]any{"description": "OK"}}
	if method == fiber.MethodPost && path == "/api/transactions" {
		responses["400"] = map[string]any{"description": "Invalid transaction payload"}
	}
	if method == fiber.MethodPatch && path == "/api/notifications" {
		responses["400"] = map[string]any{"description": "Invalid notification rowNumbers payload"}
	}
	if strings.HasPrefix(path, "/api/") && path != "/api/health" {
		responses["503"] = map[string]any{"description": "Google Sheets configuration unavailable"}
	}
	return responses
}

func queryParameter(name string, parameterType string, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "query",
		"required":    false,
		"description": description,
		"schema":      map[string]any{"type": parameterType},
	}
}

func toParameters(params []RouteParameter) []map[string]any {
	parameters := make([]map[string]any, 0, len(params))
	for _, param := range params {
		parameter := map[string]any{
			"name":        param.Name,
			"in":          param.In,
			"required":    param.Required,
			"description": param.Description,
			"schema":      map[string]any{"type": param.Type},
		}
		if param.Default != nil {
			parameter["schema"].(map[string]any)["default"] = param.Default
		}
		parameters = append(parameters, parameter)
	}
	return parameters
}

func jsonRequestBody(schema map[string]any) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
	}
}

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

func forwardedProto(c *fiber.Ctx) string {
	if proto := strings.TrimSpace(c.Get("X-Forwarded-Proto")); proto != "" {
		return strings.ToLower(proto)
	}
	return c.Protocol()
}
