package copilot

import (
	"apps/wealth-management-engine/domain"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CopilotClient struct {
	config     domain.AIConfig
	httpClient *http.Client

	mu           sync.Mutex
	cachedToken  string
	cachedExpiry int64
}

func NewCopilotClient(config domain.AIConfig) (*CopilotClient, error) {
	if config.GitHubToken == "" && config.CopilotBearerToken == "" {
		return nil, errors.New("missing GitHub token or Copilot bearer token")
	}
	if strings.TrimSpace(config.EditorVersion) == "" {
		config.EditorVersion = "vscode/1.80.0"
	}
	if strings.TrimSpace(config.EditorPluginVersion) == "" {
		config.EditorPluginVersion = "copilot-chat/0.1.0"
	}
	if strings.TrimSpace(config.CopilotIntegrationID) == "" {
		config.CopilotIntegrationID = "vscode-chat"
	}
	if strings.TrimSpace(config.UserAgent) == "" {
		config.UserAgent = "GitHubCopilotChat/0.1.0"
	}

	return &CopilotClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func NewCopilotClientWithHTTPClient(config domain.AIConfig, httpClient *http.Client) (*CopilotClient, error) {
	client, err := NewCopilotClient(config)
	if err != nil {
		return nil, err
	}
	client.httpClient = httpClient
	return client, nil
}

func (c *CopilotClient) applyCopilotHeaders(request *http.Request) {
	request.Header.Set("Editor-Version", c.config.EditorVersion)
	request.Header.Set("Editor-Plugin-Version", c.config.EditorPluginVersion)
	request.Header.Set("Copilot-Integration-Id", c.config.CopilotIntegrationID)
	request.Header.Set("User-Agent", c.config.UserAgent)
}
