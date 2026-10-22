package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestSwaggerDocsUpToDate regenerates the spec into a temp dir and fails when
// the committed docs/ differ, so a new or changed endpoint annotation cannot
// ship without its spec. Fix a failure with `pnpx nx run ai-gateway:swagger`.
func TestSwaggerDocsUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("runs swag init")
	}

	// The output dir must be named "docs": swag derives the package name of docs.go from it.
	outDir := filepath.Join(t.TempDir(), "docs")
	cmd := exec.Command("go", "run", "github.com/swaggo/swag/cmd/swag", "init", "--quiet", "--output", outDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("swag init failed: %v\n%s", err, out)
	}

	for _, name := range []string{"docs.go", "swagger.json", "swagger.yaml"} {
		want, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatalf("read generated %s: %v", name, err)
		}
		got, err := os.ReadFile(filepath.Join("docs", name))
		if err != nil {
			t.Fatalf("read committed %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("docs/%s is stale; run `pnpx nx run ai-gateway:swagger` and commit the result", name)
		}
	}
}
