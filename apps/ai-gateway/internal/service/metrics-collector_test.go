package service

import (
	"apps/ai-gateway/internal/domain"
	"os"
	"testing"
	"time"
)

func TestMetricsCollector_RecordAndSummary(t *testing.T) {
	collector := NewMetricsCollector(10, "", 0)

	r1 := domain.RequestRecord{
		Timestamp:  time.Now(),
		Route:      "/v1/chat/completions",
		Provider:   "openai",
		Model:      "gpt-4",
		StatusCode: 200,
		DurationMs: 100,
		Usage:      domain.Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
	}
	collector.Record(r1)

	r2 := domain.RequestRecord{
		Timestamp:  time.Now(),
		Route:      "/v1/chat/completions",
		Provider:   "anthropic",
		Model:      "claude-3",
		StatusCode: 500,
		DurationMs: 200,
		Usage:      domain.Usage{PromptTokens: 5, CompletionTokens: 5, TotalTokens: 10},
	}
	collector.Record(r2)

	summary := collector.Summary()

	if summary.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", summary.TotalRequests)
	}
	if summary.TotalErrors != 1 {
		t.Errorf("Expected 1 error, got %d", summary.TotalErrors)
	}
	if summary.TotalTokens != 40 {
		t.Errorf("Expected 40 tokens, got %d", summary.TotalTokens)
	}
	if summary.AvgDurationMs != 150 {
		t.Errorf("Expected 150ms avg duration, got %f", summary.AvgDurationMs)
	}

	// Check ByProvider
	openai := summary.ByProvider["openai"]
	if openai == nil || openai.Count != 1 {
		t.Errorf("Expected openai stats, got %+v", openai)
	}
	if openai.TotalTokens != 30 {
		t.Errorf("Expected 30 tokens for openai, got %d", openai.TotalTokens)
	}

	anthro := summary.ByProvider["anthropic"]
	if anthro == nil || anthro.Count != 1 {
		t.Errorf("Expected anthropic stats, got %+v", anthro)
	}
	if anthro.Errors != 1 {
		t.Errorf("Expected 1 error for anthropic, got %d", anthro.Errors)
	}

	// Check recent requests
	if len(summary.RecentRequests) != 2 {
		t.Errorf("Expected 2 recent requests, got %d", len(summary.RecentRequests))
	}
}

func TestMetricsCollector_RingBuffer(t *testing.T) {
	size := 5
	collector := NewMetricsCollector(size, "", 0)

	for i := 0; i < 10; i++ {
		collector.Record(domain.RequestRecord{
			StatusCode: 200,
			DurationMs: int64(i),
		})
	}

	summary := collector.Summary()
	if len(summary.RecentRequests) != 5 {
		t.Errorf("Expected 5 recent requests (buffer size), got %d", len(summary.RecentRequests))
	}
	// Latency of newest should be 9
	if summary.RecentRequests[0].DurationMs != 9 {
		t.Errorf("Expected newest request duration 9, got %d", summary.RecentRequests[0].DurationMs)
	}
}

func TestMetricsCollector_Persistence(t *testing.T) {
	path := "test_metrics.json"
	defer os.Remove(path)

	collector := NewMetricsCollector(10, path, 0)
	collector.Record(domain.RequestRecord{
		Route:      "/test",
		Provider:   "test",
		StatusCode: 200,
		DurationMs: 42,
		Usage:      domain.Usage{TotalTokens: 100},
	})

	collector.Flush()

	// New collector to load
	collector2 := NewMetricsCollector(10, path, 0)
	collector2.LoadFromDisk()

	summary := collector2.Summary()
	if summary.TotalRequests != 1 {
		t.Errorf("Expected 1 request after load, got %d", summary.TotalRequests)
	}
	if summary.TotalTokens != 100 {
		t.Errorf("Expected 100 tokens, got %d", summary.TotalTokens)
	}
	if len(summary.RecentRequests) != 1 {
		t.Errorf("Expected 1 recent request, got %d", len(summary.RecentRequests))
	}
	if summary.RecentRequests[0].DurationMs != 42 {
		t.Errorf("Expected duration 42, got %d", summary.RecentRequests[0].DurationMs)
	}
}
