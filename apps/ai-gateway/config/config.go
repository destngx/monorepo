package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type ProviderConfig struct {
	RPM   int
	Burst int
}

type Config struct {
	GitHubToken   string
	OpenAIKey     string
	AnthropicKey  string
	OllamaBaseURL string
	ListenAddr    string

	GitHubRate    ProviderConfig
	OpenAIRate    ProviderConfig
	AnthropicRate ProviderConfig
	OllamaRate    ProviderConfig
	BedrockRate   ProviderConfig

	BedrockRegion string
	Verbose       int
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

	return &Config{
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
		OpenAIKey:     os.Getenv("OPENAI_API_KEY"),
		AnthropicKey:  os.Getenv("ANTHROPIC_API_KEY"),
		OllamaBaseURL: ollamaBase,
		ListenAddr:    addr,
		Verbose:       verbose,

		GitHubRate:    loadProviderRate("GITHUB"),
		OpenAIRate:    loadProviderRate("OPENAI"),
		AnthropicRate: loadProviderRate("ANTHROPIC"),
		OllamaRate:    loadProviderRate("OLLAMA"),
		BedrockRate:   loadProviderRate("BEDROCK"),

		BedrockRegion: getEnv("BEDROCK_REGION", "ap-southeast-1"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
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
