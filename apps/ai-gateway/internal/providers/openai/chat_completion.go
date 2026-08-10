package openai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	if p.useCodex() {
		return p.chatCodex(ctx, req)
	}
	req = withPromptCacheConfig(req)
	if req.PromptCacheOptions != nil {
		return p.chatViaResponses(ctx, req)
	}
	body, _ := json.Marshal(req)
	resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathChatCompletions, body, contentTypeJSON)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		if isInsufficientQuota(b) {
			return nil, fmt.Errorf("%s: openai error %d: %s", openAIInsufficientQuotaMsg, resp.StatusCode, b)
		}
		return nil, fmt.Errorf("openai error %d: %s", resp.StatusCode, b)
	}
	var result domain.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	if p.useCodex() {
		return p.chatCodexStream(ctx, req, w)
	}
	req = withPromptCacheConfig(req)
	if req.PromptCacheOptions != nil {
		return p.chatStreamViaResponses(ctx, req, w)
	}
	req.Stream = true
	req.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	body, _ := json.Marshal(req)
	resp, err := p.doOpenAIRequest(ctx, http.MethodPost, pathChatCompletions, body, contentTypeJSON)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()
	return shared.StreamSSEAndCountTokens(resp.Body, w)
}

func withPromptCacheConfig(req domain.ChatRequest) domain.ChatRequest {
	hasBreakpoint := false
	h := sha256.New()
	for _, message := range req.Messages {
		if message.Role == domain.RoleSystem {
			_, _ = h.Write([]byte(message.Content))
		}
		for _, part := range message.Parts {
			if part.PromptCacheBreakpoint != nil {
				hasBreakpoint = true
			}
		}
	}
	if !hasBreakpoint {
		return req
	}
	for _, tool := range req.Tools {
		b, _ := json.Marshal(tool)
		_, _ = h.Write(b)
	}
	req.PromptCacheKey = "cc:" + req.Model + ":" + fmt.Sprintf("%x", h.Sum(nil))
	req.PromptCacheOptions = &domain.PromptCacheOptions{Mode: "explicit"}
	return req
}
