package openai

import (
	"apps/ai-gateway/internal/domain"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

func (p *Provider) doOpenAIRequest(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	resp, err := p.doOpenAIRequestOnce(ctx, method, path, body, contentType)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || p.apiKey != "" {
		return resp, err
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if _, err := p.refreshAccessToken(ctx, true); err != nil {
		return nil, err
	}
	return p.doOpenAIRequestOnce(ctx, method, path, body, contentType)
}

func (p *Provider) doOpenAIRequestOnce(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	if contentType != "" {
		httpReq.Header.Set(headerContentType, contentType)
	}
	return p.client.Do(httpReq)
}

func (p *Provider) doResponsesRequest(ctx context.Context, req domain.ResponsesRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		return p.doOpenAIRequest(ctx, http.MethodPost, pathResponses, body, contentTypeJSON)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTURL+pathCodexResponses, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	httpReq.Header.Set(headerOpenAIBeta, codexResponsesExperimental)
	httpReq.Header.Set(headerOriginator, codexOriginator)
	httpReq.Header.Set(headerSessionID, newCodexSessionID())
	httpReq.Header.Set(headerUserAgent, "")
	httpReq.Header.Set(headerVersion, getEnv(envOpenAICodexVersion, codexDefaultVersion))

	return p.client.Do(httpReq)
}
