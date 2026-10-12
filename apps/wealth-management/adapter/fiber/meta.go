package fiber

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

type RouteParameter struct {
	Name        string
	In          string
	Type        string
	Description string
	Required    bool
	Default     any
}

type RouteMeta struct {
	Summary     string
	Tags        []string
	QueryParams []RouteParameter
	PathParams  []RouteParameter
	RequestBody map[string]any
	Responses   map[string]any
}

type RouteRegistrar struct {
	router     fiber.Router
	base       string
	defaultTag string
}

type routeMetaStore struct {
	sync.RWMutex
	items map[string]RouteMeta
}

var routeMetas = routeMetaStore{items: map[string]RouteMeta{}}

func NewRouteRegistrar(router fiber.Router, basePath string, defaultTag string) RouteRegistrar {
	base := normalizeRoutePath(basePath)
	if base != "" {
		router = router.Group(base)
	}
	return RouteRegistrar{router: router, base: base, defaultTag: defaultTag}
}

func (r RouteRegistrar) Group(path string, tag string) RouteRegistrar {
	fullPath := joinRoutePath(r.base, path)
	grouped := r.router.Group(path)
	if fullPath != "" && fullPath != normalizeRoutePath(path) {
		grouped = r.router.Group(path)
	}
	return RouteRegistrar{router: grouped, base: fullPath, defaultTag: tag}
}

func (r RouteRegistrar) Get(path string, handler fiber.Handler, meta RouteMeta) {
	r.register(fiber.MethodGet, path, handler, meta)
}

func (r RouteRegistrar) Post(path string, handler fiber.Handler, meta RouteMeta) {
	r.register(fiber.MethodPost, path, handler, meta)
}

func (r RouteRegistrar) Patch(path string, handler fiber.Handler, meta RouteMeta) {
	r.register(fiber.MethodPatch, path, handler, meta)
}

func (r RouteRegistrar) Delete(path string, handler fiber.Handler, meta RouteMeta) {
	r.register(fiber.MethodDelete, path, handler, meta)
}

func (r RouteRegistrar) register(method string, path string, handler fiber.Handler, meta RouteMeta) {
	fullPath := joinRoutePath(r.base, path)
	if len(meta.Tags) == 0 && r.defaultTag != "" {
		meta.Tags = []string{r.defaultTag}
	}
	storeRouteMeta(method, fullPath, meta)
	switch method {
	case fiber.MethodGet:
		r.router.Get(path, handler)
	case fiber.MethodPost:
		r.router.Post(path, handler)
	case fiber.MethodPatch:
		r.router.Patch(path, handler)
	case fiber.MethodDelete:
		r.router.Delete(path, handler)
	default:
		panic(fmt.Sprintf("unsupported method %s", method))
	}
}

func QueryParam(name string, parameterType string, description string, defaultValue ...any) RouteParameter {
	param := RouteParameter{Name: name, In: "query", Type: parameterType, Description: description, Required: false}
	if len(defaultValue) > 0 {
		param.Default = defaultValue[0]
	}
	return param
}

func PathParam(name string, parameterType string, description string) RouteParameter {
	return RouteParameter{Name: name, In: "path", Type: parameterType, Description: description, Required: true}
}

func lookupRouteMeta(method string, path string) (RouteMeta, bool) {
	routeMetas.RLock()
	defer routeMetas.RUnlock()
	meta, ok := routeMetas.items[routeMetaKey(method, path)]
	return meta, ok
}

func storeRouteMeta(method string, path string, meta RouteMeta) {
	routeMetas.Lock()
	defer routeMetas.Unlock()
	routeMetas.items[routeMetaKey(method, path)] = meta
}

func routeMetaKey(method string, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + ":" + normalizeRoutePath(path)
}

func normalizeRoutePath(path string) string {
	if path == "" {
		return ""
	}
	if path == "/" {
		return "/"
	}
	trimmed := strings.TrimSpace(path)
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func joinRoutePath(base string, path string) string {
	base = normalizeRoutePath(base)
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if base == "" {
		return normalizeRoutePath(path)
	}
	return normalizeRoutePath(base + path)
}
