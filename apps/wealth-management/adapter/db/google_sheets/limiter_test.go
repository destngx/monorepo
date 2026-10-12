package google_sheets

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeClock advances only when the bucket sleeps.
type fakeClock struct {
	now    time.Time
	sleeps []time.Duration
	err    error
}

func (c *fakeClock) bucket(perMinute int) *tokenBucket {
	bucket := newTokenBucket(perMinute)
	bucket.now = func() time.Time { return c.now }
	bucket.sleep = func(_ context.Context, delay time.Duration) error {
		c.sleeps = append(c.sleeps, delay)
		if c.err != nil {
			return c.err
		}
		c.now = c.now.Add(delay)
		return nil
	}
	return bucket
}

func TestGivenTheDefaultQuotaWhenABurstArrivesThenTheFirstTenPassAndTheRestArePacedOnePerSecond(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	bucket := clock.bucket(DefaultReadsPerMinute)

	for range 13 {
		if err := bucket.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if len(clock.sleeps) != 3 {
		t.Fatalf("sleeps = %v, want 3 paced requests", clock.sleeps)
	}
	for _, sleep := range clock.sleeps {
		if sleep < 990*time.Millisecond || sleep > 1010*time.Millisecond {
			t.Fatalf("sleeps = %v, want about 1s each", clock.sleeps)
		}
	}
}

func TestGivenAnIdleMinuteWhenRequestsArriveThenTheBurstIsAvailableAgain(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC)}
	bucket := clock.bucket(DefaultReadsPerMinute)
	for range 10 {
		_ = bucket.Wait(context.Background())
	}

	clock.now = clock.now.Add(time.Minute)
	for range 10 {
		if err := bucket.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(clock.sleeps) != 0 {
		t.Fatalf("sleeps = %v, want none after refill", clock.sleeps)
	}
}

func TestGivenACancelledContextWhenWaitingThenItFailsAndReturnsTheReservation(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 13, 9, 0, 0, 0, time.UTC), err: context.Canceled}
	bucket := clock.bucket(6)
	if err := bucket.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := bucket.Wait(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if bucket.tokens != 0 {
		t.Fatalf("tokens = %v, want the cancelled reservation handed back", bucket.tokens)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bucket.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want an already cancelled context to fail at once", err)
	}
}

func TestGivenTheRealClockWhenTheContextEndsWhileWaitingThenWaitReturns(t *testing.T) {
	bucket := newTokenBucket(1)
	if err := bucket.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	if err := bucket.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("waited %v, want the wait to end with the context", elapsed)
	}
}
