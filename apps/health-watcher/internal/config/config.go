// Package config reads the engine's settings from the environment.
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"apps/health-watcher/domain"
)

// Config is the engine's runtime configuration.
type Config struct {
	Host       string
	Port       int
	DataDir    string
	UIBuildDir string
	MaxFileMB  int
	MaxBatchMB int
	// MacOCRBin is the mac-ocr binary (HW-READ-1).
	MacOCRBin string
	// Workers is how many background jobs run at once.
	Workers int
	AI      AI
}

// The model and reasoning effort are fixed, not settings: readings were
// measured with this pair (D6), and the gateway account serves few models.
const (
	AIModel  = "gpt-6-luna"
	AIEffort = "low"
)

// AI is the AI provider setting (HW-AI-1). An empty BaseURL turns AI
// reading off; OCR and search still work.
type AI struct {
	BaseURL string
	APIKey  string
	Model   string
	Effort  string
	Timeout time.Duration
}

// Enabled reports whether AI reading is configured.
func (a AI) Enabled() bool { return a.BaseURL != "" }

// WorkDir is the background reader's private scratch directory.
func (c Config) WorkDir() string { return filepath.Join(c.DataDir, "work") }

// DBPath is the database file inside the data directory (HW-OPS-1).
func (c Config) DBPath() string { return filepath.Join(c.DataDir, "health.db") }

// FilesDir is the file store inside the data directory (HW-OPS-2).
func (c Config) FilesDir() string { return filepath.Join(c.DataDir, "files") }

// SetupCodePath is the first-run setup code file (HW-AUTH-4).
func (c Config) SetupCodePath() string { return filepath.Join(c.DataDir, "setup-code") }

// Addr is the listen address.
func (c Config) Addr() string {
	host := c.Host
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("%s:%d", host, c.Port)
}

// Load reads HOST, PORT, HW_DATA_DIR, HW_UI_BUILD_DIR, HW_MAX_FILE_MB,
// HW_MAX_BATCH_MB, HW_MAC_OCR_BIN, HW_WORKERS, AI_BASE_URL, AI_API_KEY and
// AI_TIMEOUT_SECONDS, with
// defaults for each (HW-OPS-6, HW-IMP-2, HW-AI-1).
func Load(getenv func(string) string) (Config, error) {
	c := Config{
		Host:       envOr(getenv, "HOST", "127.0.0.1"),
		UIBuildDir: envOr(getenv, "HW_UI_BUILD_DIR", "ui/build"),
		DataDir:    strings.TrimSpace(getenv("HW_DATA_DIR")),
		MacOCRBin:  envOr(getenv, "HW_MAC_OCR_BIN", filepath.Join("..", "mac-ocr", "bin", "mac-ocr")),
	}
	var err error
	if c.MacOCRBin, err = filepath.Abs(c.MacOCRBin); err != nil {
		return c, fmt.Errorf("resolve HW_MAC_OCR_BIN: %w", err)
	}
	if c.Workers, err = intEnv(getenv, "HW_WORKERS", 2, 1, 8); err != nil {
		return c, err
	}
	if c.AI, err = loadAI(getenv); err != nil {
		return c, err
	}
	if c.Port, err = intEnv(getenv, "PORT", 9191, 1, 65535); err != nil {
		return c, err
	}
	if c.MaxFileMB, err = intEnv(getenv, "HW_MAX_FILE_MB", domain.DefaultMaxFileMB, 1, 1024); err != nil {
		return c, err
	}
	if c.MaxBatchMB, err = intEnv(getenv, "HW_MAX_BATCH_MB", domain.DefaultMaxBatchMB, 1, 4096); err != nil {
		return c, err
	}
	if c.MaxBatchMB < c.MaxFileMB {
		return c, fmt.Errorf("HW_MAX_BATCH_MB (%d) is smaller than HW_MAX_FILE_MB (%d)", c.MaxBatchMB, c.MaxFileMB)
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

func loadAI(getenv func(string) string) (AI, error) {
	a := AI{
		BaseURL: strings.TrimRight(strings.TrimSpace(getenv("AI_BASE_URL")), "/"),
		APIKey:  strings.TrimSpace(getenv("AI_API_KEY")),
		Model:   AIModel,
		Effort:  AIEffort,
	}
	if a.BaseURL != "" {
		u, err := url.Parse(a.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return a, fmt.Errorf("AI_BASE_URL must be an http or https URL, got %q", a.BaseURL)
		}
	}
	secs, err := intEnv(getenv, "AI_TIMEOUT_SECONDS", 180, 5, 1800)
	if err != nil {
		return a, err
	}
	a.Timeout = time.Duration(secs) * time.Second
	return a, nil
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}

func intEnv(getenv func(string) string, key string, fallback, lo, hi int) (int, error) {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return 0, fmt.Errorf("%s must be a whole number from %d to %d, got %q", key, lo, hi, raw)
	}
	return n, nil
}
