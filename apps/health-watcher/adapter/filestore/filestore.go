// Package filestore keeps file bytes on disk by SHA-256 (HW-OPS-2):
// <root>/ab/cd/<hash>, directories 0700 and files 0600.
package filestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Store is a directory of files named by their hash.
type Store struct {
	root string
}

var _ port.FileStore = (*Store)(nil)

// New creates root when missing and restricts it to the user.
func New(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create file store: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("restrict file store: %w", err)
	}
	return &Store{root: root}, nil
}

// Path returns where the bytes of sha are on disk.
func (s *Store) Path(sha string) (string, error) { return s.path(sha) }

func (s *Store) path(sha string) (string, error) {
	if !shaPattern.MatchString(sha) {
		return "", domain.Invalid("file hash is not a SHA-256 hex string")
	}
	return filepath.Join(s.root, sha[:2], sha[2:4], sha), nil
}

// Put writes data under sha unless it is already there. The bytes must
// hash to sha; a mismatch is refused.
func (s *Store) Put(_ context.Context, sha string, data []byte) error {
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != sha {
		return fmt.Errorf("store file: content does not match hash %s", sha)
	}
	path, err := s.path(sha)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("store file: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return fmt.Errorf("store file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func(cause error) error {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("store file: %w", cause)
	}
	if err := tmp.Chmod(0o600); err != nil {
		return cleanup(err)
	}
	if _, err := tmp.Write(data); err != nil {
		return cleanup(err)
	}
	if err := tmp.Sync(); err != nil {
		return cleanup(err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("store file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("store file: %w", err)
	}
	return nil
}

// Open returns the file's bytes; a missing file is domain.ErrNotFound.
func (s *Store) Open(_ context.Context, sha string) (io.ReadCloser, error) {
	path, err := s.path(sha)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, domain.NotFound("file content")
	}
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

// Delete removes the file; a missing file is not an error.
func (s *Store) Delete(_ context.Context, sha string) error {
	path, err := s.path(sha)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
