package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"apps/health-watcher/domain"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestGivenNoDataDirWhenLoadingThenApplicationSupportIsUsed(t *testing.T) {
	c, err := Load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(c.DataDir) != "health-watcher" || c.DBPath() != filepath.Join(c.DataDir, "health.db") ||
		c.FilesDir() != filepath.Join(c.DataDir, "files") || c.WorkDir() != filepath.Join(c.DataDir, "work") {
		t.Fatalf("config = %+v", c)
	}
}

func TestGivenADataDirWhenLoadingThenItIsUsed(t *testing.T) {
	c, err := Load(env(map[string]string{"HW_DATA_DIR": " /tmp/hw-test "}))
	if err != nil || c.DataDir != "/tmp/hw-test" || c.SetupCodePath() != "/tmp/hw-test/setup-code" {
		t.Fatalf("config = %+v, %v", c, err)
	}
}

func TestGivenRetiredVariablesWhenCheckingThenTheSetOnesAreListed(t *testing.T) {
	got := Ignored(env(map[string]string{"AI_BASE_URL": "http://localhost:8080/v1", "PORT": "9292", "HW_DATA_DIR": "/x", "AI_MODEL": " "}))
	if !reflect.DeepEqual(got, []string{"PORT", "AI_BASE_URL"}) {
		t.Fatalf("ignored = %v", got)
	}
}

func TestGivenRetiredVariablesWhenLoadingThenTheyBecomeStoredSettingsToImport(t *testing.T) {
	c, err := Load(env(map[string]string{"AI_BASE_URL": "http://localhost:8080/v1", "AI_API_KEY": "sk-test", "HW_WORKERS": "3",
		"HW_MAC_OCR_BIN": "bin/mac-ocr", "AI_MODEL": "other"}))
	if err != nil {
		t.Fatal(err)
	}
	s, rejected := domain.SettingsFromValues(c.Legacy)
	if rejected != nil || s.AIBaseURL != "http://localhost:8080/v1" || s.AIAPIKey != "sk-test" || s.Workers != 3 ||
		!filepath.IsAbs(s.MacOCRBin) || len(c.Legacy) != 4 {
		t.Fatalf("legacy = %v, settings = %+v, rejected = %v", c.Legacy, s, rejected)
	}
}

func TestGivenTheMonorepoLayoutWhenFindingMacOCRThenTheSiblingBuildIsUsed(t *testing.T) {
	p, err := DefaultMacOCRBin()
	if err != nil || !filepath.IsAbs(p) || !strings.HasSuffix(p, filepath.Join("mac-ocr", "bin", "mac-ocr")) {
		t.Fatalf("path = %q, %v", p, err)
	}
}
