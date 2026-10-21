// Package config finds the engine's data folder. Every other setting lives
// in the database and is edited on the Engine settings page (HW-OPS-9).
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UIBuildDir is the built UI the engine serves, relative to its working
// directory.
const UIBuildDir = "ui/build"

// Config is what the engine needs before it can open the database.
type Config struct {
	DataDir string
	// Legacy are the retired environment variables that are set, under
	// their stored setting keys. The engine imports them once, into an
	// empty settings table.
	Legacy map[string]string
}

// WorkDir is the background reader's private scratch directory.
func (c Config) WorkDir() string { return filepath.Join(c.DataDir, "work") }

// DBPath is the database file inside the data directory (HW-OPS-1).
func (c Config) DBPath() string { return filepath.Join(c.DataDir, "health.db") }

// FilesDir is the file store inside the data directory (HW-OPS-2).
func (c Config) FilesDir() string { return filepath.Join(c.DataDir, "files") }

// SetupCodePath is the first-run setup code file (HW-AUTH-4).
func (c Config) SetupCodePath() string { return filepath.Join(c.DataDir, "setup-code") }

// Load reads HW_DATA_DIR, defaulting to the user's Application Support
// folder.
func Load(getenv func(string) string) (Config, error) {
	c := Config{DataDir: strings.TrimSpace(getenv("HW_DATA_DIR")), Legacy: map[string]string{}}
	for env, key := range legacyKeys {
		if v := strings.TrimSpace(getenv(env)); v != "" {
			c.Legacy[key] = v
		}
	}
	if bin, ok := c.Legacy["mac_ocr_bin"]; ok {
		abs, err := filepath.Abs(bin)
		if err != nil {
			return c, fmt.Errorf("resolve HW_MAC_OCR_BIN: %w", err)
		}
		c.Legacy["mac_ocr_bin"] = abs
	}
	if c.DataDir == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return c, fmt.Errorf("find the data directory (set HW_DATA_DIR): %w", err)
		}
		c.DataDir = filepath.Join(dir, "health-watcher")
	}
	return c, nil
}

// DefaultMacOCRBin is the mac-ocr build next to the engine in the
// monorepo, used while the setting is empty.
func DefaultMacOCRBin() (string, error) {
	p, err := filepath.Abs(filepath.Join("..", "mac-ocr", "bin", "mac-ocr"))
	if err != nil {
		return "", fmt.Errorf("resolve the mac-ocr path: %w", err)
	}
	return p, nil
}

// legacyKeys maps the retired variables that had a setting to its stored key.
var legacyKeys = map[string]string{"HOST": "host", "PORT": "port", "HW_MAX_FILE_MB": "max_file_mb", "HW_MAX_BATCH_MB": "max_batch_mb",
	"HW_MAC_OCR_BIN": "mac_ocr_bin", "HW_WORKERS": "workers", "AI_BASE_URL": "ai_base_url", "AI_API_KEY": "ai_api_key",
	"AI_TIMEOUT_SECONDS": "ai_timeout_seconds"}

// retired are the environment variables that set the engine before the
// settings moved into the database.
var retired = []string{"HOST", "PORT", "HW_UI_BUILD_DIR", "HW_MAX_FILE_MB", "HW_MAX_BATCH_MB", "HW_MAC_OCR_BIN", "HW_WORKERS",
	"AI_BASE_URL", "AI_API_KEY", "AI_TIMEOUT_SECONDS", "AI_MODEL", "AI_MODEL_READ", "AI_REASONING_EFFORT"}

// Ignored lists the retired variables that are set, so the engine can say
// they no longer do anything.
func Ignored(getenv func(string) string) []string {
	var out []string
	for _, k := range retired {
		if strings.TrimSpace(getenv(k)) != "" {
			out = append(out, k)
		}
	}
	return out
}
