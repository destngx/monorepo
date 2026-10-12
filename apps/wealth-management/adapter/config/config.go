package config

import (
	"apps/wealth-management/domain"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var envFiles = []string{
	"../../.env.local",
	".env.local",
}

func Load() {
	for _, path := range envFiles {
		_ = godotenv.Overload(path)
	}
}

func LoadSheetsConfig() (domain.SheetsConfig, error) {
	config := domain.SheetsConfig{
		ClientID:      os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret:  os.Getenv("GOOGLE_CLIENT_SECRET"),
		RefreshToken:  os.Getenv("GOOGLE_REFRESH_TOKEN"),
		SpreadsheetID: os.Getenv("GOOGLE_SHEETS_ID"),
		RedirectURL:   getOrDefault("GOOGLE_REDIRECT_URL", "http://127.0.0.1:3000"),
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

	return config, nil
}

func getOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

var ErrMissingSheetsConfig = errors.New("missing Google Sheets configuration")
var ErrMissingCacheConfig = errors.New("missing Upstash Redis configuration")

func LoadCacheConfig() (domain.CacheConfig, error) {
	config := domain.CacheConfig{
		RESTURL:   os.Getenv("UPSTASH_REDIS_REST_URL"),
		RESTToken: os.Getenv("UPSTASH_REDIS_REST_TOKEN"),
		KeyPrefix: getOrDefault("CACHE_KEY_PREFIX", "wm:"),
	}

	var missing []string
	if config.RESTURL == "" {
		missing = append(missing, "UPSTASH_REDIS_REST_URL")
	}
	if config.RESTToken == "" {
		missing = append(missing, "UPSTASH_REDIS_REST_TOKEN")
	}

	if len(missing) > 0 {
		return domain.CacheConfig{}, fmt.Errorf("%w: %v", ErrMissingCacheConfig, missing)
	}

	return config, nil
}

func LoadMarketDataProviderConfig(baseURLEnv string, fallbackBaseURL string) domain.MarketDataProviderConfig {
	return domain.MarketDataProviderConfig{
		BaseURL: getOrDefault(baseURLEnv, fallbackBaseURL),
	}
}

func getOrDefaultInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// LoadServerConfig reads HOST (default 127.0.0.1), PORT (default 8080) and
// API_TOKEN, which guards write endpoints when set.
func LoadServerConfig() (domain.ServerConfig, error) {
	port, err := strconv.Atoi(getOrDefault("PORT", "8080"))
	if err != nil || port < 1 || port > 65535 {
		return domain.ServerConfig{}, fmt.Errorf("%w: PORT must be 1-65535", ErrInvalidServerConfig)
	}
	return domain.ServerConfig{
		Host:     getOrDefault("HOST", "127.0.0.1"),
		Port:     port,
		APIToken: os.Getenv("API_TOKEN"),
	}, nil
}

var ErrInvalidServerConfig = errors.New("invalid server configuration")

// BudgetSheetTemplate names the budget tab; "{year}" is replaced (WM-DATA-2).
func BudgetSheetTemplate() string {
	return getOrDefault("BUDGET_SHEET_TEMPLATE", "Budget_{year}")
}

func LoadLoggerConfig() (logLevel string, colorEnabled bool) {
	logLevel = os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	colorEnabled = os.Getenv("LOG_COLOR") != "false"
	return logLevel, colorEnabled
}
