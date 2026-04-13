package proxy

import (
	"testing"
)

func BenchmarkModelMapper_Resolve(b *testing.B) {
	m := NewModelMapper()
	model := "claude-3-5-sonnet-20241022"
	provider := "github-copilot"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Resolve(provider, model)
	}
}

func BenchmarkModelMapper_ResolvePrefix(b *testing.B) {
	m := NewModelMapper()
	model := "claude-haiku-very-long-suffix-to-test-performance"
	provider := "github-copilot"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Resolve(provider, model)
	}
}
