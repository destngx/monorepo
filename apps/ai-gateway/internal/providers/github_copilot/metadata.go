package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	token, apiBase, err := p.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}

	url := apiBase + pathModels
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerAuthorization, tokenPrefixBearer+token)
	req.Header.Set(headerAccept, contentTypeJSON)
	p.setClientHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github copilot models error %d: %s", resp.StatusCode, b)
	}

	var result domain.ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode github copilot models: %w", err)
	}

	// Ensure OwnedBy is set for consistency if missing in response
	for i := range result.Data {
		if result.Data[i].OwnedBy == "" {
			result.Data[i].OwnedBy = "github-copilot"
		}
	}

	return &result, nil
}

func (p *Provider) IsConfigured() bool {
	return p.githubToken != ""
}

func (p *Provider) Ping(ctx context.Context) error {
	if !p.IsConfigured() {
		return fmt.Errorf("github copilot not configured")
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(headerAuthorization, tokenPrefixToken+p.githubToken)
	req.Header.Set(headerAccept, contentTypeJSON)
	p.setClientHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github copilot usage error %d: %s", resp.StatusCode, b)
	}

	var usage UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usage); err != nil {
		return nil, err
	}

	return usage, nil
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
