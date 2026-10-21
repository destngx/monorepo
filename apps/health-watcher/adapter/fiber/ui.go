package fiber

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// The SvelteKit UI (apps/health-watcher/ui) is built with adapter-static under the /ui base path
// and served by this process. API routes stay under /api, so the UI never shadows them.
const (
	UIMountPath       = "/ui"
	DefaultUIBuildDir = "ui/build"
	uiIndexFile       = "index.html"
	uiImmutablePrefix = "/_app/immutable/"
)

// MountUI serves the built SPA at /ui and redirects / to it.
// Client-side routes fall back to index.html; paths with a file extension that do not exist return 404.
// Returns false, registering nothing, when buildDir has no index.html.
func MountUI(app *fiber.App, buildDir string) bool {
	indexPath := filepath.Join(buildDir, uiIndexFile)
	if info, err := os.Stat(indexPath); err != nil || info.IsDir() {
		return false
	}

	redirect := func(c *fiber.Ctx) error {
		// Non-strict routing also matches "/ui/" here; let the SPA handler take it.
		if c.Path() == UIMountPath+"/" {
			return c.Next()
		}
		return c.Redirect(UIMountPath+"/", fiber.StatusFound)
	}
	app.Get("/", redirect)
	app.Get(UIMountPath, redirect)

	app.Get(UIMountPath+"/*", func(c *fiber.Ctx) error {
		// Clean against a rooted path so ".." can never climb out of buildDir.
		rel := path.Clean("/" + c.Params("*"))
		full := filepath.Join(buildDir, filepath.FromSlash(rel))

		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			if strings.HasPrefix(rel, uiImmutablePrefix) {
				c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
			}
			return c.SendFile(full)
		}
		if path.Ext(rel) != "" {
			return fiber.ErrNotFound
		}

		c.Set(fiber.HeaderCacheControl, "no-cache")
		return c.SendFile(indexPath)
	})
	return true
}
