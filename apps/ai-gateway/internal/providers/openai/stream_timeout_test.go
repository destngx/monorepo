package openai

import (
	"testing"
	"time"
)

func TestNewUsesNoWholeStreamTimeout(t *testing.T) {
	provider := New("test-key", nil)

	if provider.client.Timeout != 120*time.Second {
		t.Fatalf("regular client timeout = %s, want %s", provider.client.Timeout, 120*time.Second)
	}
	if provider.streamClient.Timeout != 0 {
		t.Fatalf("stream client timeout = %s, want no whole-stream timeout", provider.streamClient.Timeout)
	}
}
