package config

import (
	"apps/wealth-management/domain"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var envFiles = []string{
	"../../.env.local",
	".env.local",
}

// Load reads .env.local (repo root, then the app directory), overriding
// the process environment. WM_ENV_FILE names one file to load instead, and
// "none" loads nothing, so a harness can hand the engine its whole
// environment (WM-TEST-1).
func Load() {
	paths := envFiles
	switch override := strings.TrimSpace(os.Getenv("WM_ENV_FILE")); override {
	case "":
	case "none":
		return
	default:
		paths = []string{override}
	}
	for _, path := range paths {
		_ = godotenv.Overload(path)
	}
}

// LoadTestMode reads WM_TEST_MODE: the engine then refuses to start unless
// the spreadsheet is a disposable test copy (WM-TEST-2).
func LoadTestMode() bool {
	enabled, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("WM_TEST_MODE")))
	return enabled
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

	readsPerMinute, err := loadSheetsReadsPerMinute()
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
func loadSheetsReadsPerMinute() (int, error) {
	value := strings.TrimSpace(os.Getenv("SHEETS_READS_PER_MINUTE"))
	if value == "" {
		return DefaultSheetsReadsPerMinute, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%w: SHEETS_READS_PER_MINUTE must be a positive integer", ErrInvalidSheetsConfig)
	}
	return parsed, nil
}

func getOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

var ErrMissingSheetsConfig = errors.New("missing Google Sheets configuration")
var ErrInvalidSheetsConfig = errors.New("invalid Google Sheets configuration")
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

// LoadCoinGeckoConfig reads COINGECKO_BASE_URL and the optional
// COINGECKO_API_KEY (a demo key; it raises the public rate limit).
func LoadCoinGeckoConfig(fallbackBaseURL string) (domain.MarketDataProviderConfig, string) {
	return LoadMarketDataProviderConfig("COINGECKO_BASE_URL", fallbackBaseURL), strings.TrimSpace(os.Getenv("COINGECKO_API_KEY"))
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

var ErrMissingAIConfig = errors.New("missing AI configuration")

const defaultAITimeoutSeconds = 90

// LoadAIConfig reads the OpenAI-compatible provider settings (D10, WM-AI-4):
// AI_BASE_URL (e.g. http://127.0.0.1:8081/v1 for ai-gateway), AI_API_KEY,
// AI_PROVIDER (X-AI-Provider for ai-gateway), AI_MODEL as the default model,
// AI_MODEL_{INSIGHT,SUGGEST,VISION,CHAT} per use case, AI_FALLBACK_MODEL,
// AI_TIMEOUT_SECONDS and AI_PII_NAMES (comma separated names to mask).
func LoadAIConfig() (domain.AIConfig, error) {
	config := domain.AIConfig{
		BaseURL:       strings.TrimSpace(os.Getenv("AI_BASE_URL")),
		APIKey:        os.Getenv("AI_API_KEY"),
		Provider:      strings.TrimSpace(os.Getenv("AI_PROVIDER")),
		FallbackModel: strings.TrimSpace(os.Getenv("AI_FALLBACK_MODEL")),
		Timeout:       time.Duration(getOrDefaultInt("AI_TIMEOUT_SECONDS", defaultAITimeoutSeconds)) * time.Second,
		Models:        map[domain.AIUseCase]string{},
	}
	defaultModel := strings.TrimSpace(os.Getenv("AI_MODEL"))
	for _, useCase := range domain.AIUseCases {
		model := strings.TrimSpace(os.Getenv("AI_MODEL_" + strings.ToUpper(string(useCase))))
		if model == "" {
			model = defaultModel
		}
		if model != "" {
			config.Models[useCase] = model
		}
	}
	for _, name := range strings.Split(os.Getenv("AI_PII_NAMES"), ",") {
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
func LoadCaptureLimits() (domain.CaptureLimits, error) {
	defaults := domain.DefaultCaptureLimits()
	read := func(key string, fallback int) (int, error) {
		value := strings.TrimSpace(os.Getenv(key))
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
func LoadSearchConfig() domain.SearchConfig {
	return domain.SearchConfig{
		TavilyAPIKey: strings.TrimSpace(os.Getenv("TAVILY_API_KEY")),
		ExaAPIKey:    strings.TrimSpace(os.Getenv("EXA_API_KEY")),
	}
}
