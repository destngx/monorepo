package fiber

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// WriteGuard protects state-changing endpoints (WM-SEC-1). With a token set,
// requests need "Authorization: Bearer <token>". Without one, writes are only
// allowed when the server is bound to a loopback address.
type WriteGuard struct {
	token        string
	loopbackOnly bool
}

func NewWriteGuard(token string, host string) WriteGuard {
	return WriteGuard{token: strings.TrimSpace(token), loopbackOnly: isLoopbackHost(host)}
}

// Wrap returns handler guarded by the write policy.
func (g WriteGuard) Wrap(handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if g.token == "" {
			if g.loopbackOnly {
				return handler(c)
			}
			return writeError(c, http.StatusForbidden, CodeForbidden, "writes are disabled: set API_TOKEN when binding to a non-loopback host")
		}
		presented, found := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !found || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(presented)), []byte(g.token)) != 1 {
			c.Set(fiber.HeaderWWWAuthenticate, "Bearer")
			return writeError(c, http.StatusUnauthorized, CodeUnauthorized, "missing or invalid API token")
		}
		return handler(c)
	}
}

func isLoopbackHost(host string) bool {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
