package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (p *Provider) newChatRequest(ctx context.Context, req domain.ChatRequest, stream bool) (*http.Request, error) {
	payloadStart := time.Now()
	if stream {
		req.Stream = true
		req.StreamOptions = &domain.StreamOptions{IncludeUsage: true}
	}

	payload, err := p.buildPayload(req)
	if err != nil {
		return nil, err
	}
	p.vlogf(2, "[github-copilot] payload marshal took=%s", time.Since(payloadStart))

	token, apiBase, err := p.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+pathChatCompletions, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	p.setClientHeaders(httpReq)

	return httpReq, nil
}

func (p *Provider) newResponsesRequest(ctx context.Context, req domain.ChatRequest) (*http.Request, error) {
	payload, err := p.buildResponsesPayload(req)
	if err != nil {
		return nil, err
	}

	token, apiBase, err := p.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+pathResponses, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	p.setClientHeaders(httpReq)

	return httpReq, nil
}

func (p *Provider) newNativeResponsesRequest(ctx context.Context, req domain.ResponsesRequest) (*http.Request, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	token, apiBase, err := p.getCopilotSession(ctx)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+pathResponses, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerAuthorization, tokenPrefixBearer+token)
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	p.setClientHeaders(httpReq)

	return httpReq, nil
}

func (p *Provider) buildPayload(req domain.ChatRequest) ([]byte, error) {
	messages := p.batchMessagesWithOversizedToolCalls(req.Messages)

	reasoningEffort := req.ReasoningEffort
	if reasoningEffort == "" && isReasoningModel(req.Model) {
		reasoningEffort = "high"
	}

	payload := copilotChatRequest{
		Model:               req.Model,
		Messages:            messages,
		Stream:              req.Stream,
		StreamOptions:       req.StreamOptions,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		MaxTokens:           req.MaxTokens,
		Stop:                req.Stop,
		N:                   req.N,
		Tools:               make([]copilotTool, 0, len(req.Tools)),
		ToolChoice:          req.ToolChoice,
		ReasoningEffort:     reasoningEffort,
		MaxCompletionTokens: req.MaxCompletionTokens,
	}

	if isNoTemperatureModel(req.Model) {
		payload.Temperature = nil
		payload.TopP = nil
	}

	for _, tool := range req.Tools {
		if tool.Function == nil {
			continue
		}
		payload.Tools = append(payload.Tools, copilotTool{
			Type: tool.Type,
			Function: copilotFunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  sanitizeParameters(tool.Function.Parameters),
			},
		})
	}

	return json.Marshal(payload)
}

func (p *Provider) buildResponsesPayload(req domain.ChatRequest) ([]byte, error) {
	messages := p.batchMessagesWithOversizedToolCalls(req.Messages)
	instructions := make([]string, 0)
	input := make([]responsesInputItem, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == domain.RoleSystem {
			if msg.Content != "" {
				instructions = append(instructions, msg.Content)
			}
			continue
		}
		if msg.Content == "" {
			continue
		}

		role := msg.Role
		if role == domain.RoleTool {
			role = domain.RoleUser
		}
		if role == "" {
			role = domain.RoleUser
		}

		contentType := "input_text"
		if role == domain.RoleAssistant {
			contentType = "output_text"
		}
		input = append(input, responsesInputItem{
			Role: role,
			Content: []responsesInputContent{
				{Type: contentType, Text: msg.Content},
			},
		})
	}

	if len(input) == 0 {
		input = append(input, responsesInputItem{
			Role: domain.RoleUser,
			Content: []responsesInputContent{
				{Type: "input_text", Text: ""},
			},
		})
	}

	reasoningEffort := req.ReasoningEffort
	if reasoningEffort == "" && isReasoningModel(req.Model) {
		reasoningEffort = domain.ReasoningEffortHigh
	}

	maxOutputTokens := req.MaxCompletionTokens
	if maxOutputTokens == nil {
		maxOutputTokens = req.MaxTokens
	}

	payload := responsesRequest{
		Model:           req.Model,
		Instructions:    strings.Join(instructions, "\n\n"),
		Input:           input,
		Stream:          true,
		Temperature:     nil,
		TopP:            nil,
		Tools:           make([]copilotTool, 0, len(req.Tools)),
		ToolChoice:      req.ToolChoice,
		Reasoning:       &responsesReasoning{Effort: reasoningEffort},
		MaxOutputTokens: maxOutputTokens,
	}
	if payload.Instructions == "" {
		payload.Instructions = "You are a helpful assistant."
	}
	if payload.Reasoning.Effort == "" {
		payload.Reasoning = nil
	}

	for _, tool := range req.Tools {
		if tool.Function == nil {
			continue
		}
		payload.Tools = append(payload.Tools, copilotTool{
			Type: tool.Type,
			Function: copilotFunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  sanitizeParameters(tool.Function.Parameters),
			},
		})
	}

	return json.Marshal(payload)
}

func (p *Provider) batchMessagesWithOversizedToolCalls(messages []domain.Message) []domain.Message {
	toolResponses := make(map[string]domain.Message)
	for _, msg := range messages {
		if msg.Role == domain.RoleTool && msg.ToolCallID != "" {
			toolResponses[msg.ToolCallID] = msg
		}
	}

	var result []domain.Message
	consumedToolMessages := make(map[string]bool)

	for _, msg := range messages {
		if msg.Role == domain.RoleTool && consumedToolMessages[msg.ToolCallID] {
			continue
		}

		if msg.Role != domain.RoleAssistant || len(msg.ToolCalls) <= maxToolCallsPerMessage {
			result = append(result, msg)
			continue
		}

		batches := chunkToolCalls(msg.ToolCalls, maxToolCallsPerMessage)
		p.vlogf(1, "[github-copilot] batching oversized tool_calls and interleaving: original=%d calls split into %d batches", len(msg.ToolCalls), len(batches))

		for batchIdx, batch := range batches {
			batchMsg := msg
			batchMsg.ToolCalls = batch

			if batchIdx > 0 {
				batchMsg.Content = ""
			}

			result = append(result, batchMsg)

			for _, tc := range batch {
				if resp, ok := toolResponses[tc.ID]; ok {
					result = append(result, resp)
					consumedToolMessages[tc.ID] = true
				}
			}
		}
	}

	return result
}

func chunkToolCalls(toolCalls []domain.ToolCall, maxSize int) [][]domain.ToolCall {
	if maxSize <= 0 {
		return nil
	}
	var chunks [][]domain.ToolCall
	for i := 0; i < len(toolCalls); i += maxSize {
		end := min(i+maxSize, len(toolCalls))
		chunks = append(chunks, toolCalls[i:end])
	}
	return chunks
}
