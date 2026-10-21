package filestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"apps/health-watcher/domain"
)

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestGivenBytesWhenPuttingAndOpeningThenTheyRoundTripPrivately(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join(t.TempDir(), "files")
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("%PDF-1.7 synthetic")
	sha := hashOf(data)
	if err := store.Put(ctx, sha, data); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := store.Put(ctx, sha, data); err != nil {
		t.Fatalf("second Put: %v", err)
	}
	path := filepath.Join(root, sha[:2], sha[2:4], sha)
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("stat = %v, %v", info, err)
	}
	r, err := store.Open(ctx, sha)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(r)
	_ = r.Close()
	if string(got) != string(data) {
		t.Fatalf("read %q", got)
	}
	if err := store.Delete(ctx, sha); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Open(ctx, sha); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Open after delete = %v", err)
	}
}

func TestGivenAWrongHashWhenPuttingThenItIsRefused(t *testing.T) {
	store, _ := New(t.TempDir())
	if err := store.Put(context.Background(), hashOf([]byte("a")), []byte("b")); err == nil {
		t.Fatal("mismatched hash accepted")
	}
	if _, err := store.Open(context.Background(), "../../etc/passwd"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("path traversal = %v", err)
	}
}
