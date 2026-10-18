package domain

import (
	"fmt"
	"net"
	"strings"
)

// APITokenSettingKey stores the write token (WM-SEC-1) in the settings store.
// It is not in the settings registry: it is set on its own endpoint and
// applies at once, never through the generic save and restart.
const APITokenSettingKey = "API_TOKEN"

const (
	minAPITokenLength = 32
	maxAPITokenLength = 256
)

// APITokenStatus says whether writes need a token. The token itself is never
// returned.
type APITokenStatus struct {
	Set bool `json:"set"`
	// Loopback is true when the engine listens on a loopback address, the
	// only case where writes work without a token.
	Loopback bool `json:"loopback"`
}

// APITokenUpdate replaces the token; it cannot be cleared.
type APITokenUpdate struct {
	Token string `json:"token"`
}

// NormalizeAPIToken trims value and checks it can travel in an
// "Authorization: Bearer" header. Errors never include the value.
func NormalizeAPIToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < minAPITokenLength || len(value) > maxAPITokenLength {
		return "", fmt.Errorf("%w: the API token must be %d to %d characters", ErrValidation, minAPITokenLength, maxAPITokenLength)
	}
	for _, char := range value {
		if char < '!' || char > '~' {
			return "", fmt.Errorf("%w: the API token may only use visible ASCII characters, no spaces", ErrValidation)
		}
	}
	return value, nil
}

// IsLoopbackHost reports whether host (a HOST bind address) only accepts
// connections from this machine.
func IsLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
