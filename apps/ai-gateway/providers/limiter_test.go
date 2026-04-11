package providers

import (
	"context"
	"testing"

	"apps/ai-gateway/types"
)

type MockProvider struct {
	Provider // Embed to fulfill interface easily
	name     string
}

func (m *MockProvider) Name() string { return m.name }
func (m *MockProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	return &types.ChatResponse{}, nil
}

func TestRateLimitedProvider(t *testing.T) {
	mock := &MockProvider{name: "mock"}
	// Set a very low limit for testing: 60 RPM = 1 request per second, 1 burst
	lp := NewRateLimitedProvider(mock, 60, 1)

	t.Run("Allows first request", func(t *testing.T) {
		_, err := lp.Chat(context.Background(), types.ChatRequest{})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Blocks request exceeding burst", func(t *testing.T) {
		// NewRateLimitedProvider with 1 RPM, 1 burst
		tinyLP := NewRateLimitedProvider(mock, 1, 1)

		// First one allowed
		_, err := tinyLP.Chat(context.Background(), types.ChatRequest{})
		if err != nil {
			t.Fatalf("first request should be allowed")
		}

		// Second one immediate should be blocked
		_, err = tinyLP.Chat(context.Background(), types.ChatRequest{})
		if err == nil {
			t.Errorf("expected rate limit error, got nil")
		}

		if _, ok := err.(*ErrRateLimitExceeded); !ok {
			t.Errorf("expected ErrRateLimitExceeded, got %T", err)
		}
	})
}
