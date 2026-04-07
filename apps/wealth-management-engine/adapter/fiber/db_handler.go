package fiber

import (
	configAdapter "apps/wealth-management-engine/adapter/config"
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type DatabaseHandler struct {
	databaseService port.DatabaseService
	log             *logger.Logger
}

func NewDatabaseHandler(service port.DatabaseService, log *logger.Logger) *DatabaseHandler {
	return &DatabaseHandler{databaseService: service, log: log}
}

func (h *DatabaseHandler) GetSheetsAccounts(c *fiber.Ctx) error {
	accounts, err := h.databaseService.ReadAccounts()
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(accounts)
}

func (h *DatabaseHandler) GetAccounts(c *fiber.Ctx) error {
	accounts, err := h.databaseService.ListAccounts()
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(accounts)
}

func (h *DatabaseHandler) CreateAccount(c *fiber.Ctx) error {
	var body domain.AccountCreateInput
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(body.Name) == "" || body.Type == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "name and type are required"})
	}
	if err := h.databaseService.CreateAccount(body); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) DeleteAccount(c *fiber.Ctx) error {
	name := c.Params("name")
	if strings.TrimSpace(name) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "account name is required"})
	}
	if err := h.databaseService.DeleteAccount(name); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}


func (h *DatabaseHandler) GetTransactions(c *fiber.Ctx) error {
	transactions, err := h.databaseService.ListTransactions(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(transactions)
}

func (h *DatabaseHandler) CreateTransaction(c *fiber.Ctx) error {
	var body domain.TransactionCreateInput
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if strings.TrimSpace(body.AccountName) == "" || strings.TrimSpace(body.Payee) == "" || strings.TrimSpace(body.Category) == "" || strings.TrimSpace(body.Date) == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "accountName, date, payee, and category are required"})
	}
	body.Tags = ensureSlice(body.Tags)
	if err := h.databaseService.CreateTransaction(body); err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) GetBudget(c *fiber.Ctx) error {
	items, err := h.databaseService.ListBudget(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(items)
}

func (h *DatabaseHandler) GetCategories(c *fiber.Ctx) error {
	items, err := h.databaseService.ListCategories(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(items)
}

func (h *DatabaseHandler) GetGoals(c *fiber.Ctx) error {
	items, err := h.databaseService.ListGoals(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(items)
}

func (h *DatabaseHandler) GetLoans(c *fiber.Ctx) error {
	items, err := h.databaseService.ListLoans(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(items)
}

func (h *DatabaseHandler) GetNotifications(c *fiber.Ctx) error {
	items, err := h.databaseService.ListPendingNotifications(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(items)
}

func (h *DatabaseHandler) MarkNotificationsDone(c *fiber.Ctx) error {
	var body struct {
		RowNumbers []int `json:"rowNumbers"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if len(body.RowNumbers) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "rowNumbers must be a non-empty array"})
	}
	for _, rowNumber := range body.RowNumbers {
		if rowNumber <= 0 {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "rowNumbers must contain positive row indices"})
		}
		if err := h.databaseService.MarkNotificationDone(rowNumber); err != nil {
			return h.handleError(c, err)
		}
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) GetTags(c *fiber.Ctx) error {
	tags, err := h.databaseService.ListTags(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(tags)
}

func (h *DatabaseHandler) Sync(c *fiber.Ctx) error {
	result, err := h.databaseService.SyncCache()
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(result)
}

func (h *DatabaseHandler) Init(c *fiber.Ctx) error {
	result, err := h.databaseService.WarmAIContent(forceFresh(c))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, configAdapter.ErrMissingSheetsConfig) {
			statusCode = http.StatusServiceUnavailable
		}
		return c.Status(statusCode).JSON(fiber.Map{"ready": false, "error": err.Error()})
	}
	return c.JSON(result)
}

func (h *DatabaseHandler) GetInvestmentAssets(c *fiber.Ctx) error {
	assets, err := h.databaseService.GetInvestmentAssets(forceFresh(c))
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(assets)
}

func (h *DatabaseHandler) handleError(c *fiber.Ctx, err error) error {
	requestID := c.Get("X-Request-ID")
	statusCode := fiber.StatusInternalServerError
	if errors.Is(err, configAdapter.ErrMissingSheetsConfig) {
		statusCode = fiber.StatusServiceUnavailable
	}
	h.log.LogError(c.UserContext(), "db_handler: operation failed", err,
		slog.String("request_id", requestID),
		slog.String("endpoint", c.Path()),
	)
	return c.Status(statusCode).JSON(fiber.Map{"error": err.Error()})
}

func forceFresh(c *fiber.Ctx) bool {
	return strings.EqualFold(c.Query("force"), "true")
}

func ensureSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
