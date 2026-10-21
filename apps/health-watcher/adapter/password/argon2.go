// Package password hashes passwords with argon2id (HW-AUTH-1) in the PHC
// string format: $argon2id$v=19$m=<KiB>,t=<passes>,p=<lanes>$<salt>$<hash>.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"apps/health-watcher/port"
)

// Params are the argon2id cost settings.
type Params struct {
	MemoryKiB uint32
	Passes    uint32
	Lanes     uint8
}

// Default is the RFC 9106 second recommended option: 64 MiB, 3 passes, 4 lanes.
var Default = Params{MemoryKiB: 64 * 1024, Passes: 3, Lanes: 4}

const (
	saltLength = 16
	keyLength  = 32
)

// MaxConcurrent caps hashes running at once. Each takes MemoryKiB, so a
// burst of logins can't exhaust memory (64 MiB x 2 at the default).
const MaxConcurrent = 2

// Hasher hashes new passwords with its params and verifies hashes made
// with any params, so raising the cost keeps old hashes valid.
type Hasher struct {
	params Params
	slots  chan struct{}
}

var _ port.PasswordHasher = Hasher{}

// New returns a hasher with the given params.
func New(p Params) Hasher { return Hasher{params: p, slots: make(chan struct{}, MaxConcurrent)} }

func (h Hasher) key(password string, salt []byte, p Params, n uint32) []byte {
	h.slots <- struct{}{}
	defer func() { <-h.slots }()
	return argon2.IDKey([]byte(password), salt, p.Passes, p.MemoryKiB, p.Lanes, n)
}

var b64 = base64.RawStdEncoding

func (h Hasher) Hash(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password salt: %w", err)
	}
	key := h.key(password, salt, h.params, keyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version,
		h.params.MemoryKiB, h.params.Passes, h.params.Lanes, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func (h Hasher) Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, fmt.Errorf("password hash: unknown format")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, fmt.Errorf("password hash: unsupported version")
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Passes, &p.Lanes); err != nil {
		return false, fmt.Errorf("password hash: bad parameters: %w", err)
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("password hash: bad salt: %w", err)
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("password hash: bad key: %w", err)
	}
	got := h.key(password, salt, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
