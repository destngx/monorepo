package settings

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	store, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestGivenANewStoreWhenReadingThenItIsEmpty(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "settings.db"))
	values, err := store.Values(context.Background())
	if err != nil || len(values) != 0 {
		t.Fatalf("values = %v err %v", values, err)
	}
}

func TestGivenSavedChangesWhenReopeningThenTheyPersistAndEmptyValuesDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.db")
	ctx := context.Background()
	first := openTestStore(t, path)
	if err := first.Save(ctx, map[string]string{"GOOGLE_SHEETS_ID": "abc", "AI_MODEL": "gpt"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := first.Save(ctx, map[string]string{"AI_MODEL": "", "GOOGLE_SHEETS_ID": "xyz"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	second := openTestStore(t, path)
	values, err := second.Values(ctx)
	if err != nil || !reflect.DeepEqual(values, map[string]string{"GOOGLE_SHEETS_ID": "xyz"}) {
		t.Fatalf("values = %v err %v", values, err)
	}
	if second.Path() != path {
		t.Fatalf("path = %s", second.Path())
	}
}

func TestGivenACancelledContextWhenSavingThenNothingIsWritten(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "settings.db"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.Save(ctx, map[string]string{"A": "1", "B": "2"}); err == nil {
		t.Fatal("expected a cancelled save to fail")
	}
	values, _ := store.Values(context.Background())
	if len(values) != 0 {
		t.Fatalf("values = %v, want none", values)
	}
}

func TestGivenTheDefaultPathThenItIsInTheUserConfigDir(t *testing.T) {
	path, err := DefaultPath()
	if err != nil || !strings.HasSuffix(path, filepath.Join("wealth-management", "settings.db")) {
		t.Fatalf("path = %s err %v", path, err)
	}
}

func TestGivenAnOverrideWhenResolvingThePathThenItWins(t *testing.T) {
	if path, err := ResolvePath("/x/settings.db"); err != nil || path != "/x/settings.db" {
		t.Fatalf("path = %s err %v", path, err)
	}
	fallback, _ := DefaultPath()
	if path, err := ResolvePath(""); err != nil || path != fallback {
		t.Fatalf("path = %s err %v", path, err)
	}
}
