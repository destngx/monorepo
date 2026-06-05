package domain

const (
	ProviderDefault       = ProviderOpenAI
	ProviderGitHubCopilot = "github-copilot"
	ProviderGitHubModels  = "github-models"
	ProviderBedrock       = "bedrock"
	ProviderOpenAI        = "openai"
	ProviderAnthropic     = "anthropic"
	ProviderOllama        = "ollama"
	ProviderXiaomiMimo    = "xiaomi-mimo"

	HeaderAIProvider = "X-AI-Provider"
	ContentTypeJSON  = "application/json"
)
