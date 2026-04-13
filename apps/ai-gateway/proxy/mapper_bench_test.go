package proxy

import (
	"testing"
)

func BenchmarkModelMapper_Resolve(b *testing.B) {
	m := NewModelMapper()
	model := "claude-3-5-sonnet-20241022"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Resolve(model)
	}
}

func BenchmarkModelMapper_ResolvePrefix(b *testing.B) {
	m := NewModelMapper()
	model := "claude-haiku-very-long-suffix-to-test-performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Resolve(model)
	}
}
