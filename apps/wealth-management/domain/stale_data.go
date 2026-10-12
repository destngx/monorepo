package domain

import (
	"context"
	"sync"
	"time"
)

// Stale-while-error (WM-NFR-4): when the sheet or every market provider
// fails, the engine serves the last good copy of the data and tells the
// client how old it is instead of failing the request.

// LastGoodTTLSeconds keeps last good copies for a week, long enough to ride
// out an outage or an expired OAuth token over a weekend.
const LastGoodTTLSeconds = 7 * 24 * 3600

// LastGoodKey is the cache key of the last good copy of key. The "stale:"
// prefix keeps it out of the invalidation patterns, so writes and sync drop
// the fresh copy but never the fallback.
func LastGoodKey(key string) string {
	return "stale:" + key
}

// LastGood is a cached value with the time it was read from its source.
type LastGood[T any] struct {
	AsOf  time.Time `json:"asOf"`
	Value T         `json:"value"`
}

// StaleRecorder collects, for one request, the oldest data time served
// from a last good copy. It is safe for concurrent use.
type StaleRecorder struct {
	mu    sync.Mutex
	asOf  time.Time
	stale bool
}

// Mark records that data as of asOf was served stale; the oldest time wins.
func (r *StaleRecorder) Mark(asOf time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.stale || asOf.Before(r.asOf) {
		r.asOf = asOf
	}
	r.stale = true
}

// AsOf returns the oldest stale data time, if any data was served stale.
func (r *StaleRecorder) AsOf() (time.Time, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.asOf, r.stale
}

type staleRecorderContextKey struct{}

func WithStaleRecorder(ctx context.Context, recorder *StaleRecorder) context.Context {
	return context.WithValue(ctx, staleRecorderContextKey{}, recorder)
}

// MarkStale records stale data on the request's recorder; without one
// (background work, MCP) it does nothing.
func MarkStale(ctx context.Context, asOf time.Time) {
	if recorder, ok := ctx.Value(staleRecorderContextKey{}).(*StaleRecorder); ok && recorder != nil {
		recorder.Mark(asOf)
	}
}
