package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

type DatabaseHandler struct {
	databaseService port.DatabaseService
	log             *logger.Logger
}

func NewDatabaseHandler(service port.DatabaseService, log *logger.Logger) *DatabaseHandler {
	return &DatabaseHandler{databaseService: service, log: log}
}

func (h *DatabaseHandler) fail(c *fiber.Ctx, err error) error {
	return respondError(c, h.log, "db_handler", err)
}

func (h *DatabaseHandler) GetAccounts(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListAccounts(forceFresh(c)))
}

func (h *DatabaseHandler) CreateAccount(c *fiber.Ctx) error {
	var body domain.AccountCreateInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if err := h.databaseService.CreateAccount(body); err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) UpdateAccount(c *fiber.Ctx) error {
	var body domain.AccountUpdateInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.UpdateAccount(pathParam(c, "name"), body))
}

func (h *DatabaseHandler) DeleteAccount(c *fiber.Ctx) error {
	name := pathParam(c, "name")
	if name == "" {
		return badRequest(c, "account name is required")
	}
	if err := h.databaseService.DeleteAccount(name); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) GetTransactions(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListTransactions(forceFresh(c)))
}

func (h *DatabaseHandler) CreateTransaction(c *fiber.Ctx) error {
	var body domain.TransactionCreateInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	result, err := h.databaseService.CreateTransaction(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(result)
}

func (h *DatabaseHandler) CreateTransactionBatch(c *fiber.Ctx) error {
	var body domain.TransactionBatchInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.CreateTransactions(body))
}

func (h *DatabaseHandler) GetCardPrograms(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListCardConfig(forceFresh(c)))
}

func (h *DatabaseHandler) GetCaptureRules(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListCaptureRules(forceFresh(c)))
}

func (h *DatabaseHandler) SaveCaptureRules(c *fiber.Ctx) error {
	var body struct {
		Rules []domain.CaptureRule `json:"rules"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.SaveCaptureRules(body.Rules))
}

func (h *DatabaseHandler) GetPreferences(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListPreferences(forceFresh(c)))
}

func (h *DatabaseHandler) SavePreferences(c *fiber.Ctx) error {
	var body domain.PreferencesUpdate
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.SavePreferences(body))
}

func (h *DatabaseHandler) UpdateTransaction(c *fiber.Ctx) error {
	var body domain.TransactionUpdateInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if body.ETag == "" {
		body.ETag = utils.CopyString(c.Get(fiber.HeaderIfMatch))
	}
	body.ETag = strings.Trim(body.ETag, `"`)
	return h.respond(c)(h.databaseService.UpdateTransaction(pathParam(c, "id"), body))
}

// DeleteTransaction takes the ETag from If-Match or the ?etag= query.
func (h *DatabaseHandler) DeleteTransaction(c *fiber.Ctx) error {
	etag := c.Get(fiber.HeaderIfMatch)
	if etag == "" {
		etag = c.Query("etag")
	}
	if err := h.databaseService.DeleteTransaction(pathParam(c, "id"), strings.Trim(utils.CopyString(etag), `"`)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) GetBudget(c *fiber.Ctx) error {
	year := 0
	if raw := strings.TrimSpace(c.Query("year")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 2000 || parsed > 2100 {
			return badRequest(c, "year must be a four-digit year")
		}
		year = parsed
	}
	return h.respond(c)(h.databaseService.ListBudget(year, forceFresh(c)))
}

func (h *DatabaseHandler) UpdateBudgetLimit(c *fiber.Ctx) error {
	var body domain.BudgetLimitUpdate
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if err := h.databaseService.UpdateBudgetLimit(body); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) ReallocateBudget(c *fiber.Ctx) error {
	var body domain.BudgetReallocation
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.ReallocateBudget(body))
}

func (h *DatabaseHandler) GetSheetInfo(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.SheetInfo())
}

func (h *DatabaseHandler) GetCategories(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListCategories(forceFresh(c)))
}

func (h *DatabaseHandler) GetGoals(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListGoals(forceFresh(c)))
}

// UpdateCategoryNature sets a category's nature (WM-LEND-5).
func (h *DatabaseHandler) UpdateCategoryNature(c *fiber.Ctx) error {
	var body domain.CategoryNatureUpdate
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.UpdateCategoryNature(body))
}

func (h *DatabaseHandler) GetTags(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.ListTags(forceFresh(c)))
}

func (h *DatabaseHandler) Sync(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.SyncCache())
}

func (h *DatabaseHandler) Init(c *fiber.Ctx) error {
	result, err := h.databaseService.WarmAIContent(forceFresh(c))
	if err != nil {
		status, code := classifyError(err)
		if status >= http.StatusInternalServerError {
			h.log.LogError(c.UserContext(), "db_handler: warm AI content failed", err)
		}
		return c.Status(status).JSON(fiber.Map{"ready": false, "error": err.Error(), "code": code})
	}
	return c.JSON(result)
}

func (h *DatabaseHandler) GetInvestmentAssets(c *fiber.Ctx) error {
	return h.respond(c)(h.databaseService.GetInvestmentAssets(forceFresh(c)))
}

// respond writes value as JSON, or the mapped error.
func (h *DatabaseHandler) respond(c *fiber.Ctx) func(value any, err error) error {
	return respondWith(c, h.log, "db_handler")
}

func respondWith(c *fiber.Ctx, log *logger.Logger, scope string) func(value any, err error) error {
	return func(value any, err error) error {
		if err != nil {
			return respondError(c, log, scope, err)
		}
		return c.JSON(value)
	}
}

func forceFresh(c *fiber.Ctx) bool {
	return strings.EqualFold(c.Query("force"), "true")
}

// pathParam returns the URL-decoded, trimmed route parameter. It copies the
// value because Fiber reuses the request buffer once the handler returns.
func pathParam(c *fiber.Ctx, name string) string {
	value := utils.CopyString(c.Params(name))
	if decoded, err := url.PathUnescape(value); err == nil {
		value = decoded
	}
	return strings.TrimSpace(value)
}
