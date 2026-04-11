package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubToken   string
	OpenAIKey     string
	AnthropicKey  string
	OllamaBaseURL string
	ListenAddr    string
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
	return &Config{
		GitHubToken:   os.Getenv("GITHUB_TOKEN"),
		OpenAIKey:     os.Getenv("OPENAI_API_KEY"),
		AnthropicKey:  os.Getenv("ANTHROPIC_API_KEY"),
		OllamaBaseURL: ollamaBase,
		ListenAddr:    addr,
	}
}
