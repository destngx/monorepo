package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestEachFeatureAsksForTheEffortItNeeds(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		run  func(s *Service) error
		want domain.AIEffort
	}{
		{"briefing", func(s *Service) error {
			_, err := s.Briefing(ctx, false, "")
			return err
		}, domain.AIEffortLow},
		{"account review", func(s *Service) error {
			_, err := s.Review(ctx, port.ReviewRequest{Kind: port.ReviewAccounts})
			return err
		}, domain.AIEffortMedium},
		{"chart insight", func(s *Service) error {
			_, err := s.ChartInsight(ctx, port.ChartInsightRequest{ChartType: "bar", Title: "Cash flow", Data: []int{1}})
			return err
		}, domain.AIEffortLow},
		{"chat", func(s *Service) error {
			return s.Chat(ctx, port.ChatRequest{Messages: []port.ChatMessage{{Role: "user", Content: "hi"}}}, func(port.ChatEvent) error { return nil })
		}, domain.AIEffortLow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := &scriptedClient{route: func(domain.AIRequest) reply {
				return reply{content: `{"briefing":"Fine.","alerts":[],"patterns":[],"summary":"Fine.","sections":[{"icon":"wallet","title":"Cash","content":"Fine.","severity":"success"}],"actionItems":["Keep going"]}`}
			}}
			if err := tc.run(newTestService(t, fixtureDatabase(), client, nil)); err != nil {
				t.Fatal(err)
			}
			if got := client.requests[0].Effort; got != tc.want {
				t.Fatalf("effort %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGateAdmitsWaitingCallsByPriorityThenArrival(t *testing.T) {
	g := newGate(1)
	if err := g.acquire(context.Background(), priorityBatch); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var order []string
	var done sync.WaitGroup
	var queued int
	enqueue := func(name string, p priority) {
		done.Add(1)
		go func() {
			defer done.Done()
			if err := g.acquire(context.Background(), p); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			g.release()
		}()
		// Each waiter is queued before the next arrives.
		queued++
		waitFor(t, func() bool { return g.waitingCount() == queued })
	}
	enqueue("capture", priorityBatch)
	enqueue("briefing", priorityPage)
	enqueue("chat", priorityTyping)
	enqueue("advisor", priorityPage)
	g.release()
	done.Wait()

	if want := []string{"chat", "briefing", "advisor", "capture"}; !slices.Equal(order, want) {
		t.Fatalf("order %v, want %v", order, want)
	}
}

func TestAWaiterThatGivesUpLeavesTheQueueAndTheSlotIsNotLost(t *testing.T) {
	g := newGate(1)
	if err := g.acquire(context.Background(), priorityBatch); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	gaveUp := make(chan error)
	go func() { gaveUp <- g.acquire(ctx, priorityTyping) }()
	waitFor(t, func() bool { return g.waitingCount() == 1 })
	cancel()
	if err := <-gaveUp; !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v, want canceled", err)
	}
	g.release()

	acquired := make(chan error)
	go func() { acquired <- g.acquire(context.Background(), priorityBatch) }()
	select {
	case err := <-acquired:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("the freed slot was lost")
	}
}

// heldClient holds every Complete until release is closed.
type heldClient struct {
	*scriptedClient
	release chan struct{}
}

func (c heldClient) Complete(ctx context.Context, request domain.AIRequest) (domain.AIResponse, error) {
	<-c.release
	return c.scriptedClient.Complete(ctx, request)
}

func TestIdenticalRequestsInFlightShareOneModelCall(t *testing.T) {
	scripted := &scriptedClient{route: func(domain.AIRequest) reply { return reply{content: "Flat."} }}
	client := heldClient{scripted, make(chan struct{})}
	s := newTestService(t, fixtureDatabase(), scripted, &memoryCache{})
	s.client = client
	request := port.ChartInsightRequest{ChartType: "bar", Title: "Cash flow", Data: []int{1}}

	results := make(chan string, 2)
	for range 2 {
		go func() {
			text, err := s.ChartInsight(context.Background(), request)
			if err != nil {
				t.Error(err)
			}
			results <- text
		}()
	}
	waitFor(t, func() bool { return s.flights.running() == 1 })
	close(client.release)
	if a, b := <-results, <-results; a != "Flat." || b != "Flat." {
		t.Fatalf("results %q %q", a, b)
	}
	if calls := len(scripted.requests); calls != 1 {
		t.Fatalf("%d model calls, want 1", calls)
	}
}

func TestAResultTheCallerStoppedWaitingForIsStillCached(t *testing.T) {
	scripted := &scriptedClient{route: func(domain.AIRequest) reply { return reply{content: "Flat."} }}
	client := heldClient{scripted, make(chan struct{})}
	s := newTestService(t, fixtureDatabase(), scripted, &memoryCache{})
	s.client = client
	request := port.ChartInsightRequest{ChartType: "bar", Title: "Cash flow", Data: []int{1}}

	ctx, cancel := context.WithCancel(context.Background())
	gaveUp := make(chan error)
	go func() {
		_, err := s.ChartInsight(ctx, request)
		gaveUp <- err
	}()
	waitFor(t, func() bool { return s.flights.running() == 1 })
	cancel()
	if err := <-gaveUp; !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v, want canceled", err)
	}
	close(client.release)
	waitFor(t, func() bool { return s.flights.running() == 0 })

	text, err := s.ChartInsight(context.Background(), request)
	if err != nil || text != "Flat." || len(scripted.requests) != 1 {
		t.Fatalf("text %q, err %v after %d model calls", text, err, len(scripted.requests))
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(time.Millisecond)
	}
}

func (g *gate) waitingCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.queued()
}

func (f *flights) running() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}
