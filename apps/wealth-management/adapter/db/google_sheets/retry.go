package google_sheets

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
)

// retryPolicy retries throttled and failed Sheets calls with bounded
// exponential backoff (WM-NFR-3): base, 2x base, 4x base... capped at
// maxDelay, at most attempts calls in total. A Retry-After header wins when
// it is longer, within maxDelay.
type retryPolicy struct {
	attempts int
	base     time.Duration
	maxDelay time.Duration
	sleep    func(ctx context.Context, delay time.Duration) error
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{attempts: 4, base: 500 * time.Millisecond, maxDelay: 8 * time.Second, sleep: sleepContext}
}

// callKind says which failures are safe to retry. Reads are idempotent, so
// any 429 or 5xx is retried. A write is retried only on 429, which Google
// returns before applying the request; a 5xx may have applied it.
type callKind int

const (
	readCall callKind = iota
	writeCall
)

func (p retryPolicy) do(ctx context.Context, c *SheetsClient, operation string, kind callKind, call func() error) error {
	var err error
	for attempt := 1; ; attempt++ {
		err = call()
		if err == nil || attempt >= p.attempts || !retryable(err, kind) {
			return err
		}
		delay := p.delay(attempt, err)
		c.log.LogApplicationEvent(ctx, "sheets call failed, retrying",
			slog.String("component", "google_sheets"),
			slog.String("operation", operation),
			slog.Int("attempt", attempt),
			slog.Duration("delay", delay),
			slog.String("error", err.Error()),
		)
		if sleepErr := p.sleep(ctx, delay); sleepErr != nil {
			return errors.Join(err, sleepErr)
		}
	}
}

func (p retryPolicy) delay(attempt int, err error) time.Duration {
	delay := min(p.base<<(attempt-1), p.maxDelay)
	if after := retryAfter(err); after > delay {
		delay = min(after, p.maxDelay)
	}
	return delay
}

func retryable(err error, kind callKind) bool {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		// A transport timeout or reset never reached the API for sure, so
		// only reads retry it.
		var netErr net.Error
		return kind == readCall && errors.As(err, &netErr) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	if apiErr.Code == http.StatusTooManyRequests {
		return true
	}
	return kind == readCall && apiErr.Code >= 500 && apiErr.Code <= 599
}

func retryAfter(err error) time.Duration {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) || apiErr.Header == nil {
		return 0
	}
	seconds, parseErr := strconv.Atoi(strings.TrimSpace(apiErr.Header.Get("Retry-After")))
	if parseErr != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
