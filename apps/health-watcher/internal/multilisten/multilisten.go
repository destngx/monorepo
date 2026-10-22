// Package multilisten joins several listeners into one, so one HTTP server
// answers on the Mac's loopback address and its Tailscale address.
package multilisten

import (
	"errors"
	"net"
	"sync"
)

type accepted struct {
	conn net.Conn
	err  error
}

// Listener accepts connections from every listener it joins. Its address
// is the first one's.
type Listener struct {
	lns    []net.Listener
	conns  chan accepted
	done   chan struct{}
	closed sync.Once
	wg     sync.WaitGroup
}

// New joins lns, which must not be empty.
func New(lns ...net.Listener) *Listener {
	l := &Listener{lns: lns, conns: make(chan accepted), done: make(chan struct{})}
	for _, ln := range lns {
		l.wg.Add(1)
		go l.accept(ln)
	}
	return l
}

func (l *Listener) accept(ln net.Listener) {
	defer l.wg.Done()
	for {
		conn, err := ln.Accept()
		select {
		case l.conns <- accepted{conn, err}:
		case <-l.done:
			if conn != nil {
				_ = conn.Close()
			}
			return
		}
		if err != nil && errors.Is(err, net.ErrClosed) {
			return
		}
	}
}

// Accept returns the next connection from any of the listeners.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case a := <-l.conns:
		return a.conn, a.err
	case <-l.done:
		return nil, net.ErrClosed
	}
}

// Close closes every listener.
func (l *Listener) Close() error {
	var errs []error
	l.closed.Do(func() {
		close(l.done)
		for _, ln := range l.lns {
			errs = append(errs, ln.Close())
		}
		l.wg.Wait()
	})
	return errors.Join(errs...)
}

// Addr is the first listener's address.
func (l *Listener) Addr() net.Addr { return l.lns[0].Addr() }

// Addrs lists every listener's address.
func (l *Listener) Addrs() []net.Addr {
	out := make([]net.Addr, len(l.lns))
	for i, ln := range l.lns {
		out[i] = ln.Addr()
	}
	return out
}
