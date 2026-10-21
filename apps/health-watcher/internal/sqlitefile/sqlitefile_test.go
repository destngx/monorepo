package sqlitefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGivenAMissingDirectoryWhenOpeningThenDirectoryAndFileArePrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested dir", "data.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE t (v TEXT)`); err != nil {
		t.Fatalf("exec: %v", err)
	}
	for target, want := range map[string]os.FileMode{filepath.Dir(path): 0o700, path: 0o600} {
		info, err := os.Stat(target)
		if err != nil {
			t.Fatalf("stat %s: %v", target, err)
		}
		if perm := info.Mode().Perm(); perm != want {
			t.Fatalf("%s perm = %o, want %o", target, perm, want)
		}
	}
}

func TestGivenAnExistingLooseFileWhenOpeningThenItIsRestricted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o, want 600", info.Mode().Perm())
	}
}

func TestGivenAnEmptyPathWhenOpeningThenItFails(t *testing.T) {
	if _, err := Open("  "); err == nil {
		t.Fatal("expected an error")
	}
}
