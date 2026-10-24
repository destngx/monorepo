package web

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestHubPollsOnlyWhileSomeoneWatches(t *testing.T) {
	var calls atomic.Int64
	h := newHub(5*time.Millisecond, func(context.Context) []byte {
		return []byte(strconv.FormatInt(calls.Add(1), 10))
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	time.Sleep(30 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("polled %d times with nobody watching", calls.Load())
	}

	updates, unsubscribe := h.subscribe()
	<-updates
	<-updates
	unsubscribe()

	time.Sleep(20 * time.Millisecond)
	stopped := calls.Load()
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != stopped {
		t.Fatalf("kept polling after the last watcher left: %d then %d", stopped, calls.Load())
	}
}

func TestHubSendsOnlyChanges(t *testing.T) {
	h := newHub(5*time.Millisecond, func(context.Context) []byte { return []byte("same") },
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	updates, unsubscribe := h.subscribe()
	defer unsubscribe()

	<-updates
	select {
	case data := <-updates:
		t.Fatalf("sent an unchanged snapshot again: %s", data)
	case <-time.After(40 * time.Millisecond):
	}
}
