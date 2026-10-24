package ai

import (
	"apps/wealth-management/domain"
	"context"
	"sync"
)

// maxConcurrentCalls bounds the model calls in flight (WM-AI-16). More
// calls wait for a slot, the ones the user waits on first.
const maxConcurrentCalls = 4

// priority orders calls waiting for a slot; higher goes first.
type priority int

const (
	// priorityBatch: one of the many calls of a Smart Capture batch.
	priorityBatch priority = iota
	// priorityPage: a card that loads with a page without being asked.
	priorityPage
	// priorityAsked: something the user clicked to get.
	priorityAsked
	// priorityTyping: an answer to what the user just typed.
	priorityTyping
	priorities
)

// class is how a feature calls the model: how long the model may think and
// how soon the call runs when calls queue (WM-AI-16).
type class struct {
	effort   domain.AIEffort
	priority priority
}

var (
	// classTyping answers what the user typed: chat, category suggestion,
	// form fill.
	classTyping = class{domain.AIEffortLow, priorityTyping}
	// classExplain is short commentary the user asked for on figures the
	// engine computed: chart insight, cashback, ticker, news.
	classExplain = class{domain.AIEffortLow, priorityAsked}
	// classReview is a full review the user asked for and reads closely:
	// the structured reviews, the health commentary, the think tank.
	classReview = class{domain.AIEffortMedium, priorityAsked}
	// classPage loads with a page: the briefing and the budget advisor.
	classPage = class{domain.AIEffortLow, priorityPage}
	// classCapture reads and types a Smart Capture batch.
	classCapture = class{domain.AIEffortLow, priorityBatch}
)

// gate admits model calls up to its capacity: waiting calls by priority,
// first come first served within one priority.
type gate struct {
	mu      sync.Mutex
	free    int
	waiting [priorities][]*waiter
}

type waiter struct {
	ready   chan struct{}
	granted bool
}

func newGate(capacity int) *gate {
	return &gate{free: capacity}
}

// acquire waits for a slot; the caller must call release once done.
func (g *gate) acquire(ctx context.Context, p priority) error {
	g.mu.Lock()
	if g.free > 0 && g.queued() == 0 {
		g.free--
		g.mu.Unlock()
		return nil
	}
	w := &waiter{ready: make(chan struct{})}
	g.waiting[p] = append(g.waiting[p], w)
	g.mu.Unlock()

	select {
	case <-w.ready:
		return nil
	case <-ctx.Done():
		g.mu.Lock()
		defer g.mu.Unlock()
		if w.granted {
			// The slot arrived as the wait ended: hand it on.
			g.grant()
		} else {
			g.remove(p, w)
		}
		return ctx.Err()
	}
}

func (g *gate) release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.grant()
}

// grant gives a freed slot to the first waiter of the highest priority.
func (g *gate) grant() {
	for p := priorities - 1; p >= 0; p-- {
		if queue := g.waiting[p]; len(queue) > 0 {
			w := queue[0]
			g.waiting[p] = queue[1:]
			w.granted = true
			close(w.ready)
			return
		}
	}
	g.free++
}

func (g *gate) queued() int {
	total := 0
	for _, queue := range g.waiting {
		total += len(queue)
	}
	return total
}

func (g *gate) remove(p priority, w *waiter) {
	queue := g.waiting[p]
	for i, other := range queue {
		if other == w {
			g.waiting[p] = append(queue[:i:i], queue[i+1:]...)
			return
		}
	}
}

// flights shares one computation between identical requests in flight, so
// reopening a page while its card is still loading does not ask the model
// twice.
type flights struct {
	mu    sync.Mutex
	calls map[string]*flight
}

type flight struct {
	done  chan struct{}
	value any
	err   error
}

// do runs compute once per key at a time. compute gets a context that
// outlives any one caller, so a result someone stopped waiting for is still
// cached for the next visit; each caller stops waiting when its own context
// ends.
func (f *flights) do(ctx context.Context, key string, compute func(context.Context) (any, error)) (any, error) {
	f.mu.Lock()
	if f.calls == nil {
		f.calls = map[string]*flight{}
	}
	call, running := f.calls[key]
	if !running {
		call = &flight{done: make(chan struct{})}
		f.calls[key] = call
		go func() {
			call.value, call.err = compute(context.WithoutCancel(ctx))
			f.mu.Lock()
			delete(f.calls, key)
			f.mu.Unlock()
			close(call.done)
		}()
	}
	f.mu.Unlock()

	select {
	case <-call.done:
		return call.value, call.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
