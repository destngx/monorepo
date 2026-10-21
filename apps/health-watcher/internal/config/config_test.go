package config

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestGivenNoEnvironmentWhenLoadingThenDefaultsApply(t *testing.T) {
	c, err := Load(env(map[string]string{"HW_DATA_DIR": "/data/hw"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr() != "127.0.0.1:9191" || c.MaxFileMB != 50 || c.MaxBatchMB != 200 || c.DBPath() != "/data/hw/health.db" || c.FilesDir() != "/data/hw/files" {
		t.Fatalf("config = %+v", c)
	}
}

func TestGivenBadNumbersWhenLoadingThenItFailsNamingTheVariable(t *testing.T) {
	cases := map[string]map[string]string{
		"PORT":            {"PORT": "http"},
		"HW_MAX_FILE_MB":  {"HW_MAX_FILE_MB": "0"},
		"HW_MAX_BATCH_MB": {"HW_MAX_FILE_MB": "100", "HW_MAX_BATCH_MB": "50"},
	}
	for name, vars := range cases {
		vars["HW_DATA_DIR"] = "/d"
		if _, err := Load(env(vars)); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestGivenAnIPv6HostWhenFormattingTheAddressThenItIsBracketed(t *testing.T) {
	c, _ := Load(env(map[string]string{"HOST": "fd7a:115c::1", "HW_DATA_DIR": "/d"}))
	if c.Addr() != "[fd7a:115c::1]:9191" {
		t.Fatalf("Addr = %s", c.Addr())
	}
}

func TestGivenNoAISettingsWhenLoadingThenReadingIsOffAndOCRUsesTheSiblingBuild(t *testing.T) {
	c, err := Load(env(map[string]string{"HW_DATA_DIR": "/data/hw"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.AI.Enabled() || c.AI.Model != "gpt-6-luna" || c.AI.Effort != "low" || c.AI.Timeout.Seconds() != 180 {
		t.Fatalf("ai = %+v", c.AI)
	}
	if !strings.HasSuffix(c.MacOCRBin, "/mac-ocr/bin/mac-ocr") || c.Workers != 2 || c.WorkDir() != "/data/hw/work" {
		t.Fatalf("config = %+v", c)
	}
}

func TestGivenModelSettingsWhenLoadingThenReadingStaysOnGPT6LunaAtLowEffort(t *testing.T) {
	c, err := Load(env(map[string]string{"HW_DATA_DIR": "/d", "AI_BASE_URL": "http://127.0.0.1:8080/v1", "AI_MODEL": "gpt-5.6-luna",
		"AI_MODEL_READ": "gpt-5.6-terra", "AI_REASONING_EFFORT": "medium", "AI_TIMEOUT_SECONDS": "60", "HW_MAC_OCR_BIN": "/opt/mac-ocr"}))
	if err != nil {
		t.Fatal(err)
	}
	if !c.AI.Enabled() || c.AI.Model != "gpt-6-luna" || c.AI.Effort != "low" || c.AI.Timeout.Seconds() != 60 || c.MacOCRBin != "/opt/mac-ocr" {
		t.Fatalf("config = %+v", c)
	}
}

func TestGivenBadAISettingsWhenLoadingThenItFailsNamingTheVariable(t *testing.T) {
	cases := map[string]map[string]string{
		"AI_TIMEOUT_SECONDS": {"AI_TIMEOUT_SECONDS": "0"},
		"AI_BASE_URL":        {"AI_BASE_URL": "localhost:8080"},
		"HW_WORKERS":         {"HW_WORKERS": "99"},
	}
	for name, vars := range cases {
		vars["HW_DATA_DIR"] = "/d"
		if _, err := Load(env(vars)); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
