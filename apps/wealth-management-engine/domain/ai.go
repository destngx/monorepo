package domain

type AIConfig struct {
	GitHubToken          string
	GitHubAPIBase        string
	CopilotAPIBase       string
	DefaultModel         string
	CopilotBearerToken   string
	EditorVersion        string
	EditorPluginVersion  string
	CopilotIntegrationID string
	UserAgent            string
}
