package fiber

import (
	"apps/wealth-management/domain"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// TokenSource gives the current write token; empty means none is set. It is
// read on every write, so a token set on the Settings page applies at once.
type TokenSource interface {
	Token() string
}

// WriteGuard protects state-changing endpoints (WM-SEC-1). With a token set,
// requests need "Authorization: Bearer <token>". Without one, writes are only
// allowed when the server is bound to a loopback address.
type WriteGuard struct {
	tokens       TokenSource
	loopbackOnly bool
}

func NewWriteGuard(tokens TokenSource, host string) WriteGuard {
	return WriteGuard{tokens: tokens, loopbackOnly: domain.IsLoopbackHost(host)}
}

// Wrap returns handler guarded by the write policy.
func (g WriteGuard) Wrap(handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := ""
		if g.tokens != nil {
			token = g.tokens.Token()
		}
		if token == "" {
			if g.loopbackOnly {
				return handler(c)
			}
			return writeError(c, http.StatusForbidden, CodeForbidden, "writes are disabled: open the engine on localhost and set an API token on the Settings page")
		}
		presented, found := strings.CutPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		if !found || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(presented)), []byte(token)) != 1 {
			c.Set(fiber.HeaderWWWAuthenticate, "Bearer")
			return writeError(c, http.StatusUnauthorized, CodeUnauthorized, "missing or invalid API token")
		}
		return handler(c)
	}
}
