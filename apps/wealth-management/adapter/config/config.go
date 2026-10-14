package config

import (
	"apps/wealth-management/domain"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// There is no env file. Integration settings come from the settings store
// (edited in the UI) as Values; only bootstrap values the engine needs
// before it can open the store are read from the process environment:
// HOST, PORT, API_TOKEN, WM_SETTINGS_PATH, CACHE_SQLITE_PATH,
// WM_TEST_WORKBOOK and LOG_COLOR.

// Values are the stored settings, keyed by the names in
// domain.SettingDefinitions.
type Values map[string]string

func (v Values) get(key string) string {
	return strings.TrimSpace(v[key])
}

func (v Values) getOrDefault(key string, fallback string) string {
	if value := v.get(key); value != "" {
		return value
	}
	return fallback
}

func (v Values) getOrDefaultInt(key string, fallback int) int {
	parsed, err := strconv.Atoi(v.get(key))
	if err != nil {
		return fallback
	}
	return parsed
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// LoadSettingsPath reads WM_SETTINGS_PATH; empty means the default location.
func LoadSettingsPath() string {
	return strings.TrimSpace(os.Getenv("WM_SETTINGS_PATH"))
}

// LoadTestWorkbook reads WM_TEST_WORKBOOK, a local .xlsx export the engine
// then serves the sheet from, in memory, instead of Google Sheets
// (WM-TEST-2). Empty means Google Sheets; a leading "~/" is expanded.
func LoadTestWorkbook() (string, error) {
	return ExpandHome(os.Getenv("WM_TEST_WORKBOOK"))
}

// ExpandHome trims path and expands a leading "~/" to the home directory.
func ExpandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	rest, ok := strings.CutPrefix(path, "~/")
	if !ok {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand %s: %w", path, err)
	}
	return filepath.Join(home, rest), nil
}

func LoadSheetsConfig(values Values) (domain.SheetsConfig, error) {
	config := domain.SheetsConfig{
		ClientID:      values.get("GOOGLE_CLIENT_ID"),
		ClientSecret:  values.get("GOOGLE_CLIENT_SECRET"),
		RefreshToken:  values.get("GOOGLE_REFRESH_TOKEN"),
		SpreadsheetID: values.get("GOOGLE_SHEETS_ID"),
		RedirectURL:   values.get("GOOGLE_REDIRECT_URL"),
	}

	var missing []string
	if config.ClientID == "" {
		missing = append(missing, "GOOGLE_CLIENT_ID")
	}
	if config.ClientSecret == "" {
		missing = append(missing, "GOOGLE_CLIENT_SECRET")
	}
	if config.RefreshToken == "" {
		missing = append(missing, "GOOGLE_REFRESH_TOKEN")
	}
	if config.SpreadsheetID == "" {
		missing = append(missing, "GOOGLE_SHEETS_ID")
	}

	if len(missing) > 0 {
		return domain.SheetsConfig{}, fmt.Errorf("%w: %v", ErrMissingSheetsConfig, missing)
	}

	readsPerMinute, err := loadSheetsReadsPerMinute(values)
	if err != nil {
		return domain.SheetsConfig{}, err
	}
	config.ReadsPerMinute = readsPerMinute

	return config, nil
}

// DefaultSheetsReadsPerMinute matches the Sheets API default read quota per
// user (WM-NFR-3).
const DefaultSheetsReadsPerMinute = 60

// loadSheetsReadsPerMinute reads SHEETS_READS_PER_MINUTE, the pace of Sheets
// read requests (default 60).
func loadSheetsReadsPerMinute(values Values) (int, error) {
	value := values.get("SHEETS_READS_PER_MINUTE")
	if value == "" {
		return DefaultSheetsReadsPerMinute, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%w: SHEETS_READS_PER_MINUTE must be a positive integer", ErrInvalidSheetsConfig)
	}
	return parsed, nil
}

var ErrMissingSheetsConfig = errors.New("missing Google Sheets configuration")
var ErrInvalidSheetsConfig = errors.New("invalid Google Sheets configuration")
var ErrMissingCacheConfig = errors.New("missing Upstash Redis configuration")

// LoadCacheConfig picks Upstash when both UPSTASH_REDIS_REST_URL and
// UPSTASH_REDIS_REST_TOKEN are set, and a local SQLite file otherwise
// (CACHE_SQLITE_PATH from the environment, default <system
// temp>/wealth-management/cache.db). Only one of the two Upstash settings is
// a mistake and fails rather than silently switching to SQLite.
func LoadCacheConfig(values Values) (domain.CacheConfig, error) {
	prefix := values.getOrDefault("CACHE_KEY_PREFIX", "wm:")
	url := values.get("UPSTASH_REDIS_REST_URL")
	token := values.get("UPSTASH_REDIS_REST_TOKEN")

	switch {
	case url != "" && token != "":
		return domain.CacheConfig{Backend: domain.CacheBackendUpstash, RESTURL: url, RESTToken: token, KeyPrefix: prefix}, nil
	case url != "":
		return domain.CacheConfig{}, fmt.Errorf("%w: [UPSTASH_REDIS_REST_TOKEN]", ErrMissingCacheConfig)
	case token != "":
		return domain.CacheConfig{}, fmt.Errorf("%w: [UPSTASH_REDIS_REST_URL]", ErrMissingCacheConfig)
	}

	return domain.CacheConfig{Backend: domain.CacheBackendSQLite, SQLitePath: envOrDefault("CACHE_SQLITE_PATH", DefaultSQLiteCachePath()), KeyPrefix: prefix}, nil
}

// DefaultSQLiteCachePath is the SQLite cache file in the system temp
// directory; the cache only holds copies, so losing it on reboot is fine.
func DefaultSQLiteCachePath() string {
	return filepath.Join(os.TempDir(), "wealth-management", "cache.db")
}

// LoadMarketDataServerURL reads MARKET_DATA_SERVER_URL, the address of
// apps/market-data-server, which serves every market source (WM-MKT-1).
func LoadMarketDataServerURL(values Values) string {
	return values.getOrDefault("MARKET_DATA_SERVER_URL", "http://localhost:8000")
}

// LoadServerConfig reads HOST (default 127.0.0.1), PORT (default 8080) and
// API_TOKEN, which guards write endpoints when set, from the environment.
func LoadServerConfig() (domain.ServerConfig, error) {
	port, err := strconv.Atoi(envOrDefault("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		return domain.ServerConfig{}, fmt.Errorf("%w: PORT must be 1-65535", ErrInvalidServerConfig)
	}
	return domain.ServerConfig{
		Host:     envOrDefault("HOST", "127.0.0.1"),
		Port:     port,
		APIToken: strings.TrimSpace(os.Getenv("API_TOKEN")),
	}, nil
}

var ErrInvalidServerConfig = errors.New("invalid server configuration")

// BudgetSheetTemplate names the budget tab; "{year}" is replaced (WM-DATA-2).
func BudgetSheetTemplate(values Values) string {
	return values.getOrDefault("BUDGET_SHEET_TEMPLATE", "Budget_{year}")
}

// LoadLogLevel reads LOG_LEVEL (default info).
func LoadLogLevel(values Values) string {
	return values.getOrDefault("LOG_LEVEL", "info")
}

// LoadLogColor reads LOG_COLOR from the environment: colour is a property
// of the terminal, not of the engine.
func LoadLogColor() bool {
	return os.Getenv("LOG_COLOR") != "false"
}

var ErrMissingAIConfig = errors.New("missing AI configuration")

const defaultAITimeoutSeconds = 90

// LoadAIConfig reads the OpenAI-compatible provider settings (D10, WM-AI-4):
// AI_BASE_URL (an OpenAI Responses API base, e.g. http://127.0.0.1:8081/v1 for ai-gateway), AI_API_KEY,
// AI_PROVIDER (X-AI-Provider for ai-gateway), AI_MODEL as the default model,
// AI_MODEL_{INSIGHT,SUGGEST,VISION,CHAT} per use case, AI_FALLBACK_MODEL,
// AI_TIMEOUT_SECONDS and AI_PII_NAMES (comma separated names to mask).
func LoadAIConfig(values Values) (domain.AIConfig, error) {
	config := domain.AIConfig{
		BaseURL:       values.get("AI_BASE_URL"),
		APIKey:        values.get("AI_API_KEY"),
		Provider:      values.get("AI_PROVIDER"),
		FallbackModel: values.get("AI_FALLBACK_MODEL"),
		Timeout:       time.Duration(values.getOrDefaultInt("AI_TIMEOUT_SECONDS", defaultAITimeoutSeconds)) * time.Second,
		Models:        map[domain.AIUseCase]string{},
	}
	defaultModel := values.get("AI_MODEL")
	for _, useCase := range domain.AIUseCases {
		model := values.get("AI_MODEL_" + strings.ToUpper(string(useCase)))
		if model == "" {
			model = defaultModel
		}
		if model != "" {
			config.Models[useCase] = model
		}
	}
	for _, name := range strings.Split(values.get("AI_PII_NAMES"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			config.PIINames = append(config.PIINames, name)
		}
	}

	var missing []string
	if config.BaseURL == "" {
		missing = append(missing, "AI_BASE_URL")
	}
	for _, useCase := range domain.AIUseCases {
		if config.Models[useCase] == "" {
			missing = append(missing, "AI_MODEL")
			break
		}
	}
	if len(missing) > 0 {
		return domain.AIConfig{}, fmt.Errorf("%w: %v", ErrMissingAIConfig, missing)
	}
	return config, nil
}

var ErrInvalidCaptureConfig = errors.New("invalid Smart Capture configuration")

// LoadCaptureLimits reads the Smart Capture batch limits (WM-CAP-1):
// CAPTURE_MAX_ITEMS (default 20), CAPTURE_MAX_ITEM_MB (default 5) and
// CAPTURE_MAX_BATCH_MB (default 20).
func LoadCaptureLimits(values Values) (domain.CaptureLimits, error) {
	defaults := domain.DefaultCaptureLimits()
	read := func(key string, fallback int) (int, error) {
		value := values.get(key)
		if value == "" {
			return fallback, nil
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return 0, fmt.Errorf("%w: %s must be a positive integer", ErrInvalidCaptureConfig, key)
		}
		return parsed, nil
	}
	items, err := read("CAPTURE_MAX_ITEMS", defaults.MaxItems)
	if err != nil {
		return domain.CaptureLimits{}, err
	}
	itemMB, err := read("CAPTURE_MAX_ITEM_MB", defaults.MaxItemBytes>>20)
	if err != nil {
		return domain.CaptureLimits{}, err
	}
	batchMB, err := read("CAPTURE_MAX_BATCH_MB", defaults.MaxBatchBytes>>20)
	if err != nil {
		return domain.CaptureLimits{}, err
	}
	if itemMB > batchMB {
		return domain.CaptureLimits{}, fmt.Errorf("%w: CAPTURE_MAX_ITEM_MB must not exceed CAPTURE_MAX_BATCH_MB", ErrInvalidCaptureConfig)
	}
	return domain.CaptureLimits{MaxItems: items, MaxItemBytes: itemMB << 20, MaxBatchBytes: batchMB << 20}, nil
}

// LoadSearchConfig reads the optional TAVILY_API_KEY and EXA_API_KEY.
func LoadSearchConfig(values Values) domain.SearchConfig {
	return domain.SearchConfig{
		TavilyAPIKey: values.get("TAVILY_API_KEY"),
		ExaAPIKey:    values.get("EXA_API_KEY"),
	}
}
