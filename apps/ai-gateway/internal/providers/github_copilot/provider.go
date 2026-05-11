package github_copilot

import (
	"log"
	"net/http"
	"time"
)

func New(githubToken, accountType string, verbose int, headers ...ClientHeaders) *Provider {
	clientHeaders := defaultClientHeaders()
	if len(headers) > 0 {
		clientHeaders = headers[0].withDefaults()
	}
	return &Provider{
		githubToken: githubToken,
		accountType: accountType,
		client:      &http.Client{Timeout: 120 * time.Second},
		verbose:     verbose,
		headers:     clientHeaders,
	}
}

func defaultClientHeaders() ClientHeaders {
	return ClientHeaders{
		EditorVersion:       defaultEditorVersion,
		EditorPluginVersion: defaultEditorPluginVersion,
		IntegrationID:       defaultIntegrationID,
		UserAgent:           defaultUserAgent,
	}
}

func (h ClientHeaders) withDefaults() ClientHeaders {
	defaults := defaultClientHeaders()
	if h.EditorVersion == "" {
		h.EditorVersion = defaults.EditorVersion
	}
	if h.EditorPluginVersion == "" {
		h.EditorPluginVersion = defaults.EditorPluginVersion
	}
	if h.IntegrationID == "" {
		h.IntegrationID = defaults.IntegrationID
	}
	if h.UserAgent == "" {
		h.UserAgent = defaults.UserAgent
	}
	return h
}

func (p *Provider) vlogf(level int, format string, args ...any) {
	if p.verbose < level {
		return
	}
	log.Printf(format, args...)
}
