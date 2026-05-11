package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (p *Provider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	req = req.WithStream(false)
	httpReq, err := p.newNativeResponsesRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github copilot responses error %d: %s", resp.StatusCode, b)
	}

	var result domain.ResponsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode github copilot responses: %w", err)
	}
	return &result, nil
}

func (p *Provider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	req = req.WithStream(true)
	httpReq, err := p.newNativeResponsesRequest(ctx, req)
	if err != nil {
		return domain.Usage{}, err
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("github copilot responses error %d: %s", resp.StatusCode, b)
	}

	return shared.StreamResponsesSSEAndCountUsage(resp.Body, w)
}
