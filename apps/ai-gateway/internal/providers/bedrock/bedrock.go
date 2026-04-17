package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

const (
	roleSystem    = "system"
	roleUser      = "user"
	roleAssistant = "assistant"
	roleTool      = "tool"

	objectChatCompletion      = "chat.completion"
	objectChatCompletionChunk = "chat.completion.chunk"

	sseDataPrefix = "data: "
	sseDone       = "[DONE]"
)

type Provider struct {
	client *bedrockruntime.Client
	region string
	ready  bool
}

func New(ctx context.Context, region string) (*Provider, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)
	return &Provider{
		client: client,
		region: region,
	}, nil
}

func (p *Provider) Name() string { return domain.ProviderBedrock }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	req = shared.NormalizeTools(req)
	input, err := p.convertToBedrockRequest(req)
	if err != nil {
		return nil, err
	}

	output, err := p.client.Converse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("bedrock converse error: %w", err)
	}

	return p.convertToOpenAIResponse(output, req.Model), nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	req = shared.NormalizeTools(req)
	input, err := p.convertToBedrockStreamRequest(req)
	if err != nil {
		return domain.Usage{}, err
	}

	output, err := p.client.ConverseStream(ctx, input)
	if err != nil {
		return domain.Usage{}, fmt.Errorf("bedrock converse stream error: %w", err)
	}

	return p.processStream(output, w, req.Model)
}

func (p *Provider) convertToBedrockRequest(req domain.ChatRequest) (*bedrockruntime.ConverseInput, error) {
	messages, system, err := p.mapMessages(req.Messages)
	if err != nil {
		return nil, err
	}

	input := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(req.Model),
		Messages: messages,
		System:   system,
	}

	if req.Temperature != nil {
		input.InferenceConfig = &types.InferenceConfiguration{
			Temperature: aws.Float32(float32(*req.Temperature)),
		}
	}
	if req.TopP != nil {
		if input.InferenceConfig == nil {
			input.InferenceConfig = &types.InferenceConfiguration{}
		}
		input.InferenceConfig.TopP = aws.Float32(float32(*req.TopP))
	}
	if req.MaxTokens != nil {
		if input.InferenceConfig == nil {
			input.InferenceConfig = &types.InferenceConfiguration{}
		}
		input.InferenceConfig.MaxTokens = aws.Int32(int32(*req.MaxTokens))
	}

	if len(req.Tools) > 0 {
		toolConfig := &types.ToolConfiguration{}
		for _, t := range req.Tools {
			if t.Type == "function" {
				// Bedrock Converse API uses ToolMemberToolSpec
				toolConfig.Tools = append(toolConfig.Tools, &types.ToolMemberToolSpec{
					Value: types.ToolSpecification{
						Name:        aws.String(t.Function.Name),
						Description: aws.String(t.Function.Description),
						InputSchema: &types.ToolInputSchemaMemberJson{
							Value: document.NewLazyDocument(t.Function.Parameters),
						},
					},
				})
			}
		}
		input.ToolConfig = toolConfig
	}

	return input, nil
}

func (p *Provider) convertToBedrockStreamRequest(req domain.ChatRequest) (*bedrockruntime.ConverseStreamInput, error) {
	messages, system, err := p.mapMessages(req.Messages)
	if err != nil {
		return nil, err
	}

	input := &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(req.Model),
		Messages: messages,
		System:   system,
	}

	if req.Temperature != nil {
		input.InferenceConfig = &types.InferenceConfiguration{
			Temperature: aws.Float32(float32(*req.Temperature)),
		}
	}
	if req.TopP != nil {
		if input.InferenceConfig == nil {
			input.InferenceConfig = &types.InferenceConfiguration{}
		}
		input.InferenceConfig.TopP = aws.Float32(float32(*req.TopP))
	}
	if req.MaxTokens != nil {
		if input.InferenceConfig == nil {
			input.InferenceConfig = &types.InferenceConfiguration{}
		}
		input.InferenceConfig.MaxTokens = aws.Int32(int32(*req.MaxTokens))
	}

	if len(req.Tools) > 0 {
		toolConfig := &types.ToolConfiguration{}
		for _, t := range req.Tools {
			if t.Type == "function" {
				toolConfig.Tools = append(toolConfig.Tools, &types.ToolMemberToolSpec{
					Value: types.ToolSpecification{
						Name:        aws.String(t.Function.Name),
						Description: aws.String(t.Function.Description),
						InputSchema: &types.ToolInputSchemaMemberJson{
							Value: document.NewLazyDocument(t.Function.Parameters),
						},
					},
				})
			}
		}
		input.ToolConfig = toolConfig
	}

	return input, nil
}

func (p *Provider) mapMessages(inMessages []domain.Message) ([]types.Message, []types.SystemContentBlock, error) {
	var messages []types.Message
	var systemBlocks []types.SystemContentBlock

	for _, m := range inMessages {
		switch m.Role {
		case roleSystem:
			systemBlocks = append(systemBlocks, &types.SystemContentBlockMemberText{
				Value: m.Content,
			})
		case roleUser:
			messages = append(messages, types.Message{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: m.Content},
				},
			})
		case roleAssistant:
			var content []types.ContentBlock
			if m.Content != "" {
				content = append(content, &types.ContentBlockMemberText{Value: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
					return nil, nil, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
				}
				content = append(content, &types.ContentBlockMemberToolUse{
					Value: types.ToolUseBlock{
						ToolUseId: aws.String(tc.ID),
						Name:      aws.String(tc.Function.Name),
						Input:     document.NewLazyDocument(input),
					},
				})
			}
			messages = append(messages, types.Message{
				Role:    types.ConversationRoleAssistant,
				Content: content,
			})
		case roleTool:
			messages = append(messages, types.Message{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberToolResult{
						Value: types.ToolResultBlock{
							ToolUseId: aws.String(m.ToolCallID),
							Content: []types.ToolResultContentBlock{
								&types.ToolResultContentBlockMemberText{Value: m.Content},
							},
						},
					},
				},
			})
		}
	}

	return messages, systemBlocks, nil
}

func (p *Provider) convertToOpenAIResponse(output *bedrockruntime.ConverseOutput, model string) *domain.ChatResponse {
	var content string
	var toolCalls []domain.ToolCall

	if msg, ok := output.Output.(*types.ConverseOutputMemberMessage); ok {
		for _, b := range msg.Value.Content {
			if t, ok := b.(*types.ContentBlockMemberText); ok {
				content += t.Value
			} else if tu, ok := b.(*types.ContentBlockMemberToolUse); ok {
				inputBytes, _ := json.Marshal(tu.Value.Input)
				toolCalls = append(toolCalls, domain.ToolCall{
					ID:   *tu.Value.ToolUseId,
					Type: "function",
					Function: &domain.FunctionCall{
						Name:      *tu.Value.Name,
						Arguments: string(inputBytes),
					},
				})
			}
		}
	}

	finishReason := "stop"
	switch output.StopReason {
	case types.StopReasonMaxTokens:
		finishReason = "length"
	case types.StopReasonToolUse:
		finishReason = "tool_calls"
	}

	usage := domain.Usage{}
	if output.Usage != nil {
		usage.PromptTokens = int(*output.Usage.InputTokens)
		usage.CompletionTokens = int(*output.Usage.OutputTokens)
		usage.TotalTokens = int(*output.Usage.TotalTokens)
	}

	return &domain.ChatResponse{
		ID:     fmt.Sprintf("br-%d", time.Now().Unix()),
		Object: objectChatCompletion,
		Model:  model,
		Choices: []domain.Choice{{
			Index: 0,
			Message: domain.Message{
				Role:      roleAssistant,
				Content:   content,
				ToolCalls: toolCalls,
			},
			FinishReason: finishReason,
		}},
		Usage: usage,
	}
}

func (p *Provider) processStream(output *bedrockruntime.ConverseStreamOutput, w io.Writer, model string) (domain.Usage, error) {
	var usage domain.Usage
	stream := output.GetStream()
	defer stream.Close()

	id := fmt.Sprintf("brs-%d", time.Now().Unix())

	for event := range stream.Events() {
		switch e := event.(type) {
		case *types.ConverseStreamOutputMemberMessageStart:
			// Usually role assistant
		case *types.ConverseStreamOutputMemberContentBlockDelta:
			if d, ok := e.Value.Delta.(*types.ContentBlockDeltaMemberText); ok {
				p.sendDelta(w, id, model, d.Value, nil)
			}
		case *types.ConverseStreamOutputMemberMessageStop:
			// Done
		case *types.ConverseStreamOutputMemberMetadata:
			if e.Value.Usage != nil {
				usage.PromptTokens = int(*e.Value.Usage.InputTokens)
				usage.CompletionTokens = int(*e.Value.Usage.OutputTokens)
				usage.TotalTokens = int(*e.Value.Usage.TotalTokens)
			}
		}
	}

	io.WriteString(w, sseDataPrefix+sseDone+"\n\n")
	return usage, nil
}

func (p *Provider) sendDelta(w io.Writer, id, model, content string, toolCalls []map[string]interface{}) {
	chunk := map[string]interface{}{
		"id":     id,
		"object": objectChatCompletionChunk,
		"model":  model,
		"choices": []map[string]interface{}{{
			"index": 0,
			"delta": map[string]interface{}{
				"content": content,
			},
		}},
	}
	if toolCalls != nil {
		chunk["choices"].([]map[string]interface{})[0]["delta"].(map[string]interface{})["tool_calls"] = toolCalls
	}

	b, _ := json.Marshal(chunk)
	io.WriteString(w, sseDataPrefix+string(b)+"\n\n")
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	knownModels := []domain.ModelInfo{
		{ID: "anthropic.claude-3-5-sonnet-20240620-v1:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "anthropic.claude-3-5-sonnet-20241022-v2:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "anthropic.claude-3-opus-20240229-v1:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "anthropic.claude-3-haiku-20240307-v1:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "meta.llama3-70b-instruct-v1:0", Object: "model", OwnedBy: "meta"},
	}
	return &domain.ModelsResponse{
		Object: "list",
		Data:   knownModels,
	}, nil
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not implemented for bedrock in this gateway yet")
}

func (p *Provider) IsConfigured() bool {
	return p.client != nil
}

func (p *Provider) Ping(ctx context.Context) error {
	_, err := p.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String("anthropic.claude-3-haiku-20240307-v1:0"),
		Messages: []types.Message{
			{
				Role: types.ConversationRoleUser,
				Content: []types.ContentBlock{
					&types.ContentBlockMemberText{Value: "ping"},
				},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(1),
		},
	})
	if err != nil && !strings.Contains(err.Error(), "AccessDenied") {
		if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "timeout") {
			return err
		}
	}
	return nil
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage not supported by bedrock")
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
