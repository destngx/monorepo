package bedrock

import (
	"apps/ai-gateway/internal/domain"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

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
