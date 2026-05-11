package bedrock

import (
	"apps/ai-gateway/internal/domain"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

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
