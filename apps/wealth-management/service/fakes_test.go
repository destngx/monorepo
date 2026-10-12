package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type sheetUpdate struct {
	rangeValue string
	values     []any
}

type sheetAppend struct {
	sheetName string
	anchor    string
	rows      [][]any
}

// fakeSheetClient serves fixed ranges and records writes.
type fakeSheetClient struct {
	ranges  map[string][][]string
	readErr map[string]error
	titles  []string
	nextRow int
	updates []sheetUpdate
	appends []sheetAppend
	reads   []string
	// requests counts client calls; a batch read of several ranges is one.
	requests int
	writeErr error
	// writeErrFor fails UpdateRow for one range only.
	writeErrFor map[string]error
	created     []string
	title       string
	titleErr    error
}

func newFakeSheetClient() *fakeSheetClient {
	return &fakeSheetClient{ranges: map[string][][]string{}, readErr: map[string]error{}, writeErrFor: map[string]error{}, nextRow: 100}
}

func (f *fakeSheetClient) ReadSheet(rangeValue string) ([][]string, error) {
	f.requests++
	return f.read(rangeValue)
}

func (f *fakeSheetClient) ReadSheets(ranges []string) ([][][]string, error) {
	f.requests++
	result := make([][][]string, 0, len(ranges))
	for _, rangeValue := range ranges {
		rows, err := f.read(rangeValue)
		if err != nil {
			return nil, err
		}
		result = append(result, rows)
	}
	return result, nil
}

func (f *fakeSheetClient) read(rangeValue string) ([][]string, error) {
	f.reads = append(f.reads, rangeValue)
	if err := f.readErr[rangeValue]; err != nil {
		return nil, err
	}
	rows, ok := f.ranges[rangeValue]
	if !ok {
		return nil, errors.New("unable to parse range: " + rangeValue)
	}
	return rows, nil
}

func (f *fakeSheetClient) UpdateRow(rangeValue string, values []any) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	if err := f.writeErrFor[rangeValue]; err != nil {
		return err
	}
	f.updates = append(f.updates, sheetUpdate{rangeValue: rangeValue, values: values})
	return nil
}

func (f *fakeSheetClient) WriteRowsAtFirstEmptyRow(sheetName string, anchor string, rows [][]any) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	f.appends = append(f.appends, sheetAppend{sheetName: sheetName, anchor: anchor, rows: rows})
	row := f.nextRow
	f.nextRow += len(rows)
	return row, nil
}

func (f *fakeSheetClient) SpreadsheetTitle() (string, error) {
	return f.title, f.titleErr
}

func (f *fakeSheetClient) SheetTitles() ([]string, error) {
	return f.titles, nil
}

func (f *fakeSheetClient) CreateSheet(title string, header []any) error {
	if f.writeErr != nil {
		return f.writeErr
	}
	f.titles = append(f.titles, title)
	f.created = append(f.created, title)
	f.updates = append(f.updates, sheetUpdate{rangeValue: title + "!A1", values: header})
	return nil
}

// fakeCache is an in-memory CacheService; failing makes every call error.
type fakeCache struct {
	values      map[string]string
	failing     bool
	invalidated []string
}

func newFakeCache() *fakeCache {
	return &fakeCache{values: map[string]string{}}
}

var errCacheDown = errors.New("cache down")

func (c *fakeCache) Set(_ context.Context, key string, value string, _ int) error {
	if c.failing {
		return errCacheDown
	}
	c.values[key] = value
	return nil
}

func (c *fakeCache) Get(_ context.Context, key string) (domain.CacheEntry, error) {
	if c.failing {
		return domain.CacheEntry{}, errCacheDown
	}
	value, ok := c.values[key]
	return domain.CacheEntry{Key: key, Value: value, Found: ok}, nil
}

func (c *fakeCache) Invalidate(_ context.Context, pattern string) (int, error) {
	if c.failing {
		return 0, errCacheDown
	}
	c.invalidated = append(c.invalidated, pattern)
	deleted := 0
	for key := range c.values {
		if key == pattern || (strings.HasSuffix(pattern, "*") && strings.HasPrefix(key, strings.TrimSuffix(pattern, "*"))) {
			delete(c.values, key)
			deleted++
		}
	}
	return deleted, nil
}

func (c *fakeCache) Ping(context.Context) error {
	if c.failing {
		return errCacheDown
	}
	return nil
}

var fixedNow = time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

func newTestDBService(t *testing.T, client *fakeSheetClient, cache *fakeCache) *dbService {
	t.Helper()
	svc := NewDatabaseService(client, cache, logger.NewTestLogger(t), DatabaseServiceOptions{
		Now: func() time.Time { return fixedNow },
	})
	return svc.(*dbService)
}

func ptr[T any](value T) *T {
	return &value
}
