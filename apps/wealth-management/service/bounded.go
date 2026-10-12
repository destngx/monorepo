package service

import "sync"

// boundedGroup runs tasks with at most limit in flight, so fan-out market
// calls stay inside public provider rate limits.
type boundedGroup struct {
	wg    sync.WaitGroup
	slots chan struct{}
}

func newBoundedGroup(limit int) *boundedGroup {
	return &boundedGroup{slots: make(chan struct{}, max(limit, 1))}
}

func (g *boundedGroup) Go(task func()) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		g.slots <- struct{}{}
		defer func() { <-g.slots }()
		task()
	}()
}

func (g *boundedGroup) Wait() {
	g.wg.Wait()
}
