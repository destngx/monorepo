package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (p *Provider) getCopilotSession(ctx context.Context) (string, string, error) {
	p.mu.Lock()
	if p.cachedToken != "" && p.expiresAt > time.Now().Unix()+300 {
		token := p.cachedToken
		apiBase := p.copilotAPIBase
		if apiBase == "" {
			apiBase = baseURL
		}
		p.mu.Unlock()
		p.vlogf(2, "[github-copilot] session cache hit token=hit base_url=%q", apiBase)
		return token, apiBase, nil
	}
	p.mu.Unlock()

	sessionStart := time.Now()
	v, err, _ := p.tokenGroup.Do("copilot-token", func() (any, error) {
		p.vlogf(1, "[github-copilot] session cache miss, fetching token")
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set(headerAuthorization, tokenPrefixToken+p.githubToken)
		httpReq.Header.Set(headerAccept, contentTypeJSON)
		p.setClientHeaders(httpReq)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("github copilot token error %d: %s", resp.StatusCode, b)
		}

		var tr tokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
			return nil, fmt.Errorf("failed to decode github copilot token: %w", err)
		}
		if tr.Token == "" {
			return nil, fmt.Errorf("github copilot token response missing token")
		}

		apiBase := p.getCopilotBaseURL()
		if tr.Endpoints.API != "" {
			apiBase = tr.Endpoints.API
		}

		p.mu.Lock()
		p.cachedToken = tr.Token
		p.expiresAt = tr.ExpiresAt
		p.copilotAPIBase = apiBase
		p.mu.Unlock()
		p.vlogf(1, "[github-copilot] token fetch took=%s base_url=%q expires_at=%d", time.Since(sessionStart), apiBase, tr.ExpiresAt)

		return sessionData{
			token:   tr.Token,
			baseURL: apiBase,
		}, nil
	})
	if err != nil {
		return "", "", err
	}
	session := v.(sessionData)
	p.vlogf(2, "[github-copilot] session ready base_url=%q", session.baseURL)
	return session.token, session.baseURL, nil
}

func (p *Provider) getCopilotBaseURL() string {
	if p.accountType == "" || p.accountType == "individual" {
		return baseURL
	}
	return fmt.Sprintf("https://api.%s.githubcopilot.com", p.accountType)
}

func (p *Provider) setClientHeaders(req *http.Request) {
	req.Header.Set(headerEditorVersion, p.headers.EditorVersion)
	req.Header.Set(headerEditorPluginVer, p.headers.EditorPluginVersion)
	req.Header.Set(headerIntegrationID, p.headers.IntegrationID)
	req.Header.Set(headerUserAgent, p.headers.UserAgent)
}

func isReasoningModel(model string) bool {
	m := strings.ToLower(model)
	return strings.HasPrefix(m, "o1") ||
		strings.HasPrefix(m, "o3") ||
		strings.HasPrefix(m, "gpt-5") ||
		strings.Contains(m, "reasoning")
}

func isResponsesModel(model string) bool {
	switch strings.ToLower(model) {
	case domain.ModelGPT54, domain.ModelGPT54Mini:
		return true
	default:
		return false
	}
}

func isNoTemperatureModel(model string) bool {
	m := strings.ToLower(model)
	return isReasoningModel(m) || isResponsesModel(m) || strings.Contains(m, "mini")
}
