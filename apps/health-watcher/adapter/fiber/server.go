package fiber

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"mime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"

	"apps/health-watcher/domain"
	"apps/health-watcher/service"
)

// Cookie and header names.
const (
	SessionCookie = "hw_session"
	CSRFHeader    = "X-CSRF-Token"
)

const (
	localRequestID = "request_id"
	localSession   = "session"
	localToken     = "session_token"
	localScope     = "scope"
)

// Services are the use cases the routes call.
type Services struct {
	Accounts *service.Accounts
	Tenants  *service.Tenants
	Records  *service.Records
	Inbox    *service.Inbox
	Clinical *service.Clinical
	Engine   *service.Engine
	Capture  *service.Capture
}

// Options configure the HTTP server.
type Options struct {
	Log *slog.Logger
	// BodyLimit bounds every request body; uploads need the batch limit plus
	// multipart overhead.
	BodyLimit int
	// UIBuildDir is the built SvelteKit app; empty or missing skips the mount.
	UIBuildDir string
}

type server struct {
	svc Services
	log *slog.Logger
}

// New builds the Fiber app with every route.
func New(svc Services, opts Options) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "health-watcher",
		BodyLimit:             opts.BodyLimit,
		ErrorHandler:          ErrorHandler(opts.Log),
		DisableStartupMessage: true,
		ReadTimeout:           2 * time.Minute,
		WriteTimeout:          2 * time.Minute,
		IdleTimeout:           2 * time.Minute,
	})
	s := &server{svc: svc, log: opts.Log}
	app.Use(recover.New(), s.requestLog, securityHeaders)

	api := app.Group("/api")
	api.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	s.authRoutes(api)
	// The iPhone Shortcut posts here with an upload token (HW-IMP-4).
	api.Post("/inbox", s.tokenUpload)
	api.Get("/capture", s.captureTarget)
	api.Post("/uploads", s.tokenStage)

	signedIn := api.Group("", s.requireSession, s.requireCSRF)
	s.accountRoutes(signedIn)
	s.engineRoutes(signedIn)

	tenant := signedIn.Group("/tenants/:tenantId", s.requireTenant)
	s.tenantRoutes(tenant)
	s.recordRoutes(tenant)
	s.reviewRoutes(tenant)
	s.inboxRoutes(tenant)
	s.clinicalRoutes(tenant)

	api.Use(func(c *fiber.Ctx) error { return fiber.ErrNotFound })
	if opts.UIBuildDir != "" {
		MountUI(app, opts.UIBuildDir)
	}
	return app
}

func (s *server) requestLog(c *fiber.Ctx) error {
	id := uuid.NewString()
	c.Locals(localRequestID, id)
	c.Set("X-Request-ID", id)
	start := time.Now()
	err := c.Next()
	status := c.Response().StatusCode()
	if err != nil {
		// The error handler runs after this middleware returns; take its status.
		var fe *fiber.Error
		if errors.As(err, &fe) {
			status = fe.Code
		}
	}
	// Route patterns only: paths carry ids, never names or values (HW-OPS-5).
	s.log.Info("request", "request_id", id, "method", c.Method(), "route", c.Route().Path, "status", status,
		"ms", time.Since(start).Milliseconds())
	return err
}

func securityHeaders(c *fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Referrer-Policy", "no-referrer")
	c.Set("X-Frame-Options", "SAMEORIGIN")
	return c.Next()
}

func (s *server) setSessionCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(domain.SessionLifetime.Seconds()),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

func (s *server) clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1, HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode})
}

func (s *server) requireSession(c *fiber.Ctx) error {
	token := c.Cookies(SessionCookie)
	sess, err := s.svc.Accounts.Authenticate(c.UserContext(), token)
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			s.clearSessionCookie(c)
		}
		return respond(c, s.log, err)
	}
	c.Locals(localSession, sess)
	c.Locals(localToken, token)
	return c.Next()
}

// requireCSRF checks the session's token on every request that changes
// state (HW-AUTH-2).
func (s *server) requireCSRF(c *fiber.Ctx) error {
	switch c.Method() {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return c.Next()
	}
	sess := session(c)
	got := c.Get(CSRFHeader)
	if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(sess.CSRFToken)) != 1 {
		return writeError(c, fiber.StatusForbidden, CodeForbidden, "Missing or wrong CSRF token. Reload the page.")
	}
	return c.Next()
}

func (s *server) requireTenant(c *fiber.Ctx) error {
	sc, err := s.svc.Accounts.ScopeFor(c.UserContext(), session(c).UserID, c.Params("tenantId"))
	if err != nil {
		return respond(c, s.log, err)
	}
	c.Locals(localScope, sc)
	return c.Next()
}

func session(c *fiber.Ctx) domain.Session {
	sess, _ := c.Locals(localSession).(domain.Session)
	return sess
}

func scope(c *fiber.Ctx) domain.Scope {
	sc, _ := c.Locals(localScope).(domain.Scope)
	return sc
}

// parseJSON decodes a JSON body. Requiring the JSON content type means a
// cross-site form can't send it without a CORS preflight.
func parseJSON(c *fiber.Ctx, out any) error {
	mt, _, err := mime.ParseMediaType(string(c.Request().Header.ContentType()))
	if err != nil || mt != fiber.MIMEApplicationJSON {
		return domain.Invalid("send the body as application/json")
	}
	if err := c.BodyParser(out); err != nil {
		return domain.Invalid("the body is not valid JSON")
	}
	return nil
}

func bearer(c *fiber.Ctx) string {
	h := c.Get(fiber.HeaderAuthorization)
	if token, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(token)
	}
	return ""
}
