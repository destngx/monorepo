package service

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// staleTracker remembers which cache keys were last served from their last
// good copy (WM-NFR-4). Each stale serve gets a sequence number, so a
// request can ask what was served stale while it ran.
type staleTracker struct {
	mu   sync.Mutex
	seq  uint64
	keys map[string]staleServe
}

type staleServe struct {
	asOf time.Time
	seq  uint64
}

func newStaleTracker() *staleTracker {
	return &staleTracker{keys: map[string]staleServe{}}
}

func (t *staleTracker) served(key string, asOf time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seq++
	t.keys[key] = staleServe{asOf: asOf, seq: t.seq}
}

func (t *staleTracker) recovered(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.keys, key)
}

func (t *staleTracker) cursor() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.seq
}

func (t *staleTracker) since(cursor uint64) (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var oldest time.Time
	stale := false
	for _, serve := range t.keys {
		if serve.seq > cursor && (!stale || serve.asOf.Before(oldest)) {
			oldest, stale = serve.asOf, true
		}
	}
	return oldest, stale
}

func (s *dbService) StaleCursor() uint64 {
	return s.stale.cursor()
}

func (s *dbService) StaleSince(cursor uint64) (time.Time, bool) {
	return s.stale.since(cursor)
}

// storeFresh caches a value read from the sheet together with its last good
// copy, and clears the key's stale state.
func (s *dbService) storeFresh(key string, value any, ttlSeconds int) {
	s.writeCache(key, value, ttlSeconds)
	s.writeCache(domain.LastGoodKey(key), domain.LastGood[any]{AsOf: s.now().UTC(), Value: value}, domain.LastGoodTTLSeconds)
	s.stale.recovered(key)
}

// serveLastGood answers a failed sheet read with the key's last good copy.
// Domain errors (a missing tab, invalid input) are answers, not outages, so
// they are returned as they are; so is cause when there is no copy.
func serveLastGood[T any](s *dbService, key string, cause error) (T, error) {
	var zero T
	if isDomainError(cause) {
		return zero, cause
	}
	lastGood, ok := readCached[domain.LastGood[T]](s, domain.LastGoodKey(key))
	if !ok {
		return zero, cause
	}
	s.log.LogError(context.Background(), "sheet read failed, serving the last good copy", cause,
		slog.String("key", key), slog.String("as_of", lastGood.AsOf.Format(time.RFC3339)))
	s.stale.served(key, lastGood.AsOf)
	return lastGood.Value, nil
}

func isDomainError(err error) bool {
	return errors.Is(err, domain.ErrValidation) || errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrConflict)
}

// sheetDataset is one cache key of a batched read: the ranges it needs and
// how it decodes into its destination from the cache, the sheet rows (in
// the order of ranges) or its last good copy.
type sheetDataset struct {
	key       string
	ranges    []string
	fromCache func() bool
	build     func(rows [][][]string) (any, error)
	lastGood  func(cause error) error
}

func dataset[T any](s *dbService, key string, target *T, build func(rows [][][]string) (T, error), ranges ...string) sheetDataset {
	return sheetDataset{
		key:    key,
		ranges: ranges,
		fromCache: func() bool {
			value, ok := readCached[T](s, key)
			if ok {
				*target = value
			}
			return ok
		},
		build: func(rows [][][]string) (any, error) {
			value, err := build(rows)
			if err != nil {
				return nil, err
			}
			*target = value
			return value, nil
		},
		lastGood: func(cause error) error {
			value, err := serveLastGood[T](s, key, cause)
			if err == nil {
				*target = value
			}
			return err
		},
	}
}

// batchRead fills every dataset, from the cache unless force is set, and
// reads all the misses in one Sheets request (WM-NFR-3). Datasets are built
// in order, so a later one may use an earlier one's value. When the request
// fails, each miss falls back to its last good copy.
func (s *dbService) batchRead(force bool, datasets ...sheetDataset) error {
	misses := make([]sheetDataset, 0, len(datasets))
	ranges := []string{}
	for _, set := range datasets {
		if !force && set.fromCache() {
			continue
		}
		misses = append(misses, set)
		ranges = append(ranges, set.ranges...)
	}
	if len(misses) == 0 {
		return nil
	}
	rows, err := s.readRanges(ranges...)
	if err != nil {
		for _, set := range misses {
			if err := set.lastGood(err); err != nil {
				return err
			}
		}
		return nil
	}
	offset := 0
	for _, set := range misses {
		value, err := set.build(rows[offset : offset+len(set.ranges)])
		if err != nil {
			return err
		}
		offset += len(set.ranges)
		s.storeFresh(set.key, value, sheetDataTTL)
	}
	return nil
}
