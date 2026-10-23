package fiber

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
	"apps/health-watcher/service"
)

func (s *server) recordRoutes(r fiber.Router) {
	r.Get("/people", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Records.People(c.UserContext(), scope(c)))
	})
	r.Post("/people", func(c *fiber.Ctx) error {
		var in domain.PersonInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		p, err := s.svc.Records.CreatePerson(c.UserContext(), scope(c), in)
		return s.created(c, p, err)
	})
	r.Get("/people/:id", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Records.Person(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Patch("/people/:id", func(c *fiber.Ctx) error {
		var in domain.PersonInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Records.UpdatePerson(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Delete("/people/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Records.DeletePerson(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Get("/people/:id/visits", func(c *fiber.Ctx) error {
		f := port.VisitFilter{Type: c.Query("type"), ProviderID: c.Query("providerId"), From: c.Query("from"), To: c.Query("to")}
		return s.list(c)(s.svc.Records.Visits(c.UserContext(), scope(c), c.Params("id"), f))
	})
	r.Get("/providers", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Records.Providers(c.UserContext(), scope(c)))
	})
	r.Post("/visits", func(c *fiber.Ctx) error {
		var in domain.VisitInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		v, err := s.svc.Records.CreateVisit(c.UserContext(), scope(c), in)
		return s.created(c, v, err)
	})
	r.Get("/visits/:id", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Records.Visit(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Patch("/visits/:id", func(c *fiber.Ctx) error {
		var in domain.VisitInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Records.UpdateVisit(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Delete("/visits/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Records.DeleteVisit(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Delete("/documents/:id", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Records.DeleteDocument(c.UserContext(), scope(c), c.Params("id")))
	})
	r.Post("/documents/:id/unfile", func(c *fiber.Ctx) error {
		itemID, err := s.svc.Inbox.Unfile(c.UserContext(), scope(c), c.Params("id"))
		return s.list(c)(fiber.Map{"itemId": itemID}, err)
	})
	r.Post("/documents/:id/pages/:pageId/check", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Inbox.CheckDocumentPage(c.UserContext(), scope(c), c.Params("id"), c.Params("pageId")))
	})
	r.Delete("/documents/:id/pages/:pageId", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Clinical.RemoveDocumentPage(c.UserContext(), scope(c), c.Params("id"), c.Params("pageId")))
	})
	r.Get("/files/:id", s.file)
}

// file serves an original after the tenant check, never cached (HW-OPS-2).
// ?download=1 asks the browser to save it instead of showing it.
func (s *server) file(c *fiber.Ctx) error {
	f, body, err := s.svc.Records.OpenFile(c.UserContext(), scope(c), c.Params("id"))
	if err != nil {
		return respond(c, s.log, err)
	}
	disposition := "inline"
	if c.Query("download") == "1" {
		disposition = "attachment"
	}
	name := f.OriginalName
	if name == "" {
		name = f.ID
	}
	c.Set(fiber.HeaderContentType, f.MIME)
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(name)))
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.SendStream(body, int(f.Size))
}

func (s *server) inboxRoutes(r fiber.Router) {
	r.Get("/inbox", func(c *fiber.Ctx) error {
		return s.list(c)(s.svc.Inbox.Items(c.UserContext(), scope(c)))
	})
	r.Post("/inbox", func(c *fiber.Ctx) error {
		return s.upload(c, scope(c), domain.SourceWeb)
	})
	r.Post("/inbox/:id/file", func(c *fiber.Ctx) error {
		var in domain.FileInput
		if err := parseJSON(c, &in); err != nil {
			return respond(c, s.log, err)
		}
		return s.list(c)(s.svc.Inbox.File(c.UserContext(), scope(c), c.Params("id"), in))
	})
	r.Post("/uploads", func(c *fiber.Ctx) error {
		return s.stage(c, scope(c))
	})
	r.Post("/inbox/:id/files", func(c *fiber.Ctx) error {
		return s.addFiles(c, scope(c), c.Params("id"))
	})
	r.Post("/inbox/:id/discard", func(c *fiber.Ctx) error {
		return s.noContent(c, s.svc.Inbox.Discard(c.UserContext(), scope(c), c.Params("id")))
	})
}

// tokenUpload is the Shortcut's endpoint: an upload token instead of a
// session, so it needs no CSRF token (HW-IMP-4).
func (s *server) tokenUpload(c *fiber.Ctx) error {
	sc, err := s.svc.Accounts.UploadScope(c.UserContext(), bearer(c))
	if err != nil {
		return respond(c, s.log, err)
	}
	source := domain.SourceShortcut
	if sc.Credential == domain.CredentialCaptureLink {
		source = domain.SourcePhone
	}
	if sc.ItemID != "" {
		return s.addFiles(c, sc, sc.ItemID)
	}
	return s.upload(c, sc, source)
}

// tokenStage is the phone's send-ahead endpoint with a capture link
// (HW-IMP-11).
func (s *server) tokenStage(c *fiber.Ctx) error {
	sc, err := s.svc.Accounts.UploadScope(c.UserContext(), bearer(c))
	if err != nil {
		return respond(c, s.log, err)
	}
	return s.stage(c, sc)
}

// stage stores files sent ahead of Send; the answer lists each file's id.
func (s *server) stage(c *fiber.Ctx, sc domain.Scope) error {
	files, err := uploadedFiles(c)
	if err != nil {
		return respond(c, s.log, err)
	}
	results, err := s.svc.Inbox.Stage(c.UserContext(), sc, files)
	if err != nil {
		return respond(c, s.log, err)
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"files": results})
}

// captureTarget tells the phone capture page where its link adds photos:
// a new inbox item, or the one item the link was made on (HW-IMP-10).
func (s *server) captureTarget(c *fiber.Ctx) error {
	sc, err := s.svc.Accounts.UploadScope(c.UserContext(), bearer(c))
	if err != nil {
		return respond(c, s.log, err)
	}
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	return c.JSON(fiber.Map{"item": sc.ItemID != ""})
}

func (s *server) addFiles(c *fiber.Ctx, sc domain.Scope, itemID string) error {
	files, err := uploadedFiles(c)
	if err != nil {
		return respond(c, s.log, err)
	}
	form, err := c.MultipartForm()
	if err != nil {
		return respond(c, s.log, err)
	}
	resp, err := s.svc.Inbox.AddFiles(c.UserContext(), sc, itemID, files, form.Value["fileId"])
	if err != nil {
		return respond(c, s.log, err)
	}
	return c.Status(fiber.StatusAccepted).JSON(resp)
}

// upload reads a multipart body: one or more `files` parts (the Shortcut
// may also send `file`), an optional `note`, `oneItem=true` and, in
// backlog mode, the `personId` every new item belongs to.
func (s *server) upload(c *fiber.Ctx, sc domain.Scope, source domain.InboxSource) error {
	if err := sc.RequireUpload(); err != nil {
		return respond(c, s.log, err)
	}
	files, err := uploadedFiles(c)
	if err != nil {
		return respond(c, s.log, err)
	}
	form, err := c.MultipartForm()
	if err != nil {
		return respond(c, s.log, err)
	}
	req := service.UploadRequest{Source: source, Files: files, Note: first(form.Value["note"]), OneItem: first(form.Value["oneItem"]) == "true",
		PersonID: first(form.Value["personId"]), FileIDs: form.Value["fileId"]}
	resp, err := s.svc.Inbox.Upload(c.UserContext(), sc, req)
	if err != nil {
		return respond(c, s.log, err)
	}
	return c.Status(fiber.StatusAccepted).JSON(resp)
}

// uploadedFiles reads the `files` parts of a multipart body (the Shortcut
// may also send `file`).
func uploadedFiles(c *fiber.Ctx) ([]domain.Upload, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, domain.Invalid("send the files as multipart/form-data")
	}
	headers := append(form.File["files"], form.File["file"]...)
	files := make([]domain.Upload, 0, len(headers))
	for _, h := range headers {
		data, err := readPart(h)
		if err != nil {
			return nil, err
		}
		files = append(files, domain.Upload{Name: h.Filename, Data: data})
	}
	return files, nil
}

func readPart(h *multipart.FileHeader) ([]byte, error) {
	f, err := h.Open()
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	return data, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
