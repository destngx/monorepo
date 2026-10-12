package ai

import (
	"apps/wealth-management/domain"
	"fmt"
	"strings"
)

// prompts is the registry view used while building one request.
type prompts struct {
	content domain.AIContent
	used    []string
}

func (s *Service) loadPrompts() (*prompts, error) {
	content, err := s.db.LoadAIContent(false)
	if err != nil {
		return nil, fmt.Errorf("load prompts: %w", err)
	}
	return &prompts{content: content}, nil
}

// render returns a required prompt with placeholders filled (WM-AI-1).
func (p *prompts) render(key string, vars map[string]string) (string, error) {
	text, ok := p.lookup(key, vars)
	if !ok {
		return "", fmt.Errorf("%w: %s", domain.ErrPromptMissing, key)
	}
	return text, nil
}

// lookup returns an optional prompt.
func (p *prompts) lookup(key string, vars map[string]string) (string, bool) {
	text := strings.TrimSpace(p.content.Prompts[key])
	if text == "" {
		return "", false
	}
	p.used = append(p.used, key)
	return domain.RenderPrompt(text, vars), true
}
