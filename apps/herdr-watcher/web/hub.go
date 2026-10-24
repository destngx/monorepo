package web

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"time"
)

// hub polls herdr while at least one phone is watching and hands each
// subscriber the latest snapshot whenever it changes. With nobody watching it
// stops, so the laptop does no work for a closed page.
type hub struct {
	every time.Duration
	fetch func(context.Context) []byte
	log   *slog.Logger

	mu   sync.Mutex
	subs map[chan []byte]struct{}
	last []byte
	stop context.CancelFunc
}

func newHub(every time.Duration, fetch func(context.Context) []byte, log *slog.Logger) *hub {
	return &hub{every: every, fetch: fetch, log: log, subs: map[chan []byte]struct{}{}}
}

// subscribe returns a channel that receives each new snapshot, starting with
// the current one. A slow reader only ever misses older snapshots.
func (h *hub) subscribe() (<-chan []byte, func()) {
	ch := make(chan []byte, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	if h.last != nil {
		ch <- h.last
	}
	if h.stop == nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.stop = cancel
		h.last = nil
		go h.run(ctx)
	}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.subs, ch)
		if len(h.subs) == 0 && h.stop != nil {
			h.stop()
			h.stop = nil
		}
	}
}

func (h *hub) run(ctx context.Context) {
	ticker := time.NewTicker(h.every)
	defer ticker.Stop()
	for {
		pollCtx, cancel := context.WithTimeout(ctx, h.every+5*time.Second)
		data := h.fetch(pollCtx)
		cancel()
		h.publish(ctx, data)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// publish runs under the lock that stops the poller, so a stopped poller can
// never overwrite the snapshot of the one that replaced it.
func (h *hub) publish(ctx context.Context, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ctx.Err() != nil || bytes.Equal(data, h.last) {
		return
	}
	h.last = data
	for ch := range h.subs {
		// Replace an unread older snapshot with this one.
		select {
		case <-ch:
		default:
		}
		ch <- data
	}
}
