package fiber

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"time"

	fiberLib "github.com/gofiber/fiber/v2"
)

// StaleDataHeader marks a response built from last good copies because the
// sheet or every market provider failed (WM-NFR-4). Its value is the RFC 3339
// UTC time of the oldest data served, so the UI can say "data as of ...".
const StaleDataHeader = "X-Data-Stale"

// StaleDataMiddleware sets StaleDataHeader on responses that served stale
// data. Services with a context mark it on the request (domain.MarkStale);
// services without one are watched through port.StaleReporter.
type StaleDataMiddleware struct {
	reporters []port.StaleReporter
}

func NewStaleDataMiddleware() *StaleDataMiddleware {
	return &StaleDataMiddleware{}
}

// Watch adds a reporter. Call it during setup, before the server starts.
func (m *StaleDataMiddleware) Watch(reporter port.StaleReporter) {
	m.reporters = append(m.reporters, reporter)
}

func (m *StaleDataMiddleware) Handler() fiberLib.Handler {
	return func(c *fiberLib.Ctx) error {
		recorder := &domain.StaleRecorder{}
		c.SetUserContext(domain.WithStaleRecorder(c.UserContext(), recorder))
		cursors := make([]uint64, len(m.reporters))
		for i, reporter := range m.reporters {
			cursors[i] = reporter.StaleCursor()
		}

		err := c.Next()

		for i, reporter := range m.reporters {
			if asOf, stale := reporter.StaleSince(cursors[i]); stale {
				recorder.Mark(asOf)
			}
		}
		if asOf, stale := recorder.AsOf(); stale {
			c.Set(StaleDataHeader, asOf.UTC().Format(time.RFC3339))
		}
		return err
	}
}
