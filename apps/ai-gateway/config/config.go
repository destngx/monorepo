package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	openAIOAuthPathEnv = "OPENAI_OAUTH_PATH"
	codexAuthPathEnv   = "CODEX_AUTH_PATH"
)

type ProviderConfig struct {
	RPM   int
	Burst int
}

type OpenAIOAuth struct {
	AccountID    string `json:"account_id"`
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Config struct {
	GitHubToken       string
	GitHubAccountType string
	OpenAIKey         string
	OpenAIOAuth       *OpenAIOAuth
	AnthropicKey      string
	AnthropicRoute    string
	OllamaBaseURL     string
	ListenAddr        string

	GitHubRate    ProviderConfig
	OpenAIRate    ProviderConfig
	AnthropicRate ProviderConfig
	OllamaRate    ProviderConfig
	BedrockRate   ProviderConfig

	BedrockRegion string
	Verbose       int
	LogLevel      string
	EnableColor   bool

	MetricsBufferSize   int
	MetricsSaveInterval int // seconds
}

func Load() *Config {
	envPath := os.Getenv("ENV_PATH")
	if envPath == "" {
		envPath = ".env.local"
	}

	if err := godotenv.Load(envPath); err != nil {
		if os.Getenv("ENV_PATH") != "" {
			log.Printf("Warning: custom env file %q not found", envPath)
		} else {
			log.Println("No env file found, consuming environment variables directly")
		}
	} else {
		log.Printf("Loaded environment from %q", envPath)
	}

	ollamaBase := os.Getenv("OLLAMA_BASE_URL")
	if ollamaBase == "" {
		ollamaBase = "http://localhost:11434"
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	verboseStr := os.Getenv("VERBOSE")
	verbose, _ := strconv.Atoi(verboseStr)

	logLevel := getEnv("LOG_LEVEL", "info")
	enableColor := getEnv("LOG_COLOR", "true") != "false"

	return &Config{
		GitHubToken:       os.Getenv("GITHUB_TOKEN"),
		GitHubAccountType: getEnv("GITHUB_ACCOUNT_TYPE", "business"),
		OpenAIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIOAuth:       ReloadOpenAIOAuth(),
		AnthropicKey:      os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicRoute:    getEnv("ANTHROPIC_ROUTE", "default"),
		OllamaBaseURL:     ollamaBase,
		ListenAddr:        addr,
		Verbose:           verbose,
		LogLevel:          logLevel,
		EnableColor:       enableColor,

		GitHubRate:    loadProviderRate("GITHUB"),
		OpenAIRate:    loadProviderRate("OPENAI"),
		AnthropicRate: loadProviderRate("ANTHROPIC"),
		OllamaRate:    loadProviderRate("OLLAMA"),
		BedrockRate:   loadProviderRate("BEDROCK"),

		BedrockRegion: getEnv("BEDROCK_REGION", "us-east-1"),

		MetricsBufferSize:   getEnvInt("METRICS_BUFFER_SIZE", 2000),
		MetricsSaveInterval: getEnvInt("METRICS_SAVE_INTERVAL_SECS", 60),
	}
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type codexAuth struct {
	AuthMode     string      `json:"auth_mode,omitempty"`
	OpenAIAPIKey *string     `json:"OPENAI_API_KEY"`
	OAuthToken   string      `json:"oauth_token,omitempty"`
	AccessToken  string      `json:"access_token,omitempty"`
	Token        string      `json:"token,omitempty"`
	Tokens       OpenAIOAuth `json:"tokens"`
	LastRefresh  string      `json:"last_refresh,omitempty"`
}

func ReloadOpenAIOAuth() *OpenAIOAuth {
	oauth, _ := ReloadOpenAIOAuthSource()
	return oauth
}

func ReloadOpenAIOAuthSource() (*OpenAIOAuth, string) {
	paths := []string{}
	if override := os.Getenv(openAIOAuthPathEnv); override != "" {
		paths = append(paths, override)
	}
	if override := os.Getenv(codexAuthPathEnv); override != "" {
		paths = append(paths, override)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, ""
	}

	paths = append(paths,
		filepath.Join(home, ".codex", "auth.json"),
		filepath.Join(home, ".code", "auth.json"),
	)

	seen := make(map[string]struct{}, len(paths))
	for _, authPath := range paths {
		if _, ok := seen[authPath]; ok {
			continue
		}
		seen[authPath] = struct{}{}

		data, err := os.ReadFile(authPath)
		if err != nil {
			continue
		}

		var auth codexAuth
		if err := json.Unmarshal(data, &auth); err != nil {
			continue
		}
		if auth.Tokens.AccessToken == "" {
			auth.Tokens.AccessToken = firstNonEmpty(auth.OAuthToken, auth.AccessToken, auth.Token)
		}

		if auth.Tokens.AccessToken != "" || auth.Tokens.RefreshToken != "" || auth.Tokens.IDToken != "" {
			return &auth.Tokens, authPath
		}
	}
	return nil, ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func SaveOpenAIOAuthSource(authPath string, tokens *OpenAIOAuth) error {
	data, err := os.ReadFile(authPath)
	if err != nil {
		return err
	}

	var auth codexAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return err
	}

	if tokens.AccountID != "" {
		auth.Tokens.AccountID = tokens.AccountID
	}
	if tokens.IDToken != "" {
		auth.Tokens.IDToken = tokens.IDToken
	}
	if tokens.AccessToken != "" {
		auth.Tokens.AccessToken = tokens.AccessToken
	}
	if tokens.RefreshToken != "" {
		auth.Tokens.RefreshToken = tokens.RefreshToken
	}
	auth.LastRefresh = time.Now().UTC().Format(time.RFC3339Nano)

	updated, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	updated = append(updated, '\n')
	return os.WriteFile(authPath, updated, 0600)
}

func loadProviderRate(prefix string) ProviderConfig {
	rpm, _ := strconv.Atoi(os.Getenv(prefix + "_RATE_RPM"))
	burst, _ := strconv.Atoi(os.Getenv(prefix + "_RATE_BURST"))

	// Default burst to RPM / 10 if not specified, at least 1
	if rpm > 0 && burst == 0 {
		burst = rpm / 10
		if burst < 1 {
			burst = 1
		}
	}

	return ProviderConfig{
		RPM:   rpm,
		Burst: burst,
	}
}
