package openai

import (
	"context"
	"io"

	"apps/ai-gateway/internal/domain"
)

type MockTestProvider struct {
	name               string
	chatCallCount      int
	responsesCallCount int
	lastChatModel      string
	lastResponsesModel string
	responsesStreamErr error
}

func (m *MockTestProvider) Name() string { return m.name }
func (m *MockTestProvider) Chat(_ context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	m.chatCallCount++
	m.lastChatModel = req.Model
	return &domain.ChatResponse{}, nil
}
func (m *MockTestProvider) ChatStream(context.Context, domain.ChatRequest, io.Writer) (domain.Usage, error) {
	return domain.Usage{}, nil
}
func (m *MockTestProvider) Responses(_ context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	m.responsesCallCount++
	m.lastResponsesModel = req.Model
	return &domain.ResponsesResponse{}, nil
}
func (m *MockTestProvider) ResponsesStream(context.Context, domain.ResponsesRequest, io.Writer) (domain.Usage, error) {
	if m.responsesStreamErr != nil {
		return domain.Usage{}, m.responsesStreamErr
	}
	return domain.Usage{}, nil
}
func (m *MockTestProvider) ListModels(context.Context) (*domain.ModelsResponse, error) {
	return &domain.ModelsResponse{Object: "list", Data: []domain.ModelInfo{{ID: "mock-model", Object: "model", OwnedBy: m.name}}}, nil
}
func (m *MockTestProvider) Embeddings(context.Context, domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, nil
}
func (m *MockTestProvider) IsConfigured() bool                 { return true }
func (m *MockTestProvider) Ping(context.Context) error         { return nil }
func (m *MockTestProvider) Usage(context.Context) (any, error) { return nil, nil }
func (m *MockTestProvider) IsReady() bool                      { return true }
func (m *MockTestProvider) SetReady(bool)                      {}
