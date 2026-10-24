package service

import "sync"

// readFlights lets concurrent reads of one cache key share one sheet read
// (WM-NFR-3): a read that finds its key already being read waits for it
// and takes the value from the cache. Without this, a cold page whose
// sections each read the same tabs sends the same request many times and
// runs into the Sheets per-minute quota.
type readFlights struct {
	mu      sync.Mutex
	running map[string]*sheetRead
}

// sheetRead is one sheet request in flight and the cache keys it fills.
type sheetRead struct {
	keys []string
	done chan struct{}
}

// claim claims every key, or claims none and returns a read in flight that
// has one of them; its done channel closes when it ends. A caller waits
// only while it holds no claim, so two reads never wait for each other.
func (f *readFlights) claim(keys ...string) (busy *sheetRead, release func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, key := range keys {
		if read, ok := f.running[key]; ok {
			return read, nil
		}
	}
	if f.running == nil {
		f.running = map[string]*sheetRead{}
	}
	read := &sheetRead{keys: keys, done: make(chan struct{})}
	for _, key := range keys {
		f.running[key] = read
	}
	return nil, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, key := range keys {
			delete(f.running, key)
		}
		close(read.done)
	}
}
