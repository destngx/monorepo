package openai

import (
	"apps/ai-gateway/internal/domain"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

func (p *Provider) doOpenAIRequest(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	return p.doOpenAIRequestWithClient(ctx, method, path, body, contentType, p.client)
}

func (p *Provider) doOpenAIStreamRequest(ctx context.Context, method, path string, body []byte, contentType string) (*http.Response, error) {
	return p.doOpenAIRequestWithClient(ctx, method, path, body, contentType, p.streamingClient())
}

func (p *Provider) doOpenAIRequestWithClient(ctx context.Context, method, path string, body []byte, contentType string, client *http.Client) (*http.Response, error) {
	resp, err := p.doOpenAIRequestOnce(ctx, method, path, body, contentType, client)
	if err != nil || resp.StatusCode != http.StatusUnauthorized || !p.useCodex() {
		return resp, err
	}

	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if _, err := p.refreshAccessToken(ctx, true); err != nil {
		return nil, err
	}
	return p.doOpenAIRequestOnce(ctx, method, path, body, contentType, client)
}

func (p *Provider) doOpenAIRequestOnce(ctx context.Context, method, path string, body []byte, contentType string, client *http.Client) (*http.Response, error) {
	slog.Debug("OpenAI upstream request", "method", method, "path", path)
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
	return client.Do(httpReq)
}

func (p *Provider) doResponsesRequest(ctx context.Context, req domain.ResponsesRequest) (*http.Response, error) {
	return p.doResponsesRequestWithClient(ctx, req, p.client)
}

func (p *Provider) doResponsesStreamRequest(ctx context.Context, req domain.ResponsesRequest) (*http.Response, error) {
	return p.doResponsesRequestWithClient(ctx, req, p.streamingClient())
}

func (p *Provider) doResponsesRequestWithClient(ctx context.Context, req domain.ResponsesRequest, client *http.Client) (*http.Response, error) {
	slog.Debug("OpenAI upstream request", "method", http.MethodPost, "path", pathResponses)
	useCodex := p.useCodex()
	if useCodex {
		req = codexCompatibleResponsesRequest(req)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if !useCodex {
		return p.doOpenAIRequestWithClient(ctx, http.MethodPost, pathResponses, body, contentTypeJSON, client)
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
	httpReq.Header.Set(headerVersion, p.codexVersion)

	return client.Do(httpReq)
}

func (p *Provider) streamingClient() *http.Client {
	if p.streamClient != nil {
		return p.streamClient
	}
	return p.client
}

// codexCompatibleResponsesRequest removes standard Responses fields that the
// ChatGPT Codex transport rejects. The public gateway continues accepting the
// standard request shape; provider-specific compatibility stays in this adapter.
func codexCompatibleResponsesRequest(req domain.ResponsesRequest) domain.ResponsesRequest {
	body := req.CloneBody()
	delete(body, responsesFieldMaxOutputTokens)
	body[responsesFieldStore] = false
	req.Body = body
	return req
}
