package ai

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"strings"
	"sync"
)

// scriptedClient answers each call with the next scripted reply.
type scriptedClient struct {
	mu       sync.Mutex
	replies  []reply
	requests []domain.AIRequest
	// route, when set, answers requests whose order is not fixed.
	route func(request domain.AIRequest) reply
}

type reply struct {
	content   string
	toolCalls []domain.AIToolCall
	err       error
}

func (c *scriptedClient) next(request domain.AIRequest) (reply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, request)
	if c.route != nil {
		return c.route(request), nil
	}
	if len(c.replies) == 0 {
		return reply{}, errors.New("no scripted reply left")
	}
	r := c.replies[0]
	c.replies = c.replies[1:]
	return r, nil
}

func (c *scriptedClient) Complete(_ context.Context, request domain.AIRequest) (domain.AIResponse, error) {
	r, err := c.next(request)
	if err != nil {
		return domain.AIResponse{}, err
	}
	if r.err != nil {
		return domain.AIResponse{}, r.err
	}
	return domain.AIResponse{Model: request.Model, Content: r.content, ToolCalls: r.toolCalls}, nil
}

func (c *scriptedClient) Stream(_ context.Context, request domain.AIRequest, onDelta func(string) error) (domain.AIResponse, error) {
	r, err := c.next(request)
	if err != nil {
		return domain.AIResponse{}, err
	}
	if r.err != nil {
		return domain.AIResponse{}, r.err
	}
	for _, word := range strings.SplitAfter(r.content, " ") {
		if word == "" {
			continue
		}
		if err := onDelta(word); err != nil {
			return domain.AIResponse{}, err
		}
	}
	return domain.AIResponse{Model: request.Model, Content: r.content, ToolCalls: r.toolCalls}, nil
}

func (c *scriptedClient) Ping(context.Context) error { return nil }

func (c *scriptedClient) models() []string {
	out := []string{}
	for _, request := range c.requests {
		out = append(out, request.Model)
	}
	return out
}

type memoryCache struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *memoryCache) Set(_ context.Context, key string, value string, _ int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[key] = value
	return nil
}

func (m *memoryCache) Get(_ context.Context, key string) (domain.CacheEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.values[key]
	return domain.CacheEntry{Key: key, Value: value, Found: ok}, nil
}

func (m *memoryCache) Invalidate(context.Context, string) (int, error) { return 0, nil }
func (m *memoryCache) Ping(context.Context) error                      { return nil }
