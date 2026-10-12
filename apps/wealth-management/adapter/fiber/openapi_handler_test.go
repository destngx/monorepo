package fiber

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestGivenOpenAPISpecRouteWhenRequestedThenReturnsJSONSpec(t *testing.T) {
	app := fiber.New()
	api := NewRouteRegistrar(app, "/api", "")
	cache := api.Group("/cache", "cache")
	cache.Get("/get/:key", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) }, RouteMeta{
		Summary: "GET /api/cache/get/{key}",
		Tags:    []string{"cache"},
		PathParams: []RouteParameter{
			PathParam("key", "string", "Cache entry key."),
		},
	})
	cache.Delete("/invalidate", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) }, RouteMeta{
		Summary: "DELETE /api/cache/invalidate",
		Tags:    []string{"cache"},
		QueryParams: []RouteParameter{
			QueryParam("pattern", "string", "Cache key pattern to invalidate.", "*"),
		},
	})
	api.Get("/market/ticker", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) }, RouteMeta{
		Summary: "GET /api/market/ticker",
		Tags:    []string{"market"},
		QueryParams: []RouteParameter{
			QueryParam("symbol", "string", "Ticker symbol to fetch."),
			QueryParam("type", "string", "Ticker type."),
		},
	})
	api.Get("/market/exchange-rate", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) }, RouteMeta{
		Summary: "GET /api/market/exchange-rate",
		Tags:    []string{"market"},
		QueryParams: []RouteParameter{
			QueryParam("skipCache", "boolean", "Bypass market cache and fetch fresh provider data.", false),
			QueryParam("from", "string", "Source currency code."),
			QueryParam("to", "string", "Target currency code."),
		},
	})
	app.Get("/api/docs", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	handler := NewOpenAPIHandler(app)
	app.Get("/api/openapi.json", handler.Spec)

	request := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	contentType := response.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected json content type, got %s", contentType)
	}

	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), `"openapi":"3.0.3"`) {
		t.Fatalf("expected openapi version in spec")
	}
	if !strings.Contains(string(body), "/api/market/ticker") {
		t.Fatalf("expected market ticker path in spec")
	}
	if !strings.Contains(string(body), "/api/cache/get/{key}") {
		t.Fatalf("expected cache path parameter route in spec")
	}
	if !strings.Contains(string(body), `"tags":["market"]`) {
		t.Fatalf("expected market route tag to be market, got body=%s", string(body))
	}
	if !strings.Contains(string(body), `"default":false`) {
		t.Fatalf("expected boolean default value in spec, got body=%s", string(body))
	}
	if !strings.Contains(string(body), `"name":"key"`) || !strings.Contains(string(body), `"in":"path"`) {
		t.Fatalf("expected key path parameter in spec, got body=%s", string(body))
	}
	if !strings.Contains(string(body), `"name":"skipCache"`) {
		t.Fatalf("expected skipCache query parameter in spec, got body=%s", string(body))
	}
	if !strings.Contains(string(body), `"default":"*"`) {
		t.Fatalf("expected string default value in spec, got body=%s", string(body))
	}
	if strings.Contains(string(body), "/api/docs") {
		t.Fatalf("expected /api/docs to be excluded from openapi spec, got body=%s", string(body))
	}
	if !strings.Contains(string(body), `"url":"http://example.com"`) {
		t.Fatalf("expected dynamic server url from request host, got body=%s", string(body))
	}
}

func TestGivenOpenAPIDocsRouteWhenRequestedThenReturnsSwaggerUIHTML(t *testing.T) {
	app := fiber.New()
	handler := NewOpenAPIHandler(app)
	app.Get("/api/docs", handler.Docs)

	request := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	body, _ := io.ReadAll(response.Body)
	html := string(body)
	if !strings.Contains(html, "SwaggerUIBundle") {
		t.Fatalf("expected SwaggerUIBundle script in docs html")
	}
	if !strings.Contains(html, "/api/openapi.json") {
		t.Fatalf("expected docs to point to /api/openapi.json")
	}
}

func TestGivenForwardedProtoWhenOpenAPISpecRequestedThenUsesForwardedScheme(t *testing.T) {
	app := fiber.New()
	handler := NewOpenAPIHandler(app)
	app.Get("/api/openapi.json", handler.Spec)

	request := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	request.Host = "api.example.com"
	request.Header.Set("X-Forwarded-Proto", "https")
	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("fiber test request failed: %v", err)
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), `"url":"https://api.example.com"`) {
		t.Fatalf("expected forwarded proto aware server url, got body=%s", string(body))
	}
}
