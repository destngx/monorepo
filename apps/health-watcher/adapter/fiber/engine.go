package fiber

import (
	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/service"
)

// engineRoutes are the Engine settings page: settings that apply to every
// tenant on the Mac, and who may change them (HW-OPS-9). The service lets
// only engine admins in, except to the AI quota (HW-OPS-10).
func (s *server) engineRoutes(r fiber.Router) {
	g := r.Group("/engine", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "private, no-store")
		return c.Next()
	})
	g.Get("/ai-usage", func(c *fiber.Ctx) error {
		return c.JSON(s.svc.Engine.AIUsage(c.UserContext()))
	})
	g.Get("/settings", func(c *fiber.Ctx) error {
		v, err := s.svc.Engine.View(c.UserContext(), session(c).UserID)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(v)
	})
	g.Put("/settings", func(c *fiber.Ctx) error {
		var in service.SettingsInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		v, err := s.svc.Engine.Update(c.UserContext(), session(c).UserID, in)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(v)
	})
	g.Post("/settings/check-ai", func(c *fiber.Ctx) error {
		res, err := s.svc.Engine.CheckAI(c.UserContext(), session(c).UserID)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(res)
	})
	g.Get("/users", func(c *fiber.Ctx) error {
		users, err := s.svc.Engine.Users(c.UserContext(), session(c).UserID)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(users)
	})
	g.Put("/users/:id/admin", func(c *fiber.Ctx) error {
		var in struct {
			Admin bool `json:"admin"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		u, err := s.svc.Engine.SetAdmin(c.UserContext(), session(c).UserID, c.Params("id"), in.Admin)
		if err != nil {
			return respond(c, s.log, err)
		}
		return c.JSON(u)
	})
}
