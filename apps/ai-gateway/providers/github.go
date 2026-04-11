package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"apps/ai-gateway/types"
)

const githubBaseURL = "https://models.github.ai/inference"
const githubCatalogURL = "https://models.github.ai/catalog/models"

type GitHubProvider struct {
	token  string
	client *http.Client
}

func NewGitHub(token string) *GitHubProvider {
	return &GitHubProvider{
		token:  token,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (g *GitHubProvider) Name() string { return "github" }

func (g *GitHubProvider) headers() map[string]string {
	return map[string]string{
		"Authorization":       "Bearer " + g.token,
		"Accept":              "application/vnd.github+json",
		"X-GitHub-Api-Version": "2026-03-10",
		"Content-Type":        "application/json",
	}
}

func (g *GitHubProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		githubBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range g.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github models error %d: %s", resp.StatusCode, b)
	}

	var result types.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (g *GitHubProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	req.Stream = true
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		githubBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return types.Usage{}, err
	}
	for k, v := range g.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return types.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return types.Usage{}, fmt.Errorf("github models error %d: %s", resp.StatusCode, b)
	}

	// Pass raw SSE bytes directly to writer — zero copy, zero latency
	usage, err := streamSSEAndCountTokens(resp.Body, w)
	return usage, err
}

// ListModels fetches the model catalog from GitHub Models API.
func (g *GitHubProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, githubCatalogURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range g.headers() {
		httpReq.Header.Set(k, v)
	}

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github models catalog error %d: %s", resp.StatusCode, b)
	}

	// GitHub catalog returns an array of model objects; convert to OpenAI-compatible format
	var catalog []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Publisher string `json:"publisher"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("failed to decode github catalog: %w", err)
	}

	models := make([]types.ModelInfo, 0, len(catalog))
	for _, m := range catalog {
		models = append(models, types.ModelInfo{
			ID:      m.Name,
			Object:  "model",
			OwnedBy: m.Publisher,
		})
	}

	return &types.ModelsResponse{
		Object: "list",
		Data:   models,
	}, nil
}
