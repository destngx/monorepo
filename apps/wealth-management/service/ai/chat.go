package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"strings"
	"time"
)

// chatMode frames the assistant for conversation; engine code like the directive.
const chatMode = `## CHAT MODE
You are chatting with the owner of this ledger. Answer in concise markdown. Use the tools to fetch figures you do not have in the snapshot; never guess ledger numbers.`

const (
	defaultToolTransactions = 20
	maxToolTransactions     = 100
)

// Chat runs a tool-using conversation and streams events (WM-AI-11).
func (s *Service) Chat(ctx context.Context, request port.ChatRequest, emit func(port.ChatEvent) error) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if err := s.requireConfigured(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, chatTimeout)
	defer cancel()

	snap, err := s.loadSnapshot()
	if err != nil {
		return err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return err
	}
	task := chatMode
	if extra, ok := p.lookup(keyChatTask, nil); ok {
		task += "\n\n" + extra
	}
	if page := request.Page; page != nil && strings.TrimSpace(page.Path) != "" {
		task += fmt.Sprintf("\n\n## CURRENT PAGE\nThe user is looking at %s", page.Path)
		if page.Title != "" {
			task += " (" + s.masker.Mask(page.Title) + ")"
		}
		if page.Summary != "" {
			task += ":\n" + s.masker.Mask(page.Summary)
		}
	}
	system, err := s.systemPrompt(p, snap, task)
	if err != nil {
		return err
	}

	history := request.Messages
	if len(history) > maxChatHistory {
		history = history[len(history)-maxChatHistory:]
	}
	messages := []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}}
	for _, message := range history {
		messages = append(messages, domain.AIMessage{Role: domain.AIRole(message.Role), Content: message.Content})
	}
	tools := s.chatTools()
	definitions := make([]domain.AITool, 0, len(tools))
	for _, tool := range tools {
		definitions = append(definitions, tool.definition)
	}

	for step := 0; step < maxChatSteps; step++ {
		lastStep := step == maxChatSteps-1
		turn := domain.AIRequest{Messages: messages}
		if !lastStep {
			turn.Tools = definitions
		}
		response, err := s.streamStep(ctx, turn, p.used, request.Model, emit)
		if err != nil {
			return err
		}
		if len(response.ToolCalls) == 0 || lastStep {
			return emit(port.ChatEvent{Type: "done", Model: response.Model})
		}
		messages = append(messages, domain.AIMessage{Role: domain.AIRoleAssistant, Content: response.Content, ToolCalls: response.ToolCalls})
		for _, toolCall := range response.ToolCalls {
			if err := emit(port.ChatEvent{Type: "tool", Tool: toolCall.Name, Status: "running"}); err != nil {
				return err
			}
			raw, result, toolErr := s.runTool(ctx, tools, toolCall)
			status := "ok"
			if toolErr != nil {
				status = "error"
				s.log.LogError(ctx, "ai chat tool failed", toolErr)
				result = toJSON(map[string]string{"error": toolMessage(toolErr)})
			}
			if err := emit(port.ChatEvent{Type: "tool", Tool: toolCall.Name, Status: status}); err != nil {
				return err
			}
			if form, ok := raw.(domain.FillChatForm); ok {
				if err := emit(port.ChatEvent{Type: "form", Form: &form}); err != nil {
					return err
				}
			}
			messages = append(messages, domain.AIMessage{Role: domain.AIRoleTool, ToolCallID: toolCall.ID, Content: result})
		}
	}
	return nil
}

// streamStep streams one model turn. The fallback model is tried only when
// the primary fails before any text reached the user.
func (s *Service) streamStep(ctx context.Context, request domain.AIRequest, promptKeys []string, override string, emit func(port.ChatEvent) error) (domain.AIResponse, error) {
	var lastErr error
	for _, model := range s.models(domain.AIUseCaseChat, override) {
		request.Model = model
		request.Effort = classTyping.effort
		streamed := false
		var emitErr error
		if err := s.gate.acquire(ctx, classTyping.priority); err != nil {
			return domain.AIResponse{}, err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
		started := time.Now()
		response, err := s.client.Stream(attemptCtx, request, func(delta string) error {
			streamed = true
			emitErr = emit(port.ChatEvent{Type: "text", Delta: delta})
			return emitErr
		})
		cancel()
		s.gate.release()
		s.logCall(ctx, "chat", model, promptKeys, time.Since(started), response.Usage, err)
		if emitErr != nil {
			return domain.AIResponse{}, emitErr
		}
		if err == nil {
			if response.Model == "" {
				response.Model = model
			}
			return response, nil
		}
		lastErr = err
		if streamed || ctx.Err() != nil {
			break
		}
	}
	return domain.AIResponse{}, lastErr
}

func toolMessage(err error) string {
	if IsUserFacing(err) {
		return err.Error()
	}
	return "the tool failed; tell the user the data is unavailable"
}
