package fiber

import (
	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/domain"
	"apps/health-watcher/service"
)

// reviewRoutes are the inbox review page's routes (HW-READ-4, HW-AI-7,
// HW-IMP-6). Static paths come before /inbox/:id so "groups" is no id.
func (s *server) reviewRoutes(r fiber.Router) {
	r.Get("/inbox/groups", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Inbox.Groups(c.UserContext(), scope(c)))
	})
	r.Post("/inbox/groups/file", func(c *fiber.Ctx) error {
		var in service.GroupFileInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		v, err := s.svc.Inbox.FileGroup(c.UserContext(), scope(c), in)
		return s.created(c, v, err)
	})
	r.Get("/inbox/:id", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Inbox.Item(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Post("/inbox/:id/reading/send", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Inbox.SendReading(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Post("/inbox/:id/reading/skip", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Inbox.SkipReading(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Post("/inbox/:id/reading/retry", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Inbox.RetryReading(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Put("/inbox/:id/pages/:pageId/boxes", func(c *fiber.Ctx) error {
		var in struct {
			Boxes []domain.Quad `json:"boxes"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Inbox.SetPageBoxes(c.UserContext(), scope(c), c.Params("id"), c.Params("pageId"), in.Boxes))
	})
	r.Put("/inbox/:id/pages/:pageId/off", func(c *fiber.Ctx) error {
		var in struct {
			Off []string `json:"off"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Inbox.SetPageOff(c.UserContext(), scope(c), c.Params("id"), c.Params("pageId"), in.Off))
	})
}

// clinicalRoutes are the document viewer, record editing, search and
// catalog routes (HW-FIND-2, HW-FIND-3, HW-READ-5, HW-READ-7, HW-REC-6, HW-MET-1).
func (s *server) clinicalRoutes(r fiber.Router) {
	r.Get("/pages/:id/image", s.pageImage)
	r.Get("/documents/:id", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.Document(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Patch("/documents/:id", func(c *fiber.Ctx) error {
		var in service.DocumentInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Clinical.UpdateDocument(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Post("/documents/:id/confirm-approvals", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.ConfirmApprovals(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Post("/documents/:id/observations", func(c *fiber.Ctx) error {
		var in domain.ObservationInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		o, err := s.svc.Clinical.AddObservation(c.UserContext(), scope(c), c.Params("id"), in)
		return s.created(c, o, err)
	})
	r.Post("/documents/:id/findings", func(c *fiber.Ctx) error {
		var in domain.FindingInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		f, err := s.svc.Clinical.AddFinding(c.UserContext(), scope(c), c.Params("id"), in)
		return s.created(c, f, err)
	})
	r.Patch("/observations/:id", func(c *fiber.Ctx) error {
		var in service.ObservationPatch
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Clinical.UpdateObservation(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Put("/observations/:id/metric", func(c *fiber.Ctx) error {
		var in struct {
			Code string `json:"code"`
		}
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Clinical.LinkMetric(c.UserContext(), scope(c), c.Params("id"), in.Code))
	})
	r.Delete("/observations/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Clinical.DeleteObservation(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Get("/observations/:id/history", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.History(c.UserContext(), scope(c), domain.AuditObservation, c.Params("id")))
	})
	r.Patch("/findings/:id", func(c *fiber.Ctx) error {
		var in service.FindingPatch
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Clinical.UpdateFinding(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Delete("/findings/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Clinical.DeleteFinding(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Get("/findings/:id/history", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.History(c.UserContext(), scope(c), domain.AuditFinding, c.Params("id")))
	})
	r.Get("/search", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.Search(c.UserContext(), scope(c), c.Query("q")))
	})
	r.Get("/metrics", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.Metrics(c.UserContext(), scope(c)))
	})
	// Suggestions for printed names no metric knows (HW-MET-8).
	r.Get("/metric-suggestions", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.Suggestions(c.UserContext(), scope(c)))
	})
	r.Post("/metric-suggestions/:id/approve", func(c *fiber.Ctx) error {
		var in service.SuggestionEdit
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Clinical.ApproveSuggestion(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Post("/metric-suggestions/:id/reject", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.RejectSuggestion(c.UserContext(), scope(c), c.Params("id")))
	})
}

// metricRoutes are a person's metrics dashboard, trends and typed-in
// vitals (HW-MET-2..5).
func (s *server) metricRoutes(r fiber.Router) {
	r.Get("/people/:id/metrics", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.PersonMetrics(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Get("/people/:id/metrics/:code", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.PersonMetric(c.UserContext(), scope(c), c.Params("id"), c.Params("code")))
	})
	r.Post("/people/:id/measurements", func(c *fiber.Ctx) error {
		var in domain.MeasurementInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		o, err := s.svc.Clinical.AddMeasurements(c.UserContext(), scope(c), c.Params("id"), in)
		return s.created(c, o, err)
	})
}

// pageImage serves a page's rendered JPEG, or with ?masked=1 the masked
// copy the AI provider saw, after the tenant check and never cached.
func (s *server) pageImage(c *fiber.Ctx) error {
	body, err := s.svc.Records.OpenPageImage(c.UserContext(), scope(c), c.Params("id"), c.Query("masked") == "1")
	if err != nil {
		return respond(c, s.log, err)
	}
	c.Set(fiber.HeaderContentType, "image/jpeg")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.SendStream(body)
}
