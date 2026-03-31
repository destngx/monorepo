package config

import (
	"apps/wealth-management-engine/domain"
	"errors"
	"fmt"
	"os"

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
var ErrMissingAIConfig = errors.New("missing GitHub Copilot configuration")

func LoadCacheConfig() (domain.CacheConfig, error) {
	config := domain.CacheConfig{
		RESTURL:   os.Getenv("UPSTASH_REDIS_REST_URL"),
		RESTToken: os.Getenv("UPSTASH_REDIS_REST_TOKEN"),
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

func LoadAIConfig() (domain.AIConfig, error) {
	config := domain.AIConfig{
		GitHubToken:          os.Getenv("GITHUB_TOKEN"),
		GitHubAPIBase:        getOrDefault("GITHUB_API_BASE_URL", "https://api.github.com"),
		CopilotAPIBase:       getOrDefault("COPILOT_API_BASE_URL", "https://api.githubcopilot.com"),
		DefaultModel:         getOrDefault("COPILOT_DEFAULT_MODEL", "gpt-4.1"),
		CopilotBearerToken:   os.Getenv("COPILOT_BEARER_TOKEN"),
		EditorVersion:        getOrDefault("COPILOT_EDITOR_VERSION", "vscode/1.80.0"),
		EditorPluginVersion:  getOrDefault("COPILOT_EDITOR_PLUGIN_VERSION", "copilot-chat/0.1.0"),
		CopilotIntegrationID: getOrDefault("COPILOT_INTEGRATION_ID", "vscode-chat"),
		UserAgent:            getOrDefault("COPILOT_USER_AGENT", "GitHubCopilotChat/0.1.0"),
	}

	if config.GitHubToken == "" && config.CopilotBearerToken == "" {
		return domain.AIConfig{}, fmt.Errorf("%w: [GITHUB_TOKEN or COPILOT_BEARER_TOKEN]", ErrMissingAIConfig)
	}

	return config, nil
}

func LoadMarketDataProviderConfig(baseURLEnv string, fallbackBaseURL string) domain.MarketDataProviderConfig {
	return domain.MarketDataProviderConfig{
		BaseURL: getOrDefault(baseURLEnv, fallbackBaseURL),
	}
}
