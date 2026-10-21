package fiber

import (
	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/service"
)

// authRoutes need no session: first-run setup, login and invite links.
func (s *server) authRoutes(api fiber.Router) {
	api.Get("/setup", func(c *fiber.Ctx) error {
		need, err := s.svc.Accounts.NeedsSetup(c.UserContext())
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(fiber.Map{"needsSetup": need})
	})
	api.Post("/setup", func(c *fiber.Ctx) error {
		var in service.SetupInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		signed, err := s.svc.Accounts.Setup(c.UserContext(), in)
		return s.signedIn(c, signed, err, fiber.StatusCreated)
	})
	api.Post("/auth/login", func(c *fiber.Ctx) error {
		var in struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		signed, err := s.svc.Accounts.Login(c.UserContext(), in.Email, in.Password)
		return s.signedIn(c, signed, err, fiber.StatusOK)
	})
	api.Post("/auth/logout", func(c *fiber.Ctx) error {
		if err := s.svc.Accounts.Logout(c.UserContext(), c.Cookies(SessionCookie)); err != nil {
			return respond(c, s.log, err)
		}
		s.clearSessionCookie(c)
		return c.SendStatus(fiber.StatusNoContent)
	})
	api.Get("/invites/:token", func(c *fiber.Ctx) error {
		info, err := s.svc.Tenants.Invite(c.UserContext(), c.Params("token"))
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(info)
	})
	api.Post("/invites/:token/signup", func(c *fiber.Ctx) error {
		var in service.SignUpInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		signed, err := s.svc.Tenants.AcceptInviteAsNewUser(c.UserContext(), c.Params("token"), in)
		return s.signedIn(c, signed, err, fiber.StatusCreated)
	})
}

func (s *server) signedIn(c *fiber.Ctx, signed service.SignedIn, err error, status int) error {
	if err != nil {
		return respond(c, s.log, err)
	}
	s.setSessionCookie(c, signed.Token)
	return c.Status(status).JSON(signed.Me)
}

// accountRoutes act on the signed-in user, outside any tenant.
func (s *server) accountRoutes(r fiber.Router) {
	r.Get("/auth/me", func(c *fiber.Ctx) error {
		me, err := s.svc.Accounts.Me(c.UserContext(), session(c))
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(me)
	})
	r.Post("/auth/password", func(c *fiber.Ctx) error {
		var in struct {
			Current string `json:"current"`
			Next    string `json:"next"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		signed, err := s.svc.Accounts.ChangePassword(c.UserContext(), session(c).UserID, in.Current, in.Next)
		return s.signedIn(c, signed, err, fiber.StatusOK)
	})
	r.Post("/tenants", func(c *fiber.Ctx) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		t, err := s.svc.Tenants.Create(c.UserContext(), session(c).UserID, in.Name)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.Status(fiber.StatusCreated).JSON(t)
	})
	r.Post("/invites/:token/accept", func(c *fiber.Ctx) error {
		t, err := s.svc.Tenants.AcceptInvite(c.UserContext(), c.Params("token"), session(c).UserID)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(t)
	})
}

// tenantRoutes manage the tenant itself: name, members, invites, tokens.
func (s *server) tenantRoutes(r fiber.Router) {
	r.Patch("", func(c *fiber.Ctx) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		if err := s.svc.Tenants.Rename(c.UserContext(), scope(c), in.Name); err != nil {
			return respond(c, s.log, err)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
	r.Get("/members", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Tenants.Members(c.UserContext(), scope(c)))
	})
	r.Patch("/members/:userId", func(c *fiber.Ctx) error {
		var in struct {
			Role string `json:"role"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.noContent(c, s.svc.Tenants.SetRole(c.UserContext(), scope(c), c.Params("userId"), in.Role))
	})
	r.Delete("/members/:userId", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Tenants.RemoveMember(c.UserContext(), scope(c), c.Params("userId")))
	})
	r.Get("/invites", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Tenants.PendingInvites(c.UserContext(), scope(c)))
	})
	r.Post("/invites", func(c *fiber.Ctx) error {
		var in struct {
			Role string `json:"role"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		inv, err := s.svc.Tenants.CreateInvite(c.UserContext(), scope(c), in.Role)
		return s.created(c, inv, err)
	})
	r.Delete("/invites/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Tenants.RevokeInvite(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Get("/upload-tokens", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Tenants.UploadTokens(c.UserContext(), scope(c)))
	})
	r.Post("/upload-tokens", func(c *fiber.Ctx) error {
		var in struct {
			Name string `json:"name"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		tok, err := s.svc.Tenants.CreateUploadToken(c.UserContext(), scope(c), in.Name)
		return s.created(c, tok, err)
	})
	r.Delete("/upload-tokens/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Tenants.RevokeUploadToken(c.UserContext(), scope(c), c.Params("id")))
	})
}

// list returns a writer for a (slice, error) pair.
func (s *server) list(c *fiber.Ctx) func(any, error) error {
	return func(v any, err error) error {
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(v)
	}
}

func (s *server) created(c *fiber.Ctx, v any, err error) error {
	if err != nil {
		return respond(c, s.log, err)
	}
	return c.Status(fiber.StatusCreated).JSON(v)
}

func (s *server) noContent(c *fiber.Ctx, err error) error {
	if err != nil {
		return respond(c, s.log, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
