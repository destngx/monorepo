package domain

import (
	"testing"
	"time"
)

func TestGivenFailedAttemptsWhenRetryingThenTheBackoffFollowsTheSpecAndStopsAtFive(t *testing.T) {
	want := []time.Duration{time.Minute, 5 * time.Minute, 30 * time.Minute, 6 * time.Hour}
	for i, w := range want {
		got, ok := RetryAfter(i + 1)
		if !ok || got != w {
			t.Errorf("RetryAfter(%d) = %v, %v; want %v", i+1, got, ok, w)
		}
	}
	if _, ok := RetryAfter(MaxJobAttempts); ok {
		t.Fatalf("a job is retried after %d attempts", MaxJobAttempts)
	}
}
