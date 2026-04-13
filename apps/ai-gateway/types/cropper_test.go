package types

import (
	"strings"
	"testing"
)

func TestCropRequest(t *testing.T) {
	tests := []struct {
		name       string
		messages   []Message
		maxTokens  int
		wantCount  int
		mustExist  []string // content patterns that must exist
		mustAbsent []string // content patterns that must be gone
	}{
		{
			name: "Small request - no change",
			messages: []Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "hi"},
			},
			maxTokens: 8000,
			wantCount: 2,
		},
		{
			name: "Large request - prune middle",
			messages: []Message{
				{Role: "system", Content: "Instruction"},
				{Role: "user", Content: strings.Repeat("a", 10000)},      // ~2500 tokens
				{Role: "assistant", Content: strings.Repeat("b", 20000)}, // ~5000 tokens
				{Role: "user", Content: "Now answer this small query"},   // latest
			},
			maxTokens:  5000, // Safe limit ~17500 chars. Total ~30000.
			wantCount:  2,
			mustExist:  []string{"system", "small query"},
			mustAbsent: []string{"aaaaa", "bbbbb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChatRequest{Messages: tt.messages}
			cropped := CropRequest(req, tt.maxTokens)

			if len(cropped.Messages) != tt.wantCount && tt.wantCount != 0 {
				t.Errorf("len(Messages) = %v, want %v", len(cropped.Messages), tt.wantCount)
			}

			// Verify mandatory existence
			fullContent := ""
			for _, m := range cropped.Messages {
				fullContent += m.Role + m.Content
			}

			for _, pattern := range tt.mustExist {
				if !strings.Contains(fullContent, pattern) {
					t.Errorf("expected pattern %q not found in cropped request", pattern)
				}
			}

			for _, pattern := range tt.mustAbsent {
				if strings.Contains(fullContent, pattern) {
					t.Errorf("prohibited pattern %q found in cropped request", pattern)
				}
			}
		})
	}
}
