package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchMessagesWithOversizedToolCalls(t *testing.T) {
	p := &Provider{verbose: 1}

	t.Run("UnderLimit", func(t *testing.T) {
		msgs := []domain.Message{
			{Role: domain.RoleUser, Content: "hi"},
			{Role: domain.RoleAssistant, ToolCalls: []domain.ToolCall{{ID: "1"}}},
			{Role: domain.RoleTool, ToolCallID: "1", Content: "res"},
		}
		result := p.batchMessagesWithOversizedToolCalls(msgs)
		assert.Equal(t, msgs, result)
	})

	t.Run("OverLimitWithResponses", func(t *testing.T) {
		// Mock max limit to 2 for easier testing
		// We'll temporarily override the constant if possible, but it's a const.
		// Since it's a constant, we'll have to use the real limit or
		// we can make it a variable in the provider for testing.
		// For now, let's assume we use a large number but for test we'll use a small one if we can.
		// Wait, maxToolCallsPerMessage is a constant 128.

		toolCalls := make([]domain.ToolCall, 130)
		toolMsgs := make([]domain.Message, 130)
		for i := 0; i < 130; i++ {
			id := fmt.Sprintf("call_%d", i)
			toolCalls[i] = domain.ToolCall{ID: id, Type: "function"}
			toolMsgs[i] = domain.Message{Role: domain.RoleTool, ToolCallID: id, Content: fmt.Sprintf("res_%d", i)}
		}

		input := append([]domain.Message{
			{Role: domain.RoleUser, Content: "complex task"},
			{Role: domain.RoleAssistant, Content: "thinking", ToolCalls: toolCalls},
		}, toolMsgs...)

		result := p.batchMessagesWithOversizedToolCalls(input)

		// Expected:
		// 1. User
		// 2. Assistant (1-128)
		// 3. Tool (1-128)
		// 4. Assistant (129-130)
		// 5. Tool (129-130)

		assert.Greater(t, len(result), len(input))

		// Check interleaving
		assert.Equal(t, domain.RoleAssistant, result[1].Role)
		assert.Equal(t, 128, len(result[1].ToolCalls))
		assert.Equal(t, "thinking", result[1].Content)

		assert.Equal(t, domain.RoleTool, result[2].Role)
		assert.Equal(t, "call_0", result[2].ToolCallID)

		assert.Equal(t, domain.RoleTool, result[129].Role)
		assert.Equal(t, "call_127", result[129].ToolCallID)

		assert.Equal(t, domain.RoleAssistant, result[130].Role)
		assert.Equal(t, 2, len(result[130].ToolCalls))
		assert.Equal(t, "", result[130].Content) // Content should be empty in subsequent batches

		assert.Equal(t, domain.RoleTool, result[131].Role)
		assert.Equal(t, "call_128", result[131].ToolCallID)
	})

	t.Run("OverLimitAtEnd", func(t *testing.T) {
		toolCalls := make([]domain.ToolCall, 130)
		for i := 0; i < 130; i++ {
			toolCalls[i] = domain.ToolCall{ID: fmt.Sprintf("call_%d", i)}
		}
		input := []domain.Message{
			{Role: domain.RoleAssistant, ToolCalls: toolCalls},
		}
		result := p.batchMessagesWithOversizedToolCalls(input)

		// Should still split even if no responses
		assert.Equal(t, 2, len(result))
		assert.Equal(t, 128, len(result[0].ToolCalls))
		assert.Equal(t, 2, len(result[1].ToolCalls))
	})
}
