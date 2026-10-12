package google_sheets

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultReadsPerMinute is the Sheets API default read quota per user
// (WM-NFR-3).
const DefaultReadsPerMinute = 60

// tokenBucket paces requests to a per-minute rate. Callers wait for their
// turn rather than fail: each Wait reserves a token, possibly ahead of the
// refill, and sleeps until it is due, so waiters are served in order.
type tokenBucket struct {
	mu        sync.Mutex
	capacity  float64
	tokens    float64
	perSecond float64
	last      time.Time
	now       func() time.Time
	sleep     func(ctx context.Context, delay time.Duration) error
}

// newTokenBucket allows a burst of a sixth of the per-minute rate, so any
// 60 s window stays close to the quota while short bursts (a page loading
// several datasets) are not delayed.
func newTokenBucket(perMinute int) *tokenBucket {
	perMinute = max(perMinute, 1)
	capacity := max(float64(perMinute)/6, 1)
	return &tokenBucket{
		capacity:  capacity,
		tokens:    capacity,
		perSecond: float64(perMinute) / 60,
		now:       time.Now,
		sleep:     sleepContext,
	}
}

// Wait blocks until a request may be sent or ctx ends. A cancelled wait
// hands its reservation back.
func (b *tokenBucket) Wait(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("waiting for the sheets read quota: %w", err)
	}
	b.mu.Lock()
	now := b.now()
	if !b.last.IsZero() {
		b.tokens = min(b.capacity, b.tokens+now.Sub(b.last).Seconds()*b.perSecond)
	}
	b.last = now
	b.tokens--
	delay := time.Duration(0)
	if b.tokens < 0 {
		delay = time.Duration(-b.tokens / b.perSecond * float64(time.Second))
	}
	b.mu.Unlock()
	if delay == 0 {
		return nil
	}
	if err := b.sleep(ctx, delay); err != nil {
		b.mu.Lock()
		b.tokens++
		b.mu.Unlock()
		return fmt.Errorf("waiting for the sheets read quota: %w", err)
	}
	return nil
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
