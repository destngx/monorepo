// Package service holds the use cases. Every method that touches tenant
// data takes a domain.Scope and checks its role first (HW-TEN-6).
package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// Deps are what the services share.
type Deps struct {
	Store  port.Store
	Files  port.FileStore
	Hasher port.PasswordHasher
	Clock  port.Clock
	// Location is the calendar timezone for "today" (Asia/Ho_Chi_Minh).
	Location *time.Location
	// FileLock serialises every decision about file bytes: an upload holds
	// it from writing the bytes until its rows commit, a release from
	// deleting the row until deleting the bytes. Without it a release could
	// delete bytes an upload in flight is about to reference. Shared by all
	// services; must not be nil.
	FileLock *sync.Mutex
}

// NewID returns a time-ordered unique id.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// Token prefixes make a leaked token easy to recognise.
const (
	prefixSession = "hws_"
	prefixUpload  = "hwu_"
	prefixInvite  = "hwi_"
)

// newToken returns a random token with a prefix. Only its hash is stored.
func newToken(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken is the stored form of a token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (d Deps) today() string {
	return d.Clock().In(d.Location).Format(domain.DateLayout)
}

const dayDuration = 24 * time.Hour
