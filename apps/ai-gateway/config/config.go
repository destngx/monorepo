package config

import "os"

type Config struct {
	GitHubToken   string
	OpenAIKey     string
	AnthropicKey  string
	OllamaBaseURL string
	ListenAddr    string
}

func Load() *Config {
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
