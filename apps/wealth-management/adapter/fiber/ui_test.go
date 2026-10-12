package fiber

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
)

const testIndexHTML = "<!doctype html><title>wm</title>"

func writeUIBuild(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"index.html":                  testIndexHTML,
		"robots.txt":                  "User-agent: *",
		"_app/immutable/entry/app.js": "console.log('app')",
	}
	for name, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

func newUIApp(t *testing.T, buildDir string) (*fiber.App, bool) {
	t.Helper()
	app := fiber.New()
	app.Get("/api/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
	mounted := MountUI(app, buildDir)
	return app, mounted
}

func doGet(t *testing.T, app *fiber.App, target string) (int, string, map[string]string) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", target, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	headers := map[string]string{
		"Location":      resp.Header.Get("Location"),
		"Cache-Control": resp.Header.Get("Cache-Control"),
	}
	return resp.StatusCode, string(body), headers
}

func TestMountUIRedirectsRootToUI(t *testing.T) {
	app, mounted := newUIApp(t, writeUIBuild(t))
	if !mounted {
		t.Fatal("expected UI to be mounted")
	}
	for _, target := range []string{"/", "/ui"} {
		status, _, headers := doGet(t, app, target)
		if status != fiber.StatusFound || headers["Location"] != "/ui/" {
			t.Fatalf("GET %s: expected 302 to /ui/, got %d %q", target, status, headers["Location"])
		}
	}
}

func TestMountUIServesIndexAndClientRoutes(t *testing.T) {
	app, _ := newUIApp(t, writeUIBuild(t))
	for _, target := range []string{"/ui/", "/ui/transactions", "/ui/accounts/credit-cards/visa"} {
		status, body, headers := doGet(t, app, target)
		if status != fiber.StatusOK || body != testIndexHTML {
			t.Fatalf("GET %s: expected SPA shell, got %d %q", target, status, body)
		}
		if headers["Cache-Control"] != "no-cache" {
			t.Fatalf("GET %s: expected no-cache on SPA shell, got %q", target, headers["Cache-Control"])
		}
	}
}

func TestMountUIServesStaticAssets(t *testing.T) {
	app, _ := newUIApp(t, writeUIBuild(t))

	status, body, _ := doGet(t, app, "/ui/robots.txt")
	if status != fiber.StatusOK || body != "User-agent: *" {
		t.Fatalf("expected robots.txt, got %d %q", status, body)
	}

	status, _, headers := doGet(t, app, "/ui/_app/immutable/entry/app.js")
	if status != fiber.StatusOK {
		t.Fatalf("expected immutable asset, got %d", status)
	}
	if headers["Cache-Control"] != "public, max-age=31536000, immutable" {
		t.Fatalf("expected long-lived cache on immutable asset, got %q", headers["Cache-Control"])
	}
}

func TestMountUIReturns404ForMissingAssets(t *testing.T) {
	app, _ := newUIApp(t, writeUIBuild(t))
	status, _, _ := doGet(t, app, "/ui/_app/immutable/missing.js")
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 for missing asset, got %d", status)
	}
}

func TestMountUIDoesNotEscapeBuildDir(t *testing.T) {
	buildDir := writeUIBuild(t)
	secret := filepath.Join(filepath.Dir(buildDir), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	app, _ := newUIApp(t, buildDir)
	for _, target := range []string{"/ui/../secret.txt", "/ui/%2e%2e/secret.txt"} {
		_, body, _ := doGet(t, app, target)
		if body == "secret" {
			t.Fatalf("GET %s leaked a file outside the build dir", target)
		}
	}
}

func TestMountUIKeepsAPIRoutes(t *testing.T) {
	app, _ := newUIApp(t, writeUIBuild(t))
	status, body, _ := doGet(t, app, "/api/health")
	if status != fiber.StatusOK || body != "ok" {
		t.Fatalf("expected API route to win, got %d %q", status, body)
	}
}

func TestMountUISkipsWhenBuildMissing(t *testing.T) {
	app, mounted := newUIApp(t, t.TempDir())
	if mounted {
		t.Fatal("expected MountUI to report false without index.html")
	}
	status, _, _ := doGet(t, app, "/ui/")
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 when UI is not built, got %d", status)
	}
}
